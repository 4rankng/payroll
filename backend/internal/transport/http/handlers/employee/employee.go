package employee

import (
	"context"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/validation"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
)

type Handler struct {
	employeeService        *employee.EmployeeService
	timesheetService       *timesheet.TimesheetService
	projectEmployeeService *project.ProjectEmployeeService
	validator              *validation.RequestValidator
	auditService           interface{} // Will be *infrastructure.AuditService
	clock                  clock.Clock
}

func NewHandlerWithServices(employeeService *employee.EmployeeService, timesheetService *timesheet.TimesheetService, projectEmployeeService *project.ProjectEmployeeService, auditService interface{}, clk clock.Clock) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	return &Handler{
		employeeService:        employeeService,
		timesheetService:       timesheetService,
		projectEmployeeService: projectEmployeeService,
		validator:              validation.NewRequestValidator(),
		auditService:           auditService,
		clock:                  clk,
	}
}

func formatOptionalDate(date *time.Time) *string {
	if date == nil {
		return nil
	}
	formatted := date.Format("2006-01-02")
	return &formatted
}

// buildProjectInfoFromAssignment creates an EmployeeProjectInfo DTO from a domain assignment.
// Shared by buildEmployeeResponse and buildDetailedEmployeeResponse to keep mapping in one place.
func buildProjectInfoFromAssignment(assignment domain.ProjectEmployee) dto.EmployeeProjectInfo {
	projectName := ""
	projectCode := ""
	clientName := ""
	if assignment.Project.ID > 0 {
		projectName = assignment.Project.Name
		projectCode = assignment.Project.Code
		clientName = assignment.Project.ClientName
	}
	return dto.EmployeeProjectInfo{
		ProjectID:              assignment.ProjectID,
		ProjectEmployeeID:      assignment.ID,
		Name:                   projectName,
		Code:                   projectCode,
		ClientName:             clientName,
		Position:               assignment.Position,
		StartDate:              assignment.StartDate.Format("2006-01-02"),
		LastDate:               formatOptionalDate(assignment.LastDate),
		PaymentSchedule:        assignment.PaymentSchedule,
		PendingPaymentSchedule: assignment.PendingPaymentSchedule,
		ScheduleEffectiveFrom:  formatOptionalDate(assignment.ScheduleEffectiveFrom),
		IsFlexible:             assignment.Project.IsFlexible,
		CheckInEnabled:         assignment.CheckInEnabled,
	}
}

// Helper function to convert employee to response DTO
func (h *Handler) buildEmployeeResponse(ctx context.Context, employee *domain.Employee) dto.EmployeeResponse {
	var dateOfBirthStr *string
	if employee.DateOfBirth != nil {
		dobStr := employee.DateOfBirth.Format("2006-01-02")
		dateOfBirthStr = &dobStr
	}

	var bankInfo *dto.EmployeeBankInfo
	if employee.Bank != nil {
		bankInfo = &dto.EmployeeBankInfo{
			ID:         employee.Bank.ID,
			BranchName: employee.Bank.BranchName,
		}
	}

	var username *string
	if employee.User != nil {
		u := employee.User.Username
		username = &u
	}

	response := dto.EmployeeResponse{
		ID:                employee.ID,
		Username:          username,
		Fullname:          employee.Fullname,
		Email:             employee.Email,
		CCCD:              employee.CCCD,
		Address:           employee.Address,
		Mobile:            employee.Mobile,
		Bank:              bankInfo,
		BankAccountNumber: employee.BankAccountNumber,
		BankAccountName:   employee.BankAccountName,
		DateOfBirth:       dateOfBirthStr,
		CreatedBy:         employee.CreatedBy,
		CreatedAt:         employee.CreatedAt,
		UpdatedAt:         employee.UpdatedAt,
		CanDelete:         true,
		CurrentProjects:   []dto.EmployeeProjectInfo{}, // Initialize empty array
	}
	response.BankAccountStatus, response.BankAccountInvalidReason, response.BankAccountValidatedAt = bankAccountStatusFields(employee)

	// Get current project assignments if service is available
	if h.projectEmployeeService != nil {
		filters := domain.ProjectEmployeeFilters{
			EmployeeID: &employee.ID,
			ActiveOnly: true, // Only get active assignments
			SortBy:     "created_at",
			SortOrder:  "desc",
		}

		if assignments, err := h.projectEmployeeService.ListAssignments(ctx, filters); err == nil {
			var currentProjects []dto.EmployeeProjectInfo

			for _, assignment := range assignments {
				// Only include active assignments (LastDate is nil)
				if assignment.LastDate == nil {
					if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
						response.CanDelete = false
					}
					currentProjects = append(currentProjects, buildProjectInfoFromAssignment(*assignment))
				}
			}

			response.CurrentProjects = currentProjects
		}
	}

	return response
}

// Helper function to build comprehensive employee response with related data
func (h *Handler) buildDetailedEmployeeResponse(ctx context.Context, emp *domain.Employee) (*dto.EmployeeDetailedResponse, error) {
	var dateOfBirthStr *string
	if emp.DateOfBirth != nil {
		dobStr := emp.DateOfBirth.Format("2006-01-02")
		dateOfBirthStr = &dobStr
	}

	var bankInfo *dto.EmployeeBankInfo
	if emp.Bank != nil {
		bankInfo = &dto.EmployeeBankInfo{
			ID:         emp.Bank.ID,
			BranchName: emp.Bank.BranchName,
		}
	}

	log := observability.GetLogger()
	var username *string
	if emp.User != nil {
		username = &emp.User.Username
	} else if emp.UserID != nil {
		if user, err := h.employeeService.GetUserByID(ctx, *emp.UserID); err == nil {
			username = &user.Username
		} else {
			log.Error("Failed to fetch user for employee", "employee_id", emp.ID, "error", err)
		}
	}

	resp := &dto.EmployeeDetailedResponse{
		ID:                emp.ID,
		Username:          username,
		Fullname:          emp.Fullname,
		Email:             emp.Email,
		CCCD:              emp.CCCD,
		Address:           emp.Address,
		Mobile:            emp.Mobile,
		Bank:              bankInfo,
		BankAccountNumber: emp.BankAccountNumber,
		BankAccountName:   emp.BankAccountName,
		DateOfBirth:       dateOfBirthStr,
		CreatedBy:         emp.CreatedBy,
		CreatedAt:         emp.CreatedAt,
		UpdatedAt:         emp.UpdatedAt,
	}
	resp.BankAccountStatus, resp.BankAccountInvalidReason, resp.BankAccountValidatedAt = bankAccountStatusFields(emp)

	// Fetch payroll summary, timesheet summary, and project assignments concurrently.
	type payrollResult struct {
		summary *employee.EmployeeSummary
		err     error
	}
	type timesheetResult struct {
		summary *domain.EmployeeTimesheetSummary
		err     error
	}
	type projectResult struct {
		assignments []*domain.ProjectEmployee
		err         error
	}

	payrollCh := make(chan payrollResult, 1)
	timesheetCh := make(chan timesheetResult, 1)
	projectCh := make(chan projectResult, 1)

	go func() {
		s, err := h.employeeService.GetEmployeeSummary(ctx, emp.ID)
		payrollCh <- payrollResult{s, err}
	}()

	go func() {
		if h.timesheetService == nil {
			timesheetCh <- timesheetResult{}
			return
		}
		s, err := h.timesheetService.GetEmployeeTimesheetSummary(ctx, emp.ID, domain.TimesheetFilters{})
		timesheetCh <- timesheetResult{s, err}
	}()

	go func() {
		if h.projectEmployeeService == nil {
			projectCh <- projectResult{}
			return
		}
		assignments, err := h.projectEmployeeService.ListAssignments(ctx, domain.ProjectEmployeeFilters{
			EmployeeID: &emp.ID,
			ActiveOnly: true,
			SortBy:     "created_at",
			SortOrder:  "desc",
		})
		projectCh <- projectResult{assignments, err}
	}()

	// Collect payroll summary
	if pr := <-payrollCh; pr.err == nil && pr.summary != nil {
		lastPaymentDate := ""
		if pr.summary.LastPaymentDate != nil {
			lastPaymentDate = pr.summary.LastPaymentDate.Format("2006-01-02")
		}
		resp.PayrollSummary = &dto.EmployeeSummaryResponse{
			TotalPayrollPayments: pr.summary.TotalPayrollPayments,
			TotalEarningsVND:     pr.summary.TotalEarningsVND,
			LastPaymentDate:      lastPaymentDate,
			AvgWeeklyEarningsVND: pr.summary.AvgWeeklyEarningsVND,
		}
	}

	// Collect timesheet summary
	if tr := <-timesheetCh; tr.err == nil && tr.summary != nil && tr.summary.EmployeeName != "" {
		totalHours := 0.0
		for _, h := range tr.summary.TotalHours {
			totalHours += h
		}
		lastDate := ""
		if tr.summary.LastEntryDate != nil {
			lastDate = *tr.summary.LastEntryDate
		}
		resp.TimesheetSummary = &dto.EmployeeTimesheetSummary{
			TotalTimesheets:    tr.summary.TotalEntries,
			TotalHoursWorked:   utils.RoundToTwoDecimals(totalHours),
			PendingTimesheets:  tr.summary.PendingEntries,
			ApprovedTimesheets: tr.summary.ApprovedEntries,
			RejectedTimesheets: tr.summary.RejectedEntries,
			CurrentWeekHours:   utils.RoundToTwoDecimals(tr.summary.CurrentWeekHours),
			LastTimesheetDate:  lastDate,
		}
	}

	// Collect project assignments
	if ar := <-projectCh; ar.err == nil {
		var currentProjects []dto.EmployeeCurrentProject
		for _, assignment := range ar.assignments {
			if assignment.LastDate != nil {
				continue // skip ended assignments
			}
			p := dto.EmployeeCurrentProject{
				ProjectID:              assignment.ProjectID,
				ProjectEmployeeID:      assignment.ID,
				Position:               assignment.Position,
				StartDate:              assignment.StartDate.Format("2006-01-02"),
				LastDate:               nil,
				PaymentSchedule:        assignment.PaymentSchedule,
				PendingPaymentSchedule: assignment.PendingPaymentSchedule,
				ScheduleEffectiveFrom:  formatOptionalDate(assignment.ScheduleEffectiveFrom),
				IsFlexible:             assignment.Project.IsFlexible,
				CheckInEnabled:         assignment.CheckInEnabled,
			}
			if assignment.Project.ID > 0 {
				p.Name = assignment.Project.Name
				p.Code = assignment.Project.Code
				p.ClientName = assignment.Project.ClientName
			}
			currentProjects = append(currentProjects, p)
		}
		resp.CurrentProjects = currentProjects
	}

	return resp, nil
}

// CreateEmployee creates a new employee
// @Summary Create a new employee
// @Description Create a new employee with the provided information
// @Tags employees
// @Accept json
// @Produce json
// @Param employee body dto.CreateEmployeeRequest true "Employee creation data"
// @Success 201 {object} dto.EmployeeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees [post]
func (h *Handler) CreateEmployee(c *gin.Context) {
	// 1. Validate request format
	var req dto.CreateEmployeeRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// 2. Validate user context
	userID, err := h.validator.ValidateUserContext(c)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 3. Parse optional date field (HTTP concern - date format conversion)
	var dateOfBirth *time.Time
	if req.DateOfBirth != "" {
		if parsed, parseErr := time.Parse("2006-01-02", req.DateOfBirth); parseErr == nil {
			dateOfBirth = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidDateFormatVN)
			return
		}
	}

	// 4. Convert DTO to domain entity (HTTP concern - data transformation)
	employee := &domain.Employee{
		Fullname:          req.Fullname,
		Email:             req.Email,
		CCCD:              req.CCCD,
		Address:           req.Address,
		Mobile:            req.Mobile,
		BankID:            req.BankID,
		BankAccountNumber: req.BankAccountNumber,
		BankAccountName:   req.BankAccountName,
		DateOfBirth:       dateOfBirth,
	}

	// 5. Delegate to service layer (orchestration)
	createdEmployee, err := h.employeeService.CreateEmployee(c.Request.Context(), employee, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 6. Convert domain entity to response DTO (HTTP concern - data transformation)
	employeeResponse := h.buildEmployeeResponse(c.Request.Context(), createdEmployee)
	response.SuccessCreated(c, employeeResponse, constants.MsgEmployeeCreatedSuccessfullyVN)
}

// GetEmployee retrieves an employee by ID with comprehensive information
// @Summary Get an employee by ID with comprehensive information
// @Description Get comprehensive employee information including current project, timesheet summary, payroll summary, and project history
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Success 200 {object} dto.EmployeeDetailedResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id} [get]
func (h *Handler) GetEmployee(c *gin.Context) {
	// 1. Validate ID parameter
	id, err := h.validator.ValidateIDParam(c, "id")
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 2. Delegate to service layer
	employee, err := h.employeeService.GetEmployee(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 3. Build response DTO (HTTP concern - data transformation)
	employeeResponse, err := h.buildDetailedEmployeeResponse(c.Request.Context(), employee)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToBuildComprehensiveResponseVN)
		return
	}

	response.Success(c, employeeResponse, constants.MsgEmployeeRetrievedSuccessfullyVN)
}

// UpdateEmployee updates an existing employee
// @Summary Update an employee
// @Description Update employee information
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param employee body dto.UpdateEmployeeRequest true "Employee update data"
// @Success 200 {object} dto.EmployeeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id} [put]
func (h *Handler) UpdateEmployee(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	var req dto.UpdateEmployeeRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Get existing employee for update (without preloaded relationships)
	employee, err := h.employeeService.GetEmployeeForUpdate(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Update fields
	if req.Fullname != nil {
		employee.Fullname = *req.Fullname
	}
	if req.Email != nil {
		employee.Email = req.Email
	}
	if req.CCCD != nil {
		employee.CCCD = *req.CCCD
	}
	if req.Address != nil {
		employee.Address = *req.Address
	}
	if req.Mobile != nil {
		employee.Mobile = *req.Mobile
	}
	if req.BankID != nil {
		employee.BankID = req.BankID
	}
	if req.BankAccountNumber != nil {
		employee.BankAccountNumber = *req.BankAccountNumber
	}
	if req.BankAccountName != nil {
		employee.BankAccountName = *req.BankAccountName
	}
	if req.DateOfBirth != nil && *req.DateOfBirth != "" {
		if parsed, err := time.Parse("2006-01-02", *req.DateOfBirth); err == nil {
			employee.DateOfBirth = &parsed
		}
	}
	// Status field no longer exists on Employee - using soft-delete pattern instead

	if err := h.employeeService.UpdateEmployee(c.Request.Context(), employee, userID.(uint)); err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to update employee", "employee_id", uint(id), "error", err)
		response.HandleDomainError(c, err)
		return
	}

	// Reload the employee with updated bank relationship
	updatedEmployee, err := h.employeeService.GetEmployee(c.Request.Context(), uint(id))
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to reload updated employee", "employee_id", uint(id), "error", err)
		response.HandleDomainError(c, err)
		return
	}

	employeeResponse := h.buildEmployeeResponse(c.Request.Context(), updatedEmployee)

	response.Success(c, employeeResponse, constants.MsgEmployeeUpdatedSuccessfullyVN)
}

// DeleteEmployee deletes an employee
// @Summary Delete an employee
// @Description Delete an employee by ID
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id} [delete]
func (h *Handler) DeleteEmployee(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Role-based access control for partner users
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		// Get employee to check ownership
		employee, err := h.employeeService.GetEmployee(c.Request.Context(), uint(id))
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
		if employee.CreatedBy != userID {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	result, err := h.employeeService.DeleteEmployee(c.Request.Context(), uint(id), userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if result.RetainedForFinancialHistory {
		response.Success(c, nil, constants.MsgEmployeeUnlinkedFromProjectsVN)
		return
	}
	response.Success(c, nil, constants.MsgEmployeeDeletedWithAssignmentsVN)
}

// GetEmployeeByCCCD retrieves an employee by CCCD
// @Summary Get an employee by CCCD
// @Description Get employee information by CCCD (Citizen ID)
// @Tags employees
// @Accept json
// @Produce json
// @Param cccd path string true "Employee CCCD"
// @Success 200 {object} dto.EmployeeResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/cccd/{cccd} [get]
func (h *Handler) GetEmployeeByCCCD(c *gin.Context) {
	cccd := c.Param("cccd")
	if cccd == "" {
		response.BadRequest(c, constants.MsgCCCDRequiredVN)
		return
	}

	employee, err := h.employeeService.GetEmployeeByCCCD(c.Request.Context(), cccd)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	employeeResponse := h.buildEmployeeResponse(c.Request.Context(), employee)
	response.Success(c, employeeResponse, constants.MsgEmployeeRetrievedSuccessfullyVN)
}

// GetEmployeeTimesheetSummary retrieves timesheet summary for a specific employee
// @Summary Get employee timesheet summary
// @Description Get timesheet summary statistics for a specific employee with optional filtering
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param project_id query int false "Filter by project ID"
// @Param from_date query string false "Start date for the timesheet range in YYYY-MM-DD format"
// @Param to_date query string false "End date for the timesheet range in YYYY-MM-DD format"
// @Success 200 {object} dto.EmployeeTimesheetSummaryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/timesheets/summary [get]
func (h *Handler) GetEmployeeTimesheetSummary(c *gin.Context) {
	// 1. Validate ID parameter
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	// 2. Verify employee exists
	_, err := h.employeeService.GetEmployee(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 4. Initialize filters
	filters := domain.TimesheetFilters{}

	// Parse optional filters
	if projectID := c.Query("project_id"); projectID != "" {
		if pID, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			uid := uint(pID)
			filters.ProjectIDs = []uint{uid}
		} else {
			response.BadRequest(c, constants.MsgInvalidProjectIDVN)
			return
		}
	}

	if fromDate := c.Query("fromDate"); fromDate != "" {
		if parsed, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if parsed, err := time.Parse("2006-01-02", toDate); err == nil {
			filters.ToDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
	}

	// Use injected timesheet service
	if h.timesheetService == nil {
		response.InternalServerError(c, constants.MsgTimesheetServiceNotAvailableVN)
		return
	}

	summary, err := h.timesheetService.GetEmployeeTimesheetSummary(c.Request.Context(), uint(id), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveTimesheetSummaryVN)
		return
	}

	// Convert to response DTO with float formatting
	summaryResponse := dto.EmployeeTimesheetSummaryResponse{
		EmployeeID:         summary.EmployeeID,
		EmployeeName:       summary.EmployeeName,
		TotalHours:         utils.FormatFloatMap(summary.TotalHours),
		TotalAmount:        summary.TotalAmount,
		AverageHoursPerDay: utils.RoundToTwoDecimals(summary.AverageHoursPerDay),
		WorkingDays:        summary.WorkingDays,
		LastEntryDate:      summary.LastEntryDate,
		PendingEntries:     summary.PendingEntries,
	}

	response.Success(c, summaryResponse, constants.MsgEmployeeTimesheetSummaryRetrievedVN)
}

// UpdateEmployeeProject updates a project assignment for an employee
func (h *Handler) UpdateEmployeeProject(c *gin.Context) {
	logger := observability.GetLogger()
	logger.InfoContext(c.Request.Context(), "Starting update employee project assignment request",
		"employee_id", c.Param("id"),
		"method", c.Request.Method,
		"path", c.Request.URL.Path,
	)

	employeeID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		logger.ErrorContext(c.Request.Context(), "User ID not found in context")
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse request body
	var req dto.UpdateAssignmentRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Check if employee exists
	_, err := h.employeeService.GetEmployee(c.Request.Context(), employeeID)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "Employee not found or error accessing employee",
			"employee_id", employeeID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Use injected project employee service
	if h.projectEmployeeService == nil {
		logger.ErrorContext(c.Request.Context(), "Project employee service not available")
		response.InternalServerError(c, constants.MsgProjectEmployeeServiceNotAvailableVN)
		return
	}

	// Find existing assignment
	assignment, err := h.projectEmployeeService.GetAssignmentByProjectAndEmployee(c.Request.Context(), req.ProjectID, employeeID)
	if err != nil {
		logger.ErrorContext(c.Request.Context(), "Assignment not found",
			"employee_id", employeeID,
			"project_id", req.ProjectID,
			"error", err.Error(),
		)
		response.HandleDomainError(c, err)
		return
	}

	// Update assignment fields
	if req.Position != nil {
		assignment.Position = *req.Position
	}
	if req.StartDate != nil && *req.StartDate != "" {
		parsedDate, err := time.Parse("2006-01-02", *req.StartDate)
		if err != nil {
			logger.ErrorContext(c.Request.Context(), "Invalid start date format",
				"start_date", *req.StartDate,
				"error", err,
			)
			response.BadRequest(c, constants.MsgInvalidStartDateFormatVN)
			return
		}
		assignment.StartDate = parsedDate
	}
	if req.EndDate != nil {
		if *req.EndDate == "" {
			assignment.LastDate = nil
		} else {
			parsedEndDate, err := time.Parse("2006-01-02", *req.EndDate)
			if err != nil {
				logger.ErrorContext(c.Request.Context(), "Invalid end date format",
					"end_date", *req.EndDate,
					"error", err,
				)
				response.BadRequest(c, constants.MsgInvalidEndDateFormatVN)
				return
			}
			assignment.LastDate = &parsedEndDate
		}
	}
	if req.PaymentSchedule != nil {
		assignment.PaymentSchedule = *req.PaymentSchedule
	}

	// Update the assignment
	if err := h.projectEmployeeService.UpdateAssignment(c.Request.Context(), assignment, userID.(uint)); err != nil {
		logger.ErrorContext(c.Request.Context(), "Failed to update employee project assignment",
			"assignment_id", assignment.ID,
			"employee_id", employeeID,
			"project_id", req.ProjectID,
			"error", err,
		)
		response.HandleDomainError(c, err)
		return
	}

	// Return updated assignment
	assignmentResponse := dto.ProjectEmployeeAssignmentResponse{
		ID:                     assignment.ID,
		ProjectID:              assignment.ProjectID,
		EmployeeID:             assignment.EmployeeID,
		EmployeeCode:           assignment.EmployeeCode,
		Position:               assignment.Position,
		StartDate:              assignment.StartDate,
		LastDate:               assignment.LastDate,
		PaymentSchedule:        assignment.PaymentSchedule,
		PendingPaymentSchedule: assignment.PendingPaymentSchedule,
		ScheduleEffectiveFrom:  assignment.ScheduleEffectiveFrom,
		CreatedBy:              assignment.CreatedBy,
		CreatedAt:              assignment.CreatedAt,
		UpdatedAt:              assignment.UpdatedAt,
	}

	logger.InfoContext(c.Request.Context(), "Successfully updated employee project assignment",
		"assignment_id", assignment.ID,
		"employee_id", employeeID,
		"project_id", req.ProjectID,
	)

	response.Success(c, assignmentResponse, constants.MsgEmployeeProjectAssignmentUpdatedVN)
}

// GetEmployeeCurrentProjects retrieves current projects with timesheets for an employee
// @Summary Get employee current projects with timesheets
// @Description Get current projects that the employee is assigned to with timesheet entries for the specified date range
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param fromDate query string false "Start date (YYYY-MM-DD format)"
// @Param toDate query string false "End date (YYYY-MM-DD format)"
// @Success 200 {object} dto.EmployeeCurrentProjectsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/current-projects [get]
func (h *Handler) GetEmployeeCurrentProjects(c *gin.Context) {
	// 1. Validate ID parameter
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	// 2. Check if employee exists first
	employee, err := h.employeeService.GetEmployee(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 4. Parse optional date range parameters - support both camelCase (new) and snake_case (legacy)
	var fromDate, toDate *time.Time
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			fromDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
	}

	if toDateStr := c.Query("toDate"); toDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", toDateStr); err == nil {
			toDate = &parsed
		} else {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
	}

	// 6. Check required services are available
	if h.projectEmployeeService == nil {
		response.InternalServerError(c, "Project employee service not available")
		return
	}
	if h.timesheetService == nil {
		response.InternalServerError(c, "Timesheet service not available")
		return
	}

	// 7. Get current project assignments for the employee
	projectFilters := domain.ProjectEmployeeFilters{
		EmployeeID: &employee.ID,
		Status:     "current", // Only current/active projects
		SortBy:     "start_date",
		SortOrder:  "desc",
	}

	assignments, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), projectFilters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveCurrentProjectsVN)
		return
	}

	// 8. Build response with timesheets for each project
	var projects []dto.EmployeeCurrentProjectWithTimesheets

	for _, assignment := range assignments {

		var lastDateStr *string
		if assignment.LastDate != nil {
			dateStr := assignment.LastDate.Format("2006-01-02")
			lastDateStr = &dateStr
		}

		// Get project details from preloaded relationship
		var projectName, projectCode, clientName string
		if assignment.Project.ID != 0 {
			projectName = assignment.Project.Name
			projectCode = assignment.Project.Code
			clientName = assignment.Project.ClientName
		}

		project := dto.EmployeeCurrentProjectWithTimesheets{
			ProjectID:   assignment.ProjectID,
			ProjectName: projectName,
			ProjectCode: projectCode,
			ClientName:  clientName,
			Position:    assignment.Position,
			StartDate:   assignment.StartDate.Format("2006-01-02"),
			LastDate:    lastDateStr,
			Timesheets:  []dto.EmployeeCurrentProjectTimesheet{},
		}

		// 9. Get timesheets for this project within the date range
		timesheetFilters := domain.TimesheetFilters{
			ProjectIDs: []uint{assignment.ProjectID},
			EmployeeID: &employee.ID,
			FromDate:   fromDate,
			ToDate:     toDate,
			SortBy:     "date",
			SortOrder:  "desc",
		}

		timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), timesheetFilters)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToRetrieveTimesheetsVN)
			return
		}

		// 10. Convert timesheets to DTO
		for _, timesheet := range timesheets {
			// Extract hour type and day type from paytype
			var hourType, dayType string
			payTypeParts := strings.Split(timesheet.PayType, ".")
			if len(payTypeParts) >= 3 {
				dayType = payTypeParts[1]  // "Ngày thường"
				hourType = payTypeParts[2] // "Ca ngày"
			}

			// Convert payment status to string pointer
			var paymentStatus *string
			if timesheet.PaymentStatus != "" {
				status := string(timesheet.PaymentStatus)
				paymentStatus = &status
			}

			project.Timesheets = append(project.Timesheets, dto.EmployeeCurrentProjectTimesheet{
				ID:              timesheet.ID,
				Date:            timesheet.Date.Format("2006-01-02"),
				HoursWorked:     utils.RoundToTwoDecimals(timesheet.HoursWorked),
				PayType:         timesheet.PayType,
				HourType:        hourType,
				DayType:         dayType,
				PayRate:         utils.RoundToTwoDecimals(float64(timesheet.PayRate)),
				Amount:          utils.RoundToTwoDecimals(float64(timesheet.Amount)),
				Status:          string(timesheet.Status),
				PaymentStatus:   paymentStatus,
				PaidAmount:      utils.RoundToTwoDecimals(float64(timesheet.PaidAmount)),
				PaidAt:          timesheet.PaidAt,
				ApprovedAt:      timesheet.ApprovedAt,
				RejectionReason: timesheet.RejectionReason,
				CreatedAt:       timesheet.CreatedAt,
			})
		}

		projects = append(projects, project)
	}

	response.Success(c, dto.EmployeeCurrentProjectsResponse(projects), "Employee current projects retrieved successfully")
}

// ChangeEmployeePassword changes the password for an employee's user account
// @Summary Change employee password
// @Description Change the password for an employee's user account (ADMIN can change any, PARTNER can only change for employees they created)
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param password body dto.ChangeEmployeePasswordRequest true "Password change data"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/change-password [put]
func (h *Handler) ChangeEmployeePassword(c *gin.Context) {
	// 1. Validate ID parameter
	id, err := h.validator.ValidateIDParam(c, "id")
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 2. Validate request body
	var req dto.ChangeEmployeePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	// 3. Validate user context
	userID, err := h.validator.ValidateUserContext(c)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 4. Get user role
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == "" {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return
	}

	// 5. Call service to change password
	if err := h.employeeService.ChangeEmployeePassword(c.Request.Context(), id, req.NewPassword, userID, domain.UserRole(userRole)); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// 6. Return success response
	response.Success(c, nil, constants.MsgEmployeePasswordChangedSuccessfullyVN)
}
