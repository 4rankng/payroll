package persistence

import (
	"context"
	"errors"
	"strings"
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
	if err := r.getDB(ctx).Create(attendance).Error; err != nil {
		// A duplicate-key here means a concurrent check-in inserted the OPEN
		// attendance for this employee/day first and the partial unique
		// uq_attendances_employee_open (migration 081) rejected this one. Map it
		// to the same friendly message the sequential dup-check in CheckIn returns.
		if isAttendanceDuplicateKeyErr(err) {
			return domain.NewValidationError("Bạn đã vào làm trong ngày hôm nay rồi")
		}
		return err
	}
	return nil
}

// isAttendanceDuplicateKeyErr reports whether err is a MySQL duplicate-key
// violation, regardless of whether GORM error translation is enabled. Matches the
// repo-layer precedent in user_repository.go.
func isAttendanceDuplicateKeyErr(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "Error 1062")
}

func (r *attendanceRepository) Update(ctx context.Context, attendance *domain.Attendance) error {
	return r.getDB(ctx).Save(attendance).Error
}

func (r *attendanceRepository) HardDeleteByEmployeeID(ctx context.Context, employeeID uint) error {
	return r.getDB(ctx).
		Unscoped().
		Where("employee_id = ?", employeeID).
		Delete(&domain.Attendance{}).Error
}

// MarkAutoRejected atomically rejects an open, unrejected attendance. The
// conditional WHERE (check_out_time IS NULL AND salary_reject_reason IS NULL
// AND review_action IS NULL) is the race guard: a concurrent CheckOut, prior
// rejection, or authoritative admin review makes RowsAffected=0 — a safe no-op.
// This prevents a delayed worker from overwriting an admin decision.
func (r *attendanceRepository) MarkAutoRejected(ctx context.Context, id uint, reason string) (bool, error) {
	res := r.getDB(ctx).Model(&domain.Attendance{}).
		Where("id = ? AND check_out_time IS NULL AND salary_reject_reason IS NULL AND review_action IS NULL", id).
		Updates(map[string]interface{}{
			"earning_amount":       0,
			"salary_reject_reason": reason,
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkAdminReviewed stamps an admin review (approve/reject) on an attendance
// and applies the earning override atomically. The update is conditional on the
// id only — unlike MarkAutoRejected we WANT to allow reviewing an already-final
// row (e.g. an auto-rejected record being approved to restore earning, or a
// completed record being admin-rejected to claw back pay). The earningAmount +
// salary_reject_reason columns are derived from the action by the service layer
// so this method stays a faithful partial-update:
//   - approved: earningAmount is the recomputed value, salary_reject_reason cleared
//   - rejected: earningAmount is &zero, salary_reject_reason set to note
//
// Rejection is guarded against a concurrent approval or quota credit. Returns
// true if the transition applied, false if the row is missing or terminal.
func (r *attendanceRepository) MarkAdminReviewed(
	ctx context.Context,
	id uint,
	action domain.AttendanceReviewAction,
	note string,
	adminID uint,
	reviewedAt time.Time,
	earningAmount *int64,
	checkOutTime *time.Time,
	checkOutGate *string,
) (bool, error) {
	updates := map[string]interface{}{
		"review_action":  string(action),
		"review_note":    note,
		"reviewed_by":    adminID,
		"reviewed_at":    reviewedAt,
		"earning_amount": *earningAmount,
	}
	// Approve restores pay and clears any prior reject reason; reject zeros pay
	// and records the reason so GetStatus() still reads as rejected consistently.
	if action == domain.AttendanceReviewActionApproved {
		updates["salary_reject_reason"] = nil
		// Close the shift so the record is genuinely completed. Without this,
		// GetStatus() fakes "completed" via IsApproved() while check_out_time
		// stays NULL — which makes GetTodayAttendance's cross-day fallback
		// surface the stale record and block the employee's next check-in.
		if checkOutTime != nil {
			updates["check_out_time"] = *checkOutTime
		}
		if checkOutGate != nil {
			updates["check_out_gate"] = *checkOutGate
		}
	} else {
		updates["salary_reject_reason"] = note
	}

	query := r.getDB(ctx).Model(&domain.Attendance{}).Where("id = ?", id)
	if action == domain.AttendanceReviewActionRejected {
		// Rejection cannot claw back quota that has already been banked. Keep the
		// transition atomic with approval/quota credit so a stale service read
		// cannot overwrite an approval that won the race.
		query = query.Where("quota_credited_at IS NULL").
			Where("review_action IS NULL OR review_action <> ? OR salary_reject_reason IS NOT NULL", domain.AttendanceReviewActionApproved)
	}

	res := query.Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// MarkQuotaCredited atomically stamps quota_credited_at on an attendance whose
// earning is still payable. The conditional WHERE is the race guard: a
// concurrent credit that already stamped the row, or a rejection that zeroed
// the earning, makes RowsAffected=0. The service banks only after this claim.
func (r *attendanceRepository) MarkQuotaCredited(ctx context.Context, id uint, at time.Time) (bool, error) {
	res := r.getDB(ctx).Model(&domain.Attendance{}).
		Where("id = ? AND quota_credited_at IS NULL AND earning_amount > 0", id).
		Where("salary_reject_reason IS NULL").
		Where("review_action IS NULL OR review_action <> ?", domain.AttendanceReviewActionRejected).
		Update("quota_credited_at", at)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// GetOverdueQuotaCreditCandidates returns IDs whose persisted checkout deadline
// has elapsed but whose earnings have not been banked. Null deadlines are only
// possible when an older server completed checkout during the migration rollout;
// those retain the former 24-hour contract. Capped at limit per pass so the
// sweep stays bounded. The per-id MarkQuotaCredited guard makes overlapping runs
// with the per-attendance deferred task safe.
func (r *attendanceRepository) GetOverdueQuotaCreditCandidates(ctx context.Context, eligibleBefore time.Time, limit int) ([]uint, error) {
	if limit <= 0 {
		limit = 500
	}
	var ids []uint
	err := r.getDB(ctx).
		Model(&domain.Attendance{}).
		Where("quota_credited_at IS NULL").
		Where("earning_amount > 0").
		Where("(quota_credit_eligible_at IS NOT NULL AND quota_credit_eligible_at <= ?) OR (quota_credit_eligible_at IS NULL AND check_out_time <= ?)", eligibleBefore, eligibleBefore.Add(-domain.QuotaCreditHoldDuration)).
		Limit(limit).
		Order("quota_credit_eligible_at ASC, check_out_time ASC").
		Pluck("id", &ids).Error
	return ids, err
}

// RecalculatePendingQuotaCreditSchedules applies an Admin-updated hold duration
// to every completed, payable attendance whose earning has not been credited.
// The rows are locked while their persisted deadline is replaced, preventing a
// concurrent quota worker from banking an old deadline during the update.
func (r *attendanceRepository) RecalculatePendingQuotaCreditSchedules(ctx context.Context, hold time.Duration) ([]domain.QuotaCreditSchedule, error) {
	type pendingAttendance struct {
		ID           uint
		CheckOutTime time.Time
	}

	db := r.getDB(ctx)
	var pending []pendingAttendance
	if err := db.Model(&domain.Attendance{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("id", "check_out_time").
		Where("check_out_time IS NOT NULL").
		Where("earning_amount > 0").
		Where("quota_credited_at IS NULL").
		Where("salary_reject_reason IS NULL").
		Where("review_action IS NULL OR review_action <> ?", domain.AttendanceReviewActionRejected).
		Find(&pending).Error; err != nil {
		return nil, err
	}

	schedules := make([]domain.QuotaCreditSchedule, 0, len(pending))
	for _, attendance := range pending {
		eligibleAt := attendance.CheckOutTime.Add(hold)
		if err := db.Model(&domain.Attendance{}).
			Where("id = ? AND quota_credited_at IS NULL", attendance.ID).
			Update("quota_credit_eligible_at", eligibleAt).Error; err != nil {
			return nil, err
		}
		schedules = append(schedules, domain.QuotaCreditSchedule{
			AttendanceID: attendance.ID,
			EligibleAt:   eligibleAt,
		})
	}

	return schedules, nil
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

// GetApprovedOpenBefore finds a legacy row whose approval was represented only
// in status and never persisted a checkout. The open-row unique key permits at
// most one today, but newest-first remains safe for older installations.
func (r *attendanceRepository) GetApprovedOpenBefore(ctx context.Context, employeeID uint, before time.Time) (*domain.Attendance, error) {
	var att domain.Attendance
	err := r.getDB(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("employee_id = ? AND date < ?", employeeID, before.Format("2006-01-02")).
		Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND review_action = ?", domain.AttendanceReviewActionApproved).
		Order("date DESC, id DESC").
		First(&att).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &att, nil
}

func (r *attendanceRepository) CompleteApprovedOpen(ctx context.Context, id uint, checkOutTime time.Time, checkOutGate string) (bool, error) {
	result := r.getDB(ctx).Model(&domain.Attendance{}).
		Where("id = ? AND check_out_time IS NULL AND salary_reject_reason IS NULL AND review_action = ?", id, domain.AttendanceReviewActionApproved).
		Updates(map[string]interface{}{
			"check_out_time": checkOutTime,
			"check_out_gate": checkOutGate,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

func (r *attendanceRepository) GetOrphanCandidates(ctx context.Context, after, before time.Time) ([]*domain.Attendance, error) {
	var attendances []*domain.Attendance
	// Open (no checkout) AND unrejected records checked in within [after, before).
	// Reviewed rows are excluded because an admin decision is authoritative over
	// the delayed fallback worker.
	err := r.getDB(ctx).
		Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND review_action IS NULL AND check_in_time >= ? AND check_in_time < ?", after, before).
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
	useCheckInTimeWindow := filters.UseCheckInTimeWindow || (filters.SuccessfulCheckout != nil && *filters.SuccessfulCheckout)
	if filters.FromDate != nil {
		if useCheckInTimeWindow {
			query = query.Where("check_in_time >= ?", *filters.FromDate)
		} else {
			query = query.Where("date >= ?", *filters.FromDate)
		}
	}
	if filters.ToDate != nil {
		if useCheckInTimeWindow {
			query = query.Where("check_in_time < ?", filters.ToDate.AddDate(0, 0, 1))
		} else {
			query = query.Where("date <= ?", *filters.ToDate)
		}
	}
	if filters.Status != nil {
		switch *filters.Status {
		case domain.AttendanceStatusCompleted:
			query = query.Where(
				"(review_action IS NULL OR review_action <> ?) AND (check_out_time IS NOT NULL OR (review_action = ? AND salary_reject_reason IS NULL))",
				domain.AttendanceReviewActionRejected,
				domain.AttendanceReviewActionApproved,
			)
		case domain.AttendanceStatusRejected:
			query = query.Where("review_action = ? OR (check_out_time IS NULL AND salary_reject_reason IS NOT NULL)", domain.AttendanceReviewActionRejected)
		case domain.AttendanceStatusCheckedIn:
			query = query.Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND (review_action IS NULL OR review_action <> ?) AND check_in_time >= ?", domain.AttendanceReviewActionApproved, clock.Now().Add(-18*time.Hour))
		case domain.AttendanceStatusOrphaned:
			query = query.Where("check_out_time IS NULL AND salary_reject_reason IS NULL AND (review_action IS NULL OR review_action <> ?) AND check_in_time < ?", domain.AttendanceReviewActionApproved, clock.Now().Add(-18*time.Hour))
		}
	}
	if filters.SuccessfulCheckout != nil && *filters.SuccessfulCheckout {
		query = query.
			Where("check_out_time IS NOT NULL AND earning_amount > 0").
			Where(checkInEnabledScope("attendances"))
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
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL AND (review_action IS NULL OR review_action <> 'approved') AND check_in_time >= ? THEN 1 ELSE 0 END) as open_checked_in,
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NULL AND (review_action IS NULL OR review_action <> 'approved') AND check_in_time < ? THEN 1 ELSE 0 END) as orphaned,
			SUM(CASE WHEN check_out_time IS NULL AND salary_reject_reason IS NOT NULL THEN 1 ELSE 0 END) as auto_rejected,
			SUM(CASE WHEN check_out_time IS NOT NULL AND (review_action IS NULL OR review_action <> 'rejected') AND (earning_amount IS NULL OR earning_amount = 0) THEN 1 ELSE 0 END) as completed_zero_earning,
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
