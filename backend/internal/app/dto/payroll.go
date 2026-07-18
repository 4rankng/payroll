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

	// IfMatchSnapshot optionally carries the snapshot_epoch returned by a prior
	// /payrolls/simulate-settlement call. If set, Export rejects with 409
	// (domain.ErrStaleSimulation) when any relevant row's updated_at is newer
	// than this timestamp — preventing an export from silently using stale
	// simulation results. Absent = today's backward-compatible behavior.
	IfMatchSnapshot *time.Time `json:"if_match_snapshot,omitempty"`

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

// ============================================================================
// Settlement Simulation DTOs (Phase 2-3 — /payrolls/simulate-settlement)
//
// The simulation projects the current payroll cycle + next N−1 cycles by
// reusing the production ExportPlanner.Plan(). It returns a full-pool verdict
// ("will these N exports cover every outstanding approved timesheet?") plus
// a remainders list. Payroll-only scope: every remainder is unpaid wages
// (the eligible pool is payment_status IN (pending, failed) by construction).
// ============================================================================

// SimulateSettlementRequest is the request body for the simulation endpoint.
type SimulateSettlementRequest struct {
	ProjectIDs         []uint `json:"project_ids,omitempty"`
	EmployeeIDs        []uint `json:"employee_ids,omitempty"`
	ProjectedCycleCount int    `json:"projected_cycle_count,omitempty"` // default 4, clamped [1,6]
	ForMonth           string `json:"for_month,omitempty"`              // optional YYYY-MM override
	CreatedBy          uint   `json:"-"`                                // set from auth context
}

// SimulationResult is the full response payload.
type SimulationResult struct {
	SnapshotEpoch       time.Time           `json:"snapshot_epoch"`
	StartingCycle       CycleMeta           `json:"starting_cycle"`
	ProjectedCycleCount int                 `json:"projected_cycle_count"`
	Verdict             string              `json:"verdict"` // AN_TOAN_DE_XUAT | CAN_KIEM_TRA | KHONG_THE_TAT_TOAN
	Summary             SimulationSummary   `json:"summary"`
	Reconciliation      ReconciliationResult `json:"reconciliation"`
	Cycles              []CycleProjection   `json:"cycles"`
	Remainders          []RemainderRow      `json:"remainders"`
	Warnings            []SimWarning        `json:"warnings"`
}

// CycleMeta describes a cycle's window + pay date.
type CycleMeta struct {
	Index    int    `json:"index"`
	MonthRef string `json:"month_ref"`
	FromDate string `json:"from_date"`
	ToDate   string `json:"to_date"`
	PayDate  string `json:"pay_date"`
}

// SimulationSummary is the answer-first totals block.
type SimulationSummary struct {
	TotalEligibleCount     int   `json:"total_eligible_count"`
	TotalEligibleAmount    int64 `json:"total_eligible_amount"`
	TotalIncludedCount     int   `json:"total_included_count"`
	TotalIncludedAmount    int64 `json:"total_included_amount"`
	RemainingAfterAllCount int   `json:"remaining_after_all_count"`
	RemainingAfterAllAmount int64 `json:"remaining_after_all_amount"`
	AllSettled             bool  `json:"all_settled"`
}

// ReconciliationResult compares the projected export total against the ledger receivable.
type ReconciliationResult struct {
	ExportedTotal    int64 `json:"exported_total"`
	LedgerReceivable int64 `json:"ledger_receivable"`
	Delta            int64 `json:"delta"`
	Reconciled       bool  `json:"reconciled"`
}

// CycleProjection is one projected export batch.
type CycleProjection struct {
	Sequence        int                    `json:"sequence"`
	Label           string                 `json:"label"`
	FromDate        string                 `json:"from_date"`
	ToDate          string                 `json:"to_date"`
	PayDate         string                 `json:"pay_date"`
	IncludedCount   int                    `json:"included_count"`
	IncludedAmount  int64                  `json:"included_amount"`
	ExcludedCount   int                    `json:"excluded_count"`
	RemainingAfter  int                    `json:"remaining_after_count"`
	RemainingAmount int64                  `json:"remaining_after_amount"`
	Included        []SimulationRow        `json:"included"`
	Excluded        []SimulationExcludedRow `json:"excluded"`
	Findings        []SimFinding           `json:"findings"`
}

// SimulationRow is one included (valid, will-be-exported) row.
type SimulationRow struct {
	EmployeeID        uint   `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	ProjectID         uint   `json:"project_id"`
	ProjectName       string `json:"project_name"`
	Amount            int64  `json:"amount"`
	TimesheetIDs      []uint `json:"timesheet_ids"`
	BankAccountMasked string `json:"bank_account_masked"`
}

// SimulationExcludedRow is one row filtered out by validation.
type SimulationExcludedRow struct {
	EmployeeID   uint   `json:"employee_id"`
	EmployeeName string `json:"employee_name"`
	ProjectID    uint   `json:"project_id"`
	ProjectName  string `json:"project_name"`
	Amount       int64  `json:"amount"`
	TimesheetIDs []uint `json:"timesheet_ids"`
	Reason       string `json:"reason"`
	InProduction bool   `json:"in_production"`
}

// RemainderRow is an outstanding item not covered by any projected cycle.
// Always unpaid wages in payroll-only scope.
type RemainderRow struct {
	EmployeeID        uint   `json:"employee_id"`
	EmployeeName      string `json:"employee_name"`
	ProjectID         uint   `json:"project_id"`
	ProjectName       string `json:"project_name"`
	Amount            int64  `json:"amount"`
	TimesheetIDs      []uint `json:"timesheet_ids"`
	Reason            string `json:"reason"`
	BankAccountMasked string `json:"bank_account_masked"`
}

// SimFinding is a validation finding (sim-only or shared with production).
type SimFinding struct {
	Severity    string `json:"severity"`              // blocking | warning | info
	Code        string `json:"code"`
	Message     string `json:"message"`
	InProduction bool  `json:"in_production"`
}

// SimWarning is a top-level warning surfaced above the per-cycle findings.
type SimWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
