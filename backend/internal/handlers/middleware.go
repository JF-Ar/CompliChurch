package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/services"
)

// AuthContext is injected into the request context by the Authenticate middleware.
type AuthContext struct {
	MemberID    uuid.UUID
	ChurchID    uuid.UUID
	BaseProfile string // pastor | leadership | musician | member
	ChurchIDs   []uuid.UUID
}

type contextKey string

const authKey contextKey = "auth"

// AuthContextFromContext extracts the AuthContext from a request context.
// Returns nil if the request was not authenticated.
func AuthContextFromContext(ctx context.Context) *AuthContext {
	v, _ := ctx.Value(authKey).(*AuthContext)
	return v
}

// Authenticate validates the Bearer access token and injects AuthContext into the request context.
func Authenticate(authSvc *services.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid Authorization header", "")
				return
			}

			claims, err := authSvc.ValidateAccessToken(strings.TrimPrefix(authHeader, "Bearer "))
			if err != nil {
				writeError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Token is invalid or expired", "")
				return
			}

			ctx := context.WithValue(r.Context(), authKey, &AuthContext{
				MemberID:    claims.MemberID,
				ChurchID:    claims.ChurchID,
				BaseProfile: claims.BaseProfile,
				ChurchIDs:   claims.ChurchIDs,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireProfile enforces a minimum base_profile level.
// Hierarchy: pastor(4) > leadership(3) > musician(2) > member(1).
// Passing "leadership" also allows "pastor" through.
func RequireProfile(minProfile string) func(http.Handler) http.Handler {
	minRank := profileRank(minProfile)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := AuthContextFromContext(r.Context())
			if auth == nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Not authenticated", "")
				return
			}
			if profileRank(auth.BaseProfile) < minRank {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "Insufficient permissions", "")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func profileRank(p string) int {
	switch p {
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
