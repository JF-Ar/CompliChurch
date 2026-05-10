package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/jf-ar/compli-church/internal/ports"
)

var (
	ErrScheduleNotFound        = errors.New("schedule not found")
	ErrScheduleAlreadyExists   = errors.New("a schedule for this Sunday already exists")
	ErrScheduleAlreadyCancelled = errors.New("schedule is already cancelled")
	ErrScheduleNotDraft        = errors.New("schedule must be in draft status for this action")
	ErrSlotNotFound            = errors.New("slot not found")
	ErrSlotAlreadyExists       = errors.New("member is already in this schedule")
	ErrSlotNotOwned            = errors.New("you can only confirm your own slot")
	ErrExceptionNotFound       = errors.New("availability exception not found")
	ErrExceptionAlreadyExists  = errors.New("you already have an exception for this date")
)

type ScheduleService struct {
	repo   ports.WorshipRepository
	mailer ports.Mailer
}

func NewScheduleService(repo ports.WorshipRepository, mailer ports.Mailer) *ScheduleService {
	return &ScheduleService{repo: repo, mailer: mailer}
}

// ── Exceptions ────────────────────────────────────────────────────────────────

func (s *ScheduleService) CreateException(ctx context.Context, churchID, memberID uuid.UUID, date string, reason *string) (*ports.AvailabilityException, error) {
	e, err := s.repo.CreateException(ctx, churchID, memberID, date, reason)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrExceptionAlreadyExists
		}
		return nil, err
	}
	return e, nil
}

func (s *ScheduleService) DeleteException(ctx context.Context, id, memberID, churchID uuid.UUID) error {
	err := s.repo.DeleteException(ctx, id, memberID, churchID)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrExceptionNotFound
	}
	return err
}

func (s *ScheduleService) ListMyExceptions(ctx context.Context, memberID, churchID uuid.UUID, month string) ([]*ports.AvailabilityException, error) {
	return s.repo.ListMyExceptions(ctx, memberID, churchID, month)
}

func (s *ScheduleService) ListAllExceptions(ctx context.Context, churchID uuid.UUID, month string) ([]*ports.AvailabilityExceptionWithMember, error) {
	return s.repo.ListAllExceptions(ctx, churchID, month)
}

// ── Schedules ─────────────────────────────────────────────────────────────────

func (s *ScheduleService) CreateSchedule(ctx context.Context, churchID, createdBy uuid.UUID, sundayDate string, notes *string) (*ports.Schedule, error) {
	sched, err := s.repo.CreateSchedule(ctx, churchID, createdBy, sundayDate, notes)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrScheduleAlreadyExists
		}
		return nil, err
	}
	return sched, nil
}

func (s *ScheduleService) GetSchedule(ctx context.Context, id, churchID uuid.UUID) (*ports.Schedule, error) {
	sched, err := s.repo.GetSchedule(ctx, id, churchID)
	if errors.Is(err, ports.ErrNotFound) {
		return nil, ErrScheduleNotFound
	}
	return sched, err
}

func (s *ScheduleService) ListSchedules(ctx context.Context, churchID uuid.UUID, status *string, page, perPage int) ([]*ports.ScheduleSummary, int, error) {
	return s.repo.ListSchedules(ctx, churchID, status, page, perPage)
}

func (s *ScheduleService) UpdateSchedule(ctx context.Context, id, churchID uuid.UUID, notes *string) (*ports.Schedule, error) {
	sched, err := s.repo.UpdateSchedule(ctx, id, churchID, notes)
	if errors.Is(err, ports.ErrNotFound) {
		return nil, ErrScheduleNotFound
	}
	return sched, err
}

func (s *ScheduleService) CancelSchedule(ctx context.Context, id, churchID uuid.UUID) error {
	sched, err := s.repo.GetSchedule(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return err
	}
	if sched.Status == "cancelled" {
		return ErrScheduleAlreadyCancelled
	}
	return s.repo.CancelSchedule(ctx, id, churchID)
}

func (s *ScheduleService) PublishSchedule(ctx context.Context, id, churchID uuid.UUID) (*ports.Schedule, error) {
	sched, err := s.repo.GetSchedule(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	if sched.Status == "cancelled" {
		return nil, ErrScheduleAlreadyCancelled
	}
	if sched.Status != "draft" {
		return nil, ErrScheduleNotDraft
	}

	published, err := s.repo.PublishSchedule(ctx, id, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	if s.mailer != nil {
		snap := published
		go func() {
			for _, slot := range snap.Slots {
				_ = s.mailer.Send(context.Background(), ports.EmailMessage{
					To:       slot.Member.Email,
					Template: "schedule_published",
					Data: map[string]any{
						"member_name": slot.Member.Name,
						"sunday_date": snap.SundayDate,
						"function":    slot.FunctionInScale,
					},
				})
			}
		}()
	}
	return published, nil
}

// ── Slots ─────────────────────────────────────────────────────────────────────

func (s *ScheduleService) ListSlots(ctx context.Context, scheduleID, churchID uuid.UUID) ([]*ports.ScheduleSlot, error) {
	_, err := s.repo.GetSchedule(ctx, scheduleID, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	return s.repo.ListSlots(ctx, scheduleID, churchID)
}

func (s *ScheduleService) AddSlot(ctx context.Context, scheduleID, churchID, memberID uuid.UUID, instrumentID *uuid.UUID, functionInScale string) (*ports.ScheduleSlot, error) {
	sched, err := s.repo.GetSchedule(ctx, scheduleID, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}
	if sched.Status != "draft" {
		return nil, ErrScheduleNotDraft
	}

	slot, err := s.repo.AddSlot(ctx, scheduleID, churchID, memberID, instrumentID, functionInScale)
	if err != nil {
		if errors.Is(err, ports.ErrAlreadyExists) {
			return nil, ErrSlotAlreadyExists
		}
		return nil, err
	}
	return slot, nil
}

func (s *ScheduleService) RemoveSlot(ctx context.Context, slotID, scheduleID, churchID uuid.UUID) error {
	_, err := s.repo.GetSchedule(ctx, scheduleID, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return ErrScheduleNotFound
		}
		return err
	}
	err = s.repo.RemoveSlot(ctx, slotID, scheduleID, churchID)
	if errors.Is(err, ports.ErrNotFound) {
		return ErrSlotNotFound
	}
	return err
}

func (s *ScheduleService) ConfirmSlot(ctx context.Context, slotID, scheduleID, churchID, memberID uuid.UUID) (*ports.ScheduleSlot, error) {
	sched, err := s.repo.GetSchedule(ctx, scheduleID, churchID)
	if err != nil {
		if errors.Is(err, ports.ErrNotFound) {
			return nil, ErrScheduleNotFound
		}
		return nil, err
	}

	var target *ports.ScheduleSlot
	for i := range sched.Slots {
		if sched.Slots[i].ID == slotID {
			cp := sched.Slots[i]
			target = &cp
			break
		}
	}
	if target == nil {
		return nil, ErrSlotNotFound
	}
	if target.Member.ID != memberID {
		return nil, ErrSlotNotOwned
	}

	slot, err := s.repo.ConfirmSlot(ctx, slotID, scheduleID, memberID)
	if errors.Is(err, ports.ErrNotFound) {
		return nil, ErrSlotNotFound
	}
	return slot, err
}

// ── Suggestion ────────────────────────────────────────────────────────────────

func (s *ScheduleService) GetScheduleSuggestion(ctx context.Context, churchID uuid.UUID, sundayDate string) (*ports.ScheduleSuggestion, error) {
	t, err := time.Parse("2006-01-02", sundayDate)
	if err != nil {
		return nil, fmt.Errorf("invalid sunday_date format (expected YYYY-MM-DD): %w", err)
	}
	precedingSunday := t.AddDate(0, 0, -7).Format("2006-01-02")

	// 1. All musicians in church
	musicians, err := s.repo.ListMusiciansInChurch(ctx, churchID)
	if err != nil {
		return nil, err
	}

	// 2. Exceptions for this Sunday
	exceptions, err := s.repo.ListExceptionsForDate(ctx, churchID, sundayDate)
	if err != nil {
		return nil, err
	}
	unavailableMap := make(map[uuid.UUID]*string, len(exceptions))
	for _, e := range exceptions {
		unavailableMap[e.MemberID] = e.Reason
	}

	// 3. Split into available / unavailable
	var available []*ports.MemberWithInstruments
	var unavailableMembers []ports.UnavailableMember
	for _, m := range musicians {
		if reason, bad := unavailableMap[m.Member.ID]; bad {
			unavailableMembers = append(unavailableMembers, ports.UnavailableMember{Member: m.Member, Reason: reason})
		} else {
			available = append(available, m)
		}
	}

	// 4. Slots from preceding Sunday — flag (NOT exclude) consecutive members
	precedingSlots, err := s.repo.ListSlotsForDate(ctx, churchID, precedingSunday)
	if err != nil {
		return nil, err
	}
	consecutiveIDs := make(map[uuid.UUID]bool, len(precedingSlots))
	for _, sl := range precedingSlots {
		consecutiveIDs[sl.Member.ID] = true
	}

	// 5. Group available by primary instrument
	type group struct {
		instrID   uuid.UUID
		instrName string
		members   []*ports.MemberWithInstruments
	}
	var instrOrder []uuid.UUID
	instrGroups := map[uuid.UUID]*group{}
	for _, m := range available {
		if m.PrimaryInstrument == nil {
			continue
		}
		g, ok := instrGroups[m.PrimaryInstrument.ID]
		if !ok {
			g = &group{instrID: m.PrimaryInstrument.ID, instrName: m.PrimaryInstrument.Name}
			instrGroups[m.PrimaryInstrument.ID] = g
			instrOrder = append(instrOrder, m.PrimaryInstrument.ID)
		}
		g.members = append(g.members, m)
	}

	// 6. Per group: pick member with fewest slots in last 4 weeks
	var suggested []ports.SuggestedSlot
	for _, iid := range instrOrder {
		g := instrGroups[iid]
		var best *ports.MemberWithInstruments
		bestCount := math.MaxInt
		for _, m := range g.members {
			count, err := s.repo.CountSlotsInLastNWeeks(ctx, churchID, m.Member.ID, 4)
			if err != nil {
				return nil, err
			}
			if count < bestCount {
				bestCount = count
				best = m
			}
		}
		if best == nil {
			continue
		}
		var warning *string
		if consecutiveIDs[best.Member.ID] {
			w := "consecutive_sunday"
			warning = &w
		}
		instrID := g.instrID
		suggested = append(suggested, ports.SuggestedSlot{
			MemberID:       best.Member.ID,
			MemberName:     best.Member.Name,
			InstrumentID:   &instrID,
			InstrumentName: g.instrName,
			Warning:        warning,
		})
	}

	// Build available_members list
	availableMembers := make([]ports.MemberSummary, len(available))
	for i, m := range available {
		availableMembers[i] = m.Member
	}

	if suggested == nil {
		suggested = []ports.SuggestedSlot{}
	}
	if unavailableMembers == nil {
		unavailableMembers = []ports.UnavailableMember{}
	}

	return &ports.ScheduleSuggestion{
		SundayDate:         sundayDate,
		SuggestedSlots:     suggested,
		AvailableMembers:   availableMembers,
		UnavailableMembers: unavailableMembers,
	}, nil
}
