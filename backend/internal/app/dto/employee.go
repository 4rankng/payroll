package dto

import (
	"time"
)

// CreateEmployeeRequest represents the request to create a new employee
type CreateEmployeeRequest struct {
	Fullname          string  `json:"fullname" binding:"required"`
	Email             *string `json:"email,omitempty" binding:"omitempty,email"`
	CCCD              string  `json:"cccd" binding:"required"`
	Address           string  `json:"address,omitempty"`
	Mobile            string  `json:"mobile,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber string  `json:"bank_account_number,omitempty"`
	BankAccountName   string  `json:"bank_account_name,omitempty"`
	DateOfBirth       string  `json:"date_of_birth,omitempty"`
}

// UpdateEmployeeRequest represents the request to update an employee
type UpdateEmployeeRequest struct {
	Fullname          *string `json:"fullname,omitempty"`
	Email             *string `json:"email,omitempty" binding:"omitempty,email"`
	CCCD              *string `json:"cccd,omitempty"`
	Address           *string `json:"address,omitempty"`
	Mobile            *string `json:"mobile,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`
	DateOfBirth       *string `json:"date_of_birth,omitempty"`
}

// ChangeEmployeePasswordRequest represents the request to change an employee's password
type ChangeEmployeePasswordRequest struct {
	NewPassword string `json:"new_password" binding:"required"`
}

// EmployeeBankInfo represents bank information in employee responses
type EmployeeBankInfo struct {
	ID         uint   `json:"id"`
	BranchName string `json:"branch_name"`
}

// EmployeeResponse represents the response containing employee data
type EmployeeResponse struct {
	ID                uint              `json:"id"`
	Username          *string           `json:"username,omitempty"`
	Fullname          string            `json:"fullname"`
	Email             *string           `json:"email"`
	CCCD              string            `json:"cccd"`
	Address           string            `json:"address"`
	Mobile            string            `json:"mobile"`
	Bank              *EmployeeBankInfo `json:"bank,omitempty"`
	BankAccountNumber string            `json:"bank_account_number"`
	BankAccountName   string            `json:"bank_account_name"`
	// BankAccountStatus / InvalidReason / ValidatedAt surface the OnePay
	// verification outcome so the warning-list UI can show why an account
	// is flagged. status is "valid" by default (also for pre-existing rows).
	BankAccountStatus        string                `json:"bank_account_status"`
	BankAccountInvalidReason *string               `json:"bank_account_invalid_reason,omitempty"`
	BankAccountValidatedAt   *string               `json:"bank_account_validated_at,omitempty"`
	DateOfBirth              *string               `json:"date_of_birth"`
	CreatedBy                uint                  `json:"created_by"`
	CreatedAt                time.Time             `json:"created_at"`
	UpdatedAt                time.Time             `json:"updated_at"`
	CanDelete                bool                  `json:"can_delete"`
	CurrentProjects          []EmployeeProjectInfo `json:"current_projects"`
	// IsAccessible indicates whether the requesting partner already manages this
	// employee (created/shared/project-assigned). Only set on global-pool listings
	// (scope=global, partner role); omitted elsewhere.
	IsAccessible *bool `json:"is_accessible,omitempty"`
	// CreatorName is the display name of the employee's creator. Exposed so the
	// global pool can show "Được quản lý bởi" without exposing the creator user ID.
	CreatorName *string `json:"creator_name,omitempty"`
}

// EmployeeProjectInfo represents project information for an employee
type EmployeeProjectInfo struct {
	ProjectID              uint    `json:"project_id"`
	ProjectEmployeeID      uint    `json:"project_employee_id"`
	Name                   string  `json:"name"`
	Code                   string  `json:"code"`
	ClientName             string  `json:"client_name"`
	Position               string  `json:"position"`
	StartDate              string  `json:"start_date"`
	LastDate               *string `json:"last_date"`
	PaymentSchedule        string  `json:"payment_schedule"`
	PendingPaymentSchedule *string `json:"pending_payment_schedule"`
	ScheduleEffectiveFrom  *string `json:"schedule_effective_from"`
	IsFlexible             bool    `json:"is_flexible"`
	CheckInEnabled         bool    `json:"check_in_enabled"`
}

// ListEmployeesResponse represents the response for listing employees
type ListEmployeesResponse struct {
	Employees []EmployeeResponse `json:"employees"`
	Total     int64              `json:"total"`
	Limit     int                `json:"limit"`
	Offset    int                `json:"offset"`
}

// DuplicateCheckItemResponse is one existing employee matching the typed
// identifiers. Only identifiers the requester typed are revealed, masked.
type DuplicateCheckItemResponse struct {
	ID                  uint      `json:"id"`
	Fullname            string    `json:"fullname"`
	CCCDMasked          string    `json:"cccd_masked,omitempty"`
	MobileMasked        string    `json:"mobile_masked,omitempty"`
	EmailMasked         string    `json:"email_masked,omitempty"`
	CurrentProjectNames []string  `json:"current_project_names"`
	CreatedByName       string    `json:"created_by_name"`
	CreatedAt           time.Time `json:"created_at"`
	MatchedOn           []string  `json:"matched_on"`
}

// DuplicateCheckResponse is the payload for GET /employees/duplicate-check
type DuplicateCheckResponse struct {
	HasDuplicates bool                         `json:"has_duplicates"`
	Data          []DuplicateCheckItemResponse `json:"data"`
}

// EmployeesSummaryResponse represents the response for employees summary
type EmployeesSummaryResponse struct {
	TotalEmployees          int64 `json:"total_employees"`
	TotalWorkingEmployees   int64 `json:"total_working_employees"`
	EmployeesHiredThisMonth int64 `json:"employees_hired_this_month"`
	SalaryMonthToDate       int64 `json:"salary_month_to_date"`
	PaidMonthToDate         int64 `json:"paid_month_to_date"`
}

// EmployeeSummaryResponse represents the response for individual employee summary
type EmployeeSummaryResponse struct {
	TotalPayrollPayments int    `json:"total_payroll_payments"`
	TotalEarningsVND     int64  `json:"total_earnings_vnd"`
	LastPaymentDate      string `json:"last_payment_date"`
	AvgWeeklyEarningsVND int64  `json:"avg_weekly_earnings_vnd"`
}

// EmployeeTimesheetSummary represents timesheet summary for an employee
type EmployeeTimesheetSummary struct {
	TotalTimesheets    int     `json:"total_timesheets"`
	TotalHoursWorked   float64 `json:"total_hours_worked"`
	PendingTimesheets  int     `json:"pending_timesheets"`
	ApprovedTimesheets int     `json:"approved_timesheets"`
	RejectedTimesheets int     `json:"rejected_timesheets"`
	CurrentWeekHours   float64 `json:"current_week_hours"`
	LastTimesheetDate  string  `json:"last_timesheet_date,omitempty"`
}

// EmployeeCurrentProject represents current active project for an employee
type EmployeeCurrentProject struct {
	ProjectID              uint    `json:"project_id"`
	ProjectEmployeeID      uint    `json:"project_employee_id"`
	Name                   string  `json:"name"`
	Code                   string  `json:"code"`
	ClientName             string  `json:"client_name"`
	Position               string  `json:"position"`
	StartDate              string  `json:"start_date"`
	LastDate               *string `json:"last_date"`
	PaymentSchedule        string  `json:"payment_schedule"`
	PendingPaymentSchedule *string `json:"pending_payment_schedule"`
	ScheduleEffectiveFrom  *string `json:"schedule_effective_from"`
	IsFlexible             bool    `json:"is_flexible"`
	CheckInEnabled         bool    `json:"check_in_enabled"`
}

// EmployeeCurrentProjectTimesheet represents a timesheet entry for current projects endpoint
type EmployeeCurrentProjectTimesheet struct {
	ID              uint       `json:"id"`
	Date            string     `json:"date"`
	HoursWorked     float64    `json:"hours_worked"`
	PayType         string     `json:"paytype"`
	HourType        string     `json:"hour_type"`
	DayType         string     `json:"day_type"`
	PayRate         float64    `json:"payrate"`
	Amount          float64    `json:"amount"`
	Status          string     `json:"status"`
	PaymentStatus   *string    `json:"payment_status,omitempty"`
	PaidAmount      float64    `json:"paid_amount,omitempty"`
	PaidAt          *time.Time `json:"paid_at,omitempty"`
	ApprovedAt      *time.Time `json:"approved_at"`
	RejectionReason string     `json:"rejection_reason"`
	CreatedAt       time.Time  `json:"created_at"`
}

// EmployeeCurrentProjectWithTimesheets represents current project with timesheets
type EmployeeCurrentProjectWithTimesheets struct {
	ProjectID   uint                              `json:"project_id"`
	ProjectName string                            `json:"project_name"`
	ProjectCode string                            `json:"project_code"`
	ClientName  string                            `json:"client_name"`
	Position    string                            `json:"position"`
	StartDate   string                            `json:"start_date"`
	LastDate    *string                           `json:"last_date"`
	Timesheets  []EmployeeCurrentProjectTimesheet `json:"timesheets"`
}

// EmployeeCurrentProjectsResponse represents the response for employee current projects endpoint
type EmployeeCurrentProjectsResponse []EmployeeCurrentProjectWithTimesheets

// EmployeeDetailedResponse represents comprehensive employee information
type EmployeeDetailedResponse struct {
	// Basic employee information
	ID                uint              `json:"id"`
	Username          *string           `json:"username,omitempty"`
	Fullname          string            `json:"fullname"`
	Email             *string           `json:"email"`
	CCCD              string            `json:"cccd"`
	Address           string            `json:"address"`
	Mobile            string            `json:"mobile"`
	Bank              *EmployeeBankInfo `json:"bank,omitempty"`
	BankAccountNumber string            `json:"bank_account_number"`
	BankAccountName   string            `json:"bank_account_name"`
	DateOfBirth       *string           `json:"date_of_birth"`
	CreatedBy         uint              `json:"created_by"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`

	// Bank account validation outcome (mirrors EmployeeResponse).
	BankAccountStatus        string  `json:"bank_account_status"`
	BankAccountInvalidReason *string `json:"bank_account_invalid_reason,omitempty"`
	BankAccountValidatedAt   *string `json:"bank_account_validated_at,omitempty"`

	// Current project assignments
	CurrentProjects []EmployeeCurrentProject `json:"current_projects"`

	// Timesheet summary
	TimesheetSummary *EmployeeTimesheetSummary `json:"timesheet_summary,omitempty"`

	// Payroll summary
	PayrollSummary *EmployeeSummaryResponse `json:"payroll_summary,omitempty"`

	// True when the requesting partner already manages this employee
	// (created/shared/project-assigned). When false, bank/address are masked
	// and edit actions should be hidden. Only meaningful for partner callers.
	IsAccessible *bool `json:"is_accessible,omitempty"`
}

// GrantEmployeeAccessRequest represents the request to grant employee access to a user
type GrantEmployeeAccessRequest struct {
	UserID uint `json:"user_id" binding:"required"`
}

// EmployeeUserResponse represents employee sharing information
type EmployeeUserResponse struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	UserFullname string    `json:"user_fullname"`
	UserEmail    string    `json:"user_email"`
	GrantedBy    uint      `json:"granted_by"`
	GrantedAt    time.Time `json:"granted_at"`
}

// EmployeePayrollHistoryItem represents a single payroll entry for an employee
type EmployeePayrollHistoryItem struct {
	ID               uint       `json:"id"`
	TimesheetID      uint       `json:"timesheet_id"`
	ProjectID        uint       `json:"project_id"`
	ProjectName      string     `json:"project_name"`
	Date             string     `json:"date"`
	HoursWorked      float64    `json:"hours_worked"`
	PayType          string     `json:"paytype"`
	PayRate          float64    `json:"payrate"`
	Amount           float64    `json:"amount"`
	PaymentStatus    string     `json:"payment_status"`
	PaymentReference *string    `json:"payment_reference,omitempty"`
	PaymentDate      *string    `json:"payment_date,omitempty"`
	PeriodStart      *string    `json:"period_start,omitempty"`
	PeriodEnd        *string    `json:"period_end,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	PaidAmount       float64    `json:"paid_amount"`
	PaidAt           *time.Time `json:"paid_at,omitempty"`
}

// EmployeePayrollHistoryResponse represents the response for employee payroll history
type EmployeePayrollHistoryResponse struct {
	Data       []EmployeePayrollHistoryItem `json:"data"`
	Pagination PaginationResponse           `json:"pagination"`
}
