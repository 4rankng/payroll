package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"
	dbhelper "api-server/internal/pkg/db"
	"api-server/internal/pkg/retry"

	"gorm.io/gorm"
)

// MaxTemporalDate represents the far future date used for open-ended payrates
// Using a deterministic constant instead of time.Now() + 100 years for consistency
var MaxTemporalDate = time.Date(2999, 12, 31, 0, 0, 0, 0, time.UTC)

// Date utility functions to avoid repetitive conversions
func toUTCDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func formatDateString(t time.Time) string {
	return t.Format("2006-01-02")
}

// PayrateTemporalService provides temporal data management specifically for payrates.
//
// This service handles the complex business rules around payrate effective dating:
// - Ensures temporal consistency across payrate date ranges
// - Manages conflicts between overlapping payrate periods
// - Enforces restrictions when payrates have approved timesheets attached
// - Provides concurrency-safe operations through database transactions
//
// IMPORTANT: This service assumes single-project operations per transaction.
// For multi-project operations, handle concurrency at the application level.
type PayrateTemporalService struct {
	db          *gorm.DB
	payrateRepo domain.PayrateRepository
	dbHelper    *dbhelper.DatabaseHelper
	retryConfig retry.DatabaseOperationConfig
}

// NewPayrateTemporalService creates a new payrate temporal service
func NewPayrateTemporalService(db *gorm.DB, payrateRepo domain.PayrateRepository) *PayrateTemporalService {
	dbHelper := dbhelper.NewDatabaseHelper(db)
	return &PayrateTemporalService{
		db:          db,
		payrateRepo: payrateRepo,
		dbHelper:    dbHelper,
		retryConfig: retry.DefaultDatabaseConfig(),
	}
}

// CreateEffectiveDatedPayrate implements the temporal algorithm for payrates
func (s *PayrateTemporalService) CreateEffectiveDatedPayrate(ctx context.Context, payrate *domain.Payrate) error {
	if err := s.validateEffectiveDate(ctx, payrate.ProjectID, payrate.FromDate); err != nil {
		return err
	}

	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		if err := s.manageActivePayrates(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		payrate.ToDate = nil // Open-ended
		return s.payrateRepo.CreateWithTx(ctx, tx, payrate)
	})
}

// GetActivePayrateForProject retrieves the currently active payrate for a project
func (s *PayrateTemporalService) GetActivePayrateForProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	activePayrates, err := s.payrateRepo.FindActiveByProject(ctx, nil, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active payrate: %w", err)
	}

	if len(activePayrates) == 0 {
		return nil, domain.NewNotFoundError(constants.MsgPayrateNotFoundVN)
	}

	// Return the first (earliest) active payrate if multiple exist
	return activePayrates[0], nil
}

// GetActivePayrateOnDate retrieves the active payrate for a project on a specific date
func (s *PayrateTemporalService) GetActivePayrateOnDate(ctx context.Context, projectID uint, date time.Time) (*domain.Payrate, error) {
	// Use UTC date-only comparison to avoid timezone issues
	date = toUTCDateOnly(date)

	var payrate domain.Payrate
	err := s.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		return db.Where("project_id = ? AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)",
			projectID, date, date).
			Order("from_date DESC").
			First(&payrate).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError(constants.MsgPayrateNotFoundVN)
		}
		return nil, s.dbHelper.WrapDatabaseError(err)
	}

	return &payrate, nil
}

// EndActivePayrateForProject manually ends the currently active payrate for a project
func (s *PayrateTemporalService) EndActivePayrateForProject(ctx context.Context, projectID uint, endDate time.Time) error {
	// Use UTC date-only comparison to avoid timezone issues
	endDate = toUTCDateOnly(endDate)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activePayrates, err := s.payrateRepo.FindActiveByProject(ctx, tx, projectID)
		if err != nil {
			return fmt.Errorf("failed to find active payrate: %w", err)
		}

		if len(activePayrates) == 0 {
			return domain.NewNotFoundError(constants.MsgPayrateNotFoundVN)
		}

		activePayrate := activePayrates[0] // Use the first active payrate

		// Validate end date is after start date
		if endDate.Before(activePayrate.FromDate) || endDate.Equal(activePayrate.FromDate) {
			return domain.NewValidationError(constants.MsgFromDateAfterToDateVN)
		}

		activePayrate.ToDate = &endDate
		return tx.Save(activePayrate).Error
	})
}

// UpdateEffectiveDatedPayrate handles updates to payrates following these rules:
// 1. If payrate has attached timesheets (any status), can only update end date to today onward
// 2. If no timesheets attached, can update anything
// 3. Date range conflicts with other payrates that have timesheets (any status) reject the change
// 4. Otherwise user's date range takes priority and may delete/modify other payrates
func (s *PayrateTemporalService) UpdateEffectiveDatedPayrate(ctx context.Context, payrate *domain.Payrate) error {
	logger := observability.GetLogger()

	// Get existing payrate to validate
	var existingPayrate *domain.Payrate
	err := s.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		var err error
		existingPayrate, err = s.payrateRepo.GetByID(ctx, payrate.ID)
		return err
	})
	if err != nil {
		logger.Error("Failed to get existing payrate", "payrate_id", payrate.ID, "error", err)
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to get existing payrate: %w", err))
	}

	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Check if this payrate has any attached timesheets (any status)
		timesheetCount, err := s.getTimesheetCount(ctx, tx, payrate.ID)
		if err != nil {
			logger.Error("Failed to check timesheet usage", "payrate_id", payrate.ID, "error", err)
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check timesheet usage: %w", err))
		}

		hasTimesheets := timesheetCount > 0

		if hasTimesheets {
			// Rule 1 & 2: If has any timesheets, validate restrictions
			if err := s.validatePayrateWithTimesheets(ctx, tx, payrate, existingPayrate); err != nil {
				return err
			}
		} else {
			// Rule 3 & 4: No timesheets - check for conflicts with other payrates
			if err := s.handlePayrateConflicts(ctx, tx, payrate); err != nil {
				return err
			}
		}

		// Update the existing record with new data
		if err := tx.Save(payrate).Error; err != nil {
			logger.Error("Failed to save payrate", "payrate_id", payrate.ID, "error", err)
			return s.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Payrate updated successfully", "payrate_id", payrate.ID)
		return nil
	})
}

// dateRangesOverlap checks if two payrate date ranges overlap.
//
// Uses deterministic MaxTemporalDate for open-ended payrates to ensure
// consistent behavior regardless of when the function is called.
//
// Algorithm: Two ranges [a,b] and [c,d] overlap if !(b < c || d < a)
func (s *PayrateTemporalService) dateRangesOverlap(p1, p2 *domain.Payrate) bool {
	p1Start := toUTCDateOnly(p1.FromDate)
	p1End := MaxTemporalDate // Use deterministic far future date
	if p1.ToDate != nil {
		p1End = toUTCDateOnly(*p1.ToDate)
	}

	p2Start := toUTCDateOnly(p2.FromDate)
	p2End := MaxTemporalDate // Use deterministic far future date
	if p2.ToDate != nil {
		p2End = toUTCDateOnly(*p2.ToDate)
	}

	// Ranges overlap if one starts before the other ends
	return !p1End.Before(p2Start) && !p2End.Before(p1Start)
}

// ValidatePayrateTemporalIntegrity performs comprehensive validation of payrate temporal data
func (s *PayrateTemporalService) ValidatePayrateTemporalIntegrity(ctx context.Context, projectID uint) error {
	logger := observability.GetLogger()

	// Check 1: Validate date order constraints using repository
	invalidDateCount, err := s.payrateRepo.ValidateDateOrdering(ctx, nil, projectID)
	if err != nil {
		logger.Error("Failed to check date order constraints", "project_id", projectID, "error", err)
		return fmt.Errorf("failed to validate date constraints: %w", err)
	}
	if invalidDateCount > 0 {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateInvalidDateVN)
	}

	// Check 2: Ensure at most one active (open-ended) payrate using repository
	activeCount, err := s.payrateRepo.ValidateActiveCount(ctx, nil, projectID)
	if err != nil {
		logger.Error("Failed to count active payrates", "project_id", projectID, "error", err)
		return fmt.Errorf("failed to count active payrates: %w", err)
	}
	if activeCount > 1 {
		return domain.NewValidationError(constants.MsgInvalidPayrateConfigurationVN)
	}

	return nil
}

// Private helper methods

// getTimesheetCount returns the count of all timesheets for a given payrate (any status)
func (s *PayrateTemporalService) getTimesheetCount(ctx context.Context, tx *gorm.DB, payrateID uint) (int64, error) {
	var count int64
	err := tx.Model(&domain.Timesheet{}).
		Where("payrate_id = ?", payrateID).
		Count(&count).Error
	return count, err
}

// hasProjectTimesheetsFromDateTx checks if a project has any timesheets on or after the specified date within a transaction
func (s *PayrateTemporalService) hasProjectTimesheetsFromDateTx(ctx context.Context, tx *gorm.DB, projectID uint, fromDate time.Time) (bool, error) {
	var count int64
	err := tx.Model(&domain.Timesheet{}).
		Where("project_id = ? AND date >= ?", projectID, fromDate).
		Count(&count).Error
	return count > 0, err
}

// validatePayrateWithTimesheets validates updates to payrates that have any attached timesheets
func (s *PayrateTemporalService) validatePayrateWithTimesheets(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate, existingPayrate *domain.Payrate) error {
	logger := observability.GetLogger()
	today := toUTCDateOnly(clock.Now().UTC())

	logger.Info("Payrate has attached timesheets, validating update restrictions",
		"payrate_id", payrate.ID,
		"project_id", payrate.ProjectID)

	// Check if project has any timesheets from the new effective date onwards
	newFromDate := toUTCDateOnly(payrate.FromDate)
	hasProjectTimesheetsFromNewDate, err := s.hasProjectTimesheetsFromDateTx(ctx, tx, payrate.ProjectID, newFromDate)
	if err != nil {
		logger.Error("Failed to check project timesheets from new date",
			"payrate_id", payrate.ID,
			"project_id", payrate.ProjectID,
			"from_date", newFromDate,
			"error", err)
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check project timesheets from date: %w", err))
	}

	// If no project timesheets exist from the new start date onwards, allow all changes
	if !hasProjectTimesheetsFromNewDate {
		logger.Info("No project timesheets exist from new effective date, allowing update",
			"payrate_id", payrate.ID,
			"project_id", payrate.ProjectID,
			"new_from_date", newFromDate.Format("2006-01-02"))
		return nil
	}

	// Project timesheets exist from the new date onwards - apply restrictions
	logger.Info("Project timesheets exist from new effective date, applying restrictions",
		"payrate_id", payrate.ID,
		"project_id", payrate.ProjectID,
		"new_from_date", newFromDate.Format("2006-01-02"))

	// Cannot change payrate configuration
	existingJSON := string(existingPayrate.Payrate)
	newJSON := string(payrate.Payrate)
	if existingJSON != newJSON {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateWithTimesheetsVN)
	}

	// Cannot change start date
	if !newFromDate.Equal(toUTCDateOnly(existingPayrate.FromDate)) {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateWithTimesheetsVN)
	}

	// Can only set end date to today or later
	if payrate.ToDate != nil {
		toDateOnly := toUTCDateOnly(*payrate.ToDate)
		if toDateOnly.Before(today) {
			return domain.NewValidationError(constants.MsgCannotUpdatePayrateWithTimesheetsVN)
		}
	}

	return nil
}

// handlePayrateConflicts processes conflicts with other payrates when updating
func (s *PayrateTemporalService) handlePayrateConflicts(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	logger := observability.GetLogger()

	logger.Info("No timesheets attached, checking for conflicts with other payrates",
		"payrate_id", payrate.ID)

	// Get all other payrates for this project
	var otherPayrates []*domain.Payrate
	if err := tx.Where("project_id = ? AND id != ?", payrate.ProjectID, payrate.ID).
		Find(&otherPayrates).Error; err != nil {
		logger.Error("Failed to get other payrates", "project_id", payrate.ProjectID, "error", err)
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to get other payrates: %w", err))
	}

	// Check each other payrate for conflicts
	for _, other := range otherPayrates {
		// Check if this other payrate has any timesheets
		otherTimesheetCount, err := s.getTimesheetCount(ctx, tx, other.ID)
		if err != nil {
			logger.Error("Failed to check other payrate timesheet usage", "payrate_id", other.ID, "error", err)
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check timesheet usage: %w", err))
		}

		if otherTimesheetCount > 0 {
			// Rule: Check for date range conflict with payrates that have timesheets
			if s.dateRangesOverlap(payrate, other) {
				return domain.NewValidationError(constants.MsgCannotUpdatePayrateInvalidDateVN)
			}
		} else {
			// Rule: User's date range takes priority - handle overlapping payrates without timesheets
			if s.dateRangesOverlap(payrate, other) {
				logger.Info("Deleting overlapping payrate without timesheets",
					"deleted_payrate_id", other.ID,
					"updated_payrate_id", payrate.ID)

				// Delete the conflicting payrate that has no timesheets
				if err := tx.Delete(other).Error; err != nil {
					logger.Error("Failed to delete conflicting payrate", "payrate_id", other.ID, "error", err)
					return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to delete conflicting payrate: %w", err))
				}
			}
		}
	}

	return nil
}

// validateEffectiveDate checks if a date is valid for payrate creation
// Allows past dates only if no timesheets exist for the project from that date to present
func (s *PayrateTemporalService) validateEffectiveDate(ctx context.Context, projectID uint, date time.Time) error {
	today := toUTCDateOnly(clock.Now().UTC())
	dateOnly := toUTCDateOnly(date)

	// Allow today or future dates (existing behavior)
	if !dateOnly.Before(today) {
		return nil
	}

	// For past dates, check if timesheets exist from that date to present
	hasTimesheets, err := s.payrateRepo.HasProjectTimesheetsFromDate(ctx, projectID, dateOnly)
	if err != nil {
		return fmt.Errorf("failed to check for existing timesheets: %w", err)
	}

	if hasTimesheets {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateInvalidDateVN)
	}

	return nil
}

// manageActivePayrates validates and manages existing active payrates using repository
func (s *PayrateTemporalService) manageActivePayrates(ctx context.Context, tx *gorm.DB, projectID uint, newFromDate time.Time) error {
	logger := observability.GetLogger()

	// Get existing active payrates using repository
	activePayrates, err := s.payrateRepo.FindActiveByProject(ctx, tx, projectID)
	if err != nil {
		logger.Error("Failed to get active payrates", "project_id", projectID, "error", err)
		return fmt.Errorf("failed to get active payrates: %w", err)
	}

	if len(activePayrates) == 0 {
		return nil // Nothing to manage
	}

	// Check for duplicate dates
	for _, p := range activePayrates {
		if newFromDate.Equal(p.FromDate) {
			return domain.NewValidationError(fmt.Sprintf(constants.MsgPayrateAlreadyExistsForDateVN, formatDateString(p.FromDate)))
		}
	}

	// Determine if new payrate is earlier or later than existing ones
	earliestExisting := activePayrates[0].FromDate
	for _, p := range activePayrates {
		if p.FromDate.Before(earliestExisting) {
			earliestExisting = p.FromDate
		}
	}

	if newFromDate.Before(earliestExisting) {
		// Earlier payrate replaces all future ones - delete them (no timesheets since all dates are future)
		if err := s.payrateRepo.DeleteActiveByProject(ctx, tx, projectID); err != nil {
			logger.Error("Failed to delete later payrates", "project_id", projectID, "error", err)
			return fmt.Errorf("failed to delete later payrates: %w", err)
		}
		logger.Info("Deleted later payrates for earlier replacement", "project_id", projectID)
	} else {
		// Later payrate - close existing ones with end date
		toDate := newFromDate.AddDate(0, 0, -1)
		if err := s.payrateRepo.CloseActiveByProject(ctx, tx, projectID, toDate); err != nil {
			logger.Error("Failed to close active payrates", "project_id", projectID, "error", err)
			return fmt.Errorf("failed to close active payrates: %w", err)
		}
		logger.Info("Closed active payrates", "project_id", projectID, "to_date", formatDateString(toDate))
	}

	return nil
}
