package workers

import (
	"context"
	"log/slog"
)

// AttendanceAutoRejecter rejects an attendance whose checkout window [K, K+3h)
// closed with no checkout. Implemented by *attendance.AttendanceService; defined
// here (consumer-side) so the workers package does not import the service
// package.
type AttendanceAutoRejecter interface {
	AutoRejectIfExpired(ctx context.Context, attendanceID uint) error
}

// AutoRejectCheckoutWorker finalizes attendances whose checkout window expired.
// It is the bridge between the attendance:auto_reject asynq task (scheduled at
// K+3h when the employee checks in) and the attendance service. The service's
// AutoRejectIfExpired is idempotent, so asynq retries are safe.
type AutoRejectCheckoutWorker struct {
	rejecter AttendanceAutoRejecter
	logger   *slog.Logger
}

// NewAutoRejectCheckoutWorker creates a new AutoRejectCheckoutWorker.
func NewAutoRejectCheckoutWorker(rejecter AttendanceAutoRejecter) *AutoRejectCheckoutWorker {
	return &AutoRejectCheckoutWorker{
		rejecter: rejecter,
		logger:   slog.Default().With("component", "AutoRejectCheckoutWorker"),
	}
}

// ProcessJob rejects the attendance with the given ID if its checkout window
// has expired. Returns nil for already-completed / already-rejected records
// (idempotent), so duplicate or retried tasks are safe.
func (w *AutoRejectCheckoutWorker) ProcessJob(ctx context.Context, attendanceID uint) error {
	if err := w.rejecter.AutoRejectIfExpired(ctx, attendanceID); err != nil {
		w.logger.Error("auto-reject checkout failed",
			"attendance_id", attendanceID, "error", err)
		return err
	}
	return nil
}
