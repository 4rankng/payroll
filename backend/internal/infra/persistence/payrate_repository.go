package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type PayrateRepository struct {
	*BaseRepository
}

func NewPayrateRepository(db *Database) domain.PayrateRepository {
	return &PayrateRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *PayrateRepository) Create(ctx context.Context, payrate *domain.Payrate) error {
	return r.SafeCreate(ctx, payrate)
}

func (r *PayrateRepository) GetByID(ctx context.Context, id uint) (*domain.Payrate, error) {
	var payrate domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		First(&payrate, id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("payrate not found")
		}
		return nil, err
	}

	// Manually unmarshal payrate configuration since AfterFind hook may not be reliable
	if payrate.PayrateJSON != "" {
		payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
	}

	return &payrate, nil
}

func (r *PayrateRepository) Update(ctx context.Context, payrate *domain.Payrate) error {
	return r.SafeUpdate(ctx, payrate)
}

func (r *PayrateRepository) Delete(ctx context.Context, id uint) error {
	return r.SafeDelete(ctx, &domain.Payrate{}, id)
}

func (r *PayrateRepository) List(ctx context.Context, filters domain.PayrateFilters) ([]*domain.Payrate, error) {
	var payrates []*domain.Payrate
	query := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser")

	query = r.applyFilters(query, filters)

	// Apply sorting
	sortBy := "created_at"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}

	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "created_at"), common.SanitizeSortOrder(sortOrder, "DESC")))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&payrates).Error
	if err != nil {
		return nil, err
	}

	// Manually unmarshal payrate configurations since AfterFind hook may not be reliable
	for _, payrate := range payrates {
		if payrate.PayrateJSON != "" {
			payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
		}
	}

	return payrates, nil
}

func (r *PayrateRepository) Count(ctx context.Context, filters domain.PayrateFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.Payrate{})
	query = r.applyFilters(query, filters)
	err := query.Count(&count).Error
	return count, err
}

func (r *PayrateRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.Payrate, error) {
	var payrates []*domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ?", projectID).
		Order("from_date DESC").
		Find(&payrates).Error
	if err != nil {
		return nil, err
	}

	// Manually unmarshal payrate configurations since AfterFind hook may not be reliable
	for _, payrate := range payrates {
		if payrate.PayrateJSON != "" {
			payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
		}
	}

	return payrates, nil
}

func (r *PayrateRepository) GetActiveByProjectAndDate(ctx context.Context, projectID uint, date time.Time) (*domain.Payrate, error) {
	var payrate domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		// From-date-only resolution: the config in force is the latest
		// from_date on or before the date; the id tiebreaker keeps same-date
		// ties deterministic.
		Where("project_id = ? AND from_date <= ?", projectID, date).
		Order("from_date DESC, id DESC").
		First(&payrate).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("no active payrate found for this project and date")
		}
		return nil, err
	}

	// Manually unmarshal payrate configuration since AfterFind hook may not be reliable
	if payrate.PayrateJSON != "" {
		payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
	}

	return &payrate, nil
}

func (r *PayrateRepository) applyFilters(query *gorm.DB, filters domain.PayrateFilters) *gorm.DB {
	if filters.ProjectID != nil {
		query = query.Where("project_id = ?", *filters.ProjectID)
	}

	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}

	if filters.FromDate != nil {
		query = query.Where("from_date >= ?", *filters.FromDate)
	}

	// No ToDate filter: payrates.to_date was dropped by migration 138; end
	// dates are derived per row from each config's successor.

	return query
}

// HasTimesheetsFromDate checks if a payrate has any timesheets on or after the specified date
func (r *PayrateRepository) HasTimesheetsFromDate(ctx context.Context, id uint, fromDate time.Time) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payrate_id = ? AND date >= ?", id, fromDate).
		Count(&count).Error

	return count > 0, err
}

// HasProjectTimesheetsFromDate checks if a project has any timesheets on or after the specified date
func (r *PayrateRepository) HasProjectTimesheetsFromDate(ctx context.Context, projectID uint, fromDate time.Time) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND date >= ?", projectID, fromDate).
		Count(&count).Error

	return count > 0, err
}

// GetLatestTimesheetDateForPayrate returns the most recent timesheet date linked to a payrate.
// Returns nil if no timesheets exist.
func (r *PayrateRepository) GetLatestTimesheetDateForPayrate(ctx context.Context, payrateID uint) (*time.Time, error) {
	var result struct {
		MaxDate *time.Time
	}
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("MAX(date) as max_date").
		Where("payrate_id = ?", payrateID).
		Scan(&result).Error
	if err != nil {
		return nil, err
	}
	return result.MaxDate, nil
}

// GetActiveByProject gets the payrates in force for a project at a specific
// date. From-date-only resolution keeps at most the latest qualifying config;
// the slice return preserves the historical call shape.
func (r *PayrateRepository) GetActiveByProject(ctx context.Context, projectID uint, date time.Time) ([]*domain.Payrate, error) {
	var payrates []*domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ? AND from_date <= ?", projectID, date).
		Order("from_date DESC, id DESC").
		Limit(1).
		Find(&payrates).Error
	if err != nil {
		return nil, err
	}

	// Manually unmarshal payrate configurations since AfterFind hook may not be reliable
	for _, payrate := range payrates {
		if payrate.PayrateJSON != "" {
			payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
		}
	}

	return payrates, nil
}

// GetCurrentOrUpcomingByProject gets the currently active payrate or the nearest upcoming one for a project
func (r *PayrateRepository) GetCurrentOrUpcomingByProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	now := clock.Now()
	var payrate domain.Payrate

	// First try to get the current active payrate: latest from_date on or
	// before now (from-date-only resolution; the id tiebreaker keeps
	// same-date ties deterministic).
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ? AND from_date <= ?", projectID, now).
		Order("from_date DESC, id DESC").
		First(&payrate).Error

	if err == nil {
		// Found active payrate
		if payrate.PayrateJSON != "" {
			payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
		}
		return &payrate, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// No active payrate found, look for the nearest upcoming one
	err = r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ? AND from_date > ?", projectID, now).
		Order("from_date ASC").
		First(&payrate).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("no active or upcoming payrate found for this project")
		}
		return nil, err
	}

	// Manually unmarshal payrate configuration since AfterFind hook may not be reliable
	if payrate.PayrateJSON != "" {
		payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
	}

	return &payrate, nil
}

// Transaction-aware methods for temporal operations

// CreateWithTx creates a payrate within a transaction
func (r *PayrateRepository) CreateWithTx(ctx context.Context, tx interface{}, payrate *domain.Payrate) error {
	db := r.getDB(tx)
	return db.WithContext(ctx).Create(payrate).Error
}

// getDB returns transaction or regular DB
func (r *PayrateRepository) getDB(tx interface{}) *gorm.DB {
	if tx != nil {
		if gormTx, ok := tx.(*gorm.DB); ok {
			return gormTx
		}
	}
	return r.DB
}
