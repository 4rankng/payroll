package domain

import (
	"context"
	"time"
)

// AttendanceStatus represents the status of an attendance record
type AttendanceStatus string

const (
	AttendanceStatusCheckedIn AttendanceStatus = "checked_in"
	AttendanceStatusCompleted AttendanceStatus = "completed"
	AttendanceStatusOrphaned  AttendanceStatus = "orphaned"
	// AttendanceStatusRejected marks an attendance whose checkout window [K, K+1h)
	// closed with no checkout, so it was auto-rejected by the scheduled task
	// (earning 0, final). Derived from CheckOutTime==nil && SalaryRejectReason!=nil.
	AttendanceStatusRejected AttendanceStatus = "rejected"
)

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
	CheckOutLat        *float64   `json:"check_out_lat" gorm:"type:decimal(10,7)"`
	CheckOutLng        *float64   `json:"check_out_lng" gorm:"type:decimal(10,7)"`
	CheckInGate        string     `json:"check_in_gate" gorm:"type:varchar(50);not null"`
	CheckOutGate       *string    `json:"check_out_gate" gorm:"type:varchar(50)"`
	EarningAmount      *int64     `json:"earning_amount" gorm:"type:bigint;default:0"`
	SalaryRejectReason *string    `json:"salary_reject_reason" gorm:"type:text"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	// Relationships
	Employee Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
	Project  Project  `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
}

// AttendanceHealthStats holds conditional-aggregation counts for health monitoring.
type AttendanceHealthStats struct {
	OpenCheckedIn         int `json:"open_checked_in" gorm:"column:open_checked_in"`
	Orphaned              int `json:"orphaned" gorm:"column:orphaned"`
	AutoRejected          int `json:"auto_rejected" gorm:"column:auto_rejected"`
	CompletedZeroEarning  int `json:"completed_zero_earning" gorm:"column:completed_zero_earning"`
	SuccessfulCheckouts   int `json:"successful_checkouts" gorm:"column:successful_checkouts"`
}

// AttendanceRepository defines the interface for attendance persistence operations
type AttendanceRepository interface {
	Create(ctx context.Context, attendance *Attendance) error
	GetByID(ctx context.Context, id uint) (*Attendance, error)
	GetByEmployeeAndDate(ctx context.Context, employeeID uint, date time.Time) (*Attendance, error)
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
	// GetOrphanCandidates returns open (no checkout), unrejected attendances
	// checked in within [after, before). Used by the auto-reject fallback sweep
	// to finalize records whose scheduled K+1h task was lost.
	GetOrphanCandidates(ctx context.Context, after, before time.Time) ([]*Attendance, error)
	// GetHealthStats returns conditional-aggregation counts for the admin health
	// dashboard. The since/until window bounds check_in_time; the open/orphaned
	// split additionally uses an 18h cutoff against the current time.
	GetHealthStats(ctx context.Context, since, until time.Time) (*AttendanceHealthStats, error)
}

// AttendanceFilters represents filtering options for attendance queries
type AttendanceFilters struct {
	EmployeeID  *uint
	ProjectID   *uint
	FromDate    *time.Time
	ToDate      *time.Time
	Status      *AttendanceStatus
	ZeroEarning *bool // when true, filter completed attendances with earning_amount = 0 or NULL
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
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
	ErrorMessage   *string    `json:"error_message" gorm:"type:varchar(500)"`
	CreatedAt      time.Time  `json:"created_at"`
	Employee       Employee  `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
}

func (AttendanceFailedAttempt) TableName() string { return "attendance_failed_attempts" }

// FailedAttemptCategoryCount holds the count of failed attempts grouped by category.
type FailedAttemptCategoryCount struct {
	Category string `json:"category"`
	Count    int    `json:"count"`
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
	List(ctx context.Context, filters FailedAttemptFilters) ([]*AttendanceFailedAttempt, error)
	Count(ctx context.Context, filters FailedAttemptFilters) (int64, error)
	// GetCategoryCounts returns counts grouped by reason_category within the window.
	GetCategoryCounts(ctx context.Context, since, until time.Time) ([]FailedAttemptCategoryCount, error)
	// GetTotalCount returns the total count within the window.
	GetTotalCount(ctx context.Context, since, until time.Time) (int64, error)
}

// IsCompleted returns true if the attendance record has both check-in and check-out
func (a *Attendance) IsCompleted() bool {
	return a.CheckOutTime != nil
}

// GetStatus derives the status based on fields using the provided current time.
// Callers should pass clock.Now() (or s.clock.Now() for injectable clock) to
// respect the centralized clock convention used in tests and non-prod environments.
func (a *Attendance) GetStatus(now time.Time) AttendanceStatus {
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
