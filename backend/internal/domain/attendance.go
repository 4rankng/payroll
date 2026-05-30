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
)

// Attendance represents a flexi employee check-in/out record
type Attendance struct {
	ID            uint       `json:"id" gorm:"primarykey;type:bigint unsigned"`
	EmployeeID    uint       `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	ProjectID     uint       `json:"project_id" gorm:"not null;type:bigint unsigned"`
	Date          time.Time  `json:"date" gorm:"type:date;not null"`
	CheckInTime   time.Time  `json:"check_in_time" gorm:"type:datetime(3);not null"`
	CheckOutTime  *time.Time `json:"check_out_time" gorm:"type:datetime(3)"`
	CheckInLat    float64    `json:"check_in_lat" gorm:"type:decimal(10,7);not null"`
	CheckInLng    float64    `json:"check_in_lng" gorm:"type:decimal(10,7);not null"`
	CheckOutLat   *float64   `json:"check_out_lat" gorm:"type:decimal(10,7)"`
	CheckOutLng   *float64   `json:"check_out_lng" gorm:"type:decimal(10,7)"`
	CheckInGate   string     `json:"check_in_gate" gorm:"type:varchar(50);not null"`
	CheckOutGate  *string    `json:"check_out_gate" gorm:"type:varchar(50)"`
	EarningAmount *int64     `json:"earning_amount" gorm:"type:bigint;default:0"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`

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

	// If check-in time is older than 18 hours, it's considered orphaned
	if now.Sub(a.CheckInTime) > 18*time.Hour {
		return AttendanceStatusOrphaned
	}

	return AttendanceStatusCheckedIn
}
