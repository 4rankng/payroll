package dto

import "time"

// CheckInRequest represents the request to check in.
// ProjectID is optional — when omitted, the server auto-detects the employee's
// active flexible project. When the employee belongs to multiple flexible
// projects, project_id must be provided.
type CheckInRequest struct {
	ProjectID uint    `json:"project_id"`
	Lat       float64 `json:"lat" binding:"required"`
	Lng       float64 `json:"lng" binding:"required"`
}

// CheckOutRequest represents the request to check out
type CheckOutRequest struct {
	Lat float64 `json:"lat" binding:"required"`
	Lng float64 `json:"lng" binding:"required"`
}

// AttendanceResponse represents an attendance record
type AttendanceResponse struct {
	ID                 uint       `json:"id"`
	ProjectID          uint       `json:"project_id"`
	EmployeeID         uint       `json:"employee_id"`
	Date               time.Time  `json:"date"`
	CheckInTime        time.Time  `json:"check_in_time"`
	CheckInGate        string     `json:"check_in_gate"`
	CheckOutTime       *time.Time `json:"check_out_time,omitempty"`
	CheckOutGate       *string    `json:"check_out_gate,omitempty"`
	EarningAmount      *int64     `json:"earning_amount,omitempty"`
	SalaryRejectReason *string    `json:"salary_reject_reason,omitempty"`
	SalaryStatus       string     `json:"salary_status"`
	SalaryMessage      string     `json:"salary_message"`
	Status             string     `json:"status"`
}

// AdminAttendanceResponse represents the detailed attendance record for admin view
type AdminAttendanceResponse struct {
	ID                 uint       `json:"id"`
	ProjectID          uint       `json:"project_id"`
	ProjectName        string     `json:"project_name"`
	EmployeeID         uint       `json:"employee_id"`
	EmployeeName       string     `json:"employee_name"`
	Date               time.Time  `json:"date"`
	CheckInTime        time.Time  `json:"check_in_time"`
	CheckInGate        string     `json:"check_in_gate"`
	CheckOutTime       *time.Time `json:"check_out_time,omitempty"`
	CheckOutGate       *string    `json:"check_out_gate,omitempty"`
	EarningAmount      *int64     `json:"earning_amount,omitempty"`
	SalaryRejectReason *string    `json:"salary_reject_reason,omitempty"`
	Status             string     `json:"status"`
}

// PaginatedAttendanceResponse represents a paginated list of attendances
type PaginatedAttendanceResponse struct {
	Data  []AdminAttendanceResponse `json:"data"`
	Total int64                     `json:"total"`
}
