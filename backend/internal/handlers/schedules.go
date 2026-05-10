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

type ScheduleHandler struct {
	svc *services.ScheduleService
}

func NewScheduleHandler(svc *services.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// ── response types ────────────────────────────────────────────────────────────

type memberSummaryResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	IsActive bool      `json:"is_active"`
}

type slotResponse struct {
	ID              uuid.UUID              `json:"id"`
	ScheduleID      uuid.UUID              `json:"schedule_id"`
	Member          memberSummaryResponse  `json:"member"`
	Instrument      *catalogInstrumentResponse `json:"instrument"`
	FunctionInScale string                 `json:"function_in_scale"`
	Confirmed       bool                   `json:"confirmed"`
	NotifiedAt      *time.Time             `json:"notified_at"`
}

type scheduleResponse struct {
	ID          uuid.UUID              `json:"id"`
	ChurchID    uuid.UUID              `json:"church_id"`
	SundayDate  string                 `json:"sunday_date"`
	Status      string                 `json:"status"`
	CreatedBy   memberSummaryResponse  `json:"created_by"`
	ApprovedBy  *memberSummaryResponse `json:"approved_by"`
	Notes       *string                `json:"notes"`
	PublishedAt *time.Time             `json:"published_at"`
	Slots       []slotResponse         `json:"slots"`
	CreatedAt   time.Time              `json:"created_at"`
}

type scheduleSummaryResponse struct {
	ID          uuid.UUID  `json:"id"`
	SundayDate  string     `json:"sunday_date"`
	Status      string     `json:"status"`
	SlotCount   int        `json:"slot_count"`
	PublishedAt *time.Time `json:"published_at"`
}

type exceptionResponse struct {
	ID              uuid.UUID `json:"id"`
	MemberID        uuid.UUID `json:"member_id"`
	ChurchID        uuid.UUID `json:"church_id"`
	UnavailableDate string    `json:"unavailable_date"`
	Reason          *string   `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
}

type exceptionWithMemberResponse struct {
	ID              uuid.UUID             `json:"id"`
	Member          memberSummaryResponse `json:"member"`
	UnavailableDate string                `json:"unavailable_date"`
	Reason          *string               `json:"reason"`
}

type suggestedSlotResponse struct {
	MemberID       uuid.UUID  `json:"member_id"`
	MemberName     string     `json:"member_name"`
	InstrumentID   *uuid.UUID `json:"instrument_id"`
	InstrumentName string     `json:"instrument_name"`
	Warning        *string    `json:"warning"`
}

type unavailableMemberResponse struct {
	Member memberSummaryResponse `json:"member"`
	Reason *string               `json:"reason"`
}

type suggestionResponse struct {
	SundayDate         string                      `json:"sunday_date"`
	SuggestedSlots     []suggestedSlotResponse     `json:"suggested_slots"`
	AvailableMembers   []memberSummaryResponse     `json:"available_members"`
	UnavailableMembers []unavailableMemberResponse `json:"unavailable_members"`
}

// ── builders ──────────────────────────────────────────────────────────────────

func buildMemberSummaryResponse(m ports.MemberSummary) memberSummaryResponse {
	return memberSummaryResponse{ID: m.ID, Name: m.Name, Email: m.Email, IsActive: m.IsActive}
}

func buildSlotResponse(s ports.ScheduleSlot) slotResponse {
	resp := slotResponse{
		ID:              s.ID,
		ScheduleID:      s.ScheduleID,
		Member:          buildMemberSummaryResponse(s.Member),
		FunctionInScale: s.FunctionInScale,
		Confirmed:       s.Confirmed,
		NotifiedAt:      s.NotifiedAt,
	}
	if s.Instrument != nil {
		resp.Instrument = &catalogInstrumentResponse{
			ID:       s.Instrument.ID,
			ChurchID: s.Instrument.ChurchID,
			Name:     s.Instrument.Name,
			IsSystem: s.Instrument.IsSystem,
		}
	}
	return resp
}

func buildScheduleResponse(s *ports.Schedule) scheduleResponse {
	resp := scheduleResponse{
		ID:          s.ID,
		ChurchID:    s.ChurchID,
		SundayDate:  s.SundayDate,
		Status:      s.Status,
		CreatedBy:   buildMemberSummaryResponse(s.CreatedBy),
		Notes:       s.Notes,
		PublishedAt: s.PublishedAt,
		CreatedAt:   s.CreatedAt,
		Slots:       make([]slotResponse, len(s.Slots)),
	}
	if s.ApprovedBy != nil {
		ab := buildMemberSummaryResponse(*s.ApprovedBy)
		resp.ApprovedBy = &ab
	}
	for i, slot := range s.Slots {
		resp.Slots[i] = buildSlotResponse(slot)
	}
	return resp
}

func buildScheduleSummaryResponse(s *ports.ScheduleSummary) scheduleSummaryResponse {
	return scheduleSummaryResponse{
		ID:          s.ID,
		SundayDate:  s.SundayDate,
		Status:      s.Status,
		SlotCount:   s.SlotCount,
		PublishedAt: s.PublishedAt,
	}
}

func buildExceptionResponse(e *ports.AvailabilityException) exceptionResponse {
	return exceptionResponse{
		ID:              e.ID,
		MemberID:        e.MemberID,
		ChurchID:        e.ChurchID,
		UnavailableDate: e.UnavailableDate,
		Reason:          e.Reason,
		CreatedAt:       e.CreatedAt,
	}
}

func buildExceptionWithMemberResponse(e *ports.AvailabilityExceptionWithMember) exceptionWithMemberResponse {
	return exceptionWithMemberResponse{
		ID:              e.ID,
		Member:          buildMemberSummaryResponse(e.Member),
		UnavailableDate: e.UnavailableDate,
		Reason:          e.Reason,
	}
}

func buildSuggestionResponse(s *ports.ScheduleSuggestion) suggestionResponse {
	suggested := make([]suggestedSlotResponse, len(s.SuggestedSlots))
	for i, sl := range s.SuggestedSlots {
		suggested[i] = suggestedSlotResponse{
			MemberID:       sl.MemberID,
			MemberName:     sl.MemberName,
			InstrumentID:   sl.InstrumentID,
			InstrumentName: sl.InstrumentName,
			Warning:        sl.Warning,
		}
	}
	available := make([]memberSummaryResponse, len(s.AvailableMembers))
	for i, m := range s.AvailableMembers {
		available[i] = buildMemberSummaryResponse(m)
	}
	unavailable := make([]unavailableMemberResponse, len(s.UnavailableMembers))
	for i, u := range s.UnavailableMembers {
		unavailable[i] = unavailableMemberResponse{Member: buildMemberSummaryResponse(u.Member), Reason: u.Reason}
	}
	return suggestionResponse{
		SundayDate:         s.SundayDate,
		SuggestedSlots:     suggested,
		AvailableMembers:   available,
		UnavailableMembers: unavailable,
	}
}

// ── error mapping ─────────────────────────────────────────────────────────────

func (h *ScheduleHandler) mapScheduleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, services.ErrScheduleAlreadyExists):
		writeError(w, http.StatusConflict, "SCHEDULE_ALREADY_EXISTS", err.Error(), "sunday_date")
	case errors.Is(err, services.ErrScheduleNotFound):
		writeError(w, http.StatusNotFound, "SCHEDULE_NOT_FOUND", err.Error(), "")
	case errors.Is(err, services.ErrScheduleNotDraft):
		writeError(w, http.StatusUnprocessableEntity, "SCHEDULE_NOT_DRAFT", err.Error(), "")
	case errors.Is(err, services.ErrScheduleAlreadyCancelled):
		writeError(w, http.StatusConflict, "SCHEDULE_ALREADY_CANCELLED", err.Error(), "")
	case errors.Is(err, services.ErrSlotAlreadyExists):
		writeError(w, http.StatusConflict, "SLOT_ALREADY_EXISTS", err.Error(), "member_id")
	case errors.Is(err, services.ErrSlotNotFound):
		writeError(w, http.StatusNotFound, "SLOT_NOT_FOUND", err.Error(), "")
	case errors.Is(err, services.ErrSlotNotOwned):
		writeError(w, http.StatusForbidden, "SLOT_NOT_OWNED", err.Error(), "")
	case errors.Is(err, services.ErrExceptionAlreadyExists):
		writeError(w, http.StatusConflict, "EXCEPTION_ALREADY_EXISTS", err.Error(), "unavailable_date")
	case errors.Is(err, services.ErrExceptionNotFound):
		writeError(w, http.StatusNotFound, "EXCEPTION_NOT_FOUND", err.Error(), "")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", "")
	}
}

// ── Exceptions ────────────────────────────────────────────────────────────────

// GET /availability/exceptions
func (h *ScheduleHandler) ListMyExceptions(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())
	month := r.URL.Query().Get("month")

	exceptions, err := h.svc.ListMyExceptions(r.Context(), auth.MemberID, auth.ChurchID, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", "")
		return
	}
	data := make([]exceptionResponse, len(exceptions))
	for i, e := range exceptions {
		data[i] = buildExceptionResponse(e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /availability/exceptions
func (h *ScheduleHandler) CreateException(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		UnavailableDate string  `json:"unavailable_date"`
		Reason          *string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", "")
		return
	}
	if body.UnavailableDate == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "unavailable_date is required", "unavailable_date")
		return
	}

	e, err := h.svc.CreateException(r.Context(), auth.ChurchID, auth.MemberID, body.UnavailableDate, body.Reason)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, buildExceptionResponse(e))
}

// DELETE /availability/exceptions/{id}
func (h *ScheduleHandler) DeleteException(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid exception id", "id")
		return
	}

	if err := h.svc.DeleteException(r.Context(), id, auth.MemberID, auth.ChurchID); err != nil {
		h.mapScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /availability/exceptions/all
func (h *ScheduleHandler) ListAllExceptions(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())
	month := r.URL.Query().Get("month")

	exceptions, err := h.svc.ListAllExceptions(r.Context(), auth.ChurchID, month)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", "")
		return
	}
	data := make([]exceptionWithMemberResponse, len(exceptions))
	for i, e := range exceptions {
		data[i] = buildExceptionWithMemberResponse(e)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// ── Schedules ─────────────────────────────────────────────────────────────────

// GET /schedules
func (h *ScheduleHandler) ListSchedules(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 12
	}
	var status *string
	if s := r.URL.Query().Get("status"); s != "" {
		status = &s
	}

	schedules, total, err := h.svc.ListSchedules(r.Context(), auth.ChurchID, status, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error", "")
		return
	}
	data := make([]scheduleSummaryResponse, len(schedules))
	for i, s := range schedules {
		data[i] = buildScheduleSummaryResponse(s)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]any{
			"total":    total,
			"page":     page,
			"per_page": perPage,
		},
	})
}

// POST /schedules
func (h *ScheduleHandler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		SundayDate string  `json:"sunday_date"`
		Notes      *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", "")
		return
	}
	if body.SundayDate == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "sunday_date is required", "sunday_date")
		return
	}

	sched, err := h.svc.CreateSchedule(r.Context(), auth.ChurchID, auth.MemberID, body.SundayDate, body.Notes)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, buildScheduleResponse(sched))
}

// GET /schedules/suggest/{sunday_date}  — must be registered BEFORE /{id}
func (h *ScheduleHandler) SuggestSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())
	sundayDate := chi.URLParam(r, "sunday_date")

	suggestion, err := h.svc.GetScheduleSuggestion(r.Context(), auth.ChurchID, sundayDate)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "INVALID_DATE", err.Error(), "sunday_date")
		return
	}
	writeJSON(w, http.StatusOK, buildSuggestionResponse(suggestion))
}

// GET /schedules/{id}
func (h *ScheduleHandler) GetSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	sched, err := h.svc.GetSchedule(r.Context(), id, auth.ChurchID)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildScheduleResponse(sched))
}

// PUT /schedules/{id}
func (h *ScheduleHandler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	var body struct {
		Notes *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", "")
		return
	}

	sched, err := h.svc.UpdateSchedule(r.Context(), id, auth.ChurchID, body.Notes)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildScheduleResponse(sched))
}

// DELETE /schedules/{id}
func (h *ScheduleHandler) CancelSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	if err := h.svc.CancelSchedule(r.Context(), id, auth.ChurchID); err != nil {
		h.mapScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /schedules/{id}/publish
func (h *ScheduleHandler) PublishSchedule(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	sched, err := h.svc.PublishSchedule(r.Context(), id, auth.ChurchID)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildScheduleResponse(sched))
}

// ── Slots ─────────────────────────────────────────────────────────────────────

// GET /schedules/{id}/slots
func (h *ScheduleHandler) ListSlots(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	scheduleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	slots, err := h.svc.ListSlots(r.Context(), scheduleID, auth.ChurchID)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	data := make([]slotResponse, len(slots))
	for i, s := range slots {
		data[i] = buildSlotResponse(*s)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /schedules/{id}/slots
func (h *ScheduleHandler) AddSlot(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	scheduleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}

	var body struct {
		MemberID        uuid.UUID  `json:"member_id"`
		InstrumentID    *uuid.UUID `json:"instrument_id"`
		FunctionInScale string     `json:"function_in_scale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body", "")
		return
	}
	if body.MemberID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "member_id is required", "member_id")
		return
	}
	if body.FunctionInScale == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "function_in_scale is required", "function_in_scale")
		return
	}

	slot, err := h.svc.AddSlot(r.Context(), scheduleID, auth.ChurchID, body.MemberID, body.InstrumentID, body.FunctionInScale)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, buildSlotResponse(*slot))
}

// DELETE /schedules/{id}/slots/{slot_id}
func (h *ScheduleHandler) RemoveSlot(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	scheduleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}
	slotID, err := uuid.Parse(chi.URLParam(r, "slot_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid slot id", "slot_id")
		return
	}

	if err := h.svc.RemoveSlot(r.Context(), slotID, scheduleID, auth.ChurchID); err != nil {
		h.mapScheduleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /schedules/{id}/slots/{slot_id}/confirm
func (h *ScheduleHandler) ConfirmSlot(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	scheduleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid schedule id", "id")
		return
	}
	slotID, err := uuid.Parse(chi.URLParam(r, "slot_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid slot id", "slot_id")
		return
	}

	slot, err := h.svc.ConfirmSlot(r.Context(), slotID, scheduleID, auth.ChurchID, auth.MemberID)
	if err != nil {
		h.mapScheduleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildSlotResponse(*slot))
}
