package employee

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetEmployeeTimesheet retrieves timesheet entries for an employee
// @Summary Get employee timesheet
// @Description Get timesheet entries for a specific employee with pagination and filtering
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param from_date query string false "Start date for the timesheet range in YYYY-MM-DD format"
// @Param to_date query string false "End date for the timesheet range in YYYY-MM-DD format"
// @Param offset query int false "Number of items to skip (default: 0)" default(0)
// @Param page query int false "Page number, starts from 1 (default: 1)" default(1)
// @Param pageSize query int false "Items per page (default: 100, max: 100)" default(100)
// @Param sortBy query string false "Field to sort by (default: created_at)" default(created_at)
// @Param sortOrder query string false "Sort direction asc or desc (default: desc)" default(desc)
// @Success 200 {object} dto.ListTimesheetsResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/{id}/timesheet [get]
func (h *Handler) GetEmployeeTimesheet(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidEmployeeIDVN)
		return
	}

	// Verify employee exists
	_, err = h.employeeService.GetEmployee(c.Request.Context(), uint(id))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Initialize filters with defaults
	filters := domain.TimesheetFilters{
		EmployeeID: &[]uint{uint(id)}[0],
		Offset:     0,
		Limit:      100,
		SortBy:     "created_at",
		SortOrder:  "desc",
	}

	// Parse query parameters

	// Handle pagination - prefer page/pageSize but fall back to offset
	page := 1
	pageSize := 100

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if pageSizeParam := c.Query("pageSize"); pageSizeParam != "" {
		if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Handle offset parameter (overrides page if provided)
	if offset := c.Query("offset"); offset != "" {
		if o, err := strconv.Atoi(offset); err == nil && o >= 0 {
			filters.Offset = o
			// Recalculate page from offset
			page = (o / pageSize) + 1
		} else {
			// Use page-based offset calculation
			filters.Offset = (page - 1) * pageSize
		}
	} else {
		// Use page-based offset calculation
		filters.Offset = (page - 1) * pageSize
	}

	filters.Limit = pageSize

	// Handle sorting
	if sortBy := c.Query("sortBy"); sortBy != "" {
		// Validate allowed sort fields
		validSortFields := map[string]bool{
			"created_at":     true,
			"updated_at":     true,
			"date":           true,
			"hours_worked":   true,
			"amount":         true,
			"status":         true,
			"payment_status": true,
		}
		if validSortFields[sortBy] {
			filters.SortBy = sortBy
		}
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		if sortOrder == "asc" || sortOrder == "desc" {
			filters.SortOrder = sortOrder
		}
	}

	// Handle project_id filtering
	if projectID := c.Query("project_id"); projectID != "" {
		if projectIDParsed, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			projectIDUint := uint(projectIDParsed)
			filters.ProjectIDs = []uint{projectIDUint}
		} else {
			response.BadRequest(c, constants.MsgInvalidProjectIDVN)
			return
		}
	}

	// Handle status filtering (searches both timesheet_status and payment_status)
	if status := c.Query("status"); status != "" {
		// Create a special marker to indicate combined status search
		filters.TimesheetStatus = []domain.TimesheetStatus{domain.TimesheetStatus(status)}
		filters.PaymentStatus = []domain.PaymentStatus{domain.PaymentStatus(status)}
	}

	// Handle date filtering
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

	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveTimesheetSummaryVN)
		return
	}

	total, err := h.timesheetService.CountTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	// Build response DTOs
	var timesheetResponses []dto.TimesheetWithDetailsResponse
	for _, timesheet := range timesheets {
		var projectName, employeeName string

		if timesheet.Project != nil {
			projectName = timesheet.Project.Name
		}

		if timesheet.Employee != nil {
			employeeName = timesheet.Employee.Fullname
		}

		// Extract hour type and day type from paytype
		var hourType, dayType string
		payTypeParts := strings.Split(timesheet.PayType, ".")
		if len(payTypeParts) >= 3 {
			dayType = payTypeParts[1]  // "Ngày thường"
			hourType = payTypeParts[2] // "Ca ngày"
		}

		// Build the base timesheet response first
		timesheetResponse := dto.TimesheetResponse{
			ID:              timesheet.ID,
			ProjectID:       timesheet.ProjectID,
			EmployeeID:      timesheet.EmployeeID,
			Date:            timesheet.Date.Format("2006-01-02"),
			HoursWorked:     utils.RoundToTwoDecimals(timesheet.HoursWorked),
			PayType:         timesheet.PayType,
			HourType:        hourType,
			DayType:         dayType,
			PayrateID:       timesheet.PayrateID,
			PayRate:         utils.RoundToTwoDecimals(float64(timesheet.PayRate)),
			Amount:          utils.RoundToTwoDecimals(float64(timesheet.Amount)),
			Status:          string(timesheet.Status),
			PaymentStatus:   string(timesheet.PaymentStatus),
			ForcePayroll:    timesheet.ForcePayroll,
			PaidAmount:      timesheet.PaidAmount,
			PaidAt:          timesheet.PaidAt,
			CreatedBy:       timesheet.CreatedBy,
			ApprovedBy:      timesheet.ApprovedBy,
			ApprovedAt:      timesheet.ApprovedAt,
			RejectionReason: timesheet.RejectionReason,
			CreatedAt:       timesheet.CreatedAt,
			UpdatedAt:       timesheet.UpdatedAt,
		}

		// Generate employee code
		employeeCode := ""
		if timesheet.Employee != nil {
			employeeCode = fmt.Sprintf("EMP-%03d", timesheet.Employee.ID)
		}

		// Get submitted by information
		var submittedBy *uint
		var submittedByName *string
		var submittedAt *time.Time

		if timesheet.CreatedBy > 0 {
			submittedBy = &timesheet.CreatedBy
			submittedAt = &timesheet.CreatedAt // Use created_at as submitted_at
			if timesheet.CreatedUser != nil {
				submittedByName = &timesheet.CreatedUser.Fullname
			}
		}

		// Create the detailed response with project/employee names
		timesheetResponses = append(timesheetResponses, dto.TimesheetWithDetailsResponse{
			TimesheetResponse: timesheetResponse,
			ProjectName:       projectName,
			EmployeeName:      employeeName,
			EmployeeCode:      employeeCode,
			SubmittedBy:       submittedBy,
			SubmittedByName:   submittedByName,
			SubmittedAt:       submittedAt,
		})
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(timesheetResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, "No timesheet entries found", pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := "Timesheet entries retrieved successfully"
	if page > 1 {
		message = "Page " + strconv.Itoa(page) + " of timesheet entries retrieved successfully"
	}

	response.SuccessWithPagination(c, timesheetResponses, message, pagination)
}
