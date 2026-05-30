package db

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
)

// TemporalHelper provides utilities for temporal data management patterns
type TemporalHelper struct {
	dbHelper *DatabaseHelper
}

// NewTemporalHelper creates a new temporal helper
func NewTemporalHelper(dbHelper *DatabaseHelper) *TemporalHelper {
	return &TemporalHelper{
		dbHelper: dbHelper,
	}
}

// TemporalRecord interface defines the structure for temporal records
type TemporalRecord interface {
	GetID() uint
	GetFromDate() time.Time
	GetToDate() *time.Time
	SetToDate(*time.Time)
	GetStatus() string
	SetStatus(string)
}

// TemporalConfig configures temporal operations
type TemporalConfig struct {
	ValidateNotInPast  bool
	MaxPastDays        int
	AutoCloseExisting  bool
	ValidateNoGaps     bool
	ValidateNoOverlaps bool
	DateFormat         string
}

// DefaultTemporalConfig returns default temporal configuration
func DefaultTemporalConfig() TemporalConfig {
	return TemporalConfig{
		ValidateNotInPast:  true,
		MaxPastDays:        365, // Allow up to 1 year in the past
		AutoCloseExisting:  true,
		ValidateNoGaps:     true,
		ValidateNoOverlaps: true,
		DateFormat:         "2006-01-02",
	}
}

// CreateEffectiveDatedRecord creates a new temporal record with proper validation and conflict resolution
func (h *TemporalHelper) CreateEffectiveDatedRecord(ctx context.Context, record TemporalRecord, scopeColumns []string, scopeValues []any, config TemporalConfig) error {
	logger := observability.GetLogger()

	// Validate the from date
	if err := h.validateFromDate(record.GetFromDate(), config); err != nil {
		return err
	}

	return h.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Close existing active records if configured
		if config.AutoCloseExisting {
			if err := h.closeActiveRecords(tx, record, scopeColumns, scopeValues, config); err != nil {
				return err
			}
		} else {
			// Validate no conflicts if not auto-closing
			if err := h.validateNoConflicts(tx, record, scopeColumns, scopeValues, config); err != nil {
				return err
			}
		}

		// Set record as active with open-ended to_date
		record.SetToDate(nil)
		record.SetStatus("active")

		// Create the new record
		if err := tx.Create(record).Error; err != nil {
			logger.Error("Failed to create temporal record",
				"record_type", fmt.Sprintf("%T", record),
				"from_date", record.GetFromDate().Format(config.DateFormat),
				"error", err)
			return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to create temporal record: %w", err))
		}

		logger.Info("Temporal record created successfully",
			"record_type", fmt.Sprintf("%T", record),
			"record_id", record.GetID(),
			"from_date", record.GetFromDate().Format(config.DateFormat))

		return nil
	})
}

// UpdateEffectiveDatedRecord handles updates to temporal records with proper validation
func (h *TemporalHelper) UpdateEffectiveDatedRecord(ctx context.Context, record TemporalRecord, existingRecord TemporalRecord, scopeColumns []string, scopeValues []any, config TemporalConfig) error {
	_ = observability.GetLogger() // Logger for potential future use

	// Validate new from date
	if err := h.validateFromDate(record.GetFromDate(), config); err != nil {
		return err
	}

	// Use UTC and date-only comparison to avoid timezone issues
	newFromDate := h.toUTCDateOnly(record.GetFromDate())
	existingFromDate := h.toUTCDateOnly(existingRecord.GetFromDate())

	if newFromDate.Equal(existingFromDate) {
		// Same effective date - perform in-place update if safe
		return h.updateInPlace(ctx, record, config)
	}

	// Different effective date - use temporal approach
	if !newFromDate.After(existingFromDate) {
		return domain.NewValidationError(fmt.Sprintf("New effective date (%s) must be after current effective date (%s)",
			newFromDate.Format(config.DateFormat), existingFromDate.Format(config.DateFormat)))
	}

	return h.createTemporalUpdate(ctx, record, existingRecord, config)
}

// GetActiveRecordOnDate retrieves the active record for a scope on a specific date
func (h *TemporalHelper) GetActiveRecordOnDate(ctx context.Context, dest TemporalRecord, scopeColumns []string, scopeValues []any, date time.Time, config TemporalConfig) error {
	date = h.toUTCDateOnly(date)

	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		query := db

		// Build where clause for scope
		whereClause := ""
		allValues := []any{}

		for i, col := range scopeColumns {
			if i > 0 {
				whereClause += " AND "
			}
			whereClause += col + " = ?"
			allValues = append(allValues, scopeValues[i])
		}

		// Add temporal constraints
		whereClause += " AND from_date <= ? AND (to_date IS NULL OR to_date >= ?)"
		allValues = append(allValues, date, date)

		err := query.Where(whereClause, allValues...).
			Order("from_date DESC").
			First(dest).Error

		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return domain.NewNotFoundError(fmt.Sprintf("no active record found on date %s", date.Format(config.DateFormat)))
			}
			return h.dbHelper.WrapDatabaseError(err)
		}

		return nil
	})
}

// ValidateTemporalIntegrity performs comprehensive validation of temporal data integrity
func (h *TemporalHelper) ValidateTemporalIntegrity(ctx context.Context, model any, scopeColumns []string, scopeValues []any, config TemporalConfig) error {
	return h.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		// Build scope where clause
		whereClause := ""
		for i, col := range scopeColumns {
			if i > 0 {
				whereClause += " AND "
			}
			whereClause += col + " = ?"
		}

		// Check 1: Validate date order constraints
		if err := h.validateDateOrder(db, model, whereClause, scopeValues, config); err != nil {
			return err
		}

		// Check 2: Ensure at most one active record
		if err := h.validateSingleActive(db, model, whereClause, scopeValues, config); err != nil {
			return err
		}

		// Check 3: Validate temporal sequencing
		if config.ValidateNoGaps || config.ValidateNoOverlaps {
			if err := h.validateTemporalSequencing(db, model, whereClause, scopeValues, config); err != nil {
				return err
			}
		}

		return nil
	})
}

// EndActiveRecord manually ends the currently active record
func (h *TemporalHelper) EndActiveRecord(ctx context.Context, record TemporalRecord, endDate time.Time, config TemporalConfig) error {
	logger := observability.GetLogger()
	endDate = h.toUTCDateOnly(endDate)

	return h.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Validate end date is after start date
		if endDate.Before(record.GetFromDate()) || endDate.Equal(record.GetFromDate()) {
			return domain.NewValidationError(fmt.Sprintf("end date (%s) must be after start date (%s)",
				endDate.Format(config.DateFormat), record.GetFromDate().Format(config.DateFormat)))
		}

		record.SetToDate(&endDate)
		record.SetStatus("ended")

		if err := tx.Save(record).Error; err != nil {
			logger.Error("Failed to end temporal record",
				"record_id", record.GetID(),
				"end_date", endDate.Format(config.DateFormat),
				"error", err)
			return h.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Temporal record ended successfully",
			"record_id", record.GetID(),
			"end_date", endDate.Format(config.DateFormat))

		return nil
	})
}

// Private helper methods

func (h *TemporalHelper) validateFromDate(fromDate time.Time, config TemporalConfig) error {
	if !config.ValidateNotInPast {
		return nil
	}

	now := clock.Now().UTC()
	maxPastDate := h.toUTCDateOnly(now.AddDate(0, 0, -config.MaxPastDays))
	dateOnly := h.toUTCDateOnly(fromDate)

	if dateOnly.Before(maxPastDate) {
		return domain.NewValidationError(fmt.Sprintf("cannot create record with from_date more than %d days in the past: %s",
			config.MaxPastDays, dateOnly.Format(config.DateFormat)))
	}

	return nil
}

func (h *TemporalHelper) closeActiveRecords(tx *gorm.DB, record TemporalRecord, scopeColumns []string, scopeValues []any, config TemporalConfig) error {
	logger := observability.GetLogger()

	// Build where clause for active records
	whereClause := ""
	allValues := []any{}

	for i, col := range scopeColumns {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += col + " = ?"
		allValues = append(allValues, scopeValues[i])
	}
	whereClause += " AND to_date IS NULL AND status = ?"
	allValues = append(allValues, "active")

	// Calculate to_date as day before new from_date
	toDate := record.GetFromDate().AddDate(0, 0, -1)

	// Validate that new from_date is after any existing from_date
	var maxFromDate time.Time
	err := tx.Model(record).
		Where(whereClause, allValues...).
		Select("COALESCE(MAX(from_date), '1970-01-01') as max_from_date").
		Scan(&maxFromDate).Error

	if err != nil {
		return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to check existing records: %w", err))
	}

	if !record.GetFromDate().After(maxFromDate) && !maxFromDate.Equal(time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return domain.NewValidationError(fmt.Sprintf("new from_date (%s) must be after existing active record from_date (%s)",
			record.GetFromDate().Format(config.DateFormat), maxFromDate.Format(config.DateFormat)))
	}

	// Close active records
	updates := map[string]any{
		"to_date": toDate,
		"status":  "ended",
	}

	result := tx.Model(record).Where(whereClause, allValues...).Updates(updates)
	if result.Error != nil {
		logger.Error("Failed to close active records", "error", result.Error)
		return h.dbHelper.WrapDatabaseError(result.Error)
	}

	if result.RowsAffected > 0 {
		logger.Info("Closed active records",
			"count", result.RowsAffected,
			"to_date", toDate.Format(config.DateFormat))
	}

	return nil
}

func (h *TemporalHelper) validateNoConflicts(tx *gorm.DB, record TemporalRecord, scopeColumns []string, scopeValues []any, config TemporalConfig) error {
	// Build conflict detection query
	whereClause := ""
	allValues := []any{}

	for i, col := range scopeColumns {
		if i > 0 {
			whereClause += " AND "
		}
		whereClause += col + " = ?"
		allValues = append(allValues, scopeValues[i])
	}

	// Check for overlapping periods
	whereClause += " AND ((from_date <= ? AND (to_date IS NULL OR to_date >= ?)) OR (from_date <= ? AND to_date >= ?))"
	fromDate := record.GetFromDate()
	toDate := record.GetToDate()

	if toDate != nil {
		allValues = append(allValues, fromDate, fromDate, *toDate, *toDate)
	} else {
		// Open-ended record - check for any active records
		allValues = append(allValues, fromDate, fromDate, fromDate, fromDate)
	}

	var count int64
	err := tx.Model(record).Where(whereClause, allValues...).Count(&count).Error
	if err != nil {
		return h.dbHelper.WrapDatabaseError(err)
	}

	if count > 0 {
		return domain.NewValidationError(fmt.Sprintf("conflicting temporal record exists for the same period starting %s", fromDate.Format(config.DateFormat)))
	}

	return nil
}

func (h *TemporalHelper) updateInPlace(ctx context.Context, record TemporalRecord, config TemporalConfig) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		if err := tx.Save(record).Error; err != nil {
			logger.Error("Failed to save record in-place", "record_id", record.GetID(), "error", err)
			return h.dbHelper.WrapDatabaseError(err)
		}

		logger.Info("Record updated in-place successfully", "record_id", record.GetID())
		return nil
	})
}

func (h *TemporalHelper) createTemporalUpdate(ctx context.Context, newRecord TemporalRecord, existingRecord TemporalRecord, config TemporalConfig) error {
	logger := observability.GetLogger()

	return h.dbHelper.ExecuteInTransactionWithRetry(ctx, func(tx *gorm.DB) error {
		// Close the existing record
		toDate := newRecord.GetFromDate().AddDate(0, 0, -1)
		existingRecord.SetToDate(&toDate)
		existingRecord.SetStatus("ended")

		if err := tx.Save(existingRecord).Error; err != nil {
			logger.Error("Failed to close existing record", "record_id", existingRecord.GetID(), "error", err)
			return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to close existing record: %w", err))
		}

		// Create the new record
		originalID := newRecord.GetID()
		// Clear ID to ensure a new record is created (this would need to be implemented per type)
		newRecord.SetToDate(nil) // Open-ended
		newRecord.SetStatus("active")

		if err := tx.Create(newRecord).Error; err != nil {
			logger.Error("Failed to create new record", "original_id", originalID, "error", err)
			return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to create new record: %w", err))
		}

		logger.Info("Temporal update completed successfully",
			"old_record_id", existingRecord.GetID(),
			"new_record_id", newRecord.GetID())

		return nil
	})
}

func (h *TemporalHelper) validateDateOrder(db *gorm.DB, model any, whereClause string, scopeValues []any, config TemporalConfig) error {
	var invalidCount int64
	err := db.Raw(fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE %s AND to_date IS NOT NULL AND from_date >= to_date
	`, db.Statement.Table, whereClause), scopeValues...).Scan(&invalidCount).Error

	if err != nil {
		return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to validate date constraints: %w", err))
	}

	if invalidCount > 0 {
		return domain.NewValidationError(fmt.Sprintf("found %d records with invalid date ordering", invalidCount))
	}

	return nil
}

func (h *TemporalHelper) validateSingleActive(db *gorm.DB, model any, whereClause string, scopeValues []any, config TemporalConfig) error {
	activeWhereClause := whereClause + " AND to_date IS NULL AND status = 'active'"

	var activeCount int64
	err := db.Model(model).Where(activeWhereClause, append(scopeValues, "active")...).Count(&activeCount).Error
	if err != nil {
		return h.dbHelper.WrapDatabaseError(fmt.Errorf("failed to count active records: %w", err))
	}

	if activeCount > 1 {
		return domain.NewValidationError(fmt.Sprintf("multiple active records found: %d", activeCount))
	}

	return nil
}

func (h *TemporalHelper) validateTemporalSequencing(db *gorm.DB, model any, whereClause string, scopeValues []any, config TemporalConfig) error {
	// This would need to be implemented with database-specific queries
	// For now, return nil as a placeholder
	return nil
}

func (h *TemporalHelper) toUTCDateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
