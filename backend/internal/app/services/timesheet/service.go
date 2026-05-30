package timesheet

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/constants"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"gorm.io/gorm"
)

// TimesheetService demonstrates the new orchestration-only pattern
type TimesheetService struct {
	// Repositories for data access
	timesheetRepo       domain.TimesheetRepository
	employeeRepo        domain.EmployeeRepository
	projectRepo         domain.ProjectRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	payrateRepo         domain.PayrateRepository

	// Domain services for business logic
	validationService          *domainServices.TimesheetValidationService
	calculationService         *domainServices.PayrollCalculationService
	assignmentService          *domainServices.EmployeeAssignmentService
	paytypeConstructionService *domainServices.PaytypeConstructionService
	timesheetDomainService     *domainServices.TimesheetDomainService

	// Bulk operations service
	bulkOperations *BulkOperations

	// Infrastructure services
	transactionManager *infrastructure.TransactionManager
	events             domain.EventBus
	cache              infraports.CachePort
	logger             *slog.Logger
}

// NewTimesheetService creates a new timesheet service (orchestration only)
func NewTimesheetService(
	timesheetRepo domain.TimesheetRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	payrateRepo domain.PayrateRepository,
	transactionManager *infrastructure.TransactionManager,
	cache infraports.CachePort,
	events domain.EventBus,
) *TimesheetService {
	// Initialize domain services
	validationService := domainServices.NewTimesheetValidationService(timesheetRepo, projectEmployeeRepo, payrateRepo, employeeRepo, projectRepo, cache)
	calculationService := domainServices.NewPayrollCalculationService(timesheetRepo, payrateRepo, employeeRepo)
	assignmentService := domainServices.NewEmployeeAssignmentService(projectEmployeeRepo, employeeRepo, projectRepo, timesheetRepo)
	paytypeConstructionService := domainServices.NewPaytypeConstructionService(projectEmployeeRepo)
	timesheetDomainService := domainServices.NewTimesheetDomainService(timesheetRepo, projectEmployeeRepo, payrateRepo, employeeRepo, projectRepo, validationService, calculationService, paytypeConstructionService)

	// Initialize bulk operations with transaction support
	bulkOperations := NewBulkOperations(timesheetRepo, transactionManager)

	return &TimesheetService{
		timesheetRepo:              timesheetRepo,
		employeeRepo:               employeeRepo,
		projectRepo:                projectRepo,
		projectEmployeeRepo:        projectEmployeeRepo,
		payrateRepo:                payrateRepo,
		validationService:          validationService,
		calculationService:         calculationService,
		assignmentService:          assignmentService,
		paytypeConstructionService: paytypeConstructionService,
		timesheetDomainService:     timesheetDomainService,
		bulkOperations:             bulkOperations,
		transactionManager:         transactionManager,
		events:                     events,
		cache:                      cache,
		logger:                     observability.GetLogger(),
	}
}

// ApproveTimesheet orchestrates timesheet approval
func (s *TimesheetService) ApproveTimesheet(ctx context.Context, timesheetID uint, approvedBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Approve timesheet using domain service (includes validation)
		return s.timesheetDomainService.ApproveTimesheet(ctx, timesheetID, approvedBy)
	})

	if err != nil {
		return err
	}

	// Get timesheet to publish event (cache will be invalidated asynchronously by event handler)
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err == nil {
		var employeeName, projectName string
		if timesheet.Employee != nil {
			employeeName = timesheet.Employee.Fullname
		}
		if timesheet.Project != nil {
			projectName = timesheet.Project.Name
		}
		event := domain.NewTimesheetApprovedEvent(ctx, timesheet.ID, timesheet.EmployeeID, timesheet.ProjectID, approvedBy, employeeName, projectName)
		if publishErr := s.events.Publish(ctx, event); publishErr != nil {
			s.logger.Warn("Failed to publish TimesheetApprovedEvent", "timesheet_id", timesheet.ID, "error", publishErr)
		}
	}

	return nil
}

// RejectTimesheet orchestrates timesheet rejection
func (s *TimesheetService) RejectTimesheet(ctx context.Context, timesheetID uint, rejectionReason string, rejectedBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Reject timesheet using domain service (includes validation)
		return s.timesheetDomainService.RejectTimesheet(ctx, timesheetID, rejectedBy, rejectionReason)
	})

	if err != nil {
		return err
	}

	// Get timesheet to publish event (cache will be invalidated asynchronously by event handler)
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err == nil {
		var employeeName, projectName string
		if timesheet.Employee != nil {
			employeeName = timesheet.Employee.Fullname
		}
		if timesheet.Project != nil {
			projectName = timesheet.Project.Name
		}
		event := domain.NewTimesheetRejectedEvent(ctx, timesheet.ID, timesheet.EmployeeID, timesheet.ProjectID, rejectedBy, employeeName, projectName)
		if publishErr := s.events.Publish(ctx, event); publishErr != nil {
			s.logger.Warn("Failed to publish TimesheetRejectedEvent", "timesheet_id", timesheet.ID, "error", publishErr)
		}
	}

	return nil
}

// GetTimesheet orchestrates single timesheet retrieval (simple delegation)
func (s *TimesheetService) GetTimesheet(ctx context.Context, timesheetID uint) (*domain.Timesheet, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.GetByID(ctx, timesheetID)
}

// GetTimesheetsByEmployee orchestrates data retrieval (simple delegation)
func (s *TimesheetService) GetTimesheetsByEmployee(ctx context.Context, employeeID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.GetByEmployeeAndPeriod(ctx, employeeID, fromDate, toDate)
}

// GetEntryTableData orchestrates entry table data retrieval (simple delegation)
func (s *TimesheetService) GetEntryTableData(ctx context.Context, projectID uint, employeeIDs []uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.GetEntryTableData(ctx, projectID, employeeIDs, fromDate, toDate)
}

// GetPayrollSummary orchestrates payroll summary generation
func (s *TimesheetService) GetPayrollSummary(ctx context.Context, fromDate, toDate time.Time) (*domain.PayrollSummary, error) {
	// 1. Validate payroll period using domain service
	if err := s.calculationService.ValidatePayrollPeriod(ctx, fromDate, toDate); err != nil {
		return nil, err
	}

	// 2. Generate summary using domain service
	return s.calculationService.GetPayrollSummary(ctx, fromDate, toDate)
}

// BulkApproveTimesheets orchestrates bulk operations
func (s *TimesheetService) BulkApproveTimesheets(ctx context.Context, timesheetIDs []uint, approvedBy uint) (*domain.BulkOperationResult, error) {
	result, err := s.bulkOperations.BulkApprove(ctx, timesheetIDs, approvedBy, "")
	if err != nil {
		return nil, err
	}

	// Invalidate timesheet caches synchronously so subsequent reads see fresh data
	_ = s.cache.InvalidatePattern(ctx, "timesheets:list:*")
	_ = s.cache.InvalidatePattern(ctx, "timesheets:summary:*")

	// Publish bulk approved event for other subscribers
	event := domain.NewTimesheetBulkApprovedEvent(ctx, len(timesheetIDs), nil, approvedBy)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TimesheetBulkApprovedEvent", "count", len(timesheetIDs), "error", err)
	}

	return result, nil
}

// GetEmployeeTimesheetSummary orchestrates employee timesheet summary retrieval
func (s *TimesheetService) GetEmployeeTimesheetSummary(ctx context.Context, employeeID uint, filters domain.TimesheetFilters) (*domain.EmployeeTimesheetSummary, error) {
	// Get employee details first
	employee, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// Get timesheets for the employee with filters
	filters.EmployeeID = &employeeID
	timesheets, err := s.timesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Calculate summary data
	summary := &domain.EmployeeTimesheetSummary{
		EmployeeID:   employeeID,
		EmployeeName: employee.Fullname,
		TotalHours:   make(map[string]float64),
		TotalAmount:  0,
	}

	var totalHours float64
	var totalWorkingDays int
	var lastDate *time.Time

	for _, timesheet := range timesheets {
		// Aggregate by hour type (extracted from paytype)
		_, _, hourType := s.ParsePaytype(timesheet.PayType)
		summary.TotalHours[hourType] += timesheet.HoursWorked
		totalHours += timesheet.HoursWorked
		summary.TotalAmount += timesheet.Amount

		// Count entries by status
		switch timesheet.Status {
		case domain.TimesheetStatusPendingApproval:
			summary.PendingEntries++
		case domain.TimesheetStatusApproved:
			summary.ApprovedEntries++
		case domain.TimesheetStatusRejected:
			summary.RejectedEntries++
		}
		summary.TotalEntries++

		// Track latest date
		if lastDate == nil || timesheet.Date.After(*lastDate) {
			lastDate = &timesheet.Date
		}
		totalWorkingDays++
	}

	// Calculate averages
	if totalWorkingDays > 0 {
		summary.AverageHoursPerDay = totalHours / float64(totalWorkingDays)
		summary.WorkingDays = totalWorkingDays
	}

	// Format last entry date
	if lastDate != nil {
		formatted := lastDate.Format("2006-01-02")
		summary.LastEntryDate = &formatted
	}

	return summary, nil
}

// ListTimesheets orchestrates timesheet listing with pagination
func (s *TimesheetService) ListTimesheets(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error) {
	// Apply a short-lived microcache for high-traffic list endpoints.
	// Only cache when a sensible limit is provided to avoid unbounded payloads.
	if filters.Limit > 0 {
		cacheKey := s.generateListCacheKey(filters)

		var cached []*domain.Timesheet
		if err := s.cache.Get(ctx, cacheKey, &cached); err == nil && len(cached) > 0 {
			return cached, nil
		}

		result, err := s.timesheetRepo.List(ctx, filters)
		if err != nil {
			return nil, err
		}

		if len(result) > 0 {
			_ = s.cache.Set(ctx, cacheKey, result, constants.TimesheetListCacheTTL)
		}

		return result, nil
	}

	// Fallback: no limit provided, delegate directly.
	return s.timesheetRepo.List(ctx, filters)
}

// CountTimesheets orchestrates timesheet counting
func (s *TimesheetService) CountTimesheets(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.Count(ctx, filters)
}

// AddTimesheetToNextPayroll marks a timesheet to be included in the next payroll run
// by setting its payment status to pending (with validation: must be approved and not paid)
func (s *TimesheetService) AddTimesheetToNextPayroll(ctx context.Context, timesheetID uint, requestedBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1) Load timesheet
		ts, err := s.timesheetRepo.GetByID(ctx, timesheetID)
		if err != nil {
			return err
		}

		// 2) Reject only if already paid (failed/cancelled can be retried)
		if ts.PaymentStatus == domain.PaymentStatusPaid {
			return domain.NewValidationError(constants.MsgCannotEditPaidTimesheetVN)
		}

		// 3) Force-approve if not approved (admin override)
		if !ts.IsApproved() {
			if err := s.timesheetRepo.Approve(ctx, timesheetID, requestedBy); err != nil {
				return err
			}
		}

		// 4) Ensure payment status is pending for inclusion in next payroll
		if ts.PaymentStatus != domain.PaymentStatusPending {
			update := domain.PaymentStatusUpdate{
				TimesheetID:   timesheetID,
				PaymentStatus: domain.PaymentStatusPending,
			}
			if err := s.timesheetDomainService.UpdatePaymentStatus(ctx, []domain.PaymentStatusUpdate{update}); err != nil {
				return err
			}
		}

		// 5) Mark for forced payroll inclusion until paid
		return s.timesheetRepo.SetForcePayroll(ctx, timesheetID, true)
	})

	if err != nil {
		return err
	}

	// Publish TimesheetUpdated event for payment status changes
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err == nil {
		event := domain.NewTimesheetUpdatedEvent(ctx, timesheet, nil)
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish TimesheetUpdated event for payment status change",
				"timesheetID", timesheet.ID,
				"error", err)
		}
	}

	return nil
}

// UpdateTimesheet orchestrates timesheet updates
func (s *TimesheetService) UpdateTimesheet(ctx context.Context, timesheetID uint, updates *domain.Timesheet, updatedBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Check if timesheet is paid (must be first validation)
		if err := s.validationService.ValidateTimesheetNotPaid(ctx, timesheetID); err != nil {
			return err
		}

		// 2. Validate update using domain service
		if err := s.validationService.ValidateTimesheet(ctx, updates); err != nil {
			return err
		}

		// 3. Calculate amount using domain service if hours changed
		if updates.HoursWorked > 0 {
			amount, err := s.calculationService.CalculateTimesheetAmount(ctx, updates)
			if err != nil {
				return err
			}
			updates.Amount = amount
		}

		// 4. Set audit fields
		updates.CreatedBy = updatedBy

		// 5. Update timesheet using repository
		updates.ID = timesheetID
		return s.timesheetRepo.Update(ctx, updates)
	})

	if err != nil {
		return err
	}

	// Get updated timesheet to publish event
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err == nil {
		// Publish domain event (cache will be invalidated asynchronously by event handler)
		event := domain.NewTimesheetUpdatedEvent(ctx, timesheet, nil)
		if publishErr := s.events.Publish(ctx, event); publishErr != nil {
			s.logger.Warn("Failed to publish TimesheetUpdatedEvent", "timesheet_id", timesheet.ID, "error", publishErr)
		}
	}

	return nil
}

// CreateTimesheet orchestrates timesheet creation with day type and hour type
func (s *TimesheetService) CreateTimesheet(ctx context.Context, timesheet *domain.Timesheet, hourType string, dayType string, createdBy uint, userRole string) (*domain.Timesheet, error) {
	var result *domain.Timesheet

	// Orchestrate the operation within a transaction
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Construct paytype using domain service
		payType, err := s.paytypeConstructionService.ConstructPaytype(ctx, timesheet.ProjectID, timesheet.EmployeeID, timesheet.Date, hourType, dayType)
		if err != nil {
			return err
		}
		timesheet.PayType = payType

		// 2. Validate timesheet using domain service
		if err := s.validationService.ValidateTimesheet(ctx, timesheet); err != nil {
			return err
		}

		// 3. Calculate amount using domain service
		amount, err := s.calculationService.CalculateTimesheetAmount(ctx, timesheet)
		if err != nil {
			return err
		}
		timesheet.Amount = amount

		// 4. Set audit fields and status using domain service
		timesheet.CreatedBy = createdBy
		if err := s.timesheetDomainService.SetInitialTimesheetStatus(ctx, timesheet, createdBy, userRole); err != nil {
			return err
		}

		// 5. Create timesheet using repository
		return s.timesheetRepo.Create(ctx, timesheet)
	})

	if err != nil {
		return nil, err
	}

	// Publish domain event
	event := domain.NewTimesheetCreatedEvent(ctx, timesheet)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TimesheetCreatedEvent", "timesheet_id", result.ID, "error", err)
	}

	result = timesheet
	return result, nil
}

// BulkApprove orchestrates bulk approval of timesheets
func (s *TimesheetService) BulkApprove(ctx context.Context, timesheetIDs []uint, approvedBy uint) (*domain.BulkOperationResult, error) {
	return s.BulkApproveTimesheets(ctx, timesheetIDs, approvedBy)
}

// BulkReject orchestrates bulk rejection of timesheets
func (s *TimesheetService) BulkReject(ctx context.Context, timesheetIDs []uint, rejectionReason string, rejectedBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, timesheetID := range timesheetIDs {
			// 1. Validate each timesheet using domain service
			if err := s.validationService.CanRejectTimesheet(ctx, timesheetID); err != nil {
				return err
			}

			// 2. Update timesheet status using repository
			if err := s.timesheetRepo.Reject(ctx, timesheetID, rejectionReason); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Publish bulk rejected event after transaction
	event := domain.NewTimesheetBulkRejectedEvent(ctx, len(timesheetIDs), nil, rejectedBy)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TimesheetBulkRejectedEvent", "count", len(timesheetIDs), "error", err)
	}

	return nil
}

// BulkReset orchestrates bulk reset of timesheets back to pending approval
func (s *TimesheetService) BulkReset(ctx context.Context, timesheetIDs []uint, resetBy uint) error {
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		for _, timesheetID := range timesheetIDs {
			// Reset timesheet using domain service (includes validation)
			if err := s.timesheetDomainService.ResetTimesheet(ctx, timesheetID); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Publish bulk reset event after transaction
	event := domain.NewTimesheetBulkResetEvent(ctx, len(timesheetIDs), nil, resetBy)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TimesheetBulkResetEvent", "count", len(timesheetIDs), "error", err)
	}

	return nil
}

// GetTimesheetsByProject orchestrates timesheet retrieval by project
func (s *TimesheetService) GetTimesheetsByProject(ctx context.Context, projectID uint, fromDate, toDate time.Time) ([]*domain.Timesheet, error) {
	// Simple delegation to repository - no business logic needed
	filters := domain.TimesheetFilters{
		ProjectIDs: []uint{projectID},
		FromDate:   &fromDate,
		ToDate:     &toDate,
	}
	return s.timesheetRepo.List(ctx, filters)
}

// DeleteTimesheet orchestrates timesheet deletion
func (s *TimesheetService) DeleteTimesheet(ctx context.Context, timesheetID uint, deletedBy uint) error {
	// Get timesheet before deletion for event
	timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
	if err != nil {
		return err
	}

	err = s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Check if timesheet is paid (must be first validation)
		if err := s.validationService.ValidateTimesheetNotPaid(ctx, timesheetID); err != nil {
			return err
		}

		// 2. Delete timesheet using repository
		return s.timesheetRepo.Delete(ctx, timesheetID)
	})

	if err != nil {
		return err
	}

	// Publish domain event
	event := domain.NewTimesheetDeletedEvent(ctx, timesheet)
	if err := s.events.Publish(ctx, event); err != nil {
		s.logger.Warn("Failed to publish TimesheetDeletedEvent", "timesheet_id", timesheetID, "error", err)
	}

	return err
}

// ParsePaytype orchestrates paytype parsing (delegates to validation service)
func (s *TimesheetService) ParsePaytype(paytype string) (position, dayType, hourType string) {
	// Parse the paytype format: {position}.{dayType}.{hourType}
	// All values are customer-entered, no hardcoded defaults
	parts := strings.Split(paytype, ".")

	switch len(parts) {
	case 3:
		position = parts[0]
		dayType = parts[1]
		hourType = parts[2]
	case 2:
		position = ""
		dayType = parts[0]
		hourType = parts[1]
	case 1:
		position = ""
		dayType = ""
		hourType = parts[0]
	default:
		position = ""
		dayType = ""
		hourType = ""
	}

	return position, dayType, hourType
}

// GetSummaryStats orchestrates summary statistics retrieval with caching
func (s *TimesheetService) GetSummaryStats(ctx context.Context, filters domain.TimesheetFilters) (*domain.TimesheetSummaryStats, error) {
	// Generate cache key from filters
	cacheKey := s.generateSummaryStatsCacheKey(filters)

	// Try to get from cache
	var stats domain.TimesheetSummaryStats
	err := s.cache.Get(ctx, cacheKey, &stats)
	if err == nil {
		// Cache hit
		return &stats, nil
	}

	// Cache miss - fetch from database
	result, err := s.timesheetRepo.GetSummaryStats(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Cache the result for 5 minutes
	_ = s.cache.Set(ctx, cacheKey, result, constants.TimesheetSummaryCacheTTL)

	return result, nil
}

// generateSummaryStatsCacheKey creates a deterministic cache key from filters
func (s *TimesheetService) generateSummaryStatsCacheKey(filters domain.TimesheetFilters) string {
	params := []string{"summary_stats"}

	// Include relevant filter values in cache key
	if len(filters.ProjectIDs) > 0 {
		projectIDs := append([]uint(nil), filters.ProjectIDs...)
		sort.Slice(projectIDs, func(i, j int) bool { return projectIDs[i] < projectIDs[j] })
		projectIDsStr := make([]string, len(projectIDs))
		for i, id := range projectIDs {
			projectIDsStr[i] = fmt.Sprintf("%d", id)
		}
		params = append(params, "projects", strings.Join(projectIDsStr, ","))
	}
	if filters.EmployeeID != nil {
		params = append(params, "employee", fmt.Sprintf("%d", *filters.EmployeeID))
	}
	if len(filters.TimesheetStatus) > 0 {
		statusStr := make([]string, len(filters.TimesheetStatus))
		for i, status := range filters.TimesheetStatus {
			statusStr[i] = string(status)
		}
		sort.Strings(statusStr)
		params = append(params, "status", strings.Join(statusStr, ","))
	}
	if len(filters.PaymentStatus) > 0 {
		paymentStatusStr := make([]string, len(filters.PaymentStatus))
		for i, status := range filters.PaymentStatus {
			paymentStatusStr[i] = string(status)
		}
		sort.Strings(paymentStatusStr)
		params = append(params, "payment", strings.Join(paymentStatusStr, ","))
	}
	if len(filters.PayType) > 0 {
		payTypes := append([]string(nil), filters.PayType...)
		sort.Strings(payTypes)
		params = append(params, "paytypes", strings.Join(payTypes, ","))
	}
	if filters.PayrateID != nil {
		params = append(params, "payrate", fmt.Sprintf("%d", *filters.PayrateID))
	}
	if filters.CreatedBy != nil {
		params = append(params, "created_by", fmt.Sprintf("%d", *filters.CreatedBy))
	}
	if filters.ApprovedBy != nil {
		params = append(params, "approved_by", fmt.Sprintf("%d", *filters.ApprovedBy))
	}
	if filters.EmployeeCreatedBy != nil {
		params = append(params, "employee_created_by", fmt.Sprintf("%d", *filters.EmployeeCreatedBy))
	}
	if filters.EmployeeAssignedByPartner != nil {
		params = append(params, "employee_assigned_by_partner", fmt.Sprintf("%d", *filters.EmployeeAssignedByPartner))
	}
	if filters.Date != nil {
		params = append(params, "date", filters.Date.Format("2006-01-02"))
	}
	if filters.FromDate != nil {
		params = append(params, "from", filters.FromDate.Format("2006-01-02"))
	}
	if filters.ToDate != nil {
		params = append(params, "to", filters.ToDate.Format("2006-01-02"))
	}
	if filters.PendingApproval {
		params = append(params, "pending_approval", "true")
	}
	if filters.ForcePayroll != nil {
		params = append(params, "force_payroll", fmt.Sprintf("%t", *filters.ForcePayroll))
	}
	if filters.AllowedEdit != nil {
		params = append(params, "allowed_edit", fmt.Sprintf("%t", *filters.AllowedEdit))
	}
	if filters.RequestEditID != nil {
		params = append(params, "request_edit_id", fmt.Sprintf("%d", *filters.RequestEditID))
	}
	if filters.HasRequestEdit != nil {
		params = append(params, "has_request_edit", fmt.Sprintf("%t", *filters.HasRequestEdit))
	}
	if filters.Limit > 0 {
		params = append(params, "limit", fmt.Sprintf("%d", filters.Limit))
	}
	if filters.Offset > 0 {
		params = append(params, "offset", fmt.Sprintf("%d", filters.Offset))
	}
	if filters.SortBy != "" {
		params = append(params, "sort_by", filters.SortBy)
	}
	if filters.SortOrder != "" {
		params = append(params, "sort_order", filters.SortOrder)
	}

	// Generate cache key: timesheets:summary:{param1}:{param2}:...
	key := "timesheets:summary"
	for _, param := range params {
		key = fmt.Sprintf("%s:%s", key, param)
	}
	return key
}

// generateListCacheKey reuses the summary filters but with a different prefix
// to provide microcaching for list endpoints.
func (s *TimesheetService) generateListCacheKey(filters domain.TimesheetFilters) string {
	summaryKey := s.generateSummaryStatsCacheKey(filters)
	// summaryKey always starts with "timesheets:summary"
	return strings.Replace(summaryKey, "timesheets:summary", "timesheets:list", 1)
}

// BulkCreateTimesheets orchestrates bulk timesheet creation
func (s *TimesheetService) BulkCreateTimesheets(ctx context.Context, requests []domainServices.BulkCreateTimesheetEntry, createdBy uint, userRole string) (*domainServices.BulkCreateTimesheetResult, error) {
	var result *domainServices.BulkCreateTimesheetResult

	// Orchestrate the operation within a transaction
	err := s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// Process requests through domain service (handles all business logic including status setting)
		domainResult, err := s.timesheetDomainService.BulkCreateTimesheets(ctx, requests, createdBy, userRole)
		if err != nil {
			return err
		}

		result = domainResult
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Publish bulk created event after transaction
	if len(result.CreatedTimesheets) > 0 {
		event := domain.NewTimesheetBulkCreatedEvent(ctx, len(result.CreatedTimesheets), nil, createdBy)
		if err := s.events.Publish(ctx, event); err != nil {
			s.logger.Warn("Failed to publish TimesheetBulkCreatedEvent", "count", len(result.CreatedTimesheets), "error", err)
		}
	}

	return result, nil
}

// PreviewTimesheets performs dry-run validation for bulk timesheet creation
func (s *TimesheetService) PreviewTimesheets(ctx context.Context, requests []domainServices.BulkCreateTimesheetEntry, createdBy uint, userRole string) (*domainServices.PreviewTimesheetResult, error) {
	// Call domain service for validation (no transaction needed since no DB operations)
	result, err := s.timesheetDomainService.PreviewTimesheets(ctx, requests, createdBy, userRole)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// RecalculateTimesheetsForAssignment recalculates editable timesheets after assignment update
// Only recalculates timesheets within the assignment's temporal window:
//   - On or after the assignment's StartDate
//   - On or before the assignment's LastDate (if LastDate is not NULL)
//
// This prevents touching timesheets from previous or future assignments
// Uses batch updates for performance and better error handling
func (s *TimesheetService) RecalculateTimesheetsForAssignment(ctx context.Context, assignment *domain.ProjectEmployee, updatedBy uint) error {
	return s.transactionManager.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get editable timesheets within the assignment's temporal window
		// FromDate: Only timesheets on or after the assignment start date
		// ToDate: Only timesheets on or before the assignment end date (if it exists)
		// This ensures we don't touch timesheets from before OR after the assignment period
		filters := domain.TimesheetFilters{
			ProjectIDs:      []uint{assignment.ProjectID},
			EmployeeID:      &assignment.EmployeeID,
			FromDate:        &assignment.StartDate,
			TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusPendingApproval, domain.TimesheetStatusRejected},
			PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPending, domain.PaymentStatusFailed, domain.PaymentStatusCancelled},
		}

		// Cap the upper bound if assignment has an end date
		// This prevents touching timesheets from future assignments
		if assignment.LastDate != nil {
			filters.ToDate = assignment.LastDate
		}

		timesheets, err := s.timesheetRepo.List(ctx, filters)
		if err != nil {
			return fmt.Errorf("failed to list editable timesheets: %w", err)
		}

		if len(timesheets) == 0 {
			return nil // No timesheets to recalculate
		}

		// 2. Recalculate each timesheet using the updated assignment's position
		var timesheetsToUpdate []*domain.Timesheet
		var recalcErrors []error

		for _, timesheet := range timesheets {
			// Extract day type and hour type from existing paytype
			_, dayType, hourType := s.ParsePaytype(timesheet.PayType)

			// Reconstruct paytype with the updated assignment's position
			newPayType, err := s.paytypeConstructionService.ConstructPaytype(
				ctx, assignment.ProjectID, assignment.EmployeeID, timesheet.Date, hourType, dayType,
			)
			if err != nil {
				recalcErrors = append(recalcErrors, fmt.Errorf("timesheet %d: failed to reconstruct paytype: %w", timesheet.ID, err))
				continue
			}

			// Only recalculate if paytype actually changed
			if newPayType == timesheet.PayType {
				continue
			}

			// Update paytype
			timesheet.PayType = newPayType

			// Recalculate amount using domain service
			amount, err := s.calculationService.CalculateTimesheetAmount(ctx, timesheet)
			if err != nil {
				recalcErrors = append(recalcErrors, fmt.Errorf("timesheet %d: failed to calculate amount: %w", timesheet.ID, err))
				continue
			}

			// Update timesheet with new payrate and amount
			timesheet.Amount = amount
			timesheetsToUpdate = append(timesheetsToUpdate, timesheet)
		}

		// 3. Log any recalculation errors (non-critical)
		if len(recalcErrors) > 0 {
			for _, recalcErr := range recalcErrors {
				s.logger.Warn("Timesheet recalculation error", "error", recalcErr)
			}
		}

		// 4. Batch update all timesheets that changed
		if len(timesheetsToUpdate) > 0 {
			if err := s.timesheetRepo.BulkUpdate(ctx, timesheetsToUpdate); err != nil {
				return fmt.Errorf("failed to bulk update timesheets: %w", err)
			}

			s.logger.Info("Recalculated timesheets for assignment update",
				"assignmentID", assignment.ID,
				"projectID", assignment.ProjectID,
				"employeeID", assignment.EmployeeID,
				"startDate", assignment.StartDate.Format("2006-01-02"),
				"updatedCount", len(timesheetsToUpdate),
				"skippedCount", len(timesheets)-len(timesheetsToUpdate),
			)
		}

		return nil
	})
}

// ListGroupedByEmployee retrieves timesheets grouped by employee with server-side pagination
// This ensures consistent pagination by employee (not by individual timesheet entries)
func (s *TimesheetService) ListGroupedByEmployee(ctx context.Context, filters domain.TimesheetFilters) ([]domain.EmployeeGroupResult, []*domain.Timesheet, int64, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.ListGroupedByEmployee(ctx, filters)
}

// CountDistinctEmployees returns the count of distinct employees matching filters
func (s *TimesheetService) CountDistinctEmployees(ctx context.Context, filters domain.TimesheetFilters) (int64, error) {
	// Simple delegation to repository - no business logic needed
	return s.timesheetRepo.CountDistinctEmployees(ctx, filters)
}
