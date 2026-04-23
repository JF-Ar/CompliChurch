package services

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jf-ar/compli-church/internal/ports"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrTokenExpired       = errors.New("token expired")
	ErrInvalidToken       = errors.New("invalid token")
)

// AccessClaims are extracted from a validated access token.
type AccessClaims struct {
	MemberID    uuid.UUID
	ChurchID    uuid.UUID
	BaseProfile string
	ChurchIDs   []uuid.UUID
}

// LoginResult is returned by AuthService.Login.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	Member       *ports.LoginMember
}

// RefreshResult is returned by AuthService.Refresh.
type RefreshResult struct {
	AccessToken  string
	RefreshToken string
}

// jwtClaims is the internal JWT payload for both token types.
type jwtClaims struct {
	jwt.RegisteredClaims
	MemberID    string   `json:"member_id,omitempty"`
	ChurchID    string   `json:"church_id,omitempty"`
	BaseProfile string   `json:"base_profile,omitempty"`
	ChurchIDs   []string `json:"church_ids,omitempty"`
	TokenType   string   `json:"type,omitempty"` // "access" | "refresh"
}

type AuthService struct {
	repo          ports.AuthRepository
	privateKey    *rsa.PrivateKey
	publicKey     *rsa.PublicKey
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewAuthService(
	repo ports.AuthRepository,
	privateKey *rsa.PrivateKey,
	publicKey *rsa.PublicKey,
	accessTTLMinutes int,
	refreshTTLDays int,
) *AuthService {
	return &AuthService{
		repo:       repo,
		privateKey: privateKey,
		publicKey:  publicKey,
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLDays) * 24 * time.Hour,
	}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	member, err := s.repo.GetMemberForLogin(ctx, email)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(member.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.mintAccessToken(member.ID, member.PrimaryChurchID, member.BaseProfile, member.ChurchIDs)
	if err != nil {
		return nil, fmt.Errorf("mint access token: %w", err)
	}

	refreshJTI := uuid.New()
	expiresAt := time.Now().Add(s.refreshTTL)
	refreshToken, err := s.mintRefreshToken(member.ID, refreshJTI, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("mint refresh token: %w", err)
	}

	if err := s.repo.CreateRefreshToken(ctx, member.ID, refreshJTI, expiresAt); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		Member:       member,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawRefreshToken string) (*RefreshResult, error) {
	// 1. Parse and validate signature
	claims, err := s.parseRefreshToken(rawRefreshToken)
	if err != nil {
		return nil, ErrInvalidToken
	}

	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 2. Look up JTI in DB
	stored, err := s.repo.GetRefreshTokenByJTI(ctx, jti)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
	}
	if stored.RevokedAt != nil {
		return nil, ErrTokenRevoked
	}
	if time.Now().After(stored.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// 3. Fetch member for new token claims
	member, err := s.repo.GetMemberForToken(ctx, stored.MemberID)
	if err != nil {
		return nil, err
	}

	// 4. Generate new tokens
	newAccessToken, err := s.mintAccessToken(member.ID, member.PrimaryChurchID, member.BaseProfile, member.ChurchIDs)
	if err != nil {
		return nil, fmt.Errorf("mint access token: %w", err)
	}

	newJTI := uuid.New()
	expiresAt := time.Now().Add(s.refreshTTL)
	newRefreshToken, err := s.mintRefreshToken(member.ID, newJTI, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("mint refresh token: %w", err)
	}

	// 5. Rotate: revoke old, store new (best-effort: if store fails we still revoke old)
	if err := s.repo.RevokeRefreshToken(ctx, jti); err != nil {
		return nil, fmt.Errorf("revoke old token: %w", err)
	}
	if err := s.repo.CreateRefreshToken(ctx, member.ID, newJTI, expiresAt); err != nil {
		return nil, fmt.Errorf("store new token: %w", err)
	}

	return &RefreshResult{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawRefreshToken string) error {
	claims, err := s.parseRefreshToken(rawRefreshToken)
	if err != nil {
		// Token invalid — treat as already logged out
		return nil
	}
	jti, err := uuid.Parse(claims.ID)
	if err != nil {
		return nil
	}
	return s.repo.RevokeRefreshToken(ctx, jti)
}

func (s *AuthService) LogoutAll(ctx context.Context, memberID uuid.UUID) error {
	return s.repo.RevokeAllMemberRefreshTokens(ctx, memberID)
}

// ValidateAccessToken parses and validates an access token, returning its claims.
func (s *AuthService) ValidateAccessToken(tokenStr string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid || claims.TokenType != "access" {
		return nil, ErrInvalidToken
	}

	memberID, err := uuid.Parse(claims.MemberID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	churchID, err := uuid.Parse(claims.ChurchID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	churchIDs := make([]uuid.UUID, 0, len(claims.ChurchIDs))
	for _, idStr := range claims.ChurchIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, ErrInvalidToken
		}
		churchIDs = append(churchIDs, id)
	}

	return &AccessClaims{
		MemberID:    memberID,
		ChurchID:    churchID,
		BaseProfile: claims.BaseProfile,
		ChurchIDs:   churchIDs,
	}, nil
}

// ── private helpers ───────────────────────────────────────────────────────────

func (s *AuthService) mintAccessToken(memberID, churchID uuid.UUID, baseProfile string, churchIDs []uuid.UUID) (string, error) {
	churchIDStrs := make([]string, len(churchIDs))
	for i, id := range churchIDs {
		churchIDStrs[i] = id.String()
	}

	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "compli-church",
			Subject:   memberID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			ID:        uuid.New().String(),
		},
		MemberID:    memberID.String(),
		ChurchID:    churchID.String(),
		BaseProfile: baseProfile,
		ChurchIDs:   churchIDStrs,
		TokenType:   "access",
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
}

func (s *AuthService) mintRefreshToken(memberID, jti uuid.UUID, expiresAt time.Time) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "compli-church",
			Subject:   memberID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        jti.String(),
		},
		TokenType: "refresh",
	}
	return jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(s.privateKey)
}

func (s *AuthService) parseRefreshToken(tokenStr string) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.publicKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid || claims.TokenType != "refresh" {
		return nil, errors.New("invalid refresh token")
	}
	return claims, nil
}
