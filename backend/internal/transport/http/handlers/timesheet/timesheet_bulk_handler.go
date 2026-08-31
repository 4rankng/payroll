package timesheet

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"api-server/internal/app/dto"
	constants "api-server/internal/constants"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// BulkCreateTimesheets creates or updates multiple timesheets at once (upsert)
// @Summary Create or update multiple timesheets (upsert)
// @Description Create new timesheets or update existing ones in bulk. If a timesheet with the same project, employee, date, and paytype exists, it will be updated.
// @Tags timesheets
// @Accept json
// @Produce json
// @Param timesheets body dto.BulkCreateTimesheetRequest true "Array of timesheet creation data"
// @Success 201 {object} dto.BulkCreateTimesheetResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets [post]
func (h *Handler) BulkCreateTimesheets(c *gin.Context) {
	logger := observability.GetLogger()
	logger.Info("BulkCreateTimesheets: Starting bulk timesheet upsert")

	// Read raw JSON from request body
	jsonData, err := c.GetRawData()
	if err != nil {
		logger.Error("BulkCreateTimesheets: Failed to read request body",
			"error", err.Error())
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	// Manual JSON unmarshaling without validation
	var requests dto.BulkCreateTimesheetRequest
	if err := json.Unmarshal(jsonData, &requests); err != nil {
		logger.Error("BulkCreateTimesheets: Failed to unmarshal JSON",
			"error", err.Error())
		response.BadRequest(c, constants.MsgInvalidRequestBodyVN)
		return
	}

	// Custom validation that allows 0 for HoursWorked
	if err := h.validateBulkCreateRequest(requests); err != nil {
		logger.Error("BulkCreateTimesheets: Validation failed",
			"error", err.Error())
		response.BadRequest(c, err.Error())
		return
	}

	logger.Info("BulkCreateTimesheets: Successfully parsed request",
		"count", len(requests),
		"request_details", h.formatBulkRequestForLogging(requests))

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		logger.Error("BulkCreateTimesheets: Failed to get user context")
		return
	}

	logger.Info("BulkCreateTimesheets: Got user context", "userID", userID, "userRole", userRole)

	// Convert DTO requests to domain service requests
	domainRequests := make([]domainServices.BulkCreateTimesheetEntry, len(requests))
	for i, req := range requests {
		domainRequests[i] = domainServices.BulkCreateTimesheetEntry{
			ProjectID:   req.ProjectID,
			EmployeeID:  req.EmployeeID,
			Date:        req.Date,
			HoursWorked: req.HoursWorked,
			HourType:    req.HourType,
			DayType:     req.DayType,
		}
		logger.Info("BulkCreateTimesheets: Processing entry",
			"index", i,
			"project_id", req.ProjectID,
			"employee_id", req.EmployeeID,
			"date", req.Date,
			"hours_worked", req.HoursWorked,
			"hour_type", req.HourType,
			"day_type", h.formatDayTypeForLogging(req.DayType))
	}

	logger.Info("BulkCreateTimesheets: Calling domain service", "entries_count", len(domainRequests))

	// Call the domain service to handle bulk upsert with business logic
	importCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 5*time.Minute)
	defer cancel()

	result, err := h.timesheetService.BulkCreateTimesheets(importCtx, domainRequests, userID, userRole)
	if err != nil {
		logger.Error("BulkCreateTimesheets: Service failed",
			"error", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
			"entries_processed", len(domainRequests))
		response.HandleDomainError(c, err)
		return
	}

	logger.Info("BulkCreateTimesheets: Service completed",
		"created_count", len(result.CreatedTimesheets),
		"deleted_count", len(result.DeletedTimesheets),
		"failed_count", len(result.FailedEntries))

	// Convert domain results to DTO responses
	var succeeded []dto.TimesheetResponse
	var deleted []dto.TimesheetResponse
	var failed []dto.BulkCreateError

	// Convert successfully processed timesheets (created or updated)
	for _, timesheet := range result.CreatedTimesheets {
		succeeded = append(succeeded, h.buildTimesheetResponse(timesheet))
	}

	// Convert deleted timesheets
	for _, timesheet := range result.DeletedTimesheets {
		deleted = append(deleted, h.buildTimesheetResponse(timesheet))
	}

	// Convert failed entries
	for _, failure := range result.FailedEntries {
		createReq := dto.CreateTimesheetRequest{
			ProjectID:   failure.Request.ProjectID,
			EmployeeID:  failure.Request.EmployeeID,
			Date:        failure.Request.Date,
			HoursWorked: failure.Request.HoursWorked,
			HourType:    failure.Request.HourType,
			DayType:     failure.Request.DayType,
		}

		// Convert technical error to user-friendly message using centralized translator
		translator := response.GetErrorTranslator()
		cleanedErrorMsg := translator.TranslateError(fmt.Errorf("%s", failure.Error))

		failed = append(failed, dto.BulkCreateError{
			Index:   failure.Index,
			Error:   cleanedErrorMsg,
			Request: createReq,
		})
	}

	bulkResponse := dto.BulkCreateTimesheetResponse{
		Succeeded:    succeeded,
		Deleted:      deleted,
		Failed:       failed,
		TotalSuccess: len(succeeded),
		TotalDeleted: len(deleted),
		TotalFailed:  len(failed),
	}

	totalOperations := len(succeeded) + len(deleted)
	if len(failed) == 0 {
		if len(deleted) > 0 {
			response.SuccessCreated(c, bulkResponse, fmt.Sprintf(constants.MsgBulkTimesheetsCreatedSuccessfullyVN+": %d (đã xóa: %d)", len(succeeded), len(deleted)))
		} else {
			response.SuccessCreated(c, bulkResponse, fmt.Sprintf(constants.MsgBulkTimesheetsCreatedSuccessfullyVN+": %d", len(succeeded)))
		}
	} else if totalOperations > 0 {
		if len(deleted) > 0 {
			response.Success(c, bulkResponse, fmt.Sprintf(constants.MsgBulkTimesheetsCreatedSuccessfullyVN+": %d (đã xóa: %d) với %d lỗi", len(succeeded), len(deleted), len(failed)))
		} else {
			response.Success(c, bulkResponse, fmt.Sprintf(constants.MsgBulkTimesheetsCreatedSuccessfullyVN+": %d với %d lỗi", len(succeeded), len(failed)))
		}
	} else {
		// All failed - return the first error message instead of combining all
		firstError := failed[0].Error
		translator := response.GetErrorTranslator()
		cleanError := translator.CleanMessage(firstError)

		message := fmt.Sprintf(constants.MsgValidationFailedWithDetailsVN, cleanError)
		response.BadRequest(c, message)
	}
}

// BulkApprove approves multiple timesheet entries
func (h *Handler) BulkApprove(c *gin.Context) {
	var req dto.BulkApproveRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	result, err := h.timesheetService.BulkApprove(c.Request.Context(), req.TimesheetIds, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Build response from detailed results
	bulkResponse := dto.BulkOperationResponse{
		Approved: result.Approved,
		Skipped:  result.Skipped,
		Failed:   result.Failed,
		Results:  make([]dto.BulkOperationResult, len(result.Results)),
	}

	// Convert domain results to DTO results
	for i, domainResult := range result.Results {
		bulkResponse.Results[i] = dto.BulkOperationResult{
			ID:     domainResult.ID,
			Status: domainResult.Status,
			Error:  domainResult.Error,
		}
	}

	// Generate appropriate message
	var message string
	if result.Failed > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công, bỏ qua %d bảng đã phê duyệt, %d thất bại", result.Approved, result.Skipped, result.Failed)
	} else if result.Skipped > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công, bỏ qua %d bảng đã phê duyệt", result.Approved, result.Skipped)
	} else {
		message = fmt.Sprintf("Đã phê duyệt thành công %d bảng chấm công", result.Approved)
	}

	response.Success(c, bulkResponse, message)
}

// BulkReject rejects multiple timesheet entries
func (h *Handler) BulkReject(c *gin.Context) {
	var req dto.BulkRejectRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	err := h.timesheetService.BulkReject(c.Request.Context(), req.TimesheetIds, req.RejectionReason, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	bulkResponse := dto.BulkOperationResponse{
		Rejected:          len(req.TimesheetIds),
		Failed:            0,
		NotificationsSent: 0,
		Results:           make([]dto.BulkOperationResult, len(req.TimesheetIds)),
	}

	for i, id := range req.TimesheetIds {
		bulkResponse.Results[i] = dto.BulkOperationResult{
			ID:              id,
			Status:          "rejected",
			RejectionReason: req.RejectionReason,
			Error:           "",
		}
	}

	response.Success(c, bulkResponse, "Bulk rejection completed successfully")
}

// BulkReset resets multiple timesheet entries back to pending approval
func (h *Handler) BulkReset(c *gin.Context) {
	var req dto.BulkResetRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	err := h.timesheetService.BulkReset(c.Request.Context(), req.TimesheetIds, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	bulkResponse := dto.BulkOperationResponse{
		Approved: len(req.TimesheetIds),
		Failed:   0,
		Results:  make([]dto.BulkOperationResult, len(req.TimesheetIds)),
	}

	for i, id := range req.TimesheetIds {
		bulkResponse.Results[i] = dto.BulkOperationResult{
			ID:     id,
			Status: "pending_approval",
			Error:  "",
		}
	}

	response.Success(c, bulkResponse, "Bulk reset completed successfully")
}

// PreviewTimesheets performs dry-run validation for bulk timesheet creation
// @Summary Preview bulk timesheet creation
// @Description Performs all validations for bulk timesheet creation without writing to database
// @Tags timesheets
// @Accept json
// @Produce json
// @Param timesheets body dto.BulkCreateTimesheetRequest true "Array of timesheet creation data"
// @Success 200 {object} dto.TimesheetPreviewResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/preview [post]
func (h *Handler) PreviewTimesheets(c *gin.Context) {
	logger := observability.GetLogger()
	logger.Info("PreviewTimesheets: Starting timesheet preview validation")

	var requests dto.BulkCreateTimesheetRequest
	if !helpers.BindJSON(c, &requests) {
		logger.Error("PreviewTimesheets: Failed to bind JSON")
		return
	}

	logger.Info("PreviewTimesheets: Successfully parsed request", "count", len(requests))

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		logger.Error("PreviewTimesheets: Failed to get user context")
		return
	}

	logger.Info("PreviewTimesheets: Got user context", "userID", userID, "userRole", userRole)

	// Convert DTO requests to domain service requests
	domainRequests := make([]domainServices.BulkCreateTimesheetEntry, len(requests))
	for i, req := range requests {
		domainRequests[i] = domainServices.BulkCreateTimesheetEntry{
			ProjectID:   req.ProjectID,
			EmployeeID:  req.EmployeeID,
			Date:        req.Date,
			HoursWorked: req.HoursWorked,
			HourType:    req.HourType,
			DayType:     req.DayType,
		}
	}

	// Call the domain service to handle preview validation
	result, err := h.timesheetService.PreviewTimesheets(c.Request.Context(), domainRequests, userID, userRole)
	if err != nil {
		logger.Error("PreviewTimesheets: Service failed", "error", err.Error())
		response.HandleDomainError(c, err)
		return
	}

	// Build preview response — map domain errors to DTO
	dtoErrors := make([]dto.TimesheetPreviewError, len(result.Errors))
	for i, e := range result.Errors {
		dtoErrors[i] = dto.TimesheetPreviewError{
			EmployeeID: e.EmployeeID,
			Date:       e.Date,
			Message:    e.Message,
		}
	}
	previewResponse := dto.TimesheetPreviewResponse{
		Errors: dtoErrors,
	}

	response.Success(c, previewResponse, "Preview completed")
}

// BulkApproveByProject approves all timesheets for a specific project
func (h *Handler) BulkApproveByProject(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	var req dto.BulkApproveRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Get all pending timesheets for the project
	timesheets, err := h.timesheetService.GetTimesheetsByProject(c.Request.Context(), projectID, time.Time{}, time.Time{})
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Filter to only pending approval timesheets
	var timesheetIDs []uint
	for _, ts := range timesheets {
		if ts.Status == "pending_approval" || ts.Status == "draft" {
			timesheetIDs = append(timesheetIDs, ts.ID)
		}
	}

	if len(timesheetIDs) == 0 {
		response.Success(c, dto.BulkOperationResponse{
			Approved: 0,
			Skipped:  0,
			Failed:   0,
			Results:  []dto.BulkOperationResult{},
		}, "No timesheets found for approval")
		return
	}

	result, err := h.timesheetService.BulkApprove(c.Request.Context(), timesheetIDs, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Build response from detailed results
	bulkResponse := dto.BulkOperationResponse{
		Approved: result.Approved,
		Skipped:  result.Skipped,
		Failed:   result.Failed,
		Results:  make([]dto.BulkOperationResult, len(result.Results)),
	}

	// Convert domain results to DTO results
	for i, domainResult := range result.Results {
		bulkResponse.Results[i] = dto.BulkOperationResult{
			ID:     domainResult.ID,
			Status: domainResult.Status,
			Error:  domainResult.Error,
		}
	}

	// Generate appropriate message
	var message string
	if result.Failed > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công cho dự án %d, bỏ qua %d bảng đã phê duyệt, %d thất bại", result.Approved, projectID, result.Skipped, result.Failed)
	} else if result.Skipped > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công cho dự án %d, bỏ qua %d bảng đã phê duyệt", result.Approved, projectID, result.Skipped)
	} else {
		message = fmt.Sprintf("Đã phê duyệt thành công %d bảng chấm công cho dự án %d", result.Approved, projectID)
	}

	response.Success(c, bulkResponse, message)
}

// ApproveAllTimesheets approves all timesheets with status pending approval
// @Summary Approve all pending timesheets
// @Description Approve all timesheets that have status pending_approval
// @Tags timesheets
// @Accept json
// @Produce json
// @Success 200 {object} dto.BulkOperationResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/approve-all [post]
func (h *Handler) ApproveAllTimesheets(c *gin.Context) {
	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// Bind optional filter body (empty body = approve all, backward compatible)
	var req dto.ApproveAllRequest
	_ = c.ShouldBindJSON(&req)

	// Build filters to get all pending approval timesheets
	filters := domain.TimesheetFilters{
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusPendingApproval},
		Limit:           10000,
		SkipRelations:   true,
	}

	// Apply optional filters from request body
	if len(req.ProjectIDs) > 0 {
		filters.ProjectIDs = req.ProjectIDs
	}
	if len(req.EmployeeIDs) > 0 {
		filters.EmployeeIDs = req.EmployeeIDs
	}
	if req.FromDate != "" {
		if t, err := time.Parse("2006-01-02", req.FromDate); err == nil {
			filters.FromDate = &t
		}
	}
	if req.ToDate != "" {
		if t, err := time.Parse("2006-01-02", req.ToDate); err == nil {
			filters.ToDate = &t
		}
	}

	// Apply partner role filtering - only show timesheets for employees created by the current user
	if userRole == string(domain.RolePartner) {
		filters.EmployeeCreatedBy = &userID
	}

	// Get all pending timesheets matching filters
	timesheets, err := h.timesheetService.ListTimesheets(c.Request.Context(), filters)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Extract timesheet IDs
	var timesheetIDs []uint
	for _, ts := range timesheets {
		timesheetIDs = append(timesheetIDs, ts.ID)
	}

	if len(timesheetIDs) == 0 {
		response.Success(c, dto.BulkOperationResponse{
			Approved: 0,
			Skipped:  0,
			Failed:   0,
			Results:  []dto.BulkOperationResult{},
		}, "Không có bảng chấm công nào đang chờ phê duyệt")
		return
	}

	// Approve all pending timesheets
	result, err := h.timesheetService.BulkApprove(c.Request.Context(), timesheetIDs, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Build response from detailed results
	bulkResponse := dto.BulkOperationResponse{
		Approved: result.Approved,
		Skipped:  result.Skipped,
		Failed:   result.Failed,
		Results:  make([]dto.BulkOperationResult, len(result.Results)),
	}

	// Convert domain results to DTO results
	for i, domainResult := range result.Results {
		bulkResponse.Results[i] = dto.BulkOperationResult{
			ID:     domainResult.ID,
			Status: domainResult.Status,
			Error:  domainResult.Error,
		}
	}

	// Generate appropriate message
	var message string
	if result.Failed > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công, bỏ qua %d bảng đã phê duyệt, %d thất bại", result.Approved, result.Skipped, result.Failed)
	} else if result.Skipped > 0 {
		message = fmt.Sprintf("Đã phê duyệt %d bảng chấm công, bỏ qua %d bảng đã phê duyệt", result.Approved, result.Skipped)
	} else {
		message = fmt.Sprintf("Đã phê duyệt thành công tất cả %d bảng chấm công", result.Approved)
	}

	response.Success(c, bulkResponse, message)
}

// ResetAllTimesheets resets all approved timesheets back to pending approval
// (admin "Huỷ duyệt hết" — inverse of approve-all). Paid timesheets are never
// touched (excluded by the repository predicate).
// @Summary Cancel approval for all approved timesheets
// @Description Resets every approved, unpaid timesheet back to pending_approval
// @Tags timesheets
// @Accept json
// @Produce json
// @Success 200 {object} dto.BulkOperationResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/reset-all [post]
func (h *Handler) ResetAllTimesheets(c *gin.Context) {
	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	reset, err := h.timesheetService.ResetAllTimesheets(c.Request.Context(), userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if reset == 0 {
		response.Success(c, dto.BulkOperationResponse{
			Approved: 0,
			Skipped:  0,
			Failed:   0,
			Results:  []dto.BulkOperationResult{},
		}, "Không có bảng chấm công nào có thể hủy duyệt")
		return
	}

	response.Success(c, dto.BulkOperationResponse{
		Approved: int(reset),
		Skipped:  0,
		Failed:   0,
		Results:  []dto.BulkOperationResult{},
	}, fmt.Sprintf("Đã hủy duyệt thành công tất cả %d bảng chấm công", reset))
}

// formatBulkRequestForLogging creates a log-friendly representation of bulk requests
func (h *Handler) formatBulkRequestForLogging(requests dto.BulkCreateTimesheetRequest) []map[string]interface{} {
	formatted := make([]map[string]interface{}, len(requests))
	for i, req := range requests {
		formatted[i] = map[string]interface{}{
			"project_id":   req.ProjectID,
			"employee_id":  req.EmployeeID,
			"date":         req.Date,
			"hours_worked": req.HoursWorked,
			"hour_type":    req.HourType,
			"day_type":     h.formatDayTypeForLogging(req.DayType),
		}
	}
	return formatted
}

// formatDayTypeForLogging safely formats day type for logging
func (h *Handler) formatDayTypeForLogging(dayType *string) string {
	if dayType == nil {
		return "nil"
	}
	return *dayType
}

// validateBulkCreateRequest performs custom validation that allows 0 for HoursWorked
func (h *Handler) validateBulkCreateRequest(requests dto.BulkCreateTimesheetRequest) error {
	for i, req := range requests {
		// Validate ProjectID
		if req.ProjectID == 0 {
			return fmt.Errorf("projectID is required at index %d", i)
		}

		// Validate EmployeeID
		if req.EmployeeID == 0 {
			return fmt.Errorf("employeeID is required at index %d", i)
		}

		// Validate Date
		if req.Date == "" {
			return fmt.Errorf("date is required at index %d", i)
		}

		// Validate HoursWorked - allow 0 but prevent negative values
		if req.HoursWorked < 0 {
			return fmt.Errorf("hoursWorked cannot be negative at index %d", i)
		}

		// Validate HourType
		if req.HourType == "" {
			return fmt.Errorf("hourType is required at index %d", i)
		}
	}
	return nil
}
