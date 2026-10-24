package services

import (
	"context"
	"errors"
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

// PayrateTemporalService manages the project payrate timeline under the
// from-date-only model: the config in force for a work date is the project's
// latest from_date on or before it, a config's end is implied by its
// successor, and to_date is derived bookkeeping that is never written.
//
// Write rules:
//   - Create appends a new open-ended config strictly after the latest one.
//   - Update corrects the latest config in place (rates and/or from_date),
//     re-pricing unpaid and unapproved timesheets in the same transaction.
//   - Delete soft-deletes and lets resolution fall back to the covering
//     config; mutable linked timesheets are re-priced under it first.
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

// CreateEffectiveDatedPayrate appends a new version to the project's payrate
// timeline. The row is inserted open-ended and must start strictly after the
// latest existing config: an earlier start would rewrite history that
// timesheets were priced under. Correcting the present goes through the
// update flow, which re-prices mutable timesheets in place.
func (s *PayrateTemporalService) CreateEffectiveDatedPayrate(ctx context.Context, payrate *domain.Payrate) error {
	payrate.FromDate = toLocalDateOnly(payrate.FromDate)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, payrate.ProjectID); err != nil {
			return err
		}

		if err := s.validateEffectiveDateTx(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		if err := s.ensureCreateAfterLatestTx(ctx, tx, payrate); err != nil {
			return err
		}

		if err := s.payrateRepo.CreateWithTx(ctx, tx, payrate); err != nil {
			return err
		}

		return s.recalculateMutableTimesheets(ctx, tx, payrate)
	})
}

// UpdateEffectiveDatedPayrate corrects the project's latest configuration in
// place: rates and/or from_date, with unpaid and unapproved timesheets
// re-priced in the same transaction. From-date-only model — there is no split:
// scheduling a future change is the create flow, and replaced configs are
// view-only. Guards, in order:
//  1. the row must still be the project's latest config
//  2. the paid floor: start date after the most recent paid timesheet
//  3. the new from_date must not collide with or pass any sibling config
//  4. linked timesheets keep their config: the start date cannot move past
//     the earliest work date priced under it
func (s *PayrateTemporalService) UpdateEffectiveDatedPayrate(ctx context.Context, payrate *domain.Payrate) error {
	logger := observability.GetLogger()
	payrate.FromDate = toLocalDateOnly(payrate.FromDate)

	// Capture the target row's ID outside the transaction closure: the closure
	// is retried on transient errors and must always reload the original row.
	targetID := payrate.ID

	return s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, payrate.ProjectID); err != nil {
			return err
		}

		var existing domain.Payrate
		if err := tx.WithContext(ctx).First(&existing, targetID).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to load existing payrate: %w", err))
		}

		if err := s.ensureRowIsLatestTx(ctx, tx, &existing); err != nil {
			return err
		}

		if err := s.validateEffectiveDateTx(ctx, tx, payrate.ProjectID, payrate.FromDate); err != nil {
			return err
		}

		if err := s.ensureLatestConfigTx(ctx, tx, payrate); err != nil {
			return err
		}

		if err := s.ensureEarliestPaidTimesheetCoveredTx(ctx, tx, payrate); err != nil {
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

// DeleteEffectiveDatedPayrate soft-deletes one config and lets the timeline
// heal: resolution falls back to the covering config by date, with no
// end-date rows to repair. Mutable timesheets linked to the deleted row are
// re-linked and re-priced under that covering config in the same
// transaction; approved or paid links block the delete outright, because
// those rows keep the config that priced them.
func (s *PayrateTemporalService) DeleteEffectiveDatedPayrate(ctx context.Context, payrateID uint) (*domain.Payrate, error) {
	// Project ID is needed to take the timeline lock; load it before opening
	// the transaction, then re-read the row under the lock.
	var ref domain.Payrate
	if err := s.db.WithContext(ctx).Select("id", "project_id").First(&ref, payrateID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError(constants.MsgPayrateNotFoundVN)
		}
		return nil, s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to load payrate: %w", err))
	}

	var deleted *domain.Payrate
	err := s.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		if err := s.lockProjectTimelineTx(ctx, tx, ref.ProjectID); err != nil {
			return err
		}

		var payrate domain.Payrate
		if err := tx.WithContext(ctx).First(&payrate, payrateID).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to load payrate: %w", err))
		}

		var immutable int64
		if err := tx.WithContext(ctx).Model(&domain.Timesheet{}).
			Where("payrate_id = ? AND (payment_status = ? OR timesheet_status = ?)",
				payrate.ID, domain.PaymentStatusPaid, domain.TimesheetStatusApproved).
			Count(&immutable).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to count linked timesheets: %w", err))
		}
		if immutable > 0 {
			return domain.NewValidationErrorWithCode(domain.ErrCodePayrateImmutableLinked, constants.MsgCannotDeletePayrateApprovedLinkedVN)
		}

		var linked int64
		if err := tx.WithContext(ctx).Model(&domain.Timesheet{}).
			Where("payrate_id = ? AND payment_status <> ? AND timesheet_status <> ?",
				payrate.ID, domain.PaymentStatusPaid, domain.TimesheetStatusApproved).
			Count(&linked).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to count linked timesheets: %w", err))
		}

		if linked > 0 {
			var covering domain.Payrate
			err := tx.WithContext(ctx).
				Where("project_id = ? AND id <> ? AND from_date < ?", payrate.ProjectID, payrate.ID, toLocalDateOnly(payrate.FromDate)).
				Order("from_date DESC, id DESC").
				First(&covering).Error
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					// No earlier config exists: the linked rows would lose
					// their price reference entirely.
					return domain.NewValidationError(constants.MsgCannotDeletePayrateUsedByTimesheetsVN)
				}
				return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to find covering payrate: %w", err))
			}
			// AfterFind is not reliable for this model — unmarshal manually,
			// same as every repository read path.
			if covering.PayrateJSON != "" {
				covering.Payrate = domain.PayrateConfiguration(covering.PayrateJSON)
			}
			if err := s.repriceTimesheetsLinkedToTx(ctx, tx, payrate.ID, &covering); err != nil {
				return err
			}
		}

		if err := tx.WithContext(ctx).Delete(&payrate).Error; err != nil {
			return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to delete payrate: %w", err))
		}

		observability.GetLogger().Info("self-healing timeline: payrate soft-deleted",
			"project_id", payrate.ProjectID,
			"payrate_id", payrate.ID,
		)
		deleted = &payrate
		return nil
	})
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

// GetLatestPaidTimesheetDateForProject returns the latest work date among paid
// timesheets. That date establishes the project's immutable payrate boundary;
// the payment transfer timestamp does not change it. Unapproved or unpaid
// entries remain eligible for recalculation by a later configuration.
func (s *PayrateTemporalService) GetLatestPaidTimesheetDateForProject(ctx context.Context, projectID uint) (*time.Time, error) {
	return s.getLatestPaidTimesheetDate(ctx, s.db, projectID)
}

// CountMutableTimesheetsOnOrAfter reports how many unpaid and unapproved
// timesheets a save effective from the given date would re-price. Used by the
// dry-run endpoint so the user sees the blast radius before confirming.
func (s *PayrateTemporalService) CountMutableTimesheetsOnOrAfter(ctx context.Context, projectID uint, fromDate time.Time) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&domain.Timesheet{}).
		Where("project_id = ? AND date >= ? AND payment_status <> ? AND timesheet_status <> ?",
			projectID, toLocalDateOnly(fromDate), domain.PaymentStatusPaid, domain.TimesheetStatusApproved).
		Count(&count).Error
	if err != nil {
		return 0, s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to count mutable timesheets: %w", err))
	}
	return count, nil
}

// IsLatestPayrate reports whether the given config is still the project's
// most recent one (no live row starts after it). Mirrors ensureRowIsLatestTx
// for read-side checks such as the dry-run endpoint.
func (s *PayrateTemporalService) IsLatestPayrate(ctx context.Context, payrate *domain.Payrate) (bool, *domain.Payrate, error) {
	var successor domain.Payrate
	err := s.db.WithContext(ctx).
		Where("project_id = ? AND id <> ? AND from_date > ?", payrate.ProjectID, payrate.ID, toLocalDateOnly(payrate.FromDate)).
		Order("from_date ASC, id ASC").
		First(&successor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil, nil
		}
		return false, nil, s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check successor payrate: %w", err))
	}
	return false, &successor, nil
}

// GetLatestPayrateByProject returns the project's most recent config, or nil
// when the project has none yet.
func (s *PayrateTemporalService) GetLatestPayrateByProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	var payrate domain.Payrate
	err := s.db.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ?", projectID).
		Order("from_date DESC, id DESC").
		First(&payrate).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to load latest payrate: %w", err))
	}
	if payrate.PayrateJSON != "" {
		payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
	}
	return &payrate, nil
}

// HasSiblingPayrateOnOrAfter reports whether another live config starts on or
// after the given date — the read-side mirror of ensureLatestConfigTx used by
// the dry-run endpoint for update saves.
func (s *PayrateTemporalService) HasSiblingPayrateOnOrAfter(ctx context.Context, projectID, excludeID uint, date time.Time) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&domain.Payrate{}).
		Where("project_id = ? AND id <> ? AND from_date >= ?", projectID, excludeID, toLocalDateOnly(date)).
		Count(&count).Error
	if err != nil {
		return false, s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check sibling payrates: %w", err))
	}
	return count > 0, nil
}

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

// ensureCreateAfterLatestTx enforces the append-only timeline: a new config
// must start strictly after the latest existing one. The structured code lets
// API clients branch on the cause; the context names the blocking config.
func (s *PayrateTemporalService) ensureCreateAfterLatestTx(ctx context.Context, tx *gorm.DB, payrate *domain.Payrate) error {
	var latest domain.Payrate
	err := tx.WithContext(ctx).
		Where("project_id = ? AND from_date >= ?", payrate.ProjectID, toLocalDateOnly(payrate.FromDate)).
		Order("from_date DESC, id DESC").
		First(&latest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check latest payrate: %w", err))
	}

	return domain.NewValidationErrorWithCode(domain.ErrCodePayrateNotLatest,
		fmt.Sprintf(constants.MsgPayrateCreateNotAfterLatestVN, formatDateString(latest.FromDate))).
		WithContext("latest_id", latest.ID).
		WithContext("latest_from_date", formatDateString(latest.FromDate))
}

// ensureRowIsLatestTx rejects edits to a config a newer config has replaced.
// History is view-only; the stored to_date is ignored on purpose — stale
// values from the old stitching model must not resurrect that rule.
func (s *PayrateTemporalService) ensureRowIsLatestTx(ctx context.Context, tx *gorm.DB, existing *domain.Payrate) error {
	var successor domain.Payrate
	err := tx.WithContext(ctx).
		Where("project_id = ? AND id <> ? AND from_date > ?", existing.ProjectID, existing.ID, toLocalDateOnly(existing.FromDate)).
		Order("from_date ASC, id ASC").
		First(&successor).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return s.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check successor payrate: %w", err))
	}

	return domain.NewValidationErrorWithCode(domain.ErrCodePayrateNotLatest,
		fmt.Sprintf(constants.MsgCannotUpdateNonLatestPayrateVN, formatDateString(successor.FromDate))).
		WithContext("successor_id", successor.ID).
		WithContext("successor_from_date", formatDateString(successor.FromDate))
}

// ensureLatestConfigTx keeps the edited row's start date from colliding with
// or passing any sibling config: the timeline stays ordered by from_date and
// date resolution never flips. The row being edited is the project's latest
// config (ensureRowIsLatestTx), so moving its start earlier may not cross the
// predecessor's start; moving later has no sibling beyond it to hit.
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

// ensureEarliestPaidTimesheetCoveredTx blocks start-date moves that would
// strand timesheets already linked to this config: immutable rows reference
// the config they were priced under, so any from_date later than the earliest
// linked work date would silently reassign them to an older config in
// date-based resolution. To change rates from a later date, create a new
// config — the timeline appends instead of splitting.
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

	rateLookup, err := payrate.Payrate.RateLookup()
	if err != nil {
		return domain.NewValidationError("không thể phân tích cấu hình mức lương: " + err.Error())
	}

	updatedCount := 0
	for _, timesheet := range timesheets {
		rate, err := rateLookup.Get(timesheet.PayType)
		if err != nil {
			return domain.NewValidationError(fmt.Sprintf("bảng công %d có loại lương không tồn tại trong cấu hình mới", timesheet.ID))
		}

		amount := domain.ComputeTimesheetAmount(project.IsFlexible, timesheet.HoursWorked, int64(rate))

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

// repriceTimesheetsLinkedToTx re-points and re-prices the mutable timesheets
// linked to a deleted config under its covering config. Same immutability
// rules as recalculateMutableTimesheets: paid or approved rows keep their
// price, and a paytype missing from the covering config aborts the
// transaction rather than silently repricing at zero.
func (s *PayrateTemporalService) repriceTimesheetsLinkedToTx(ctx context.Context, tx *gorm.DB, fromPayrateID uint, covering *domain.Payrate) error {
	var project domain.Project
	if err := tx.WithContext(ctx).Select("id", "is_flexible").First(&project, covering.ProjectID).Error; err != nil {
		return fmt.Errorf("failed to load project for timesheet re-pricing: %w", err)
	}

	var timesheets []*domain.Timesheet
	if err := tx.WithContext(ctx).
		Where("payrate_id = ? AND payment_status <> ? AND timesheet_status <> ?",
			fromPayrateID, domain.PaymentStatusPaid, domain.TimesheetStatusApproved,
		).
		Find(&timesheets).Error; err != nil {
		return fmt.Errorf("failed to load linked timesheets: %w", err)
	}
	if len(timesheets) == 0 {
		return nil
	}

	rateLookup, err := covering.Payrate.RateLookup()
	if err != nil {
		return domain.NewValidationError("không thể phân tích cấu hình mức lương: " + err.Error())
	}

	updatedCount := 0
	for _, timesheet := range timesheets {
		rate, err := rateLookup.Get(timesheet.PayType)
		if err != nil {
			return domain.NewValidationError(fmt.Sprintf("bảng công %d có loại lương không tồn tại trong cấu hình kế trước", timesheet.ID))
		}

		amount := domain.ComputeTimesheetAmount(project.IsFlexible, timesheet.HoursWorked, int64(rate))

		result := tx.WithContext(ctx).
			Model(&domain.Timesheet{}).
			Where("id = ? AND payment_status <> ? AND timesheet_status <> ?",
				timesheet.ID,
				domain.PaymentStatusPaid,
				domain.TimesheetStatusApproved,
			).
			Updates(map[string]any{
				"payrate_id": covering.ID,
				"payrate":    int64(rate),
				"amount":     amount,
			})
		if result.Error != nil {
			return fmt.Errorf("failed to re-price timesheet %d: %w", timesheet.ID, result.Error)
		}
		updatedCount += int(result.RowsAffected)
	}

	observability.GetLogger().Info("re-priced timesheets linked to deleted payrate",
		"project_id", covering.ProjectID,
		"covering_payrate_id", covering.ID,
		"deleted_payrate_id", fromPayrateID,
		"updated_count", updatedCount,
	)
	return nil
}
