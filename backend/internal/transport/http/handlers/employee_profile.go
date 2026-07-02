package handlers

import (
	"fmt"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/employee"
)

type EmployeeProfileHandler struct {
	EmployeeProfileService *employee.EmployeeProfileService
	EmployeeUserService    *employee.EmployeeUserService
}

func NewEmployeeProfileHandler(
	employeeProfileService *employee.EmployeeProfileService,
	employeeUserService *employee.EmployeeUserService,
) *EmployeeProfileHandler {
	return &EmployeeProfileHandler{
		EmployeeProfileService: employeeProfileService,
		EmployeeUserService:    employeeUserService,
	}
}

// GetMyProfile godoc
// @Summary Get employee's own profile
// @Description Get the authenticated employee's profile information
// @Tags Employee Profile
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=dto.EmployeeProfileResponse}
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Router /api/v1/me [get]
// @Security BearerAuth
func (h *EmployeeProfileHandler) GetMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContext)
		return
	}

	employee, err := h.EmployeeProfileService.GetMyProfile(c.Request.Context(), userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get payment schedule and check-in status from active project assignments (single query)
	paymentSchedule, checkInEnabled, checkInGeofenceRadiusMeters := h.EmployeeProfileService.GetEmployeeScheduleInfo(c.Request.Context(), employee.ID)

	// Convert to response DTO
	resp := &dto.EmployeeProfileResponse{
		ID:                          employee.ID,
		Fullname:                    employee.Fullname,
		Email:                       employee.Email,
		Mobile:                      employee.Mobile,
		Address:                     employee.Address,
		DateOfBirth:                 employee.DateOfBirth,
		BankAccountNumber:           employee.BankAccountNumber,
		BankAccountName:             employee.BankAccountName,
		PaymentSchedule:             paymentSchedule,
		CheckInEnabled:              checkInEnabled,
		CheckInGeofenceRadiusMeters: checkInGeofenceRadiusMeters,
		CreatedAt:                   employee.CreatedAt,
		UpdatedAt:                   employee.UpdatedAt,
	}

	// Add username from User relationship
	if employee.User != nil {
		resp.Username = employee.User.Username
	}

	// Add bank info
	if employee.Bank != nil {
		resp.Bank = &dto.BankInfo{
			ID:         employee.Bank.ID,
			BranchName: employee.Bank.BranchName,
		}
	}

	response.Success(c, resp, "Profile retrieved successfully")
}

// UpdateMyProfile godoc
// @Summary Update employee's own profile
// @Description Update the authenticated employee's profile (name, email, username)
// @Tags Employee Profile
// @Accept json
// @Produce json
// @Param request body dto.UpdateEmployeeProfileRequest true "Update profile request"
// @Success 200 {object} response.Response{data=dto.EmployeeProfileResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 409 {object} response.Response
// @Router /api/v1/me [put]
// @Security BearerAuth
func (h *EmployeeProfileHandler) UpdateMyProfile(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContext)
		return
	}

	var req dto.UpdateEmployeeProfileRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	employee, err := h.EmployeeProfileService.UpdateMyProfile(
		c.Request.Context(),
		userID.(uint),
		req.Fullname,
		req.Email,
		req.Username,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get payment schedule and check-in status from active project assignment (single query)
	paymentSchedule, checkInEnabled, checkInGeofenceRadiusMeters := h.EmployeeProfileService.GetEmployeeScheduleInfo(c.Request.Context(), employee.ID)

	// Convert to response DTO
	resp := &dto.EmployeeProfileResponse{
		ID:                          employee.ID,
		Fullname:                    employee.Fullname,
		Email:                       employee.Email,
		Mobile:                      employee.Mobile,
		Address:                     employee.Address,
		DateOfBirth:                 employee.DateOfBirth,
		BankAccountNumber:           employee.BankAccountNumber,
		BankAccountName:             employee.BankAccountName,
		PaymentSchedule:             paymentSchedule,
		CheckInEnabled:              checkInEnabled,
		CheckInGeofenceRadiusMeters: checkInGeofenceRadiusMeters,
		CreatedAt:                   employee.CreatedAt,
		UpdatedAt:                   employee.UpdatedAt,
	}

	// Add username from User relationship
	if employee.User != nil {
		resp.Username = employee.User.Username
	}

	// Add bank info
	if employee.Bank != nil {
		resp.Bank = &dto.BankInfo{
			ID:         employee.Bank.ID,
			BranchName: employee.Bank.BranchName,
		}
	}

	response.Success(c, resp, "Profile updated successfully")
}

// UpdateMyPassword godoc
// @Summary Update employee's password
// @Description Update the authenticated employee's password
// @Tags Employee Profile
// @Accept json
// @Produce json
// @Param request body dto.UpdateEmployeePasswordRequest true "Update password request"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/me/password [put]
// @Security BearerAuth
func (h *EmployeeProfileHandler) UpdateMyPassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContext)
		return
	}

	var req dto.UpdateEmployeePasswordRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	err := h.EmployeeProfileService.UpdateMyPassword(
		c.Request.Context(),
		userID.(uint),
		req.CurrentPassword,
		req.NewPassword,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, "Password updated successfully")
}

// GetMyTimesheets godoc
// @Summary Get employee's timesheets
// @Description Get the authenticated employee's timesheet entries with filters
// @Tags Employee Profile
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(100)
// @Param fromDate query string false "From date (YYYY-MM-DD)"
// @Param toDate query string false "To date (YYYY-MM-DD)"
// @Param status query string false "Timesheet status filter"
// @Param payment_status query string false "Payment status filter"
// @Success 200 {object} response.Response{data=[]dto.EmployeeTimesheetEntryResponse}
// @Failure 401 {object} response.Response
// @Router /api/v1/me/timesheet [get]
// @Security BearerAuth
func (h *EmployeeProfileHandler) GetMyTimesheets(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContext)
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "100"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	// Build filters
	filters := domain.TimesheetFilters{
		Limit:     pageSize,
		Offset:    offset,
		SortBy:    c.DefaultQuery("sortBy", "date"),
		SortOrder: c.DefaultQuery("sortOrder", "desc"),
	}

	// Parse date filters
	if fromDateStr := c.Query("fromDate"); fromDateStr != "" {
		if fromDate, err := time.Parse("2006-01-02", fromDateStr); err == nil {
			filters.FromDate = &fromDate
		}
	}
	if toDateStr := c.Query("toDate"); toDateStr != "" {
		if toDate, err := time.Parse("2006-01-02", toDateStr); err == nil {
			filters.ToDate = &toDate
		}
	}

	// Parse status filters
	if statusStr := c.Query("status"); statusStr != "" {
		filters.TimesheetStatus = []domain.TimesheetStatus{domain.TimesheetStatus(statusStr)}
	}
	if paymentStatusStr := c.Query("payment_status"); paymentStatusStr != "" {
		filters.PaymentStatus = []domain.PaymentStatus{domain.PaymentStatus(paymentStatusStr)}
	}

	// Get timesheets
	entries, total, err := h.EmployeeProfileService.GetMyTimesheets(c.Request.Context(), userID.(uint), filters)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Convert to response DTOs
	respEntries := make([]dto.EmployeeTimesheetEntryResponse, len(entries))
	for i, entry := range entries {
		respEntry := dto.EmployeeTimesheetEntryResponse{
			ID:            entry.ID,
			Date:          entry.Date,
			HoursWorked:   entry.HoursWorked,
			Amount:        entry.Amount,
			Status:        string(entry.Status),
			PaymentStatus: string(entry.PaymentStatus),
			PaidAmount:    entry.PaidAmount,
			PaymentDate:   entry.PaymentDate,
			ApprovedAt:    entry.ApprovedAt,
			CreatedAt:     entry.CreatedAt,
		}

		if entry.Project != nil {
			respEntry.Project = &dto.EmployeeTimesheetProject{
				ID:         entry.Project.ID,
				Name:       entry.Project.Name,
				Code:       entry.Project.Code,
				ClientName: entry.Project.ClientName,
			}
		}

		if entry.ApprovedBy != nil {
			respEntry.ApprovedBy = &dto.EmployeeTimesheetApprover{
				ID:       entry.ApprovedBy.ID,
				Fullname: entry.ApprovedBy.Fullname,
			}
		}

		respEntries[i] = respEntry
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize
	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	response.SuccessWithPagination(c, respEntries, "Timesheets retrieved successfully", pagination)
}

// GetMySummary godoc
// @Summary Get employee's salary summary
// @Description Get summary statistics for the authenticated employee
// @Tags Employee Profile
// @Accept json
// @Produce json
// @Param weeks query int false "Number of weeks to calculate" default(4)
// @Success 200 {object} response.Response{data=dto.EmployeeSummaryResponse}
// @Failure 401 {object} response.Response
// @Router /api/v1/me/summary [get]
// @Security BearerAuth
func (h *EmployeeProfileHandler) GetMySummary(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContext)
		return
	}

	// Parse weeks parameter
	weeks, _ := strconv.Atoi(c.DefaultQuery("weeks", "4"))
	if weeks < 1 || weeks > 52 {
		weeks = 4
	}

	// Get summary
	summary, err := h.EmployeeProfileService.GetMySummary(c.Request.Context(), userID.(uint), weeks)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Convert to response DTO
	resp := &dto.EmployeePortalSummaryResponse{
		WeeklySalary:            summary.WeeklySalary,
		WeeklyClockedHours:      summary.WeeklyClockedHours,
		WeeklyPaidAmount:        summary.WeeklyPaidAmount,
		TotalApprovedTimesheets: summary.TotalApprovedTimesheets,
		TotalApprovedAmount:     summary.TotalApprovedAmount,
		TotalPaidTimesheets:     summary.TotalPaidTimesheets,
		TotalPaidAmount:         summary.TotalPaidAmount,
		TotalPendingTimesheets:  summary.TotalPendingTimesheets,
		TotalPendingAmount:      summary.TotalPendingAmount,
		LastPaymentDate:         summary.LastPaymentDate,
	}

	if summary.CalculationPeriod != nil {
		resp.CalculationPeriod = &dto.EmployeeCalculationPeriod{
			Weeks:    summary.CalculationPeriod.Weeks,
			FromDate: summary.CalculationPeriod.FromDate,
			ToDate:   summary.CalculationPeriod.ToDate,
		}
	}

	if summary.CurrentProjects != nil {
		resp.CurrentProjects = make([]*dto.EmployeeCurrentProject, len(summary.CurrentProjects))
		for i, proj := range summary.CurrentProjects {
			resp.CurrentProjects[i] = &dto.EmployeeCurrentProject{
				ProjectID:  proj.ProjectID,
				Name:       proj.ProjectName,
				Code:       proj.ProjectCode,
				ClientName: "", // Not available in CurrentProjectAssignment
				Position:   proj.Position,
				StartDate:  proj.StartDate.Format("2006-01-02"),
			}
		}
	}

	response.Success(c, resp, "Summary retrieved successfully")
}

// InitializeEmployeeUsers godoc
// @Summary Initialize user accounts for employees (Admin only)
// @Description Bulk create user accounts for all employees without user_id
// @Tags Employee Management
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=dto.InitializeEmployeeUsersResponse}
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Router /api/v1/employees/init-users [post]
// @Security BearerAuth
func (h *EmployeeProfileHandler) InitializeEmployeeUsers(c *gin.Context) {
	result, err := h.EmployeeUserService.InitializeEmployeeUsers(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToInitEmployeeUsersVN)
		return
	}

	// Convert to response DTO
	resp := &dto.InitializeEmployeeUsersResponse{
		TotalEmployees:  result.TotalEmployees,
		UsersCreated:    result.UsersCreated,
		AlreadyHadUsers: result.AlreadyHadUsers,
		CreatedUsers:    make([]dto.CreatedEmployeeUser, len(result.CreatedUsers)),
	}

	for i, user := range result.CreatedUsers {
		resp.CreatedUsers[i] = dto.CreatedEmployeeUser{
			EmployeeID:      user.EmployeeID,
			EmployeeName:    user.EmployeeName,
			Username:        user.Username,
			DefaultPassword: user.DefaultPassword,
		}
	}

	message := "No employees needed user accounts"
	if result.UsersCreated > 0 {
		message = fmt.Sprintf("Created %d user accounts for employees successfully", result.UsersCreated)
	} else if result.AlreadyHadUsers == result.TotalEmployees {
		message = "All employees already have user accounts"
	}

	response.Success(c, resp, message)
}
