package attendance

import (
	"context"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

const confirmedNoSalaryCheckoutReason = "Nhân viên đã xác nhận tan ca không ghi nhận tiền lương cho ca này."
const employeeCancelledWrongShiftReason = "Nhân viên đã hủy ca do vào nhầm ca."

// TaskEnqueuer schedules deferred attendance tasks. Implemented by the asynq
// client wrapper; fakes capture the calls in tests. Nil is allowed — when unset,
// CheckIn/CheckOut skip scheduling (used in lightweight tests).
type TaskEnqueuer interface {
	EnqueueAutoRejectCheckout(attendanceID uint, at time.Time) error
	// EnqueueCreditQuota schedules the deferred quota-credit task after the
	// Admin-configured post-checkout hold. The worker is idempotent (guards on
	// quota_credited_at).
	EnqueueCreditQuota(attendanceID uint, at time.Time) error
}

type selfCheckInAdvanceHoldDurationProvider interface {
	GetSelfCheckInAdvanceHoldDuration(ctx context.Context) time.Duration
}

// AttendanceService orchestrates attendance check-in/out, admin review, quota
// crediting, and attendance reads. Its methods are split across sibling files by
// concern: attendance_checkin.go, attendance_checkout.go, attendance_review.go,
// attendance_quota.go, attendance_query.go, attendance_shift.go, and
// attendance_geofence.go.
type AttendanceService struct {
	attendanceRepo      domain.AttendanceRepository
	projectEmployeeRepo domain.ProjectEmployeeRepository
	projectRepo         domain.ProjectRepository
	payrateRepo         domain.PayrateRepository
	advancePaymentRepo  domain.AdvancePaymentRepository
	settingsConfig      interface {
		GetSelfCheckInAdvancePercentageForUpdate(ctx context.Context) (uint64, error)
	}
	transactionManager domain.TransactionManager
	taskEnqueuer       TaskEnqueuer
	clock              clock.Clock
}

func NewAttendanceService(
	attendanceRepo domain.AttendanceRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	projectRepo domain.ProjectRepository,
	payrateRepo domain.PayrateRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	settingsConfig interface {
		GetSelfCheckInAdvancePercentageForUpdate(ctx context.Context) (uint64, error)
	},
	transactionManager domain.TransactionManager,
	taskEnqueuer TaskEnqueuer,
	clk clock.Clock,
) *AttendanceService {
	if clk == nil {
		clk = clock.New()
	}
	return &AttendanceService{
		attendanceRepo:      attendanceRepo,
		projectEmployeeRepo: projectEmployeeRepo,
		projectRepo:         projectRepo,
		payrateRepo:         payrateRepo,
		advancePaymentRepo:  advancePaymentRepo,
		settingsConfig:      settingsConfig,
		transactionManager:  transactionManager,
		taskEnqueuer:        taskEnqueuer,
		clock:               clk,
	}
}

func isConfirmedNoSalaryCheckout(att *domain.Attendance) bool {
	return att != nil &&
		att.CheckOutTime != nil &&
		att.EarningAmount != nil &&
		*att.EarningAmount == 0 &&
		att.SalaryRejectReason != nil &&
		strings.Contains(*att.SalaryRejectReason, confirmedNoSalaryCheckoutReason)
}

func isAutoRejectedNoCheckout(att *domain.Attendance) bool {
	return att != nil && att.CheckOutTime == nil && att.SalaryRejectReason != nil
}

// parsedShift is a single configured shift for a position, resolved to absolute
// datetimes (night-shift / cross-midnight aware, anchored to the check-in day or
// one of its ±1 neighbors) together with its rate.
type parsedShift struct {
	start  time.Time
	end    time.Time
	amount int
}
