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
}
