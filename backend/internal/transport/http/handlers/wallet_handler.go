package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/services"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/constants"
	infrastructure "api-server/internal/domain/ports/infrastructure"
	"api-server/internal/domain/wallet"
	"api-server/internal/infra/disbursement/ninepay"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"
	"api-server/internal/transport/http/uploadguard"

	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	service  wallet.WalletService
	forecast *services.WalletDemandForecastService
	registry *disbursement.Registry
	clock    clock.Clock
	logger   *slog.Logger
}

func NewWalletHandler(service wallet.WalletService, registry *disbursement.Registry, forecast *services.WalletDemandForecastService, clk clock.Clock) *WalletHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &WalletHandler{
		service:  service,
		forecast: forecast,
		registry: registry,
		clock:    clk,
		logger:   slog.Default().With("component", "WalletHandler"),
	}
}

func (h *WalletHandler) GetBalance(c *gin.Context) {
	balance, err := h.service.GetBalance(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, balance, "Lấy số dư ví thành công")
}

// GetDemandForecast returns the advance-payment cohort chart series + the
// advisory balance prediction for the Wallet page. Display-only by contract.
func (h *WalletHandler) GetDemandForecast(c *gin.Context) {
	if h.forecast == nil {
		response.InternalServerError(c, "Dịch vụ dự báo nhu cầu ứng lương chưa được cấu hình")
		return
	}
	resp, err := h.forecast.GetDemandForecast(c.Request.Context())
	if err != nil {
		h.logger.Error("demand-forecast: failed", "error", err)
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, resp, "Lấy dự báo nhu cầu ứng lương thành công")
}

func (h *WalletHandler) SyncBalance(c *gin.Context) {
	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))

	result, err := h.service.SyncBalance(c.Request.Context(), userID)
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrProviderNotConfigured):
			response.BadRequest(c, "Chưa cấu hình nhà cung cấp thanh toán")
		case errors.Is(err, wallet.ErrProviderBalanceUnsupported):
			response.BadRequest(c, "Nhà cung cấp không hỗ trợ truy vấn số dư")
		default:
			h.logger.Error("sync-balance: failed", "error", err)
			response.InternalServerError(c, err.Error())
		}
		return
	}

	response.Success(c, gin.H{
		"provider_balance": result.ProviderBalance,
		"local_balance":    result.LocalBalance,
		"adjusted":         result.Adjusted,
		"currency":         "VND",
	}, "Đồng bộ số dư thành công")
}

func (h *WalletHandler) AdjustBalance(c *gin.Context) {
	var body struct {
		Amount int64  `json:"amount" binding:"required"`
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if body.Amount == 0 {
		response.BadRequest(c, "Amount không được bằng 0")
		return
	}

	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))

	now := h.clock.Now()

	var note *string
	adjustedNote := fmt.Sprintf("[Điều chỉnh đối soát 9pay] %s", body.Reason)
	note = &adjustedNote

	req := wallet.CreateWalletTopupRequest{
		Amount:     body.Amount,
		BankRef:    fmt.Sprintf("SYNC-ADJ-%s", now.Format("20060102-150405")),
		OccurredAt: now,
		Note:       note,
	}

	topup, err := h.service.CreateTopup(c.Request.Context(), req, userID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, toWalletTopupResponse(topup), "Điều chỉnh số dư thành công")
}

func (h *WalletHandler) GetTransactions(c *gin.Context) {
	filter := wallet.TransactionFilter{
		Type:     c.Query("type"),
		Status:   c.Query("status"),
		Page:     1,
		PageSize: 50,
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "50")); err == nil && pageSize > 0 {
		filter.PageSize = pageSize
	}

	fromDate := c.Query("from")
	if fromDate != "" {
		filter.FromDate = &fromDate
	}
	toDate := c.Query("to")
	if toDate != "" {
		filter.ToDate = &toDate
	}

	items, total, err := h.service.GetTransactions(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, items, "Lấy danh sách giao dịch thành công", response.Pagination{
		Page:         filter.Page,
		PageSize:     filter.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}

func (h *WalletHandler) CreateTopup(c *gin.Context) {
	var req wallet.CreateWalletTopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))
	topup, err := h.service.CreateTopup(c.Request.Context(), req, userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate bank_ref") {
			response.Conflict(c, err.Error())
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.SuccessCreated(c, toWalletTopupResponse(topup), "Tạo bản ghi nạp tiền thành công")
}

func (h *WalletHandler) GetTopupByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID không hợp lệ")
		return
	}

	topup, err := h.service.GetTopupByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Không tìm thấy bản ghi nạp tiền")
		return
	}

	response.Success(c, toWalletTopupResponse(topup), "Lấy chi tiết nạp tiền thành công")
}

func (h *WalletHandler) ListTopups(c *gin.Context) {
	filter := wallet.WalletTopupFilter{
		BankRef:  c.Query("bank_ref"),
		Page:     1,
		PageSize: 20,
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && pageSize > 0 {
		filter.PageSize = pageSize
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			filter.EndDate = &t
		}
	}

	topups, total, err := h.service.ListTopups(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, toWalletTopupResponses(topups), "Lấy danh sách nạp tiền thành công", response.Pagination{
		Page:         filter.Page,
		PageSize:     filter.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}

func (h *WalletHandler) GetPaymentByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID không hợp lệ")
		return
	}

	payment, err := h.service.GetPaymentByID(c.Request.Context(), id)
	if err != nil {
		response.NotFound(c, "Không tìm thấy giao dịch")
		return
	}

	response.Success(c, toWalletPaymentResponse(payment), "Lấy chi tiết giao dịch thành công")
}

func (h *WalletHandler) ListPayments(c *gin.Context) {
	filter := wallet.WalletPaymentFilter{
		TxnID:            c.Query("txn_id"),
		RequestID:        c.Query("request_id"),
		InvoiceNo:        c.Query("invoice_no"),
		Status:           c.Query("status"),
		ErrorCode:        c.Query("error_code"),
		RecipientName:    c.Query("recipient_name"),
		RecipientAccount: c.Query("recipient_account"),
		RecipientBank:    c.Query("recipient_bank"),
		Page:             1,
		PageSize:         20,
	}

	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && page > 0 {
		filter.Page = page
	}
	if pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && pageSize > 0 {
		filter.PageSize = pageSize
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			filter.EndDate = &t
		}
	}

	payments, total, err := h.service.ListPayments(c.Request.Context(), filter)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	totalPages := int(total) / filter.PageSize
	if int(total)%filter.PageSize > 0 {
		totalPages++
	}

	response.SuccessWithPagination(c, toWalletPaymentResponses(payments), "Lấy danh sách giao dịch thành công", response.Pagination{
		Page:         filter.Page,
		PageSize:     filter.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	})
}

func (h *WalletHandler) UploadReconciliation(c *gin.Context) {
	fileContent, _, ok := uploadguard.Receive(c, false)
	if !ok {
		return
	}

	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))
	jobID, err := h.service.UploadReconciliation(c.Request.Context(), fileContent, userID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"job_id": jobID}, "Tải file đối soát thành công")
}

func (h *WalletHandler) GetReconciliationJobStatus(c *gin.Context) {
	jobID := c.Param("id")
	job, err := h.service.GetReconciliationJobStatus(c.Request.Context(), jobID)
	if err != nil {
		response.NotFound(c, "Không tìm thấy job đối soát")
		return
	}
	response.Success(c, job, "Lấy trạng thái đối soát thành công")
}

func (h *WalletHandler) ExportReconciliationReport(c *gin.Context) {
	month := c.DefaultQuery("month", h.clock.Now().Format("2006-01"))

	data, err := h.service.ExportReconciliationReport(c.Request.Context(), month)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	filename := fmt.Sprintf("reconciliation-report-%s.csv", month)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", data)
}

// ResolvePayment handles POST /wallet/payments/:id/resolve.
// Manual admin override of payment status (complete, fail, or reverse).
func (h *WalletHandler) ResolvePayment(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID không hợp lệ")
		return
	}

	var body struct {
		Action string `json:"action" binding:"required"`
		Reason string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))

	payment, err := h.service.ResolvePayment(c.Request.Context(), id, body.Action, body.Reason, userID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, toWalletPaymentResponse(payment), "Cập nhật trạng thái giao dịch thành công")
}

// AutoReconcile handles POST /wallet/reconcile/auto.
func (h *WalletHandler) AutoReconcile(c *gin.Context) {
	if h.registry == nil {
		response.BadRequest(c, "Chưa cấu hình nhà cung cấp thanh toán")
		return
	}

	provider, err := h.registry.Active(c.Request.Context())
	if err != nil {
		response.BadRequest(c, "Không có nhà cung cấp thanh toán đang hoạt động")
		return
	}

	exporter, ok := provider.(infrastructure.ReportExporter)
	if !ok {
		response.BadRequest(c, "Nhà cung cấp không hỗ trợ xuất đối soát")
		return
	}

	now := h.clock.Now()
	loc := now.Location()

	var dateFrom, dateTo time.Time
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	if startDateStr != "" && endDateStr != "" {
		var parseErr error
		dateFrom, parseErr = time.ParseInLocation("02/01/2006", startDateStr, loc)
		if parseErr != nil {
			response.BadRequest(c, "Định dạng start_date không hợp lệ (DD/MM/YYYY)")
			return
		}
		dateTo, parseErr = time.ParseInLocation("02/01/2006", endDateStr, loc)
		if parseErr != nil {
			response.BadRequest(c, "Định dạng end_date không hợp lệ (DD/MM/YYYY)")
			return
		}
		dateFrom = time.Date(dateFrom.Year(), dateFrom.Month(), dateFrom.Day(), 0, 0, 0, 0, loc)
		dateTo = time.Date(dateTo.Year(), dateTo.Month(), dateTo.Day(), 23, 59, 59, 0, loc)

		if dateTo.Before(dateFrom) {
			response.BadRequest(c, "start_date phải nhỏ hơn hoặc bằng end_date")
			return
		}
		maxStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -30)
		if dateFrom.Before(maxStart) {
			response.BadRequest(c, "Khoảng ngày tối đa là 30 ngày")
			return
		}
		todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)
		if dateFrom.After(todayEnd) || dateTo.After(todayEnd) {
			response.BadRequest(c, "Không được chọn ngày trong tương lai")
			return
		}
	} else {
		days := 7
		if d, err := strconv.Atoi(c.DefaultQuery("days", "7")); err == nil && d > 0 && d <= 30 {
			days = d
		}
		dateFrom = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, -days)
		dateTo = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, loc).AddDate(0, 0, -1)
	}

	h.logger.Info("auto-reconcile: downloading from provider",
		"date_from", dateFrom.Format("02/01/2006"),
		"date_to", dateTo.Format("02/01/2006"))

	csvBytes, _, err := exporter.ExportReconciliation(c.Request.Context(), dateFrom, dateTo)
	if err != nil {
		if errors.Is(err, ninepay.ErrNoTransactions) {
			h.logger.Info("auto-reconcile: no transactions in date range")
			response.Success(c, gin.H{"job_id": nil}, "Không có giao dịch trong khoảng ngày đã chọn")
			return
		}
		h.logger.Error("auto-reconcile: export failed", "error", err)
		response.InternalServerError(c, "Tải file đối soát từ nhà cung cấp thất bại: "+err.Error())
		return
	}

	adminUserID, _ := c.Get(constants.CtxUserID)
	userID := uint64(adminUserID.(uint))
	jobID, err := h.service.UploadReconciliation(c.Request.Context(), csvBytes, userID)
	if err != nil {
		h.logger.Error("auto-reconcile: processing failed", "error", err)
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{"job_id": jobID}, "Tự động đối soát thành công")
}

func toWalletTopupResponse(t *wallet.WalletTopup) *wallet.WalletTopupResponse {
	return &wallet.WalletTopupResponse{
		ID:         t.ID,
		Amount:     t.Amount,
		BankRef:    t.BankRef,
		OccurredAt: t.OccurredAt.Format(time.RFC3339),
		Note:       t.Note,
		CreatedBy:  t.CreatedBy,
		CreatedAt:  t.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  t.UpdatedAt.Format(time.RFC3339),
	}
}

func toWalletTopupResponses(topups []*wallet.WalletTopup) []*wallet.WalletTopupResponse {
	responses := make([]*wallet.WalletTopupResponse, len(topups))
	for i, t := range topups {
		responses[i] = toWalletTopupResponse(t)
	}
	return responses
}

func toWalletPaymentResponse(p *wallet.WalletPayment) *wallet.WalletPaymentResponse {
	settledAt := (*string)(nil)
	if p.SettledAt != nil {
		timeStr := p.SettledAt.Format(time.RFC3339)
		settledAt = &timeStr
	}

	return &wallet.WalletPaymentResponse{
		ID:                 p.ID,
		TxnID:              p.TxnID,
		RequestID:          p.RequestID,
		InvoiceNo:          p.InvoiceNo,
		RequestedAmount:    p.RequestedAmount,
		Fee:                p.Fee,
		RecipientName:      p.RecipientName,
		RecipientAccountNo: p.RecipientAccountNo,
		RecipientBank:      p.RecipientBank,
		Description:        p.Description,
		Status:             p.Status,
		ErrorCode:          p.ErrorCode,
		ErrorMessage:       p.ErrorMessage,
		EntityID:           p.EntityID,
		CreatedBy:          p.CreatedBy,
		Version:            p.Version,
		CreatedAt:          p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:          p.UpdatedAt.Format(time.RFC3339),
		SettledAt:          settledAt,
	}
}

func toWalletPaymentResponses(payments []*wallet.WalletPayment) []*wallet.WalletPaymentResponse {
	responses := make([]*wallet.WalletPaymentResponse, len(payments))
	for i, p := range payments {
		responses[i] = toWalletPaymentResponse(p)
	}
	return responses
}
