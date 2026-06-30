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
		Order("CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL THEN 0 ELSE 1 END").
		Order("id DESC").
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
	if filters.ZeroEarning != nil && *filters.ZeroEarning {
		query = query.Where("check_out_time IS NOT NULL AND (earning_amount IS NULL OR earning_amount = 0)")
	}
	return query
}

// GetHealthStats returns conditional-aggregation counts for the admin health dashboard.
// It uses a single query with SUM(CASE WHEN …) to compute all five metrics. The
// since/until window bounds check_in_time; the open/orphaned split additionally uses
// an 18h cutoff against the current time (mirroring AttendanceStatusCheckedIn/Orphaned
// in buildFilterQuery). The query reuses the exact predicates from buildFilterQuery so
// the dashboard counts agree with the filtered list a drill-down opens.
func (r *attendanceRepository) GetHealthStats(ctx context.Context, since, until time.Time) (*domain.AttendanceHealthStats, error) {
	var stats domain.AttendanceHealthStats
	openCutoff := clock.Now().Add(-18 * time.Hour)

	err := r.getDB(ctx).
		Table("attendances").
		Select(`
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time >= ? THEN 1 ELSE 0 END) as open_checked_in,
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL AND check_in_time < ? THEN 1 ELSE 0 END) as orphaned,
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NOT NULL THEN 1 ELSE 0 END) as auto_rejected,
			SUM(CASE WHEN check_out_time IS NOT NULL AND (earning_amount IS NULL OR earning_amount = 0) THEN 1 ELSE 0 END) as completed_zero_earning,
			SUM(CASE WHEN check_out_time IS NOT NULL AND earning_amount > 0 THEN 1 ELSE 0 END) as successful_checkouts
		`, openCutoff, openCutoff).
		Where("check_in_time >= ? AND check_in_time < ?", since, until).
		Where(checkInEnabledScope("attendances")).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
