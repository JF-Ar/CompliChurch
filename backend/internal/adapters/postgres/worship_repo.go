package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jf-ar/compli-church/internal/ports"
)

type WorshipRepo struct {
	pool *pgxpool.Pool
}

func NewWorshipRepo(pool *pgxpool.Pool) *WorshipRepo {
	return &WorshipRepo{pool: pool}
}

var _ ports.WorshipRepository = (*WorshipRepo)(nil)

// ── scan helpers ──────────────────────────────────────────────────────────────

func scanException(row rowScanner) (*ports.AvailabilityException, error) {
	var e ports.AvailabilityException
	var date pgtype.Date
	var reason pgtype.Text
	var createdAt pgtype.Timestamptz
	if err := row.Scan(&e.ID, &e.MemberID, &e.ChurchID, &date, &reason, &createdAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if date.Valid {
		e.UnavailableDate = date.Time.Format("2006-01-02")
	}
	if reason.Valid {
		e.Reason = &reason.String
	}
	if createdAt.Valid {
		e.CreatedAt = createdAt.Time
	}
	return &e, nil
}

func scanSlot(row rowScanner) (*ports.ScheduleSlot, error) {
	var slot ports.ScheduleSlot
	var notifiedAt pgtype.Timestamptz
	var instrID pgtype.UUID
	var instrChurchID pgtype.UUID
	var instrName pgtype.Text
	var instrIsSystem pgtype.Bool

	err := row.Scan(
		&slot.ID, &slot.ScheduleID, &slot.FunctionInScale, &slot.Confirmed, &notifiedAt,
		&slot.Member.ID, &slot.Member.Name, &slot.Member.Email, &slot.Member.IsActive,
		&instrID, &instrChurchID, &instrName, &instrIsSystem,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	if notifiedAt.Valid {
		slot.NotifiedAt = &notifiedAt.Time
	}
	if instrID.Valid {
		var churchID *uuid.UUID
		if instrChurchID.Valid {
			id := uuid.UUID(instrChurchID.Bytes)
			churchID = &id
		}
		slot.Instrument = &ports.Instrument{
			ID:       uuid.UUID(instrID.Bytes),
			ChurchID: churchID,
			Name:     instrName.String,
			IsSystem: instrIsSystem.Bool,
		}
	}
	return &slot, nil
}

// ── Exceptions ────────────────────────────────────────────────────────────────

func (r *WorshipRepo) CreateException(ctx context.Context, churchID, memberID uuid.UUID, date string, reason *string) (*ports.AvailabilityException, error) {
	var reasonText pgtype.Text
	if reason != nil {
		reasonText = pgtype.Text{String: *reason, Valid: true}
	}
	e, err := scanException(r.pool.QueryRow(ctx, `
		INSERT INTO availability_exceptions (church_id, member_id, unavailable_date, reason)
		VALUES ($1, $2, $3::date, $4)
		RETURNING id, member_id, church_id, unavailable_date, reason, created_at`,
		churchID, memberID, date, reasonText,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return e, nil
}

func (r *WorshipRepo) DeleteException(ctx context.Context, id, memberID, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM availability_exceptions WHERE id = $1 AND member_id = $2 AND church_id = $3`,
		id, memberID, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *WorshipRepo) ListMyExceptions(ctx context.Context, memberID, churchID uuid.UUID, month string) ([]*ports.AvailabilityException, error) {
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, member_id, church_id, unavailable_date, reason, created_at
		FROM availability_exceptions
		WHERE member_id = $1 AND church_id = $2
		  AND unavailable_date >= $3::date AND unavailable_date < $4::date
		ORDER BY unavailable_date`,
		memberID, churchID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExceptions(rows)
}

func (r *WorshipRepo) ListAllExceptions(ctx context.Context, churchID uuid.UUID, month string) ([]*ports.AvailabilityExceptionWithMember, error) {
	start, end, err := monthRange(month)
	if err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT ae.id, ae.unavailable_date, ae.reason,
		       m.id, m.name, m.email, m.is_active
		FROM availability_exceptions ae
		JOIN members m ON ae.member_id = m.id
		WHERE ae.church_id = $1
		  AND ae.unavailable_date >= $2::date AND ae.unavailable_date < $3::date
		ORDER BY ae.unavailable_date, m.name`,
		churchID, start, end,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*ports.AvailabilityExceptionWithMember
	for rows.Next() {
		var ex ports.AvailabilityExceptionWithMember
		var date pgtype.Date
		var reason pgtype.Text
		if err := rows.Scan(&ex.ID, &date, &reason, &ex.Member.ID, &ex.Member.Name, &ex.Member.Email, &ex.Member.IsActive); err != nil {
			return nil, err
		}
		if date.Valid {
			ex.UnavailableDate = date.Time.Format("2006-01-02")
		}
		if reason.Valid {
			ex.Reason = &reason.String
		}
		result = append(result, &ex)
	}
	return result, rows.Err()
}

// ── Schedules ─────────────────────────────────────────────────────────────────

func (r *WorshipRepo) CreateSchedule(ctx context.Context, churchID, createdBy uuid.UUID, sundayDate string, notes *string) (*ports.Schedule, error) {
	var notesText pgtype.Text
	if notes != nil {
		notesText = pgtype.Text{String: *notes, Valid: true}
	}
	var sched ports.Schedule
	var date pgtype.Date
	var publishedAt pgtype.Timestamptz
	var createdAt pgtype.Timestamptz
	var notesOut pgtype.Text

	err := r.pool.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO schedules (church_id, sunday_date, created_by, notes)
			VALUES ($1, $2::date, $3, $4)
			RETURNING id, church_id, sunday_date, status, notes, published_at, created_at, created_by
		)
		SELECT ins.id, ins.church_id, ins.sunday_date, ins.status, ins.notes, ins.published_at, ins.created_at,
		       m.id, m.name, m.email, m.is_active
		FROM ins
		JOIN members m ON ins.created_by = m.id`,
		churchID, sundayDate, createdBy, notesText,
	).Scan(
		&sched.ID, &sched.ChurchID, &date, &sched.Status, &notesOut, &publishedAt, &createdAt,
		&sched.CreatedBy.ID, &sched.CreatedBy.Name, &sched.CreatedBy.Email, &sched.CreatedBy.IsActive,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	if date.Valid {
		sched.SundayDate = date.Time.Format("2006-01-02")
	}
	if notesOut.Valid {
		sched.Notes = &notesOut.String
	}
	if publishedAt.Valid {
		sched.PublishedAt = &publishedAt.Time
	}
	if createdAt.Valid {
		sched.CreatedAt = createdAt.Time
	}
	sched.Slots = []ports.ScheduleSlot{}
	return &sched, nil
}

func (r *WorshipRepo) GetSchedule(ctx context.Context, id, churchID uuid.UUID) (*ports.Schedule, error) {
	return r.fetchSchedule(ctx, id, churchID)
}

func (r *WorshipRepo) ListSchedules(ctx context.Context, churchID uuid.UUID, status *string, page, perPage int) ([]*ports.ScheduleSummary, int, error) {
	args := []any{churchID}
	n := 1
	where := `s.church_id = $1`
	if status != nil {
		n++
		where += fmt.Sprintf(` AND s.status = $%d`, n)
		args = append(args, *status)
	}

	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM schedules s WHERE `+where, args...,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	n++
	limitArg := n
	n++
	offsetArg := n
	args = append(args, perPage, (page-1)*perPage)

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT s.id, s.sunday_date, s.status, s.published_at, COUNT(ss.id) AS slot_count
		FROM schedules s
		LEFT JOIN schedule_slots ss ON ss.schedule_id = s.id
		WHERE %s
		GROUP BY s.id
		ORDER BY s.sunday_date DESC
		LIMIT $%d OFFSET $%d`, where, limitArg, offsetArg),
		args...,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var result []*ports.ScheduleSummary
	for rows.Next() {
		var s ports.ScheduleSummary
		var date pgtype.Date
		var publishedAt pgtype.Timestamptz
		if err := rows.Scan(&s.ID, &date, &s.Status, &publishedAt, &s.SlotCount); err != nil {
			return nil, 0, err
		}
		if date.Valid {
			s.SundayDate = date.Time.Format("2006-01-02")
		}
		if publishedAt.Valid {
			s.PublishedAt = &publishedAt.Time
		}
		result = append(result, &s)
	}
	return result, total, rows.Err()
}

func (r *WorshipRepo) UpdateSchedule(ctx context.Context, id, churchID uuid.UUID, notes *string) (*ports.Schedule, error) {
	var notesText pgtype.Text
	if notes != nil {
		notesText = pgtype.Text{String: *notes, Valid: true}
	}
	tag, err := r.pool.Exec(ctx,
		`UPDATE schedules SET notes = $1, updated_at = NOW() WHERE id = $2 AND church_id = $3`,
		notesText, id, churchID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}
	return r.fetchSchedule(ctx, id, churchID)
}

func (r *WorshipRepo) CancelSchedule(ctx context.Context, id, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE schedules SET status = 'cancelled', updated_at = NOW() WHERE id = $1 AND church_id = $2`,
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

func (r *WorshipRepo) PublishSchedule(ctx context.Context, id, churchID uuid.UUID) (*ports.Schedule, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	tag, err := tx.Exec(ctx,
		`UPDATE schedules SET status = 'published', published_at = NOW(), updated_at = NOW()
		 WHERE id = $1 AND church_id = $2 AND status = 'draft'`,
		id, churchID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}

	if _, err = tx.Exec(ctx,
		`UPDATE schedule_slots SET notified_at = NOW() WHERE schedule_id = $1 AND notified_at IS NULL`,
		id,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.fetchSchedule(ctx, id, churchID)
}

// ── Slots ─────────────────────────────────────────────────────────────────────

func (r *WorshipRepo) ListSlots(ctx context.Context, scheduleID, churchID uuid.UUID) ([]*ports.ScheduleSlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ss.id, ss.schedule_id, ss.function_in_scale, ss.confirmed, ss.notified_at,
		       m.id, m.name, m.email, m.is_active,
		       i.id, i.church_id, i.name, i.is_system
		FROM schedule_slots ss
		JOIN schedules s ON ss.schedule_id = s.id AND s.church_id = $2
		JOIN members m ON ss.member_id = m.id
		LEFT JOIN instruments i ON ss.instrument_id = i.id
		WHERE ss.schedule_id = $1
		ORDER BY ss.id`,
		scheduleID, churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSlots(rows)
}

func (r *WorshipRepo) AddSlot(ctx context.Context, scheduleID, churchID, memberID uuid.UUID, instrumentID *uuid.UUID, functionInScale string) (*ports.ScheduleSlot, error) {
	var instrID pgtype.UUID
	if instrumentID != nil {
		instrID = pgtype.UUID{Bytes: *instrumentID, Valid: true}
	}
	slot, err := scanSlot(r.pool.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO schedule_slots (schedule_id, member_id, instrument_id, function_in_scale)
			VALUES ($1, $2, $3, $4)
			RETURNING id, schedule_id, member_id, instrument_id, function_in_scale, confirmed, notified_at
		)
		SELECT ins.id, ins.schedule_id, ins.function_in_scale, ins.confirmed, ins.notified_at,
		       m.id, m.name, m.email, m.is_active,
		       i.id, i.church_id, i.name, i.is_system
		FROM ins
		JOIN members m ON ins.member_id = m.id
		LEFT JOIN instruments i ON ins.instrument_id = i.id`,
		scheduleID, memberID, instrID, functionInScale,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return slot, nil
}

func (r *WorshipRepo) RemoveSlot(ctx context.Context, slotID, scheduleID, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM schedule_slots
		WHERE id = $1 AND schedule_id = $2
		  AND schedule_id IN (SELECT id FROM schedules WHERE id = $2 AND church_id = $3)`,
		slotID, scheduleID, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *WorshipRepo) ConfirmSlot(ctx context.Context, slotID, scheduleID, memberID uuid.UUID) (*ports.ScheduleSlot, error) {
	slot, err := scanSlot(r.pool.QueryRow(ctx, `
		WITH upd AS (
			UPDATE schedule_slots SET confirmed = TRUE
			WHERE id = $1 AND schedule_id = $2 AND member_id = $3
			RETURNING id, schedule_id, member_id, instrument_id, function_in_scale, confirmed, notified_at
		)
		SELECT upd.id, upd.schedule_id, upd.function_in_scale, upd.confirmed, upd.notified_at,
		       m.id, m.name, m.email, m.is_active,
		       i.id, i.church_id, i.name, i.is_system
		FROM upd
		JOIN members m ON upd.member_id = m.id
		LEFT JOIN instruments i ON upd.instrument_id = i.id`,
		slotID, scheduleID, memberID,
	))
	if err != nil {
		return nil, err
	}
	return slot, nil
}

// ── Suggestion helpers ────────────────────────────────────────────────────────

func (r *WorshipRepo) ListMusiciansInChurch(ctx context.Context, churchID uuid.UUID) ([]*ports.MemberWithInstruments, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT m.id, m.name, m.email, m.is_active
		FROM members m
		JOIN member_church_memberships mcm ON m.id = mcm.member_id
		  AND mcm.church_id = $1 AND mcm.left_at IS NULL
		JOIN member_role_assignments mra ON mra.membership_id = mcm.id
		JOIN roles r ON mra.role_id = r.id
		WHERE r.base_profile = 'musician' AND m.is_active = TRUE
		ORDER BY m.name`,
		churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*ports.MemberWithInstruments
	var ids []uuid.UUID
	for rows.Next() {
		var mwi ports.MemberWithInstruments
		if err := rows.Scan(&mwi.Member.ID, &mwi.Member.Name, &mwi.Member.Email, &mwi.Member.IsActive); err != nil {
			return nil, err
		}
		members = append(members, &mwi)
		ids = append(ids, mwi.Member.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return members, nil
	}

	// Batch-fetch instruments for all musicians in one query (avoid N+1).
	irows, err := r.pool.Query(ctx, `
		SELECT mi.member_id, i.id, i.church_id, i.name, i.is_system, mi.is_primary
		FROM member_instruments mi
		JOIN instruments i ON mi.instrument_id = i.id
		WHERE mi.member_id = ANY($1)
		ORDER BY mi.member_id, mi.is_primary DESC, i.name`,
		ids,
	)
	if err != nil {
		return nil, err
	}
	defer irows.Close()

	// Build lookup maps.
	primaryMap := make(map[uuid.UUID]*ports.Instrument)
	allMap := make(map[uuid.UUID][]ports.Instrument)
	for irows.Next() {
		var memberID uuid.UUID
		var instrChurchID pgtype.UUID
		var isPrimary bool
		var instr ports.Instrument
		if err := irows.Scan(&memberID, &instr.ID, &instrChurchID, &instr.Name, &instr.IsSystem, &isPrimary); err != nil {
			return nil, err
		}
		if instrChurchID.Valid {
			id := uuid.UUID(instrChurchID.Bytes)
			instr.ChurchID = &id
		}
		allMap[memberID] = append(allMap[memberID], instr)
		if isPrimary {
			cp := instr
			primaryMap[memberID] = &cp
		}
	}
	if err := irows.Err(); err != nil {
		return nil, err
	}

	for _, mwi := range members {
		mwi.Instruments = allMap[mwi.Member.ID]
		mwi.PrimaryInstrument = primaryMap[mwi.Member.ID]
	}
	return members, nil
}

func (r *WorshipRepo) ListExceptionsForDate(ctx context.Context, churchID uuid.UUID, date string) ([]*ports.AvailabilityException, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, member_id, church_id, unavailable_date, reason, created_at
		FROM availability_exceptions
		WHERE church_id = $1 AND unavailable_date = $2::date`,
		churchID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectExceptions(rows)
}

func (r *WorshipRepo) ListSlotsForDate(ctx context.Context, churchID uuid.UUID, date string) ([]*ports.ScheduleSlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ss.id, ss.schedule_id, ss.function_in_scale, ss.confirmed, ss.notified_at,
		       m.id, m.name, m.email, m.is_active,
		       i.id, i.church_id, i.name, i.is_system
		FROM schedule_slots ss
		JOIN schedules s ON ss.schedule_id = s.id
		JOIN members m ON ss.member_id = m.id
		LEFT JOIN instruments i ON ss.instrument_id = i.id
		WHERE s.church_id = $1 AND s.sunday_date = $2::date AND s.status != 'cancelled'
		ORDER BY ss.id`,
		churchID, date,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectSlots(rows)
}

func (r *WorshipRepo) CountSlotsInLastNWeeks(ctx context.Context, churchID, memberID uuid.UUID, weeks int) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM schedule_slots ss
		JOIN schedules s ON ss.schedule_id = s.id
		WHERE s.church_id = $1 AND ss.member_id = $2
		  AND s.sunday_date >= CURRENT_DATE - ($3::int * INTERVAL '7 days')
		  AND s.status != 'cancelled'`,
		churchID, memberID, weeks,
	).Scan(&count)
	return count, err
}

// ── private helpers ───────────────────────────────────────────────────────────

func (r *WorshipRepo) fetchSchedule(ctx context.Context, id, churchID uuid.UUID) (*ports.Schedule, error) {
	var sched ports.Schedule
	var date pgtype.Date
	var publishedAt pgtype.Timestamptz
	var createdAt pgtype.Timestamptz
	var notes pgtype.Text
	var abID pgtype.UUID
	var abName, abEmail pgtype.Text
	var abIsActive pgtype.Bool

	err := r.pool.QueryRow(ctx, `
		SELECT s.id, s.church_id, s.sunday_date, s.status, s.notes, s.published_at, s.created_at,
		       cb.id, cb.name, cb.email, cb.is_active,
		       ab.id, ab.name, ab.email, ab.is_active
		FROM schedules s
		JOIN members cb ON s.created_by = cb.id
		LEFT JOIN members ab ON s.approved_by = ab.id
		WHERE s.id = $1 AND s.church_id = $2`,
		id, churchID,
	).Scan(
		&sched.ID, &sched.ChurchID, &date, &sched.Status, &notes, &publishedAt, &createdAt,
		&sched.CreatedBy.ID, &sched.CreatedBy.Name, &sched.CreatedBy.Email, &sched.CreatedBy.IsActive,
		&abID, &abName, &abEmail, &abIsActive,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	if date.Valid {
		sched.SundayDate = date.Time.Format("2006-01-02")
	}
	if notes.Valid {
		sched.Notes = &notes.String
	}
	if publishedAt.Valid {
		sched.PublishedAt = &publishedAt.Time
	}
	if createdAt.Valid {
		sched.CreatedAt = createdAt.Time
	}
	if abID.Valid {
		sched.ApprovedBy = &ports.MemberSummary{
			ID:       uuid.UUID(abID.Bytes),
			Name:     abName.String,
			Email:    abEmail.String,
			IsActive: abIsActive.Bool,
		}
	}

	slots, err := r.fetchSlots(ctx, id)
	if err != nil {
		return nil, err
	}
	sched.Slots = slots
	return &sched, nil
}

func (r *WorshipRepo) fetchSlots(ctx context.Context, scheduleID uuid.UUID) ([]ports.ScheduleSlot, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ss.id, ss.schedule_id, ss.function_in_scale, ss.confirmed, ss.notified_at,
		       m.id, m.name, m.email, m.is_active,
		       i.id, i.church_id, i.name, i.is_system
		FROM schedule_slots ss
		JOIN members m ON ss.member_id = m.id
		LEFT JOIN instruments i ON ss.instrument_id = i.id
		WHERE ss.schedule_id = $1
		ORDER BY ss.id`,
		scheduleID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var slots []ports.ScheduleSlot
	for rows.Next() {
		slot, err := scanSlot(rows)
		if err != nil {
			return nil, err
		}
		slots = append(slots, *slot)
	}
	if slots == nil {
		slots = []ports.ScheduleSlot{}
	}
	return slots, rows.Err()
}

func collectExceptions(rows pgx.Rows) ([]*ports.AvailabilityException, error) {
	var result []*ports.AvailabilityException
	for rows.Next() {
		e, err := scanException(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, rows.Err()
}

func collectSlots(rows pgx.Rows) ([]*ports.ScheduleSlot, error) {
	var result []*ports.ScheduleSlot
	for rows.Next() {
		slot, err := scanSlot(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, slot)
	}
	return result, rows.Err()
}

// monthRange returns [start, end) dates for the given "YYYY-MM" string.
// If month is empty, defaults to current month.
func monthRange(month string) (string, string, error) {
	var t time.Time
	if month == "" {
		now := time.Now().UTC()
		t = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else {
		var err error
		t, err = time.Parse("2006-01", month)
		if err != nil {
			return "", "", fmt.Errorf("invalid month format (expected YYYY-MM): %w", err)
		}
	}
	start := t.Format("2006-01-02")
	end := t.AddDate(0, 1, 0).Format("2006-01-02")
	return start, end, nil
}
