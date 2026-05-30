package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"

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

	query = query.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

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
		Where(`
			project_id = ? 
			AND from_date <= ? 
			AND (to_date IS NULL OR to_date >= ?)
		`, projectID, date, date).
		Order("from_date DESC").
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

// GetByProjectAndToDate finds a payrate for the project with a specific end date.
func (r *PayrateRepository) GetByProjectAndToDate(ctx context.Context, projectID uint, toDate time.Time) (*domain.Payrate, error) {
	var payrate domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where("project_id = ? AND to_date = ? AND deleted_at IS NULL", projectID, toDate).
		Order("from_date DESC").
		First(&payrate).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("payrate not found")
		}
		return nil, err
	}

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

	if filters.ToDate != nil {
		query = query.Where("to_date <= ?", *filters.ToDate)
	}

	return query
}

// IsUsedByTimesheets checks if a payrate is referenced by any timesheets
func (r *PayrateRepository) IsUsedByTimesheets(ctx context.Context, id uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payrate_id = ?", id).
		Count(&count).Error

	return count > 0, err
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

// GetActiveByProject gets active payrates for a project at a specific date
func (r *PayrateRepository) GetActiveByProject(ctx context.Context, projectID uint, date time.Time) ([]*domain.Payrate, error) {
	var payrates []*domain.Payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where(`
			project_id = ? 
			AND from_date <= ? 
			AND (to_date IS NULL OR to_date >= ?)
		`, projectID, date, date).
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

// EndActivePayrates sets to_date for active payrates to avoid overlaps
func (r *PayrateRepository) EndActivePayrates(ctx context.Context, projectID uint, endDate time.Time, excludeID uint) error {
	query := r.DB.WithContext(ctx).
		Model(&domain.Payrate{}).
		Where("project_id = ? AND to_date IS NULL", projectID)

	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}

	return query.Update("to_date", endDate).Error
}

// GetCurrentOrUpcomingByProject gets the currently active payrate or the nearest upcoming one for a project
func (r *PayrateRepository) GetCurrentOrUpcomingByProject(ctx context.Context, projectID uint) (*domain.Payrate, error) {
	now := clock.Now()
	var payrate domain.Payrate

	// First try to get the current active payrate
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("CreatedUser").
		Where(`
			project_id = ?
			AND from_date <= ?
			AND (to_date IS NULL OR to_date >= ?)
		`, projectID, now, now).
		Order("from_date DESC").
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

// FindActiveByProject retrieves all active payrates for a project
func (r *PayrateRepository) FindActiveByProject(ctx context.Context, tx interface{}, projectID uint) ([]*domain.Payrate, error) {
	db := r.getDB(tx)
	var payrates []*domain.Payrate

	err := db.WithContext(ctx).
		Where("project_id = ? AND to_date IS NULL", projectID).
		Order("from_date ASC").
		Find(&payrates).Error

	if err != nil {
		return nil, err
	}

	// Manually unmarshal payrate configurations
	for _, payrate := range payrates {
		if payrate.PayrateJSON != "" {
			payrate.Payrate = domain.PayrateConfiguration(payrate.PayrateJSON)
		}
	}

	return payrates, nil
}

// CloseActiveByProject sets end date for all active payrates of a project
func (r *PayrateRepository) CloseActiveByProject(ctx context.Context, tx interface{}, projectID uint, toDate time.Time) error {
	db := r.getDB(tx)
	return db.WithContext(ctx).
		Model(&domain.Payrate{}).
		Where("project_id = ? AND to_date IS NULL", projectID).
		Update("to_date", toDate).Error
}

// DeleteActiveByProject deletes all active payrates for a project
func (r *PayrateRepository) DeleteActiveByProject(ctx context.Context, tx interface{}, projectID uint) error {
	db := r.getDB(tx)
	return db.WithContext(ctx).
		Where("project_id = ? AND to_date IS NULL", projectID).
		Delete(&domain.Payrate{}).Error
}

// ValidateDateOrdering counts payrates with invalid date ordering
func (r *PayrateRepository) ValidateDateOrdering(ctx context.Context, tx interface{}, projectID uint) (int64, error) {
	db := r.getDB(tx)
	var count int64

	err := db.WithContext(ctx).
		Model(&domain.Payrate{}).
		Where("project_id = ? AND to_date IS NOT NULL AND from_date >= to_date", projectID).
		Count(&count).Error

	return count, err
}

// ValidateActiveCount counts active payrates for a project
func (r *PayrateRepository) ValidateActiveCount(ctx context.Context, tx interface{}, projectID uint) (int64, error) {
	db := r.getDB(tx)
	var count int64

	err := db.WithContext(ctx).
		Model(&domain.Payrate{}).
		Where("project_id = ? AND to_date IS NULL", projectID).
		Count(&count).Error

	return count, err
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
