package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/infra/persistence/repositories"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

type TimesheetRepository struct {
	*BaseRepository
	queryRepo      *repositories.TimesheetQueryRepository
	commandRepo    *repositories.TimesheetCommandRepository
	analyticsRepo  *repositories.TimesheetAnalyticsRepository
	batchProcessor *common.BatchProcessor
	errorHandler   *common.RepoErrorHandler
}

func NewTimesheetRepository(db *Database, projectEmployeeRepo domain.ProjectEmployeeRepository) domain.TimesheetRepository {
	return &TimesheetRepository{
		BaseRepository: NewBaseRepository(db),
		queryRepo:      repositories.NewTimesheetQueryRepository(db.DB),
		commandRepo:    repositories.NewTimesheetCommandRepository(db.DB),
		analyticsRepo:  repositories.NewTimesheetAnalyticsRepository(db.DB),
		batchProcessor: common.NewBatchProcessor(common.DefaultBatchConfig()),
		errorHandler:   common.NewRepoErrorHandler(),
	}
}

func (r *TimesheetRepository) Create(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.commandRepo.Create(ctx, timesheet)
}

func (r *TimesheetRepository) GetByID(ctx context.Context, id uint) (*domain.Timesheet, error) {
	return r.queryRepo.GetByID(ctx, id)
}

func (r *TimesheetRepository) GetByIDs(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByIDs(ctx, ids)
}

func (r *TimesheetRepository) GetByIDsWithoutRelations(ctx context.Context, ids []uint) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByIDsWithoutRelations(ctx, ids)
}

func (r *TimesheetRepository) GetByTransactionID(ctx context.Context, transactionID uint) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet
	db := r.getDB(ctx)
	if err := db.Where("transaction_id = ?", transactionID).Find(&timesheets).Error; err != nil {
		return nil, err
	}
	return timesheets, nil
}

func (r *TimesheetRepository) Update(ctx context.Context, timesheet *domain.Timesheet) error {
	return r.commandRepo.Update(ctx, timesheet)
}

func (r *TimesheetRepository) Delete(ctx context.Context, id uint) error {
	return r.commandRepo.Delete(ctx, id)
}

// HardDelete permanently removes a timesheet row.
// Used by BCC import "latest wins" overwrite — stale entries should not accumulate as soft-deleted rows.
func (r *TimesheetRepository) HardDelete(ctx context.Context, id uint) error {
	return r.commandRepo.HardDelete(ctx, id)
}

func (r *TimesheetRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	return r.DB.WithContext(ctx).
		Where("project_id = ?", projectID).
		Delete(&domain.Timesheet{}).Error
}

func (r *TimesheetRepository) List(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	return r.queryRepo.List(ctx, filters)
}

func (r *TimesheetRepository) Count(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	return r.queryRepo.Count(ctx, filters)
}

func (r *TimesheetRepository) GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByProjectAndEmployee(ctx, projectID, employeeID, fromDate, toDate)
}

func (r *TimesheetRepository) GetByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByProject(ctx, projectID, fromDate, toDate)
}

func (r *TimesheetRepository) GetByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByEmployee(ctx, employeeID, fromDate, toDate)
}

func (r *TimesheetRepository) BulkCreate(ctx context.Context, timesheets []*domain.Timesheet) error {
	return r.commandRepo.BulkCreate(ctx, timesheets)
}

func (r *TimesheetRepository) BulkUpdate(ctx context.Context, timesheets []*domain.Timesheet) error {
	return r.commandRepo.BulkUpdate(ctx, timesheets)
}

func (r *TimesheetRepository) Approve(ctx context.Context, id uint, approvedBy uint) error {
	return r.commandRepo.Approve(ctx, id, approvedBy)
}

func (r *TimesheetRepository) BulkApprove(ctx context.Context, ids []uint, approvedBy uint) error {
	return r.commandRepo.BulkApprove(ctx, ids, approvedBy)
}

func (r *TimesheetRepository) BulkReject(ctx context.Context, ids []uint, rejectionReason string) error {
	return r.commandRepo.BulkReject(ctx, ids, rejectionReason)
}

func (r *TimesheetRepository) Reject(ctx context.Context, id uint, rejectionReason string) error {
	return r.commandRepo.Reject(ctx, id, rejectionReason)
}

func (r *TimesheetRepository) Reset(ctx context.Context, id uint) error {
	return r.commandRepo.Reset(ctx, id)
}

func (r *TimesheetRepository) GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	return r.analyticsRepo.GetSummaryStats(ctx, filters)
}

func (r *TimesheetRepository) GetByProjectEmployeeDatePaytype(ctx context.Context, projectID, employeeID uint, date time.Time, paytype string) (*domain.Timesheet, error) {
	return r.queryRepo.GetByProjectEmployeeDatePaytype(ctx, projectID, employeeID, date, paytype)
}

func (r *TimesheetRepository) GetByProjectEmployeeDateHourType(ctx context.Context, projectID, employeeID uint, date time.Time, hourType string) (*domain.Timesheet, error) {
	return r.queryRepo.GetByProjectEmployeeDateHourType(ctx, projectID, employeeID, date, hourType)
}

func (r *TimesheetRepository) GetByProjectEmployeeDate(ctx context.Context, projectID, employeeID uint, date time.Time) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByProjectEmployeeDate(ctx, projectID, employeeID, date)
}

func (r *TimesheetRepository) GetByEmployeeDateCombos(ctx context.Context, combos []domain.EmployeeDateCombo) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetByEmployeeDateCombos(ctx, combos)
}

func (r *TimesheetRepository) GetSummaryByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) (*domain.TimesheetSummary, error) {
	return r.analyticsRepo.GetSummaryByProject(ctx, projectID, fromDate, toDate)
}

// Note: applyFilters method has been moved to TimesheetQueryBuilder

func (r *TimesheetRepository) SetForcePayroll(ctx context.Context, id uint, flag bool) error {
	return r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id = ?", id).
		Update("force_payroll", flag).Error
}

// GetPendingApprovalCount returns count of timesheets pending approval within date range
func (r *TimesheetRepository) GetPendingApprovalCount(ctx context.Context, startDate, endDate time.Time) (int, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ? AND ?", domain.TimesheetStatusPendingApproval, startDate, endDate).
		Count(&count).Error

	return int(count), err
}

// GetApprovedSalaryTotal returns total approved salary amount within date range
func (r *TimesheetRepository) GetApprovedSalaryTotal(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("timesheet_status = ? AND date BETWEEN ? AND ?", domain.TimesheetStatusApproved, startDate, endDate).
		Select("COALESCE(SUM(amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetByEmployeeAndPeriod retrieves timesheets for an employee within a date range
func (r *TimesheetRepository) GetByEmployeeAndPeriod(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	var timesheets []*domain.Timesheet
	err := r.DB.WithContext(ctx).
		Preload("Project").
		Preload("Employee").
		Preload("CreatedUser").
		Preload("ApprovedUser").
		Where("timesheets.employee_id = ? AND timesheets.date BETWEEN ? AND ?", employeeID, fromDate, toDate).
		Order("timesheets.date DESC").
		Find(&timesheets).Error

	return timesheets, err
}

// BulkUpdatePaymentStatus updates payment status for multiple timesheets using batch CASE WHEN updates.
func (r *TimesheetRepository) BulkUpdatePaymentStatus(ctx context.Context, updates []domain.PaymentStatusUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	db := r.getDB(ctx)

	return db.Transaction(func(tx *gorm.DB) error {
		// Build CASE WHEN expressions with parameterized queries to prevent SQL injection
		idCases := make([]string, len(updates))
		statusCases := make([]string, len(updates))
		paidAtCases := make([]string, len(updates))
		statusArgs := make([]interface{}, 0, len(updates))
		paidAtArgs := make([]interface{}, 0, len(updates))

		now := clock.Now()
		for i, u := range updates {
			idCases[i] = fmt.Sprintf("WHEN %d THEN %d", u.TimesheetID, u.TimesheetID)
			statusCases[i] = fmt.Sprintf("WHEN %d THEN ?", u.TimesheetID)
			statusArgs = append(statusArgs, u.PaymentStatus)

			if u.PaidAt != nil {
				paidAtCases[i] = fmt.Sprintf("WHEN %d THEN ?", u.TimesheetID)
				paidAtArgs = append(paidAtArgs, u.PaidAt)
			} else if u.PaymentStatus == domain.PaymentStatusPaid {
				paidAtCases[i] = fmt.Sprintf("WHEN %d THEN ?", u.TimesheetID)
				paidAtArgs = append(paidAtArgs, now)
			} else {
				paidAtCases[i] = fmt.Sprintf("WHEN %d THEN NULL", u.TimesheetID)
			}
		}

		ids := make([]uint, len(updates))
		for i, u := range updates {
			ids[i] = u.TimesheetID
		}

		statusExpr := "CASE id " + strings.Join(statusCases, " ") + " END"
		paidAtExpr := "CASE id " + strings.Join(paidAtCases, " ") + " END"

		updateFields := map[string]interface{}{
			"payment_status": gorm.Expr(statusExpr, statusArgs...),
			"paid_at":        gorm.Expr(paidAtExpr, paidAtArgs...),
		}

		// Add optional fields if any update has them
		hasPaymentRef := false
		hasPaymentDate := false
		hasPaidAmount := false
		for _, u := range updates {
			if u.PaymentReference != nil {
				hasPaymentRef = true
			}
			if u.PaymentDate != nil {
				hasPaymentDate = true
			}
			if u.PaidAmount != nil {
				hasPaidAmount = true
			}
		}

		if hasPaymentRef {
			refCases := make([]string, len(updates))
			refArgs := make([]interface{}, 0, len(updates))
			for i, u := range updates {
				if u.PaymentReference != nil {
					refCases[i] = fmt.Sprintf("WHEN %d THEN ?", u.TimesheetID)
					refArgs = append(refArgs, *u.PaymentReference)
				} else {
					refCases[i] = fmt.Sprintf("WHEN %d THEN payment_reference", u.TimesheetID)
				}
			}
			updateFields["payment_reference"] = gorm.Expr("CASE id "+strings.Join(refCases, " ")+" END", refArgs...)
		}

		if hasPaymentDate {
			dateCases := make([]string, len(updates))
			dateArgs := make([]interface{}, 0, len(updates))
			for i, u := range updates {
				if u.PaymentDate != nil {
					dateCases[i] = fmt.Sprintf("WHEN %d THEN ?", u.TimesheetID)
					dateArgs = append(dateArgs, u.PaymentDate.Format("2006-01-02"))
				} else {
					dateCases[i] = fmt.Sprintf("WHEN %d THEN payment_date", u.TimesheetID)
				}
			}
			updateFields["payment_date"] = gorm.Expr("CASE id "+strings.Join(dateCases, " ")+" END", dateArgs...)
		}

		if hasPaidAmount {
			amountCases := make([]string, len(updates))
			for i, u := range updates {
				if u.PaidAmount != nil {
					amountCases[i] = fmt.Sprintf("WHEN %d THEN %d", u.TimesheetID, *u.PaidAmount)
				} else {
					amountCases[i] = fmt.Sprintf("WHEN %d THEN paid_amount", u.TimesheetID)
				}
			}
			updateFields["paid_amount"] = gorm.Expr("CASE id " + strings.Join(amountCases, " ") + " END")
		}

		return tx.Model(&domain.Timesheet{}).
			Where("id IN ?", ids).
			Updates(updateFields).Error
	})
}

// CountDistinctWorkingDays counts unique dates with approved timesheets for an employee-project combination
func (r *TimesheetRepository) CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, fromDate, toDate time.Time) (int, error) {
	var dates []time.Time
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ? AND project_id = ? AND timesheet_status = ? AND date BETWEEN ? AND ?",
			employeeID, projectID, domain.TimesheetStatusApproved, fromDate, toDate).
		Distinct("date").
		Pluck("date", &dates).Error
	return len(dates), err
}

// GetProjectsForEmployeeOnDate returns distinct project IDs for an employee on a specific date
func (r *TimesheetRepository) GetProjectsForEmployeeOnDate(ctx context.Context, employeeID uint, date time.Time) ([]uint, error) {
	var projectIDs []uint
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ? AND date = ?", employeeID, date).
		Distinct("project_id").
		Pluck("project_id", &projectIDs).Error
	return projectIDs, err
}

// GetLatestTimesheetDate returns the most recent timesheet date for an employee in a project
func (r *TimesheetRepository) GetLatestTimesheetDate(ctx context.Context, projectID, employeeID uint) (*time.Time, error) {
	var latestDate sql.NullTime
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Select("MAX(date)").
		Where("project_id = ? AND employee_id = ?", projectID, employeeID).
		Scan(&latestDate).Error

	if err != nil {
		return nil, err
	}

	if !latestDate.Valid {
		return nil, nil
	}

	return &latestDate.Time, nil
}

// CountTimesheets returns the total number of timesheets for an employee in a project
func (r *TimesheetRepository) CountTimesheets(ctx context.Context, projectID, employeeID uint) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ?", projectID, employeeID).
		Count(&count).Error

	return count, err
}

// CountTimesheetsByEmployeeID returns the total number of timesheets for an employee across all projects
func (r *TimesheetRepository) CountTimesheetsByEmployeeID(ctx context.Context, employeeID uint) (int64, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ?", employeeID).
		Count(&count).Error

	return count, err
}

// HasTimesheetsAfterDate checks if there are any timesheets after the given date
func (r *TimesheetRepository) HasTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ? AND date > ?", projectID, employeeID, date).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasNonEditableTimesheetsAfterDate checks if there are approved or paid timesheets on or after the given date
func (r *TimesheetRepository) HasNonEditableTimesheetsAfterDate(ctx context.Context, projectID, employeeID uint, date time.Time) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ? AND date >= ?", projectID, employeeID, date).
		Where("timesheet_status = ? OR payment_status = ?", domain.TimesheetStatusApproved, domain.PaymentStatusPaid).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// HasTimesheetsForAssignment checks if there are any timesheets for the given assignment
func (r *TimesheetRepository) HasTimesheetsForAssignment(ctx context.Context, projectID, employeeID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ?", projectID, employeeID).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetPaymentHistories retrieves payment histories by aggregating paid timesheets using SQL.
// Returns one record per employee per payment date per project to show all payment occurrences.
func (r *TimesheetRepository) GetPaymentHistories(ctx context.Context, filters domain.PaymentHistoryFilters) ([]*domain.PaymentHistory, int64, error) {
	// Build base aggregation query with all filters pushed to SQL
	baseQuery := r.DB.WithContext(ctx).
		Table("timesheets t").
		Select(`t.employee_id, t.project_id, DATE(t.paid_at) as paid_date, MAX(t.paid_at) as paid_at, SUM(t.paid_amount) as total_paid`).
		Where("t.payment_status = ?", domain.PaymentStatusPaid).
		Where("t.paid_at IS NOT NULL").
		Where("t.deleted_at IS NULL")

	// Apply project filter
	if len(filters.ProjectIDs) > 0 {
		baseQuery = baseQuery.Where("t.project_id IN ?", filters.ProjectIDs)
	}

	// Apply employee filter
	if len(filters.EmployeeIDs) > 0 {
		baseQuery = baseQuery.Where("t.employee_id IN ?", filters.EmployeeIDs)
	}

	// Apply paid_at date range
	if filters.FromDate != nil {
		baseQuery = baseQuery.Where("t.paid_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		endOfDay := filters.ToDate.Add(24*time.Hour - time.Nanosecond)
		baseQuery = baseQuery.Where("t.paid_at <= ?", endOfDay)
	}

	// Apply partner access control via JOINs
	if filters.EmployeeCreatedBy != nil {
		baseQuery = baseQuery.Joins("JOIN employees e ON e.id = t.employee_id AND e.created_by = ? AND e.deleted_at IS NULL", *filters.EmployeeCreatedBy)
	}
	if filters.EmployeeAssignedByPartner != nil {
		baseQuery = baseQuery.Joins("JOIN project_employees pe ON pe.employee_id = t.employee_id AND pe.project_id = t.project_id AND pe.created_by = ? AND pe.deleted_at IS NULL", *filters.EmployeeAssignedByPartner)
	}

	baseQuery = baseQuery.Group("t.employee_id, t.project_id, DATE(t.paid_at)")

	// Apply position filter via subquery
	if filters.Position != "" {
		baseQuery = baseQuery.Where(`EXISTS (
			SELECT 1 FROM project_employees pe2
			WHERE pe2.employee_id = t.employee_id
				AND pe2.project_id = t.project_id
				AND pe2.position = ?
				AND pe2.deleted_at IS NULL
		)`, filters.Position)
	}

	// Apply search filter on employee fullname/CCCD
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		baseQuery = baseQuery.Where(`EXISTS (
			SELECT 1 FROM employees e2
			WHERE e2.id = t.employee_id
				AND (e2.search_normalized LIKE ? OR e2.cccd LIKE ?)
				AND e2.deleted_at IS NULL
		)`, normalizedSearch, "%"+filters.Search+"%")
	}

	// Count total aggregated records
	countQuery := r.DB.WithContext(ctx).Table("(?) as agg", baseQuery)
	var totalCount int64
	if err := countQuery.Count(&totalCount).Error; err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "payment histories")
	}

	// Determine sort column
	sortBy := "paid_at"
	switch filters.SortBy {
	case "employee_name":
		sortBy = "e3.fullname"
	case "amount":
		sortBy = "total_paid"
	case "project_name":
		sortBy = "p.name"
	}
	sortOrder := "DESC"
	if filters.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Fetch paginated results with employee/project names and position
	selectQuery := r.DB.WithContext(ctx).
		Table("(?) as agg", baseQuery).
		Select(`agg.employee_id, agg.project_id, agg.paid_at, agg.total_paid as total_paid_amount,
			e3.fullname as employee_name, e3.cccd as employee_cccd,
			p.name as project_name,
			(SELECT pe3.position FROM project_employees pe3
				WHERE pe3.employee_id = agg.employee_id
					AND pe3.project_id = agg.project_id
					AND pe3.deleted_at IS NULL
				ORDER BY pe3.created_at DESC LIMIT 1) as position`).
		Joins("JOIN employees e3 ON e3.id = agg.employee_id AND e3.deleted_at IS NULL").
		Joins("JOIN projects p ON p.id = agg.project_id AND p.deleted_at IS NULL").
		Order(sortBy + " " + sortOrder)

	// Handle special case: return all results
	if filters.Limit != -1 && filters.Limit > 0 {
		selectQuery = selectQuery.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		selectQuery = selectQuery.Offset(filters.Offset)
	}

	var results []*domain.PaymentHistory
	if err := selectQuery.Scan(&results).Error; err != nil {
		return nil, 0, r.errorHandler.HandleListError(err, "payment histories")
	}

	return results, totalCount, nil
}

// GetTotalPaidSalary returns the total paid salary of all time
func (r *TimesheetRepository) GetTotalPaidSalary(ctx context.Context) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ? AND deleted_at IS NULL", domain.PaymentStatusPaid).
		Select("COALESCE(SUM(paid_amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetPendingSalaryForMonth returns pending salary for the current month
// Calculates sum of timesheets that are approved or pending_approval but not yet paid
func (r *TimesheetRepository) GetPendingSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Where("timesheet_status IN (?)", []domain.TimesheetStatus{
			domain.TimesheetStatusApproved,
			domain.TimesheetStatusPendingApproval,
		}).
		Where("payment_status = ?", domain.PaymentStatusPending).
		Select("COALESCE(SUM(amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// GetPendingSalaryForMonthBySchedule computes pending salary by payment schedule.
// Weekly: use caller date range directly.
// Monthly: derive the actual salary window from project's salary_period_from/to.
func (r *TimesheetRepository) GetPendingSalaryForMonthBySchedule(
	ctx context.Context,
	startDate, endDate time.Time,
	schedule domain.PaymentSchedule,
) (int64, error) {

	if schedule == domain.PaymentScheduleWeekly {
		return r.sumPendingWeekly(ctx, startDate, endDate)
	}

	// Monthly: resolve real range once, not in SQL
	// Example: salary_period_from/to = 26 → 25 means 26 previous month to 25 this month
	// Resolve using helper
	periodStart, periodEnd := deriveMonthlySalaryWindow(startDate)

	return r.sumPendingMonthly(ctx, periodStart, periodEnd)
}

// deriveMonthlySalaryWindow calculates the salary period window for monthly schedules
// Returns the start and end dates for the salary period that includes the reference date
func deriveMonthlySalaryWindow(ref time.Time) (time.Time, time.Time) {
	// ref is any date inside the target month
	year, month, _ := ref.Date()
	loc := ref.Location()

	// Get previous month
	prev := ref.AddDate(0, -1, 0)
	py, pm, _ := prev.Date()

	return time.Date(py, pm, 26, 0, 0, 0, 0, loc), // inclusive start: 26th of previous month
		time.Date(year, month, 25, 23, 59, 59, 999000000, loc) // inclusive end: 25th of current month
}

// sumPendingBySchedule calculates pending salary for a given payment schedule.
func (r *TimesheetRepository) sumPendingBySchedule(
	ctx context.Context,
	start, end time.Time,
	schedule domain.PaymentSchedule,
) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).
		Table("timesheets t").
		Joins("JOIN project_employees pe ON t.employee_id = pe.employee_id AND t.project_id = pe.project_id").
		Where("t.date BETWEEN ? AND ?", start, end).
		Where("t.timesheet_status IN ?", []domain.TimesheetStatus{
			domain.TimesheetStatusApproved,
			domain.TimesheetStatusPendingApproval,
		}).
		Where("t.payment_status <> ?", domain.PaymentStatusPaid).
		Where("pe.payment_schedule = ?", schedule).
		Select("COALESCE(SUM(t.amount), 0)").
		Row().
		Scan(&total)
	return total, err
}

func (r *TimesheetRepository) sumPendingWeekly(ctx context.Context, start, end time.Time) (int64, error) {
	return r.sumPendingBySchedule(ctx, start, end, domain.PaymentScheduleWeekly)
}

func (r *TimesheetRepository) sumPendingMonthly(ctx context.Context, start, end time.Time) (int64, error) {
	return r.sumPendingBySchedule(ctx, start, end, domain.PaymentScheduleMonthly)
}

// GetPaidSalaryForMonth returns paid salary for timesheets with work date in the given month
func (r *TimesheetRepository) GetPaidSalaryForMonth(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var total int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("date BETWEEN ? AND ?", startDate, endDate).
		Select("COALESCE(SUM(paid_amount), 0)").
		Row().
		Scan(&total)

	return total, err
}

// CountEmployeesPaidInLast30Days returns count of distinct employees paid in the last 30 days
func (r *TimesheetRepository) CountEmployeesPaidInLast30Days(ctx context.Context) (int, error) {
	thirtyDaysAgo := clock.Now().AddDate(0, 0, -30)

	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ? AND paid_at >= ?", domain.PaymentStatusPaid, thirtyDaysAgo).
		Distinct("employee_id").
		Count(&count).Error

	return int(count), err
}

// GetFirstPaidTimesheetDate returns the earliest paid_at date
func (r *TimesheetRepository) GetFirstPaidTimesheetDate(ctx context.Context) (*time.Time, error) {
	var firstDate time.Time
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Order("paid_at ASC").
		Limit(1).
		Select("paid_at").
		Row().
		Scan(&firstDate)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &firstDate, nil
}

// GetPaidSalaryByEmployee returns total paid salary for each employee in the given period
func (r *TimesheetRepository) GetPaidSalaryByEmployee(ctx context.Context, startDate, endDate time.Time) (map[uint]int64, error) {
	var results []struct {
		EmployeeID uint
		TotalPaid  int64
	}

	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("paid_at BETWEEN ? AND ?", startDate, endDate).
		Select("employee_id, SUM(paid_amount) as total_paid").
		Group("employee_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	salaryMap := make(map[uint]int64)
	for _, r := range results {
		salaryMap[r.EmployeeID] = r.TotalPaid
	}
	return salaryMap, nil
}

// BatchUpdatePaymentStatusToFailed marks timesheets as failed
// Must be called within a transaction
func (r *TimesheetRepository) BatchUpdatePaymentStatusToFailed(ctx context.Context, tx interface{}, timesheetIDs []uint) error {
	db, err := common.ExtractDB(tx)
	if err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	if len(timesheetIDs) == 0 {
		return nil
	}

	return db.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("id IN ?", timesheetIDs).
		Update("payment_status", domain.PaymentStatusFailed).Error
}

// BulkUpdateRevenuePaid marks revenue as paid for specified timesheets
func (r *TimesheetRepository) BulkUpdateRevenuePaid(ctx context.Context, timesheetIDs []uint) error {
	if len(timesheetIDs) == 0 {
		return nil
	}

	// Process in batches to avoid hitting MySQL's IN clause limit
	const batchSize = 1000
	for i := 0; i < len(timesheetIDs); i += batchSize {
		end := i + batchSize
		if end > len(timesheetIDs) {
			end = len(timesheetIDs)
		}

		batch := timesheetIDs[i:end]
		if err := r.getDB(ctx).
			Model(&domain.Timesheet{}).
			Where("id IN ?", batch).
			Update("revenue_paid", true).Error; err != nil {
			return fmt.Errorf("failed to update revenue_paid for batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

// BulkUpdateRevenueReceivable updates revenue_receivable for specified timesheets using batch CASE WHEN.
func (r *TimesheetRepository) BulkUpdateRevenueReceivable(ctx context.Context, updates map[uint]int64) error {
	if len(updates) == 0 {
		return nil
	}

	ids := make([]uint, 0, len(updates))
	caseClauses := make([]string, 0, len(updates))
	for id, val := range updates {
		ids = append(ids, id)
		caseClauses = append(caseClauses, fmt.Sprintf("WHEN %d THEN %d", id, val))
	}

	caseExpr := "CASE id " + strings.Join(caseClauses, " ") + " END"
	return r.getDB(ctx).
		Model(&domain.Timesheet{}).
		Where("id IN ?", ids).
		Update("revenue_receivable", gorm.Expr(caseExpr)).Error
}

// BulkUpdateTransactionID links timesheets to a single transaction
func (r *TimesheetRepository) BulkUpdateTransactionID(ctx context.Context, transactionID uint, timesheetIDs []uint) error {
	if len(timesheetIDs) == 0 {
		return nil
	}

	// Get DB connection (respect existing transaction context)
	db := r.getDB(ctx)

	// Process in batches to avoid hitting MySQL's IN clause limit
	const batchSize = 1000
	for i := 0; i < len(timesheetIDs); i += batchSize {
		end := i + batchSize
		if end > len(timesheetIDs) {
			end = len(timesheetIDs)
		}

		batch := timesheetIDs[i:end]
		if err := db.Model(&domain.Timesheet{}).
			Where("id IN ?", batch).
			Update("transaction_id", transactionID).Error; err != nil {
			return fmt.Errorf("failed to update transaction_id for batch %d-%d: %w", i, end, err)
		}
	}

	return nil
}

// GetEmployeeTimesheetSummaryAggregated returns aggregated summary data using SQL aggregation
func (r *TimesheetRepository) GetEmployeeTimesheetSummaryAggregated(ctx context.Context, employeeID uint) (*domain.TimesheetSummaryAggregated, error) {
	var result domain.TimesheetSummaryAggregated

	// Get all aggregated data in a single query
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ? AND deleted_at IS NULL", employeeID).
		Select(`
			COUNT(*) as total_entries,
			COALESCE(SUM(hours_worked), 0) as total_hours,
			COALESCE(SUM(amount), 0) as total_amount,
			COUNT(CASE WHEN timesheet_status = ? THEN 1 END) as pending_entries,
			COUNT(CASE WHEN timesheet_status = ? THEN 1 END) as approved_entries,
			COUNT(CASE WHEN timesheet_status = ? THEN 1 END) as rejected_entries,
			COUNT(DISTINCT DATE(date)) as working_days,
			MAX(date) as last_entry_date
		`, domain.TimesheetStatusPendingApproval, domain.TimesheetStatusApproved, domain.TimesheetStatusRejected).
		Row().
		Scan(&result.TotalEntries, &result.TotalHours, &result.TotalAmount,
			&result.PendingEntries, &result.ApprovedEntries, &result.RejectedEntries,
			&result.WorkingDays, &result.LastEntryDate)

	if err != nil {
		return nil, err
	}

	// Calculate average hours per day
	if result.WorkingDays > 0 {
		result.AverageHoursPerDay = result.TotalHours / float64(result.WorkingDays)
	}

	return &result, nil
}

// GetEmployeeCurrentWeekHours returns total hours for current ISO week using SQL aggregation
func (r *TimesheetRepository) GetEmployeeCurrentWeekHours(ctx context.Context, employeeID uint) (float64, error) {
	now := clock.Now()

	// Get start of current ISO week (Monday)
	weekday := int(now.Weekday())
	if weekday == 0 { // Sunday is 0 in Go, but ISO week starts with Monday
		weekday = 7
	}
	startOfWeek := now.AddDate(0, 0, -(weekday - 1))
	startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, startOfWeek.Location())

	// Get end of current ISO week (Sunday)
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	endOfWeek = time.Date(endOfWeek.Year(), endOfWeek.Month(), endOfWeek.Day(), 23, 59, 59, 999999999, endOfWeek.Location())

	var totalHours float64
	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ? AND deleted_at IS NULL AND date BETWEEN ? AND ?",
			employeeID, startOfWeek, endOfWeek).
		Select("COALESCE(SUM(hours_worked), 0)").
		Row().
		Scan(&totalHours)

	return totalHours, err
}

// GetEmployeeMonthlyPayrollSummary returns payroll summary for current month using SQL aggregation
func (r *TimesheetRepository) GetEmployeeMonthlyPayrollSummary(ctx context.Context, employeeID uint) (*domain.PayrollSummaryAggregated, error) {
	now := clock.Now()
	year, month, _ := now.Date()
	firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, now.Location())
	lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1).Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	var result domain.PayrollSummaryAggregated

	err := r.DB.WithContext(ctx).
		Model(&domain.Timesheet{}).
		Where("employee_id = ? AND deleted_at IS NULL AND payment_status = ? AND paid_at BETWEEN ?",
			employeeID, domain.PaymentStatusPaid, firstDayOfMonth, lastDayOfMonth).
		Select(`
			COUNT(*) as total_payments,
			COALESCE(SUM(paid_amount), 0) as total_paid_amount,
			MAX(paid_at) as last_payment_date
		`).
		Row().
		Scan(&result.TotalPayments, &result.TotalPaidAmount, &result.LastPaymentDate)

	if err != nil {
		return nil, err
	}

	// Calculate average weekly earnings: (total received amount in a month) / 4
	if result.TotalPaidAmount > 0 {
		result.AvgWeeklyEarnings = result.TotalPaidAmount / 4
	}

	return &result, nil
}

// GetEntryTableData retrieves timesheets for entry table reporting with specific filters
func (r *TimesheetRepository) GetEntryTableData(ctx context.Context, projectID uint, employeeIDs []uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	return r.queryRepo.GetEntryTableData(ctx, projectID, employeeIDs, fromDate, toDate)
}

// ListGroupedByEmployee retrieves timesheets grouped by employee with server-side pagination
func (r *TimesheetRepository) ListGroupedByEmployee(ctx context.Context, filters domain.TimesheetFilters) ([]domain.EmployeeGroupResult, []*domain.Timesheet, int64, error) {
	return r.queryRepo.ListGroupedByEmployee(ctx, filters)
}

// CountDistinctEmployees returns the count of distinct employees matching filters
func (r *TimesheetRepository) CountDistinctEmployees(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	return r.queryRepo.CountDistinctEmployees(ctx, filters)
}

// GetPaidTimesheetsInDateRange retrieves all timesheets with payment_status='paid' in the given date range
func (r *TimesheetRepository) GetPaidTimesheetsInDateRange(ctx context.Context, startDate, endDate time.Time, timesheets *[]*domain.Timesheet) error {
	err := r.getDB(ctx).
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("(paid_at BETWEEN ? AND ? OR payment_date BETWEEN ? AND ?)",
			startDate, endDate, startDate, endDate).
		Find(timesheets).Error

	return err
}

// CountDistinctEmployeesPaidInPeriod counts distinct employees who have paid timesheets in the given period
func (r *TimesheetRepository) CountDistinctEmployeesPaidInPeriod(ctx context.Context, startDate, endDate time.Time) (int, error) {
	var result struct {
		Count int `gorm:"column:count"`
	}

	err := r.getDB(ctx).
		Model(&domain.Timesheet{}).
		Select("COUNT(DISTINCT employee_id) as count").
		Where("payment_status = ?", domain.PaymentStatusPaid).
		Where("(paid_at BETWEEN ? AND ? OR payment_date BETWEEN ? AND ?)",
			startDate, endDate, startDate, endDate).
		Scan(&result).Error

	return result.Count, err
}

// CountUnsettledPaidTimesheets delegates to query repository
func (r *TimesheetRepository) CountUnsettledPaidTimesheets(ctx context.Context, projectID uint) (int64, error) {
	return r.queryRepo.CountUnsettledPaidTimesheets(ctx, projectID)
}

// getDB gets the appropriate DB instance from context or falls back to default
func (r *TimesheetRepository) getDB(ctx context.Context) *gorm.DB {
	// Check if there's a transaction context
	if txCtx, ok := domain.GetTransactionFromContext(ctx); ok && txCtx.TX != nil {
		return txCtx.TX.WithContext(ctx)
	}
	return r.DB.WithContext(ctx)
}
