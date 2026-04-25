package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/ports"
	"github.com/jf-ar/compli-church/internal/services"
)

const refreshCookieName = "refresh_token"

type AuthHandler struct {
	svc            *services.AuthService
	refreshTTLDays int
	secure         bool // set false in development
}

func NewAuthHandler(svc *services.AuthService, refreshTTLDays int, secure bool) *AuthHandler {
	return &AuthHandler{svc: svc, refreshTTLDays: refreshTTLDays, secure: secure}
}

// POST /auth/register  (public)
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChurchName string `json:"church_name"`
		PastorName string `json:"pastor_name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.ChurchName == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "church_name is required", "church_name")
		return
	}
	if body.PastorName == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "pastor_name is required", "pastor_name")
		return
	}
	if body.Email == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email is required", "email")
		return
	}
	if len(body.Password) < 8 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "password must be at least 8 characters", "password")
		return
	}

	result, err := h.svc.Register(r.Context(), services.RegisterInput{
		ChurchName: body.ChurchName,
		PastorName: body.PastorName,
		Email:      body.Email,
		Password:   body.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrEmailAlreadyTaken) {
			writeError(w, http.StatusConflict, "EMAIL_ALREADY_EXISTS", "An account with this email already exists", "email")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusCreated, buildLoginResponse(result.AccessToken, result.Member))
}

// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", "")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required", "")
		return
	}

	result, err := h.svc.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", "")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred", "")
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, buildLoginResponse(result.AccessToken, result.Member))
}

// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Refresh token cookie missing", "")
		return
	}

	result, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTokenRevoked):
			writeError(w, http.StatusUnauthorized, "TOKEN_REVOKED", "Session has been revoked", "")
		case errors.Is(err, services.ErrTokenExpired):
			writeError(w, http.StatusUnauthorized, "TOKEN_EXPIRED", "Refresh token has expired", "")
		default:
			writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid refresh token", "")
		}
		return
	}

	h.setRefreshCookie(w, result.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"access_token": result.AccessToken})
}

// POST /auth/logout  (requires Authenticate middleware)
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(refreshCookieName)
	if err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// POST /auth/logout-all  (requires Authenticate middleware)
func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	auth := AuthContextFromContext(r.Context())
	if auth == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Not authenticated", "")
		return
	}
	_ = h.svc.LogoutAll(r.Context(), auth.MemberID)
	h.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// ── cookie helpers ────────────────────────────────────────────────────────────

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   h.refreshTTLDays * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *AuthHandler) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
}

// ── response builders ─────────────────────────────────────────────────────────

type loginResponse struct {
	AccessToken string         `json:"access_token"`
	Member      memberResponse `json:"member"`
	Church      churchResponse `json:"church"`
}

type memberResponse struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Email       string                 `json:"email"`
	Phone       *string                `json:"phone"`
	BirthDate   *string                `json:"birth_date"`
	AvatarURL   *string                `json:"avatar_url"`
	IsActive    bool                   `json:"is_active"`
	Roles       []roleSummaryResponse  `json:"roles"`
	Instruments []instrumentResponse   `json:"instruments"`
	CreatedAt   time.Time              `json:"created_at"`
}

type roleSummaryResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	BaseProfile string    `json:"base_profile"`
}

type instrumentResponse struct {
	ID             uuid.UUID `json:"id"`
	InstrumentID   uuid.UUID `json:"instrument_id"`
	InstrumentName string    `json:"instrument_name"`
	IsPrimary      bool      `json:"is_primary"`
}

type churchResponse struct {
	ID               uuid.UUID  `json:"id"`
	ParentChurchID   *uuid.UUID `json:"parent_church_id"`
	Name             string     `json:"name"`
	DenominationName *string    `json:"denomination_name"`
	CNPJ             *string    `json:"cnpj"`
	Address          *string    `json:"address"`
	IsAutonomous     bool       `json:"is_autonomous"`
	PlanTier         string     `json:"plan_tier"`
	MemberCountCache int        `json:"member_count_cache"`
	CreatedAt        time.Time  `json:"created_at"`
}

func buildLoginResponse(accessToken string, m *ports.LoginMember) loginResponse {
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

	return loginResponse{
		AccessToken: accessToken,
		Member: memberResponse{
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
		},
		Church: churchResponse{
			ID:               m.Church.ID,
			ParentChurchID:   m.Church.ParentChurchID,
			Name:             m.Church.Name,
			DenominationName: m.Church.DenominationName,
			CNPJ:             m.Church.CNPJ,
			Address:          m.Church.Address,
			IsAutonomous:     m.Church.IsAutonomous,
			PlanTier:         m.Church.PlanTier,
			MemberCountCache: m.Church.MemberCountCache,
			CreatedAt:        m.Church.CreatedAt,
		},
	}
}
