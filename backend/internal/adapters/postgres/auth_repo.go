package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jf-ar/compli-church/internal/ports"
)

var _ ports.AuthRepository = (*AuthRepo)(nil)

type AuthRepo struct {
	pool *pgxpool.Pool
}

func NewAuthRepo(pool *pgxpool.Pool) *AuthRepo {
	return &AuthRepo{pool: pool}
}

func (r *AuthRepo) GetMemberForLogin(ctx context.Context, email string) (*ports.LoginMember, error) {
	// Step 1: member + primary membership + church in one query
	const memberQ = `
		SELECT m.id, m.name, m.email, m.password_hash, m.phone, m.birth_date, m.avatar_url, m.is_active, m.created_at,
		       mcm.id AS membership_id, mcm.church_id,
		       c.id, c.parent_church_id, c.name, c.denomination_name, c.cnpj, c.address,
		       c.is_autonomous, c.plan_tier, c.member_count_cache, c.created_at
		FROM members m
		JOIN member_church_memberships mcm
		     ON m.id = mcm.member_id AND mcm.is_primary = TRUE AND mcm.left_at IS NULL
		JOIN churches c ON mcm.church_id = c.id
		WHERE m.email = $1 AND m.is_active = TRUE`

	var (
		lm           ports.LoginMember
		membershipID uuid.UUID
		phone        pgtype.Text
		birthDate    pgtype.Date
		avatarURL    pgtype.Text
		parentID     pgtype.UUID
		denomName    pgtype.Text
		cnpj         pgtype.Text
		address      pgtype.Text
	)

	row := r.pool.QueryRow(ctx, memberQ, email)
	err := row.Scan(
		&lm.ID, &lm.Name, &lm.Email, &lm.PasswordHash,
		&phone, &birthDate, &avatarURL, &lm.IsActive, &lm.CreatedAt,
		&membershipID, &lm.PrimaryChurchID,
		&lm.Church.ID, &parentID, &lm.Church.Name, &denomName, &cnpj, &address,
		&lm.Church.IsAutonomous, &lm.Church.PlanTier, &lm.Church.MemberCountCache, &lm.Church.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	if phone.Valid {
		lm.Phone = &phone.String
	}
	if birthDate.Valid {
		t := birthDate.Time
		lm.BirthDate = &t
	}
	if avatarURL.Valid {
		lm.AvatarURL = &avatarURL.String
	}
	if parentID.Valid {
		id := uuid.UUID(parentID.Bytes)
		lm.Church.ParentChurchID = &id
	}
	if denomName.Valid {
		lm.Church.DenominationName = &denomName.String
	}
	if cnpj.Valid {
		lm.Church.CNPJ = &cnpj.String
	}
	if address.Valid {
		lm.Church.Address = &address.String
	}

	// Step 2: roles for primary membership
	roles, err := r.getMemberRoles(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	lm.Roles = roles
	lm.BaseProfile = highestProfile(roles)

	// Step 3: all church IDs
	churchIDs, err := r.getMemberChurchIDs(ctx, lm.ID)
	if err != nil {
		return nil, err
	}
	lm.ChurchIDs = churchIDs

	// Step 4: instruments
	instruments, err := r.getMemberInstruments(ctx, lm.ID)
	if err != nil {
		return nil, err
	}
	lm.Instruments = instruments

	return &lm, nil
}

func (r *AuthRepo) GetMemberForToken(ctx context.Context, memberID uuid.UUID) (*ports.TokenMember, error) {
	const q = `
		SELECT m.id, m.is_active, mcm.church_id
		FROM members m
		JOIN member_church_memberships mcm
		     ON m.id = mcm.member_id AND mcm.is_primary = TRUE AND mcm.left_at IS NULL
		WHERE m.id = $1 AND m.is_active = TRUE`

	var tm ports.TokenMember
	var membershipID uuid.UUID

	// We need membership_id to get roles; run a separate lookup
	const membershipQ = `
		SELECT mcm.id, mcm.church_id
		FROM member_church_memberships mcm
		WHERE mcm.member_id = $1 AND mcm.is_primary = TRUE AND mcm.left_at IS NULL`

	row := r.pool.QueryRow(ctx, q, memberID)
	err := row.Scan(&tm.ID, &tm.IsActive, &tm.PrimaryChurchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	mRow := r.pool.QueryRow(ctx, membershipQ, memberID)
	if err := mRow.Scan(&membershipID, &tm.PrimaryChurchID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	roles, err := r.getMemberRoles(ctx, membershipID)
	if err != nil {
		return nil, err
	}
	tm.BaseProfile = highestProfile(roles)

	churchIDs, err := r.getMemberChurchIDs(ctx, memberID)
	if err != nil {
		return nil, err
	}
	tm.ChurchIDs = churchIDs

	return &tm, nil
}

func (r *AuthRepo) CreateRefreshToken(ctx context.Context, memberID, jti uuid.UUID, expiresAt time.Time) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (member_id, jti, expires_at) VALUES ($1, $2, $3)`,
		memberID, jti, expiresAt,
	)
	return err
}

func (r *AuthRepo) GetRefreshTokenByJTI(ctx context.Context, jti uuid.UUID) (*ports.RefreshToken, error) {
	var rt ports.RefreshToken
	var revokedAt pgtype.Timestamptz

	row := r.pool.QueryRow(ctx,
		`SELECT id, member_id, jti, expires_at, revoked_at, created_at FROM refresh_tokens WHERE jti = $1`,
		jti,
	)
	err := row.Scan(&rt.ID, &rt.MemberID, &rt.JTI, &rt.ExpiresAt, &revokedAt, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		rt.RevokedAt = &t
	}
	return &rt, nil
}

func (r *AuthRepo) RevokeRefreshToken(ctx context.Context, jti uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE jti = $1 AND revoked_at IS NULL`,
		jti,
	)
	return err
}

func (r *AuthRepo) RevokeAllMemberRefreshTokens(ctx context.Context, memberID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE refresh_tokens SET revoked_at = NOW() WHERE member_id = $1 AND revoked_at IS NULL`,
		memberID,
	)
	return err
}

func (r *AuthRepo) CreateChurchWithPastor(ctx context.Context, params ports.RegisterParams) (*ports.LoginMember, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	// a. Insert church
	var churchID uuid.UUID
	var churchCreatedAt time.Time
	err = tx.QueryRow(ctx,
		`INSERT INTO churches (name, plan_tier, is_autonomous)
		 VALUES ($1, 'free', FALSE)
		 RETURNING id, created_at`,
		params.ChurchName,
	).Scan(&churchID, &churchCreatedAt)
	if err != nil {
		return nil, err
	}

	// b. Insert member
	var memberID uuid.UUID
	var memberCreatedAt time.Time
	err = tx.QueryRow(ctx,
		`INSERT INTO members (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		params.PastorName, params.Email, params.PasswordHash,
	).Scan(&memberID, &memberCreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}

	// c. Insert membership (is_primary = true)
	var membershipID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO member_church_memberships (member_id, church_id, is_primary)
		 VALUES ($1, $2, TRUE)
		 RETURNING id`,
		memberID, churchID,
	).Scan(&membershipID)
	if err != nil {
		return nil, err
	}

	// d. Fetch system Pastor role
	var roleID uuid.UUID
	var roleName, roleBaseProfile string
	err = tx.QueryRow(ctx,
		`SELECT id, name, base_profile FROM roles WHERE name = 'Pastor' AND church_id IS NULL AND is_system = TRUE LIMIT 1`,
	).Scan(&roleID, &roleName, &roleBaseProfile)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("system Pastor role not found — check seed data")
		}
		return nil, err
	}

	// e. Assign Pastor role (assigned_by = the pastor themselves)
	_, err = tx.Exec(ctx,
		`INSERT INTO member_role_assignments (membership_id, role_id, assigned_by) VALUES ($1, $2, $3)`,
		membershipID, roleID, memberID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	lm := &ports.LoginMember{
		ID:              memberID,
		Name:            params.PastorName,
		Email:           params.Email,
		PasswordHash:    params.PasswordHash,
		IsActive:        true,
		CreatedAt:       memberCreatedAt,
		PrimaryChurchID: churchID,
		BaseProfile:     roleBaseProfile,
		ChurchIDs:       []uuid.UUID{churchID},
		Church: ports.ChurchData{
			ID:           churchID,
			Name:         params.ChurchName,
			IsAutonomous: false,
			PlanTier:     "free",
			CreatedAt:    churchCreatedAt,
		},
		Roles: []ports.MemberRole{
			{ID: roleID, Name: roleName, BaseProfile: roleBaseProfile},
		},
		Instruments: []ports.MemberInstrument{},
	}

	return lm, nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func (r *AuthRepo) getMemberRoles(ctx context.Context, membershipID uuid.UUID) ([]ports.MemberRole, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.name, r.base_profile
		 FROM member_role_assignments mra
		 JOIN roles r ON mra.role_id = r.id
		 WHERE mra.membership_id = $1`,
		membershipID,
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

func (r *AuthRepo) getMemberChurchIDs(ctx context.Context, memberID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT church_id FROM member_church_memberships WHERE member_id = $1 AND left_at IS NULL`,
		memberID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *AuthRepo) getMemberInstruments(ctx context.Context, memberID uuid.UUID) ([]ports.MemberInstrument, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT mi.id, mi.instrument_id, i.name AS instrument_name, mi.is_primary
		 FROM member_instruments mi
		 JOIN instruments i ON mi.instrument_id = i.id
		 WHERE mi.member_id = $1`,
		memberID,
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

// highestProfile returns the most-privileged base_profile from a set of roles.
func highestProfile(roles []ports.MemberRole) string {
	best := 1
	for _, r := range roles {
		if rank := profileRank(r.BaseProfile); rank > best {
			best = rank
		}
	}
	return profileFromRank(best)
}

func profileRank(p string) int {
	switch p {
	case "pastor":
		return 4
	case "leadership":
		return 3
	case "musician":
		return 2
	default:
		return 1
	}
}

func profileFromRank(rank int) string {
	switch rank {
	case 4:
		return "pastor"
	case 3:
		return "leadership"
	case 2:
		return "musician"
	default:
		return "member"
	}
}
