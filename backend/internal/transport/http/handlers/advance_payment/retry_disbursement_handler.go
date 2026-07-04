package advance_payment

import (
	"encoding/json"
	"errors"
	"fmt"
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

	// Step 5.5: Refuse if a wallet_payment for this advance request is still
	// in flight (pending/verified/authorised). The execute worker is already
	// idempotent, so this is not a double-pay guard — it stops a second click
	// from piling on a duplicate task while one is mid-flight. A stuck row must
	// be cleared via reconciliation before retrying.
	if h.txWalletPaymentRepo != nil {
		inFlight, err := h.txWalletPaymentRepo.HasNonTerminalByEntityID(ctx, requestID)
		if err != nil {
			slog.Error("retry disbursement: in-flight check failed",
				"request_id", requestID, "error", err)
			response.InternalServerError(c, "Không thể kiểm tra trạng thái chuyển tiền. Vui lòng thử lại sau.")
			return
		}
		if inFlight {
			response.Conflict(c, "Đang có lệnh chuyển tiền đang xử lý cho yêu cầu này. Vui lòng đợi hoặc xoá khoản đang kẹt qua đối soát.")
			return
		}
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

	// TaskID binds the in-flight task to the advance request. A second enqueue
	// while this task is still pending/running/retrying collides at the queue
	// (ErrTaskIDConflict) instead of spawning a duplicate. The ID is released
	// once the task reaches a terminal state, so a later retry (after the prior
	// task is truly done) still enqueues cleanly.
	task := asynqlib.NewTask(workers.TaskDisbursementExecute, data)
	taskID := fmt.Sprintf("disbursement:execute:%d", requestID)
	if _, err := h.asynqClient.AsynqClient().Enqueue(task,
		asynqlib.MaxRetry(5),
		asynqlib.TaskID(taskID),
	); err != nil {
		if errors.Is(err, asynqlib.ErrTaskIDConflict) {
			slog.Info("retry disbursement: task already in flight (TaskID conflict)",
				"request_id", requestID)
			response.Conflict(c, "Đang có lệnh chuyển tiền đang xử lý cho yêu cầu này.")
			return
		}
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
