package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/ports"
	"github.com/jf-ar/compli-church/internal/services"
)

type InstrumentHandler struct {
	svc *services.MemberService
}

func NewInstrumentHandler(svc *services.MemberService) *InstrumentHandler {
	return &InstrumentHandler{svc: svc}
}

type catalogInstrumentResponse struct {
	ID       uuid.UUID  `json:"id"`
	ChurchID *uuid.UUID `json:"church_id"`
	Name     string     `json:"name"`
	IsSystem bool       `json:"is_system"`
}

// GET /instruments
func (h *InstrumentHandler) ListInstruments(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	instruments, err := h.svc.ListInstruments(r.Context(), auth.ChurchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]catalogInstrumentResponse, len(instruments))
	for i, inst := range instruments {
		data[i] = toCatalogInstrumentResponse(&inst)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /instruments
func (h *InstrumentHandler) CreateInstrument(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", "name")
		return
	}

	inst, err := h.svc.CreateInstrument(r.Context(), auth.ChurchID, body.Name)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInstrumentAlreadyAdded):
			writeError(w, http.StatusConflict, "INSTRUMENT_NAME_EXISTS", "An instrument with this name already exists", "name")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toCatalogInstrumentResponse(inst))
}

// DELETE /instruments/{id}
func (h *InstrumentHandler) DeleteInstrument(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid instrument id", "id")
		return
	}

	if err := h.svc.DeleteInstrument(r.Context(), auth.ChurchID, id); err != nil {
		switch {
		case errors.Is(err, services.ErrInstrumentNotFound):
			writeError(w, http.StatusNotFound, "INSTRUMENT_NOT_FOUND", "Instrument not found", "")
		case errors.Is(err, services.ErrSystemResource):
			writeError(w, http.StatusForbidden, "SYSTEM_RESOURCE", "System instruments cannot be deleted", "")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toCatalogInstrumentResponse(inst *ports.Instrument) catalogInstrumentResponse {
	return catalogInstrumentResponse{
		ID:       inst.ID,
		ChurchID: inst.ChurchID,
		Name:     inst.Name,
		IsSystem: inst.IsSystem,
	}
}
