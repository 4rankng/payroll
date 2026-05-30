package timesheet

import (
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/app/services/reporting"
	"api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// UploadTimesheetEntries uploads and processes a timesheet template Excel file
// @Summary Upload timesheet entries from Excel
// @Description Upload an Excel file in timesheet template format and create timesheet entries
// @Tags timesheets
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Excel file to upload"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/upload-entries-excel [post]
func (h *Handler) UploadTimesheetEntries(c *gin.Context) {
	// Get user context
	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Parse multipart form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Vui lòng tải lên file Excel")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	// Validate file extension
	if !isExcelFile(header.Filename) {
		response.BadRequest(c, "File phải có định dạng .xlsx")
		return
	}

	// Process Excel file
	excelConverter := reporting.NewExcelConverterService()
	excelFile, err := excelConverter.ProcessExcelFile(header)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Không thể đọc file Excel: %v", err))
		return
	}
	defer func() {
		_ = excelFile.Data.Close()
	}()

	// Parse timesheet template
	importData, err := excel.ParseTimesheetTemplateFile(excelFile.Data)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("Không thể phân tích file Excel: %v", err))
		return
	}

	// Validate that we have data to import
	if len(importData.Employees) == 0 {
		response.BadRequest(c, "Không tìm thấy dữ liệu nhân viên trong file Excel")
		return
	}

	// For partners, validate they have access to all employees in the upload
	if userRole == string(domain.RolePartner) {
		// Check if partner has full project access
		canModify, err := h.projectPermissionService.CanUserModifyProject(c.Request.Context(), importData.ProjectID, userID)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToCheckProjectAccessVN)
			return
		}

		if !canModify {
			grantedEmployeeIDs, err := h.employeePermissionService.GetAccessibleEmployeeIDs(c.Request.Context(), userID)
			if err != nil {
				response.InternalServerError(c, constants.MsgFailedToGetAccessibleEmployeesVN)
				return
			}

			accessibleEmployeeIDs := make(map[uint]bool)
			for _, empID := range grantedEmployeeIDs {
				accessibleEmployeeIDs[empID] = true
			}

			for _, employee := range importData.Employees {
				if !accessibleEmployeeIDs[employee.EmployeeID] {
					response.Forbidden(c, fmt.Sprintf(constants.MsgEmployeeAccessDeniedVN, employee.EmployeeID))
					return
				}
			}
		}
	}

	// Convert parsed data to bulk create requests
	var bulkRequests []domainServices.BulkCreateTimesheetEntry
	for _, employee := range importData.Employees {
		for _, entry := range employee.Entries {
			// Create one request per shift type with non-zero hours
			for shiftType, hours := range entry.ShiftHours {
				bulkRequests = append(bulkRequests, domainServices.BulkCreateTimesheetEntry{
					ProjectID:   importData.ProjectID,
					EmployeeID:  employee.EmployeeID,
					Date:        entry.Date,
					HoursWorked: hours,
					HourType:    shiftType,
					DayType:     &entry.DayType,
				})
			}
		}
	}

	// Validate that we have entries to create
	if len(bulkRequests) == 0 {
		response.BadRequest(c, "Không tìm thấy bản ghi chấm công hợp lệ trong file Excel")
		return
	}

	// Create timesheets using bulk create service
	result, err := h.timesheetService.BulkCreateTimesheets(c.Request.Context(), bulkRequests, userID, userRole)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Calculate counts
	createdCount := len(result.CreatedTimesheets)
	errorCount := len(result.FailedEntries)
	totalCount := len(bulkRequests)
	skippedCount := totalCount - createdCount - errorCount

	// Prepare response message
	message := fmt.Sprintf("Đã xử lý file Excel thành công. Tạo mới: %d, Bỏ qua: %d, Lỗi: %d",
		createdCount, skippedCount, errorCount)

	// Map failed entries to DTO
	failedEntries := make([]dto.TimesheetImportFailure, len(result.FailedEntries))
	for i, f := range result.FailedEntries {
		failedEntries[i] = dto.TimesheetImportFailure{Index: f.Index, Error: f.Error}
	}

	response.Success(c, dto.TimesheetImportResult{
		CreatedCount:  createdCount,
		SkippedCount:  skippedCount,
		ErrorCount:    errorCount,
		FailedEntries: failedEntries,
	}, message)
}

// isExcelFile checks if the filename has a valid Excel extension
func isExcelFile(filename string) bool {
	return len(filename) > 5 && filename[len(filename)-5:] == ".xlsx"
}
