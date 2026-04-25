package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jf-ar/compli-church/internal/ports"
)

type MemberRepo struct {
	pool *pgxpool.Pool
}

func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{pool: pool}
}

// Compile-time interface checks.
var _ ports.MemberRepository = (*MemberRepo)(nil)
var _ ports.RoleRepository = (*MemberRepo)(nil)
var _ ports.InstrumentRepository = (*MemberRepo)(nil)

// ── MemberRepository ──────────────────────────────────────────────────────────

func (r *MemberRepo) ListMembers(ctx context.Context, churchID uuid.UUID, f ports.ListMembersFilter) ([]ports.Member, int, error) {
	args := []any{churchID}
	n := 1
	where := `mcm.church_id = $1 AND mcm.left_at IS NULL`

	if f.IsActive != nil {
		n++
		where += fmt.Sprintf(` AND m.is_active = $%d`, n)
		args = append(args, *f.IsActive)
	}
	if f.Search != nil && *f.Search != "" {
		n++
		where += fmt.Sprintf(` AND (m.name ILIKE $%d OR m.email ILIKE $%d)`, n, n)
		args = append(args, "%"+*f.Search+"%")
	}
	if f.Role != nil && *f.Role != "" {
		n++
		where += fmt.Sprintf(` AND EXISTS (
			SELECT 1 FROM member_role_assignments mra2
			JOIN roles r2 ON mra2.role_id = r2.id
			JOIN member_church_memberships mcm2 ON mra2.membership_id = mcm2.id
			WHERE mcm2.member_id = m.id AND mcm2.church_id = $1
			  AND r2.base_profile = $%d AND mcm2.left_at IS NULL
		)`, n)
		args = append(args, *f.Role)
	}

	var total int
	countQ := `SELECT COUNT(*) FROM members m JOIN member_church_memberships mcm ON m.id = mcm.member_id WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	n++
	args = append(args, f.PerPage)
	n++
	args = append(args, (f.Page-1)*f.PerPage)

	q := fmt.Sprintf(`
		SELECT m.id, m.name, m.email, m.phone, m.birth_date, m.avatar_url, m.is_active, m.created_at
		FROM members m
		JOIN member_church_memberships mcm ON m.id = mcm.member_id
		WHERE %s
		ORDER BY m.name
		LIMIT $%d OFFSET $%d`, where, n-1, n)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var members []ports.Member
	var ids []uuid.UUID
	for rows.Next() {
		m, err := scanMemberBasic(rows)
		if err != nil {
			return nil, 0, err
		}
		members = append(members, *m)
		ids = append(ids, m.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return members, total, nil
	}

	roleMap, err := r.batchFetchRoles(ctx, ids, churchID)
	if err != nil {
		return nil, 0, err
	}
	instrMap, err := r.batchFetchInstruments(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range members {
		members[i].Roles = roleMap[members[i].ID]
		members[i].Instruments = instrMap[members[i].ID]
	}

	return members, total, nil
}

func (r *MemberRepo) CreateMember(ctx context.Context, churchID uuid.UUID, input ports.MemberCreateInput, assignedBy uuid.UUID, passwordHash string) (*ports.Member, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var m ports.Member
	var phone pgtype.Text
	var birthDate pgtype.Date
	var avatarURL pgtype.Text

	if input.Phone != nil {
		phone = pgtype.Text{String: *input.Phone, Valid: true}
	}
	if input.BirthDate != nil {
		birthDate = pgtype.Date{Time: *input.BirthDate, Valid: true}
	}

	err = tx.QueryRow(ctx,
		`INSERT INTO members (name, email, phone, birth_date, password_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, email, phone, birth_date, avatar_url, is_active, created_at`,
		input.Name, input.Email, phone, birthDate, passwordHash,
	).Scan(&m.ID, &m.Name, &m.Email, &phone, &birthDate, &avatarURL, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	if phone.Valid {
		m.Phone = &phone.String
	}
	if birthDate.Valid {
		t := birthDate.Time
		m.BirthDate = &t
	}
	if avatarURL.Valid {
		m.AvatarURL = &avatarURL.String
	}

	var membershipID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO member_church_memberships (member_id, church_id, is_primary)
		 VALUES ($1, $2, TRUE)
		 RETURNING id`,
		m.ID, churchID,
	).Scan(&membershipID)
	if err != nil {
		return nil, err
	}

	for _, roleID := range input.RoleIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO member_role_assignments (membership_id, role_id, assigned_by)
			 VALUES ($1, $2, $3)`,
			membershipID, roleID, assignedBy,
		)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetMemberByID(ctx, m.ID, churchID)
}

func (r *MemberRepo) GetMemberByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Member, error) {
	var m ports.Member
	var phone pgtype.Text
	var birthDate pgtype.Date
	var avatarURL pgtype.Text

	err := r.pool.QueryRow(ctx,
		`SELECT m.id, m.name, m.email, m.phone, m.birth_date, m.avatar_url, m.is_active, m.created_at
		 FROM members m
		 JOIN member_church_memberships mcm ON m.id = mcm.member_id
		 WHERE m.id = $1 AND mcm.church_id = $2 AND mcm.left_at IS NULL`,
		id, churchID,
	).Scan(&m.ID, &m.Name, &m.Email, &phone, &birthDate, &avatarURL, &m.IsActive, &m.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	if phone.Valid {
		m.Phone = &phone.String
	}
	if birthDate.Valid {
		t := birthDate.Time
		m.BirthDate = &t
	}
	if avatarURL.Valid {
		m.AvatarURL = &avatarURL.String
	}

	roles, err := r.GetMemberRoles(ctx, id, churchID)
	if err != nil {
		return nil, err
	}
	m.Roles = roles

	instruments, err := r.GetMemberInstruments(ctx, id, churchID)
	if err != nil {
		return nil, err
	}
	m.Instruments = instruments

	return &m, nil
}

func (r *MemberRepo) UpdateMember(ctx context.Context, id, churchID uuid.UUID, input ports.MemberUpdateInput) (*ports.Member, error) {
	var phone pgtype.Text
	var birthDate pgtype.Date

	if input.Phone != nil {
		phone = pgtype.Text{String: *input.Phone, Valid: true}
	}
	if input.BirthDate != nil {
		birthDate = pgtype.Date{Time: *input.BirthDate, Valid: true}
	}

	// Verify member belongs to church before updating.
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM member_church_memberships
			WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL
		)`, id, churchID,
	).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ports.ErrNotFound
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE members SET name = $1, phone = $2, birth_date = $3, updated_at = NOW()
		 WHERE id = $4`,
		input.Name, phone, birthDate, id,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}

	return r.GetMemberByID(ctx, id, churchID)
}

func (r *MemberRepo) DeactivateMember(ctx context.Context, id, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE members SET is_active = FALSE, updated_at = NOW()
		 WHERE id = $1
		 AND EXISTS (
			SELECT 1 FROM member_church_memberships
			WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL
		 )`, id, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *MemberRepo) GetMemberRoles(ctx context.Context, memberID, churchID uuid.UUID) ([]ports.MemberRole, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.base_profile
		 FROM member_role_assignments mra
		 JOIN roles r ON mra.role_id = r.id
		 JOIN member_church_memberships mcm ON mra.membership_id = mcm.id
		 WHERE mcm.member_id = $1 AND mcm.church_id = $2 AND mcm.left_at IS NULL`,
		memberID, churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []ports.MemberRole
	for rows.Next() {
		var role ports.MemberRole
		if err := rows.Scan(&role.ID, &role.Name, &role.BaseProfile); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *MemberRepo) AssignRole(ctx context.Context, memberID, churchID, roleID, assignedBy uuid.UUID) error {
	var membershipID uuid.UUID
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM member_church_memberships
		 WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL`,
		memberID, churchID,
	).Scan(&membershipID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ports.ErrNotFound
		}
		return err
	}

	_, err = r.pool.Exec(ctx,
		`INSERT INTO member_role_assignments (membership_id, role_id, assigned_by)
		 VALUES ($1, $2, $3)`,
		membershipID, roleID, assignedBy,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return ports.ErrAlreadyExists
		}
		return err
	}
	return nil
}

func (r *MemberRepo) RemoveRole(ctx context.Context, memberID, roleID, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM member_role_assignments mra
		 USING member_church_memberships mcm
		 WHERE mra.membership_id = mcm.id
		   AND mcm.member_id = $1
		   AND mcm.church_id = $2
		   AND mra.role_id = $3
		   AND mcm.left_at IS NULL`,
		memberID, churchID, roleID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *MemberRepo) GetMemberInstruments(ctx context.Context, memberID, churchID uuid.UUID) ([]ports.MemberInstrument, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT mi.id, mi.instrument_id, i.name AS instrument_name, mi.is_primary
		 FROM member_instruments mi
		 JOIN instruments i ON mi.instrument_id = i.id
		 JOIN member_church_memberships mcm ON mi.member_id = mcm.member_id
		 WHERE mi.member_id = $1 AND mcm.church_id = $2 AND mcm.left_at IS NULL`,
		memberID, churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []ports.MemberInstrument
	for rows.Next() {
		var inst ports.MemberInstrument
		if err := rows.Scan(&inst.ID, &inst.InstrumentID, &inst.InstrumentName, &inst.IsPrimary); err != nil {
			return nil, err
		}
		instruments = append(instruments, inst)
	}
	return instruments, rows.Err()
}

func (r *MemberRepo) AddMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID, isPrimary bool) (*ports.MemberInstrument, error) {
	// Validate member belongs to this church.
	var memberExists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM member_church_memberships
			WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL
		)`,
		memberID, churchID,
	).Scan(&memberExists); err != nil {
		return nil, err
	}
	if !memberExists {
		return nil, ports.ErrNotFound
	}

	var inst ports.MemberInstrument
	err := r.pool.QueryRow(ctx,
		`WITH first_check AS (
			SELECT NOT EXISTS (
				SELECT 1 FROM member_instruments WHERE member_id = $1
			) AS is_first
		),
		ins AS (
			INSERT INTO member_instruments (member_id, instrument_id, is_primary)
			SELECT $1, $2, CASE WHEN fc.is_first THEN TRUE ELSE $3 END
			FROM first_check fc
			RETURNING id, instrument_id, is_primary
		)
		SELECT ins.id, ins.instrument_id, i.name AS instrument_name, ins.is_primary
		FROM ins JOIN instruments i ON ins.instrument_id = i.id`,
		memberID, instrumentID, isPrimary,
	).Scan(&inst.ID, &inst.InstrumentID, &inst.InstrumentName, &inst.IsPrimary)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return &inst, nil
}

func (r *MemberRepo) RemoveMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM member_instruments
		 WHERE member_id = $1
		   AND instrument_id = $3
		   AND EXISTS (
			 SELECT 1 FROM member_church_memberships
			 WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL
		   )`,
		memberID, churchID, instrumentID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ── RoleRepository ────────────────────────────────────────────────────────────

func (r *MemberRepo) ListRoles(ctx context.Context, churchID uuid.UUID) ([]ports.Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, church_id, name, base_profile, is_system
		 FROM roles
		 WHERE church_id IS NULL OR church_id = $1
		 ORDER BY is_system DESC, name ASC`,
		churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []ports.Role
	for rows.Next() {
		r2, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, *r2)
	}
	return roles, rows.Err()
}

func (r *MemberRepo) CreateRole(ctx context.Context, churchID uuid.UUID, name, baseProfile string) (*ports.Role, error) {
	role, err := scanRole(r.pool.QueryRow(ctx,
		`INSERT INTO roles (church_id, name, base_profile, is_system)
		 VALUES ($1, $2, $3, FALSE)
		 RETURNING id, church_id, name, base_profile, is_system`,
		churchID, name, baseProfile,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return role, nil
}

func (r *MemberRepo) GetRoleByID(ctx context.Context, id uuid.UUID) (*ports.Role, error) {
	role, err := scanRole(r.pool.QueryRow(ctx,
		`SELECT id, church_id, name, base_profile, is_system FROM roles WHERE id = $1`,
		id,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *MemberRepo) UpdateRole(ctx context.Context, id, churchID uuid.UUID, name, baseProfile string) (*ports.Role, error) {
	role, err := scanRole(r.pool.QueryRow(ctx,
		`UPDATE roles SET name = $1, base_profile = $2
		 WHERE id = $3 AND church_id = $4 AND is_system = FALSE
		 RETURNING id, church_id, name, base_profile, is_system`,
		name, baseProfile, id, churchID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return role, nil
}

func (r *MemberRepo) DeleteRole(ctx context.Context, id, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM roles WHERE id = $1 AND church_id = $2 AND is_system = FALSE`,
		id, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ── InstrumentRepository ──────────────────────────────────────────────────────

func (r *MemberRepo) ListInstruments(ctx context.Context, churchID uuid.UUID) ([]ports.Instrument, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, church_id, name, is_system
		 FROM instruments
		 WHERE church_id IS NULL OR church_id = $1
		 ORDER BY is_system DESC, name ASC`,
		churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []ports.Instrument
	for rows.Next() {
		inst, err := scanInstrument(rows)
		if err != nil {
			return nil, err
		}
		instruments = append(instruments, *inst)
	}
	return instruments, rows.Err()
}

func (r *MemberRepo) CreateInstrument(ctx context.Context, churchID uuid.UUID, name string) (*ports.Instrument, error) {
	inst, err := scanInstrument(r.pool.QueryRow(ctx,
		`INSERT INTO instruments (church_id, name, is_system)
		 VALUES ($1, $2, FALSE)
		 RETURNING id, church_id, name, is_system`,
		churchID, name,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return inst, nil
}

func (r *MemberRepo) GetInstrumentByID(ctx context.Context, id uuid.UUID) (*ports.Instrument, error) {
	inst, err := scanInstrument(r.pool.QueryRow(ctx,
		`SELECT id, church_id, name, is_system FROM instruments WHERE id = $1`,
		id,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return inst, nil
}

func (r *MemberRepo) DeleteInstrument(ctx context.Context, id, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM instruments WHERE id = $1 AND church_id = $2 AND is_system = FALSE`,
		id, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMemberBasic(row rowScanner) (*ports.Member, error) {
	var m ports.Member
	var phone pgtype.Text
	var birthDate pgtype.Date
	var avatarURL pgtype.Text

	if err := row.Scan(&m.ID, &m.Name, &m.Email, &phone, &birthDate, &avatarURL, &m.IsActive, &m.CreatedAt); err != nil {
		return nil, err
	}
	if phone.Valid {
		m.Phone = &phone.String
	}
	if birthDate.Valid {
		t := birthDate.Time
		m.BirthDate = &t
	}
	if avatarURL.Valid {
		m.AvatarURL = &avatarURL.String
	}
	return &m, nil
}

func scanRole(row rowScanner) (*ports.Role, error) {
	var role ports.Role
	var churchID pgtype.UUID
	if err := row.Scan(&role.ID, &churchID, &role.Name, &role.BaseProfile, &role.IsSystem); err != nil {
		return nil, err
	}
	if churchID.Valid {
		id := uuid.UUID(churchID.Bytes)
		role.ChurchID = &id
	}
	return &role, nil
}

func scanInstrument(row rowScanner) (*ports.Instrument, error) {
	var inst ports.Instrument
	var churchID pgtype.UUID
	if err := row.Scan(&inst.ID, &churchID, &inst.Name, &inst.IsSystem); err != nil {
		return nil, err
	}
	if churchID.Valid {
		id := uuid.UUID(churchID.Bytes)
		inst.ChurchID = &id
	}
	return &inst, nil
}

func (r *MemberRepo) batchFetchRoles(ctx context.Context, memberIDs []uuid.UUID, churchID uuid.UUID) (map[uuid.UUID][]ports.MemberRole, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT mcm.member_id, r.id, r.name, r.base_profile
		 FROM member_role_assignments mra
		 JOIN roles r ON mra.role_id = r.id
		 JOIN member_church_memberships mcm ON mra.membership_id = mcm.id
		 WHERE mcm.member_id = ANY($1) AND mcm.church_id = $2 AND mcm.left_at IS NULL`,
		memberIDs, churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]ports.MemberRole)
	for rows.Next() {
		var memberID uuid.UUID
		var role ports.MemberRole
		if err := rows.Scan(&memberID, &role.ID, &role.Name, &role.BaseProfile); err != nil {
			return nil, err
		}
		result[memberID] = append(result[memberID], role)
	}
	return result, rows.Err()
}

func (r *MemberRepo) batchFetchInstruments(ctx context.Context, memberIDs []uuid.UUID) (map[uuid.UUID][]ports.MemberInstrument, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT mi.member_id, mi.id, mi.instrument_id, i.name AS instrument_name, mi.is_primary
		 FROM member_instruments mi
		 JOIN instruments i ON mi.instrument_id = i.id
		 WHERE mi.member_id = ANY($1)`,
		memberIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[uuid.UUID][]ports.MemberInstrument)
	for rows.Next() {
		var memberID uuid.UUID
		var inst ports.MemberInstrument
		if err := rows.Scan(&memberID, &inst.ID, &inst.InstrumentID, &inst.InstrumentName, &inst.IsPrimary); err != nil {
			return nil, err
		}
		result[memberID] = append(result[memberID], inst)
	}
	return result, rows.Err()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
