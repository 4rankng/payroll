package attendance

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/attendance"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	attendanceService *attendance.AttendanceService
	employeeRepo      domain.EmployeeRepository
	failedAttemptRepo  domain.AttendanceFailedAttemptRepository
	clk               clock.Clock
	logger            *slog.Logger
}

func NewHandler(
	attendanceService *attendance.AttendanceService,
	employeeRepo domain.EmployeeRepository,
	failedAttemptRepo domain.AttendanceFailedAttemptRepository,
	clk clock.Clock,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		attendanceService: attendanceService,
		employeeRepo:      employeeRepo,
		failedAttemptRepo:  failedAttemptRepo,
		clk:               clk,
		logger:            logger,
	}
}

// resolveEmployeeID resolves the Employee.ID from the JWT's User.ID via the employee repository.
// The JWT stores User.ID (from the users table), but the attendance service needs Employee.ID.
func (h *Handler) resolveEmployeeID(c *gin.Context) (uint, bool) {
	uid, ok := helpers.GetUserIDOrRespond(c)
	if !ok {
		return 0, false
	}
	employee, err := h.employeeRepo.GetByUserID(c.Request.Context(), uid)
	if err != nil {
		response.HandleDomainError(c, err)
		return 0, false
	}
	return employee.ID, true
}

// recordFailedAttempt fire-and-forgets a failed-attempt log for a validation error.
// It uses context.Background() with a timeout so the write never blocks the
// request and survives the request-context cancellation after the response is sent.
func (h *Handler) recordFailedAttempt(employeeID, projectID uint, attemptType, category, errorMsg string, lat, lng float64) {
	if h.failedAttemptRepo == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("panic recording failed attempt", "panic", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		msg := errorMsg
		var latPtr, lngPtr *float64
		if lat != 0 || lng != 0 {
			latPtr, lngPtr = &lat, &lng
		}
		attempt := &domain.AttendanceFailedAttempt{
			EmployeeID:     employeeID,
			AttemptType:    attemptType,
			ReasonCategory: category,
			ProjectID:      projectID,
			Lat:            latPtr,
			Lng:            lngPtr,
			ErrorMessage:   &msg,
		}
		if err := h.failedAttemptRepo.Create(ctx, attempt); err != nil {
			h.logger.Warn("failed to log attendance failed attempt", "error", err)
		}
	}()
}

func (h *Handler) mapToResponse(att *domain.Attendance) *dto.AttendanceResponse {
	salaryStatus := "pending"
	salaryMessage := "Lương sẽ được ghi nhận sau khi bạn tan ca."
	if att.SalaryRejectReason != nil && att.CheckOutTime == nil {
		// Auto-rejected (checkout window expired, no checkout) — salary will
		// never be recorded. Surface the reject reason instead of the default
		// "salary will be recorded after checkout" message, which is misleading.
		salaryStatus = "not_recorded"
		salaryMessage = *att.SalaryRejectReason
	} else if att.CheckOutTime != nil {
		salaryStatus = "not_recorded"
		salaryMessage = "Chưa ghi nhận lương cho ca này. Vui lòng kiểm tra khung giờ ca làm hoặc liên hệ quản lý."
		if att.EarningAmount != nil && *att.EarningAmount > 0 {
			salaryStatus = "recorded"
			salaryMessage = "Đã ghi nhận lương cho ca này. Bạn có thể yêu cầu ứng lương nếu còn hạn mức."
		} else if att.SalaryRejectReason != nil && *att.SalaryRejectReason != "" {
			salaryMessage = *att.SalaryRejectReason
		}
	}

	return &dto.AttendanceResponse{
		ID:                 att.ID,
		ProjectID:          att.ProjectID,
		EmployeeID:         att.EmployeeID,
		Date:               att.Date,
		CheckInTime:        att.CheckInTime,
		CheckInGate:        att.CheckInGate,
		CheckOutTime:       att.CheckOutTime,
		CheckOutGate:       att.CheckOutGate,
		EarningAmount:      att.EarningAmount,
		SalaryRejectReason: att.SalaryRejectReason,
		SalaryStatus:       salaryStatus,
		SalaryMessage:      salaryMessage,
		Status:             string(att.GetStatus(h.clk.Now())),
	}
}

// CheckIn handles POST /api/v1/mobile/attendance/check-in
func (h *Handler) CheckIn(c *gin.Context) {
	var req dto.CheckInRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ: "+err.Error())
		return
	}

	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.CheckIn(c.Request.Context(), employeeID, req.ProjectID, req.Lat, req.Lng)
	if err != nil {
		if domain.IsValidationError(err) {
			h.recordFailedAttempt(employeeID, req.ProjectID, "check_in",
				attendance.ClassifyAttemptError(err.Error()), err.Error(), req.Lat, req.Lng)
		}
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, h.mapToResponse(att), "Vào làm thành công")
}

// CheckOut handles POST /api/v1/mobile/attendance/check-out
func (h *Handler) CheckOut(c *gin.Context) {
	var req dto.CheckOutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Dữ liệu không hợp lệ: "+err.Error())
		return
	}

	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.CheckOut(c.Request.Context(), employeeID, req.Lat, req.Lng, req.ConfirmNoSalary)
	if err != nil {
		if domain.IsValidationError(err) {
			// CheckOut request has no projectID field; the project is resolved
			// from the attendance record which we don't have here. Use 0.
			h.recordFailedAttempt(employeeID, 0, "check_out",
				attendance.ClassifyAttemptError(err.Error()), err.Error(), req.Lat, req.Lng)
		}
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, h.mapToResponse(att), "Tan ca thành công")
}

// GetToday handles GET /api/v1/mobile/attendance/today
func (h *Handler) GetToday(c *gin.Context) {
	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	att, err := h.attendanceService.GetTodayAttendance(c.Request.Context(), employeeID)
	if err != nil {
		h.logger.Error("failed to get today attendance", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy thông tin chấm công")
		return
	}

	if att == nil {
		response.Success(c, nil, "Chưa có dữ liệu chấm công hôm nay")
		return
	}

	response.Success(c, h.mapToResponse(att), "Lấy thông tin chấm công thành công")
}

// List handles GET /api/v1/mobile/attendance/history
func (h *Handler) List(c *gin.Context) {
	employeeID, ok := h.resolveEmployeeID(c)
	if !ok {
		return
	}

	filters := domain.AttendanceFilters{
		EmployeeID: &employeeID,
		Limit:      20,
	}

	// Allow client to override limit (capped at 100)
	if limitStr := c.Query("limit"); limitStr != "" {
		if lim, err := strconv.Atoi(limitStr); err == nil && lim > 0 && lim <= 100 {
			filters.Limit = lim
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if off, err := strconv.Atoi(offsetStr); err == nil && off >= 0 {
			filters.Offset = off
		}
	}

	attendances, _, err := h.attendanceService.List(c.Request.Context(), filters)
	if err != nil {
		h.logger.Error("failed to get attendance history", "error", err)
		response.InternalServerError(c, "Lỗi hệ thống khi lấy lịch sử chấm công")
		return
	}

	res := make([]*dto.AttendanceResponse, len(attendances))
	for i, att := range attendances {
		res[i] = h.mapToResponse(att)
	}

	response.Success(c, res, "Lấy lịch sử chấm công thành công")
}
