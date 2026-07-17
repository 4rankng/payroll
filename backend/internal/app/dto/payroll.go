package dto

import (
	"encoding/json"
	"strings"
	"time"
)

// ExportBulkTransferRequest represents the request to export bulk transfer data
type ExportBulkTransferRequest struct {
	ProjectIDs  []uint `json:"project_ids,omitempty"`
	EmployeeIDs []uint `json:"employee_ids,omitempty"`
	FromDate    string `json:"fromDate,omitempty"`
	ToDate      string `json:"toDate,omitempty"`
	ForMonth    string `json:"for_month,omitempty"`
	CreatedBy   uint   `json:"-"` // Set from context, not from request body

	// NoDateFilter is set ONLY by the settlement simulation's full-pool scan
	// (Phase 2). Production export never sets it. When true, ExportPlanner.Plan
	// skips date resolution and selects every outstanding approved timesheet
	// (payment_status pending|failed) matching only the project/employee
	// filters — so the simulation can compute the full-pool coverage verdict
	// without a second selection algorithm. Exported so the simulation service
	// (different package) can set it; the json:"-" tag keeps it off the wire.
	NoDateFilter bool `json:"-"`
}

// ExportBulkTransferResponse represents the response from bulk transfer export
type ExportBulkTransferResponse struct {
	Data             []byte                 `json:"-"` // Download bytes (XLSX or ZIP, not serialized to JSON)
	ContentType      string                 `json:"-"`
	FileExtension    string                 `json:"-"`
	Files            []BulkTransferFileInfo `json:"files"`
	SkippedEmployees []SkippedEmployeeInfo  `json:"skipped_employees,omitempty"`
	TotalEmployees   int                    `json:"total_employees"`
	IncludedCount    int                    `json:"included_count"`
	SkippedCount     int                    `json:"skipped_count"`
	FromDate         string                 `json:"fromDate"`
	ToDate           string                 `json:"toDate"`
	Cycle            string                 `json:"cycle"`
	Filename         string                 `json:"filename,omitempty"`
}

// BulkTransferFileInfo represents information about a file in the bulk transfer
type BulkTransferFileInfo struct {
	BankID        uint   `json:"bank_id"`
	BankPrefix    string `json:"bank_prefix"`
	Filename      string `json:"filename"`
	EmployeeCount int    `json:"employee_count"`
}

// SkippedEmployeeInfo represents information about employees skipped due to missing bank info
type SkippedEmployeeInfo struct {
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	Reason       string `json:"reason"`
	ProjectID    uint   `json:"project_id"`
	ProjectName  string `json:"project_name"`
}

// BulkTransferResultResponse represents the response from processing bulk transfer results
type BulkTransferResultResponse struct {
	Data         []BulkTransferResultItem `json:"data"`
	TotalTxn     int                      `json:"total_txn"`
	CompletedTxn int                      `json:"completed_txn"`
	FailedTxn    int                      `json:"failed_txn"`
}

// BulkTransferResultItem represents a single transaction in the bulk transfer result
type BulkTransferResultItem struct {
	Row                   int     `json:"row"`
	EmployeeBank          string  `json:"employee_bank"`
	EmployeeAccountNumber string  `json:"employee_account_number"`
	EmployeeName          string  `json:"employee_name"`
	EmployeeCCCD          string  `json:"employee_cccd"`
	Amount                string  `json:"amount"`
	PaymentStatus         string  `json:"payment_status"` // "paid" or "failed"
	PaidAt                *string `json:"paid_at,omitempty"`
}

// BulkTransferResultError represents an error during bulk transfer processing (kept for internal use)
type BulkTransferResultError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
	Value   string `json:"value"`
}

// BulkTransferResultItemDetail represents details of a processed bulk transfer item (kept for internal use)
type BulkTransferResultItemDetail struct {
	Row                   int    `json:"row"`
	EmployeeAccountNumber string `json:"employee_account_number"`
	EmployeeAccountName   string `json:"employee_account_name"`
	EmployeeID            *uint  `json:"employee_id,omitempty"`
	EmployeeFullname      string `json:"employee_fullname"`
	Amount                string `json:"amount"`
	PaymentDescription    string `json:"payment_description"`
	TransferStatus        string `json:"transfer_status"`
	PaymentStatus         string `json:"payment_status"`
	TimesheetsUpdated     int    `json:"timesheets_updated"`
	Status                string `json:"status"`
	Message               string `json:"message"`
	TrackingData          string `json:"tracking_data"`
}

// BulkTransferResultMetadata stores result summary and details in Asset.Metadata JSON
type BulkTransferResultMetadata struct {
	TotalTxn     int                             `json:"total_txn"`
	CompletedTxn int                             `json:"completed_txn"`
	FailedTxn    int                             `json:"failed_txn"`
	Data         []BulkTransferHistoryDetailItem `json:"data"`
}

// BulkTransferHistorySummary represents a summary of a bulk transfer upload history
type BulkTransferHistorySummary struct {
	ID           uint   `json:"id"`
	Filename     string `json:"filename"`
	TotalTxn     int    `json:"total_txn"`
	CompletedTxn int    `json:"completed_txn"`
	FailedTxn    int    `json:"failed_txn"`
	UploadedBy   string `json:"uploaded_by"`
	UploadedAt   string `json:"uploaded_at"`
}

// BulkTransferHistoryDetail represents the detailed processing result of a bulk transfer upload
type BulkTransferHistoryDetail struct {
	Data         []BulkTransferHistoryDetailItem `json:"data"`
	TotalTxn     int                             `json:"total_txn"`
	CompletedTxn int                             `json:"completed_txn"`
	FailedTxn    int                             `json:"failed_txn"`
}

// BulkTransferHistoryDetailItem represents a single transaction detail in the bulk transfer history
type BulkTransferHistoryDetailItem struct {
	Row                   int     `json:"row"`
	EmployeeBank          string  `json:"employee_bank"`
	EmployeeBankCode      string  `json:"employee_bank_code"`
	EmployeeAccountNumber string  `json:"employee_account_number"`
	EmployeeName          string  `json:"employee_name"`
	EmployeeCCCD          string  `json:"employee_cccd"`
	Amount                string  `json:"amount"`
	PaymentStatus         string  `json:"payment_status"`
	PaidAt                *string `json:"paid_at,omitempty"`
}

// ListBulkTransferHistoriesRequest represents the request to list bulk transfer histories with pagination
type ListBulkTransferHistoriesRequest struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"pageSize,default=20" binding:"min=1,max=100"`
	SortBy    string `form:"sortBy,default=created_at"`
	SortOrder string `form:"sortOrder,default=desc" binding:"omitempty,oneof=asc desc"`
	FromDate  string `form:"fromDate"` // YYYY-MM-DD format
	ToDate    string `form:"toDate"`   // YYYY-MM-DD format
}

// ListBulkTransferHistoriesResponse represents the response for bulk transfer histories list
type ListBulkTransferHistoriesResponse struct {
	Data       []BulkTransferHistorySummary `json:"data"`
	Pagination PaginationResponse           `json:"pagination"`
}

// ListPayrollHistoriesRequest represents the request to list payment histories with pagination
type ListPayrollHistoriesRequest struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	PageSize   int    `form:"pageSize,default=100" binding:"min=1,max=100"`
	SortBy     string `form:"sortBy,default=paid_date"`
	SortOrder  string `form:"sortOrder,default=desc" binding:"omitempty,oneof=asc desc"`
	FromDate   string `form:"fromDate"`   // YYYY-MM-DD format
	ToDate     string `form:"toDate"`     // YYYY-MM-DD format
	ProjectID  []uint `form:"projectId"`  // Array of project IDs
	EmployeeID []uint `form:"employeeId"` // Array of employee IDs
	Position   string `form:"position"`   // Employee position filter
	Search     string `form:"search"`     // Search across employee fullname and CCCD
}

// PayrollHistoryItem represents a single payment history record
type PayrollHistoryItem struct {
	EmployeeID      uint   `json:"employee_id"`
	EmployeeName    string `json:"employee_name"`
	EmployeeCCCD    string `json:"employee_cccd"`
	ProjectID       uint   `json:"project_id"`
	ProjectName     string `json:"project_name"`
	Position        string `json:"position"`
	TotalPaidAmount int64  `json:"total_paid_amount"`
	PaidDate        string `json:"paid_date"` // ISO 8601 format
}

// ListPayrollHistoriesResponse represents the response for payment histories list
type ListPayrollHistoriesResponse struct {
	Data       []PayrollHistoryItem `json:"data"`
	Pagination PaginationResponse   `json:"pagination"`
}

// BulkTransferSuccessfulPayment represents a single successful payment in the bulk transfer result
type BulkTransferSuccessfulPayment struct {
	Amount                string `json:"amount"`
	ProjectID             uint   `json:"project_id"`
	EmployeeID            uint   `json:"employee_id"`
	ProjectName           string `json:"project_name"`
	EmployeeCCCD          string `json:"employee_cccd"`
	TimesheetIDs          []uint `json:"timesheet_ids"`
	EmployeeFullname      string `json:"employee_fullname"`
	PaymentDescription    string `json:"payment_description"`
	EmployeeAccountName   string `json:"employee_account_name"`
	EmployeeAccountNumber string `json:"employee_account_number"`
}

// BulkTransferFileData represents a single transaction in the bulk transfer file data
type BulkTransferFileData struct {
	STT             int     `json:"stt"`
	EmployeeID      uint    `json:"employee_id"`
	ProjectID       uint    `json:"project_id"`
	TimesheetIDs    []uint  `json:"timesheet_ids,omitempty"`
	AdvPayReqIDs    []uint  `json:"adv_pay_req_ids,omitempty"` // For flexible cycle
	AccountNumber   string  `json:"account_number"`
	AccountName     string  `json:"account_name"`
	BankName        string  `json:"bank_name"`
	BankCode        string  `json:"bank_code,omitempty"`
	Amount          int64   `json:"amount"`
	TransactionCode string  `json:"transaction_code"`
	TransferStatus  string  `json:"transfer_status,omitempty"` // "completed" or "failed"
	BankTxnRef      string  `json:"bank_txn_ref,omitempty"`    // Bank reference number for completed
	ErrorMessage    string  `json:"error_message,omitempty"`   // Error message for failed
	UploadedAt      *string `json:"uploaded_at,omitempty"`     // Payment date stored per record
}

// ParseBulkTransferFileData converts stored JSON into a transaction slice and treats "{}" as empty data.
func ParseBulkTransferFileData(raw string) ([]BulkTransferFileData, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || trimmed == "{}" {
		return []BulkTransferFileData{}, nil
	}

	var data []BulkTransferFileData
	if err := json.Unmarshal([]byte(trimmed), &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ExportPayrollHistoriesRequest represents the request to export payroll histories
type ExportPayrollHistoriesRequest struct {
	FromDate string `json:"fromDate" binding:"required"` // YYYY-MM-DD format
	ToDate   string `json:"toDate" binding:"required"`   // YYYY-MM-DD format
}

// Validate validates the export request parameters
func (r *ExportPayrollHistoriesRequest) Validate() error {
	// Validate date format
	if _, err := time.Parse("2006-01-02", r.FromDate); err != nil {
		return ValidationError("invalid fromDate format, expected YYYY-MM-DD")
	}

	if _, err := time.Parse("2006-01-02", r.ToDate); err != nil {
		return ValidationError("invalid toDate format, expected YYYY-MM-DD")
	}

	// Validate date range
	from, _ := time.Parse("2006-01-02", r.FromDate)
	to, _ := time.Parse("2006-01-02", r.ToDate)
	if from.After(to) {
		return ValidationError("fromDate cannot be after toDate")
	}

	return nil
}

// ValidationError represents a validation error
type ValidationError string

func (e ValidationError) Error() string {
	return string(e)
}

// TimesheetImportResult represents the result of a timesheet template import
type TimesheetImportResult struct {
	CreatedCount  int                      `json:"created_count"`
	SkippedCount  int                      `json:"skipped_count"`
	ErrorCount    int                      `json:"error_count"`
	FailedEntries []TimesheetImportFailure `json:"failed_entries,omitempty"`
}

// TimesheetImportFailure represents a single failed entry during timesheet import
type TimesheetImportFailure struct {
	Index int    `json:"index"`
	Error string `json:"error"`
}

// MarkExternallyPaidRequest is the request body for marking timesheets as paid externally.
type MarkExternallyPaidRequest struct {
	TimesheetIDs []uint `json:"timesheet_ids" binding:"required,min=1"`
	Reference    string `json:"reference" binding:"required"`
	Note         string `json:"note,omitempty"`
}

// MarkExternallyPaidResponse is the response for marking timesheets as paid externally.
type MarkExternallyPaidResponse struct {
	MarkedCount int `json:"marked_count"`
}

// BulkTransferFailedItem represents a single failed transfer row in a batch status response.
type BulkTransferFailedItem struct {
	TimesheetIDs []uint `json:"timesheet_ids"`
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name,omitempty"`
	Amount       int64  `json:"amount"`
	Reason       string `json:"reason,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
}

// EstimateFeeRequest is the request body for estimating 9Pay disbursement fees.
type EstimateFeeRequest struct {
	ProjectIDs  []uint `json:"project_ids,omitempty"`
	EmployeeIDs []uint `json:"employee_ids,omitempty"`
	ForMonth    string `json:"for_month,omitempty"`
	FromDate    string `json:"from_date,omitempty"`
	ToDate      string `json:"to_date,omitempty"`
}

// EstimateFeeResponse is the response for fee estimation.
type EstimateFeeResponse struct {
	TotalFee       int64 `json:"total_fee"`
	TransferCount  int   `json:"transfer_count"`
	FeePerTransfer int64 `json:"fee_per_transfer"`
}
