package asynq

import (
	"fmt"

	asynqlib "github.com/hibiken/asynq"
)

// RegisterHandlers registers all task handlers on the server's mux
func RegisterHandlers(srv *Server, h *Handlers) {
	srv.Mux().Handle(TaskEmployeeImport, asynqlib.HandlerFunc(h.HandleEmployeeImport))
	srv.Mux().Handle(TaskBCCImport, asynqlib.HandlerFunc(h.HandleBCCImport))
	srv.Mux().Handle(TaskBCCImportSweep, asynqlib.HandlerFunc(h.HandleBCCImportSweep))
	srv.Mux().Handle(TaskImportJob, asynqlib.HandlerFunc(h.HandleImportJob))
	srv.Mux().Handle(TaskFlexPaySalaryNotification, asynqlib.HandlerFunc(h.HandleFlexPaySalaryNotification))
	srv.Mux().Handle(TaskFlexPaySalaryNotificationSweep, asynqlib.HandlerFunc(h.HandleFlexPaySalaryNotificationSweep))
	srv.Mux().Handle(TaskIPNProcess, asynqlib.HandlerFunc(h.HandleIPNProcess))
	srv.Mux().Handle(TaskBulkTransferTransaction, asynqlib.HandlerFunc(h.HandleBulkTransferTransaction))
	srv.Mux().Handle(TaskBulkTransferPayment, asynqlib.HandlerFunc(h.HandleBulkTransferPayment))
	srv.Mux().Handle(TaskAuditLogWrite, asynqlib.HandlerFunc(h.HandleAuditLogWrite))
	srv.Mux().Handle(TaskPayrollReportEmail, asynqlib.HandlerFunc(h.HandlePayrollReportEmail))
	srv.Mux().Handle(TaskAutoRejectCheckout, asynqlib.HandlerFunc(h.HandleAutoRejectCheckout))
	srv.Mux().Handle(TaskAutoRejectSweep, asynqlib.HandlerFunc(h.HandleAutoRejectSweep))
	srv.Mux().Handle(TaskCreditQuota, asynqlib.HandlerFunc(h.HandleCreditQuota))
	srv.Mux().Handle(TaskCreditQuotaSweep, asynqlib.HandlerFunc(h.HandleCreditQuotaSweep))

	if h.disbursementPollerWorker != nil {
		srv.Mux().Handle(TaskDisbursementPoller, asynqlib.HandlerFunc(h.HandleDisbursementPoller))
		srv.Mux().Handle(TaskDisbursementExecute, asynqlib.HandlerFunc(h.HandleDisbursementExecute))
	}

	if h.ninePayBulkExecuteWorker != nil {
		srv.Mux().Handle(TaskNinePayBulkExecute, asynqlib.HandlerFunc(h.HandleNinePayBulkExecute))
	}

	if h.walletSettlementWorker != nil {
		srv.Mux().Handle(TaskWalletSettlement, asynqlib.HandlerFunc(h.HandleWalletSettlement))
	}

	if h.statusInquiryPollerWorker != nil {
		srv.Mux().Handle(TaskStatusInquiry, asynqlib.HandlerFunc(h.HandleStatusInquiry))
	}

	if h.walletBulkRowWorker != nil && h.walletBulkSvc != nil {
		srv.Mux().Handle(TaskWalletBulkTransferRow, asynqlib.HandlerFunc(h.HandleWalletBulkTransferRow))
		srv.Mux().Handle(TaskWalletBookBatchLedger, asynqlib.HandlerFunc(h.HandleWalletBookBatchLedger))
		srv.Mux().Handle(TaskWalletStaleEnqueueSweeper, asynqlib.HandlerFunc(h.HandleWalletStaleEnqueueSweeper))
		srv.Mux().Handle(TaskWalletCompletingRecovery, asynqlib.HandlerFunc(h.HandleWalletCompletingRecovery))
	}

	registered := []string{
		TaskEmployeeImport, TaskBCCImport, TaskBCCImportSweep, TaskImportJob, TaskFlexPaySalaryNotification, TaskFlexPaySalaryNotificationSweep, TaskIPNProcess,
		TaskBulkTransferTransaction, TaskBulkTransferPayment,
		TaskAuditLogWrite, TaskPayrollReportEmail, TaskAutoRejectCheckout, TaskAutoRejectSweep,
		TaskCreditQuota, TaskCreditQuotaSweep,
	}
	if h.disbursementPollerWorker != nil {
		registered = append(registered, TaskDisbursementPoller, TaskDisbursementExecute)
	}
	if h.ninePayBulkExecuteWorker != nil {
		registered = append(registered, TaskNinePayBulkExecute)
	}
	if h.walletSettlementWorker != nil {
		registered = append(registered, TaskWalletSettlement)
	}
	if h.statusInquiryPollerWorker != nil {
		registered = append(registered, TaskStatusInquiry)
	}
	if h.walletBulkRowWorker != nil && h.walletBulkSvc != nil {
		registered = append(registered,
			TaskWalletBulkTransferRow, TaskWalletBookBatchLedger,
			TaskWalletStaleEnqueueSweeper, TaskWalletCompletingRecovery)
	}

	logger.Info("Registered asynq task handlers", "tasks", registered)
}

// RegisterPeriodicTasks registers periodic tasks on the scheduler
func RegisterPeriodicTasks(srv *Server, _ *Client) error {
	if _, err := srv.Scheduler().Register(
		"@every 1m",
		asynqlib.NewTask(TaskBCCImportSweep, nil),
		asynqlib.Queue(QueueLow),
	); err != nil {
		return fmt.Errorf("failed to register BCC import recovery sweep: %w", err)
	}
	if _, err := srv.Scheduler().Register(
		"@every 1m",
		asynqlib.NewTask(TaskFlexPaySalaryNotificationSweep, nil),
		asynqlib.Queue(QueueLow),
	); err != nil {
		return fmt.Errorf("failed to register salary notification recovery sweep: %w", err)
	}
	logger.Info("Registered asynq periodic tasks", "bcc_import_recovery", "1m", "salary_notification_recovery", "1m")
	return nil
}

// RegisterAutoRejectSweep registers the periodic auto-reject fallback sweep.
// Runs every 30 minutes on the low-priority queue. It is the safety net for the
// per-attendance K+4h task: it finalizes attendance records the scheduled task
// missed (Redis/process outage at check-in). The sweeper is idempotent.
func RegisterAutoRejectSweep(srv *Server) error {
	_, err := srv.Scheduler().Register("@every 30m", asynqlib.NewTask(TaskAutoRejectSweep, nil),
		asynqlib.Queue(QueueLow),
	)
	if err != nil {
		return fmt.Errorf("failed to register auto-reject sweep periodic task: %w", err)
	}
	logger.Info("Registered auto-reject sweep periodic task", "interval", "30m")
	return nil
}

// RegisterCreditQuotaSweep registers the periodic quota-credit fallback sweep.
// Runs every 30 minutes on the low-priority queue. It is the safety net for the
// per-attendance credit task: it banks earnings the scheduled task missed
// (Redis/process outage at check-out). The sweeper is idempotent on
// quota_credited_at, so retries and overlapping runs are safe.
func RegisterCreditQuotaSweep(srv *Server) error {
	_, err := srv.Scheduler().Register("@every 30m", asynqlib.NewTask(TaskCreditQuotaSweep, nil),
		asynqlib.Queue(QueueLow),
	)
	if err != nil {
		return fmt.Errorf("failed to register quota-credit sweep periodic task: %w", err)
	}
	logger.Info("Registered quota-credit sweep periodic task", "interval", "30m")
	return nil
}

// RegisterDisbursementPoller registers the periodic disbursement poller task.
// Called only when ENABLE_NINEPAY and ENABLE_NINEPAY_FOR_EMPLOYEE are both true.
func RegisterDisbursementPoller(srv *Server) error {
	_, err := srv.Scheduler().Register("@every 20s", asynqlib.NewTask(TaskDisbursementPoller, nil),
		asynqlib.Queue(QueueLow),
	)
	if err != nil {
		return fmt.Errorf("failed to register disbursement poller periodic task: %w", err)
	}

	logger.Info("Registered disbursement poller periodic task", "interval", "20s")
	return nil
}

// RegisterWalletSettlement registers the periodic EOD wallet settlement task.
// Fires daily at 00:01 Asia/Ho_Chi_Minh — the scheduler evaluates crontab specs
// in clock.DefaultLocation (see NewServer), not asynq's UTC default. The worker
// is self-healing: each run sweeps every stranded payment regardless of
// completion day, so a missed run is backfilled on the next tick, and the manual
// admin trigger (RunWalletSettlement) covers on-demand needs. The TaskID mirrors
// the admin trigger so a cron fire and an on-demand click — or two clicks — can
// never run two settlement tasks at once and race the guard.
func RegisterWalletSettlement(srv *Server) error {
	_, err := srv.Scheduler().Register("1 0 * * *", asynqlib.NewTask(TaskWalletSettlement, nil),
		asynqlib.Queue(QueueLow),
		asynqlib.TaskID("wallet:settlement:run"),
	)
	if err != nil {
		return fmt.Errorf("failed to register wallet settlement periodic task: %w", err)
	}

	logger.Info("Registered wallet settlement periodic task", "schedule", "00:01 daily")
	return nil
}

// RegisterStatusInquiryPoller registers the periodic status inquiry poller task.
// Runs every 2 minutes to poll stuck authorised payments via the provider's
// inquiry endpoint. Only effective when the active provider implements StatusPoller.
func RegisterStatusInquiryPoller(srv *Server) error {
	_, err := srv.Scheduler().Register("@every 2m", asynqlib.NewTask(TaskStatusInquiry, nil),
		asynqlib.Queue(QueueLow),
	)
	if err != nil {
		return fmt.Errorf("failed to register status inquiry poller: %w", err)
	}

	logger.Info("Registered status inquiry poller periodic task", "interval", "2m")
	return nil
}

// RegisterWalletBulkSweepers registers the two periodic sweepers for the
// wallet bulk transfer pipeline:
//   - stale-enqueue sweeper (@every 1m): re-enqueues per-row tasks for batches
//     whose outbox state is still 'pending' (worker crashed between batch
//     INSERT and enqueue).
//   - completing-batch recovery (@every 5m): re-attempts ledger booking for
//     batches stuck in 'completing' >10min.
//
// Both are idempotent — safe to run repeatedly.
func RegisterWalletBulkSweepers(srv *Server) error {
	if _, err := srv.Scheduler().Register("@every 1m",
		asynqlib.NewTask(TaskWalletStaleEnqueueSweeper, nil),
		asynqlib.Queue(QueueLow),
	); err != nil {
		return fmt.Errorf("register stale-enqueue sweeper: %w", err)
	}
	if _, err := srv.Scheduler().Register("@every 5m",
		asynqlib.NewTask(TaskWalletCompletingRecovery, nil),
		asynqlib.Queue(QueueLow),
	); err != nil {
		return fmt.Errorf("register completing-batch recovery: %w", err)
	}
	logger.Info("Registered wallet bulk transfer periodic sweepers",
		"intervals", "stale_enqueue=1m, completing_recovery=5m")
	return nil
}
