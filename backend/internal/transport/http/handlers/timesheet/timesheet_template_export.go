package timesheet

import (
	"fmt"
	"net/http"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ExportTimesheetTemplate exports a timesheet template for data entry
// @Summary Export timesheet entries template
// @Description Export an Excel template for timesheet data entry with project and employee information
// @Tags timesheets
// @Accept json
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param request body dto.ExportTimesheetTemplateRequest true "Export template request"
// @Success 200 {file} binary "Excel file"
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/export-entries-template [post]
func (h *Handler) ExportTimesheetTemplate(c *gin.Context) {
	var req dto.ExportTimesheetTemplateRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Get user context
	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Parse dates
	fromDate, err := time.Parse("2006-01-02", req.FromDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
		return
	}

	toDate, err := time.Parse("2006-01-02", req.ToDate)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
		return
	}

	// Validate date range
	if toDate.Before(fromDate) {
		response.BadRequest(c, "toDate must be after or equal to fromDate")
		return
	}

	// Check if date range is too large (e.g., max 31 days)
	daysDiff := toDate.Sub(fromDate).Hours() / 24
	if daysDiff > 31 {
		response.BadRequest(c, "Date range cannot exceed 31 days")
		return
	}

	// Get project
	project, err := h.projectService.GetProject(c.Request.Context(), req.ProjectID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Check access permission - partners can only access their own projects
	if userRole == string(domain.RolePartner) {
		canAccess, err := h.projectPermissionService.CanUserAccessProject(c.Request.Context(), req.ProjectID, userID)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}
		if !canAccess {
			response.Forbidden(c, constants.MsgForbiddenVN)
			return
		}
	}

	// Get employees based on role
	filters := domain.ProjectEmployeeFilters{
		ProjectID:  &req.ProjectID,
		ActiveOnly: true, // Only active employees
		Limit:      1000,
		SortBy:     "employee_name",
		SortOrder:  "asc",
	}

	var employees []*domain.ProjectEmployee

	// For partners, check if they have full project access or limited employee-based access
	if userRole == string(domain.RolePartner) {
		// Check if partner has full project access (creator or explicit project_users access)
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), req.ProjectID, userID)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}

		if canModify {
			employees, err = h.projectEmployeeService.ListAssignments(c.Request.Context(), filters)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToGetProjectEmployeesVN)
				return
			}
		} else {
			allEmployees, err := h.projectEmployeeService.ListAssignments(c.Request.Context(), filters)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToGetProjectEmployeesVN)
				return
			}

			grantedEmployeeIDs, err := h.employeePermissionService.GetAccessibleEmployeeIDs(c.Request.Context(), userID)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToGetAccessibleEmployeesVN)
				return
			}

			// Build a map for O(1) lookup
			accessibleEmployeeIDs := make(map[uint]bool)
			for _, empID := range grantedEmployeeIDs {
				accessibleEmployeeIDs[empID] = true
			}

			// Filter to only accessible employees
			for _, emp := range allEmployees {
				if accessibleEmployeeIDs[emp.EmployeeID] {
					employees = append(employees, emp)
				}
			}
		}
	} else {
		employees, err = h.projectEmployeeService.ListAssignments(c.Request.Context(), filters)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToGetProjectEmployeesVN)
			return
		}
	}

	if len(employees) == 0 {
		response.BadRequest(c, "No active employees found in this project")
		return
	}

	// Get current payrate for the project
	payrate, err := h.payrateService.GetActivePayrateByProjectAndDate(c.Request.Context(), req.ProjectID, h.clock.Now())
	if err != nil {
		// If no payrate found, use default values
		payrate = nil
	}

	// Extract day types and shift types from payrate
	dayTypes, err := excel.ExtractDayTypesFromPayrate(payrate)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToExtractDayTypesVN)
		return
	}

	shiftTypes, err := excel.ExtractShiftTypesFromPayrate(payrate)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToExtractShiftTypesVN)
		return
	}

	// Build template data
	templateData := excel.TimesheetTemplateData{
		Project:    project,
		Employees:  employees,
		FromDate:   fromDate,
		ToDate:     toDate,
		DayTypes:   dayTypes,
		ShiftTypes: shiftTypes,
	}

	// Generate Excel file
	excelService := excel.NewExportService()
	f, err := excelService.GenerateTimesheetTemplate(templateData)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGenerateTemplateVN)
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			// Log the error but don't return it since the response has already been sent
			observability.GetLogger().Error("Error closing Excel file", "error", err)
		}
	}()

	// Save to buffer
	buffer, err := excelService.SaveToBuffer(f)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToSaveExcelFileVN)
		return
	}

	// Generate filename
	filename := fmt.Sprintf("timesheet_template_%s_%s_%s.xlsx",
		project.Code,
		req.FromDate,
		req.ToDate)

	// Return Excel file
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", len(buffer)))
	c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer)
}
