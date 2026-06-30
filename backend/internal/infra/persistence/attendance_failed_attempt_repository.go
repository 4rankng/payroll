package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type attendanceFailedAttemptRepository struct {
	db *gorm.DB
}

// NewAttendanceFailedAttemptRepository creates a new failed-attempt repository.
func NewAttendanceFailedAttemptRepository(db *gorm.DB) domain.AttendanceFailedAttemptRepository {
	return &attendanceFailedAttemptRepository{db: db}
}

func (r *attendanceFailedAttemptRepository) Create(ctx context.Context, attempt *domain.AttendanceFailedAttempt) error {
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *attendanceFailedAttemptRepository) List(ctx context.Context, filters domain.FailedAttemptFilters) ([]*domain.AttendanceFailedAttempt, error) {
	var attempts []*domain.AttendanceFailedAttempt
	query := r.db.WithContext(ctx).
		Model(&domain.AttendanceFailedAttempt{}).
		Preload("Employee")

	if filters.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filters.EmployeeID)
	}
	if filters.AttemptType != nil {
		query = query.Where("attempt_type = ?", *filters.AttemptType)
	}
	if filters.ReasonCategory != nil {
		query = query.Where("reason_category = ?", *filters.ReasonCategory)
	}
	if filters.FromDate != nil {
		query = query.Where("created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("created_at < ?", *filters.ToDate)
	}

	query = query.Order("created_at DESC")

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&attempts).Error
	return attempts, err
}

func (r *attendanceFailedAttemptRepository) Count(ctx context.Context, filters domain.FailedAttemptFilters) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&domain.AttendanceFailedAttempt{})

	if filters.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filters.EmployeeID)
	}
	if filters.AttemptType != nil {
		query = query.Where("attempt_type = ?", *filters.AttemptType)
	}
	if filters.ReasonCategory != nil {
		query = query.Where("reason_category = ?", *filters.ReasonCategory)
	}
	if filters.FromDate != nil {
		query = query.Where("created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("created_at < ?", *filters.ToDate)
	}

	err := query.Count(&count).Error
	return count, err
}

func (r *attendanceFailedAttemptRepository) GetCategoryCounts(ctx context.Context, since, until time.Time) ([]domain.FailedAttemptCategoryCount, error) {
	var counts []domain.FailedAttemptCategoryCount
	err := r.db.WithContext(ctx).
		Model(&domain.AttendanceFailedAttempt{}).
		Select("reason_category as category, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", since, until).
		Where(checkInEnabledScope()).
		Group("reason_category").
		Order("count DESC").
		Scan(&counts).Error
	return counts, err
}

func (r *attendanceFailedAttemptRepository) GetTotalCount(ctx context.Context, since, until time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&domain.AttendanceFailedAttempt{}).
		Where("created_at >= ? AND created_at < ?", since, until).
		Where(checkInEnabledScope()).
		Count(&count).Error
	return count, err
}

func (r *attendanceFailedAttemptRepository) GetCountByAttemptType(ctx context.Context, since, until time.Time) ([]domain.FailedAttemptTypeCount, error) {
	var counts []domain.FailedAttemptTypeCount
	err := r.db.WithContext(ctx).
		Model(&domain.AttendanceFailedAttempt{}).
		Select("attempt_type, COUNT(*) as count").
		Where("created_at >= ? AND created_at < ?", since, until).
		Where(checkInEnabledScope()).
		Group("attempt_type").
		Scan(&counts).Error
	return counts, err
}

// checkInEnabledScope returns a GORM WHERE clause that restricts results to
// employees with at least one active, check-in-enabled project assignment.
// When project_id = 0 (e.g. check-out failures where the handler cannot resolve
// the project), the subquery matches on employee_id alone so those rows are not
// silently excluded.
func checkInEnabledScope() string {
	return `EXISTS (SELECT 1 FROM project_employees pe WHERE pe.employee_id = attendance_failed_attempts.employee_id AND (attendance_failed_attempts.project_id = 0 OR (pe.project_id = attendance_failed_attempts.project_id AND pe.deleted_at IS NULL AND pe.last_date IS NULL)) AND pe.check_in_enabled = 1)`
}
