package dto

import "time"

// EmployeeProfileResponse represents employee's own profile data
type EmployeeProfileResponse struct {
	ID                uint       `json:"id"`
	Fullname          string     `json:"fullname"`
	Email             *string    `json:"email"`
	Username          string     `json:"username"`
	Mobile            string     `json:"mobile"`
	Address           string     `json:"address"`
	DateOfBirth       *time.Time `json:"date_of_birth"`
	BankAccountNumber string     `json:"bank_account_number"`
	BankAccountName   string     `json:"bank_account_name"`
	Bank              *BankInfo  `json:"bank,omitempty"`
	PaymentSchedule   string     `json:"payment_schedule"`
	CheckInEnabled    bool       `json:"check_in_enabled"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type BankInfo struct {
	ID         uint   `json:"id"`
	BranchName string `json:"branch_name"`
}

// UpdateEmployeeProfileRequest represents update request for employee's own profile
type UpdateEmployeeProfileRequest struct {
	Fullname *string `json:"fullname" binding:"omitempty,min=2"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Username *string `json:"username" binding:"omitempty,min=3"`
}

// UpdateEmployeePasswordRequest represents password update request
type UpdateEmployeePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// EmployeeTimesheetEntryResponse represents a single timesheet entry for employee
type EmployeeTimesheetEntryResponse struct {
	ID            uint                       `json:"id"`
	Date          time.Time                  `json:"date"`
	Project       *EmployeeTimesheetProject  `json:"project"`
	HoursWorked   float64                    `json:"hours_worked"`
	Amount        int64                      `json:"amount"`
	Status        string                     `json:"timesheet_status"`
	PaymentStatus string                     `json:"payment_status"`
	PaidAmount    int64                      `json:"paid_amount"`
	PaymentDate   *time.Time                 `json:"payment_date"`
	ApprovedAt    *time.Time                 `json:"approved_at"`
	ApprovedBy    *EmployeeTimesheetApprover `json:"approved_by"`
	CreatedAt     time.Time                  `json:"created_at"`
}

type EmployeeTimesheetProject struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	ClientName string `json:"client_name"`
}

type EmployeeTimesheetApprover struct {
	ID       uint   `json:"id"`
	Fullname string `json:"fullname"`
}

// EmployeePortalSummaryResponse represents summary statistics for employee self-service portal
type EmployeePortalSummaryResponse struct {
	WeeklySalary            int64                      `json:"weekly_salary"`
	WeeklyClockedHours      float64                    `json:"weekly_clocked_hours"`
	WeeklyPaidAmount        int64                      `json:"weekly_paid_amount"`
	TotalApprovedTimesheets int                        `json:"total_approved_timesheets"`
	TotalApprovedAmount     int64                      `json:"total_approved_amount"`
	TotalPaidTimesheets     int                        `json:"total_paid_timesheets"`
	TotalPaidAmount         int64                      `json:"total_paid_amount"`
	TotalPendingTimesheets  int                        `json:"total_pending_timesheets"`
	TotalPendingAmount      int64                      `json:"total_pending_amount"`
	CalculationPeriod       *EmployeeCalculationPeriod `json:"calculation_period"`
	LastPaymentDate         *time.Time                 `json:"last_payment_date"`
	CurrentProjects         []*EmployeeCurrentProject  `json:"current_projects"`
}

type EmployeeCalculationPeriod struct {
	Weeks    int    `json:"weeks"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

// InitializeEmployeeUsersResponse represents response for bulk user initialization
type InitializeEmployeeUsersResponse struct {
	TotalEmployees  int                   `json:"total_employees"`
	UsersCreated    int                   `json:"users_created"`
	AlreadyHadUsers int                   `json:"already_had_users"`
	CreatedUsers    []CreatedEmployeeUser `json:"created_users"`
}

type CreatedEmployeeUser struct {
	EmployeeID      uint   `json:"employee_id"`
	EmployeeName    string `json:"employee_name"`
	Username        string `json:"username"`
	DefaultPassword string `json:"default_password"`
}
