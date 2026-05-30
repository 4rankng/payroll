package timesheet

import (
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/timesheet"
)

// EditRequestHandler handles timesheet edit request endpoints
type EditRequestHandler struct {
	editRequestService *timesheet.TimesheetEditRequestService
}

// NewEditRequestHandler creates a new edit request handler
func NewEditRequestHandler(editRequestService *timesheet.TimesheetEditRequestService) *EditRequestHandler {
	return &EditRequestHandler{
		editRequestService: editRequestService,
	}
}

// CreateEditRequest creates a new request to edit an approved timesheet
// @Summary Create timesheet edit request
// @Description Partner creates a request to edit an approved timesheet
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} dto.TimesheetEditRequestResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id}/request-edit [post]
func (h *EditRequestHandler) CreateEditRequest(c *gin.Context) {
	timesheetID, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	editRequest, err := h.editRequestService.CreateEditRequest(c.Request.Context(), timesheetID, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Reload with relationships
	editRequest, err = h.editRequestService.GetEditRequest(c.Request.Context(), editRequest.ID)
	if err != nil {
		response.InternalServerError(c, "Không thể tải thông tin yêu cầu chỉnh sửa")
		return
	}

	editRequestResponse := h.buildEditRequestResponse(editRequest)
	response.Success(c, editRequestResponse, "Yêu cầu chỉnh sửa đã được tạo thành công")
}

// ListEditRequests lists all edit requests
// @Summary List timesheet edit requests
// @Description Get paginated list of timesheet edit requests
// @Tags timesheets
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(20)
// @Param status query string false "Filter by status (pending,approved,rejected)"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/edit-requests [get]
func (h *EditRequestHandler) ListEditRequests(c *gin.Context) {
	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}

	filters := h.parseEditRequestFilters(c)

	// Partner can only see their own requests
	if userRole == string(domain.RolePartner) {
		filters.RequestedBy = &userID
	}

	editRequests, err := h.editRequestService.ListEditRequests(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể tải danh sách yêu cầu chỉnh sửa")
		return
	}

	total, err := h.editRequestService.CountEditRequests(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, "Không thể đếm số lượng yêu cầu chỉnh sửa")
		return
	}

	editRequestResponses := make([]dto.TimesheetEditRequestResponse, len(editRequests))
	for i, req := range editRequests {
		editRequestResponses[i] = h.buildEditRequestResponse(req)
	}

	// Calculate pagination
	page := (filters.Offset / filters.Limit) + 1
	totalPages := (int(total) + filters.Limit - 1) / filters.Limit

	pagination := response.Pagination{
		Page:         page,
		PageSize:     filters.Limit,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	response.SuccessWithPagination(c, editRequestResponses, "Danh sách yêu cầu chỉnh sửa đã được tải thành công", pagination)
}

// GetEditRequest gets a single edit request by ID
// @Summary Get timesheet edit request
// @Description Get details of a timesheet edit request
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Edit Request ID"
// @Success 200 {object} dto.TimesheetEditRequestResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/edit-requests/{id} [get]
func (h *EditRequestHandler) GetEditRequest(c *gin.Context) {
	requestID, ok := h.validateEditRequestID(c)
	if !ok {
		return
	}

	editRequest, err := h.editRequestService.GetEditRequest(c.Request.Context(), requestID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	editRequestResponse := h.buildEditRequestResponse(editRequest)
	response.Success(c, editRequestResponse, "Thông tin yêu cầu chỉnh sửa đã được tải thành công")
}

// ApproveEditRequest approves an edit request
// @Summary Approve timesheet edit request
// @Description Admin approves a timesheet edit request
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Edit Request ID"
// @Success 200 {object} dto.TimesheetEditRequestResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/edit-requests/{id}/approve [put]
func (h *EditRequestHandler) ApproveEditRequest(c *gin.Context) {
	requestID, ok := h.validateEditRequestID(c)
	if !ok {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	err := h.editRequestService.ApproveEditRequest(c.Request.Context(), requestID, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Reload with relationships
	editRequest, err := h.editRequestService.GetEditRequest(c.Request.Context(), requestID)
	if err != nil {
		response.InternalServerError(c, "Không thể tải thông tin yêu cầu chỉnh sửa")
		return
	}

	editRequestResponse := h.buildEditRequestResponse(editRequest)
	response.Success(c, editRequestResponse, "Yêu cầu chỉnh sửa đã được phê duyệt thành công")
}

// RejectEditRequest rejects an edit request
// @Summary Reject timesheet edit request
// @Description Admin rejects a timesheet edit request
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Edit Request ID"
// @Success 200 {object} dto.TimesheetEditRequestResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/edit-requests/{id}/reject [put]
func (h *EditRequestHandler) RejectEditRequest(c *gin.Context) {
	requestID, ok := h.validateEditRequestID(c)
	if !ok {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	err := h.editRequestService.RejectEditRequest(c.Request.Context(), requestID, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Reload with relationships
	editRequest, err := h.editRequestService.GetEditRequest(c.Request.Context(), requestID)
	if err != nil {
		response.InternalServerError(c, "Không thể tải thông tin yêu cầu chỉnh sửa")
		return
	}

	editRequestResponse := h.buildEditRequestResponse(editRequest)
	response.Success(c, editRequestResponse, "Yêu cầu chỉnh sửa đã bị từ chối")
}

// CancelEditRequest allows a PARTNER to cancel their own pending edit request
// @Summary Cancel timesheet edit request
// @Description Partner cancels their own pending timesheet edit request
// @Tags timesheets
// @Accept json
// @Produce json
// @Param id path int true "Timesheet ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /timesheets/{id}/request-edit-cancel [post]
func (h *EditRequestHandler) CancelEditRequest(c *gin.Context) {
	timesheetID, ok := h.validateTimesheetID(c)
	if !ok {
		return
	}

	userID, _, ok := h.getUserContext(c)
	if !ok {
		return
	}

	// First, find the pending edit request for this timesheet by this user
	editRequest, err := h.editRequestService.GetPendingForTimesheet(c.Request.Context(), timesheetID, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if editRequest == nil {
		response.NotFound(c, "Không tìm thấy yêu cầu chỉnh sửa đang chờ phê duyệt cho bảng chấm công này")
		return
	}

	err = h.editRequestService.CancelEditRequest(c.Request.Context(), editRequest.ID, userID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, "Yêu cầu chỉnh sửa đã được hủy thành công")
}

// Helper methods

func (h *EditRequestHandler) getUserContext(c *gin.Context) (userID uint, userRole string, ok bool) {
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return 0, "", false
	}

	userRoleInterface, exists := c.Get(constants.CtxUserRole)
	if !exists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return 0, "", false
	}

	return userIDInterface.(uint), userRoleInterface.(string), true
}

func (h *EditRequestHandler) validateTimesheetID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidTimesheetIDVN)
		return 0, false
	}
	return uint(id), true
}

func (h *EditRequestHandler) validateEditRequestID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "ID yêu cầu chỉnh sửa không hợp lệ")
		return 0, false
	}
	return uint(id), true
}

func (h *EditRequestHandler) parseEditRequestFilters(c *gin.Context) domain.TimesheetEditRequestFilters {
	filters := domain.TimesheetEditRequestFilters{
		Limit:     20,
		Offset:    0,
		SortBy:    "created_at",
		SortOrder: "DESC",
	}

	// Parse pagination
	if pageSize := c.Query("pageSize"); pageSize != "" {
		if size, err := strconv.Atoi(pageSize); err == nil && size > 0 {
			filters.Limit = size
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filters.Offset = (p - 1) * filters.Limit
		}
	}

	// Parse status filter
	if status := c.Query("status"); status != "" {
		statusValue := domain.TimesheetEditRequestStatus(status)
		filters.Status = []domain.TimesheetEditRequestStatus{statusValue}
	}

	return filters
}

func (h *EditRequestHandler) buildEditRequestResponse(req *domain.TimesheetEditRequest) dto.TimesheetEditRequestResponse {
	response := dto.TimesheetEditRequestResponse{
		ID:          req.ID,
		TimesheetID: req.TimesheetID,
		Status:      string(req.Status),
		RequestedBy: req.RequestedBy,
		CreatedAt:   req.CreatedAt,
		UpdatedAt:   req.UpdatedAt,
	}

	// Add user names
	if req.RequestedUser != nil {
		response.RequestedByName = req.RequestedUser.Fullname
	}

	if req.ApprovedBy != nil {
		response.ApprovedBy = req.ApprovedBy
		if req.ApprovedUser != nil {
			name := req.ApprovedUser.Fullname
			response.ApprovedByName = &name
		}
	}

	if req.RejectedBy != nil {
		response.RejectedBy = req.RejectedBy
		if req.RejectedUser != nil {
			name := req.RejectedUser.Fullname
			response.RejectedByName = &name
		}
	}

	// Add timesheet summary if loaded
	if req.Timesheet != nil {
		ts := req.Timesheet
		timesheetInfo := &dto.TimesheetSummaryInfo{
			ID:            ts.ID,
			Date:          ts.Date.Format("2006-01-02"),
			EmployeeID:    ts.EmployeeID,
			ProjectID:     ts.ProjectID,
			Paytype:       ts.PayType,
			HoursWorked:   ts.HoursWorked,
			Amount:        ts.Amount,
			Status:        string(ts.Status),
			PaymentStatus: string(ts.PaymentStatus),
		}

		if ts.Employee != nil {
			timesheetInfo.EmployeeName = ts.Employee.Fullname
			timesheetInfo.EmployeeCCCD = ts.Employee.CCCD
		}

		if ts.Project != nil {
			timesheetInfo.ProjectName = ts.Project.Name
		}

		response.Timesheet = timesheetInfo
	}

	return response
}
