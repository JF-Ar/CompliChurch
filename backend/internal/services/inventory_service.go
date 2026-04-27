package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

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
