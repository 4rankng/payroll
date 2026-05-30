package services

import (
	"context"
	"fmt"
	"time"

	"api-server/internal/pkg/clock"
	"gorm.io/gorm"
)

// TemporalRecord defines the interface for entities with date ranges
type TemporalRecord interface {
	GetID() uint
	GetStartDate() time.Time
	GetEndDate() *time.Time
	SetEndDate(date *time.Time)
	IsActive() bool
}

// TemporalRepository defines the interface for database operations on temporal records
type TemporalRepository interface {
	FindActiveRecord(ctx context.Context, entityKeys map[string]any) (TemporalRecord, error)
	CreateRecord(ctx context.Context, record TemporalRecord) error
	UpdateRecord(ctx context.Context, record TemporalRecord) error
}

// EntityConfig defines configuration for a specific temporal entity
type EntityConfig struct {
	TableName       string
	EntityKeyFields []string // Fields that define the entity scope (e.g., ["project_id"], ["employee_id", "project_id"])
	StartDateField  string   // Field name for start date (e.g., "from_date", "start_date")
	EndDateField    string   // Field name for end date (e.g., "to_date", "last_date")
}

// CreateTemporalRequest represents a request to create a temporal record
type CreateTemporalRequest struct {
	EntityKeys map[string]any // Key-value pairs defining the entity scope
	StartDate  time.Time      // Start date for the new record
	Record     TemporalRecord // The record to create
}

// TemporalService provides shared algorithms for managing effective-dated records
type TemporalService struct {
	db     *gorm.DB
	config EntityConfig
}

// NewTemporalService creates a new temporal service instance
func NewTemporalService(db *gorm.DB, config EntityConfig) *TemporalService {
	return &TemporalService{
		db:     db,
		config: config,
	}
}

// CreateEffectiveDatedRecord implements the core algorithm for creating temporal records
// with automatic closure of previous active records. Requires a TemporalRepository.
func (s *TemporalService) CreateEffectiveDatedRecord(ctx context.Context, repo TemporalRepository, req CreateTemporalRequest) error {
	// Use UTC date-only comparison to avoid timezone issues
	now := clock.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	startDate := time.Date(req.StartDate.Year(), req.StartDate.Month(), req.StartDate.Day(), 0, 0, 0, 0, time.UTC)

	// Validate: cannot add record starting in the past
	if startDate.Before(today) {
		return fmt.Errorf("cannot add record starting in the past: start_date=%s, today=%s",
			startDate.Format("2006-01-02"), today.Format("2006-01-02"))
	}

	// Use transaction to ensure atomicity
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Find current active/open record
		activeRecord, err := repo.FindActiveRecord(ctx, req.EntityKeys)
		if err != nil && err != gorm.ErrRecordNotFound {
			return fmt.Errorf("failed to find active record: %w", err)
		}

		if activeRecord != nil {
			// Validate: new start date must be after existing active record's start date
			if activeRecord.GetStartDate().After(startDate) || activeRecord.GetStartDate().Equal(startDate) {
				return fmt.Errorf("new start date (%s) overlaps or is before existing active record start date (%s)",
					startDate.Format("2006-01-02"), activeRecord.GetStartDate().Format("2006-01-02"))
			}

			// Close previous record one day before new start date
			endDate := startDate.AddDate(0, 0, -1)
			activeRecord.SetEndDate(&endDate)

			if err := repo.UpdateRecord(ctx, activeRecord); err != nil {
				return fmt.Errorf("failed to close previous active record: %w", err)
			}
		}

		// Insert new record with open-ended end date (NULL)
		req.Record.SetEndDate(nil)
		if err := repo.CreateRecord(ctx, req.Record); err != nil {
			return fmt.Errorf("failed to create new temporal record: %w", err)
		}

		return nil
	})
}

// BuildActiveRecordQuery returns a query builder for finding active records
// Entity-specific services should use this to build their queries
func (s *TemporalService) BuildActiveRecordQuery(db *gorm.DB, entityKeys map[string]any) *gorm.DB {
	query := db.Table(s.config.TableName)

	// Apply entity key filters
	for field, value := range entityKeys {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	// Find record with NULL end date (active record)
	query = query.Where(fmt.Sprintf("%s IS NULL", s.config.EndDateField))
	return query
}

// BuildDateRangeQuery returns a query builder for finding records active on a specific date
func (s *TemporalService) BuildDateRangeQuery(db *gorm.DB, entityKeys map[string]any, date time.Time) *gorm.DB {
	date = date.Truncate(24 * time.Hour)
	query := db.Table(s.config.TableName)

	// Apply entity key filters
	for field, value := range entityKeys {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	// Apply date range filters
	query = query.Where(fmt.Sprintf("%s <= ?", s.config.StartDateField), date)
	query = query.Where(fmt.Sprintf("(%s IS NULL OR %s >= ?)", s.config.EndDateField, s.config.EndDateField), date)
	return query.Order(fmt.Sprintf("%s DESC", s.config.StartDateField))
}

// EndActiveRecord manually ends the currently active record using the provided repository
func (s *TemporalService) EndActiveRecord(ctx context.Context, repo TemporalRepository, entityKeys map[string]any, endDate time.Time) error {
	endDate = endDate.Truncate(24 * time.Hour)

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		activeRecord, err := repo.FindActiveRecord(ctx, entityKeys)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("no active record found to end")
			}
			return fmt.Errorf("failed to find active record: %w", err)
		}

		// Validate end date is after start date
		if endDate.Before(activeRecord.GetStartDate()) || endDate.Equal(activeRecord.GetStartDate()) {
			return fmt.Errorf("end date (%s) must be after start date (%s)",
				endDate.Format("2006-01-02"), activeRecord.GetStartDate().Format("2006-01-02"))
		}

		activeRecord.SetEndDate(&endDate)
		return repo.UpdateRecord(ctx, activeRecord)
	})
}

// ValidateTemporalIntegrity performs comprehensive validation of temporal data integrity
func (s *TemporalService) ValidateTemporalIntegrity(ctx context.Context, entityKeys map[string]any) error {
	type temporalRecord struct {
		ID        uint `gorm:"column:id"`
		StartDate time.Time
		EndDate   *time.Time
	}

	var records []temporalRecord

	query := s.db.WithContext(ctx).Table(s.config.TableName)

	// Apply entity key filters
	for field, value := range entityKeys {
		query = query.Where(fmt.Sprintf("%s = ?", field), value)
	}

	// Order by start date for temporal validation
	query = query.Select(fmt.Sprintf("id, %s as start_date, %s as end_date",
		s.config.StartDateField, s.config.EndDateField))
	query = query.Order(fmt.Sprintf("%s ASC", s.config.StartDateField))

	if err := query.Find(&records).Error; err != nil {
		return fmt.Errorf("failed to query temporal records: %w", err)
	}

	// Validate temporal constraints
	var activeCount int
	for i, record := range records {
		// Check for NULL end_date (active records)
		if record.EndDate == nil {
			activeCount++
			if activeCount > 1 {
				return fmt.Errorf("multiple active records found (IDs: %d and previous), only one active record allowed", record.ID)
			}
		}

		// Check date order within each record
		if record.EndDate != nil && !record.StartDate.Before(*record.EndDate) {
			return fmt.Errorf("invalid date range in record %d: start_date (%s) must be before end_date (%s)",
				record.ID, record.StartDate.Format("2006-01-02"), record.EndDate.Format("2006-01-02"))
		}

		// Check for overlapping ranges with next record
		if i < len(records)-1 {
			nextRecord := records[i+1]

			// If current record has no end date, next record's start must be in future
			if record.EndDate == nil {
				if !nextRecord.StartDate.After(record.StartDate) {
					return fmt.Errorf("overlapping records: record %d (starts %s, open-ended) overlaps with record %d (starts %s)",
						record.ID, record.StartDate.Format("2006-01-02"),
						nextRecord.ID, nextRecord.StartDate.Format("2006-01-02"))
				}
			} else {
				// Check for gaps or overlaps
				if nextRecord.StartDate.Before(*record.EndDate) {
					return fmt.Errorf("overlapping date ranges: record %d (ends %s) overlaps with record %d (starts %s)",
						record.ID, record.EndDate.Format("2006-01-02"),
						nextRecord.ID, nextRecord.StartDate.Format("2006-01-02"))
				}

				// Ensure next record starts after current record ends
				if !nextRecord.StartDate.After(*record.EndDate) {
					return fmt.Errorf("invalid temporal sequence: record %d (ends %s) should end before record %d (starts %s)",
						record.ID, record.EndDate.Format("2006-01-02"),
						nextRecord.ID, nextRecord.StartDate.Format("2006-01-02"))
				}
			}
		}
	}

	return nil
}
