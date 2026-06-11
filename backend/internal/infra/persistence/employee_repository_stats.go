package persistence

import (
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
	"context"
	"fmt"
	"time"

	"api-server/internal/domain"
)

func (r *EmployeeRepository) GetEmployeesWithBankingInfo(ctx context.Context) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.queryBuilder.BuildGetWithBankingInfoQuery()
	err := query.Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetEmployeeStatistics(ctx context.Context) (*domain.EmployeeStatistics, error) {
	var stats domain.EmployeeStatistics

	// Use conditional aggregation to get all counts in one query
	err := r.statisticsBuilder.BuildStatisticsQuery().Scan(&stats).Error
	if err != nil {
		return nil, err
	}

	// Separate query for JOIN-based count (assigned to projects)
	var assignedCount int64
	err = r.statisticsBuilder.BuildStatisticsAssignedQuery().Count(&assignedCount).Error
	if err != nil {
		return nil, err
	}
	stats.AssignedToProjects = assignedCount

	stats.UnassignedEmployees = stats.TotalEmployees - stats.AssignedToProjects

	return &stats, nil
}

func (r *EmployeeRepository) GetEmployeesByAgeRange(ctx context.Context, minAge, maxAge int) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.queryBuilder.BuildGetByAgeRangeQuery(minAge, maxAge)
	err := query.Find(&employees).Error
	return employees, err
}

func (r *EmployeeRepository) GetDuplicateCCCDs(ctx context.Context) ([]string, error) {
	var cccds []string
	query := r.queryBuilder.BuildGetDuplicateCCCDsQuery()
	err := query.Pluck("cccd", &cccds).Error
	return cccds, err
}

// GetEmployeesSummary returns aggregated metrics for all employees
func (r *EmployeeRepository) GetEmployeesSummary(ctx context.Context) (*domain.EmployeesSummary, error) {
	var summary domain.EmployeesSummary
	var err error
	now := clock.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := timeutil.EndOfDay(startOfMonth.AddDate(0, 1, -1))

	// Query 1: Get employee counts and month-to-date salary/paid
	var basicStats struct {
		TotalEmployees    int64   `json:"total_employees"`
		SalaryMonthToDate float64 `json:"salary_month_to_date"`
		PaidMonthToDate   float64 `json:"paid_month_to_date"`
	}

	err = r.statisticsBuilder.BuildSummaryQuery().Scan(&basicStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get basic employee statistics: %w", err)
	}

	summary.TotalEmployees = basicStats.TotalEmployees
	summary.SalaryMonthToDate = int64(basicStats.SalaryMonthToDate)
	summary.PaidMonthToDate = int64(basicStats.PaidMonthToDate)

	// Query 2: Get employees hired this month
	var hiredStats struct {
		EmployeesHiredThisMonth int64 `json:"employees_hired_this_month"`
	}

	err = r.statisticsBuilder.BuildEmployeesHiredThisMonthQuery(startOfMonth, endOfMonth).Scan(&hiredStats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get employees hired this month: %w", err)
	}

	// Query 3: Get total working employees (employees with active project assignments)
	var workingCount int64
	err = r.statisticsBuilder.BuildWorkingEmployeesCountQuery().Count(&workingCount).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get working employees count: %w", err)
	}

	summary.TotalWorkingEmployees = workingCount
	summary.EmployeesHiredThisMonth = hiredStats.EmployeesHiredThisMonth

	return &summary, nil
}

// GetEmployeesSummaryForCreator gets summary for employees accessible to a specific user
// Includes employees created by the user, shared with the user, or assigned to projects by the user
func (r *EmployeeRepository) GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*domain.EmployeesSummary, error) {
	var summary domain.EmployeesSummary
	var err error
	now := clock.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := timeutil.EndOfDay(startOfMonth.AddDate(0, 1, -1))

	// Query 1: Get employee counts for accessible employees
	var basicStats struct {
		TotalEmployees int64 `json:"total_employees"`
	}

	countQuery := r.statisticsBuilder.BuildSummaryForCreatorCountQuery(createdBy)
	err = countQuery.Count(&basicStats.TotalEmployees).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get basic employee statistics for creator: %w", err)
	}

	summary.TotalEmployees = basicStats.TotalEmployees

	// Query 2: Get accessible employee IDs
	var accessibleEmployeeIDs []uint
	idsQuery := r.statisticsBuilder.BuildGetAccessibleEmployeeIDsQuery(createdBy)
	err = idsQuery.Pluck("employees.id", &accessibleEmployeeIDs).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get accessible employee IDs: %w", err)
	}

	// Query 3: Get month-to-date salary/paid for accessible employees
	if len(accessibleEmployeeIDs) > 0 {
		var salaryStats struct {
			SalaryMonthToDate float64 `json:"salary_month_to_date"`
			PaidMonthToDate   float64 `json:"paid_month_to_date"`
		}

		salaryQuery := r.statisticsBuilder.BuildSummaryForCreatorSalaryQuery(accessibleEmployeeIDs)
		err = salaryQuery.Scan(&salaryStats).Error
		if err != nil {
			return nil, fmt.Errorf("failed to get salary statistics for creator: %w", err)
		}

		summary.SalaryMonthToDate = int64(salaryStats.SalaryMonthToDate)
		summary.PaidMonthToDate = int64(salaryStats.PaidMonthToDate)
	} else {
		summary.SalaryMonthToDate = 0
		summary.PaidMonthToDate = 0
	}

	// Query 4: Get project-related counts for accessible employees
	if len(accessibleEmployeeIDs) > 0 {
		var projectStats struct {
			TotalWorkingEmployees   int64 `json:"total_working_employees"`
			EmployeesHiredThisMonth int64 `json:"employees_hired_this_month"`
		}

		projectStatsQuery := r.statisticsBuilder.BuildSummaryForCreatorProjectStatsQuery(accessibleEmployeeIDs, startOfMonth, endOfMonth)
		err = projectStatsQuery.Scan(&projectStats).Error
		if err != nil {
			return nil, fmt.Errorf("failed to get project statistics for creator: %w", err)
		}

		workingCountQuery := r.statisticsBuilder.BuildSummaryForCreatorWorkingCountQuery(accessibleEmployeeIDs)
		err = workingCountQuery.Count(&projectStats.TotalWorkingEmployees).Error
		if err != nil {
			return nil, fmt.Errorf("failed to get working employees count for creator: %w", err)
		}

		summary.TotalWorkingEmployees = projectStats.TotalWorkingEmployees
		summary.EmployeesHiredThisMonth = projectStats.EmployeesHiredThisMonth
	} else {
		summary.TotalWorkingEmployees = 0
		summary.EmployeesHiredThisMonth = 0
	}

	return &summary, nil
}

func (r *EmployeeRepository) CountUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters domain.EmployeeFilters) (int64, error) {
	var count int64
	query := r.projectQueryBuilder.BuildUnassignedAtDateCountQuery(atDate, filters)

	err := query.Count(&count).Error
	return count, err
}

// GetActiveCountAtDate returns count of active employees at a specific date
func (r *EmployeeRepository) GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error) {
	var count int64
	err := r.queryBuilder.BuildGetActiveCountAtDateQuery(date).Count(&count).Error
	return int(count), err
}

// GetActiveCountAtDateForCreator returns count of employees created by a specific user at a specific date
func (r *EmployeeRepository) GetActiveCountAtDateForCreator(ctx context.Context, date time.Time, createdBy uint) (int, error) {
	var count int64
	err := r.queryBuilder.BuildGetActiveCountAtDateForCreatorQuery(date, createdBy).Count(&count).Error
	return int(count), err
}

// GetRecentEmployees returns employees created within a date range
func (r *EmployeeRepository) GetRecentEmployees(ctx context.Context, startDate, endDate time.Time, limit int) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.statisticsBuilder.BuildRecentEmployeesQuery(startDate, endDate, limit)

	err := query.Find(&employees).Error
	return employees, err
}

// GetRecentEmployeesPaginated retrieves recent employees with pagination support
func (r *EmployeeRepository) GetRecentEmployeesPaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*domain.Employee, error) {
	var employees []*domain.Employee
	query := r.statisticsBuilder.BuildRecentEmployeesPaginatedQuery(startDate, endDate, limit, offset)

	err := query.Find(&employees).Error
	return employees, err
}

// CountRecentEmployees counts employees created within a date range
func (r *EmployeeRepository) CountRecentEmployees(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var count int64
	query := r.statisticsBuilder.BuildCountRecentQuery(startDate, endDate)

	err := query.Count(&count).Error
	return count, err
}
