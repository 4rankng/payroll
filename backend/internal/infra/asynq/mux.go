package asynq

import (
	"fmt"

	asynqlib "github.com/hibiken/asynq"
)

// RegisterHandlers registers all task handlers on the server's mux
func RegisterHandlers(srv *Server, h *Handlers) {
	srv.Mux().Handle(TaskEmployeeImport, asynqlib.HandlerFunc(h.HandleEmployeeImport))
	srv.Mux().Handle(TaskImportJob, asynqlib.HandlerFunc(h.HandleImportJob))
	srv.Mux().Handle(TaskIPNProcess, asynqlib.HandlerFunc(h.HandleIPNProcess))
	srv.Mux().Handle(TaskBulkTransferTransaction, asynqlib.HandlerFunc(h.HandleBulkTransferTransaction))
	srv.Mux().Handle(TaskBulkTransferPayment, asynqlib.HandlerFunc(h.HandleBulkTransferPayment))
	srv.Mux().Handle(TaskAuditLogWrite, asynqlib.HandlerFunc(h.HandleAuditLogWrite))

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

	registered := []string{
		TaskEmployeeImport, TaskImportJob, TaskIPNProcess,
		TaskBulkTransferTransaction, TaskBulkTransferPayment,
		TaskAuditLogWrite,
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

	logger.Info("Registered asynq task handlers", "tasks", registered)
}

// RegisterPeriodicTasks registers periodic tasks on the scheduler
func RegisterPeriodicTasks(srv *Server, _ *Client) error {
	logger.Info("Registered asynq periodic tasks")
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
// Runs daily at 00:01 Vietnam time.
func RegisterWalletSettlement(srv *Server) error {
	_, err := srv.Scheduler().Register("1 0 * * *", asynqlib.NewTask(TaskWalletSettlement, nil),
		asynqlib.Queue(QueueLow),
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
