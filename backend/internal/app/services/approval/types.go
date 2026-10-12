package approval

import (
	"encoding/json"

	"api-server/internal/domain"
)

// --- Input parameters (the ONLY thing a filer may supply) --------------------
//
// Every field below identifies WHAT should happen and for which cycle/object.
// Recipients, message content, and money figures are never accepted from the
// client — they are derived server-side by the payload builders.

// SaoKeEmailParams identifies the statement cycle for a sao kê email.
type SaoKeEmailParams struct {
	ReportAtDate string `json:"reportAtDate"`
}

// PayrollEmailParams identifies the statement cycle for a payroll email.
type PayrollEmailParams struct {
	ReportAtDate string `json:"reportAtDate"`
}

// ReminderParams identifies a project + date for a reminder.
type ReminderParams struct {
	ProjectID     uint   `json:"projectId"`
	RemindForDate string `json:"remindForDate"`
}

// VoidTimesheetParams identifies the timesheet to void plus the filer's reason
// (free text — it is guidance for the approver, not message content).
type VoidTimesheetParams struct {
	TimesheetID uint   `json:"timesheetId"`
	Reason      string `json:"reason"`
}

// --- Server-derived payloads (stored on the request row) --------------------

// GeneralRequestParams carries the requester's free-text ask (chat "other").
type GeneralRequestParams struct {
	Summary string `json:"summary"`
}

// SaoKeEmailPayload is the server-derived payload for send_sao_ke_email.
type SaoKeEmailPayload struct {
	ReportAtDate   string   `json:"reportAtDate"`
	PeriodFrom     string   `json:"periodFrom"`
	PeriodTo       string   `json:"periodTo"`
	ProjectIDs     []uint   `json:"projectIds"`
	TimesheetCount int      `json:"timesheetCount"`
	TotalAmount    int64    `json:"totalAmount"`
	Recipients     []string `json:"recipients"`
	Cc             []string `json:"cc"`
}

// PayrollEmailPayload is the server-derived payload for send_payroll_email.
type PayrollEmailPayload struct {
	ReportAtDate   string   `json:"reportAtDate"`
	PeriodFrom     string   `json:"periodFrom"`
	PeriodTo       string   `json:"periodTo"`
	ProjectIDs     []uint   `json:"projectIds"`
	TimesheetCount int      `json:"timesheetCount"`
	TotalAmount    int64    `json:"totalAmount"`
	Recipients     []string `json:"recipients"`
	Cc             []string `json:"cc"`
}

// ReminderPayload is the server-derived payload for send_reminder.
type ReminderPayload struct {
	ProjectID     uint     `json:"projectId"`
	RemindForDate string   `json:"remindForDate"`
	Recipients    []string `json:"recipients"`
}

// VoidTimesheetPayload is the server-derived payload for void_timesheet. It
// snapshots the timesheet facts at filing time so the approver sees what will
// be voided even if the row changes later.
type VoidTimesheetPayload struct {
	TimesheetID uint `json:"timesheetId"` // machine field: never displayed
	// Human identification — what the admin actually reviews.
	EmployeeName   string  `json:"employeeName"`
	ProjectName    string  `json:"projectName"`
	Position       string  `json:"position"`
	DayType        string  `json:"dayType"`
	Hours          float64 `json:"hours"`
	RateVND        int64   `json:"rateVnd"`
	AmountVND      int64   `json:"amountVnd"`
	Date           string  `json:"date"`
	StatusAtFiling string  `json:"statusAtFiling"`
	Reason         string  `json:"reason"`
}

// BuiltPayload is what a payload builder produces: the stored payload, the
// canonical idempotency seed for the (type, cycle) key, and a display title.
type BuiltPayload struct {
	Payload  any
	CycleKey string
	Title    string
}

// CreateParams is the service-level input for filing a request.
type CreateParams struct {
	Type      domain.ApprovalRequestType
	Params    json.RawMessage
	ActorID   uint
	ActorRole string // re-checked against the per-type filer allowlist
	ActorName string
}
