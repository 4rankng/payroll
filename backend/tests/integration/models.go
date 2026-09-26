package main

import (
	"encoding/json"
	"time"
)

// --- API Envelope ---

// APIResponse is the universal JSON envelope from the backend.
type APIResponse struct {
	Status     string          `json:"status"`
	Data       json.RawMessage `json:"data"`
	Message    string          `json:"message"`
	Pagination *Pagination     `json:"pagination,omitempty"`
}

// APIError is returned on failure.
type APIError struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status"`
}

// Pagination represents pagination metadata.
type Pagination struct {
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
	TotalPages   int   `json:"totalPages"`
	TotalRecords int64 `json:"totalRecords"`
}

// --- Auth ---

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User        UserResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresIn   int64        `json:"expires_in"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	CCCD     string `json:"cccd,omitempty"`
	Mobile   string `json:"mobile,omitempty"`
}

// --- Projects ---

type ProjectResponse struct {
	ID                         uint   `json:"id"`
	ClientName                 string `json:"client_name"`
	Name                       string `json:"name"`
	Code                       string `json:"code"`
	Status                     string `json:"status"`
	EmployeeCount              int    `json:"employee_count"`
	WeeklySalaryEmployeeCount  int    `json:"weekly_salary_employee_count"`
	MonthlySalaryEmployeeCount int    `json:"monthly_salary_employee_count"`
	SalaryPeriodFrom           int    `json:"salary_period_from"`
	SalaryPeriodTo             int    `json:"salary_period_to"`
}

type ListProjectsResponse struct {
	Projects []ProjectResponse `json:"projects"`
	Total    int64             `json:"total"`
}

type CheckInConfigurationEmployeeResponse struct {
	AssignmentID         uint   `json:"assignment_id"`
	EmployeeID           uint   `json:"employee_id"`
	EmployeeName         string `json:"employee_name"`
	CheckInEnabled       bool   `json:"check_in_enabled"`
	PendingCheckInEnable bool   `json:"pending_check_in_enable"`
	AttendanceCount      int64  `json:"attendance_count"`
}

type CheckInConfigurationSummaryResponse struct {
	Enabled  int64 `json:"enabled"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	Pending  int64 `json:"pending"`
}

type CheckInConfigurationResponse struct {
	Employees  []CheckInConfigurationEmployeeResponse `json:"employees"`
	Summary    CheckInConfigurationSummaryResponse    `json:"summary"`
	Month      string                                 `json:"month"`
	Pagination Pagination                             `json:"pagination"`
}

type DisableCheckInEmployeesResponse struct {
	DisabledCount int `json:"disabled_count"`
}

// --- Employees ---

type EmployeeResponse struct {
	ID                uint                  `json:"id"`
	Fullname          string                `json:"fullname"`
	Email             *string               `json:"email"`
	CCCD              string                `json:"cccd"`
	Mobile            string                `json:"mobile"`
	BankAccountNumber string                `json:"bank_account_number"`
	BankAccountName   string                `json:"bank_account_name"`
	CurrentProjects   []EmployeeProjectInfo `json:"current_projects"`
}

type EmployeeDetailedResponse struct {
	ID                uint                  `json:"id"`
	Username          *string               `json:"username,omitempty"`
	Fullname          string                `json:"fullname"`
	Email             *string               `json:"email"`
	CCCD              string                `json:"cccd"`
	Mobile            string                `json:"mobile"`
	BankAccountNumber string                `json:"bank_account_number"`
	BankAccountName   string                `json:"bank_account_name"`
	CurrentProjects   []EmployeeProjectInfo `json:"current_projects"`
}

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
	CheckInEnabled         bool    `json:"check_in_enabled"`
	PendingPaymentSchedule *string `json:"pending_payment_schedule"`
}

type ListEmployeesResponse struct {
	Employees []EmployeeResponse `json:"employees"`
	Total     int64              `json:"total"`
}

// --- Banks ---

type BankResponse struct {
	ID         uint   `json:"id"`
	BranchName string `json:"branch_name"`
	BankCode   string `json:"bank_code"`
	Bin        string `json:"bin"`
	SwiftCode  string `json:"swift_code"`
}

type ListBanksResponse struct {
	Banks []BankResponse `json:"banks"`
	Total int64          `json:"total"`
}

// --- Timesheets ---

// BulkCreateTimesheetEntry uses camelCase JSON tags (matches BulkCreateTimesheetRequest).
type BulkCreateTimesheetEntry struct {
	ProjectID   uint    `json:"projectId"`
	EmployeeID  uint    `json:"employeeId"`
	Date        string  `json:"date"`
	HoursWorked float64 `json:"hoursWorked"`
	HourType    string  `json:"hourType"`
	DayType     *string `json:"dayType,omitempty"`
}

type BulkCreateTimesheetResponse struct {
	Succeeded    []TimesheetResponse `json:"succeeded"`
	Failed       []BulkCreateError   `json:"failed"`
	TotalSuccess int                 `json:"total_success"`
	TotalDeleted int                 `json:"total_deleted"`
	TotalFailed  int                 `json:"total_failed"`
}

type BulkCreateError struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

type TimesheetResponse struct {
	ID            uint       `json:"id"`
	ProjectID     uint       `json:"project_id"`
	EmployeeID    uint       `json:"employee_id"`
	Date          string     `json:"date"`
	HoursWorked   float64    `json:"hours_worked"`
	PayType       string     `json:"paytype"`
	HourType      string     `json:"hour_type"`
	DayType       string     `json:"day_type"`
	PayRate       float64    `json:"payrate"`
	Amount        float64    `json:"amount"`
	Status        string     `json:"status"`
	PaymentStatus string     `json:"payment_status"`
	PaidAmount    int64      `json:"paid_amount"`
	PaidAt        *time.Time `json:"paid_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

type BulkApproveRequest struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
}

type BulkOperationResponse struct {
	Approved int `json:"approved,omitempty"`
	Skipped  int `json:"skipped,omitempty"`
	Failed   int `json:"failed,omitempty"`
}

type ListTimesheetsResponse struct {
	Timesheets []TimesheetWithDetailsResponse `json:"timesheets"`
	Pagination Pagination                     `json:"pagination"`
}

type TimesheetWithDetailsResponse struct {
	TimesheetResponse
	ProjectName  string `json:"projectName"`
	EmployeeName string `json:"employeeName"`
}

// --- Payroll / Bulk Transfer ---

type ExportBulkTransferRequest struct {
	ProjectIDs  []uint `json:"project_ids,omitempty"`
	EmployeeIDs []uint `json:"employee_ids,omitempty"`
	FromDate    string `json:"fromDate,omitempty"`
	ToDate      string `json:"toDate,omitempty"`
	ForMonth    string `json:"for_month,omitempty"`
}

type EstimateFeeResponse struct {
	TotalFee       int64 `json:"total_fee"`
	TransferCount  int   `json:"transfer_count"`
	FeePerTransfer int64 `json:"fee_per_transfer"`
}

type AutoBulkTransferResponse struct {
	BatchID    string `json:"batch_id"`
	FileID     uint   `json:"file_id"`
	TotalCount int    `json:"total_count"`
	Status     string `json:"status"`
}

type AutoBulkTransferStatusResponse struct {
	BatchID     string                   `json:"batch_id"`
	TotalCount  int                      `json:"total_count"`
	Completed   int                      `json:"completed"`
	Failed      int                      `json:"failed"`
	Processing  int                      `json:"processing"`
	Status      string                   `json:"status"`
	FailedItems []BulkTransferFailedItem `json:"failed_items,omitempty"`
}

type BulkTransferFailedItem struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name,omitempty"`
	Amount       int64  `json:"amount"`
	Reason       string `json:"reason,omitempty"`
}

// --- Advance Payment (Employee) ---

type AdvancePaymentInfoResponse struct {
	ForMonth                  string  `json:"forMonth"`
	MaxAdvanceAmount          uint64  `json:"maxAdvanceAmount"`
	CompletedAmount           uint64  `json:"completedAmount"`
	PendingAmount             uint64  `json:"pendingAmount"`
	RemainingAmount           uint64  `json:"remainingAmount"`
	CanRequest                bool    `json:"canRequest"`
	CanRequestTitle           string  `json:"canRequestTitle"`
	CanRequestReason          string  `json:"canRequestReason"`
	FeePercentage             float64 `json:"feePercentage"`
	MinFee                    uint64  `json:"minFee"`
	HasFlexible               bool    `json:"hasFlexible"`
	ProviderMinTransferAmount uint64  `json:"providerMinTransferAmount,omitempty"`
	ProviderMaxTransferAmount uint64  `json:"providerMaxTransferAmount,omitempty"`
}

type CreateAdvancePaymentRequest struct {
	Amount   uint64 `json:"amount"`
	ForMonth string `json:"forMonth"`
}

type CalculateFeeResponse struct {
	RequestAmount uint64 `json:"requestAmount"`
	Fee           uint64 `json:"fee"`
	NetAmount     uint64 `json:"netAmount"`
}

type AdvancePaymentHistoryItem struct {
	ID            uint       `json:"id"`
	RequestAmount uint64     `json:"requestAmount"`
	Fee           uint64     `json:"fee"`
	NetAmount     uint64     `json:"netAmount"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	ProjectName   string     `json:"projectName"`
	ProjectCode   string     `json:"projectCode"`
}

type AdvancePaymentHistoryResponse struct {
	Data       []AdvancePaymentHistoryItem `json:"data"`
	Pagination Pagination                  `json:"pagination"`
}

// --- Advance Payment (Admin) ---

type AdvancePaymentRequestItem struct {
	ID            uint       `json:"id"`
	EmployeeID    uint       `json:"employeeId"`
	EmployeeName  string     `json:"employeeName,omitempty"`
	ProjectID     uint       `json:"projectId"`
	ProjectCode   string     `json:"projectCode,omitempty"`
	ProjectName   string     `json:"projectName,omitempty"`
	RequestAmount uint64     `json:"requestAmount"`
	Fee           uint64     `json:"fee"`
	NetAmount     uint64     `json:"netAmount"`
	Status        string     `json:"status"`
	PaymentRef    string     `json:"paymentReference,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
}

type WalletPaymentDetail struct {
	ID           uint64     `json:"id"`
	TxnID        string     `json:"txn_id"`
	RequestID    string     `json:"request_id"`
	InvoiceNo    string     `json:"invoice_no"`
	Amount       int64      `json:"requested_amount"`
	Fee          int64      `json:"fee"`
	Status       string     `json:"status"`
	ErrorCode    string     `json:"error_code,omitempty"`
	ErrorMessage string     `json:"error_message,omitempty"`
	EntityID     uint64     `json:"entity_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	SettledAt    *time.Time `json:"settled_at,omitempty"`
}

type ListAdvancePaymentsResponse struct {
	Data       []AdvancePaymentRequestItem `json:"data"`
	Pagination Pagination                  `json:"pagination"`
}

type AdvancePaymentSummaryResponse struct {
	TotalRequests   int64  `json:"totalRequests"`
	TotalPending    int64  `json:"totalPending"`
	TotalApproved   int64  `json:"totalApproved"`
	TotalPaid       int64  `json:"totalPaid"`
	TotalAmount     uint64 `json:"totalAmount"`
	TotalPaidAmount uint64 `json:"totalPaidAmount"`
	TotalFee        uint64 `json:"totalFee"`
}

type AvailableMonth struct {
	ForMonth      string `json:"forMonth"`
	EmployeeCount int    `json:"employeeCount"`
}

type EmployeeAdvanceListResponse struct {
	ForMonth string                `json:"forMonth"`
	Data     []EmployeeAdvanceItem `json:"data"`
}

type EmployeeAdvanceItem struct {
	EmployeeID             uint                        `json:"employeeId"`
	Fullname               string                      `json:"fullname"`
	CCCD                   string                      `json:"cccd"`
	Username               string                      `json:"username"`
	Bank                   *EmployeeAdvanceBankInfo    `json:"bank"`
	Project                *EmployeeAdvanceProjectInfo `json:"project"`
	ForMonth               string                      `json:"forMonth"`
	Salary                 uint64                      `json:"salary"`
	MaxAdvanceAmount       uint64                      `json:"maxAdvanceAmount"`
	UtilizedAmount         uint64                      `json:"utilizedAmount"`
	AvailableAmount        uint64                      `json:"availableAmount"`
	TotalFeeGenerated      uint64                      `json:"totalFeeGenerated"`
	PendingAmount          uint64                      `json:"pendingAmount"`
	CompletedRequestsCount int                         `json:"completedRequestsCount"`
	PendingRequestsCount   int                         `json:"pendingRequestsCount"`
}

type EmployeeAdvanceBankInfo struct {
	BankID        uint   `json:"bankId"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

type EmployeeAdvanceProjectInfo struct {
	ID                    uint   `json:"id"`
	Name                  string `json:"name"`
	Code                  string `json:"code"`
	AssignmentID          uint   `json:"assignment_id"`
	CheckInEnabled        bool   `json:"check_in_enabled"`
	AdvanceRequestEnabled bool   `json:"advance_request_enabled"`
}

// --- FlexPay Import ---

type ImportJobResponse struct {
	ID        uint   `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type ImportJobStatusResponse struct {
	ID            uint                     `json:"id"`
	Status        string                   `json:"status"`
	ForMonth      string                   `json:"forMonth"`
	Filename      string                   `json:"filename"`
	TotalRows     int                      `json:"totalRows"`
	ProcessedRows int                      `json:"processedRows"`
	Percentage    int                      `json:"percentage"`
	Result        *ImportFlexPayFileResult `json:"result,omitempty"`
	Error         *string                  `json:"error,omitempty"`
	CreatedAt     string                   `json:"created_at"`
	StartedAt     *string                  `json:"startedAt,omitempty"`
	CompletedAt   *string                  `json:"completedAt,omitempty"`
}

type ImportFlexPayFileResult struct {
	ForMonth               string `json:"forMonth"`
	TotalRows              int    `json:"totalRows"`
	EmployeesCreated       int    `json:"employeesCreated"`
	EmployeesSkipped       int    `json:"employeesSkipped"`
	ProjectsCreated        int    `json:"projectsCreated"`
	ProjectsSkipped        int    `json:"projectsSkipped"`
	AssignmentsCreated     int    `json:"assignmentsCreated"`
	AssignmentsSkipped     int    `json:"assignmentsSkipped"`
	AdvancePaymentsCreated int    `json:"advancePaymentsCreated"`
	AdvancePaymentsSkipped int    `json:"advancePaymentsSkipped"`
}

// --- Manual Bulk Transfer Result ---

type BulkTransferResultResponse struct {
	Data         []BulkTransferResultItem `json:"data"`
	TotalTxn     int                      `json:"total_txn"`
	CompletedTxn int                      `json:"completed_txn"`
	FailedTxn    int                      `json:"failed_txn"`
}

type BulkTransferResultItem struct {
	Row                   int     `json:"row"`
	EmployeeBank          string  `json:"employee_bank"`
	EmployeeAccountNumber string  `json:"employee_account_number"`
	EmployeeName          string  `json:"employee_name"`
	EmployeeCCCD          string  `json:"employee_cccd"`
	Amount                string  `json:"amount"`
	PaymentStatus         string  `json:"payment_status"`
	PaidAt                *string `json:"paid_at,omitempty"`
}

// --- Wallet ---

type WalletBalanceResponse struct {
	Balance      int64  `json:"balance"`
	Currency     string `json:"currency"`
	LastSyncedAt string `json:"lastSyncedAt,omitempty"`
}

type WalletDemandForecastResponse struct {
	CurrentForMonth string `json:"current_for_month"`
	CurrentCycleDay int    `json:"current_cycle_day"`
	MaxCycleDay     int    `json:"max_cycle_day"`
	Periods         []struct {
		ForMonth string `json:"for_month"`
	} `json:"periods"`
	Prediction struct {
		RecommendedBalance int64 `json:"recommended_balance"`
		P50Reference       int64 `json:"p50_reference"`
		P90Reference       int64 `json:"p90_reference"`
		P99Reference       int64 `json:"p99_reference"`
	} `json:"prediction"`
}

type WalletPaymentItem struct {
	ID           uint       `json:"id"`
	Amount       int64      `json:"amount"`
	Status       string     `json:"status"`
	EmployeeName string     `json:"employeeName,omitempty"`
	EmployeeID   uint       `json:"employeeId,omitempty"`
	Description  string     `json:"description,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
}

// --- Partner / Project User ---

type PartnerUser struct {
	ID       uint
	Username string
	Fullname string
	Token    string
}

// --- Test Data Context ---

type TestData struct {
	AdminToken string
	Projects   []ProjectResponse
	Employees  []EmployeeResponse
	Banks      []BankResponse

	// Picked for bulk transfer tests
	WeeklyProject     *ProjectResponse
	MonthlyProject    *ProjectResponse
	WeeklyEmployee    *EmployeeResponse
	MonthlyEmployee   *EmployeeResponse
	WeeklyAssignment  *EmployeeProjectInfo // assignment metadata for weekly employee
	MonthlyAssignment *EmployeeProjectInfo // assignment metadata for monthly employee
	WeeklyHourType    string               // first available hour type from payrate config
	WeeklyDayType     string               // first available day type from payrate config

	// Picked for advance payment tests
	EmployeeForAdvance  *EmployeeDetailedResponse
	EmployeeTokenForAdv string

	// Partner users for timesheet creation
	Partners []PartnerUser

	// FlexPay import job
	FlexPayJobID uint
}

// --- Auth & User Management ---

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type UpdateProfileRequest struct {
	Email    *string `json:"email,omitempty"`
	Fullname *string `json:"fullname,omitempty"`
	Password *string `json:"password,omitempty"`
	CCCD     *string `json:"cccd,omitempty"`
	Mobile   *string `json:"mobile,omitempty"`
}

type CreateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Mobile   string `json:"mobile,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	Fullname string `json:"fullname"`
	Role     string `json:"role,omitempty"`
}

type UpdateUserRequest struct {
	Email    *string `json:"email,omitempty"`
	Mobile   *string `json:"mobile,omitempty"`
	Username *string `json:"username,omitempty"`
	Fullname *string `json:"fullname,omitempty"`
	Role     *string `json:"role,omitempty"`
}

type ResetPasswordRequest struct {
	Password string `json:"password"`
}

type UserSummaryResponse struct {
	TotalUsers        int64 `json:"total_users"`
	TotalAdmins       int64 `json:"total_admins"`
	TotalPartners     int64 `json:"total_partners"`
	TotalEmployees    int64 `json:"total_employees"`
	RecentLoginsToday int64 `json:"recent_logins_today"`
}

// --- Employee Management (additional) ---

type CreateEmployeeRequest struct {
	Fullname          string  `json:"fullname"`
	Email             *string `json:"email,omitempty"`
	CCCD              string  `json:"cccd"`
	Address           string  `json:"address,omitempty"`
	Mobile            string  `json:"mobile,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber string  `json:"bank_account_number,omitempty"`
	BankAccountName   string  `json:"bank_account_name,omitempty"`
	DateOfBirth       string  `json:"date_of_birth,omitempty"`
}

type UpdateEmployeeRequest struct {
	Fullname          *string `json:"fullname,omitempty"`
	Email             *string `json:"email,omitempty"`
	CCCD              *string `json:"cccd,omitempty"`
	Address           *string `json:"address,omitempty"`
	Mobile            *string `json:"mobile,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`
	DateOfBirth       *string `json:"date_of_birth,omitempty"`
}

type EmployeeSummaryResponse struct {
	TotalPayrollPayments int    `json:"total_payroll_payments"`
	TotalEarningsVND     int64  `json:"total_earnings_vnd"`
	LastPaymentDate      string `json:"last_payment_date"`
	AvgWeeklyEarningsVND int64  `json:"avg_weekly_earnings_vnd"`
}

type EmployeesSummaryResponse struct {
	TotalEmployees          int64 `json:"total_employees"`
	TotalWorkingEmployees   int64 `json:"total_working_employees"`
	EmployeesHiredThisMonth int64 `json:"employees_hired_this_month"`
	SalaryMonthToDate       int64 `json:"salary_month_to_date"`
	PaidMonthToDate         int64 `json:"paid_month_to_date"`
}

type GrantEmployeeAccessRequest struct {
	UserID uint `json:"user_id"`
}

type EmployeeUserResponse struct {
	ID           uint      `json:"id"`
	UserID       uint      `json:"user_id"`
	UserFullname string    `json:"user_fullname"`
	UserEmail    string    `json:"user_email"`
	GrantedBy    uint      `json:"granted_by"`
	GrantedAt    time.Time `json:"granted_at"`
}

// --- Project Management (additional) ---

type CreateProjectRequest struct {
	ClientName       string `json:"client_name"`
	Name             string `json:"name"`
	Code             string `json:"code"`
	SalaryPeriodFrom *int   `json:"salary_period_from,omitempty"`
	SalaryPeriodTo   *int   `json:"salary_period_to,omitempty"`
}

type UpdateProjectRequest struct {
	ClientName       *string `json:"client_name,omitempty"`
	Name             *string `json:"name,omitempty"`
	Code             *string `json:"code,omitempty"`
	Status           *string `json:"status,omitempty"`
	SalaryPeriodFrom *int    `json:"salary_period_from,omitempty"`
	SalaryPeriodTo   *int    `json:"salary_period_to,omitempty"`
}

type ProjectSummaryResponse struct {
	TotalActiveProjects       int64   `json:"total_active_projects"`
	TotalReceivedVND          float64 `json:"total_received_vnd"`
	TotalPayoutVND            float64 `json:"total_payout_vnd"`
	TotalPendingPayableVND    float64 `json:"total_pending_payable_vnd"`
	TotalPendingReceivableVND float64 `json:"total_pending_receivable_vnd"`
}

type AssignEmployeeRequest struct {
	EmployeeID       uint    `json:"employee_id"`
	Position         string  `json:"position,omitempty"`
	PaymentSchedule  string  `json:"payment_schedule,omitempty"`
	StartDate        string  `json:"start_date,omitempty"`
	SalaryPeriodType string  `json:"salary_period_type,omitempty"`
	Rate             float64 `json:"rate,omitempty"`
}

type RemoveEmployeesRequest struct {
	EmployeeIDs []uint `json:"employee_ids"`
}

type ProjectEmployeeResponse struct {
	ID               uint   `json:"id"`
	ProjectID        uint   `json:"project_id"`
	EmployeeID       uint   `json:"employee_id"`
	Position         string `json:"position"`
	PaymentSchedule  string `json:"payment_schedule"`
	StartDate        string `json:"start_date"`
	LastDate         string `json:"last_date,omitempty"`
	EmployeeFullname string `json:"employee_fullname,omitempty"`
	EmployeeCCCD     string `json:"employee_cccd,omitempty"`
}

type AssignmentResponse struct {
	ID                     uint    `json:"id"`
	ProjectID              uint    `json:"project_id"`
	EmployeeID             uint    `json:"employee_id"`
	EmployeeCode           string  `json:"employee_code"`
	Position               string  `json:"position"`
	StartDate              string  `json:"start_date"`
	LastDate               *string `json:"last_date"`
	PaymentSchedule        string  `json:"payment_schedule"`
	PendingPaymentSchedule *string `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *string `json:"schedule_effective_from,omitempty"`
	CreatedBy              uint    `json:"created_by"`
}

// --- Bank Management (additional) ---

type CreateBankRequest struct {
	BranchName string  `json:"branch_name"`
	BankCode   *string `json:"bank_code,omitempty"`
	Bin        *string `json:"bin,omitempty"`
}

type UpdateBankRequest struct {
	BranchName *string `json:"branch_name,omitempty"`
	BankCode   *string `json:"bank_code,omitempty"`
	Bin        *string `json:"bin,omitempty"`
	Status     *string `json:"status,omitempty"`
}

// --- Payrate Management ---

type CreatePayrateRequest struct {
	Rates         json.RawMessage `json:"rates"`
	EffectiveFrom string          `json:"effective_from"`
	EffectiveTo   *string         `json:"effective_to,omitempty"`
}

type UpdatePayrateRequest struct {
	Rates         json.RawMessage `json:"rates"`
	EffectiveFrom string          `json:"effective_from"`
	EffectiveTo   *string         `json:"effective_to,omitempty"`
}

type PayrateResponse struct {
	ID        uint            `json:"id"`
	ProjectID uint            `json:"project_id"`
	Rates     json.RawMessage `json:"rates"`
	FromDate  string          `json:"fromDate"`
	ToDate    *string         `json:"toDate"`
	CreatedBy uint            `json:"created_by"`
}

// --- Loan Management ---

type CreateLenderRequest struct {
	Name              string  `json:"name"`
	CCCD              *string `json:"cccd,omitempty"`
	Email             *string `json:"email,omitempty"`
	Mobile            *string `json:"mobile,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`
}

type UpdateLenderRequest struct {
	Name              *string `json:"name,omitempty"`
	CCCD              *string `json:"cccd,omitempty"`
	Email             *string `json:"email,omitempty"`
	Mobile            *string `json:"mobile,omitempty"`
	Notes             *string `json:"notes,omitempty"`
	BankID            *uint   `json:"bank_id,omitempty"`
	BankAccountNumber *string `json:"bank_account_number,omitempty"`
	BankAccountName   *string `json:"bank_account_name,omitempty"`
}

type LenderResponse struct {
	ID                uint      `json:"id"`
	Name              string    `json:"name"`
	CCCD              *string   `json:"cccd"`
	Email             *string   `json:"email"`
	Mobile            *string   `json:"mobile"`
	Notes             *string   `json:"notes"`
	BankID            *uint     `json:"bank_id"`
	BankAccountNumber *string   `json:"bank_account_number"`
	BankAccountName   *string   `json:"bank_account_name"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type CreateLoanRequest struct {
	LenderID        uint                       `json:"lender_id"`
	PrincipalAmount int64                      `json:"principal_amount"`
	StartDate       string                     `json:"start_date"`
	Schedules       []RepaymentScheduleRequest `json:"schedules,omitempty"`
}

type RepaymentScheduleRequest struct {
	DueDate string `json:"due_date"`
	Amount  int64  `json:"amount"`
}

type LoanResponse struct {
	ID                   uint      `json:"id"`
	LenderID             uint      `json:"lender_id"`
	PrincipalAmount      int64     `json:"principal_amount"`
	InterestRate         float64   `json:"interest_rate"`
	TermMonths           int       `json:"term_months"`
	StartDate            string    `json:"start_date"`
	Status               string    `json:"status"`
	Purpose              string    `json:"purpose,omitempty"`
	PaymentSchedule      string    `json:"payment_schedule,omitempty"`
	Outstanding          int64     `json:"outstanding"`
	OutstandingPrincipal int64     `json:"outstanding_principal"`
	TotalInterestPaid    int64     `json:"total_interest_paid"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type LoanDetailResponse struct {
	ID                   uint                            `json:"id"`
	OutstandingPrincipal int64                           `json:"outstanding_principal"`
	TotalInterestPaid    int64                           `json:"total_interest_paid"`
	Schedules            []LoanRepaymentScheduleResponse `json:"schedules"`
}

type LoanRepaymentScheduleResponse struct {
	ID     uint   `json:"id"`
	Period int    `json:"period"`
	Status string `json:"status"`
}

type ProcessScheduledPaymentRequest struct {
	ScheduleID  uint   `json:"schedule_id"`
	PaymentDate string `json:"payment_date"`
}

type DisburseLoanRequest struct {
	DisbursementDate string `json:"disbursement_date,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type RepayLoanRequest struct {
	Amount        int64  `json:"amount"`
	PaymentDate   string `json:"payment_date,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Notes         string `json:"notes,omitempty"`
}

type ScheduleItemResponse struct {
	DueDate   string `json:"due_date"`
	Payment   int64  `json:"payment"`
	Principal int64  `json:"principal"`
	Interest  int64  `json:"interest"`
	Balance   int64  `json:"balance"`
	IsPaid    bool   `json:"is_paid"`
}

// --- Ledger ---

type CreateLedgerEntryRequest struct {
	Account string `json:"account"`
	Party   string `json:"party"`
	Debit   int64  `json:"debit"`
	Credit  int64  `json:"credit"`
	Date    string `json:"date"`
	AssetID *uint  `json:"asset_id,omitempty"`
}

type LedgerEntryResponse struct {
	ID        uint      `json:"id"`
	Account   string    `json:"account"`
	Party     string    `json:"party"`
	Debit     int64     `json:"debit"`
	Credit    int64     `json:"credit"`
	Balance   int64     `json:"balance"`
	NetAmount int64     `json:"net_amount"`
	Date      string    `json:"date"`
	AssetID   *uint     `json:"asset_id"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	// Reversal state: set on a mirror (the entry it offsets) and on an entry that
	// already has a mirror.
	ReversalOfEntryID *uint  `json:"reversal_of_entry_id"`
	ReversalReason    string `json:"reversal_reason"`
	IsReversed        bool   `json:"is_reversed"`
}

type CashFlowSummaryResponse struct {
	StartDate      string `json:"start_date"`
	EndDate        string `json:"end_date"`
	TotalInflow    int64  `json:"total_inflow"`
	TotalOutflow   int64  `json:"total_outflow"`
	NetCashFlow    int64  `json:"net_cash_flow"`
	OpeningBalance int64  `json:"opening_balance"`
	ClosingBalance int64  `json:"closing_balance"`
}

type LedgerSummaryResponse struct {
	Period struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"period"`
	Totals struct {
		OpeningBalance int64 `json:"opening_balance"`
		ClosingBalance int64 `json:"closing_balance"`
		NetCashflow    int64 `json:"net_cashflow"`
	} `json:"totals"`
	// Per-account debit/credit totals: their sums are the ledger's double-entry
	// invariant, so a test can assert the books still balance after a reversal.
	ByAccount map[string]struct {
		Debit     int64 `json:"debit"`
		Credit    int64 `json:"credit"`
		NetAmount int64 `json:"net_amount"`
	} `json:"by_account"`
}

type AccountMetadataResponse struct {
	Value      string `json:"value"`
	Label      string `json:"label"`
	Category   string `json:"category"`
	NormalSide string `json:"normal_side"`
}

type BulkLedgerEntriesResponse struct {
	Entries      []LedgerEntryResponse `json:"entries"`
	TotalEntries int                   `json:"total_entries"`
	TotalDebits  int64                 `json:"total_debits"`
	TotalCredits int64                 `json:"total_credits"`
}

// --- Transactions ---

type CreateTransactionRequest struct {
	Description     string `json:"description"`
	TransactionType string `json:"transaction_type"`
	Amount          int64  `json:"amount"`
	Party           string `json:"party"`
	Status          string `json:"status"`
	URL             string `json:"url,omitempty"`
	AssetID         *uint  `json:"asset_id,omitempty"`
}

type TransactionResponse struct {
	ID              uint   `json:"id"`
	Description     string `json:"description"`
	TransactionCode string `json:"transaction_code"`
	TransactionType string `json:"transaction_type"`
	Amount          int64  `json:"amount"`
	SettledAmount   int64  `json:"settled_amount"`
	PendingAmount   int64  `json:"pending_amount"`
	Party           string `json:"party"`
	Status          string `json:"status"`
	URL             string `json:"url,omitempty"`
	CreatedBy       uint   `json:"created_by"`
	CreatedAt       string `json:"created_at"`
}

type SettleTransactionRequest struct {
	Amount         int64  `json:"amount"`
	SettlementDate string `json:"settlement_date"`
	ProofURL       string `json:"proof_url,omitempty"`
	Notes          string `json:"notes,omitempty"`
}

type ReverseTransactionRequest struct {
	Reason string `json:"reason,omitempty"`
}

type TransactionMetadataResponse struct {
	TransactionTypes []struct {
		Type  string `json:"type"`
		Label string `json:"label"`
	} `json:"transaction_types"`
	Statuses []struct {
		Type  string `json:"type"`
		Label string `json:"label"`
	} `json:"statuses"`
}

// --- Settings ---

type CreateSettingRequest struct {
	Key       string  `json:"key"`
	Value     *string `json:"value,omitempty"`
	ValueType string  `json:"value_type"`
}

type UpdateSettingRequest struct {
	Key       *string `json:"key,omitempty"`
	Value     *string `json:"value,omitempty"`
	ValueType *string `json:"value_type,omitempty"`
}

type SettingResponse struct {
	ID        uint      `json:"id"`
	Key       string    `json:"key"`
	Value     *string   `json:"value"`
	ValueType string    `json:"value_type"`
	UpdatedAt time.Time `json:"updated_at"`
}

// --- Notifications ---

type NotificationResponse struct {
	ID          uint       `json:"id"`
	Type        string     `json:"type"`
	Channel     string     `json:"channel"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	ContentType string     `json:"content_type"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type CustomNotificationRequest struct {
	RecipientIDs   []uint `json:"recipient_ids,omitempty"`
	Title          string `json:"title"`
	Message        string `json:"message"`
	ContentType    string `json:"content_type,omitempty"`
	ToAllPartners  bool   `json:"to_all_partners,omitempty"`
	ToAllAdmins    bool   `json:"to_all_admins,omitempty"`
	ToAllEmployees bool   `json:"to_all_employees,omitempty"`
}

// --- Audit ---

type AuditLogResponse struct {
	ID         uint      `json:"id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	UserID     uint      `json:"user_id"`
	Username   string    `json:"username"`
	Details    string    `json:"details,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// --- Cron Jobs ---

type CronJobResponse struct {
	Name      string    `json:"name"`
	Schedule  string    `json:"schedule"`
	IsEnabled bool      `json:"is_enabled"`
	LastRunAt time.Time `json:"last_run_at,omitempty"`
	NextRunAt time.Time `json:"next_run_at,omitempty"`
}

// --- Asset ---

type AssetResponse struct {
	ID        uint      `json:"id"`
	FileName  string    `json:"file_name"`
	FileSize  int64     `json:"file_size"`
	MimeType  string    `json:"mime_type"`
	URL       string    `json:"url"`
	CreatedBy uint      `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Timesheet (additional) ---

type BulkRejectRequest struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
	Reason       string `json:"reason"`
}

type BulkResetRequest struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
}

type RejectTimesheetRequest struct {
	Reason string `json:"reason"`
}

type EditRequestResponse struct {
	ID          uint       `json:"id"`
	TimesheetID uint       `json:"timesheet_id"`
	Reason      string     `json:"reason"`
	Status      string     `json:"status"`
	RequestedBy uint       `json:"requested_by"`
	CreatedAt   time.Time  `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

type RequestEditRequest struct {
	Reason string `json:"reason"`
}

// --- Wallet ---

type WalletTopupRequest struct {
	Amount        int64  `json:"amount"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Reference     string `json:"reference,omitempty"`
}

type WalletTopupResponse struct {
	ID        uint      `json:"id"`
	Amount    int64     `json:"amount"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Project Employee ---

type PaymentScheduleChangeRequest struct {
	NewSchedule string `json:"new_schedule"`
}

// --- Disbursement ---

type ManualDisbursementRequest struct {
	RecipientName      string `json:"recipient_name"`
	RecipientAccountNo string `json:"recipient_account_no"`
	RecipientBank      string `json:"recipient_bank"`
	Amount             int64  `json:"amount"`
	Description        string `json:"description,omitempty"`
}

type CheckAccountRequest struct {
	BankCode    string `json:"bank_code"`
	AccountNo   string `json:"account_no"`
	AccountName string `json:"account_name,omitempty"`
}

type EmployeeAccountCheckRequest struct {
	EmployeeID uint `json:"employee_id"`
}

// --- Transaction wrapper ---

type TransactionWithLedgerResponse struct {
	Transaction   TransactionResponse   `json:"transaction"`
	LedgerEntries []LedgerEntryResponse `json:"ledger_entries"`
}

// --- Integration API (chatbot API-key channel) ---

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

type CreateAPIKeyResponse struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Key       string `json:"key"`
	KeyPrefix string `json:"key_prefix"`
}

type APIKeyListItem struct {
	ID        uint    `json:"id"`
	Name      string  `json:"name"`
	KeyPrefix string  `json:"key_prefix"`
	RevokedAt *string `json:"revoked_at"`
}

type IntegrationOTPRequest struct {
	Phone string `json:"phone"`
}

type IntegrationOTPResponse struct {
	Found             bool    `json:"found"`
	OTPSent           bool    `json:"otp_sent"`
	SessionID         string  `json:"session_id"`
	ExpiresIn         int     `json:"expires_in"`
	OTPLength         int     `json:"otp_length"`
	EmployeeName      string  `json:"employee_name"`
	FailureReason     *string `json:"failure_reason"`
	DeliveryErrorCode int     `json:"delivery_error_code"`
}
