package advance_payment

import (
	"encoding/json"
	"log/slog"
	"strconv"

	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/workers"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	asynqlib "github.com/hibiken/asynq"
)

// RetryDisbursement re-enqueues a disbursement task for an APPROVED or FAILED
// advance payment request. This allows admins to manually trigger a retry when
// the automatic poller missed a request or a previous attempt failed transiently.
//
// POST /api/v1/advance-payments/:id/retry-disbursement
func (h *AdvancePaymentHandler) RetryDisbursement(c *gin.Context) {
	requestIDStr := c.Param("id")
	requestID, err := strconv.ParseUint(requestIDStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "ID yêu cầu không hợp lệ")
		return
	}

	ctx := c.Request.Context()

	// Step 1: Load the request
	req, err := h.service.GetConfig().AdvancePaymentRequestRepo.GetByID(ctx, requestID)
	if err != nil {
		response.InternalServerError(c, "Không tìm thấy yêu cầu")
		return
	}

	// Step 2: Validate status — only APPROVED or FAILED can be retried
	if !req.IsApproved() && !req.IsFailed() {
		response.BadRequest(c, "Chỉ có thể thử lại yêu cầu ở trạng thái Đã duyệt hoặc Thất bại")
		return
	}

	// Step 3: If FAILED, reset to APPROVED so the execute worker accepts it
	if req.IsFailed() {
		if err := h.service.GetConfig().AdvancePaymentRequestRepo.UpdateStatus(ctx, requestID, domain.AdvancePaymentStatusApproved, "", nil); err != nil {
			response.InternalServerError(c, "Không thể cập nhật trạng thái yêu cầu")
			return
		}
		req.Status = domain.AdvancePaymentStatusApproved
	}

	// Step 4: Load employee with bank details
	employee, err := h.service.GetConfig().EmployeeRepo.GetByID(ctx, req.EmployeeID)
	if err != nil {
		response.InternalServerError(c, "Không tìm thấy thông tin nhân viên")
		return
	}

	// Step 5: Validate bank details
	if employee.BankAccountNumber == "" || employee.BankAccountName == "" || employee.BankID == nil || employee.Bank == nil {
		response.BadRequest(c, "Nhân viên chưa có thông tin ngân hàng. Vui lòng cập nhật trước khi thử lại.")
		return
	}

	// Step 6: Enqueue disbursement:execute task (same logic as poller's enqueueRequest)
	disbursementRequestID := advance_payment.GenerateTransactionCode(true, true)

	payload := workers.DisbursementExecutePayload{
		RequestID:          disbursementRequestID,
		AdvanceRequestID:   requestID,
		RequestedAmount:    int64(req.NetAmount),
		RecipientName:      employee.BankAccountName,
		RecipientAccountNo: employee.BankAccountNumber,
		RecipientBank:      employee.Bank.BankCode,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		response.InternalServerError(c, "Lỗi xử lý dữ liệu")
		return
	}

	task := asynqlib.NewTask(workers.TaskDisbursementExecute, data)
	if _, err := h.asynqClient.AsynqClient().Enqueue(task, asynqlib.MaxRetry(5)); err != nil {
		slog.Error("retry disbursement: failed to enqueue task",
			"request_id", requestID, "error", err,
		)
		response.InternalServerError(c, "Không thể tạo lệnh chuyển tiền. Vui lòng thử lại sau.")
		return
	}

	slog.Info("retry disbursement: enqueued by admin",
		"advance_request_id", requestID,
		"disbursement_request_id", disbursementRequestID,
		"employee_id", req.EmployeeID,
		"amount", req.NetAmount,
	)

	response.Success(c, gin.H{
		"requestId":          requestID,
		"disbursementTaskId": disbursementRequestID,
		"message":            "Đã tạo lệnh chuyển tiền thành công",
	}, "Thử lại chuyển tiền thành công")
}
