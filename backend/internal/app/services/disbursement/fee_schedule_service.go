package disbursement

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// FeeScheduleService manages the JSON array of disbursement-provider fee
// schedules stored under Settings.key = DisbursementFeeScheduleSettingsKey.
//
// Mirrors the structure of advance_payment.FeeScheduleService — the only
// substantive difference is shape: disbursement entries hold a single flat
// FeeVND value instead of a tier list. Concurrency, locking, and read paths
// are identical.
type FeeScheduleService struct {
	store    *infrastructure.SettingsStore[domain.DisbursementFeeScheduleEntry]
	eventBus domain.EventBus
	registry feeRegistryResolver
	logger   *slog.Logger
}

// feeRegistryResolver resolves the active provider name for fee lookups.
type feeRegistryResolver interface {
	ActiveProviderName(ctx context.Context) (string, error)
}

// NewFeeScheduleService wires the service. db is the raw *gorm.DB — we need
// transactions and FOR UPDATE locking that the SettingsRepository abstraction
// doesn't expose.
func NewFeeScheduleService(db *gorm.DB, eventBus domain.EventBus, logger *slog.Logger) *FeeScheduleService {
	if logger == nil {
		logger = observability.GetLogger()
	}
	return &FeeScheduleService{
		store: &infrastructure.SettingsStore[domain.DisbursementFeeScheduleEntry]{
			DB:          db,
			SettingsKey: domain.DisbursementFeeScheduleSettingsKey,
			SortFn:      domain.SortDisbursementFeeScheduleEntries,
			NotFoundMsg: "disbursement_fee_schedules setting row missing — run migration 050",
			EntityName:  "disbursement fee schedule",
			DateLayout:  domain.DisbursementFeeScheduleDateLayout,
		},
		eventBus: eventBus,
		logger:   logger,
	}
}

// SetRegistry injects the disbursement registry for dynamic provider resolution.
func (s *FeeScheduleService) SetRegistry(r feeRegistryResolver) {
	s.registry = r
}

// GetDisbursementFeeVND resolves the active provider from the registry and
// returns the fee. Panics if registry is nil (fail-fast: no silent zero fee).
func (s *FeeScheduleService) GetDisbursementFeeVND(ctx context.Context, provider string) int64 {
	return s.ResolveFee(ctx, provider, clock.Now())
}

// ActiveEntry resolves the active provider from the registry and returns
// the currently effective fee schedule entry, or nil if none matches.
func (s *FeeScheduleService) ActiveEntry(ctx context.Context) (*domain.DisbursementFeeScheduleEntry, error) {
	if s.registry == nil {
		panic("FeeScheduleService: registry not set (fail-fast: cannot resolve provider)")
	}
	provider, err := s.registry.ActiveProviderName(ctx)
	if err != nil {
		return nil, fmt.Errorf("fee: resolve active provider: %w", err)
	}
	entries, err := s.List(ctx)
	if err != nil {
		return nil, err
	}
	active := domain.ActiveDisbursementFeeScheduleAt(entries, provider, clock.Now())
	return active, nil
}

// List returns all schedule entries sorted by EffectiveDate descending.
// Used by the admin UI.
func (s *FeeScheduleService) List(ctx context.Context) ([]domain.DisbursementFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	domain.SortDisbursementFeeScheduleEntries(entries)
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	return entries, nil
}

// GetByID returns a single entry by its UUID.
func (s *FeeScheduleService) GetByID(ctx context.Context, id string) (*domain.DisbursementFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].ID == id {
			return &entries[i], nil
		}
	}
	return nil, domain.NewNotFoundError("disbursement fee schedule entry not found")
}

// CreateInput is the service-layer input for Create. ScheduleID is generated
// internally; the caller does not supply one.
type CreateInput struct {
	Provider      string
	EffectiveDate string
	FeeVND        int64
	Notes         string
}

// Create appends a new schedule entry to the JSON array. Past effective_date
// is rejected because back-dating would rewrite the fee charged on already-
// completed transactions, which is bad for audit. Today is allowed (an admin
// announcing today's rate change should not be blocked).
func (s *FeeScheduleService) Create(ctx context.Context, input CreateInput, actorUserID uint) (*domain.DisbursementFeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	provider := input.Provider
	if provider == "" {
		if s.registry == nil {
			panic("FeeScheduleService: registry not set (fail-fast: cannot resolve provider)")
		}
		resolved, err := s.registry.ActiveProviderName(ctx)
		if err != nil {
			return nil, fmt.Errorf("fee: resolve active provider for new schedule: %w", err)
		}
		provider = resolved
	}

	entry := domain.DisbursementFeeScheduleEntry{
		ID:              uuid.NewString(),
		Provider:        provider,
		EffectiveDate:   input.EffectiveDate,
		FeeVND:          input.FeeVND,
		Notes:           strings.TrimSpace(input.Notes),
		CreatedAt:       clock.NowUTC(),
		CreatedByUserID: infrastructure.NonZeroUserID(actorUserID),
	}
	if err := entry.Validate(); err != nil {
		return nil, err
	}

	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.DisbursementFeeScheduleEntry) ([]domain.DisbursementFeeScheduleEntry, error) {
		for _, e := range entries {
			if e.Provider == entry.Provider && e.EffectiveDate == entry.EffectiveDate {
				return nil, domain.NewValidationError("a schedule with this effective date already exists for this provider")
			}
		}
		return append(entries, entry), nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.eventBus.Publish(ctx, domain.NewDisbursementFeeScheduleCreatedEvent(ctx, entry.ID, entry.EffectiveDate, FormatFeeAmount(entry.FeeVND))); err != nil {
		s.logger.Warn("Failed to publish DisbursementFeeScheduleCreated event", "id", entry.ID, "error", err)
	}
	return &entry, nil
}

// UpdateInput mirrors CreateInput. Update replaces the entire entry payload
// (apart from ID and audit metadata) so partial updates aren't supported —
// frontend resends the full record.
type UpdateInput struct {
	EffectiveDate string
	FeeVND        int64
	Notes         string
}

// Update mutates a future-dated entry. Past or currently-active entries are
// immutable for audit integrity — admins must add a new entry to change rates
// going forward.
func (s *FeeScheduleService) Update(ctx context.Context, id string, input UpdateInput) (*domain.DisbursementFeeScheduleEntry, error) {
	if err := infrastructure.ValidateEffectiveDateNotPast(s.store.DateLayout, input.EffectiveDate); err != nil {
		return nil, err
	}

	var updated domain.DisbursementFeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.DisbursementFeeScheduleEntry) ([]domain.DisbursementFeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.DisbursementFeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("disbursement fee schedule entry not found")
		}
		if !infrastructure.IsFutureDate(s.store.DateLayout, entries[idx].EffectiveDate) {
			return nil, domain.NewValidationError("only schedules with a future effective date can be edited")
		}
		for i, e := range entries {
			if i == idx {
				continue
			}
			if e.Provider == entries[idx].Provider && e.EffectiveDate == input.EffectiveDate {
				return nil, domain.NewValidationError("a schedule with this effective date already exists for this provider")
			}
		}

		entries[idx].EffectiveDate = input.EffectiveDate
		entries[idx].FeeVND = input.FeeVND
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

	if err := s.eventBus.Publish(ctx, domain.NewDisbursementFeeScheduleUpdatedEvent(ctx, updated.ID, updated.EffectiveDate, FormatFeeAmount(updated.FeeVND))); err != nil {
		s.logger.Warn("Failed to publish DisbursementFeeScheduleUpdated event", "id", updated.ID, "error", err)
	}
	return &updated, nil
}

// Delete removes a future-dated entry. The current and past entries remain
// undeletable — the system must always have at least one schedule that is
// active right now, otherwise fee resolution would fall back to defaults.
func (s *FeeScheduleService) Delete(ctx context.Context, id string) error {
	var removed domain.DisbursementFeeScheduleEntry
	err := s.store.ExecuteWrite(ctx, func(tx *gorm.DB, entries []domain.DisbursementFeeScheduleEntry) ([]domain.DisbursementFeeScheduleEntry, error) {
		idx := infrastructure.IndexOfEntryID(entries, id, func(e domain.DisbursementFeeScheduleEntry) string { return e.ID })
		if idx < 0 {
			return nil, domain.NewNotFoundError("disbursement fee schedule entry not found")
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

	if err := s.eventBus.Publish(ctx, domain.NewDisbursementFeeScheduleDeletedEvent(ctx, removed.ID, removed.EffectiveDate, FormatFeeAmount(removed.FeeVND))); err != nil {
		s.logger.Warn("Failed to publish DisbursementFeeScheduleDeleted event", "id", removed.ID, "error", err)
	}
	return nil
}

// ActiveAt returns the entry that resolves the disbursement fee on date `at`
// for the given provider. Returns NotFoundError if no entry's effective_date
// <= at for this provider.
func (s *FeeScheduleService) ActiveAt(ctx context.Context, provider string, at time.Time) (*domain.DisbursementFeeScheduleEntry, error) {
	entries, err := s.store.LoadEntries(ctx)
	if err != nil {
		return nil, err
	}
	active := domain.ActiveDisbursementFeeScheduleAt(entries, provider, at)
	if active == nil {
		return nil, domain.NewNotFoundError("no disbursement fee schedule active for this date/provider")
	}
	return active, nil
}

// ResolveFee returns the per-transfer disbursement fee in VND for the given
// provider on date `at`. Falls back to a per-provider default on any lookup
// failure with a warning log — fee resolution must never hard-fail on the
// transfer-initiate hot path.
func (s *FeeScheduleService) ResolveFee(ctx context.Context, provider string, at time.Time) int64 {
	entry, err := s.ActiveAt(ctx, provider, at)
	if err != nil {
		fb := fallbackFeeForProvider(provider)
		s.logger.Warn("ResolveFee: no active disbursement fee schedule, falling back",
			"provider", provider, "fallback", fb, "at", at, "error", err)
		return fb
	}
	return entry.FeeVND
}

// GetDisbursementFeeVND satisfies the DisbursementFeeProvider interface
// (declared in wallet_payment_service.go). Resolves the fee at clock.Now()
// for the given provider so existing callers don't need to know about dating.
// fallbackFeeForProvider returns the default fee when the schedule is empty
// or unparseable. Per-provider so OnePay isn't accidentally charged 9Pay's rate.
func fallbackFeeForProvider(provider string) int64 {
	switch provider {
	case "9pay":
		return 200 // matches the observed 9pay rate as of 2026-05
	default:
		return 0
	}
}

// FormatFeeAmount renders an int64 VND amount with thousand separators (3.500.000)
// for use in audit summaries. Mirrors the Vietnamese locale convention used
// by the advance-payment fee formatter.
func FormatFeeAmount(amount int64) string {
	if amount < 0 {
		return fmt.Sprintf("-%s", FormatFeeAmount(-amount))
	}
	s := fmt.Sprintf("%d", amount)
	if len(s) <= 3 {
		return s
	}
	var b strings.Builder
	rem := len(s) % 3
	if rem > 0 {
		b.WriteString(s[:rem])
		if len(s) > rem {
			b.WriteByte('.')
		}
	}
	for i := rem; i < len(s); i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < len(s) {
			b.WriteByte('.')
		}
	}
	return b.String()
}
