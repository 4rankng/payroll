package dto

import (
	"time"
)

// CreateTimesheetEditRequestRequest represents the request to create a new edit request
// No body needed - timesheet ID comes from URL
type CreateTimesheetEditRequestRequest struct {
	// Empty struct - all data comes from URL params and user context
}

// TimesheetEditRequestResponse represents a timesheet edit request with details
type TimesheetEditRequestResponse struct {
	ID              uint                  `json:"id"`
	TimesheetID     uint                  `json:"timesheet_id"`
	Status          string                `json:"status"`
	RequestedBy     uint                  `json:"requested_by"`
	RequestedByName string                `json:"requested_by_name"`
	ApprovedBy      *uint                 `json:"approved_by"`
	ApprovedByName  *string               `json:"approved_by_name"`
	RejectedBy      *uint                 `json:"rejected_by"`
	RejectedByName  *string               `json:"rejected_by_name"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Timesheet       *TimesheetSummaryInfo `json:"timesheet,omitempty"`
}

// TimesheetSummaryInfo represents basic timesheet information for edit requests
type TimesheetSummaryInfo struct {
	ID            uint    `json:"id"`
	Date          string  `json:"date"`
	EmployeeID    uint    `json:"employee_id"`
	EmployeeName  string  `json:"employee_name"`
	EmployeeCCCD  string  `json:"employee_cccd"`
	ProjectID     uint    `json:"project_id"`
	ProjectName   string  `json:"project_name"`
	Paytype       string  `json:"paytype"`
	HoursWorked   float64 `json:"hours_worked"`
	Amount        int64   `json:"amount"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"payment_status"`
}

// ListTimesheetEditRequestsResponse represents the response for listing edit requests
type ListTimesheetEditRequestsResponse struct {
	EditRequests []TimesheetEditRequestResponse `json:"edit_requests"`
	Pagination   PaginationResponse             `json:"pagination"`
}
