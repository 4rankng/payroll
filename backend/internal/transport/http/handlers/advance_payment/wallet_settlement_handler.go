package advance_payment

import (
	"errors"
	"log/slog"

	"api-server/internal/app/workers"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	asynqlib "github.com/hibiken/asynq"
)

// walletSettlementTaskID deduplicates the settlement task across the daily cron
// (RegisterWalletSettlement in infra/asynq) and this manual trigger. It MUST
// match the TaskID used there. A second enqueue while one is still
// pending/running collides at the queue (ErrTaskIDConflict) instead of spawning
// a duplicate run that races the date guard and double-writes the ledger.
// Completed tasks are purged immediately (the asynq server sets no retention),
// so the ID frees as soon as a run finishes and the trigger stays re-usable.
const walletSettlementTaskID = "wallet:settlement:run"

// RunWalletSettlement manually triggers the EOD wallet-settlement task. It
// re-enqueues the same task the daily 00:01 cron runs, which consolidates every
// stranded completed wallet payment — grouped by completion day — into
// "Wallet disbursement YYYY-MM-DD" ledger records. Use it to settle on demand
// when the cron has not run yet or missed a day (e.g. the process was down at
// 00:01). The shared TaskID collapses double-clicks and cron/admin overlap into
// a single in-flight run.
//
// POST /api/v1/admin/wallet-settlement/run
func (h *AdvancePaymentHandler) RunWalletSettlement(c *gin.Context) {
	task := asynqlib.NewTask(workers.TaskWalletSettlement, nil)
	if _, err := h.asynqClient.AsynqClient().Enqueue(task,
		asynqlib.Queue(asynqinfra.QueueLow),
		asynqlib.TaskID(walletSettlementTaskID),
	); err != nil {
		if errors.Is(err, asynqlib.ErrTaskIDConflict) {
			slog.Info("wallet settlement: task already in flight (TaskID conflict)")
			response.Conflict(c, "Đang có lệnh chốt lương đang chạy. Vui lòng đợi ít phút rồi thử lại.")
			return
		}
		slog.Error("wallet settlement: failed to enqueue task", "error", err)
		response.InternalServerError(c, "Không thể kích chạy chốt lương. Vui lòng thử lại sau.")
		return
	}

	slog.Info("wallet settlement: enqueued by admin")
	response.Success(c, gin.H{
		"message": "Đã kích chạy chốt lương. Các bản ghi chưa chốt sẽ được tạo trong giây lát.",
	}, "Kích chạy chốt lương thành công")
}
