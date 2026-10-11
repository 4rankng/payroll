package domain

import (
	"context"
	"time"
)

// ChatDataScope is the data scope a chat tool executes against. It is derived
// from the CALLER's identity at call time — never from conversation content —
// so a prompt-injected message can widen nobody's access. For admin the ID
// sets are empty (company-wide); for partner they are the materialised
// PartnerScope sets (see infra/persistence/repositories/partner_scope.go).
type ChatDataScope struct {
	UserID      uint
	Role        string
	EmployeeIDs []uint
	ProjectIDs  []uint
}

// Admin reports whether the scope is company-wide.
func (s ChatDataScope) Admin() bool { return s.Role == "admin" }

// ChatTimesheetSummary is the deterministic timesheet aggregate returned by the
// get_timesheet_summary tool. Counts and money are computed here — the LLM
// only narrates them (ADR-010 boundary).
type ChatTimesheetSummary struct {
	Month              string
	TotalRows          int64
	PendingRows        int64
	ApprovedRows       int64
	RejectedRows       int64
	PaidRows           int64
	ApprovedUnpaidRows int64
	TotalAmount        int64
	PaidAmount         int64
}

// ChatImportJobStatus is one timesheet import job (BCC Excel) as seen by the
// get_import_status tool. ErrorDetailJSON is the raw stored error detail; the
// service normalises it through the import-error catalog before it ever
// reaches a prompt.
type ChatImportJobStatus struct {
	AssetID         uint
	ProjectID       uint
	OriginalName    string
	ForMonth        string
	Status          string
	TotalRows       int
	CreatedCount    int
	SkippedCount    int
	ErrorCount      int
	LastError       string
	ErrorDetailJSON string
	ProcessedAt     *time.Time
	CreatedAt       time.Time
}

// ChatAdvanceSummary is the deterministic advance-payment aggregate returned by
// the get_advance_summary tool.
type ChatAdvanceSummary struct {
	Month             string
	EmployeeCount     int64
	TotalSalary       int64
	MaxAdvance        int64
	RequestedAmount   int64
	NetPaidAmount     int64
	PendingRequests   int64
	CompletedRequests int64
	FailedRequests    int64
}

// ChatAdvanceRecipient is one employee that actually RECEIVED an advance
// payment (a COMPLETED request) inside the queried period. The bot answers
// "how many employees took an advance this year" from this list — never from
// a roster count, which would over-report people who only had a limit set.
type ChatAdvanceRecipient struct {
	EmployeeID     uint   `json:"employee_id"`
	FullName       string `json:"full_name"`
	RequestCount   int64  `json:"request_count"`
	NetPaidAmount  int64  `json:"net_paid_amount"`
	FirstPaidMonth string `json:"first_paid_month"`
	LastPaidMonth  string `json:"last_paid_month"`
}

// ChatAdvanceRecipients is the deterministic answer to "who took an advance in
// <period>" — the distinct employee list plus the period actually scanned.
type ChatAdvanceRecipients struct {
	Period     string
	From       string // "YYYY-MM" inclusive
	To         string // "YYYY-MM" inclusive
	TotalCount int64
	Rows       []*ChatAdvanceRecipient
}

// ChatReportRow is one row of a curated admin report. Column names come from a
// fixed allow-list in the infra implementation — never from model text.
type ChatReportRow struct {
	Label    string `json:"label"`
	Count    int64  `json:"count"`
	Amount   int64  `json:"amount"`
	Detail   string `json:"detail"`
	PeriodAt string `json:"period_at"`
}

// ChatReportResult is the bounded result of a curated read-only report.
type ChatReportResult struct {
	ReportKey  string
	PeriodFrom string
	PeriodTo   string
	Rows       []*ChatReportRow
	Truncated  bool
}

// ChatReportSpec is one entry of the curated allow-list. SQL is a constant in
// infra; the model may only CHOOSE a key and pass bounded filters.
type ChatReportSpec struct {
	Key         string
	Title       string
	Description string
}

// ChatListRow is one row of a scoped operational list. A single shape backs
// every list tool so the handlers stay uniform and the SQL stays auditable.
type ChatListRow struct {
	Label      string
	SubLabel   string
	EmployeeID uint
	ProjectID  uint
	Count      int64
	Amount     int64
	Status     string
	Detail     string
	PeriodAt   string
}

// ChatScopedList is a bounded, scope-filtered list plus the window it covers.
type ChatScopedList struct {
	Period     string
	From       string
	To         string
	TotalCount int64
	Rows       []*ChatListRow
}

// ChatToolReadStore is the read-only data seam behind the assistant tools. Every
// method receives the caller-derived scope and MUST apply it to every row it
// returns; implementations live in infra/persistence.
type ChatToolReadStore interface {
	// TimesheetSummary aggregates timesheets for the scope. month is "YYYY-MM"
	// or "" for the last 30 days.
	TimesheetSummary(ctx context.Context, scope ChatDataScope, month string) (*ChatTimesheetSummary, error)
	// RecentImportJobs returns the most recent import jobs visible to the
	// scope (admin: all uploaders; partner: own uploads), newest first.
	RecentImportJobs(ctx context.Context, scope ChatDataScope, limit int) ([]*ChatImportJobStatus, error)
	// AdvanceSummary aggregates advance payments for the scope. month is
	// "YYYY-MM" or "" for the current month.
	AdvanceSummary(ctx context.Context, scope ChatDataScope, month string) (*ChatAdvanceSummary, error)
	// EmployeeDirectory lists scope-visible employees with the total count.
	EmployeeDirectory(ctx context.Context, scope ChatDataScope, limit int) ([]*ChatEmployeeEntry, int64, error)
	// AdvanceRecipients lists DISTINCT employees with a COMPLETED advance
	// request in [fromMonth, toMonth] (inclusive, "YYYY-MM"). Both bounds empty
	// means the current month; from > to is normalised.
	AdvanceRecipients(ctx context.Context, scope ChatDataScope, fromMonth, toMonth string, limit int) (*ChatAdvanceRecipients, error)
	// RunCuratedReport executes one allow-listed read-only report. The key MUST
	// exist in the registry; unknown keys are refused, never interpolated.
	RunCuratedReport(ctx context.Context, scope ChatDataScope, spec ChatReportSpec, fromMonth, toMonth string, limit int) (*ChatReportResult, error)
	// MissingTimesheets lists active project assignments with no timesheet row
	// at all in the given month. month is "YYYY-MM" or "" for the current month.
	MissingTimesheets(ctx context.Context, scope ChatDataScope, month string, limit int) (*ChatScopedList, error)
	// AdvanceLimitBreaches lists advance rows whose summed live requests exceed
	// the configured max_adv_amount in [fromMonth, toMonth].
	AdvanceLimitBreaches(ctx context.Context, scope ChatDataScope, fromMonth, toMonth string, limit int) (*ChatScopedList, error)
	// ProjectCashflow aggregates paid timesheet amount and paid advance amount
	// per project over [fromMonth, toMonth].
	ProjectCashflow(ctx context.Context, scope ChatDataScope, fromMonth, toMonth string, limit int) (*ChatScopedList, error)
	// Disbursements lists wallet payment rows matching an optional txn/request
	// id and optional status. Admin-scoped (company-wide by definition).
	Disbursements(ctx context.Context, scope ChatDataScope, reference, status string, limit int) (*ChatScopedList, error)
	// MissingBankDetails lists scope-visible employees missing any of bank id,
	// account number, or account name. It reports ONLY the employee — never the
	// account values themselves.
	MissingBankDetails(ctx context.Context, scope ChatDataScope, limit int) (*ChatScopedList, error)
}

// ChatEmployeeEntry is one employee row in the chat employee-directory tool.
type ChatEmployeeEntry struct {
	ID           uint   `json:"id"`
	FullName     string `json:"full_name"`
	EmployeeCode string `json:"employee_code"`
}
