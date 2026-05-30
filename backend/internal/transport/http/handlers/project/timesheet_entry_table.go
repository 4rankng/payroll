package project

import (
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Ensure the timesheet package import is used
var _ = timesheet.TimesheetService{}

// GetTimesheetEntryTable retrieves timesheet entries formatted for entry table display
// @Summary Get project timesheet entry table
// @Description Retrieve timesheet entries grouped by employee with daily breakdowns by day type and hour type
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param fromDate query string true "From date in YYYY-MM-DD format"
// @Param toDate query string true "To date in YYYY-MM-DD format"
// @Param employeeIDs query string false "Comma-separated employee IDs"
// @Success 200 {object} dto.TimesheetEntryTableResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id}/timesheet-entry-table [get]
func (h *Handler) GetTimesheetEntryTable(c *gin.Context) {
	// Validate project ID
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidProjectIDVN)
		return
	}

	// Check if project exists
	_, err = h.projectService.GetProject(c.Request.Context(), uint(projectID))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Parse query parameters
	var params dto.TimesheetEntryTableQueryParams
	if err := c.ShouldBindQuery(&params); err != nil {
		response.BadRequest(c, constants.MsgInvalidRequestFormatVN)
		return
	}

	// Override project ID from path parameter
	params.ProjectID = uint(projectID)

	// Parse dates using time.Local to match the MySQL driver's loc=Local DSN parameter.
	// time.Parse returns UTC midnight, but the MySQL driver with loc=Local converts
	// UTC times to local wall-clock, shifting them by the local UTC offset.
	// This causes the boundary date to be excluded (e.g. fromDate=05-22 skips day 22).
	// Using time.ParseInLocation with time.Local ensures the Go time value matches
	// what the driver will send to MySQL.
	loc := time.Local
	fromDate, err := time.ParseInLocation("2006-01-02", params.FromDate, loc)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
		return
	}

	toDate, err := time.ParseInLocation("2006-01-02", params.ToDate, loc)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
		return
	}

	// Validate date range
	if toDate.Before(fromDate) {
		response.BadRequest(c, constants.MsgInvalidDateRangeVN)
		return
	}

	// Parse employee IDs if provided
	var employeeIDs []uint
	if params.EmployeeIDs != "" {
		employeeIDStrings := strings.Split(params.EmployeeIDs, ",")
		for _, idStr := range employeeIDStrings {
			if idStr = strings.TrimSpace(idStr); idStr != "" {
				if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
					employeeIDs = append(employeeIDs, uint(id))
				}
			}
		}
	}

	// Get timesheet data using the new repository method
	timesheets, err := h.timesheetService.GetEntryTableData(c.Request.Context(), params.ProjectID, employeeIDs, fromDate, toDate)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	// Process and group the data
	responseData := processTimesheetData(timesheets)

	// Return response
	response.Success(c, responseData, "Lấy dữ liệu bảng công thành công")
}

// processTimesheetData groups timesheet data according to the required response format
func processTimesheetData(timesheets []*domain.Timesheet) []dto.EmployeeEntriesData {
	// Create a map to group by employee_id -> date -> day_type -> hour_type -> hours
	employeeData := make(map[uint]map[string]map[string]map[string]float64)

	for _, ts := range timesheets {
		// Parse the paytype to extract day type and hour type
		_, dayType, hourType := parsePaytype(ts.PayType)

		// Initialize employee map if not exists
		if _, exists := employeeData[ts.EmployeeID]; !exists {
			employeeData[ts.EmployeeID] = make(map[string]map[string]map[string]float64)
		}

		// Initialize date map if not exists
		dateStr := ts.Date.Format("2006-01-02")
		if _, exists := employeeData[ts.EmployeeID][dateStr]; !exists {
			employeeData[ts.EmployeeID][dateStr] = make(map[string]map[string]float64)
		}

		// Initialize day type map if not exists
		if _, exists := employeeData[ts.EmployeeID][dateStr][dayType]; !exists {
			employeeData[ts.EmployeeID][dateStr][dayType] = make(map[string]float64)
		}

		// Add hours to the hour type
		employeeData[ts.EmployeeID][dateStr][dayType][hourType] += ts.HoursWorked
	}

	// Convert to response format
	var result []dto.EmployeeEntriesData

	// Sort employee IDs for consistent output
	for employeeID, dateData := range employeeData {
		employeeEntries := dto.EmployeeEntriesData{
			EmployeeID: employeeID,
		}

		// Sort dates for consistent output
		var dates []string
		for date := range dateData {
			dates = append(dates, date)
		}

		for _, dateStr := range dates {
			dayData := dateData[dateStr]

			// Create entry with day types and their hour types
			entry := dto.EntryResponse{
				Date:     dateStr,
				DayTypes: make(map[string]map[string]float64),
			}

			// Add day types and their hour types (only non-zero entries)
			for dayType, hourTypes := range dayData {
				dayTypeMap := make(map[string]float64)
				for hourType, hours := range hourTypes {
					if hours > 0 {
						dayTypeMap[hourType] = hours
					}
				}
				if len(dayTypeMap) > 0 {
					entry.DayTypes[dayType] = dayTypeMap
				}
			}

			employeeEntries.Entries = append(employeeEntries.Entries, entry)
		}

		result = append(result, employeeEntries)
	}

	return result
}

// parsePaytype extracts position, day type, and hour type from paytype string
// This follows the existing pattern in the codebase
func parsePaytype(paytype string) (position, dayType, hourType string) {
	parts := strings.Split(paytype, ".")
	if len(parts) >= 3 {
		position = parts[0]
		dayType = parts[1]
		hourType = strings.Join(parts[2:], ".")
	}
	return position, dayType, hourType
}
