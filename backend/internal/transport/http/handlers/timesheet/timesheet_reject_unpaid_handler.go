package timesheet

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// RejectUnpaidTimesheets rejects every non-paid timesheet in one project and
// inclusive date range. Casbin protects the route; the role check is defense in
// depth for direct handler use and future route changes.
func (h *Handler) RejectUnpaidTimesheets(c *gin.Context) {
	var req dto.RejectUnpaidTimesheetsRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, userRole, ok := h.getUserContext(c)
	if !ok {
		return
	}
	if userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, "Chỉ quản trị viên mới có thể từ chối hàng loạt bảng chấm công chưa thanh toán")
		return
	}

	fromDate, toDate, rejectionReason, err := validateRejectUnpaidRequest(req)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.timesheetService.RejectUnpaidByProjectDateRange(
		c.Request.Context(),
		req.ProjectID,
		fromDate,
		toDate,
		rejectionReason,
		userID,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, dto.RejectUnpaidTimesheetsResponse{
		RejectedCount:      result.RejectedCount,
		Committed:          true,
		PostCommitComplete: result.PostCommitComplete,
		Warnings:           result.Warnings,
	}, fmt.Sprintf("Đã từ chối %d bảng chấm công chưa thanh toán", result.RejectedCount))
}

func validateRejectUnpaidRequest(req dto.RejectUnpaidTimesheetsRequest) (time.Time, time.Time, string, error) {
	fromDate, err := time.Parse(time.DateOnly, req.FromDate)
	if err != nil || fromDate.Format(time.DateOnly) != req.FromDate {
		return time.Time{}, time.Time{}, "", fmt.Errorf("từ ngày phải có định dạng YYYY-MM-DD")
	}

	toDate, err := time.Parse(time.DateOnly, req.ToDate)
	if err != nil || toDate.Format(time.DateOnly) != req.ToDate {
		return time.Time{}, time.Time{}, "", fmt.Errorf("đến ngày phải có định dạng YYYY-MM-DD")
	}
	if fromDate.After(toDate) {
		return time.Time{}, time.Time{}, "", fmt.Errorf("từ ngày không thể sau đến ngày")
	}

	rejectionReason := strings.TrimSpace(req.RejectionReason)
	if rejectionReason == "" {
		return time.Time{}, time.Time{}, "", fmt.Errorf("lý do từ chối là bắt buộc")
	}
	if utf8.RuneCountInString(rejectionReason) > 500 {
		return time.Time{}, time.Time{}, "", fmt.Errorf("lý do từ chối không được vượt quá 500 ký tự")
	}

	return fromDate, toDate, rejectionReason, nil
}
