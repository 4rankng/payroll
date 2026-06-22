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
}

// AttendanceFilters represents filtering options for attendance queries
type AttendanceFilters struct {
	EmployeeID *uint
	ProjectID  *uint
	FromDate   *time.Time
	ToDate     *time.Time
	Status     *AttendanceStatus
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
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
	// This discriminator is safe because SalaryRejectReason is only ever set on
	// a no-checkout record by the auto-reject path — completed-but-unpaid
	// records carry a checkout and return completed above.
	if a.SalaryRejectReason != nil {
		return AttendanceStatusRejected
	}

	// If check-in time is older than 18 hours, it's considered orphaned
	if now.Sub(a.CheckInTime) > 18*time.Hour {
		return AttendanceStatusOrphaned
	}

	return AttendanceStatusCheckedIn
}
