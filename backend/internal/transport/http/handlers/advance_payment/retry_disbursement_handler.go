package advance_payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/workers"
	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
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

	// Step 4: Refuse an active provider attempt, except for a stale pending row.
	// Pending is the pre-transfer state: no account check completed and no
	// OnePay transfer was initiated. After five minutes it is safe to restart
	// the exact persisted request; verified and authorised rows remain blocked
	// because OnePay may already have received them.
	var stalePending *domaintx.WalletPayment
	if h.txWalletPaymentRepo != nil {
		inFlight, err := h.txWalletPaymentRepo.HasNonTerminalByEntityID(ctx, requestID)
		if err != nil {
			slog.Error("retry disbursement: in-flight check failed",
				"request_id", requestID, "error", err)
			response.InternalServerError(c, "Không thể kiểm tra trạng thái chuyển tiền. Vui lòng thử lại sau.")
			return
		}
		if inFlight {
			stalePending, err = h.txWalletPaymentRepo.GetStaleAdvancePendingByEntityID(ctx, requestID, h.clock.Now().Add(-5*time.Minute))
			if err != nil && !errors.Is(err, domaintx.ErrNotFound) {
				slog.Error("retry disbursement: stale pending lookup failed",
					"request_id", requestID, "error", err)
				response.InternalServerError(c, "Không thể kiểm tra trạng thái chuyển tiền. Vui lòng thử lại sau.")
				return
			}
			if stalePending == nil {
				response.Conflict(c, "Đang có lệnh chuyển tiền đang xử lý cho yêu cầu này. Vui lòng đợi hoặc xoá khoản đang kẹt qua đối soát.")
				return
			}
		}
	}

	// Step 5: Build the execute payload. Stale pending recovery must use the
	// original idempotency key and recipient snapshot. A normal retry starts a
	// new provider request using the employee's current bank data.
	var payload workers.DisbursementExecutePayload
	if stalePending != nil {
		payload = workers.DisbursementExecutePayload{
			RequestID:          stalePending.RequestID,
			AdvanceRequestID:   requestID,
			RequestedAmount:    stalePending.RequestedAmount,
			RecipientName:      stalePending.RecipientName,
			RecipientAccountNo: stalePending.RecipientAccountNo,
			RecipientBank:      stalePending.RecipientBank,
		}
	} else {
		employee, err := h.service.GetConfig().EmployeeRepo.GetByID(ctx, req.EmployeeID)
		if err != nil {
			response.InternalServerError(c, "Không tìm thấy thông tin nhân viên")
			return
		}
		if employee.BankAccountNumber == "" || employee.BankAccountName == "" || employee.BankID == nil || employee.Bank == nil {
			response.BadRequest(c, "Nhân viên chưa có thông tin ngân hàng. Vui lòng cập nhật trước khi thử lại.")
			return
		}
		payload = workers.DisbursementExecutePayload{
			RequestID:          advance_payment.GenerateTransactionCode(true, true),
			AdvanceRequestID:   requestID,
			RequestedAmount:    int64(req.NetAmount),
			RecipientName:      employee.BankAccountName,
			RecipientAccountNo: employee.BankAccountNumber,
			RecipientBank:      employee.Bank.BankCode,
		}
	}
	disbursementRequestID := payload.RequestID

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
