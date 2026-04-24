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

// ── Shared error sentinels ─────────────────────────────────────────────────────

var ErrAlreadyExists = errors.New("already exists")

// ── Member domain types ───────────────────────────────────────────────────────

type Member struct {
	ID          uuid.UUID
	Name        string
	Email       string
	Phone       *string
	BirthDate   *time.Time
	AvatarURL   *string
	IsActive    bool
	CreatedAt   time.Time
	Roles       []MemberRole
	Instruments []MemberInstrument
}

type MemberCreateInput struct {
	Name      string
	Email     string
	Phone     *string
	BirthDate *time.Time
	RoleIDs   []uuid.UUID
}

type MemberUpdateInput struct {
	Name      string
	Phone     *string
	BirthDate *time.Time
}

type ListMembersFilter struct {
	Page     int
	PerPage  int
	Search   *string
	Role     *string // base_profile
	IsActive *bool
}

type ImportRow struct {
	Name  string
	Email string
	Phone *string
}

type ImportRowError struct {
	Row    int    `json:"row"`
	Reason string `json:"reason"`
}

type ImportResult struct {
	Created int
	Skipped int
	Errors  []ImportRowError
}

// ── Role domain types ─────────────────────────────────────────────────────────

type Role struct {
	ID          uuid.UUID
	ChurchID    *uuid.UUID
	Name        string
	BaseProfile string
	IsSystem    bool
}

// ── Instrument domain types ───────────────────────────────────────────────────

type Instrument struct {
	ID       uuid.UUID
	ChurchID *uuid.UUID
	Name     string
	IsSystem bool
}

// ── Repository interfaces ─────────────────────────────────────────────────────

type MemberRepository interface {
	ListMembers(ctx context.Context, churchID uuid.UUID, filter ListMembersFilter) ([]Member, int, error)
	CreateMember(ctx context.Context, churchID uuid.UUID, input MemberCreateInput, assignedBy uuid.UUID, passwordHash string) (*Member, error)
	GetMemberByID(ctx context.Context, id, churchID uuid.UUID) (*Member, error)
	UpdateMember(ctx context.Context, id, churchID uuid.UUID, input MemberUpdateInput) (*Member, error)
	DeactivateMember(ctx context.Context, id, churchID uuid.UUID) error
	GetMemberRoles(ctx context.Context, memberID, churchID uuid.UUID) ([]MemberRole, error)
	AssignRole(ctx context.Context, memberID, churchID, roleID, assignedBy uuid.UUID) error
	RemoveRole(ctx context.Context, memberID, roleID, churchID uuid.UUID) error
	GetMemberInstruments(ctx context.Context, memberID uuid.UUID) ([]MemberInstrument, error)
	AddMemberInstrument(ctx context.Context, memberID, instrumentID uuid.UUID, isPrimary bool) (*MemberInstrument, error)
	RemoveMemberInstrument(ctx context.Context, memberID, instrumentID uuid.UUID) error
}

type RoleRepository interface {
	ListRoles(ctx context.Context, churchID uuid.UUID) ([]Role, error)
	CreateRole(ctx context.Context, churchID uuid.UUID, name, baseProfile string) (*Role, error)
	GetRoleByID(ctx context.Context, id uuid.UUID) (*Role, error)
	UpdateRole(ctx context.Context, id, churchID uuid.UUID, name, baseProfile string) (*Role, error)
	DeleteRole(ctx context.Context, id, churchID uuid.UUID) error
}

type InstrumentRepository interface {
	ListInstruments(ctx context.Context, churchID uuid.UUID) ([]Instrument, error)
	CreateInstrument(ctx context.Context, churchID uuid.UUID, name string) (*Instrument, error)
	GetInstrumentByID(ctx context.Context, id uuid.UUID) (*Instrument, error)
	DeleteInstrument(ctx context.Context, id, churchID uuid.UUID) error
}
