package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	dbhelper "api-server/internal/pkg/db"
	"api-server/internal/pkg/retry"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Date utility functions to avoid repetitive conversions
// toLocalDateOnly normalizes date-only values in time.Local. Under a loc=Local
// MySQL DSN the driver converts bound time.Time values to Local before
// sending, so a UTC-midnight bound shifts +7h on the UTC+7 box and silently
// misses same-day DATE rows (timesheet dates are stored at local midnight).
// Any value compared against stored timesheet/payrate dates must use this.
func toLocalDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
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
	payrate.FromDate = toLocalDateOnly(payrate.FromDate)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, payrate.ProjectID); err != nil {
			return err
		}

		if err := s.validateEffectiveDateTx(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		if err := s.manageActivePayrates(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		payrate.ToDate = nil // Open-ended
		if err := s.payrateRepo.CreateWithTx(ctx, tx, payrate); err != nil {
			return err
		}

		return s.recalculateMutableTimesheets(ctx, tx, payrate)
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
	date = toLocalDateOnly(date)

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
	endDate = toLocalDateOnly(endDate)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, projectID); err != nil {
			return err
		}

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
//  1. The new start date must fall after the project's most recent paid timesheet work date
//  2. The payrate must remain the project's most recent configuration
//     (no sibling payrate may start on or after the new start date)
//  3. Moving the start date later follows the create model: the existing config
//     is closed the day before and a new open-ended config takes over, so rows
//     already priced under the old config keep it (update-as-create split)
//  4. Mutable timesheets inside the config's reign are recalculated in the same
//     transaction. The config in effect for a work date is always the project
//     payrate with the latest from_date on or before that date.
func (s *PayrateTemporalService) UpdateEffectiveDatedPayrate(ctx context.Context, payrate *domain.Payrate) error {
	logger := observability.GetLogger()
	payrate.FromDate = toLocalDateOnly(payrate.FromDate)
	// End dates are not part of the model — configs close only via the
	// timeline. Mirror CreateEffectiveDatedPayrate's normalization so no
	// caller can persist one and mark the config "ended" by our own guard.
	payrate.ToDate = nil

	// Capture the target row's ID outside the transaction closure: the closure
	// is retried on transient errors, and splitUpdateTx rewrites payrate.ID when
	// it inserts the replacement config — a retried attempt must still load the
	// original row, not the rolled-back replacement's ID (or zero).
	targetID := payrate.ID

	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, payrate.ProjectID); err != nil {
			return err
		}

		var existing domain.Payrate
		if err := tx.WithContext(ctx).First(&existing, targetID).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to load existing payrate: %w", err))
		}

		// Ended configs are history: their rows are priced and immutable, and a
		// newer config owns the project's present. Reject before the floor or
		// sibling checks, whose generic errors would only confuse here.
		if existing.ToDate != nil {
			return domain.NewValidationError(constants.MsgCannotUpdateEndedPayrateVN)
		}

		if err := s.validateEffectiveDateTx(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		if payrate.FromDate.After(toLocalDateOnly(existing.FromDate)) {
			return s.splitUpdateTx(ctx, tx, payrate, &existing)
		}

		if err := s.ensureLatestConfigTx(ctx, tx, payrate); err != nil {
			return err
		}

		if err := s.ensureEarliestPaidTimesheetCoveredTx(ctx, tx, payrate); err != nil {
			return err
		}

		if err := s.synchronizePredecessorEndDateTx(ctx, tx, payrate); err != nil {
			return err
		}

		if err := tx.Save(payrate).Error; err != nil {
			logger.Error("Failed to save payrate", "payrate_id", payrate.ID, "error", err)
			return s.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Payrate updated successfully", "payrate_id", payrate.ID)
		return s.recalculateMutableTimesheets(ctx, tx, payrate)
	})
}

// splitUpdateTx applies the update-as-create model when a config's start date
// moves later. Timesheets already linked to the config — priced and possibly
// paid under it — must keep that config, so instead of re-dating the row (which
// would strand those rows before their config's reign), the existing config is
// closed the day before the new start and the submitted configuration becomes a
// new open-ended row. Mutable rows on/after the new start are re-linked and
// re-priced under the new config.
func (s *PayrateTemporalService) splitUpdateTx(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate, existing *domain.Payrate) error {
	if err := s.ensureLatestConfigTx(ctx, tx, payrate); err != nil {
		return err
	}

	// Close the existing config before inserting the new open-ended row; both
	// open at once violates uq_payrates_one_open_per_project (error 1062).
	if err := s.closePayrateTx(ctx, tx, existing.ID, payrate.FromDate); err != nil {
		return fmt.Errorf("failed to close previous config: %w", err)
	}

	// Insert as a fresh row so immutable timesheets keep pointing at the config
	// that priced them. Build from a local copy and only publish the new ID on
	// the caller's object after the insert succeeds: the enclosing closure can
	// be retried on transient errors, and a retried attempt must see payrate
	// un-mutated enough to re-load the original row.
	created := *payrate
	created.ID = 0
	created.CreatedAt = time.Time{}
	created.ToDate = nil // Open-ended
	if err := s.payrateRepo.CreateWithTx(ctx, tx, &created); err != nil {
		return err
	}
	payrate.ID = created.ID

	observability.GetLogger().Info("payrate update split into new config",
		"project_id", payrate.ProjectID,
		"previous_payrate_id", existing.ID,
		"new_payrate_id", payrate.ID,
		"from_date", formatDateString(payrate.FromDate),
	)
	return s.recalculateMutableTimesheets(ctx, tx, payrate)
}

// closePayrateTx ends a config's reign the day before nextStart, keeping the
// timeline boundary invariant (to_date = next from_date - 1) in one place for
// both the update-as-create split and predecessor synchronization.
func (s *PayrateTemporalService) closePayrateTx(ctx context.Context, tx *gorm.DB, payrateID uint, nextStart time.Time) error {
	endDate := toLocalDateOnly(nextStart).AddDate(0, 0, -1)
	if err := tx.WithContext(ctx).Model(&domain.Payrate{}).Where("id = ?", payrateID).
		Update("to_date", endDate).Error; err != nil {
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to set to_date for payrate %d: %w", payrateID, err))
	}
	return nil
}

// Private helper methods

// lockProjectTimelineTx serializes all payrate timeline changes for one project.
// Locking the project row works even when the project has no payrates yet, unlike
// locking only existing payrate rows.
func (s *PayrateTemporalService) lockProjectTimelineTx(ctx context.Context, tx *gorm.DB, projectID uint) error {
	var project domain.Project
	if err := tx.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id").
		First(&project, projectID).Error; err != nil {
		return fmt.Errorf("failed to lock payrate timeline for project %d: %w", projectID, err)
	}
	return nil
}

// synchronizePredecessorEndDateTx preserves a non-overlapping visible timeline
// whenever the most recent configuration is moved. Rate resolution already uses
// the latest from_date; this keeps to_date accurate for history and reporting.
func (s *PayrateTemporalService) synchronizePredecessorEndDateTx(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	var predecessor domain.Payrate
	err := tx.WithContext(ctx).
		Where("project_id = ? AND id <> ? AND from_date < ?", payrate.ProjectID, payrate.ID, toLocalDateOnly(payrate.FromDate)).
		Order("from_date DESC, id DESC").
		First(&predecessor).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to find preceding payrate: %w", err))
	}

	if err := s.closePayrateTx(ctx, tx, predecessor.ID, payrate.FromDate); err != nil {
		return fmt.Errorf("failed to synchronize preceding payrate: %w", err)
	}
	return nil
}

// ensureLatestConfigTx enforces the single-config-per-date model: the payrate
// being updated must remain the project's most recent configuration, so no
// sibling payrate may start on or after the new from_date. The config applied
// to a work date is the project payrate with the latest from_date on or before
// that date; effective_to is bookkeeping, never a resolution input.
func (s *PayrateTemporalService) ensureLatestConfigTx(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	var count int64
	err := tx.WithContext(ctx).
		Model(&domain.Payrate{}).
		Where("project_id = ? AND id <> ? AND from_date >= ?", payrate.ProjectID, payrate.ID, toLocalDateOnly(payrate.FromDate)).
		Count(&count).Error
	if err != nil {
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check sibling payrates: %w", err))
	}
	if count > 0 {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateConflictingConfigVN)
	}
	return nil
}

// ensureEarliestPaidTimesheetCoveredTx blocks in-place start-date moves that
// would strand timesheets already linked to this config: immutable rows
// reference the config they were priced under, so a from_date later than the
// earliest linked row would silently reassign them to an older sibling config
// in date-based resolution. Start dates moving later go through
// splitUpdateTx instead, which keeps the linked rows on the closed config.
func (s *PayrateTemporalService) ensureEarliestPaidTimesheetCoveredTx(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	var earliest struct {
		MinDate *time.Time
	}
	err := tx.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("MIN(date) as min_date").
		Where("project_id = ? AND payrate_id = ?", payrate.ProjectID, payrate.ID).
		Scan(&earliest).Error
	if err != nil {
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to find earliest linked timesheet: %w", err))
	}
	if earliest.MinDate == nil {
		return nil
	}
	if toLocalDateOnly(payrate.FromDate).After(*earliest.MinDate) {
		return domain.NewValidationError(constants.MsgCannotUpdatePayratePastLinkedTimesheetVN)
	}
	return nil
}

// GetLatestPaidTimesheetDateForProject returns the latest work date among paid
// timesheets. That date establishes the project's immutable payrate boundary;
// the payment transfer timestamp does not change it. Unapproved or unpaid
// entries remain eligible for recalculation by a later configuration.
func (s *PayrateTemporalService) GetLatestPaidTimesheetDateForProject(ctx context.Context, projectID uint) (*time.Time, error) {
	return s.getLatestPaidTimesheetDate(ctx, s.db, projectID)
}

func (s *PayrateTemporalService) validateEffectiveDateTx(ctx context.Context, tx *gorm.DB, projectID uint, date time.Time) error {
	latestPaidDate, err := s.getLatestPaidTimesheetDate(ctx, tx, projectID)
	if err != nil {
		return fmt.Errorf("failed to find latest paid timesheet: %w", err)
	}
	if latestPaidDate == nil {
		return nil
	}

	// Wall-clock comparison: converting the stored date to UTC first would
	// shift it back a day under a loc=Local MySQL DSN (GMT+7/+8).
	earliestDate := toLocalDateOnly(*latestPaidDate).AddDate(0, 0, 1)
	if toLocalDateOnly(date).Before(earliestDate) {
		return domain.NewValidationError(constants.MsgCannotUpdatePayrateInvalidDateVN)
	}

	return nil
}

func (s *PayrateTemporalService) getLatestPaidTimesheetDate(ctx context.Context, db *gorm.DB, projectID uint) (*time.Time, error) {
	var timesheet domain.Timesheet
	err := db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("date").
		Where("project_id = ? AND payment_status = ?", projectID, domain.PaymentStatusPaid).
		Order("date DESC").
		First(&timesheet).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &timesheet.Date, nil
}

// recalculateMutableTimesheets runs in the same transaction as the newly
// effective payrate. It keeps paid and approved records immutable, including
// when either state changes concurrently with this operation.
func (s *PayrateTemporalService) recalculateMutableTimesheets(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	var project domain.Project
	if err := tx.WithContext(ctx).Select("id", "is_flexible").First(&project, payrate.ProjectID).Error; err != nil {
		return fmt.Errorf("failed to load project for timesheet recalculation: %w", err)
	}

	var timesheets []*domain.Timesheet
	if err := tx.WithContext(ctx).
		Where("project_id = ? AND date >= ? AND payment_status <> ? AND timesheet_status <> ?",
			payrate.ProjectID,
			toLocalDateOnly(payrate.FromDate),
			domain.PaymentStatusPaid,
			domain.TimesheetStatusApproved,
		).
		Find(&timesheets).Error; err != nil {
		return fmt.Errorf("failed to load mutable timesheets: %w", err)
	}

	updatedCount := 0
	for _, timesheet := range timesheets {
		rate, err := payrate.Payrate.GetRate(timesheet.PayType)
		if err != nil {
			return domain.NewValidationError(fmt.Sprintf("bảng công %d có loại lương không tồn tại trong cấu hình mới", timesheet.ID))
		}

		amount := int64(timesheet.HoursWorked * float64(rate))
		if project.IsFlexible {
			amount = int64(rate)
		}

		result := tx.WithContext(ctx).
			Model(&domain.Timesheet{}).
			Where("id = ? AND payment_status <> ? AND timesheet_status <> ?",
				timesheet.ID,
				domain.PaymentStatusPaid,
				domain.TimesheetStatusApproved,
			).
			Updates(map[string]any{
				"payrate_id": payrate.ID,
				"payrate":    int64(rate),
				"amount":     amount,
			})
		if result.Error != nil {
			return fmt.Errorf("failed to recalculate timesheet %d: %w", timesheet.ID, result.Error)
		}
		updatedCount += int(result.RowsAffected)
	}

	observability.GetLogger().Info("recalculated mutable timesheets for payrate change",
		"project_id", payrate.ProjectID,
		"payrate_id", payrate.ID,
		"from_date", formatDateString(payrate.FromDate),
		"updated_count", updatedCount,
	)
	return nil
}

// manageActivePayrates validates and manages existing active payrates using repository
func (s *PayrateTemporalService) manageActivePayrates(ctx context.Context, tx *gorm.DB, projectID uint, newFromDate time.Time) error {
	logger := observability.GetLogger()
	newFromDate = toLocalDateOnly(newFromDate)

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
		if newFromDate.Equal(toLocalDateOnly(p.FromDate)) {
			return domain.NewValidationError(fmt.Sprintf(constants.MsgPayrateAlreadyExistsForDateVN, formatDateString(p.FromDate)))
		}
	}

	// Determine if new payrate is earlier or later than existing ones
	earliestExisting := toLocalDateOnly(activePayrates[0].FromDate)
	for _, p := range activePayrates {
		fromDate := toLocalDateOnly(p.FromDate)
		if fromDate.Before(earliestExisting) {
			earliestExisting = fromDate
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
