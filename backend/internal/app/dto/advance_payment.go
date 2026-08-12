package dto

import "time"

// === Employee DTOs ===

// AdvancePaymentInfoResponse represents the advance payment info for an employee
type AdvancePaymentInfoResponse struct {
	ForMonth         string                        `json:"forMonth"`
	MaxAdvanceAmount uint64                        `json:"maxAdvanceAmount"`
	CompletedAmount  uint64                        `json:"completedAmount"`
	PendingAmount    uint64                        `json:"pendingAmount"`
	RemainingAmount  uint64                        `json:"remainingAmount"`
	CanRequest       bool                          `json:"canRequest"`
	CanRequestTitle  string                        `json:"canRequestTitle,omitempty"`
	CanRequestReason string                        `json:"canRequestReason,omitempty"`
	FeePercentage    float64                       `json:"feePercentage"`
	MinFee           uint64                        `json:"minFee"`
	HasFlexible      bool                          `json:"hasFlexible"`
	Quotas           []AdvancePaymentQuotaResponse `json:"quotas"`
	// ProviderMinTransferAmount is the minimum amount the disbursement provider
	// will accept for a single transfer. The frontend must ensure
	// requestAmount - fee >= this value. Zero means no limit.
	ProviderMinTransferAmount uint64 `json:"providerMinTransferAmount,omitempty"`
	// ProviderMaxTransferAmount is the maximum amount the disbursement provider
	// will accept for a single transfer. Zero means no limit.
	ProviderMaxTransferAmount uint64 `json:"providerMaxTransferAmount,omitempty"`

	// Salary is the total earned wages (100%) for the self-check-in flow. Populated
	// only by the dedicated /me/check-in-advance endpoint; the admin-upload
	// /me/advance-payment endpoint leaves it zero (omitempty).
	Salary uint64 `json:"salary,omitempty"`
	// PendingEarnings is the total earning held in the 24h credit window this
	// month — checked-out shifts whose earning has not yet been banked into the
	// quota pool. Displayed separately ("Đang chờ 24h"); NOT part of the
	// advanceable cap. Self-check-in flow only (omitempty).
	PendingEarnings uint64 `json:"pendingEarnings,omitempty"`
	// Disclaimer is the in-app note for the self-check-in flow clarifying that
	// the displayed wages exclude overtime and company allowances.
	Disclaimer string `json:"disclaimer,omitempty"`
	// WindowOpenDay is the day of month the self-check-in request window opens.
	WindowOpenDay int `json:"windowOpenDay,omitempty"`
}

type AdvancePaymentQuotaResponse struct {
	ForMonth         string `json:"forMonth"`
	MaxAdvanceAmount uint64 `json:"maxAdvanceAmount"`
	CompletedAmount  uint64 `json:"completedAmount"`
	PendingAmount    uint64 `json:"pendingAmount"`
	RemainingAmount  uint64 `json:"remainingAmount"`
}

// CreateAdvancePaymentRequest represents a request to create an advance payment
type CreateAdvancePaymentRequest struct {
	Amount   uint64 `json:"amount" binding:"required,min=10000"`
	ForMonth string `json:"forMonth"`
}

// CalculateFeeRequest represents a fee calculation preview request
type CalculateFeeRequest struct {
	Amount uint64 `json:"amount" binding:"required,min=10000"`
}

// CalculateFeeResponse represents the fee calculation response
type CalculateFeeResponse struct {
	RequestAmount uint64 `json:"requestAmount"`
	Fee           uint64 `json:"fee"`
	NetAmount     uint64 `json:"netAmount"`
}

// AdvancePaymentHistoryItem represents a history item
type AdvancePaymentHistoryItem struct {
	ID            uint       `json:"id"`
	RequestAmount uint64     `json:"requestAmount"`
	Fee           uint64     `json:"fee"`
	NetAmount     uint64     `json:"netAmount"`
	Status        string     `json:"status"`
	ForMonth      string     `json:"forMonth"`
	CreatedAt     time.Time  `json:"createdAt"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	ProjectName   string     `json:"projectName"`
	ProjectCode   string     `json:"projectCode"`
}

// AdvancePaymentHistoryResponse represents the history response
type AdvancePaymentHistoryResponse struct {
	Data       []AdvancePaymentHistoryItem `json:"data"`
	Pagination PaginationResponse          `json:"pagination"`
}

// === Admin DTOs ===

// AdvancePaymentRequestItem represents an item in the list
type AdvancePaymentRequestItem struct {
	ID               uint       `json:"id"`
	EmployeeID       uint       `json:"employeeId"`
	EmployeeName     string     `json:"employeeName,omitempty"`
	EmployeeCCCD     string     `json:"employeeCCCD,omitempty"`
	ProjectID        uint       `json:"projectId"`
	ProjectCode      string     `json:"projectCode,omitempty"`
	ProjectName      string     `json:"projectName,omitempty"`
	RequestAmount    uint64     `json:"requestAmount"`
	Fee              uint64     `json:"fee"`
	NetAmount        uint64     `json:"netAmount"`
	Status           string     `json:"status"`
	PaymentReference *string    `json:"paymentReference,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	PaidAt           *time.Time `json:"paidAt,omitempty"`
	// SettlementTransactionID is the transaction ID from the bulk transfer
	// that settled this request. Null if not yet settled.
	SettlementTransactionID *uint `json:"settlementTransactionId,omitempty"`
	// ProviderFee is the fee charged by the disbursement provider (9Pay) for
	// this request. Zero for manually-disbursed requests.
	ProviderFee uint64 `json:"providerFee"`
}

// ListAdvancePaymentsResponse represents the list response
type ListAdvancePaymentsResponse struct {
	Data       []AdvancePaymentRequestItem `json:"data"`
	Pagination PaginationResponse          `json:"pagination"`
}

// AdvancePaymentSummaryResponse represents the admin summary response
type AdvancePaymentSummaryResponse struct {
	TotalRequests           int64   `json:"totalRequests"`
	TotalPending            int64   `json:"totalPending"`
	TotalApproved           int64   `json:"totalApproved"`
	TotalCancelled          int64   `json:"totalCancelled"`
	TotalFailed             int64   `json:"totalFailed"`
	TotalPaid               int64   `json:"totalPaid"`
	TotalAmount             uint64  `json:"totalAmount"`
	TotalPaidAmount         uint64  `json:"totalPaidAmount"`
	TotalPendingAmount      uint64  `json:"totalPendingAmount"`
	TotalFailedAmount       uint64  `json:"totalFailedAmount"`
	TotalCancelledAmount    uint64  `json:"totalCancelledAmount"`
	TotalFee                uint64  `json:"totalFee"`
	TotalFeeEarned          uint64  `json:"totalFeeEarned"`
	TotalFeeEarnedAllTime   uint64  `json:"totalFeeEarnedAllTime"`
	TotalNet                uint64  `json:"totalNet"`
	TotalProviderFee        uint64  `json:"totalProviderFee"`
	TotalProviderFeeAllTime uint64  `json:"totalProviderFeeAllTime"`
	AvgProcessingTimeSecs   float64 `json:"avgProcessingTimeSecs"`
	CompletedUnder30s       int64   `json:"completedUnder30s"`
	FeePercentage           float64 `json:"feePercentage"`
	AvgFeePerRequest        uint64  `json:"avgFeePerRequest"`
	AvgFeePerEmployee       uint64  `json:"avgFeePerEmployee"`
	SuccessRate             float64 `json:"successRate"`
	DisbursementPercentage  float64 `json:"disbursementPercentage"`
	FromDate                *string `json:"fromDate,omitempty"`
	ToDate                  *string `json:"toDate,omitempty"`
}

// ImportFlexPayFileRequest represents the import request
type ImportFlexPayFileRequest struct {
	ForMonth string `form:"forMonth" binding:"required,len=7"` // YYYY-MM
}

// ImportFlexPayFileResult represents the import result statistics
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
	// EmployeeZNSData holds employee data for ZNS notifications after successful import
	EmployeeZNSData []EmployeeZNSData `json:"-"`
}

// EmployeeZNSData holds employee data for ZNS notification
type EmployeeZNSData struct {
	ProjectID    uint
	EmployeeID   uint
	EmployeeName string    `json:"employeeName"`
	Mobile       string    `json:"mobile"`
	Amount       int64     `json:"amount"`
	ExpiryDate   time.Time `json:"expiryDate"`
}

// ImportJobResponse represents the response when starting an import job
type ImportJobResponse struct {
	ID        uint   `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// ImportJobStatusResponse represents the status of an import job
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
	CreatedAt     string                   `json:"createdAt"`
	StartedAt     *string                  `json:"startedAt,omitempty"`
	CompletedAt   *string                  `json:"completedAt,omitempty"`
}

// ExportAdvancePaymentsRequest represents the export request
type ExportAdvancePaymentsRequest struct {
	FromDate string `form:"fromDate"` // YYYY-MM-DD
	ToDate   string `form:"toDate"`   // YYYY-MM-DD
}

// ListAdvancePaymentsRequest represents query parameters for listing
type ListAdvancePaymentsRequest struct {
	Page       int    `form:"page,default=1"`
	PageSize   int    `form:"pageSize,default=20"`
	Status     string `form:"status"`
	ProjectID  string `form:"projectId"`
	EmployeeID string `form:"employeeId"`
	FromDate   string `form:"fromDate"`
	ToDate     string `form:"toDate"`
	Search     string `form:"search"`
}

// EmployeeAdvanceBankInfo represents bank information for an employee
type EmployeeAdvanceBankInfo struct {
	BankID        uint   `json:"bankId"`
	BankName      string `json:"bankName"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
}

// EmployeeAdvanceProjectInfo represents project information for an employee
type EmployeeAdvanceProjectInfo struct {
	ID             uint   `json:"id"`
	Name           string `json:"name"`
	Code           string `json:"code"`
	AssignmentID   uint   `json:"assignment_id"`
	CheckInEnabled bool   `json:"check_in_enabled"`
}

// EmployeeAdvanceItem represents an employee with advance payment statistics
type EmployeeAdvanceItem struct {
	EmployeeID             uint                        `json:"employeeId"`
	Fullname               string                      `json:"fullname"`
	CCCD                   string                      `json:"cccd"`
	Username               string                      `json:"username"`
	Email                  *string                     `json:"email,omitempty"`
	Bank                   *EmployeeAdvanceBankInfo    `json:"bank"`
	Project                *EmployeeAdvanceProjectInfo `json:"project"`
	ForMonth               string                      `json:"forMonth"`
	MaxAdvanceAmount       uint64                      `json:"maxAdvanceAmount"`
	UtilizedAmount         uint64                      `json:"utilizedAmount"`
	AvailableAmount        uint64                      `json:"availableAmount"`
	TotalFeeGenerated      uint64                      `json:"totalFeeGenerated"`
	PendingAmount          uint64                      `json:"pendingAmount"`
	CompletedRequestsCount int                         `json:"completedRequestsCount"`
	PendingRequestsCount   int                         `json:"pendingRequestsCount"`
	CreatedAt              *time.Time                  `json:"createdAt,omitempty"`
}

// EmployeeAdvanceListResponse represents the response for the employee advance list
type EmployeeAdvanceListResponse struct {
	ForMonth   string                  `json:"forMonth"`
	Data       []EmployeeAdvanceItem   `json:"data"`
	Pagination EmployeeAdvanceListMeta `json:"pagination"`
}

// EmployeeAdvanceListMeta represents the pagination for the employee advance list
type EmployeeAdvanceListMeta struct {
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
	TotalPages   int   `json:"totalPages"`
	TotalRecords int64 `json:"totalRecords"`
}

// EmployeeAdvanceFilters represents filters for the employee advance list
type EmployeeAdvanceFilters struct {
	ForMonth  *string
	Search    string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// AvailableMonth represents a month with flex pay data
type AvailableMonth struct {
	ForMonth      string `json:"forMonth"`
	EmployeeCount int    `json:"employeeCount"`
}

// === Transfer Result DTOs ===

// AdvancePaymentUploadResultResponse represents the response from processing advance payment bank results
type AdvancePaymentUploadResultResponse struct {
	Data         []AdvancePaymentResultItem `json:"data"`
	TotalTxn     int                        `json:"total_txn"`
	CompletedTxn int                        `json:"completed_txn"`
	FailedTxn    int                        `json:"failed_txn"`
}

// AdvancePaymentResultItem represents a single transaction result
type AdvancePaymentResultItem struct {
	Row                   int     `json:"row"`
	EmployeeName          string  `json:"employee_name"`
	EmployeeBank          string  `json:"employee_bank"`
	EmployeeAccountNumber string  `json:"employee_account_number"`
	EmployeeCCCD          string  `json:"employee_cccd"`
	Amount                string  `json:"amount"`
	PaymentStatus         string  `json:"payment_status"` // "paid" | "failed"
	PaidAt                *string `json:"paid_at,omitempty"`
}

// === Transfer History DTOs ===

// ImportFlexibleEmployeeListResult represents the import result for employee list
type ImportFlexibleEmployeeListResult struct {
	TotalRows          int              `json:"total_rows"`
	EmployeesCreated   int              `json:"employees_created"`
	EmployeesUpdated   int              `json:"employees_updated"`
	EmployeesSkipped   int              `json:"employees_skipped"`
	ProjectsCreated    int              `json:"projects_created"`
	ProjectsSkipped    int              `json:"projects_skipped"`
	AssignmentsCreated int              `json:"assignments_created"`
	AssignmentsSkipped int              `json:"assignments_skipped"`
	Errors             []ImportRowError `json:"errors,omitempty"`
}

// ImportRowError represents an error in a specific row
type ImportRowError struct {
	Row   int    `json:"row"`
	CCCD  string `json:"cccd"`
	Error string `json:"error"`
}

// ReconciliationSettlementResult represents the result of processing a reconciliation settlement file
type ReconciliationSettlementResult struct {
	SettledCount int64     `json:"settled_count"`
	RequestIDs   []uint64  `json:"request_ids,omitempty"`
	SettledAt    time.Time `json:"settled_at"`
}

// ReconciliationEmailResult represents the result of sending a reconciliation email
type ReconciliationEmailResult struct {
	EmailID        string `json:"email_id"`
	CancelledCount int64  `json:"cancelled_count"`
}

// UploadedFileResponse represents a single uploaded file record in list responses
type UploadedFileResponse struct {
	ID         uint      `json:"id"`
	Filename   string    `json:"filename"`
	UploadType string    `json:"uploadType"`
	CreatedAt  time.Time `json:"createdAt"`
	UploadedBy string    `json:"uploadedBy"`
}
