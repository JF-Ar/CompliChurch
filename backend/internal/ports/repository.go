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

// RegisterParams carries the data needed to create a new church + pastor atomically.
type RegisterParams struct {
	ChurchName   string
	PastorName   string
	Email        string
	PasswordHash string
}

// AuthRepository is the port the auth service depends on.
type AuthRepository interface {
	GetMemberForLogin(ctx context.Context, email string) (*LoginMember, error)
	GetMemberForToken(ctx context.Context, memberID uuid.UUID) (*TokenMember, error)
	CreateRefreshToken(ctx context.Context, memberID, jti uuid.UUID, expiresAt time.Time) error
	GetRefreshTokenByJTI(ctx context.Context, jti uuid.UUID) (*RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, jti uuid.UUID) error
	RevokeAllMemberRefreshTokens(ctx context.Context, memberID uuid.UUID) error
	CreateChurchWithPastor(ctx context.Context, params RegisterParams) (*LoginMember, error)
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
	GetMemberInstruments(ctx context.Context, memberID, churchID uuid.UUID) ([]MemberInstrument, error)
	AddMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID, isPrimary bool) (*MemberInstrument, error)
	RemoveMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID) error
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

// ── Inventory domain types ────────────────────────────────────────────────────

type ItemCategory struct {
	ID       uuid.UUID
	ChurchID uuid.UUID
	Name     string
	Icon     *string
}

type Item struct {
	ID             uuid.UUID
	ChurchID       uuid.UUID
	Category       *ItemCategory
	ItemType       string
	Name           string
	Description    *string
	AssetNumber    *string
	PhotoURL       *string
	Location       string
	Status         string
	Quantity       int
	QtyMinAlert    *int
	SerialNumber   *string
	Notes          *string
	DeletedAt      *time.Time
	DeletionReason *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LoanMember struct {
	ID    uuid.UUID
	Name  string
	Email string
}

type Loan struct {
	ID                 uuid.UUID
	Item               Item
	RequestedBy        LoanMember
	ApprovedBy         *LoanMember
	LoanToType         string
	LoanToID           uuid.UUID
	LoanToName         string
	Status             string
	ExpectedReturnDate *time.Time
	ActualReturnDate   *time.Time
	ReturnCondition    *string
	ReturnNotes        *string
	CreatedAt          time.Time
	ReturnedAt         *time.Time
}

type ItemCategoryCreateInput struct {
	Name string
	Icon *string
}

type ItemCreateInput struct {
	ItemType     string
	Name         string
	Description  *string
	CategoryID   *uuid.UUID
	AssetNumber  *string
	Location     string
	Quantity     int
	QtyMinAlert  *int
	SerialNumber *string
	Notes        *string
}

type ItemUpdateInput struct {
	Name         string
	Description  *string
	CategoryID   *uuid.UUID
	Location     string
	Status       string
	Quantity     int
	QtyMinAlert  *int
	SerialNumber *string
	Notes        *string
}

type ListItemsFilter struct {
	Page           int
	PerPage        int
	Search         *string
	CategoryID     *uuid.UUID
	Status         *string
	ItemType       *string
	IncludeDeleted bool
}

type ListLoansFilter struct {
	Page    int
	PerPage int
	Status  *string
}

type LoanCreateInput struct {
	ItemID             uuid.UUID
	LoanToType         string
	LoanToID           uuid.UUID
	ExpectedReturnDate *time.Time
}

type LoanReturnInput struct {
	ReturnCondition string
	ReturnNotes     *string
}

type InventoryRepository interface {
	// Categories
	ListCategories(ctx context.Context, churchID uuid.UUID) ([]ItemCategory, error)
	CreateCategory(ctx context.Context, churchID uuid.UUID, input ItemCategoryCreateInput) (*ItemCategory, error)
	GetCategoryByID(ctx context.Context, id, churchID uuid.UUID) (*ItemCategory, error)
	UpdateCategory(ctx context.Context, id, churchID uuid.UUID, input ItemCategoryCreateInput) (*ItemCategory, error)
	DeleteCategory(ctx context.Context, id, churchID uuid.UUID) error

	// Items
	ListItems(ctx context.Context, churchID uuid.UUID, filter ListItemsFilter) ([]Item, int, error)
	CreateItem(ctx context.Context, churchID uuid.UUID, input ItemCreateInput) (*Item, error)
	GetItemByID(ctx context.Context, id, churchID uuid.UUID) (*Item, error)
	UpdateItem(ctx context.Context, id, churchID uuid.UUID, input ItemUpdateInput) (*Item, error)
	UpdateItemPhotoURL(ctx context.Context, id, churchID uuid.UUID, photoURL string) error
	SoftDeleteItem(ctx context.Context, id, churchID uuid.UUID, reason string) error
	CountItemsWithPrefix(ctx context.Context, churchID uuid.UUID, prefix string) (int, error)

	// Loans
	ListLoans(ctx context.Context, churchID uuid.UUID, filter ListLoansFilter) ([]Loan, int, error)
	CreateLoan(ctx context.Context, churchID uuid.UUID, requestedBy uuid.UUID, input LoanCreateInput) (*Loan, error)
	GetLoanByID(ctx context.Context, id, churchID uuid.UUID) (*Loan, error)
	ApproveLoan(ctx context.Context, id, approvedBy, churchID uuid.UUID) (*Loan, error)
	RejectLoan(ctx context.Context, id, churchID uuid.UUID) (*Loan, error)
	ReturnLoan(ctx context.Context, id, churchID uuid.UUID, input LoanReturnInput) (*Loan, error)

	// Validation helpers
	MemberBelongsToChurch(ctx context.Context, memberID, churchID uuid.UUID) (bool, error)
	ChurchExists(ctx context.Context, churchID uuid.UUID) (bool, error)
}
