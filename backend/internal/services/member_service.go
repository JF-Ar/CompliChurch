package services

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/jf-ar/compli-church/internal/ports"
)

var (
	ErrMemberEmailExists       = errors.New("member with this email already exists")
	ErrMemberNotFound          = errors.New("member not found")
	ErrRoleNotFound            = errors.New("role not found")
	ErrRoleAlreadyAssigned     = errors.New("role already assigned to this member")
	ErrRoleNotAssigned         = errors.New("role not assigned to this member")
	ErrRoleAccessDenied        = errors.New("role does not belong to this church")
	ErrSystemResource          = errors.New("system resource cannot be modified or deleted")
	ErrInstrumentNotFound      = errors.New("instrument not found")
	ErrInstrumentAlreadyAdded  = errors.New("instrument already in member profile")
	ErrInstrumentNotInProfile  = errors.New("instrument not in member profile")
)

type MemberService struct {
	memberRepo ports.MemberRepository
	roleRepo   ports.RoleRepository
	instrRepo  ports.InstrumentRepository
	mailer     ports.Mailer
}

func NewMemberService(
	memberRepo ports.MemberRepository,
	roleRepo ports.RoleRepository,
	instrRepo ports.InstrumentRepository,
	mailer ports.Mailer,
) *MemberService {
	return &MemberService{
		memberRepo: memberRepo,
		roleRepo:   roleRepo,
		instrRepo:  instrRepo,
		mailer:     mailer,
	}
}

// ── Members ───────────────────────────────────────────────────────────────────

func (s *MemberService) ListMembers(ctx context.Context, churchID uuid.UUID, filter ports.ListMembersFilter) ([]ports.Member, int, error) {
	return s.memberRepo.ListMembers(ctx, churchID, filter)
}

func (s *MemberService) CreateMember(ctx context.Context, churchID, createdBy uuid.UUID, input ports.MemberCreateInput) (*ports.Member, error) {
	// Validate role IDs belong to this church or are system roles.
	for _, roleID := range input.RoleIDs {
		role, err := s.roleRepo.GetRoleByID(ctx, roleID)
		if err != nil {
			if errors.Is(err, ports.ErrNotFound) {
				return nil, ErrRoleNotFound
			}
			return nil, err
		}
		if !role.IsSystem && (role.ChurchID == nil || *role.ChurchID != churchID) {
			return nil, ErrRoleAccessDenied
		}
	}

	// Generate a random placeholder password. Member must use forgot-password flow.
	hash, err := bcrypt.GenerateFromPassword([]byte(uuid.New().String()), 12)
	if err != nil {
		return nil, err
	}

	member, err := s.memberRepo.CreateMember(ctx, churchID, input, createdBy, string(hash))
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrMemberEmailExists
		}
		return nil, err
	}

	// Fire-and-forget welcome email.
	if s.mailer != nil {
		go func() {
			_ = s.mailer.Send(context.Background(), ports.EmailMessage{
				To:       member.Email,
				Template: "member_welcome",
				Data:     map[string]any{"name": member.Name},
			})
		}()
	}

	return member, nil
}

func (s *MemberService) ImportMembers(ctx context.Context, churchID, importedBy uuid.UUID, rows []ports.ImportRow) ports.ImportResult {
	var result ports.ImportResult
	for i, row := range rows {
		input := ports.MemberCreateInput{
			Name:  row.Name,
			Email: row.Email,
			Phone: row.Phone,
		}
		_, err := s.CreateMember(ctx, churchID, importedBy, input)
		if err != nil {
			if errors.Is(err, ErrMemberEmailExists) {
				result.Skipped++
			} else {
				result.Errors = append(result.Errors, ports.ImportRowError{
					Row:    i + 1,
					Reason: err.Error(),
				})
			}
		} else {
			result.Created++
		}
	}
	if result.Errors == nil {
		result.Errors = []ports.ImportRowError{}
	}
	return result
}

func (s *MemberService) GetMemberByID(ctx context.Context, id, churchID uuid.UUID) (*ports.Member, error) {
	member, err := s.memberRepo.GetMemberByID(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return member, nil
}

func (s *MemberService) UpdateMember(ctx context.Context, id, churchID uuid.UUID, input ports.MemberUpdateInput) (*ports.Member, error) {
	member, err := s.memberRepo.UpdateMember(ctx, id, churchID, input)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return member, nil
}

func (s *MemberService) DeactivateMember(ctx context.Context, id, churchID uuid.UUID) error {
	if err := s.memberRepo.DeactivateMember(ctx, id, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrMemberNotFound
		}
		return err
	}
	return nil
}

// ── Member roles ──────────────────────────────────────────────────────────────

func (s *MemberService) GetMemberRoles(ctx context.Context, memberID, churchID uuid.UUID) ([]ports.MemberRole, error) {
	// Verify member is in this church.
	if _, err := s.memberRepo.GetMemberByID(ctx, memberID, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return s.memberRepo.GetMemberRoles(ctx, memberID, churchID)
}

func (s *MemberService) AssignRole(ctx context.Context, memberID, churchID, roleID, assignedBy uuid.UUID) error {
	// Verify the role exists and is accessible.
	role, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if !role.IsSystem && (role.ChurchID == nil || *role.ChurchID != churchID) {
		return ErrRoleAccessDenied
	}

	if err := s.memberRepo.AssignRole(ctx, memberID, churchID, roleID, assignedBy); err != nil {
		switch {
		case errors.Is(err, ports.ErrNotFound):
			return ErrMemberNotFound
		case errors.Is(err, ports.ErrAlreadyExists):
			return ErrRoleAlreadyAssigned
		}
		return err
	}
	return nil
}

func (s *MemberService) RemoveRole(ctx context.Context, memberID, roleID, churchID uuid.UUID) error {
	if err := s.memberRepo.RemoveRole(ctx, memberID, roleID, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrRoleNotAssigned
		}
		return err
	}
	return nil
}

// ── Member instruments ────────────────────────────────────────────────────────

func (s *MemberService) GetMemberInstruments(ctx context.Context, memberID, churchID uuid.UUID) ([]ports.MemberInstrument, error) {
	// Verify member belongs to church.
	if _, err := s.memberRepo.GetMemberByID(ctx, memberID, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrMemberNotFound
		}
		return nil, err
	}
	return s.memberRepo.GetMemberInstruments(ctx, memberID, churchID)
}

func (s *MemberService) AddMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID, isPrimary bool) (*ports.MemberInstrument, error) {
	// Verify instrument exists and is accessible to this church.
	instr, err := s.instrRepo.GetInstrumentByID(ctx, instrumentID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrInstrumentNotFound
		}
		return nil, err
	}
	if !instr.IsSystem && (instr.ChurchID == nil || *instr.ChurchID != churchID) {
		return nil, ErrInstrumentNotFound
	}

	inst, err := s.memberRepo.AddMemberInstrument(ctx, memberID, churchID, instrumentID, isPrimary)
	if err != nil {
		switch {
		case errors.Is(err, ports.ErrNotFound):
			return nil, ErrMemberNotFound
		case errors.Is(err, ports.ErrAlreadyExists):
			return nil, ErrInstrumentAlreadyAdded
		}
		return nil, err
	}
	return inst, nil
}

func (s *MemberService) RemoveMemberInstrument(ctx context.Context, memberID, churchID, instrumentID uuid.UUID) error {
	if err := s.memberRepo.RemoveMemberInstrument(ctx, memberID, churchID, instrumentID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrInstrumentNotInProfile
		}
		return err
	}
	return nil
}

// ── Roles ─────────────────────────────────────────────────────────────────────

func (s *MemberService) ListRoles(ctx context.Context, churchID uuid.UUID) ([]ports.Role, error) {
	return s.roleRepo.ListRoles(ctx, churchID)
}

func (s *MemberService) CreateRole(ctx context.Context, churchID uuid.UUID, name, baseProfile string) (*ports.Role, error) {
	role, err := s.roleRepo.CreateRole(ctx, churchID, name, baseProfile)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrRoleAlreadyAssigned // reuse conflict sentinel for name conflict
		}
		return nil, err
	}
	return role, nil
}

func (s *MemberService) UpdateRole(ctx context.Context, churchID, roleID uuid.UUID, name, baseProfile string) (*ports.Role, error) {
	existing, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	if existing.IsSystem {
		return nil, ErrSystemResource
	}
	if existing.ChurchID == nil || *existing.ChurchID != churchID {
		return nil, ErrRoleNotFound
	}

	role, err := s.roleRepo.UpdateRole(ctx, roleID, churchID, name, baseProfile)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *MemberService) DeleteRole(ctx context.Context, churchID, roleID uuid.UUID) error {
	existing, err := s.roleRepo.GetRoleByID(ctx, roleID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	if existing.IsSystem {
		return ErrSystemResource
	}
	if existing.ChurchID == nil || *existing.ChurchID != churchID {
		return ErrRoleNotFound
	}

	if err := s.roleRepo.DeleteRole(ctx, roleID, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrRoleNotFound
		}
		return err
	}
	return nil
}

// ── Instruments ───────────────────────────────────────────────────────────────

func (s *MemberService) ListInstruments(ctx context.Context, churchID uuid.UUID) ([]ports.Instrument, error) {
	return s.instrRepo.ListInstruments(ctx, churchID)
}

func (s *MemberService) CreateInstrument(ctx context.Context, churchID uuid.UUID, name string) (*ports.Instrument, error) {
	inst, err := s.instrRepo.CreateInstrument(ctx, churchID, name)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrInstrumentAlreadyAdded
		}
		return nil, err
	}
	return inst, nil
}

func (s *MemberService) DeleteInstrument(ctx context.Context, churchID, instrumentID uuid.UUID) error {
	existing, err := s.instrRepo.GetInstrumentByID(ctx, instrumentID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrInstrumentNotFound
		}
		return err
	}
	if existing.IsSystem {
		return ErrSystemResource
	}
	if existing.ChurchID == nil || *existing.ChurchID != churchID {
		return ErrInstrumentNotFound
	}

	if err := s.instrRepo.DeleteInstrument(ctx, instrumentID, churchID); err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrInstrumentNotFound
		}
		return err
	}
	return nil
}
