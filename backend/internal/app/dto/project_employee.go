package dto

import (
	"time"

	"api-server/internal/domain"
)

type CheckInConfigurableProjectResponse struct {
	ID         uint                 `json:"id"`
	Name       string               `json:"name"`
	Code       string               `json:"code"`
	Status     domain.ProjectStatus `json:"status"`
	IsFlexible bool                 `json:"is_flexible"`
}

// CreateProjectEmployeeRequest represents the request to assign an employee to a project
type CreateProjectEmployeeRequest struct {
	EmployeeID      uint    `json:"employee_id" binding:"required"`
	EmployeeCode    string  `json:"employee_code,omitempty"`
	Position        string  `json:"position" binding:"required"`
	StartDate       *string `json:"start_date,omitempty"`
	LastDate        *string `json:"last_date,omitempty"`
	PaymentSchedule *string `json:"payment_schedule,omitempty" binding:"omitempty,oneof=weekly monthly flexible"` // Default: "weekly"
}

// BatchAssignEmployeesRequest represents the request to assign employees to a project (batch)
type BatchAssignEmployeesRequest []AssignEmployeeToProjectRequest

// AssignEmployeeToProjectRequest represents a single employee assignment in batch operation
type AssignEmployeeToProjectRequest struct {
	EmployeeID      uint    `json:"employee_id" binding:"required"`
	EmployeeCode    string  `json:"employee_code,omitempty"`
	Position        string  `json:"position" binding:"required"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
	PaymentSchedule *string `json:"payment_schedule,omitempty" binding:"omitempty,oneof=weekly monthly flexible"` // Default: "weekly"
}

// UpdateProjectEmployeeRequest represents the request to update an employee assignment
type UpdateProjectEmployeeRequest struct {
	EmployeeCode    *string `json:"employee_code,omitempty"`
	Position        *string `json:"position,omitempty"`
	StartDate       *string `json:"start_date,omitempty"`
	LastDate        *string `json:"last_date,omitempty"`
	PaymentSchedule *string `json:"payment_schedule,omitempty" binding:"omitempty,oneof=weekly monthly flexible"`
}

// RemoveEmployeeFromProjectRequest represents the request to remove an employee from a project
type RemoveEmployeeFromProjectRequest struct {
	EmployeeID uint    `json:"employee_id" binding:"required"`
	LastDate   *string `json:"last_date,omitempty"`
}

// BatchRemoveEmployeesRequest represents the request to remove employees from a project (batch)
type BatchRemoveEmployeesRequest []RemoveEmployeeFromProjectRequest

// UpdateAssignmentRequest represents the request to update a project-employee assignment
// Used by both PUT /projects/:id/employees and PUT /employees/:id/projects endpoints
type UpdateAssignmentRequest struct {
	ProjectID       uint    `json:"project_id,omitempty"`
	EmployeeID      uint    `json:"employee_id,omitempty"`
	EmployeeCode    *string `json:"employee_code,omitempty"`
	Position        *string `json:"position,omitempty"`
	StartDate       *string `json:"start_date,omitempty"`
	EndDate         *string `json:"end_date,omitempty"`
	PaymentSchedule *string `json:"payment_schedule,omitempty" binding:"omitempty,oneof=weekly monthly flexible"`
}

// EmployeeInfo represents employee information for project employee responses
type EmployeeInfo struct {
	ID       uint   `json:"id"`
	Fullname string `json:"fullname"`
	Email    string `json:"email"`
	CCCD     string `json:"cccd"`
	Mobile   string `json:"mobile"`
}

// ProjectEmployeeAssignmentResponse represents the response for assign/remove operations
// Follows API spec - does not include employee_name and employee_cccd
type ProjectEmployeeAssignmentResponse struct {
	ID                     uint       `json:"id"`
	ProjectID              uint       `json:"project_id"`
	EmployeeID             uint       `json:"employee_id"`
	EmployeeCode           string     `json:"employee_code"`
	Position               string     `json:"position"`
	StartDate              time.Time  `json:"start_date"`
	LastDate               *time.Time `json:"last_date"`
	PaymentSchedule        string     `json:"payment_schedule"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty"`
	CheckInEnabled         bool       `json:"check_in_enabled"`
	// Deferred check-in activation: enable is pending until day 1 of next month.
	PendingCheckInEnabled *bool      `json:"pending_check_in_enabled,omitempty"`
	CheckInEffectiveFrom  *time.Time `json:"check_in_effective_from,omitempty"`
	CreatedBy             uint       `json:"created_by"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// ProjectEmployeeResponse represents the response for listing employees (GET /projects/:id/employees)
// Includes employee_name and employee_cccd per API spec
type ProjectEmployeeResponse struct {
	ID                     uint       `json:"id"`
	ProjectID              uint       `json:"project_id"`
	EmployeeID             uint       `json:"employee_id"`
	EmployeeName           string     `json:"employee_name"`
	EmployeeCCCD           string     `json:"employee_cccd"`
	EmployeeCode           string     `json:"employee_code"`
	Position               string     `json:"position"`
	StartDate              time.Time  `json:"start_date"`
	LastDate               *time.Time `json:"last_date"`
	PaymentSchedule        string     `json:"payment_schedule"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty"`
	CheckInEnabled         bool       `json:"check_in_enabled"`
	// Deferred check-in activation: enable is pending until day 1 of next month.
	PendingCheckInEnabled *bool      `json:"pending_check_in_enabled,omitempty"`
	CheckInEffectiveFrom  *time.Time `json:"check_in_effective_from,omitempty"`
	CreatedBy             uint       `json:"created_by"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

// ListProjectEmployeesResponse represents the response for listing project employee assignments
type ListProjectEmployeesResponse struct {
	Assignments []ProjectEmployeeResponse `json:"assignments"`
	Total       int64                     `json:"total"`
	Limit       int                       `json:"limit"`
	Offset      int                       `json:"offset"`
}

// ProjectEmployeeWithDetailsResponse represents project employee assignment with full details
type ProjectEmployeeWithDetailsResponse struct {
	ID                     uint       `json:"id"`
	ProjectID              uint       `json:"project_id"`
	EmployeeID             uint       `json:"employee_id"`
	EmployeeCode           string     `json:"employee_code"`
	Position               string     `json:"position"`
	StartDate              time.Time  `json:"start_date"`
	LastDate               *time.Time `json:"last_date"`
	PaymentSchedule        string     `json:"payment_schedule"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty"`
	CheckInEnabled         bool       `json:"check_in_enabled"`
	// Deferred check-in activation: enable is pending until day 1 of next month.
	PendingCheckInEnabled *bool      `json:"pending_check_in_enabled,omitempty"`
	CheckInEffectiveFrom  *time.Time `json:"check_in_effective_from,omitempty"`
	CreatedBy             uint       `json:"created_by"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`

	// Project details
	ProjectName   string `json:"project_name"`
	ProjectCode   string `json:"project_code"`
	ProjectStatus string `json:"project_status"`
	ClientName    string `json:"client_name"`

	// Employee details
	EmployeeFullname string `json:"employee_fullname"`
	EmployeeCCCD     string `json:"employee_cccd"`

	// Creator details
	CreatedByFullname string `json:"created_by_fullname"`
}

// UpdatePaymentScheduleRequest represents the request to change an employee's payment schedule
type UpdatePaymentScheduleRequest struct {
	NewSchedule string `json:"new_schedule" binding:"required,oneof=weekly monthly flexible"`
}

// CancelPaymentScheduleChangeRequest can be empty (just a confirmation)
type CancelPaymentScheduleChangeRequest struct{}

// ToggleCheckInEnabledRequest represents the request to toggle check-in for an employee
type ToggleCheckInEnabledRequest struct {
	CheckInEnabled bool `json:"check_in_enabled"`
}

// BulkToggleCheckInEnabledRequest represents the request to toggle check-in for multiple employees
type BulkToggleCheckInEnabledRequest struct {
	EmployeeIDs    []uint `json:"employee_ids" binding:"required,min=1"`
	CheckInEnabled bool   `json:"check_in_enabled"`
}

type CheckInConfigurationEmployeeResponse struct {
	AssignmentID         uint       `json:"assignment_id"`
	ProjectID            uint       `json:"project_id"`
	EmployeeID           uint       `json:"employee_id"`
	EmployeeName         string     `json:"employee_name"`
	EmployeeCCCD         string     `json:"employee_cccd"`
	EmployeeCode         string     `json:"employee_code"`
	CheckInEnabled       bool       `json:"check_in_enabled"`
	PendingCheckInEnable bool       `json:"pending_check_in_enable"`
	CheckInEffectiveFrom *time.Time `json:"check_in_effective_from,omitempty"`
	AttendanceCount      int64      `json:"attendance_count"`
	LastCheckInAt        *string    `json:"last_check_in_at,omitempty"`
}

type CheckInConfigurationSummaryResponse struct {
	Enabled  int64 `json:"enabled"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	Pending  int64 `json:"pending"`
}

type CheckInConfigurationPaginationResponse struct {
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	TotalPages   int `json:"totalPages"`
	TotalRecords int `json:"totalRecords"`
}

type CheckInConfigurationResponse struct {
	Employees  []CheckInConfigurationEmployeeResponse `json:"employees"`
	Summary    CheckInConfigurationSummaryResponse    `json:"summary"`
	Month      string                                 `json:"month"`
	Pagination CheckInConfigurationPaginationResponse `json:"pagination"`
}

type DisableInactiveCheckInEmployeesResponse struct {
	DisabledCount int `json:"disabled_count"`
}

type DisablePendingCheckInEmployeesResponse struct {
	DisabledCount int `json:"disabled_count"`
}
