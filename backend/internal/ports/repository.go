package ports

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("not found")

// ── Auth types ────────────────────────────────────────────────────────────────

type MemberRole struct {
	ID          uuid.UUID
	Name        string
	BaseProfile string // pastor | leadership | musician | member
}

type MemberInstrument struct {
	ID             uuid.UUID
	InstrumentID   uuid.UUID
	InstrumentName string
	IsPrimary      bool
}

type ChurchData struct {
	ID               uuid.UUID
	ParentChurchID   *uuid.UUID
	Name             string
	DenominationName *string
	CNPJ             *string
	Address          *string
	IsAutonomous     bool
	PlanTier         string
	MemberCountCache int
	CreatedAt        time.Time
}

// LoginMember carries everything the auth service needs after a successful credential check.
type LoginMember struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	Phone        *string
	BirthDate    *time.Time
	AvatarURL    *string
	IsActive     bool
	CreatedAt    time.Time

	PrimaryChurchID uuid.UUID
	BaseProfile     string // highest of all roles in the primary church
	ChurchIDs       []uuid.UUID

	Church      ChurchData
	Roles       []MemberRole
	Instruments []MemberInstrument
}

// TokenMember is the minimal data needed to mint a new access token during refresh.
type TokenMember struct {
	ID              uuid.UUID
	IsActive        bool
	PrimaryChurchID uuid.UUID
	BaseProfile     string
	ChurchIDs       []uuid.UUID
}

type RefreshToken struct {
	ID        uuid.UUID
	MemberID  uuid.UUID
	JTI       uuid.UUID
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

// AuthRepository is the port the auth service depends on.
type AuthRepository interface {
	GetMemberForLogin(ctx context.Context, email string) (*LoginMember, error)
	GetMemberForToken(ctx context.Context, memberID uuid.UUID) (*TokenMember, error)
	CreateRefreshToken(ctx context.Context, memberID, jti uuid.UUID, expiresAt time.Time) error
	GetRefreshTokenByJTI(ctx context.Context, jti uuid.UUID) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, jti uuid.UUID) error
	RevokeAllMemberRefreshTokens(ctx context.Context, memberID uuid.UUID) error
}
