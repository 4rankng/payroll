package dto

import (
	"time"
)

// CreateTimesheetRequest represents the request to create a new timesheet
type CreateTimesheetRequest struct {
	ProjectID   uint    `json:"project_id" binding:"required"`
	EmployeeID  uint    `json:"employee_id" binding:"required"`
	Date        string  `json:"date" binding:"required"`
	HoursWorked float64 `json:"hours_worked" binding:"min=0"`
	HourType    string  `json:"hour_type" binding:"required"` // Ca ngày, Ca đêm, Làm thêm, or time ranges like "08:00-17:00"
	DayType     *string `json:"day_type,omitempty"`           // Optional: "Ngày thường" or "Ngày nghỉ" - backend auto-infers if not provided
}

// BulkCreateTimesheetRequest represents the request to create multiple timesheets
type BulkCreateTimesheetRequest []struct {
	ProjectID   uint    `json:"projectId" binding:"required"`
	EmployeeID  uint    `json:"employeeId" binding:"required"`
	Date        string  `json:"date" binding:"required"`
	HoursWorked float64 `json:"hoursWorked" binding:"min=0"`
	HourType    string  `json:"hourType" binding:"required"` // Required: "Ca ngày", "Ca đêm", "Làm thêm", or time ranges like "08:00-17:00"
	DayType     *string `json:"dayType,omitempty"`           // Optional: "Ngày thường" or "Ngày nghỉ" for Saturday
}

// UpdateTimesheetRequest represents the request to update a timesheet
type UpdateTimesheetRequest struct {
	ProjectID   *uint    `json:"project_id,omitempty"`
	EmployeeID  *uint    `json:"employee_id,omitempty"`
	Date        *string  `json:"date,omitempty"`
	HoursWorked *float64 `json:"hours_worked,omitempty" binding:"omitempty,min=0"`
	PayType     *string  `json:"paytype,omitempty"`
	PayrateID   *uint    `json:"payrate_id,omitempty"`
	PayRate     *float64 `json:"payrate,omitempty" binding:"omitempty,min=0"`
	Amount      *float64 `json:"amount,omitempty" binding:"omitempty,min=0"`
}

// RejectTimesheetRequest represents the request to reject a timesheet
type RejectTimesheetRequest struct {
	RejectionReason string `json:"rejection_reason" binding:"required"`
}

// TimesheetResponse represents the response containing timesheet data
type TimesheetResponse struct {
	ID               uint       `json:"id"`
	ProjectID        uint       `json:"project_id"`
	EmployeeID       uint       `json:"employee_id"`
	Date             string     `json:"date"`
	HoursWorked      float64    `json:"hours_worked"`
	PayType          string     `json:"paytype"`   // Full path: "position.dayType.hourType"
	HourType         string     `json:"hour_type"` // Extracted hour type
	DayType          string     `json:"day_type"`  // Extracted day type
	PayrateID        uint       `json:"payrate_id"`
	PayRate          float64    `json:"payrate"`
	Amount           float64    `json:"amount"`
	Status           string     `json:"status"`
	PaymentStatus    string     `json:"payment_status"`
	PaymentReference *string    `json:"payment_reference,omitempty"`
	ForcePayroll     bool       `json:"force_payroll"`
	AllowedEdit      bool       `json:"allowed_edit"`
	RequestEditID    *uint      `json:"request_edit_id"`
	PaidAmount       int64      `json:"paid_amount"`
	PaidAt           *time.Time `json:"paid_at"`
	CreatedBy        uint       `json:"created_by"`
	ApprovedBy       *uint      `json:"approved_by"`
	ApprovedAt       *time.Time `json:"approved_at"`
	RejectionReason  string     `json:"rejection_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// BulkCreateTimesheetResponse represents the response for bulk timesheet creation
type BulkCreateTimesheetResponse struct {
	Succeeded    []TimesheetResponse `json:"succeeded"`
	Deleted      []TimesheetResponse `json:"deleted"`
	Failed       []BulkCreateError   `json:"failed"`
	TotalSuccess int                 `json:"total_success"`
	TotalDeleted int                 `json:"total_deleted"`
	TotalFailed  int                 `json:"total_failed"`
}

// BulkCreateError represents an error in bulk creation
type BulkCreateError struct {
	Index   int                    `json:"index"`
	Error   string                 `json:"error"`
	Request CreateTimesheetRequest `json:"request"`
}

// BulkUpdateTimesheetRequest represents the request to update multiple timesheets
type BulkUpdateTimesheetRequest []struct {
	ID          uint    `json:"id" binding:"required"`
	HoursWorked float64 `json:"hours_worked" binding:"min=0"`
	PayType     string  `json:"paytype" binding:"required"`
}

// BulkUpdateTimesheetResponse represents the response for bulk timesheet updates
type BulkUpdateTimesheetResponse struct {
	Updated      []TimesheetResponse `json:"updated"`
	Failed       []BulkUpdateError   `json:"failed"`
	TotalUpdated int                 `json:"total_updated"`
	TotalFailed  int                 `json:"total_failed"`
}

// BulkUpdateError represents an error in bulk update
type BulkUpdateError struct {
	ID    uint   `json:"id"`
	Error string `json:"error"`
}

// ListTimesheetsResponse represents the response for listing timesheets
type ListTimesheetsResponse struct {
	Timesheets []TimesheetWithDetailsResponse `json:"timesheets"`
	Pagination PaginationResponse             `json:"pagination"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
	TotalPages   int   `json:"totalPages"`
	TotalRecords int64 `json:"totalRecords"`
}

// TimesheetSummaryResponse represents aggregated timesheet metrics
type TimesheetSummaryResponse struct {
	TotalEntries         int       `json:"totalEntries"`
	TotalEmployees       int       `json:"totalEmployees"`
	PendingApproval      int       `json:"pendingApproval"`
	PendingEmployees     int       `json:"pendingEmployees"`
	PendingPaymentAmount int64     `json:"pendingPaymentAmount"`
	ApprovedEntries      int       `json:"approvedEntries"`
	PaidEntries          int       `json:"paidEntries"`
	PaidAmount           int64     `json:"paidAmount"`
	PaidEmployees        int       `json:"paidEmployees"`
	RejectedEntries      int       `json:"rejectedEntries"`
	LastUpdated          time.Time `json:"lastUpdated"`
}

// TimesheetPreviewError represents a single validation error tied to an employee
type TimesheetPreviewError struct {
	EmployeeID uint   `json:"employeeId"`     // 0 means not tied to a specific employee
	Date       string `json:"date,omitempty"` // "YYYY-MM-DD" — the specific date that caused the error
	Message    string `json:"message"`
}

// TimesheetPreviewResponse represents the response for timesheet preview endpoint
type TimesheetPreviewResponse struct {
	Errors []TimesheetPreviewError `json:"errors"`
}

// BulkApproveRequest represents the request to approve multiple timesheets
type BulkApproveRequest struct {
	TimesheetIds []uint `json:"timesheet_ids" binding:"required"`
	ApprovalNote string `json:"approval_note,omitempty"`
}

// BulkRejectRequest represents the request to reject multiple timesheets
type BulkRejectRequest struct {
	TimesheetIds    []uint `json:"timesheet_ids" binding:"required"`
	RejectionReason string `json:"rejection_reason" binding:"required"`
	NotifyPartners  bool   `json:"notify_partners,omitempty"`
}

// BulkResetRequest represents the request to reset multiple timesheets back to pending approval
type BulkResetRequest struct {
	TimesheetIds []uint `json:"timesheet_ids" binding:"required"`
}

// ApproveAllRequest represents optional filters for the approve-all endpoint.
type ApproveAllRequest struct {
	ProjectIDs  []uint `json:"project_ids,omitempty"`
	EmployeeIDs []uint `json:"employee_ids,omitempty"`
	FromDate    string `json:"from_date,omitempty"`
	ToDate      string `json:"to_date,omitempty"`
}

// BulkOperationResponse represents the response for bulk operations
type BulkOperationResponse struct {
	Approved          int                   `json:"approved,omitempty"`
	Rejected          int                   `json:"rejected_count,omitempty"`
	Skipped           int                   `json:"skipped,omitempty"`
	Failed            int                   `json:"failed,omitempty"`
	FailedCount       int                   `json:"failed_count,omitempty"`
	NotificationsSent int                   `json:"notifications_sent,omitempty"`
	Results           []BulkOperationResult `json:"results"`
}

// BulkOperationResult represents individual result in bulk operation
type BulkOperationResult struct {
	ID              uint   `json:"id"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	Error           string `json:"error,omitempty"`
}

// TimesheetImportRequest represents Excel import request
type TimesheetImportRequest struct {
	ProjectID uint `json:"project_id" binding:"required"`
}

// TimesheetImportResponse represents Excel import response
type TimesheetImportResponse struct {
	BatchID           string    `json:"batchId"`
	TotalRecords      int       `json:"totalRecords"`
	SuccessfulRecords int       `json:"successfulRecords"`
	FailedRecords     int       `json:"failedRecords"`
	Status            string    `json:"status"`
	RequiresApproval  bool      `json:"requiresApproval"`
	ErrorSummary      string    `json:"errorSummary"`
	ImportedAt        time.Time `json:"importedAt"`
}

// TimesheetWithDetailsResponse includes project/employee names
type TimesheetWithDetailsResponse struct {
	TimesheetResponse
	ProjectName     string     `json:"projectName"`
	EmployeeName    string     `json:"employeeName"`
	EmployeeCode    string     `json:"employeeCode"`
	SubmittedBy     *uint      `json:"submittedBy,omitempty"`
	SubmittedByName *string    `json:"submittedByName,omitempty"`
	SubmittedAt     *time.Time `json:"submittedAt,omitempty"`
}

// EmployeeTimesheetSummaryResponse represents timesheet summary for an employee
type EmployeeTimesheetSummaryResponse struct {
	EmployeeID         uint               `json:"employeeId"`
	EmployeeName       string             `json:"employeeName"`
	TotalHours         map[string]float64 `json:"totalHours"`
	TotalAmount        int64              `json:"totalAmount"`
	AverageHoursPerDay float64            `json:"averageHoursPerDay"`
	WorkingDays        int                `json:"workingDays"`
	LastEntryDate      *string            `json:"lastEntryDate"`
	PendingEntries     int                `json:"pendingEntries"`
}

// EmployeeGroupedTimesheetResponse represents timesheets grouped by employee (for partner view)
type EmployeeGroupedTimesheetResponse struct {
	EmployeeID       uint                           `json:"employeeId"`
	EmployeeName     string                         `json:"employeeName"`
	EmployeeCode     string                         `json:"employeeCode"`
	Entries          []TimesheetWithDetailsResponse `json:"entries"`
	TotalHours       float64                        `json:"totalHours"`
	TotalAmount      float64                        `json:"totalAmount"`
	AggregatedStatus string                         `json:"aggregatedStatus"`
}

// ListGroupedTimesheetsResponse represents the response for listing grouped timesheets
type ListGroupedTimesheetsResponse struct {
	Groups []EmployeeGroupedTimesheetResponse `json:"groups"`
}
