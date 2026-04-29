package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jf-ar/compli-church/internal/ports"
)

type InventoryRepo struct {
	pool *pgxpool.Pool
}

func NewInventoryRepo(pool *pgxpool.Pool) *InventoryRepo {
	return &InventoryRepo{pool: pool}
}

var _ ports.InventoryRepository = (*InventoryRepo)(nil)

// ── Categories ────────────────────────────────────────────────────────────────

func (r *InventoryRepo) ListCategories(ctx context.Context, churchID uuid.UUID) ([]ports.ItemCategory, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, church_id, name, icon FROM item_categories WHERE church_id = $1 ORDER BY name`,
		churchID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cats []ports.ItemCategory
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, err
		}
		cats = append(cats, *c)
	}
	return cats, rows.Err()
}

func (r *InventoryRepo) CreateCategory(ctx context.Context, churchID uuid.UUID, input ports.ItemCategoryCreateInput) (*ports.ItemCategory, error) {
	var icon pgtype.Text
	if input.Icon != nil {
		icon = pgtype.Text{String: *input.Icon, Valid: true}
	}

	cat, err := scanCategory(r.pool.QueryRow(ctx,
		`INSERT INTO item_categories (church_id, name, icon)
		 VALUES ($1, $2, $3)
		 RETURNING id, church_id, name, icon`,
		churchID, input.Name, icon,
	))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return cat, nil
}

func (r *InventoryRepo) GetCategoryByID(ctx context.Context, id, churchID uuid.UUID) (*ports.ItemCategory, error) {
	cat, err := scanCategory(r.pool.QueryRow(ctx,
		`SELECT id, church_id, name, icon FROM item_categories WHERE id = $1 AND church_id = $2`,
		id, churchID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return cat, nil
}

func (r *InventoryRepo) UpdateCategory(ctx context.Context, id, churchID uuid.UUID, input ports.ItemCategoryCreateInput) (*ports.ItemCategory, error) {
	var icon pgtype.Text
	if input.Icon != nil {
		icon = pgtype.Text{String: *input.Icon, Valid: true}
	}

	cat, err := scanCategory(r.pool.QueryRow(ctx,
		`UPDATE item_categories SET name = $1, icon = $2
		 WHERE id = $3 AND church_id = $4
		 RETURNING id, church_id, name, icon`,
		input.Name, icon, id, churchID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}
	return cat, nil
}

func (r *InventoryRepo) DeleteCategory(ctx context.Context, id, churchID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM item_categories WHERE id = $1 AND church_id = $2`,
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

// ── Items ─────────────────────────────────────────────────────────────────────

func (r *InventoryRepo) ListItems(ctx context.Context, churchID uuid.UUID, f ports.ListItemsFilter) ([]ports.Item, int, error) {
	args := []any{churchID}
	n := 1
	where := `i.church_id = $1`

	if !f.IncludeDeleted {
		where += ` AND i.deleted_at IS NULL`
	}
	if f.Search != nil && *f.Search != "" {
		n++
		where += fmt.Sprintf(` AND (i.name ILIKE $%d OR i.asset_number ILIKE $%d)`, n, n)
		args = append(args, "%"+*f.Search+"%")
	}
	if f.CategoryID != nil {
		n++
		where += fmt.Sprintf(` AND i.category_id = $%d`, n)
		args = append(args, *f.CategoryID)
	}
	if f.Status != nil && *f.Status != "" {
		n++
		where += fmt.Sprintf(` AND i.status = $%d`, n)
		args = append(args, *f.Status)
	}
	if f.ItemType != nil && *f.ItemType != "" {
		n++
		where += fmt.Sprintf(` AND i.item_type = $%d`, n)
		args = append(args, *f.ItemType)
	}

	var total int
	countQ := `SELECT COUNT(*) FROM items i WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	n++
	args = append(args, f.PerPage)
	n++
	args = append(args, (f.Page-1)*f.PerPage)

	q := fmt.Sprintf(`
		SELECT i.id, i.church_id, i.category_id,
		       c.name AS cat_name, c.icon AS cat_icon,
		       i.item_type, i.name, i.description, i.asset_number, i.photo_url,
		       i.location, i.status, i.quantity, i.qty_min_alert,
		       i.serial_number, i.notes, i.deleted_at, i.deletion_reason,
		       i.created_at, i.updated_at
		FROM items i
		LEFT JOIN item_categories c ON i.category_id = c.id
		WHERE %s
		ORDER BY i.name
		LIMIT $%d OFFSET $%d`, where, n-1, n)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []ports.Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *InventoryRepo) CreateItem(ctx context.Context, churchID uuid.UUID, input ports.ItemCreateInput) (*ports.Item, error) {
	var description, assetNumber, photoURL, serialNumber, notes pgtype.Text
	var categoryID pgtype.UUID
	var qtyMinAlert pgtype.Int4

	if input.Description != nil {
		description = pgtype.Text{String: *input.Description, Valid: true}
	}
	if input.AssetNumber != nil {
		assetNumber = pgtype.Text{String: *input.AssetNumber, Valid: true}
	}
	if input.CategoryID != nil {
		categoryID = pgtype.UUID{Bytes: *input.CategoryID, Valid: true}
	}
	if input.QtyMinAlert != nil {
		qtyMinAlert = pgtype.Int4{Int32: int32(*input.QtyMinAlert), Valid: true}
	}
	if input.SerialNumber != nil {
		serialNumber = pgtype.Text{String: *input.SerialNumber, Valid: true}
	}
	if input.Notes != nil {
		notes = pgtype.Text{String: *input.Notes, Valid: true}
	}

	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`INSERT INTO items
		 (church_id, category_id, item_type, name, description, asset_number,
		  photo_url, location, status, quantity, qty_min_alert, serial_number, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'available', $9, $10, $11, $12)
		 RETURNING id`,
		churchID, categoryID, input.ItemType, input.Name, description, assetNumber,
		photoURL, input.Location, input.Quantity, qtyMinAlert, serialNumber, notes,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ports.ErrAlreadyExists
		}
		return nil, err
	}

	return r.GetItemByID(ctx, id, churchID)
}

func (r *InventoryRepo) GetItemByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Item, error) {
	item, err := scanItem(r.pool.QueryRow(ctx,
		`SELECT i.id, i.church_id, i.category_id,
		        c.name AS cat_name, c.icon AS cat_icon,
		        i.item_type, i.name, i.description, i.asset_number, i.photo_url,
		        i.location, i.status, i.quantity, i.qty_min_alert,
		        i.serial_number, i.notes, i.deleted_at, i.deletion_reason,
		        i.created_at, i.updated_at
		 FROM items i
		 LEFT JOIN item_categories c ON i.category_id = c.id
		 WHERE i.id = $1 AND i.church_id = $2`,
		id, churchID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return item, nil
}

func (r *InventoryRepo) UpdateItem(ctx context.Context, id, churchID uuid.UUID, input ports.ItemUpdateInput) (*ports.Item, error) {
	var description, serialNumber, notes pgtype.Text
	var categoryID pgtype.UUID
	var qtyMinAlert pgtype.Int4

	if input.Description != nil {
		description = pgtype.Text{String: *input.Description, Valid: true}
	}
	if input.CategoryID != nil {
		categoryID = pgtype.UUID{Bytes: *input.CategoryID, Valid: true}
	}
	if input.QtyMinAlert != nil {
		qtyMinAlert = pgtype.Int4{Int32: int32(*input.QtyMinAlert), Valid: true}
	}
	if input.SerialNumber != nil {
		serialNumber = pgtype.Text{String: *input.SerialNumber, Valid: true}
	}
	if input.Notes != nil {
		notes = pgtype.Text{String: *input.Notes, Valid: true}
	}

	tag, err := r.pool.Exec(ctx,
		`UPDATE items
		 SET name = $1, description = $2, category_id = $3,
		     location = $4, status = $5, quantity = $6,
		     qty_min_alert = $7, serial_number = $8, notes = $9, updated_at = NOW()
		 WHERE id = $10 AND church_id = $11 AND deleted_at IS NULL`,
		input.Name, description, categoryID, input.Location, input.Status,
		input.Quantity, qtyMinAlert, serialNumber, notes, id, churchID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}
	return r.GetItemByID(ctx, id, churchID)
}

func (r *InventoryRepo) UpdateItemPhotoURL(ctx context.Context, id, churchID uuid.UUID, photoURL string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE items SET photo_url = $1, updated_at = NOW()
		 WHERE id = $2 AND church_id = $3 AND deleted_at IS NULL`,
		photoURL, id, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *InventoryRepo) SoftDeleteItem(ctx context.Context, id, churchID uuid.UUID, reason string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE items SET deleted_at = NOW(), deletion_reason = $1, updated_at = NOW()
		 WHERE id = $2 AND church_id = $3 AND deleted_at IS NULL`,
		reason, id, churchID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ports.ErrNotFound
	}
	return nil
}

func (r *InventoryRepo) CountItemsWithPrefix(ctx context.Context, churchID uuid.UUID, prefix string) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM items WHERE church_id = $1 AND asset_number LIKE $2 || '-%'`,
		churchID, prefix,
	).Scan(&count)
	return count, err
}

// ── Loans ─────────────────────────────────────────────────────────────────────

func (r *InventoryRepo) ListLoans(ctx context.Context, churchID uuid.UUID, f ports.ListLoansFilter) ([]ports.Loan, int, error) {
	args := []any{churchID}
	n := 1
	where := `i.church_id = $1`

	if f.Status != nil && *f.Status != "" {
		n++
		where += fmt.Sprintf(` AND l.status = $%d`, n)
		args = append(args, *f.Status)
	}

	var total int
	countQ := `SELECT COUNT(*) FROM loans l JOIN items i ON l.item_id = i.id WHERE ` + where
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	n++
	args = append(args, f.PerPage)
	n++
	args = append(args, (f.Page-1)*f.PerPage)

	q := fmt.Sprintf(`
		SELECT %s
		FROM loans l
		JOIN items i ON l.item_id = i.id
		LEFT JOIN item_categories c ON i.category_id = c.id
		JOIN members req ON l.requested_by = req.id
		LEFT JOIN members apr ON l.approved_by = apr.id
		WHERE %s
		ORDER BY l.created_at DESC
		LIMIT $%d OFFSET $%d`, loanSelectCols, where, n-1, n)

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var loans []ports.Loan
	for rows.Next() {
		loan, err := scanLoan(rows)
		if err != nil {
			return nil, 0, err
		}
		loans = append(loans, *loan)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return loans, total, nil
}

func (r *InventoryRepo) CreateLoan(ctx context.Context, churchID uuid.UUID, requestedBy uuid.UUID, input ports.LoanCreateInput) (*ports.Loan, error) {
	var expectedReturnDate pgtype.Date
	if input.ExpectedReturnDate != nil {
		expectedReturnDate = pgtype.Date{Time: *input.ExpectedReturnDate, Valid: true}
	}

	var id uuid.UUID
	err := r.pool.QueryRow(ctx,
		`INSERT INTO loans (item_id, requested_by, loan_to_type, loan_to_id, expected_return_date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		input.ItemID, requestedBy, input.LoanToType, input.LoanToID, expectedReturnDate,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	return r.GetLoanByID(ctx, id, churchID)
}

func (r *InventoryRepo) CreateLoanActive(ctx context.Context, churchID, requestedBy, approvedBy uuid.UUID, input ports.LoanCreateInput) (*ports.Loan, error) {
	var expectedReturnDate pgtype.Date
	if input.ExpectedReturnDate != nil {
		expectedReturnDate = pgtype.Date{Time: *input.ExpectedReturnDate, Valid: true}
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var id uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO loans (item_id, requested_by, approved_by, loan_to_type, loan_to_id, expected_return_date, status)
		 VALUES ($1, $2, $3, $4, $5, $6, 'active')
		 RETURNING id`,
		input.ItemID, requestedBy, approvedBy, input.LoanToType, input.LoanToID, expectedReturnDate,
	).Scan(&id)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx,
		`UPDATE items SET status = 'on_loan', updated_at = NOW() WHERE id = $1`,
		input.ItemID,
	)
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetLoanByID(ctx, id, churchID)
}

func (r *InventoryRepo) GetLoanByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Loan, error) {
	q := fmt.Sprintf(`
		SELECT %s
		FROM loans l
		JOIN items i ON l.item_id = i.id
		LEFT JOIN item_categories c ON i.category_id = c.id
		JOIN members req ON l.requested_by = req.id
		LEFT JOIN members apr ON l.approved_by = apr.id
		WHERE l.id = $1 AND i.church_id = $2`, loanSelectCols)

	loan, err := scanLoan(r.pool.QueryRow(ctx, q, id, churchID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}
	return loan, nil
}

func (r *InventoryRepo) ApproveLoan(ctx context.Context, id, approvedBy, churchID uuid.UUID) (*ports.Loan, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var itemID uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE loans SET status = 'active', approved_by = $1
		 WHERE id = $2 AND status = 'pending'
		 RETURNING item_id`,
		approvedBy, id,
	).Scan(&itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	_, err = tx.Exec(ctx,
		`UPDATE items SET status = 'on_loan', updated_at = NOW() WHERE id = $1`,
		itemID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetLoanByID(ctx, id, churchID)
}

func (r *InventoryRepo) RejectLoan(ctx context.Context, id, churchID uuid.UUID) (*ports.Loan, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE loans SET status = 'rejected'
		 WHERE id = $1 AND status = 'pending'
		 AND item_id IN (SELECT id FROM items WHERE church_id = $2)`,
		id, churchID,
	)
	if err != nil {
		return nil, err
	}
	if tag.RowsAffected() == 0 {
		return nil, ports.ErrNotFound
	}
	return r.GetLoanByID(ctx, id, churchID)
}

func (r *InventoryRepo) ReturnLoan(ctx context.Context, id, churchID uuid.UUID, input ports.LoanReturnInput) (*ports.Loan, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var returnNotes pgtype.Text
	if input.ReturnNotes != nil {
		returnNotes = pgtype.Text{String: *input.ReturnNotes, Valid: true}
	}

	loanStatus := "returned"
	if input.ReturnCondition == "damaged" || input.ReturnCondition == "lost" {
		loanStatus = "returned_with_issue"
	}

	var itemID uuid.UUID
	err = tx.QueryRow(ctx,
		`UPDATE loans
		 SET status = $1, return_condition = $2, return_notes = $3,
		     actual_return_date = CURRENT_DATE, returned_at = NOW()
		 WHERE id = $4 AND status = 'active'
		 AND item_id IN (SELECT id FROM items WHERE church_id = $5)
		 RETURNING item_id`,
		loanStatus, input.ReturnCondition, returnNotes, id, churchID,
	).Scan(&itemID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ports.ErrNotFound
		}
		return nil, err
	}

	itemStatus := "available"
	if input.ReturnCondition == "damaged" {
		itemStatus = "damaged"
	} else if input.ReturnCondition == "lost" {
		itemStatus = "maintenance"
	}

	_, err = tx.Exec(ctx,
		`UPDATE items SET status = $1, updated_at = NOW() WHERE id = $2`,
		itemStatus, itemID,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetLoanByID(ctx, id, churchID)
}

// ── Validation helpers ────────────────────────────────────────────────────────

func (r *InventoryRepo) MemberBelongsToChurch(ctx context.Context, memberID, churchID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM member_church_memberships
			WHERE member_id = $1 AND church_id = $2 AND left_at IS NULL
		)`,
		memberID, churchID,
	).Scan(&exists)
	return exists, err
}

func (r *InventoryRepo) ChurchExists(ctx context.Context, churchID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM churches WHERE id = $1)`,
		churchID,
	).Scan(&exists)
	return exists, err
}

// ── scan helpers ──────────────────────────────────────────────────────────────

func scanCategory(row rowScanner) (*ports.ItemCategory, error) {
	var c ports.ItemCategory
	var icon pgtype.Text
	if err := row.Scan(&c.ID, &c.ChurchID, &c.Name, &icon); err != nil {
		return nil, err
	}
	if icon.Valid {
		c.Icon = &icon.String
	}
	return &c, nil
}

func scanItem(row rowScanner) (*ports.Item, error) {
	var item ports.Item
	var catID pgtype.UUID
	var catName, catIcon pgtype.Text
	var description, assetNumber, photoURL, serialNumber, notes, deletionReason pgtype.Text
	var qtyMinAlert pgtype.Int4
	var deletedAt pgtype.Timestamptz

	err := row.Scan(
		&item.ID, &item.ChurchID, &catID,
		&catName, &catIcon,
		&item.ItemType, &item.Name, &description, &assetNumber, &photoURL,
		&item.Location, &item.Status, &item.Quantity, &qtyMinAlert,
		&serialNumber, &notes, &deletedAt, &deletionReason,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if catID.Valid && catName.Valid {
		cat := &ports.ItemCategory{
			ID:       uuid.UUID(catID.Bytes),
			ChurchID: item.ChurchID,
			Name:     catName.String,
		}
		if catIcon.Valid {
			cat.Icon = &catIcon.String
		}
		item.Category = cat
	}
	if description.Valid {
		item.Description = &description.String
	}
	if assetNumber.Valid {
		item.AssetNumber = &assetNumber.String
	}
	if photoURL.Valid {
		item.PhotoURL = &photoURL.String
	}
	if qtyMinAlert.Valid {
		v := int(qtyMinAlert.Int32)
		item.QtyMinAlert = &v
	}
	if serialNumber.Valid {
		item.SerialNumber = &serialNumber.String
	}
	if notes.Valid {
		item.Notes = &notes.String
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		item.DeletedAt = &t
	}
	if deletionReason.Valid {
		item.DeletionReason = &deletionReason.String
	}
	return &item, nil
}

// loanSelectCols is the fixed column list for loan queries.
const loanSelectCols = `
	l.id,
	i.id, i.church_id, i.category_id,
	c.name AS cat_name, c.icon AS cat_icon,
	i.item_type, i.name, i.description, i.asset_number, i.photo_url,
	i.location, i.status, i.quantity, i.qty_min_alert,
	i.serial_number, i.notes, i.deleted_at, i.deletion_reason,
	i.created_at, i.updated_at,
	req.id, req.name, req.email,
	apr.id, apr.name, apr.email,
	l.loan_to_type, l.loan_to_id,
	CASE
	    WHEN l.loan_to_type = 'church' THEN (SELECT ch.name FROM churches ch WHERE ch.id = l.loan_to_id)
	    WHEN l.loan_to_type = 'member' THEN (SELECT m2.name FROM members m2 WHERE m2.id = l.loan_to_id)
	END,
	l.status, l.expected_return_date, l.actual_return_date,
	l.return_condition, l.return_notes,
	l.created_at, l.returned_at`

func scanLoan(row rowScanner) (*ports.Loan, error) {
	var loan ports.Loan
	var item ports.Item
	var catID pgtype.UUID
	var catName, catIcon pgtype.Text
	var description, assetNumber, photoURL, serialNumber, notes, deletionReason pgtype.Text
	var qtyMinAlert pgtype.Int4
	var deletedAt pgtype.Timestamptz
	var aprID pgtype.UUID
	var aprName, aprEmail pgtype.Text
	var expectedReturn, actualReturn pgtype.Date
	var returnCondition, returnNotes pgtype.Text
	var returnedAt pgtype.Timestamptz
	var loanToName pgtype.Text

	err := row.Scan(
		&loan.ID,
		&item.ID, &item.ChurchID, &catID,
		&catName, &catIcon,
		&item.ItemType, &item.Name, &description, &assetNumber, &photoURL,
		&item.Location, &item.Status, &item.Quantity, &qtyMinAlert,
		&serialNumber, &notes, &deletedAt, &deletionReason,
		&item.CreatedAt, &item.UpdatedAt,
		&loan.RequestedBy.ID, &loan.RequestedBy.Name, &loan.RequestedBy.Email,
		&aprID, &aprName, &aprEmail,
		&loan.LoanToType, &loan.LoanToID,
		&loanToName,
		&loan.Status, &expectedReturn, &actualReturn,
		&returnCondition, &returnNotes,
		&loan.CreatedAt, &returnedAt,
	)
	if err != nil {
		return nil, err
	}

	// Populate item fields
	if catID.Valid && catName.Valid {
		cat := &ports.ItemCategory{
			ID:       uuid.UUID(catID.Bytes),
			ChurchID: item.ChurchID,
			Name:     catName.String,
		}
		if catIcon.Valid {
			cat.Icon = &catIcon.String
		}
		item.Category = cat
	}
	if description.Valid {
		item.Description = &description.String
	}
	if assetNumber.Valid {
		item.AssetNumber = &assetNumber.String
	}
	if photoURL.Valid {
		item.PhotoURL = &photoURL.String
	}
	if qtyMinAlert.Valid {
		v := int(qtyMinAlert.Int32)
		item.QtyMinAlert = &v
	}
	if serialNumber.Valid {
		item.SerialNumber = &serialNumber.String
	}
	if notes.Valid {
		item.Notes = &notes.String
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		item.DeletedAt = &t
	}
	if deletionReason.Valid {
		item.DeletionReason = &deletionReason.String
	}
	loan.Item = item

	// Populate approved_by
	if aprID.Valid {
		loan.ApprovedBy = &ports.LoanMember{
			ID:    uuid.UUID(aprID.Bytes),
			Name:  aprName.String,
			Email: aprEmail.String,
		}
	}

	// Populate loan_to_name
	if loanToName.Valid {
		loan.LoanToName = loanToName.String
	}

	if expectedReturn.Valid {
		t := expectedReturn.Time
		loan.ExpectedReturnDate = &t
	}
	if actualReturn.Valid {
		t := actualReturn.Time
		loan.ActualReturnDate = &t
	}
	if returnCondition.Valid {
		loan.ReturnCondition = &returnCondition.String
	}
	if returnNotes.Valid {
		loan.ReturnNotes = &returnNotes.String
	}
	if returnedAt.Valid {
		t := returnedAt.Time
		loan.ReturnedAt = &t
	}

	return &loan, nil
}

