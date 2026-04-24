package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/ports"
	"github.com/jf-ar/compli-church/internal/services"
)

type RoleHandler struct {
	svc *services.MemberService
}

func NewRoleHandler(svc *services.MemberService) *RoleHandler {
	return &RoleHandler{svc: svc}
}

type roleFullResponse struct {
	ID          uuid.UUID  `json:"id"`
	ChurchID    *uuid.UUID `json:"church_id"`
	Name        string     `json:"name"`
	BaseProfile string     `json:"base_profile"`
	IsSystem    bool       `json:"is_system"`
	CreatedAt   *time.Time `json:"created_at,omitempty"`
}

// GET /roles
func (h *RoleHandler) ListRoles(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	roles, err := h.svc.ListRoles(r.Context(), auth.ChurchID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	data := make([]roleFullResponse, len(roles))
	for i, role := range roles {
		data[i] = toRoleFullResponse(&role)
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

// POST /roles
func (h *RoleHandler) CreateRole(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	var body struct {
		Name        string `json:"name"`
		BaseProfile string `json:"base_profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" || body.BaseProfile == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and base_profile are required", "")
		return
	}
	if !validBaseProfile(body.BaseProfile) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "base_profile must be one of: pastor, leadership, musician, member", "base_profile")
		return
	}

	role, err := h.svc.CreateRole(r.Context(), auth.ChurchID, body.Name, body.BaseProfile)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRoleAlreadyAssigned):
			writeError(w, http.StatusConflict, "ROLE_NAME_EXISTS", "A role with this name already exists", "name")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	writeJSON(w, http.StatusCreated, toRoleFullResponse(role))
}

// PUT /roles/{id}
func (h *RoleHandler) UpdateRole(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role id", "id")
		return
	}

	var body struct {
		Name        string `json:"name"`
		BaseProfile string `json:"base_profile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Name == "" || body.BaseProfile == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name and base_profile are required", "")
		return
	}
	if !validBaseProfile(body.BaseProfile) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "base_profile must be one of: pastor, leadership, musician, member", "base_profile")
		return
	}

	role, err := h.svc.UpdateRole(r.Context(), auth.ChurchID, id, body.Name, body.BaseProfile)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found", "")
		case errors.Is(err, services.ErrSystemResource):
			writeError(w, http.StatusForbidden, "SYSTEM_RESOURCE", "System roles cannot be modified", "")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	writeJSON(w, http.StatusOK, toRoleFullResponse(role))
}

// DELETE /roles/{id}
func (h *RoleHandler) DeleteRole(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid role id", "id")
		return
	}

	if err := h.svc.DeleteRole(r.Context(), auth.ChurchID, id); err != nil {
		switch {
		case errors.Is(err, services.ErrRoleNotFound):
			writeError(w, http.StatusNotFound, "ROLE_NOT_FOUND", "Role not found", "")
		case errors.Is(err, services.ErrSystemResource):
			writeError(w, http.StatusForbidden, "SYSTEM_RESOURCE", "System roles cannot be deleted", "")
		default:
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ── helpers ───────────────────────────────────────────────────────────────────

func toRoleFullResponse(role *ports.Role) roleFullResponse {
	return roleFullResponse{
		ID:          role.ID,
		ChurchID:    role.ChurchID,
		Name:        role.Name,
		BaseProfile: role.BaseProfile,
		IsSystem:    role.IsSystem,
	}
}

func validBaseProfile(p string) bool {
	switch p {
	case "pastor", "leadership", "musician", "member":
		return true
	}
	return false
}
