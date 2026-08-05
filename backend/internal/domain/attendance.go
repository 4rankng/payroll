package domain

import (
	"context"
	"time"
)

// GeoReading bundles the device-reported GPS fields for a single check-in or
// check-out attempt. Using a struct instead of positional lat/lng/accuracy
// keeps CheckIn/CheckOut signatures stable as new fields are added.
type GeoReading struct {
	Lat, Lng float64
	Accuracy float64    // meters; 0 when unknown (device had no fix)
	GpsAt    *time.Time // device fix timestamp; nil when unknown
}

// AccuracyPtr returns a pointer to Accuracy, or nil if Accuracy <= 0.
// Useful when the GORM column is nullable and zero means "unknown".
func (g GeoReading) AccuracyPtr() *float64 {
	if g.Accuracy > 0 {
		return &g.Accuracy
	}
	return nil
}

// AttendanceStatus represents the status of an attendance record
type AttendanceStatus string

const (
	AttendanceStatusCheckedIn AttendanceStatus = "checked_in"
	AttendanceStatusCompleted AttendanceStatus = "completed"
	AttendanceStatusOrphaned  AttendanceStatus = "orphaned"
	// AttendanceStatusRejected marks an attendance whose checkout window [K-1h, K+4h]
	// closed with no checkout, so it was auto-rejected by the scheduled task
	// (earning 0, final). Derived from CheckOutTime==nil && SalaryRejectReason!=nil.
	AttendanceStatusRejected AttendanceStatus = "rejected"
)

// Attendance review actions set by an admin during dispute resolution.
const (
	AttendanceReviewActionApproved AttendanceReviewAction = "approved"
	AttendanceReviewActionRejected AttendanceReviewAction = "rejected"
)

// AttendanceReviewAction is the value stored in Attendance.ReviewAction.
type AttendanceReviewAction string

// Attendance represents a flexi employee check-in/out record
type Attendance struct {
	ID                 uint       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	EmployeeID         uint       `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	ProjectID          uint       `json:"project_id" gorm:"not null;type:bigint unsigned"`
	Date               time.Time  `json:"date" gorm:"type:date;not null"`
	CheckInTime        time.Time  `json:"check_in_time" gorm:"type:datetime(3);not null"`
	CheckOutTime       *time.Time `json:"check_out_time" gorm:"type:datetime(3)"`
	CheckInLat         float64    `json:"check_in_lat" gorm:"type:decimal(10,7);not null"`
	CheckInLng         float64    `json:"check_in_lng" gorm:"type:decimal(10,7);not null"`
	CheckInAccuracy    *float64   `json:"check_in_accuracy" gorm:"type:float"`
	CheckInGpsAt       *time.Time `json:"check_in_gps_at" gorm:"type:datetime(3)"`
	CheckOutLat        *float64   `json:"check_out_lat" gorm:"type:decimal(10,7)"`
	CheckOutLng        *float64   `json:"check_out_lng" gorm:"type:decimal(10,7)"`
	CheckOutAccuracy   *float64   `json:"check_out_accuracy" gorm:"type:float"`
	CheckOutGpsAt      *time.Time `json:"check_out_gps_at" gorm:"type:datetime(3)"`
	CheckInGate        string     `json:"check_in_gate" gorm:"type:varchar(50);not null"`
	CheckOutGate       *string    `json:"check_out_gate" gorm:"type:varchar(50)"`
	EarningAmount      *int64     `json:"earning_amount" gorm:"type:bigint;default:0"`
	SalaryRejectReason *string    `json:"salary_reject_reason" gorm:"type:text"`
	// QuotaCreditedAt marks when this attendance's earning was banked into the
	// advance-payment quota pool (advance_payments.salary/max_adv_amount). NULL
	// means the earning is held / pending its 24h credit window; non-NULL means
	// it has been credited and must not be banked again. The deferred credit task
	// and the safety-net sweep gate on this — it is the idempotency key.
	QuotaCreditedAt *time.Time `json:"quota_credited_at" gorm:"type:datetime(3)"`
	// Admin review audit (migration 083): populated when an admin manually
	// approves/rejects a disputed attendance via /admin/attendances/:id/approve|reject.
	// review_action is "approved" | "rejected"; a nil pointer means "never reviewed"
	// and the row keeps its system-derived GetStatus(). Mirrors the resolution-columns
	// convention on AttendanceFailedAttempt.
	ReviewAction *string    `json:"review_action" gorm:"type:varchar(16)"`
	ReviewNote   *string    `json:"review_note" gorm:"type:text"`
	ReviewedBy   *uint      `json:"reviewed_by" gorm:"type:bigint unsigned"`
	ReviewedAt   *time.Time `json:"reviewed_at" gorm:"type:datetime(3)"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// Relationships
	Employee Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
	Project  Project  `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
}

// AttendanceHealthStats holds conditional-aggregation counts for health monitoring.
type AttendanceHealthStats struct {
	OpenCheckedIn        int `json:"open_checked_in" gorm:"column:open_checked_in"`
	Orphaned             int `json:"orphaned" gorm:"column:orphaned"`
	AutoRejected         int `json:"auto_rejected" gorm:"column:auto_rejected"`
	CompletedZeroEarning int `json:"completed_zero_earning" gorm:"column:completed_zero_earning"`
	SuccessfulCheckouts  int `json:"successful_checkouts" gorm:"column:successful_checkouts"`
}

// AttendanceRepository defines the interface for attendance persistence operations
type AttendanceRepository interface {
	Create(ctx context.Context, attendance *Attendance) error
	GetByID(ctx context.Context, id uint) (*Attendance, error)
	GetByEmployeeAndDate(ctx context.Context, employeeID uint, date time.Time) (*Attendance, error)
	// GetApprovedOpenBefore returns the newest admin-approved attendance which
	// still has no persisted checkout before the supplied business day. Legacy
	// versions represented an approval as completed in status only, leaving such
	// rows open to the database's one-open-attendance guard.
	GetApprovedOpenBefore(ctx context.Context, employeeID uint, before time.Time) (*Attendance, error)
	// CompleteApprovedOpen atomically persists the authoritative checkout for a
	// previously approved but still-open attendance. It preserves the existing
	// earning and review audit data, returning false if another writer won.
	CompleteApprovedOpen(ctx context.Context, id uint, checkOutTime time.Time, checkOutGate string) (bool, error)
	Update(ctx context.Context, attendance *Attendance) error
	List(ctx context.Context, filters AttendanceFilters) ([]*Attendance, error)
	Count(ctx context.Context, filters AttendanceFilters) (int64, error)
	// MarkAutoRejected atomically finalizes an attendance whose checkout window
	// expired: it sets earning_amount=0 and salary_reject_reason=reason ONLY if the
	// row still has no checkout and no existing reject reason. The conditional
	// WHERE makes it race-free against a concurrent CheckOut (which would otherwise
	// be clobbered by a full-row Save of a stale, no-checkout snapshot). Returns
	// true if the row was updated, false if a checkout/prior rejection beat it
	// (both safe no-ops) or the id does not exist.
	MarkAutoRejected(ctx context.Context, id uint, reason string) (bool, error)
	// MarkAdminReviewed stamps an admin approve/reject review on an attendance
	// and applies the earning override atomically via a conditional UPDATE.
	// Rejection is accepted only before approval or quota credit. On approve,
	// earningAmount is set and salary_reject_reason is cleared; on reject,
	// earning is forced to 0 and salary_reject_reason is set to note. On approve,
	// checkOutTime/checkOutGate (when non-nil) close the shift so the record is
	// genuinely completed — not just faked-completed by IsApproved(). Reject
	// passes nil,nil (a rejection does not check out). Returns true if the
	// transition was applied, false if the row was missing or a terminal
	// financial state won the race.
	MarkAdminReviewed(ctx context.Context, id uint, action AttendanceReviewAction, note string, adminID uint, reviewedAt time.Time, earningAmount *int64, checkOutTime *time.Time, checkOutGate *string) (bool, error)
	// GetOrphanCandidates returns open (no checkout), unrejected attendances
	// checked in within [after, before). Used by the auto-reject fallback sweep
	// to finalize records whose scheduled K+4h task was lost.
	GetOrphanCandidates(ctx context.Context, after, before time.Time) ([]*Attendance, error)
	// GetHealthStats returns conditional-aggregation counts for the admin health
	// dashboard. The since/until window bounds check_in_time; the open/orphaned
	// split additionally uses an 18h cutoff against the current time.
	GetHealthStats(ctx context.Context, since, until time.Time) (*AttendanceHealthStats, error)
	// MarkQuotaCredited atomically stamps quota_credited_at = at on an attendance
	// whose earning is still payable. The conditional WHERE makes crediting
	// idempotent and race-safe against admin rejection. Returns true only if this
	// call claimed the credit.
	MarkQuotaCredited(ctx context.Context, id uint, at time.Time) (bool, error)
	// GetOverdueQuotaCreditCandidates returns IDs of checked-out attendances whose
	// 24h hold has elapsed (check_out_time < before) but whose earning has not yet
	// been banked (quota_credited_at IS NULL, earning_amount > 0). Used by the
	// safety-net sweep to finalize credits the per-attendance deferred task missed.
	GetOverdueQuotaCreditCandidates(ctx context.Context, before time.Time, limit int) ([]uint, error)
}

// AttendanceFilters represents filtering options for attendance queries
type AttendanceFilters struct {
	EmployeeID           *uint
	ProjectID            *uint
	FromDate             *time.Time
	ToDate               *time.Time
	Status               *AttendanceStatus
	SuccessfulCheckout   *bool // when true, filter completed attendances with earning_amount > 0
	ZeroEarning          *bool // when true, filter completed attendances with earning_amount = 0 or NULL
	UseCheckInTimeWindow bool
	Limit                int
	Offset               int
	SortBy               string
	SortOrder            string
}

// AttendanceFailedAttempt records a check-in or check-out attempt that
// returned a validation error (HTTP 400). It is persisted by the handler
// layer so the admin dashboard can surface aggregate failure counts by
// reason category.
type AttendanceFailedAttempt struct {
	ID             uint       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	EmployeeID     uint       `json:"employee_id" gorm:"not null;type:bigint unsigned;index"`
	AttemptType    string     `json:"attempt_type" gorm:"type:varchar(16);not null"`
	ReasonCategory string     `json:"reason_category" gorm:"type:varchar(48);not null"`
	ProjectID      uint       `json:"project_id" gorm:"not null;type:bigint unsigned;default:0"`
	Lat            *float64   `json:"lat" gorm:"type:decimal(10,7)"`
	Lng            *float64   `json:"lng" gorm:"type:decimal(10,7)"`
	Accuracy       *float64   `json:"accuracy" gorm:"type:float"`
	GpsAt          *time.Time `json:"gps_at" gorm:"type:datetime(3)"`
	ErrorMessage   *string    `json:"error_message" gorm:"type:varchar(500)"`
	// Resolution audit: populated when an admin overrides a device-GPS failure
	// (reason_category gps_*) and records the check-in manually. Nil while the
	// attempt is still unresolved.
	ResolvedAt     *time.Time `json:"resolved_at" gorm:"type:datetime(3)"`
	ResolvedBy     *uint      `json:"resolved_by" gorm:"type:bigint unsigned"`
	ResolvedReason *string    `json:"resolved_reason" gorm:"type:varchar(255)"`
	CreatedAt      time.Time  `json:"created_at"`
	Employee       Employee   `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
}

func (AttendanceFailedAttempt) TableName() string { return "attendance_failed_attempts" }

// IsResolved reports whether an admin has already overridden this failed
// attempt to record the check-in manually. Used to keep the override idempotent.
func (a AttendanceFailedAttempt) IsResolved() bool {
	return a.ResolvedAt != nil
}

// FailedAttemptCategoryCount holds the count of failed attempts grouped by category.
type FailedAttemptCategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// FailedAttemptTypeCount holds the count of failed attempts grouped by attempt type.
type FailedAttemptTypeCount struct {
	AttemptType string `json:"attempt_type"`
	Count       int    `json:"count"`
}

// FailedAttemptFilters represents filtering options for failed-attempt queries.
type FailedAttemptFilters struct {
	AttemptType    *string
	ReasonCategory *string
	FromDate       *time.Time
	ToDate         *time.Time
	EmployeeID     *uint
	Limit          int
	Offset         int
}

// AttendanceFailedAttemptRepository defines the interface for failed-attempt persistence.
type AttendanceFailedAttemptRepository interface {
	Create(ctx context.Context, attempt *AttendanceFailedAttempt) error
	// GetByID loads a single failed attempt by primary key with Employee preloaded.
	// Returns a domain NotFound error when the row does not exist.
	GetByID(ctx context.Context, id uint) (*AttendanceFailedAttempt, error)
	// Update saves all fields of the attempt, used to stamp resolution audit.
	Update(ctx context.Context, attempt *AttendanceFailedAttempt) error
	List(ctx context.Context, filters FailedAttemptFilters) ([]*AttendanceFailedAttempt, error)
	Count(ctx context.Context, filters FailedAttemptFilters) (int64, error)
	// GetCategoryCounts returns counts grouped by reason_category within the window.
	GetCategoryCounts(ctx context.Context, since, until time.Time) ([]FailedAttemptCategoryCount, error)
	// GetCountByAttemptType returns counts grouped by attempt_type within the window.
	GetCountByAttemptType(ctx context.Context, since, until time.Time) ([]FailedAttemptTypeCount, error)
	// ExistsRecent reports whether a failed attempt with the same employee,
	// attempt type, reason category, and project was recorded within the last
	// `within`. Used to collapse rapid retries of the same failure into a single
	// forensic record so the dashboard is not flooded with one row per tap.
	ExistsRecent(ctx context.Context, employeeID uint, attemptType, reasonCategory string, projectID uint, within time.Duration) (bool, error)
}

// IsCompleted returns true when the employee checked out or an Admin
// authoritatively approved the shift without a physical checkout.
func (a *Attendance) IsCompleted() bool {
	return a.CheckOutTime != nil || (a.IsApproved() && a.SalaryRejectReason == nil)
}

// IsApproved reports whether an admin has manually approved this attendance.
func (a *Attendance) IsApproved() bool {
	return a.ReviewAction != nil && *a.ReviewAction == string(AttendanceReviewActionApproved)
}

// IsRejectedByAdmin reports whether an admin has manually rejected this attendance.
// Named to distinguish from the system auto-reject (SalaryRejectReason) path.
func (a *Attendance) IsRejectedByAdmin() bool {
	return a.ReviewAction != nil && *a.ReviewAction == string(AttendanceReviewActionRejected)
}

// IsReviewed reports whether any admin review (approve or reject) has occurred.
func (a *Attendance) IsReviewed() bool {
	return a.ReviewAction != nil
}

// GetStatus derives the status based on fields using the provided current time.
// Callers should pass clock.Now() (or s.clock.Now() for injectable clock) to
// respect the centralized clock convention used in tests and non-prod environments.
func (a *Attendance) GetStatus(now time.Time) AttendanceStatus {
	// An Admin rejection is authoritative even when a physical checkout exists.
	if a.IsRejectedByAdmin() {
		return AttendanceStatusRejected
	}

	// An Admin approval is the authoritative completion of the shift even when
	// the employee did not record a physical checkout. Keep contradictory legacy
	// rows with a reject reason visible as rejected so they can still be repaired.
	if a.IsApproved() && a.SalaryRejectReason == nil {
		return AttendanceStatusCompleted
	}

	if a.CheckOutTime != nil {
		return AttendanceStatusCompleted
	}

	// Auto-rejected by the checkout-window task: no checkout and a persisted
	// reject reason (set together with EarningAmount=0). Checked before the
	// 18h orphaned fallback so a final rejection always reads as rejected.
	// SalaryRejectReason is now written by TWO paths — the auto-reject task on a
	// no-checkout record (which reaches here), and the confirmed-no-salary
	// checkout override (which also sets CheckOutTime, so it returns completed
	// above and never reaches this branch). The CheckOutTime check above is what
	// keeps the two cases distinct; do not reorder these guards.
	if a.SalaryRejectReason != nil {
		return AttendanceStatusRejected
	}

	// If check-in time is older than 18 hours, it's considered orphaned
	if now.Sub(a.CheckInTime) > 18*time.Hour {
		return AttendanceStatusOrphaned
	}

	return AttendanceStatusCheckedIn
}
