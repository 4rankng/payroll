package services

import (
	"context"
	"log/slog"
	"sort"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
)

// FlexPayReconciliationService handles reconciliation report generation for FlexPay (advance payments)
type FlexPayReconciliationService struct {
	requestRepo  domain.AdvancePaymentRequestRepository
	advPayRepo   domain.AdvancePaymentRepository
	projectRepo  domain.ProjectRepository
	employeeRepo domain.EmployeeRepository
	logger       *slog.Logger
}

// NewFlexPayReconciliationService creates a new service
func NewFlexPayReconciliationService(
	requestRepo domain.AdvancePaymentRequestRepository,
	advPayRepo domain.AdvancePaymentRepository,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
) *FlexPayReconciliationService {
	return &FlexPayReconciliationService{
		requestRepo:  requestRepo,
		advPayRepo:   advPayRepo,
		projectRepo:  projectRepo,
		employeeRepo: employeeRepo,
		logger:       observability.GetLogger(),
	}
}

// ProjectFlexPayReportData represents report data for a single project
type ProjectFlexPayReportData struct {
	Project              *domain.Project
	SalaryPeriodFrom     time.Time
	SalaryPeriodTo       time.Time
	EmployeeData         []*EmployeeFlexPayReportData
	TotalAmount          int64
	TotalRequestedAmount int64
	EmployeeCount        int
	RequestIDs           []uint
}

// EmployeeFlexPayReportData represents report data for an employee within a project
type EmployeeFlexPayReportData struct {
	EmployeeName   string
	EmployeeCCCD   string
	ProjectName    string
	PaymentDate    *time.Time
	TotalPaid      int64
	TotalRequested int64
	RequestIDs     []uint
}

// GetCompletedRequestsByMonth gets completed requests for groups by employee + project
func (s *FlexPayReconciliationService) GetCompletedRequestsByMonth(ctx context.Context, forMonth string) ([]*ProjectFlexPayReportData, error) {
	statusCompleted := string(domain.AdvancePaymentStatusCompleted)
	filters := domain.AdvancePaymentRequestFilters{
		Status:   &statusCompleted,
		ForMonth: &forMonth,
		Limit:    10000,
	}
	requests, _, err := s.requestRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	parsedMonth, err := time.Parse("2006-01", forMonth)
	if err != nil {
		return nil, domain.NewValidationError("Invalid month format")
	}

	s.logger.Info("Processing completed requests for FlexPay reconciliation",
		"forMonth", forMonth,
		"totalRequests", len(requests))

	// Group by employee + project + payment date
	type empProjectDateKey struct {
		employeeID uint
		projectID  uint
		paidAt     time.Time
	}

	projectMap := make(map[uint]*domain.Project)
	employeeMap := make(map[empProjectDateKey]*EmployeeFlexPayReportData)
	projectTotals := make(map[uint]int64)
	projectRequestedTotals := make(map[uint]int64)
	projectRequestIDs := make(map[uint][]uint)

	for _, req := range requests {
		// Get or cache project data
		if _, exists := projectMap[req.ProjectID]; !exists {
			project, err := s.projectRepo.GetByID(ctx, req.ProjectID)
			if err != nil {
				s.logger.Warn("Failed to get project", "project_id", req.ProjectID, "error", err)
				continue
			}
			projectMap[req.ProjectID] = project
		}

		paidAt := time.Time{}
		if req.PaidAt != nil {
			paidAt = *req.PaidAt
		}

		key := empProjectDateKey{employeeID: req.EmployeeID, projectID: req.ProjectID, paidAt: paidAt}

		// Create or update employee report data
		if _, exists := employeeMap[key]; !exists {
			employeeMap[key] = &EmployeeFlexPayReportData{
				EmployeeName:   req.Employee.FormattedFullname(),
				EmployeeCCCD:   req.Employee.CCCD,
				ProjectName:    req.Project.Name,
				PaymentDate:    req.PaidAt,
				TotalPaid:      0,
				TotalRequested: 0,
				RequestIDs:     []uint{uint(req.ID)},
			}
		} else {
			employeeMap[key].RequestIDs = append(employeeMap[key].RequestIDs, uint(req.ID))
		}

		employeeMap[key].TotalPaid += int64(req.NetAmount)
		employeeMap[key].TotalRequested += int64(req.RequestAmount)
		projectTotals[req.ProjectID] += int64(req.NetAmount)
		projectRequestedTotals[req.ProjectID] += int64(req.RequestAmount)
		projectRequestIDs[req.ProjectID] = append(projectRequestIDs[req.ProjectID], uint(req.ID))
	}

	// Calculate salary period (first day of month to last day of month)
	salaryPeriodFrom := time.Date(parsedMonth.Year(), parsedMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	salaryPeriodTo := time.Date(parsedMonth.Year(), parsedMonth.Month()+1, 0, 23, 59, 59, 0, time.UTC)

	// Convert map to ProjectFlexPayReportData
	var reportData []*ProjectFlexPayReportData
	for _, project := range projectMap {
		// Get all employee data for this project
		var employeeData []*EmployeeFlexPayReportData
		for key, empData := range employeeMap {
			if key.projectID == project.ID {
				employeeData = append(employeeData, empData)
			}
		}

		// Sort employee data by name, then by payment date
		sort.Slice(employeeData, func(i, j int) bool {
			if employeeData[i].EmployeeName != employeeData[j].EmployeeName {
				return employeeData[i].EmployeeName < employeeData[j].EmployeeName
			}
			if employeeData[i].PaymentDate == nil && employeeData[j].PaymentDate == nil {
				return employeeData[i].EmployeeCCCD < employeeData[j].EmployeeCCCD
			}
			if employeeData[i].PaymentDate == nil {
				return true // nil dates first
			}
			if employeeData[j].PaymentDate == nil {
				return false
			}
			return employeeData[i].PaymentDate.Before(*employeeData[j].PaymentDate)
		})

		reportData = append(reportData, &ProjectFlexPayReportData{
			Project:              project,
			SalaryPeriodFrom:     salaryPeriodFrom,
			SalaryPeriodTo:       salaryPeriodTo,
			EmployeeData:         employeeData,
			TotalAmount:          projectTotals[project.ID],
			TotalRequestedAmount: projectRequestedTotals[project.ID],
			EmployeeCount:        len(employeeData),
			RequestIDs:           projectRequestIDs[project.ID],
		})
	}

	// Sort by project name
	sort.Slice(reportData, func(i, j int) bool {
		return reportData[i].Project.Name < reportData[j].Project.Name
	})

	s.logger.Info("FlexPay reconciliation completed",
		"forMonth", forMonth,
		"totalProjects", len(reportData))

	return reportData, nil
}

// CancelAllPendingRequests cancels all pending (PENDING/APPROVED) requests for a given month
func (s *FlexPayReconciliationService) CancelAllPendingRequests(ctx context.Context, forMonth string) (int64, []uint64, error) {
	statusPending := string(domain.AdvancePaymentStatusPending)
	statusApproved := string(domain.AdvancePaymentStatusApproved)

	pendingFilters := domain.AdvancePaymentRequestFilters{
		Status: &statusPending,
		Limit:  10000,
	}
	pendingReqs, _, err := s.requestRepo.List(ctx, pendingFilters)
	if err != nil {
		return 0, nil, err
	}

	approvedFilters := domain.AdvancePaymentRequestFilters{
		Status: &statusApproved,
		Limit:  10000,
	}
	approvedReqs, _, err := s.requestRepo.List(ctx, approvedFilters)
	if err != nil {
		return 0, nil, err
	}

	allPendingReqs := append(pendingReqs, approvedReqs...)
	if len(allPendingReqs) == 0 {
		s.logger.Info("No pending requests to cancel", "for_month", forMonth)
		return 0, nil, nil
	}

	var requestIDs []uint64
	for _, req := range allPendingReqs {
		requestIDs = append(requestIDs, uint64(req.ID))
	}

	if err := s.requestRepo.BatchUpdateStatus(ctx, requestIDs, domain.AdvancePaymentStatusCancelled, "", nil); err != nil {
		s.logger.Error("Failed to batch cancel pending requests", "error", err)
		return 0, nil, err
	}

	s.logger.Info("Cancelled all pending requests before reconciliation export",
		"for_month", forMonth,
		"count", len(allPendingReqs),
		"request_ids", requestIDs)

	return int64(len(allPendingReqs)), requestIDs, nil
}

// FlexPayReconciliationSummary represents summary information for the reconciliation report
type FlexPayReconciliationSummary struct {
	TotalAmount    int64 `json:"total_amount"`
	ProjectCount   int   `json:"project_count"`
	RequestCount   int   `json:"request_count"`
	CancelledCount int   `json:"cancelled_count"`
}
