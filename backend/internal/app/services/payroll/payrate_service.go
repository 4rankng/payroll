package payroll

import (
	"context"
	"fmt"
	"sort"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/services"
	"gorm.io/gorm"
)

type PayrateService struct {
	PayrateRepo     domain.PayrateRepository
	EventBus        domain.EventBus
	TemporalService *services.PayrateTemporalService
}

func NewPayrateService(payrateRepo domain.PayrateRepository, eventBus domain.EventBus, db *gorm.DB) *PayrateService {
	temporalService := services.NewPayrateTemporalService(db, payrateRepo)
	return &PayrateService{
		PayrateRepo:     payrateRepo,
		EventBus:        eventBus,
		TemporalService: temporalService,
	}
}

func (s *PayrateService) CreatePayrate(ctx context.Context, payrate *domain.Payrate, createdBy uint, userRole string) (*domain.Payrate, error) {
	// Validate the payrate using domain validation
	if err := payrate.IsValid(); err != nil {
		return nil, err
	}

	// Use temporal service to handle date range management
	if err := s.TemporalService.CreateEffectiveDatedPayrate(ctx, payrate); err != nil {
		// Preserve domain errors as-is, wrap others
		if _, ok := err.(*domain.DomainError); ok {
			return nil, err
		}
		return nil, fmt.Errorf("failed to create temporal payrate: %w", err)
	}

	// Publish domain event
	event := domain.NewPayrateCreatedEvent(ctx, payrate)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		observability.GetLogger().Warn("failed to publish PayrateCreatedEvent", "error", err)
	}

	return payrate, nil
}

func (s *PayrateService) GetPayrate(ctx context.Context, id uint) (*domain.Payrate, error) {
	return s.PayrateRepo.GetByID(ctx, id)
}

func (s *PayrateService) UpdatePayrate(ctx context.Context, payrate *domain.Payrate, updatedBy uint) error {
	logger := observability.GetLogger()
	logger.Info("PayrateService.UpdatePayrate called",
		"payrate_id", payrate.ID,
		"project_id", payrate.ProjectID,
		"updated_by", updatedBy,
		"from_date", payrate.FromDate.Format("2006-01-02"))

	// Use temporal service to handle the update with date range management
	logger.Info("Calling temporal service to update payrate",
		"payrate_id", payrate.ID)
	if err := s.TemporalService.UpdateEffectiveDatedPayrate(ctx, payrate); err != nil {
		logger.Error("Temporal service update failed",
			"payrate_id", payrate.ID,
			"error", err)
		// Preserve domain errors as-is, wrap others
		if _, ok := err.(*domain.DomainError); ok {
			return err
		}
		return fmt.Errorf("failed to update temporal payrate: %w", err)
	}

	logger.Info("Temporal service update completed successfully",
		"payrate_id", payrate.ID)

	// Publish domain event
	event := domain.NewPayrateUpdatedEvent(ctx, payrate)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		logger.Warn("Failed to publish PayrateUpdatedEvent", "payrate_id", payrate.ID, "error", err)
	}

	logger.Info("PayrateService.UpdatePayrate completed successfully",
		"payrate_id", payrate.ID)
	return nil
}

func (s *PayrateService) DeletePayrate(ctx context.Context, id uint, deletedBy uint) error {
	logger := observability.GetLogger()

	deleted, err := s.TemporalService.DeleteEffectiveDatedPayrate(ctx, id)
	if err != nil {
		if _, ok := err.(*domain.DomainError); ok {
			return err
		}
		return fmt.Errorf("failed to delete temporal payrate: %w", err)
	}

	payrate := deleted

	// Publish domain event
	event := domain.NewPayrateDeletedEvent(ctx, payrate)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		logger.Warn("failed to publish PayrateDeletedEvent", "error", err)
	}

	return nil
}

func (s *PayrateService) ListPayrates(ctx context.Context, filters domain.PayrateFilters) ([]*domain.Payrate, error) {
	return s.PayrateRepo.List(ctx, filters)
}

func (s *PayrateService) CountPayrates(ctx context.Context, filters domain.PayrateFilters) (int64, error) {
	return s.PayrateRepo.Count(ctx, filters)
}

func (s *PayrateService) GetPayratesByProject(ctx context.Context, projectID uint) ([]*domain.Payrate, error) {
	return s.PayrateRepo.GetByProject(ctx, projectID)
}

func (s *PayrateService) GetActivePayrateByProjectAndDate(ctx context.Context, projectID uint, date time.Time) (*domain.Payrate, error) {
	return s.PayrateRepo.GetActiveByProjectAndDate(ctx, projectID, date)
}

func (s *PayrateService) GetCurrentOrUpcomingPayrateByProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	return s.PayrateRepo.GetCurrentOrUpcomingByProject(ctx, projectID)
}

// HasProjectTimesheetsFromDate checks if a project has timesheets on or after a given date
func (s *PayrateService) HasProjectTimesheetsFromDate(ctx context.Context, projectID uint, fromDate time.Time) (bool, error) {
	return s.PayrateRepo.HasProjectTimesheetsFromDate(ctx, projectID, fromDate)
}

// GetLatestTimesheetDateForPayrate returns the most recent timesheet date linked to a payrate.
func (s *PayrateService) GetLatestTimesheetDateForPayrate(ctx context.Context, payrateID uint) (*time.Time, error) {
	return s.PayrateRepo.GetLatestTimesheetDateForPayrate(ctx, payrateID)
}

// GetLatestPaidTimesheetDateForProject returns the financial cutoff for a
// new effective-dated payrate across the whole project.
func (s *PayrateService) GetLatestPaidTimesheetDateForProject(ctx context.Context, projectID uint) (*time.Time, error) {
	return s.TemporalService.GetLatestPaidTimesheetDateForProject(ctx, projectID)
}

// AttachDisplayEndDates fills each payrate's ToDate with its derived end date
// (the successor's from_date minus one day; nil for the latest config).
// From-date-only model: the stored to_date is stale by design, so responses
// always derive it from the timeline. Mutates entries for response assembly
// only — write paths nil out ToDate before persisting.
func (s *PayrateService) AttachDisplayEndDates(ctx context.Context, payrates []*domain.Payrate) error {
	if len(payrates) == 0 {
		return nil
	}

	perProject := make(map[uint][]*domain.Payrate)
	for _, p := range payrates {
		perProject[p.ProjectID] = append(perProject[p.ProjectID], p)
	}

	for projectID := range perProject {
		timeline, err := s.PayrateRepo.GetByProject(ctx, projectID)
		if err != nil {
			return fmt.Errorf("failed to load payrate timeline for display end dates: %w", err)
		}

		// Sort ascending by (from_date, id) so the walk below is immune to the
		// repository's own ordering: each row's successor is the next entry.
		sorted := make([]*domain.Payrate, len(timeline))
		copy(sorted, timeline)
		sort.Slice(sorted, func(a, b int) bool {
			if !sorted[a].FromDate.Equal(sorted[b].FromDate) {
				return sorted[a].FromDate.Before(sorted[b].FromDate)
			}
			return sorted[a].ID < sorted[b].ID
		})

		endByID := make(map[uint]*time.Time, len(sorted))
		for i := range sorted {
			if i == len(sorted)-1 {
				continue // latest config: open-ended
			}
			end := sorted[i+1].FromDate.AddDate(0, 0, -1)
			endByID[sorted[i].ID] = &end
		}

		for _, row := range perProject[projectID] {
			row.ToDate = endByID[row.ID]
		}
	}

	return nil
}

// IsLatestPayrate reports whether the config is still the project's most
// recent one, returning the successor when it is not.
func (s *PayrateService) IsLatestPayrate(ctx context.Context, payrate *domain.Payrate) (bool, *domain.Payrate, error) {
	return s.TemporalService.IsLatestPayrate(ctx, payrate)
}

// GetLatestPayrateByProject returns the project's most recent config, or nil
// when the project has none yet.
func (s *PayrateService) GetLatestPayrateByProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	return s.TemporalService.GetLatestPayrateByProject(ctx, projectID)
}

// HasSiblingPayrateOnOrAfter mirrors the update-side ordering guard: another
// live config already starts on or after the proposed date.
func (s *PayrateService) HasSiblingPayrateOnOrAfter(ctx context.Context, projectID, excludeID uint, date time.Time) (bool, error) {
	return s.TemporalService.HasSiblingPayrateOnOrAfter(ctx, projectID, excludeID, date)
}

// CountMutableTimesheetsOnOrAfter counts the timesheets a save effective from
// the given date would re-price.
func (s *PayrateService) CountMutableTimesheetsOnOrAfter(ctx context.Context, projectID uint, fromDate time.Time) (int64, error) {
	return s.TemporalService.CountMutableTimesheetsOnOrAfter(ctx, projectID, fromDate)
}
