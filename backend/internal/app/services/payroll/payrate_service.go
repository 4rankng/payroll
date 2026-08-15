package payroll

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
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
		// Check for constraint violation and return user-friendly error
		if strings.Contains(err.Error(), "Check constraint 'chk_payrates_date_order' is violated") {
			return domain.NewValidationError(constants.MsgCannotUpdatePayrateInvalidDateVN)
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
	// Check if payrate is used by timesheets
	isUsed, err := s.PayrateRepo.IsUsedByTimesheets(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to check payrate usage: %w", err)
	}
	if isUsed {
		return domain.NewValidationError(constants.MsgCannotDeletePayrateUsedByTimesheetsVN)
	}

	payrate, err := s.PayrateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	previousEndDate := payrate.FromDate.AddDate(0, 0, -1)
	previousPayrate, err := s.PayrateRepo.GetByProjectAndToDate(ctx, payrate.ProjectID, previousEndDate)
	if err != nil && !domain.IsNotFoundError(err) {
		return fmt.Errorf("failed to load previous payrate: %w", err)
	}
	if previousPayrate != nil {
		previousPayrate.ToDate = payrate.ToDate
		if err := s.PayrateRepo.Update(ctx, previousPayrate); err != nil {
			return fmt.Errorf("failed to extend previous payrate: %w", err)
		}
	}

	if err := s.PayrateRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete payrate: %w", err)
	}

	// Publish domain event
	event := domain.NewPayrateDeletedEvent(ctx, payrate)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		observability.GetLogger().Warn("failed to publish PayrateDeletedEvent", "error", err)
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

// IsPayrateUsedByTimesheets checks if a payrate has any associated timesheets
func (s *PayrateService) IsPayrateUsedByTimesheets(ctx context.Context, payrateID uint) (bool, error) {
	return s.PayrateRepo.IsUsedByTimesheets(ctx, payrateID)
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
