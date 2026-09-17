package payroll

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
)

// WeeklyPaymentFeeScheduleService manages the JSON array of weekly-payment
// fee schedules stored under Settings.key =
// domain.WeeklyPaymentFeeScheduleSettingsKey.
//
// Concurrency: every write path acquires SELECT ... FOR UPDATE on the settings
// row inside a transaction so two admins editing simultaneously can't lose
// each other's changes. Reads take no lock.
type WeeklyPaymentFeeScheduleService struct {
	store    *infrastructure.SettingsStore[domain.WeeklyPaymentFeeScheduleEntry]
	eventBus domain.EventBus
	logger   *slog.Logger
}

// NewWeeklyPaymentFeeScheduleService wires the service. db is the raw
// *gorm.DB - we need transactions and FOR UPDATE locking that the
// SettingsRepository abstraction doesn't expose.
func NewWeeklyPaymentFeeScheduleService(db *gorm.DB, eventBus domain.EventBus, logger *slog.Logger) *WeeklyPaymentFeeScheduleService {
	if logger == nil {
		logger = observability.GetLogger()
	}
	return &WeeklyPaymentFeeScheduleService{
		store: &infrastructure.SettingsStore[domain.WeeklyPaymentFeeScheduleEntry]{
			DB:          db,
			SettingsKey: domain.WeeklyPaymentFeeScheduleSettingsKey,
			SortFn:      domain.SortWeeklyPaymentFeeScheduleEntries,
			NotFoundMsg: "weekly_payment_fee_schedules setting row missing - run migration 107",
			EntityName:  "weekly payment fee schedule",
			DateLayout:  domain.WeeklyPaymentFeeScheduleDateLayout,
		},
		eventBus: eventBus,
		logger:   logger,
	}
}

// List returns every schedule entry sorted by EffectiveDate descending so
// newest entries appear first in the admin table.
func (s *WeeklyPaymentFeeScheduleService) List(ctx context.Context) ([]domain.WeeklyPaymentFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	domain.SortWeeklyPaymentFeeScheduleEntries(entries)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// GetByID returns a single entry by its UUID.
func (s *WeeklyPaymentFeeScheduleService) GetByID(ctx context.Context, id string) (*domain.WeeklyPaymentFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.WeeklyPaymentFeeScheduleEntry) string { return e.ID })
	if idx < 0 {
		return nil, domain.NewNotFoundError("weekly payment fee schedule entry not found")
	}
	return &entries[idx], nil
}

// WeeklyFeeScheduleCreateInput is the service-layer input for Create. The
// entry ID is generated internally; the caller does not supply one.
type WeeklyFeeScheduleCreateInput struct {
	EffectiveDate string
	Percentage    float64
	Notes         string
}

// Create appends a new schedule entry to the JSON array. Rejects past
// effective_date because back-dating would rewrite the fee charged on
// already-completed disbursements, which is bad for audit. Today is allowed.
func (s *WeeklyPaymentFeeScheduleService) Create(ctx context.Context, input WeeklyFeeScheduleCreateInput, actorUserID uint) (*domain.WeeklyPaymentFeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	entry := domain.WeeklyPaymentFeeScheduleEntry{
		ID:              uuid.NewString(),
		EffectiveDate:   input.EffectiveDate,
		Percentage:      input.Percentage,
		Notes:           strings.TrimSpace(input.Notes),
		CreatedAt:       clock.NowUTC(),
		CreatedByUserID: infrastructure.NonZeroUserID(actorUserID),
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}

	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.WeeklyPaymentFeeScheduleEntry) ([]domain.WeeklyPaymentFeeScheduleEntry, error) {
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

	if err := s.eventBus.Publish(ctx, domain.NewWeeklyPaymentFeeScheduleCreatedEvent(ctx, entry.ID, entry.EffectiveDate, domain.FormatWeeklyPaymentFeeSummary(entry))); err != nil {
		s.logger.Warn("Failed to publish WeeklyPaymentFeeScheduleCreated event", "id", entry.ID, "error", err)
	}
	return &entry, nil
}

// WeeklyFeeScheduleUpdateInput mirrors WeeklyFeeScheduleCreateInput. Update
// replaces the entry payload wholesale (apart from ID and audit metadata).
type WeeklyFeeScheduleUpdateInput struct {
	EffectiveDate string
	Percentage    float64
	Notes         string
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity - admins must add a new entry to change rates
// going forward.
func (s *WeeklyPaymentFeeScheduleService) Update(ctx context.Context, id string, input WeeklyFeeScheduleUpdateInput) (*domain.WeeklyPaymentFeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	var updated domain.WeeklyPaymentFeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.WeeklyPaymentFeeScheduleEntry) ([]domain.WeeklyPaymentFeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.WeeklyPaymentFeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("weekly payment fee schedule entry not found")
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
		entries[idx].Percentage = input.Percentage
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

	if err := s.eventBus.Publish(ctx, domain.NewWeeklyPaymentFeeScheduleUpdatedEvent(ctx, updated.ID, updated.EffectiveDate, domain.FormatWeeklyPaymentFeeSummary(updated))); err != nil {
		s.logger.Warn("Failed to publish WeeklyPaymentFeeScheduleUpdated event", "id", updated.ID, "error", err)
	}
	return &updated, nil
}

// Delete removes a future-dated entry. Active and past entries are
// undeletable so the system always has at least one schedule that resolves.
func (s *WeeklyPaymentFeeScheduleService) Delete(ctx context.Context, id string) error {
	var removed domain.WeeklyPaymentFeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.WeeklyPaymentFeeScheduleEntry) ([]domain.WeeklyPaymentFeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.WeeklyPaymentFeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("weekly payment fee schedule entry not found")
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

	if err := s.eventBus.Publish(ctx, domain.NewWeeklyPaymentFeeScheduleDeletedEvent(ctx, removed.ID, removed.EffectiveDate, domain.FormatWeeklyPaymentFeeSummary(removed))); err != nil {
		s.logger.Warn("Failed to publish WeeklyPaymentFeeScheduleDeleted event", "id", removed.ID, "error", err)
	}
	return nil
}

// ActiveAt returns the entry that resolves fees on date `at`.
func (s *WeeklyPaymentFeeScheduleService) ActiveAt(ctx context.Context, at time.Time) (*domain.WeeklyPaymentFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	active := domain.ActiveWeeklyPaymentFeeScheduleAt(entries, at)
	if active == nil {
		return nil, domain.NewNotFoundError("no weekly-payment fee schedule active for this date")
	}
	return active, nil
}

// PercentageAt returns the active weekly-payment fee percentage (as a
// fraction, 0.02 == 2%) for date `at`. Falls back to 2% when no schedule
// resolves so fee computation never hard-fails a disbursement.
func (s *WeeklyPaymentFeeScheduleService) PercentageAt(ctx context.Context, at time.Time) float64 {
	entry, err := s.ActiveAt(ctx, at)
	if err != nil {
		s.logger.Warn("PercentageAt: no active weekly payment fee schedule, falling back to 2%", "at", at, "error", err)
		return 0.02
	}
	return entry.Percentage / 100.0
}
