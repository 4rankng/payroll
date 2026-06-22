package persistence

import (
	"context"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/clock"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type attendanceRepository struct {
	db *gorm.DB
}

func NewAttendanceRepository(db *gorm.DB) domain.AttendanceRepository {
	return &attendanceRepository{db: db}
}

// getDB returns the transaction-aware DB session. If a transaction is active in
// the context (set via domain.WithTransactionContext), it uses that TX; otherwise
// it falls back to the repository's own DB connection.
func (r *attendanceRepository) getDB(ctx context.Context) *gorm.DB {
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.db.WithContext(ctx)
}

func (r *attendanceRepository) Create(ctx context.Context, attendance *domain.Attendance) error {
	return r.getDB(ctx).Create(attendance).Error
}

func (r *attendanceRepository) Update(ctx context.Context, attendance *domain.Attendance) error {
	return r.getDB(ctx).Save(attendance).Error
}

// MarkAutoRejected atomically rejects an open, unrejected attendance. The
// conditional WHERE (check_out_time IS NULL AND salary_reject_reason IS NULL)
// is the race guard: a concurrent CheckOut that set check_out_time, or a prior
// rejection, makes RowsAffected=0 — a safe no-op. This avoids the lost-update
// hazard of a full-row Save over a stale no-checkout snapshot.
func (r *attendanceRepository) MarkAutoRejected(ctx context.Context, id uint, reason string) (bool, error) {
	res := r.getDB(ctx).Model(&domain.Attendance{}).
		Where("id = ? AND check_out_time IS NULL AND salary_reject_reason IS NULL", id).
		Updates(map[string]interface{}{
			"earning_amount":       0,
			"salary_reject_reason": reason,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *attendanceRepository) GetByEmployeeAndDate(ctx context.Context, employeeID uint, date time.Time) (*domain.Attendance, error) {
	var att domain.Attendance
	dateOnly := date.Format("2006-01-02")
	err := r.getDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("employee_id = ? AND date = ?", employeeID, dateOnly).
		First(&att).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &att, nil
}

func (r *attendanceRepository) GetOrphanCandidates(ctx context.Context, after, before time.Time) ([]*domain.Attendance, error) {
	var attendances []*domain.Attendance
	// Open (no checkout) AND unrejected records checked in within [after, before).
	// salary_reject_reason IS NULL excludes already-auto-rejected records so the
	// fallback sweep can't double-process a finalized rejection.
	err := r.getDB(ctx).
		Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time >= ? AND check_in_time < ?", after, before).
		Find(&attendances).Error
	return attendances, err
}

func (r *attendanceRepository) GetByID(ctx context.Context, id uint) (*domain.Attendance, error) {
	var att domain.Attendance
	err := r.getDB(ctx).
		Preload("Employee").
		Preload("Project").
		First(&att, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &att, nil
}

func (r *attendanceRepository) List(ctx context.Context, filters domain.AttendanceFilters) ([]*domain.Attendance, error) {
	var attendances []*domain.Attendance
	query := r.buildFilterQuery(ctx, filters).
		Preload("Employee").
		Preload("Project")

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if filters.SortBy != "" {
		order := "ASC"
		if filters.SortOrder == "desc" || filters.SortOrder == "DESC" {
			order = "DESC"
		}
		query = query.Order(common.SanitizeSortColumn(filters.SortBy, "created_at") + " " + common.SanitizeSortOrder(order, "DESC"))
	} else {
		query = query.Order("created_at DESC")
	}

	err := query.Find(&attendances).Error
	return attendances, err
}

func (r *attendanceRepository) Count(ctx context.Context, filters domain.AttendanceFilters) (int64, error) {
	var count int64
	err := r.buildFilterQuery(ctx, filters).Count(&count).Error
	return count, err
}

func (r *attendanceRepository) buildFilterQuery(ctx context.Context, filters domain.AttendanceFilters) *gorm.DB {
	query := r.getDB(ctx).Model(&domain.Attendance{})
	if filters.EmployeeID != nil {
		query = query.Where("employee_id = ?", *filters.EmployeeID)
	}
	if filters.ProjectID != nil {
		query = query.Where("project_id = ?", *filters.ProjectID)
	}
	if filters.FromDate != nil {
		query = query.Where("date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("date <= ?", *filters.ToDate)
	}
	if filters.Status != nil {
		switch *filters.Status {
		case domain.AttendanceStatusCompleted:
			query = query.Where("check_out_time IS NOT NULL")
		case domain.AttendanceStatusRejected:
			query = query.Where("check_out_time IS NULL AND salary_reject_reason IS NOT NULL")
		case domain.AttendanceStatusCheckedIn:
			query = query.Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time >= ?", clock.Now().Add(-18*time.Hour))
		case domain.AttendanceStatusOrphaned:
			query = query.Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time < ?", clock.Now().Add(-18*time.Hour))
		}
	}
	return query
}
