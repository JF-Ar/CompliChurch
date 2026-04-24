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

type MemberHandler struct {
	svc *services.MemberService
}

func NewMemberHandler(svc *services.MemberService) *MemberHandler {
	return &MemberHandler{svc: svc}
}

// GET /members
func (h *MemberHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	filter := ports.ListMembersFilter{Page: page, PerPage: perPage}

	if s := r.URL.Query().Get("search"); s != "" {
		filter.Search = &s
	}
	if role := r.URL.Query().Get("role"); role != "" {
		filter.Role = &role
	}
	if s := r.URL.Query().Get("is_active"); s != "" {
		v := s == "true"
		filter.IsActive = &v
	}

	members, total, err := h.svc.ListMembers(r.Context(), auth.ChurchID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]memberResponse, len(members))
	for i, m := range members {
		m := m
		data[i] = buildMemberResponse(&m)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"meta": map[string]int{"total": total, "page": page, "per_page": perPage},
	})
}

// POST /members
func (h *MemberHandler) CreateMember(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		Name      string    `json:"name"`
		Email     string    `json:"email"`
		Phone     *string   `json:"phone"`
		BirthDate *string   `json:"birth_date"`
		RoleIDs   []string  `json:"role_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" || body.Email == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and email are required", "")
		return
	}

	input := ports.MemberCreateInput{
		Name:  body.Name,
		Email: body.Email,
		Phone: body.Phone,
	}
	if body.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *body.BirthDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "birth_date must be YYYY-MM-DD", "birth_date")
			return
		}
		input.BirthDate = &t
	}
	for _, idStr := range body.RoleIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role_id: "+idStr, "role_ids")
			return
		}
		input.RoleIDs = append(input.RoleIDs, id)
	}

	member, err := h.svc.CreateMember(r.Context(), auth.ChurchID, auth.MemberID, input)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMemberEmailExists):
			writeError(w, http.StatusConflict, "MEMBER_EMAIL_EXISTS", "A member with this email already exists", "email")
		case errors.Is(err, services.ErrRoleNotFound):
			writeError(w, http.StatusBadRequest, "ROLE_NOT_FOUND", "One or more roles were not found", "role_ids")
		case errors.Is(err, services.ErrRoleAccessDenied):
			writeError(w, http.StatusBadRequest, "ROLE_ACCESS_DENIED", "One or more roles do not belong to this church", "role_ids")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	writeJSON(w, http.StatusCreated, buildMemberResponse(member))
}

// POST /members/import
func (h *MemberHandler) ImportMembers(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		Members []struct {
			Name  string  `json:"name"`
			Email string  `json:"email"`
			Phone *string `json:"phone"`
		} `json:"members"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}

	rows := make([]ports.ImportRow, len(body.Members))
	for i, m := range body.Members {
		if m.Name == "" || m.Email == "" {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"each member must have name and email", "members")
			return
		}
		rows[i] = ports.ImportRow{Name: m.Name, Email: m.Email, Phone: m.Phone}
	}

	result := h.svc.ImportMembers(r.Context(), auth.ChurchID, auth.MemberID, rows)
	writeJSON(w, http.StatusOK, map[string]any{
		"created": result.Created,
		"skipped": result.Skipped,
		"errors":  result.Errors,
	})
}

// GET /members/me
func (h *MemberHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	member, err := h.svc.GetMemberByID(r.Context(), auth.MemberID, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	writeJSON(w, http.StatusOK, buildMemberResponse(member))
}

// PUT /members/me
func (h *MemberHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())
	h.updateMember(w, r, auth.MemberID, auth.ChurchID)
}

// GET /members/me/instruments
func (h *MemberHandler) GetMyInstruments(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	instruments, err := h.svc.GetMemberInstruments(r.Context(), auth.MemberID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]instrumentResponse, len(instruments))
	for i, inst := range instruments {
		data[i] = instrumentResponse{
			ID:             inst.ID,
			InstrumentID:   inst.InstrumentID,
			InstrumentName: inst.InstrumentName,
			IsPrimary:      inst.IsPrimary,
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /members/me/instruments
func (h *MemberHandler) AddMyInstrument(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		InstrumentID string `json:"instrument_id"`
		IsPrimary    bool   `json:"is_primary"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	instrumentID, err := uuid.Parse(body.InstrumentID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid instrument_id", "instrument_id")
		return
	}

	inst, err := h.svc.AddMemberInstrument(r.Context(), auth.MemberID, instrumentID, body.IsPrimary)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrInstrumentNotFound):
			writeError(w, http.StatusNotFound, "INSTRUMENT_NOT_FOUND", "Instrument not found", "instrument_id")
		case errors.Is(err, services.ErrInstrumentAlreadyAdded):
			writeError(w, http.StatusConflict, "INSTRUMENT_ALREADY_ADDED", "Instrument already in your profile", "instrument_id")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	writeJSON(w, http.StatusCreated, instrumentResponse{
		ID:             inst.ID,
		InstrumentID:   inst.InstrumentID,
		InstrumentName: inst.InstrumentName,
		IsPrimary:      inst.IsPrimary,
	})
}

// DELETE /members/me/instruments/{instrument_id}
func (h *MemberHandler) RemoveMyInstrument(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	instrumentID, err := uuid.Parse(chi.URLParam(r, "instrument_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid instrument_id", "instrument_id")
		return
	}

	if err := h.svc.RemoveMemberInstrument(r.Context(), auth.MemberID, instrumentID); err != nil {
		if errors.Is(err, services.ErrInstrumentNotInProfile) {
			writeError(w, http.StatusNotFound, "INSTRUMENT_NOT_FOUND", "Instrument not in your profile", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /members/{id}
func (h *MemberHandler) GetMemberByID(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}

	member, err := h.svc.GetMemberByID(r.Context(), id, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	writeJSON(w, http.StatusOK, buildMemberResponse(member))
}

// PUT /members/{id}
func (h *MemberHandler) UpdateMemberByID(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}
	h.updateMember(w, r, id, auth.ChurchID)
}

// DELETE /members/{id}
func (h *MemberHandler) DeactivateMember(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}

	if err := h.svc.DeactivateMember(r.Context(), id, auth.ChurchID); err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GET /members/{id}/roles
func (h *MemberHandler) GetMemberRoles(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}

	roles, err := h.svc.GetMemberRoles(r.Context(), id, auth.ChurchID)
	if err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]roleSummaryResponse, len(roles))
	for i, role := range roles {
		data[i] = roleSummaryResponse{ID: role.ID, Name: role.Name, BaseProfile: role.BaseProfile}
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /members/{id}/roles
func (h *MemberHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}

	var body struct {
		RoleID string `json:"role_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	roleID, err := uuid.Parse(body.RoleID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role_id", "role_id")
		return
	}

	if err := h.svc.AssignRole(r.Context(), id, auth.ChurchID, roleID, auth.MemberID); err != nil {
		switch {
		case errors.Is(err, services.ErrMemberNotFound):
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
		case errors.Is(err, services.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found", "role_id")
		case errors.Is(err, services.ErrRoleAccessDenied):
			writeError(w, http.StatusForbidden, "ROLE_ACCESS_DENIED", "Role does not belong to this church", "role_id")
		case errors.Is(err, services.ErrRoleAlreadyAssigned):
			writeError(w, http.StatusConflict, "ROLE_ALREADY_ASSIGNED", "Role already assigned to this member", "role_id")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// DELETE /members/{id}/roles/{role_id}
func (h *MemberHandler) RemoveMemberRole(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid member id", "id")
		return
	}
	roleID, err := uuid.Parse(chi.URLParam(r, "role_id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role_id", "role_id")
		return
	}

	if err := h.svc.RemoveRole(r.Context(), id, roleID, auth.ChurchID); err != nil {
		if errors.Is(err, services.ErrRoleNotAssigned) {
			writeError(w, http.StatusNotFound, "ROLE_NOT_FOUND", "Role not assigned to this member", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── shared helpers ────────────────────────────────────────────────────────────

func (h *MemberHandler) updateMember(w http.ResponseWriter, r *http.Request, id, churchID uuid.UUID) {
	var body struct {
		Name      string  `json:"name"`
		Phone     *string `json:"phone"`
		BirthDate *string `json:"birth_date"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name is required", "name")
		return
	}

	input := ports.MemberUpdateInput{
		Name:  body.Name,
		Phone: body.Phone,
	}
	if body.BirthDate != nil {
		t, err := time.Parse("2006-01-02", *body.BirthDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "birth_date must be YYYY-MM-DD", "birth_date")
			return
		}
		input.BirthDate = &t
	}

	member, err := h.svc.UpdateMember(r.Context(), id, churchID, input)
	if err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "Member not found", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	writeJSON(w, http.StatusOK, buildMemberResponse(member))
}

func buildMemberResponse(m *ports.Member) memberResponse {
	roles := make([]roleSummaryResponse, len(m.Roles))
	for i, r := range m.Roles {
		roles[i] = roleSummaryResponse{ID: r.ID, Name: r.Name, BaseProfile: r.BaseProfile}
	}
	instruments := make([]instrumentResponse, len(m.Instruments))
	for i, inst := range m.Instruments {
		instruments[i] = instrumentResponse{
			ID:             inst.ID,
			InstrumentID:   inst.InstrumentID,
			InstrumentName: inst.InstrumentName,
			IsPrimary:      inst.IsPrimary,
		}
	}
	var birthDateStr *string
	if m.BirthDate != nil {
		s := m.BirthDate.Format("2006-01-02")
		birthDateStr = &s
	}
	return memberResponse{
		ID:          m.ID,
		Name:        m.Name,
		Email:       m.Email,
		Phone:       m.Phone,
		BirthDate:   birthDateStr,
		AvatarURL:   m.AvatarURL,
		IsActive:    m.IsActive,
		Roles:       roles,
		Instruments: instruments,
		CreatedAt:   m.CreatedAt,
	}
}
