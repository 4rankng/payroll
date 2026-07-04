package workers

import (
	"context"
	"log/slog"
)

// AttendanceQuotaCreditr banks an attendance's earning into the advance-payment
// quota pool. Implemented by *attendance.AttendanceService; defined here
// (consumer-side) so the workers package does not import the service package.
type AttendanceQuotaCreditr interface {
	CreditAttendanceQuota(ctx context.Context, attendanceID uint) (bool, error)
}

// CreditQuotaWorker banks a single attendance's earning into the quota pool. It
// is the bridge between the attendance:credit_quota asynq task (scheduled at
// checkOutTime + QuotaCreditHoldDuration when the employee checks out) and the
// attendance service. The service's CreditAttendanceQuota is idempotent on
// quota_credited_at, so asynq retries and duplicate tasks are safe.
type CreditQuotaWorker struct {
	creditr AttendanceQuotaCreditr
	logger  *slog.Logger
}

// NewCreditQuotaWorker creates a new CreditQuotaWorker.
func NewCreditQuotaWorker(creditr AttendanceQuotaCreditr) *CreditQuotaWorker {
	return &CreditQuotaWorker{
		creditr: creditr,
		logger:  slog.Default().With("component", "CreditQuotaWorker"),
	}
}

// ProcessJob banks the attendance's earning if its 24h hold has elapsed. Returns
// nil for already-credited / nothing-to-credit records (idempotent), so duplicate
// or retried tasks are safe.
func (w *CreditQuotaWorker) ProcessJob(ctx context.Context, attendanceID uint) error {
	banked, err := w.creditr.CreditAttendanceQuota(ctx, attendanceID)
	if err != nil {
		w.logger.Error("quota credit failed",
			"attendance_id", attendanceID, "error", err)
		return err
	}
	if banked {
		w.logger.Info("banked attendance earning into quota pool",
			"attendance_id", attendanceID)
	}
	return nil
}

// AttendanceQuotaCreditSweeper is the safety-net backstop for the per-attendance
// credit task: it banks earnings whose scheduled task was lost. Implemented by
// *attendance.AttendanceService.
type AttendanceQuotaCreditSweeper interface {
	CreditOverduePendingQuota(ctx context.Context) (int, error)
}

// CreditQuotaSweepWorker is the periodic backstop for the per-attendance credit
// task: it runs CreditOverduePendingQuota on a schedule to catch earnings the
// scheduled 24h task missed. The sweeper is idempotent, so asynq retries and
// overlapping runs are safe.
type CreditQuotaSweepWorker struct {
	sweeper AttendanceQuotaCreditSweeper
	logger  *slog.Logger
}

// NewCreditQuotaSweepWorker creates a new CreditQuotaSweepWorker.
func NewCreditQuotaSweepWorker(sweeper AttendanceQuotaCreditSweeper) *CreditQuotaSweepWorker {
	return &CreditQuotaSweepWorker{
		sweeper: sweeper,
		logger:  slog.Default().With("component", "CreditQuotaSweepWorker"),
	}
}

// ProcessJob runs one sweep pass. Errors are retried by asynq; the sweeper is
// idempotent so retries are safe.
func (w *CreditQuotaSweepWorker) ProcessJob(ctx context.Context) error {
	credited, err := w.sweeper.CreditOverduePendingQuota(ctx)
	if err != nil {
		w.logger.Error("quota-credit sweep failed", "error", err)
		return err
	}
	if credited > 0 {
		w.logger.Info("quota-credit sweep banked pending earnings", "count", credited)
	}
	return nil
}
