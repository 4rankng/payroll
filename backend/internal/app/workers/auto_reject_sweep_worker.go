package workers

import (
	"context"
	"log/slog"
)

// AttendanceAutoRejectSweeper finalizes attendance records whose checkout window
// expired but were never auto-rejected (e.g. the scheduled task was lost due to a
// Redis/process outage at check-in). Implemented by *attendance.AttendanceService.
type AttendanceAutoRejectSweeper interface {
	AutoRejectSweep(ctx context.Context) (int, error)
}

// AutoRejectSweepWorker is the periodic backstop for the per-attendance
// auto-reject task: it runs AutoRejectSweep on a schedule to catch records the
// scheduled K+1h task missed. The sweeper is idempotent, so asynq retries and
// overlapping runs are safe.
type AutoRejectSweepWorker struct {
	sweeper AttendanceAutoRejectSweeper
	logger  *slog.Logger
}

// NewAutoRejectSweepWorker creates a new AutoRejectSweepWorker.
func NewAutoRejectSweepWorker(sweeper AttendanceAutoRejectSweeper) *AutoRejectSweepWorker {
	return &AutoRejectSweepWorker{
		sweeper: sweeper,
		logger:  slog.Default().With("component", "AutoRejectSweepWorker"),
	}
}

// ProcessJob runs one sweep pass. Errors are retried by asynq; the sweeper is
// idempotent so retries are safe.
func (w *AutoRejectSweepWorker) ProcessJob(ctx context.Context) error {
	rejected, err := w.sweeper.AutoRejectSweep(ctx)
	if err != nil {
		w.logger.Error("auto-reject sweep failed", "error", err)
		return err
	}
	if rejected > 0 {
		w.logger.Info("auto-reject sweep finalized attendances", "count", rejected)
	}
	return nil
}
