package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/ports"
	"github.com/jf-ar/compli-church/internal/services"
)

type InventoryHandler struct {
	svc *services.InventoryService
}

func NewInventoryHandler(svc *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{svc: svc}
}

// ── response types ────────────────────────────────────────────────────────────

type itemCategoryResponse struct {
	ID       string  `json:"id"`
	ChurchID string  `json:"church_id"`
	Name     string  `json:"name"`
	Icon     *string `json:"icon"`
}

type itemResponse struct {
	ID             string                `json:"id"`
	ChurchID       string                `json:"church_id"`
	Category       *itemCategoryResponse `json:"category"`
	ItemType       string                `json:"item_type"`
	Name           string                `json:"name"`
	Description    *string               `json:"description"`
	AssetNumber    *string               `json:"asset_number"`
	PhotoURL       *string               `json:"photo_url"`
	Location       string                `json:"location"`
	Status         string                `json:"status"`
	Quantity       int                   `json:"quantity"`
	QtyMinAlert    *int                  `json:"qty_min_alert"`
	SerialNumber   *string               `json:"serial_number"`
	Notes          *string               `json:"notes"`
	DeletedAt      *time.Time            `json:"deleted_at"`
	DeletionReason *string               `json:"deletion_reason"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type loanMemberResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type loanResponse struct {
	ID                 string              `json:"id"`
	Item               itemResponse        `json:"item"`
	RequestedBy        loanMemberResponse  `json:"requested_by"`
	ApprovedBy         *loanMemberResponse `json:"approved_by"`
	LoanToType         string              `json:"loan_to_type"`
	LoanToID           string              `json:"loan_to_id"`
	LoanToName         string              `json:"loan_to_name"`
	Status             string              `json:"status"`
	ExpectedReturnDate *string             `json:"expected_return_date"`
	ActualReturnDate   *string             `json:"actual_return_date"`
	ReturnCondition    *string             `json:"return_condition"`
	ReturnNotes        *string             `json:"return_notes"`
	CreatedAt          time.Time           `json:"created_at"`
	ReturnedAt         *time.Time          `json:"returned_at"`
}

// ── builders ──────────────────────────────────────────────────────────────────

func buildCategoryResponse(c *ports.ItemCategory) itemCategoryResponse {
	return itemCategoryResponse{
		ID:       c.ID.String(),
		ChurchID: c.ChurchID.String(),
		Name:     c.Name,
		Icon:     c.Icon,
	}
}

func buildItemResponse(item *ports.Item) itemResponse {
	r := itemResponse{
		ID:           item.ID.String(),
		ChurchID:     item.ChurchID.String(),
		ItemType:     item.ItemType,
		Name:         item.Name,
		Description:  item.Description,
		AssetNumber:  item.AssetNumber,
		PhotoURL:     item.PhotoURL,
		Location:     item.Location,
		Status:       item.Status,
		Quantity:     item.Quantity,
		QtyMinAlert:  item.QtyMinAlert,
		SerialNumber: item.SerialNumber,
		Notes:          item.Notes,
		DeletedAt:      item.DeletedAt,
		DeletionReason: item.DeletionReason,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
	if item.Category != nil {
		c := buildCategoryResponse(item.Category)
		r.Category = &c
	}
	return r
}

func buildLoanResponse(loan *ports.Loan) loanResponse {
	r := loanResponse{
		ID:          loan.ID.String(),
		Item:        buildItemResponse(&loan.Item),
		RequestedBy: loanMemberResponse{ID: loan.RequestedBy.ID.String(), Name: loan.RequestedBy.Name, Email: loan.RequestedBy.Email},
		LoanToType:  loan.LoanToType,
		LoanToID:    loan.LoanToID.String(),
		LoanToName:  loan.LoanToName,
		Status:      loan.Status,
		ReturnCondition: loan.ReturnCondition,
		ReturnNotes:     loan.ReturnNotes,
		CreatedAt:       loan.CreatedAt,
		ReturnedAt:      loan.ReturnedAt,
	}
	if loan.ApprovedBy != nil {
		m := loanMemberResponse{ID: loan.ApprovedBy.ID.String(), Name: loan.ApprovedBy.Name, Email: loan.ApprovedBy.Email}
		r.ApprovedBy = &m
	}
	if loan.ExpectedReturnDate != nil {
		s := loan.ExpectedReturnDate.Format("2006-01-02")
		r.ExpectedReturnDate = &s
	}
	if loan.ActualReturnDate != nil {
		s := loan.ActualReturnDate.Format("2006-01-02")
		r.ActualReturnDate = &s
	}
	return r
}

// ── Category handlers ─────────────────────────────────────────────────────────

// GET /inventory/categories
func (h *InventoryHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	cats, err := h.svc.ListCategories(r.Context(), auth.ChurchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]itemCategoryResponse, len(cats))
	for i := range cats {
		data[i] = buildCategoryResponse(&cats[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /inventory/categories
func (h *InventoryHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		Name string  `json:"name"`
		Icon *string `json:"icon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", "name")
		return
	}

	cat, err := h.svc.CreateCategory(r.Context(), auth.ChurchID, ports.ItemCategoryCreateInput{
		Name: body.Name,
		Icon: body.Icon,
	})
	if err != nil {
		if errors.Is(err, services.ErrCategoryNameExists) {
			writeError(w, http.StatusConflict, "CATEGORY_NAME_EXISTS", "A category with this name already exists", "name")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusCreated, buildCategoryResponse(cat))
}

// PUT /inventory/categories/{id}
func (h *InventoryHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category ID", "id")
		return
	}

	var body struct {
		Name string  `json:"name"`
		Icon *string `json:"icon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", "name")
		return
	}

	cat, err := h.svc.UpdateCategory(r.Context(), id, auth.ChurchID, ports.ItemCategoryCreateInput{
		Name: body.Name,
		Icon: body.Icon,
	})
	if err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", "Category not found", "")
			return
		}
		if errors.Is(err, services.ErrCategoryNameExists) {
			writeError(w, http.StatusConflict, "CATEGORY_NAME_EXISTS", "A category with this name already exists", "name")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildCategoryResponse(cat))
}

// DELETE /inventory/categories/{id}
func (h *InventoryHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category ID", "id")
		return
	}

	if err := h.svc.DeleteCategory(r.Context(), id, auth.ChurchID); err != nil {
		if errors.Is(err, services.ErrCategoryNotFound) {
			writeError(w, http.StatusNotFound, "CATEGORY_NOT_FOUND", "Category not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Item handlers ─────────────────────────────────────────────────────────────

// GET /inventory/items
func (h *InventoryHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	filter := ports.ListItemsFilter{Page: page, PerPage: perPage}

	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = &s
	}
	if s := r.URL.Query().Get("category_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category_id", "category_id")
			return
		}
		filter.CategoryID = &id
	}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}
	if s := r.URL.Query().Get("item_type"); s != "" {
		filter.ItemType = &s
	}
	// include_deleted only allowed for leadership+ (middleware already enforces auth,
	// but we restrict the flag to leadership here).
	if r.URL.Query().Get("include_deleted") == "true" {
		if profileLevel(auth.BaseProfile) >= profileLevel("leadership") {
			filter.IncludeDeleted = true
		}
	}

	items, total, err := h.svc.ListItems(r.Context(), auth.ChurchID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]itemResponse, len(items))
	for i := range items {
		data[i] = buildItemResponse(&items[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]int{"total": total, "page": page, "per_page": perPage},
	})
}

// POST /inventory/items
func (h *InventoryHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		ItemType     string  `json:"item_type"`
		Name         string  `json:"name"`
		Description  *string `json:"description"`
		CategoryID   *string `json:"category_id"`
		AssetNumber  *string `json:"asset_number"`
		Location     string  `json:"location"`
		Quantity     *int    `json:"quantity"`
		QtyMinAlert  *int    `json:"qty_min_alert"`
		SerialNumber *string `json:"serial_number"`
		Notes        *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.ItemType == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "item_type is required", "item_type")
		return
	}
	if body.ItemType != "asset" && body.ItemType != "consumable" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "item_type must be asset or consumable", "item_type")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", "name")
		return
	}
	if body.Location == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "location is required", "location")
		return
	}

	qty := 1
	if body.Quantity != nil {
		qty = *body.Quantity
	}

	input := ports.ItemCreateInput{
		ItemType:     body.ItemType,
		Name:         body.Name,
		Description:  body.Description,
		AssetNumber:  body.AssetNumber,
		Location:     body.Location,
		Quantity:     qty,
		QtyMinAlert:  body.QtyMinAlert,
		SerialNumber: body.SerialNumber,
		Notes:        body.Notes,
	}
	if body.CategoryID != nil {
		id, err := uuid.Parse(*body.CategoryID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category_id", "category_id")
			return
		}
		input.CategoryID = &id
	}

	item, err := h.svc.CreateItem(r.Context(), auth.ChurchID, input)
	if err != nil {
		if errors.Is(err, services.ErrAssetNumberExists) {
			writeError(w, http.StatusConflict, "ASSET_NUMBER_EXISTS", "Asset number already in use", "asset_number")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusCreated, buildItemResponse(item))
}

// GET /inventory/items/{id}
func (h *InventoryHandler) GetItemByID(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID", "id")
		return
	}

	item, err := h.svc.GetItemByID(r.Context(), id, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildItemResponse(item))
}

// PUT /inventory/items/{id}
func (h *InventoryHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID", "id")
		return
	}

	var body struct {
		Name         string  `json:"name"`
		Description  *string `json:"description"`
		CategoryID   *string `json:"category_id"`
		Location     string  `json:"location"`
		Status       string  `json:"status"`
		Quantity     int     `json:"quantity"`
		QtyMinAlert  *int    `json:"qty_min_alert"`
		SerialNumber *string `json:"serial_number"`
		Notes        *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", "name")
		return
	}

	input := ports.ItemUpdateInput{
		Name:         body.Name,
		Description:  body.Description,
		Location:     body.Location,
		Status:       body.Status,
		Quantity:     body.Quantity,
		QtyMinAlert:  body.QtyMinAlert,
		SerialNumber: body.SerialNumber,
		Notes:        body.Notes,
	}
	if body.CategoryID != nil {
		cid, err := uuid.Parse(*body.CategoryID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid category_id", "category_id")
			return
		}
		input.CategoryID = &cid
	}

	item, err := h.svc.UpdateItem(r.Context(), id, auth.ChurchID, input)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildItemResponse(item))
}

// POST /inventory/items/{id}/photo
func (h *InventoryHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID", "id")
		return
	}

	if err := r.ParseMultipartForm(5 << 20); err != nil { // 5 MB limit
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid multipart form or file too large (max 5MB)", "")
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "photo field is required", "photo")
		return
	}
	defer file.Close()

	photoURL, err := h.svc.UploadPhoto(r.Context(), id, auth.ChurchID, header.Filename)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"photo_url": photoURL})
}

// POST /inventory/items/{id}/discard
func (h *InventoryHandler) DiscardItem(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID", "id")
		return
	}

	if err := h.svc.DisposeItem(r.Context(), id, auth.ChurchID, "discarded"); err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "")
			return
		}
		if errors.Is(err, services.ErrItemAlreadyDeleted) {
			writeError(w, http.StatusConflict, "ITEM_ALREADY_DELETED", "Item has already been discarded or donated", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /inventory/items/{id}/donate
func (h *InventoryHandler) DonateItem(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item ID", "id")
		return
	}

	if err := h.svc.DisposeItem(r.Context(), id, auth.ChurchID, "donated"); err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "")
			return
		}
		if errors.Is(err, services.ErrItemAlreadyDeleted) {
			writeError(w, http.StatusConflict, "ITEM_ALREADY_DELETED", "Item has already been discarded or donated", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Loan handlers ─────────────────────────────────────────────────────────────

// GET /inventory/loans
func (h *InventoryHandler) ListLoans(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	filter := ports.ListLoansFilter{Page: page, PerPage: perPage}
	if s := r.URL.Query().Get("status"); s != "" {
		filter.Status = &s
	}

	loans, total, err := h.svc.ListLoans(r.Context(), auth.ChurchID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]loanResponse, len(loans))
	for i := range loans {
		data[i] = buildLoanResponse(&loans[i])
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]int{"total": total, "page": page, "per_page": perPage},
	})
}

// POST /inventory/loans
func (h *InventoryHandler) CreateLoan(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		ItemID             string  `json:"item_id"`
		LoanToType         string  `json:"loan_to_type"`
		LoanToID           string  `json:"loan_to_id"`
		ExpectedReturnDate *string `json:"expected_return_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.ItemID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "item_id is required", "item_id")
		return
	}
	if body.LoanToType != "church" && body.LoanToType != "member" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "loan_to_type must be church or member", "loan_to_type")
		return
	}
	if body.LoanToID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "loan_to_id is required", "loan_to_id")
		return
	}

	itemID, err := uuid.Parse(body.ItemID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid item_id", "item_id")
		return
	}
	loanToID, err := uuid.Parse(body.LoanToID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid loan_to_id", "loan_to_id")
		return
	}

	input := ports.LoanCreateInput{
		ItemID:     itemID,
		LoanToType: body.LoanToType,
		LoanToID:   loanToID,
	}
	if body.ExpectedReturnDate != nil {
		t, err := time.Parse("2006-01-02", *body.ExpectedReturnDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid expected_return_date format, use YYYY-MM-DD", "expected_return_date")
			return
		}
		input.ExpectedReturnDate = &t
	}

	loan, err := h.svc.CreateLoan(r.Context(), auth.ChurchID, auth.MemberID, auth.BaseProfile, input)
	if err != nil {
		if errors.Is(err, services.ErrItemNotFound) {
			writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", "Item not found", "item_id")
			return
		}
		if errors.Is(err, services.ErrItemNotAvailable) {
			writeError(w, http.StatusConflict, "ITEM_NOT_AVAILABLE", "Item is not available for loan", "item_id")
			return
		}
		if errors.Is(err, services.ErrLoanTargetNotFound) {
			writeError(w, http.StatusNotFound, "LOAN_TARGET_NOT_FOUND", "Loan target not found or not in this church", "loan_to_id")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusCreated, buildLoanResponse(loan))
}

// GET /inventory/loans/{id}
func (h *InventoryHandler) GetLoanByID(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid loan ID", "id")
		return
	}

	loan, err := h.svc.GetLoanByID(r.Context(), id, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			writeError(w, http.StatusNotFound, "LOAN_NOT_FOUND", "Loan not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildLoanResponse(loan))
}

// POST /inventory/loans/{id}/approve
func (h *InventoryHandler) ApproveLoan(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid loan ID", "id")
		return
	}

	loan, err := h.svc.ApproveLoan(r.Context(), id, auth.MemberID, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			writeError(w, http.StatusNotFound, "LOAN_NOT_FOUND", "Loan not found or not in pending status", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildLoanResponse(loan))
}

// POST /inventory/loans/{id}/reject
func (h *InventoryHandler) RejectLoan(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid loan ID", "id")
		return
	}

	loan, err := h.svc.RejectLoan(r.Context(), id, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			writeError(w, http.StatusNotFound, "LOAN_NOT_FOUND", "Loan not found or not in pending status", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildLoanResponse(loan))
}

// POST /inventory/loans/{id}/return
func (h *InventoryHandler) ReturnLoan(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid loan ID", "id")
		return
	}

	var body struct {
		ReturnCondition string  `json:"return_condition"`
		ReturnNotes     *string `json:"return_notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.ReturnCondition != "good" && body.ReturnCondition != "damaged" && body.ReturnCondition != "lost" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "return_condition must be good, damaged, or lost", "return_condition")
		return
	}

	loan, err := h.svc.ReturnLoan(r.Context(), id, auth.ChurchID, ports.LoanReturnInput{
		ReturnCondition: body.ReturnCondition,
		ReturnNotes:     body.ReturnNotes,
	})
	if err != nil {
		if errors.Is(err, services.ErrLoanNotFound) {
			writeError(w, http.StatusNotFound, "LOAN_NOT_FOUND", "Loan not found or not in active status", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}
	writeJSON(w, http.StatusOK, buildLoanResponse(loan))
}

// ── helpers ───────────────────────────────────────────────────────────────────

// profileLevel maps base_profile strings to a numeric level for comparison.
func profileLevel(profile string) int {
	switch profile {
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
