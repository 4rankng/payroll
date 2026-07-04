package services

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"api-server/internal/app/utils"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// PayrollReportByProjectService handles payroll report by project domain logic
type PayrollReportByProjectService struct {
	timesheetRepo domain.TimesheetRepository
	projectRepo   domain.ProjectRepository
	employeeRepo  domain.EmployeeRepository
	logger        *slog.Logger
}

// NewPayrollReportByProjectService creates a new service
func NewPayrollReportByProjectService(
	timesheetRepo domain.TimesheetRepository,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
) *PayrollReportByProjectService {
	return &PayrollReportByProjectService{
		timesheetRepo: timesheetRepo,
		projectRepo:   projectRepo,
		employeeRepo:  employeeRepo,
		logger:        observability.GetLogger(),
	}
}

// ProjectReportData represents report data for a single project
type ProjectReportData struct {
	Project          *domain.Project
	SalaryPeriodFrom time.Time
	SalaryPeriodTo   time.Time
	EmployeeData     []*EmployeeReportData
	TotalAmount      int64
	EmployeeCount    int
	TimesheetIDs     []uint
	Timesheets       []*domain.Timesheet
}

// EmployeeReportData represents report data for an employee within a project
type EmployeeReportData struct {
	EmployeeName string
	EmployeeCCCD string
	ProjectName  string
	PaymentDate  *time.Time
	TotalPaid    int64
}

// isProjectEligibleOnDay returns true if a project should be included in a sao ke export
// on a given day-of-month. The rules mirror the primary project filter logic and must be
// applied consistently to both the primary loop and the safety-net loop so that projects
// like salary_period_to=25 (exported early-next-month) are never pulled into a mid-month export.
func isProjectEligibleOnDay(day, salaryPeriodTo int) bool {
	if day >= 24 {
		// Day-26 export: cycles ending on the 20th or earlier (exclude end-of-month/weekly)
		return salaryPeriodTo > 0 && salaryPeriodTo <= 20
	}
	if day <= 10 {
		// Day-2 export: end-of-month/weekly cycles and cycles ending on the 21st or later
		return salaryPeriodTo == 0 || salaryPeriodTo >= 21
	}
	return false
}

// GetProjectsForPayrollReport gets projects based on atDate and their salary periods
func (s *PayrollReportByProjectService) GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*ProjectReportData, error) {
	// Get all active projects
	filters := domain.ProjectFilters{
		ProjectStatus: []domain.ProjectStatus{domain.ProjectStatusRunning},
	}

	projects, err := s.projectRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Payroll report generation started",
		"atDate", atDate.Format("2006-01-02"),
		"day", atDate.Day(),
		"totalProjects", len(projects))

	// Filter projects based on atDate and salary period
	var filteredProjects []*domain.Project
	day := atDate.Day()

	for _, project := range projects {
		if isProjectEligibleOnDay(day, project.SalaryPeriodTo) {
			filteredProjects = append(filteredProjects, project)
		}
	}
	if len(filteredProjects) == 0 {
		s.logger.Warn("No projects selected for payroll report", "day", day)
	}

	// Process each project and get timesheet data
	var reportData []*ProjectReportData

	for _, project := range filteredProjects {
		var fromDate, toDate time.Time
		fromDate, toDate = utils.CalculateDateRange(atDate, project.SalaryPeriodFrom, project.SalaryPeriodTo)

		s.logger.Info("Processing project",
			"projectID", project.ID,
			"projectName", project.Name,
			"salaryPeriodFrom", project.SalaryPeriodFrom,
			"salaryPeriodTo", project.SalaryPeriodTo,
			"calculatedFromDate", fromDate.Format("2006-01-02 15:04:05 -0700"),
			"calculatedToDate", toDate.Format("2006-01-02 15:04:05 -0700"),
			"atDateTZ", atDate.Location().String())

		// Get paid timesheets where revenue_paid = false
		timesheets, err := s.getPaidTimesheets(ctx, project.ID, fromDate, toDate)
		if err != nil {
			s.logger.Error("Failed to get timesheets for project",
				"projectID", project.ID,
				"projectName", project.Name,
				"error", err)
			continue // Skip projects with errors
		}

		s.logger.Info("Found timesheets for project",
			"projectID", project.ID,
			"projectName", project.Name,
			"fromDate", fromDate.Format("2006-01-02"),
			"toDate", toDate.Format("2006-01-02"),
			"timesheetCount", len(timesheets))

		// Skip projects without timesheets
		if len(timesheets) == 0 {
			s.logger.Info("Skipping project with no timesheets",
				"projectID", project.ID,
				"projectName", project.Name)
			continue
		}

		// Group by employee + payment date
		type empDateKey struct {
			employeeID  uint
			paymentDate time.Time
		}
		employeeMap := make(map[empDateKey]*EmployeeReportData)
		totalAmount := int64(0)
		var timesheetIDs []uint

		timesheetsWithoutEmployee := 0
		for _, ts := range timesheets {
			timesheetIDs = append(timesheetIDs, ts.ID)
			if ts.Employee == nil {
				timesheetsWithoutEmployee++
				continue
			}

			paymentDate := time.Time{}
			if ts.PaymentDate != nil {
				paymentDate = *ts.PaymentDate
			}

			key := empDateKey{employeeID: ts.EmployeeID, paymentDate: paymentDate}
			if _, exists := employeeMap[key]; !exists {
				employeeMap[key] = &EmployeeReportData{
					EmployeeName: ts.Employee.FormattedFullname(),
					EmployeeCCCD: ts.Employee.CCCD,
					ProjectName:  project.Name,
					PaymentDate:  ts.PaymentDate,
					TotalPaid:    0,
				}
			}

			employeeMap[key].TotalPaid += ts.PaidAmount
			totalAmount += ts.PaidAmount
		}

		if timesheetsWithoutEmployee > 0 {
			s.logger.Warn("Found timesheets without employee data",
				"projectID", project.ID,
				"count", timesheetsWithoutEmployee)
		}

		// Convert map to slice
		var employeeData []*EmployeeReportData
		for _, data := range employeeMap {
			employeeData = append(employeeData, data)
		}

		sort.Slice(employeeData, func(i, j int) bool {
			// Sort by payment date first (nil dates come first)
			if employeeData[i].PaymentDate == nil && employeeData[j].PaymentDate == nil {
				return employeeData[i].EmployeeName < employeeData[j].EmployeeName
			}
			if employeeData[i].PaymentDate == nil {
				return true
			}
			if employeeData[j].PaymentDate == nil {
				return false
			}
			if !employeeData[i].PaymentDate.Equal(*employeeData[j].PaymentDate) {
				return employeeData[i].PaymentDate.Before(*employeeData[j].PaymentDate)
			}
			return employeeData[i].EmployeeName < employeeData[j].EmployeeName
		})

		s.logger.Info("Completed project aggregation",
			"projectID", project.ID,
			"projectName", project.Name,
			"employeeCount", len(employeeData),
			"totalAmount", totalAmount)

		reportData = append(reportData, &ProjectReportData{
			Project:          project,
			SalaryPeriodFrom: fromDate,
			SalaryPeriodTo:   toDate,
			EmployeeData:     employeeData,
			TotalAmount:      totalAmount,
			EmployeeCount:    len(employeeData),
			TimesheetIDs:     timesheetIDs,
			Timesheets:       timesheets,
		})
	}

	// Safety net: include projects with unsettled paid timesheets that weren't selected
	// by the day-based filter. This catches orphaned timesheets from salary period changes.
	processedProjectIDs := make(map[uint]bool)
	for _, rd := range reportData {
		processedProjectIDs[rd.Project.ID] = true
	}

	for _, project := range projects {
		if processedProjectIDs[project.ID] {
			continue
		}
		// Skip projects that are not eligible on this day — the same rule as the primary filter.
		// Without this guard, projects like salary_period_to=25 (intended for early-next-month export)
		// would bleed into mid-month sao ke files through the safety net.
		if !isProjectEligibleOnDay(day, project.SalaryPeriodTo) {
			continue
		}

		var fromDate, toDate time.Time
		fromDate, toDate = utils.CalculateDateRange(atDate, project.SalaryPeriodFrom, project.SalaryPeriodTo)
		timesheets, err := s.getPaidTimesheetsWithoutRevenue(ctx, project.ID, fromDate, toDate)
		if err != nil {
			s.logger.Warn("Safety net: failed to check project for unsettled timesheets",
				"projectID", project.ID, "error", err)
			continue
		}
		if len(timesheets) == 0 {
			continue
		}

		s.logger.Info("Safety net: including orphaned project with unsettled timesheets",
			"projectID", project.ID,
			"projectName", project.Name,
			"timesheetCount", len(timesheets))

		var minDate, maxDate time.Time
		for _, ts := range timesheets {
			if minDate.IsZero() || ts.Date.Before(minDate) {
				minDate = ts.Date
			}
			if maxDate.IsZero() || ts.Date.After(maxDate) {
				maxDate = ts.Date
			}
		}

		type empDateKey struct {
			employeeID  uint
			paymentDate time.Time
		}
		employeeMap := make(map[empDateKey]*EmployeeReportData)
		totalAmount := int64(0)
		var orphanedTimesheetIDs []uint

		for _, ts := range timesheets {
			orphanedTimesheetIDs = append(orphanedTimesheetIDs, ts.ID)
			if ts.Employee == nil {
				continue
			}
			paymentDate := time.Time{}
			if ts.PaymentDate != nil {
				paymentDate = *ts.PaymentDate
			}
			key := empDateKey{employeeID: ts.EmployeeID, paymentDate: paymentDate}
			if _, exists := employeeMap[key]; !exists {
				employeeMap[key] = &EmployeeReportData{
					EmployeeName: ts.Employee.FormattedFullname(),
					EmployeeCCCD: ts.Employee.CCCD,
					ProjectName:  project.Name,
					PaymentDate:  ts.PaymentDate,
					TotalPaid:    0,
				}
			}
			employeeMap[key].TotalPaid += ts.PaidAmount
			totalAmount += ts.PaidAmount
		}

		var employeeData []*EmployeeReportData
		for _, data := range employeeMap {
			employeeData = append(employeeData, data)
		}

		sort.Slice(employeeData, func(i, j int) bool {
			if employeeData[i].PaymentDate == nil && employeeData[j].PaymentDate == nil {
				return employeeData[i].EmployeeName < employeeData[j].EmployeeName
			}
			if employeeData[i].PaymentDate == nil {
				return true
			}
			if employeeData[j].PaymentDate == nil {
				return false
			}
			if !employeeData[i].PaymentDate.Equal(*employeeData[j].PaymentDate) {
				return employeeData[i].PaymentDate.Before(*employeeData[j].PaymentDate)
			}
			return employeeData[i].EmployeeName < employeeData[j].EmployeeName
		})

		reportData = append(reportData, &ProjectReportData{
			Project:          project,
			SalaryPeriodFrom: minDate,
			SalaryPeriodTo:   maxDate,
			EmployeeData:     employeeData,
			TotalAmount:      totalAmount,
			EmployeeCount:    len(employeeData),
			TimesheetIDs:     orphanedTimesheetIDs,
			Timesheets:       timesheets,
		})
	}

	sort.Slice(reportData, func(i, j int) bool {
		if reportData[i].SalaryPeriodFrom.Equal(reportData[j].SalaryPeriodFrom) {
			return reportData[i].Project.Name < reportData[j].Project.Name
		}
		return reportData[i].SalaryPeriodFrom.Before(reportData[j].SalaryPeriodFrom)
	})

	s.logger.Info("Payroll report generation completed",
		"totalProjectsWithData", len(reportData))

	return reportData, nil
}

// GetPayrollReportForProjects generates payroll report data for specific projects and date range.
// If projectIDs is empty, all active projects are included.
func (s *PayrollReportByProjectService) GetPayrollReportForProjects(ctx context.Context, projectIDs []uint, fromDate, toDate time.Time) ([]*ProjectReportData, error) {
	var projects []*domain.Project
	var err error

	if len(projectIDs) > 0 {
		// Batch-fetch all projects in one query (was: one GetByID per project,
		// an N+1 flagged by the ck:debug 2026-07-04 audit). GetByIDs returns only
		// the projects it finds, so missing IDs are silently skipped — matching
		// the previous warn-and-continue behavior. Iterate the input slice to
		// preserve the caller's requested order (map iteration is unordered).
		projectMap, err := s.projectRepo.GetByIDs(ctx, projectIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to batch-fetch projects: %w", err)
		}
		for _, id := range projectIDs {
			if p, ok := projectMap[id]; ok {
				projects = append(projects, p)
			} else {
				s.logger.Warn("Project not found, skipping", "projectID", id)
			}
		}
	} else {
		projects, err = s.projectRepo.List(ctx, domain.ProjectFilters{
			ProjectStatus: []domain.ProjectStatus{domain.ProjectStatusRunning},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list projects: %w", err)
		}
	}

	var reportData []*ProjectReportData

	for _, project := range projects {
		timesheets, err := s.getPaidTimesheets(ctx, project.ID, fromDate, toDate)
		if err != nil {
			s.logger.Warn("Failed to get timesheets for project", "projectID", project.ID, "error", err)
			continue
		}

		if len(timesheets) == 0 {
			continue
		}

		rd := s.buildProjectReportData(project, timesheets, fromDate, toDate)
		reportData = append(reportData, rd)
	}

	sort.Slice(reportData, func(i, j int) bool {
		return reportData[i].Project.Name < reportData[j].Project.Name
	})

	return reportData, nil
}

// buildProjectReportData aggregates timesheet data into a ProjectReportData
func (s *PayrollReportByProjectService) buildProjectReportData(project *domain.Project, timesheets []*domain.Timesheet, fromDate, toDate time.Time) *ProjectReportData {
	type empDateKey struct {
		employeeID  uint
		paymentDate time.Time
	}
	employeeMap := make(map[empDateKey]*EmployeeReportData)
	totalAmount := int64(0)
	var timesheetIDs []uint

	for _, ts := range timesheets {
		timesheetIDs = append(timesheetIDs, ts.ID)
		if ts.Employee == nil {
			continue
		}

		paymentDate := time.Time{}
		if ts.PaymentDate != nil {
			paymentDate = *ts.PaymentDate
		}

		key := empDateKey{employeeID: ts.EmployeeID, paymentDate: paymentDate}
		if _, exists := employeeMap[key]; !exists {
			employeeMap[key] = &EmployeeReportData{
				EmployeeName: ts.Employee.FormattedFullname(),
				EmployeeCCCD: ts.Employee.CCCD,
				ProjectName:  project.Name,
				PaymentDate:  ts.PaymentDate,
				TotalPaid:    0,
			}
		}
		employeeMap[key].TotalPaid += ts.PaidAmount
		totalAmount += ts.PaidAmount
	}

	var employeeData []*EmployeeReportData
	for _, data := range employeeMap {
		employeeData = append(employeeData, data)
	}

	sort.Slice(employeeData, func(i, j int) bool {
		if employeeData[i].PaymentDate == nil && employeeData[j].PaymentDate == nil {
			return employeeData[i].EmployeeName < employeeData[j].EmployeeName
		}
		if employeeData[i].PaymentDate == nil {
			return true
		}
		if employeeData[j].PaymentDate == nil {
			return false
		}
		if !employeeData[i].PaymentDate.Equal(*employeeData[j].PaymentDate) {
			return employeeData[i].PaymentDate.Before(*employeeData[j].PaymentDate)
		}
		return employeeData[i].EmployeeName < employeeData[j].EmployeeName
	})

	return &ProjectReportData{
		Project:          project,
		SalaryPeriodFrom: fromDate,
		SalaryPeriodTo:   toDate,
		EmployeeData:     employeeData,
		TotalAmount:      totalAmount,
		EmployeeCount:    len(employeeData),
		TimesheetIDs:     timesheetIDs,
		Timesheets:       timesheets,
	}
}

// getPaidTimesheets gets paid-but-unsettled timesheets for a project and date range.
// Excludes timesheets where revenue_paid=true (already settled with client).
func (s *PayrollReportByProjectService) getPaidTimesheets(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	filters := domain.TimesheetFilters{
		ProjectIDs:    []uint{projectID},
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:      &fromDate,
		ToDate:        &toDate,
		Limit:         10000,
		SortBy:        "date",
		SortOrder:     "asc",
	}

	timesheets, err := s.timesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Exclude settled timesheets (revenue_paid=true)
	var result []*domain.Timesheet
	for _, ts := range timesheets {
		if ts.RevenuePaid {
			continue
		}
		result = append(result, ts)
	}

	s.logger.Info("Paid timesheets for export",
		"projectID", projectID,
		"fromDate", fromDate.Format("2006-01-02"),
		"toDate", toDate.Format("2006-01-02"),
		"rawCount", len(timesheets),
		"finalCount", len(result),
	)

	return result, nil
}

// getPaidTimesheetsWithoutRevenue gets timesheets that are paid but revenue not received
func (s *PayrollReportByProjectService) getPaidTimesheetsWithoutRevenue(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	filters := domain.TimesheetFilters{
		ProjectIDs:    []uint{projectID},
		PaymentStatus: []domain.PaymentStatus{domain.PaymentStatusPaid},
		Limit:         10000,
		SortBy:        "date",
		SortOrder:     "asc",
	}

	filters.FromDate = &fromDate
	filters.ToDate = &toDate

	// Get all paid timesheets
	timesheets, err := s.timesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Filter for revenue_paid = false and paid_at IS NOT NULL
	var result []*domain.Timesheet
	var revenuePaidFiltered, missingPaidAtFiltered int
	for _, ts := range timesheets {
		if ts.RevenuePaid {
			revenuePaidFiltered++
			continue
		}
		if ts.PaidAt == nil {
			missingPaidAtFiltered++
			continue
		}
		result = append(result, ts)
	}

	s.logger.Info("Filtered timesheets before report",
		"projectID", projectID,
		"rawCount", len(timesheets),
		"finalCount", len(result),
		"revenuePaidFiltered", revenuePaidFiltered,
		"missingPaidAtFiltered", missingPaidAtFiltered,
	)

	return result, nil
}
