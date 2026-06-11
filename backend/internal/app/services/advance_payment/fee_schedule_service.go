package advance_payment

import (
	"api-server/internal/pkg/clock"
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
)

// FeeScheduleService manages the JSON array of advance-payment fee schedules
// stored under Settings.key = AdvancePaymentFeeScheduleSettingsKey.
//
// Concurrency: every write path acquires SELECT … FOR UPDATE on the settings
// row inside a transaction so two admins editing simultaneously can't lose
// each other's changes. Reads (List, ResolveFee) take no lock.
//
// Caching: the active schedule is cached in process for ResolveFee's hot path
// (called per advance-payment request). Writes invalidate it.
type FeeScheduleService struct {
	store        *infrastructure.SettingsStore[domain.FeeScheduleEntry]
	cacheService *infrastructure.CacheService
	eventBus     domain.EventBus
	logger       *slog.Logger
}

// NewFeeScheduleService wires the service. db is the raw *gorm.DB — we need
// transactions and FOR UPDATE locking that the SettingsRepository abstraction
// doesn't expose.
func NewFeeScheduleService(db *gorm.DB, cacheService *infrastructure.CacheService, eventBus domain.EventBus, logger *slog.Logger) *FeeScheduleService {
	if logger == nil {
		logger = slog.Default()
	}
	return &FeeScheduleService{
		store: &infrastructure.SettingsStore[domain.FeeScheduleEntry]{
			DB:          db,
			SettingsKey: domain.AdvancePaymentFeeScheduleSettingsKey,
			SortFn:      domain.SortFeeScheduleEntries,
			NotFoundMsg: "advance_payment_fee_schedules setting row missing — run migration 044",
			EntityName:  "fee schedule",
			DateLayout:  domain.AdvancePaymentFeeScheduleDateLayout,
		},
		cacheService: cacheService,
		eventBus:     eventBus,
		logger:       logger,
	}
}

// List returns all schedule entries sorted by EffectiveDate descending.
// Used by the admin UI.
func (s *FeeScheduleService) List(ctx context.Context) ([]domain.FeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	domain.SortFeeScheduleEntries(entries)
	// Reverse for desc order so newest entries appear first in the admin table.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// GetByID returns a single entry by its UUID.
func (s *FeeScheduleService) GetByID(ctx context.Context, id string) (*domain.FeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i], nil
		}
	}
	return nil, domain.NewNotFoundError("fee schedule entry not found")
}

// CreateInput is the service-layer input for Create. ScheduleID is generated
// internally; the caller does not supply one.
type CreateInput struct {
	EffectiveDate string
	Tiers         []domain.FeeScheduleTier
	MinFeeVND     uint64
	Notes         string
}

// Create appends a new schedule entry to the JSON array. Rejects past
// effective_date because back-dating would rewrite the fee charged on
// already-completed transactions, which is bad for audit. Today is allowed
// (an admin announcing today's rate change should not be blocked).
func (s *FeeScheduleService) Create(ctx context.Context, input CreateInput, actorUserID uint) (*domain.FeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	entry := domain.FeeScheduleEntry{
		ID:              uuid.NewString(),
		EffectiveDate:   input.EffectiveDate,
		Tiers:           input.Tiers,
		MinFeeVND:       input.MinFeeVND,
		Notes:           strings.TrimSpace(input.Notes),
		CreatedAt:       clock.NowUTC(),
		CreatedByUserID: infrastructure.NonZeroUserID(actorUserID),
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}

	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.FeeScheduleEntry) ([]domain.FeeScheduleEntry, error) {
		for _, e := range entries {
			if e.EffectiveDate == entry.EffectiveDate {
				return nil, domain.NewValidationError("a schedule with this effective date already exists")
			}
		}
		return append(entries, entry), nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, domain.NewAdvancePaymentFeeScheduleCreatedEvent(ctx, entry.ID, entry.EffectiveDate, FormatScheduleSummary(entry))); err != nil {
		s.logger.Warn("Failed to publish AdvancePaymentFeeScheduleCreated event", "id", entry.ID, "error", err)
	}
	return &entry, nil
}

// UpdateInput mirrors CreateInput. Update replaces the entire entry payload
// (apart from ID and audit metadata) so partial updates aren't supported —
// frontend resends the full record.
type UpdateInput struct {
	EffectiveDate string
	Tiers         []domain.FeeScheduleTier
	MinFeeVND     uint64
	Notes         string
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity — admins must add a new entry to change rates
// going forward.
func (s *FeeScheduleService) Update(ctx context.Context, id string, input UpdateInput) (*domain.FeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	var updated domain.FeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.FeeScheduleEntry) ([]domain.FeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.FeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("fee schedule entry not found")
		}
		if !infrastructure.IsFutureDate(s.store.DateLayout, entries[idx].EffectiveDate) {
			return nil, domain.NewValidationError("only schedules with a future effective date can be edited")
		}
		for i, e := range entries {
			if i == idx {
				continue
			}
			if e.EffectiveDate == input.EffectiveDate {
				return nil, domain.NewValidationError("a schedule with this effective date already exists")
			}
		}

		entries[idx].EffectiveDate = input.EffectiveDate
		entries[idx].Tiers = input.Tiers
		entries[idx].MinFeeVND = input.MinFeeVND
		entries[idx].Notes = strings.TrimSpace(input.Notes)
		if err := entries[idx].Validate(); err != nil {
			return nil, err
		}
		updated = entries[idx]
		return entries, nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, domain.NewAdvancePaymentFeeScheduleUpdatedEvent(ctx, updated.ID, updated.EffectiveDate, FormatScheduleSummary(updated))); err != nil {
		s.logger.Warn("Failed to publish AdvancePaymentFeeScheduleUpdated event", "id", updated.ID, "error", err)
	}
	return &updated, nil
}

// Delete removes a future-dated entry. The current and past entries are
// undeletable — the system must always have at least one schedule that is
// active right now, otherwise fee resolution would fail.
func (s *FeeScheduleService) Delete(ctx context.Context, id string) error {
	var removed domain.FeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.FeeScheduleEntry) ([]domain.FeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.FeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("fee schedule entry not found")
		}
		if !infrastructure.IsFutureDate(s.store.DateLayout, entries[idx].EffectiveDate) {
			return nil, domain.NewValidationError("only schedules with a future effective date can be deleted")
		}
		if len(entries) <= 1 {
			return nil, domain.NewValidationError("cannot delete the only schedule entry")
		}
		removed = entries[idx]
		return append(entries[:idx], entries[idx+1:]...), nil
	})
	if err != nil {
		return err
	}

	if err := s.eventBus.Publish(ctx, domain.NewAdvancePaymentFeeScheduleDeletedEvent(ctx, removed.ID, removed.EffectiveDate, FormatScheduleSummary(removed))); err != nil {
		s.logger.Warn("Failed to publish AdvancePaymentFeeScheduleDeleted event", "id", removed.ID, "error", err)
	}
	return nil
}

// ActiveAt returns the entry that resolves fees on date `at`. Returns
// NotFoundError if no entry's effective_date <= at.
func (s *FeeScheduleService) ActiveAt(ctx context.Context, at time.Time) (*domain.FeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	active := domain.ActiveFeeScheduleAt(entries, at)
	if active == nil {
		return nil, domain.NewNotFoundError("no advance-payment fee schedule active for this date")
	}
	return active, nil
}

// ResolveFee is the hot path used by the calculator. Returns the fee in VND
// for a request of `amount` made on `at`. Falls back to a safe-default 2%/10k
// if no schedule is found, with a warning log — fee calculation should never
// hard-fail on a transaction.
func (s *FeeScheduleService) ResolveFee(ctx context.Context, amount uint64, at time.Time) uint64 {
	entry, err := s.ActiveAt(ctx, at)
	if err != nil {
		s.logger.Warn("ResolveFee: no active fee schedule, falling back to 2%/10000", "at", at, "error", err)
		return fallbackFee(amount)
	}
	return entry.ResolveFee(amount)
}

// FeePercentageAt returns the headline (first-tier) percentage for backwards
// compatibility with code paths that display a single rate to the user.
func (s *FeeScheduleService) FeePercentageAt(ctx context.Context, at time.Time) float64 {
	entry, err := s.ActiveAt(ctx, at)
	if err != nil {
		return 0.02
	}
	if len(entry.Tiers) == 0 {
		return 0
	}
	return entry.Tiers[0].Percentage / 100.0
}

// MinFeeAt returns the active schedule's MinFeeVND.
func (s *FeeScheduleService) MinFeeAt(ctx context.Context, at time.Time) uint64 {
	entry, err := s.ActiveAt(ctx, at)
	if err != nil {
		return 10000
	}
	return entry.MinFeeVND
}

// fallbackFee is the safety-net calculation if the schedule lookup fails.
// 2% / 10k VND mirrors the original hard-coded defaults so transactions never
// stall on a misconfigured fee table.
func fallbackFee(amount uint64) uint64 {
	pct := uint64(float64(amount) * 0.02)
	if pct > 10000 {
		return pct
	}
	return 10000
}
