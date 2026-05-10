package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/jf-ar/compli-church/internal/ports"
)

var (
	ErrCategoryNotFound    = errors.New("category not found")
	ErrCategoryNameExists  = errors.New("category with this name already exists")
	ErrItemNotFound        = errors.New("item not found")
	ErrItemAlreadyDeleted  = errors.New("item has already been discarded or donated")
	ErrItemNotAvailable    = errors.New("item is not available for loan")
	ErrAssetNumberExists   = errors.New("asset number already in use")
	ErrLoanNotFound        = errors.New("loan not found")
	ErrLoanInvalidStatus   = errors.New("loan is not in the required status for this action")
	ErrLoanTargetNotFound  = errors.New("loan target (member or church) not found or not in this church")
)

var nonAlpha = regexp.MustCompile(`[^A-Z]`)

type InventoryService struct {
	repo ports.InventoryRepository
}

func NewInventoryService(repo ports.InventoryRepository) *InventoryService {
	return &InventoryService{repo: repo}
}

// ── Categories ────────────────────────────────────────────────────────────────

func (s *InventoryService) ListCategories(ctx context.Context, churchID uuid.UUID) ([]ports.ItemCategory, error) {
	return s.repo.ListCategories(ctx, churchID)
}

func (s *InventoryService) CreateCategory(ctx context.Context, churchID uuid.UUID, input ports.ItemCategoryCreateInput) (*ports.ItemCategory, error) {
	cat, err := s.repo.CreateCategory(ctx, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrCategoryNameExists
		}
		return nil, err
	}
	return cat, nil
}

func (s *InventoryService) UpdateCategory(ctx context.Context, id, churchID uuid.UUID, input ports.ItemCategoryCreateInput) (*ports.ItemCategory, error) {
	cat, err := s.repo.UpdateCategory(ctx, id, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrCategoryNotFound
		}
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrCategoryNameExists
		}
		return nil, err
	}
	return cat, nil
}

func (s *InventoryService) DeleteCategory(ctx context.Context, id, churchID uuid.UUID) error {
	err := s.repo.DeleteCategory(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrCategoryNotFound
		}
		return err
	}
	return nil
}

// ── Items ─────────────────────────────────────────────────────────────────────

func (s *InventoryService) ListItems(ctx context.Context, churchID uuid.UUID, filter ports.ListItemsFilter) ([]ports.Item, int, error) {
	return s.repo.ListItems(ctx, churchID, filter)
}

func (s *InventoryService) CreateItem(ctx context.Context, churchID uuid.UUID, input ports.ItemCreateInput) (*ports.Item, error) {
	// Auto-generate asset_number for assets when not provided.
	if input.ItemType == "asset" && (input.AssetNumber == nil || *input.AssetNumber == "") {
		prefix := s.assetPrefix(ctx, churchID, input.CategoryID)
		next, err := s.repo.CountItemsWithPrefix(ctx, churchID, prefix)
		if err != nil {
			return nil, err
		}
		num := fmt.Sprintf("%s-%03d", prefix, next+1)
		input.AssetNumber = &num
	}

	item, err := s.repo.CreateItem(ctx, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrAssetNumberExists
		}
		return nil, err
	}
	return item, nil
}

func (s *InventoryService) GetItemByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Item, error) {
	item, err := s.repo.GetItemByID(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *InventoryService) UpdateItem(ctx context.Context, id, churchID uuid.UUID, input ports.ItemUpdateInput) (*ports.Item, error) {
	item, err := s.repo.UpdateItem(ctx, id, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	return item, nil
}

func (s *InventoryService) UploadPhoto(ctx context.Context, id, churchID uuid.UUID, filename string) (string, error) {
	// Validate item exists.
	_, err := s.repo.GetItemByID(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return "", ErrItemNotFound
		}
		return "", err
	}

	// TODO: integrate Cloudflare R2 — resize/compress to max 800KB, PutObject, return pre-signed URL.
	photoURL := fmt.Sprintf("https://r2.placeholder/%s/%s.webp", churchID, id)

	if err := s.repo.UpdateItemPhotoURL(ctx, id, churchID, photoURL); err != nil {
		return "", err
	}
	return photoURL, nil
}

func (s *InventoryService) DisposeItem(ctx context.Context, id, churchID uuid.UUID, reason string) error {
	item, err := s.repo.GetItemByID(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrItemNotFound
		}
		return err
	}
	if item.DeletedAt != nil {
		return ErrItemAlreadyDeleted
	}

	if err := s.repo.SoftDeleteItem(ctx, id, churchID, reason); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrItemNotFound
		}
		return err
	}
	return nil
}

// ── Loans ─────────────────────────────────────────────────────────────────────

func (s *InventoryService) ListLoans(ctx context.Context, churchID uuid.UUID, filter ports.ListLoansFilter) ([]ports.Loan, int, error) {
	return s.repo.ListLoans(ctx, churchID, filter)
}

func (s *InventoryService) CreateLoan(ctx context.Context, churchID, requestedBy uuid.UUID, baseProfile string, input ports.LoanCreateInput) (*ports.Loan, error) {
	// Validate item exists and is available.
	item, err := s.repo.GetItemByID(ctx, input.ItemID, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrItemNotFound
		}
		return nil, err
	}
	if item.DeletedAt != nil || item.Status != "available" {
		return nil, ErrItemNotAvailable
	}

	// Validate loan target exists and belongs to the same church.
	switch input.LoanToType {
	case "member":
		ok, err := s.repo.MemberBelongsToChurch(ctx, input.LoanToID, churchID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrLoanTargetNotFound
		}
	case "church":
		ok, err := s.repo.ChurchExists(ctx, input.LoanToID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrLoanTargetNotFound
		}
	}

	// Pastor and leadership get immediate approval — no pending step needed.
	if baseProfile == "pastor" || baseProfile == "leadership" {
		return s.repo.CreateLoanActive(ctx, churchID, requestedBy, requestedBy, input)
	}

	return s.repo.CreateLoan(ctx, churchID, requestedBy, input)
}

func (s *InventoryService) GetLoanByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Loan, error) {
	loan, err := s.repo.GetLoanByID(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	return loan, nil
}

func (s *InventoryService) ApproveLoan(ctx context.Context, id, approvedBy, churchID uuid.UUID) (*ports.Loan, error) {
	loan, err := s.repo.ApproveLoan(ctx, id, approvedBy, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	return loan, nil
}

func (s *InventoryService) RejectLoan(ctx context.Context, id, churchID uuid.UUID) (*ports.Loan, error) {
	loan, err := s.repo.RejectLoan(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	return loan, nil
}

func (s *InventoryService) ReturnLoan(ctx context.Context, id, churchID uuid.UUID, input ports.LoanReturnInput) (*ports.Loan, error) {
	loan, err := s.repo.ReturnLoan(ctx, id, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrLoanNotFound
		}
		return nil, err
	}
	return loan, nil
}

// ── Import ────────────────────────────────────────────────────────────────────

func (s *InventoryService) ImportItems(ctx context.Context, churchID uuid.UUID, file io.Reader) (*ports.ItemImportResult, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("invalid xlsx file: %w", err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet: %w", err)
	}
	if len(rows) < 2 {
		return &ports.ItemImportResult{}, nil
	}

	// Map header names (case-insensitive) to column indices.
	colIdx := map[string]int{}
	for i, cell := range rows[0] {
		colIdx[strings.ToLower(strings.TrimSpace(cell))] = i
	}

	col := func(row []string, name string) string {
		idx, ok := colIdx[name]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	optStr := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}

	// Load existing categories once; create new ones on-demand.
	cats, err := s.repo.ListCategories(ctx, churchID)
	if err != nil {
		return nil, err
	}
	catMap := make(map[string]*ports.ItemCategory, len(cats))
	for i := range cats {
		catMap[strings.ToLower(strings.TrimSpace(cats[i].Name))] = &cats[i]
	}

	result := &ports.ItemImportResult{Errors: []ports.ItemImportRowError{}}

	for i, row := range rows[1:] {
		rowNum := i + 2 // 1-based; row 1 is the header

		name := col(row, "name")
		if name == "" {
			result.Skipped++
			continue
		}

		itemType, err := normalizeItemType(col(row, "item_type"))
		if err != nil {
			result.Errors = append(result.Errors, ports.ItemImportRowError{Row: rowNum, Reason: "item_type must be asset or consumable"})
			continue
		}

		location := col(row, "location")
		if location == "" {
			result.Errors = append(result.Errors, ports.ItemImportRowError{Row: rowNum, Reason: "location is required"})
			continue
		}

		// Resolve or create category.
		var categoryID *uuid.UUID
		if catName := col(row, "category"); catName != "" {
			key := strings.ToLower(catName)
			cat, found := catMap[key]
			if !found {
				newCat, err := s.repo.CreateCategory(ctx, churchID, ports.ItemCategoryCreateInput{Name: catName})
				if err != nil {
					result.Errors = append(result.Errors, ports.ItemImportRowError{Row: rowNum, Reason: "failed to resolve category: " + err.Error()})
					continue
				}
				catMap[key] = newCat
				cat = newCat
			}
			categoryID = &cat.ID
		}

		// Asset number: provided or auto-generated.
		var assetNumber *string
		if raw := col(row, "asset_number"); raw != "" {
			assetNumber = &raw
		} else if itemType == "asset" {
			prefix := s.assetPrefix(ctx, churchID, categoryID)
			next, err := s.repo.CountItemsWithPrefix(ctx, churchID, prefix)
			if err != nil {
				result.Errors = append(result.Errors, ports.ItemImportRowError{Row: rowNum, Reason: "failed to generate asset number"})
				continue
			}
			num := fmt.Sprintf("%s-%03d", prefix, next+1)
			assetNumber = &num
		}

		qty := 1
		if q := col(row, "quantity"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 {
				qty = n
			}
		}

		var qtyMinAlert *int
		if q := col(row, "qty_min_alert"); q != "" {
			if n, err := strconv.Atoi(q); err == nil {
				qtyMinAlert = &n
			}
		}

		_, err = s.repo.CreateItem(ctx, churchID, ports.ItemCreateInput{
			ItemType:     itemType,
			Name:         name,
			Description:  optStr(col(row, "description")),
			CategoryID:   categoryID,
			AssetNumber:  assetNumber,
			Location:     location,
			Quantity:     qty,
			QtyMinAlert:  qtyMinAlert,
			SerialNumber: optStr(col(row, "serial_number")),
			Notes:        optStr(col(row, "notes")),
		})
		if err != nil {
			reason := err.Error()
			if errors.Is(err, ports.ErrAlreadyExists) {
				reason = "asset_number already exists"
			}
			result.Errors = append(result.Errors, ports.ItemImportRowError{Row: rowNum, Reason: reason})
			continue
		}
		result.Created++
	}

	return result, nil
}

func normalizeItemType(s string) (string, error) {
	switch strings.ToLower(s) {
	case "asset", "bem":
		return "asset", nil
	case "consumable", "consumível", "consumivel":
		return "consumable", nil
	default:
		return "", errors.New("invalid item_type")
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

// assetPrefix derives a 4-char uppercase prefix from the category name, or "ITEM" if no category.
func (s *InventoryService) assetPrefix(ctx context.Context, churchID uuid.UUID, categoryID *uuid.UUID) string {
	if categoryID == nil {
		return "ITEM"
	}
	cat, err := s.repo.GetCategoryByID(ctx, *categoryID, churchID)
	if err != nil {
		return "ITEM"
	}
	upper := strings.ToUpper(cat.Name)
	clean := nonAlpha.ReplaceAllString(upper, "")
	if len(clean) == 0 {
		return "ITEM"
	}
	if len(clean) > 4 {
		clean = clean[:4]
	}
	return clean
}
