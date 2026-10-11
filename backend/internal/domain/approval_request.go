package domain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"time"
)

// ApprovalRequestType enumerates the request types the approval engine knows
// how to build payloads and deterministic preconditions for. The registry in
// `app/services/approval` is open to extension: a new type needs a payload
// builder, a precondition check, and (optionally) an execution handler.
type ApprovalRequestType string

const (
	// ApprovalTypeSendSaoKeEmail sends the payroll bank-statement ("sao kê")
	// email for one statement cycle through the existing email worker.
	ApprovalTypeSendSaoKeEmail ApprovalRequestType = "send_sao_ke_email"
	// ApprovalTypeSendPayrollEmail sends the payroll statement email for one
	// statement cycle.
	ApprovalTypeSendPayrollEmail ApprovalRequestType = "send_payroll_email"
	// ApprovalTypeSendReminder sends a reminder to the partner users attached
	// to a project for a given date.
	ApprovalTypeSendReminder ApprovalRequestType = "send_reminder"
	// ApprovalTypeVoidTimesheet requests voiding a timesheet. The actual void
	// ships in a later phase; approving records the decision only.
	ApprovalTypeVoidTimesheet ApprovalRequestType = "void_timesheet"
	// ApprovalTypeGeneralRequest is the chat assistant's catch-all: permission
	// grants, config asks, anything needing an admin decision without a
	// dedicated flow. Approving records the decision; execution stays manual.
	ApprovalTypeGeneralRequest ApprovalRequestType = "general_request"
)

// ValidApprovalRequestType reports whether t is a known request type.
func ValidApprovalRequestType(t ApprovalRequestType) bool {
	switch t {
	case ApprovalTypeSendSaoKeEmail, ApprovalTypeSendPayrollEmail,
		ApprovalTypeSendReminder, ApprovalTypeVoidTimesheet, ApprovalTypeGeneralRequest:
		return true
	}
	return false
}

// EntityTypeApprovalRequest identifies approval requests in audit_logs.
const EntityTypeApprovalRequest EntityType = "approval_request"

// Audit actions used by approval-request transitions. The generic actions
// (CREATE/APPROVE/REJECT/…) cover the standard decisions; the engine's own
// lifecycle steps get their own verbs so audit filters can tell "the admin
// cancelled it" from "the scheduler expired it".
const (
	AuditActionCancel     AuditAction = "CANCEL"
	AuditActionExecute    AuditAction = "EXECUTE"
	AuditActionEscalate   AuditAction = "ESCALATE"
	AuditActionReschedule AuditAction = "RESCHEDULE"
	AuditActionExpire     AuditAction = "EXPIRE"
)

// ApprovalRequestStatus is the lifecycle state of an approval request.
//
//	pending     — filed, waiting for a decision
//	approved    — decision recorded, execution handoff in flight
//	executed    — execution finished with a recorded outcome
//	rejected    — decision recorded as "Thử lại ngày mai" (reject with note)
//	rescheduled — a retry is scheduled (retry_count consumed, scheduled_at set)
//	escalated   — retry budget exhausted; a human must decide
//	expired     — terminal, never acted on in time
//	cancelled   — terminal "Huỷ / không gửi lại" (no retry, no escalation)
type ApprovalRequestStatus string

const (
	ApprovalStatusPending     ApprovalRequestStatus = "pending"
	ApprovalStatusApproved    ApprovalRequestStatus = "approved"
	ApprovalStatusExecuted    ApprovalRequestStatus = "executed"
	ApprovalStatusRejected    ApprovalRequestStatus = "rejected"
	ApprovalStatusRescheduled ApprovalRequestStatus = "rescheduled"
	ApprovalStatusEscalated   ApprovalRequestStatus = "escalated"
	ApprovalStatusExpired     ApprovalRequestStatus = "expired"
	ApprovalStatusCancelled   ApprovalRequestStatus = "cancelled"
)

// MaxApprovalRetryAttempts is the retry budget per request. Every rejection or
// failed-precondition re-run consumes one attempt; the attempt that would
// consume the third unit escalates the request instead of rescheduling it
// (so "3 rejects → escalated", and at most two automatic 08:00 re-runs).
const MaxApprovalRetryAttempts = 3

// User-facing status labels (Vietnamese) for notification copy.
var approvalStatusLabels = map[ApprovalRequestStatus]string{
	ApprovalStatusPending:     "chờ duyệt",
	ApprovalStatusApproved:    "đã duyệt",
	ApprovalStatusExecuted:    "đã thực thi",
	ApprovalStatusRejected:    "bị từ chối",
	ApprovalStatusRescheduled: "đã lên lịch thử lại",
	ApprovalStatusEscalated:   "cần xem xét (hết số lần thử lại)",
	ApprovalStatusExpired:     "đã hết hạn",
	ApprovalStatusCancelled:   "đã huỷ",
}

// LabelVN returns the Vietnamese display label for a status.
func (s ApprovalRequestStatus) LabelVN() string {
	if label, ok := approvalStatusLabels[s]; ok {
		return label
	}
	return string(s)
}

// ApprovalRequest is the durable approval-request aggregate. All payload
// content is SERVER-DERIVED at creation time (see app/services/approval
// payload builders): a filer only chooses the request type and supplies
// identifying parameters (cycle date, timesheet id, …) — never recipients or
// message content.
type ApprovalRequest struct {
	ID             uint                  `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Type           ApprovalRequestType   `json:"type" gorm:"column:request_type;type:varchar(64);not null;index:idx_approval_requests_request_type;comment:'Request type'"`
	Status         ApprovalRequestStatus `json:"status" gorm:"type:enum('pending','approved','executed','rejected','rescheduled','escalated','expired','cancelled');not null;default:'pending';index:idx_approval_requests_status;comment:'Guarded lifecycle state'"`
	Payload        string                `json:"payload" gorm:"type:json;not null;comment:'Server-derived request payload (JSON)'"`
	IdempotencyKey string                `json:"idempotency_key" gorm:"type:varchar(191);not null;uniqueIndex:idx_approval_requests_idempotency_key;comment:'Unique per (type, cycle)'"`
	RequesterID    uint                  `json:"requester_id" gorm:"not null;type:bigint unsigned;index:idx_approval_requests_requester_id;comment:'User who filed the request'"`
	ApproverID     *uint                 `json:"approver_id" gorm:"type:bigint unsigned;comment:'User who made the latest decision (approve or reject)'"`
	DecisionNote   *string               `json:"decision_note" gorm:"type:text;comment:'Reject guidance / decision comment'"`
	Outcome        *string               `json:"outcome" gorm:"type:text;comment:'Recorded execution outcome'"`
	RetryCount     int                   `json:"retry_count" gorm:"not null;default:0;comment:'Consumed retry attempts'"`
	ScheduledAt    *time.Time            `json:"scheduled_at" gorm:"type:datetime(3);index:idx_approval_requests_scheduled_at;comment:'Next automatic retry time'"`
	ExecutedAt     *time.Time            `json:"executed_at" gorm:"type:datetime(3)"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

// ValidStatuses lists every legal ApprovalRequestStatus (used by validation
// and the SQL enum to stay in sync).
var ValidApprovalRequestStatuses = []ApprovalRequestStatus{
	ApprovalStatusPending,
	ApprovalStatusApproved,
	ApprovalStatusExecuted,
	ApprovalStatusRejected,
	ApprovalStatusRescheduled,
	ApprovalStatusEscalated,
	ApprovalStatusExpired,
	ApprovalStatusCancelled,
}

// --- Lifecycle guards -------------------------------------------------------

// CanBeApproved reports whether a decision-maker may approve now. A
// rescheduled request may be decided on directly — the scheduled retry is a
// convenience, not a lock.
func (r *ApprovalRequest) CanBeApproved() bool {
	return r.Status == ApprovalStatusPending || r.Status == ApprovalStatusRescheduled
}

// CanBeRejected reports whether a decision-maker may reject with a note
// ("Thử lại ngày mai"). Allowed while the request is still awaiting review,
// including after prior rejections (the retry budget bounds the loop).
func (r *ApprovalRequest) CanBeRejected() bool {
	return r.Status == ApprovalStatusPending || r.Status == ApprovalStatusRescheduled
}

// CanBeCancelled reports whether the request may be terminally cancelled
// ("Huỷ / không gửi lại"). Executed and expired requests are closed history.
func (r *ApprovalRequest) CanBeCancelled() bool {
	switch r.Status {
	case ApprovalStatusPending, ApprovalStatusRejected, ApprovalStatusRescheduled, ApprovalStatusEscalated:
		return true
	}
	return false
}

// CanBeRescheduled reports whether a next retry attempt may be scheduled:
// right after a rejection, or while waiting on an earlier retry whose
// preconditions still fail.
func (r *ApprovalRequest) CanBeRescheduled() bool {
	return r.Status == ApprovalStatusRejected || r.Status == ApprovalStatusRescheduled
}

// CanBeEscalated reports whether the request may move to escalation.
func (r *ApprovalRequest) CanBeEscalated() bool {
	return r.Status == ApprovalStatusRejected || r.Status == ApprovalStatusRescheduled
}

// CanBeRequeued reports whether a fired retry may put the request back in
// front of reviewers (rescheduled → pending).
func (r *ApprovalRequest) CanBeRequeued() bool {
	return r.Status == ApprovalStatusRescheduled
}

// CanBeExecuted reports whether the execution handoff may run.
func (r *ApprovalRequest) CanBeExecuted() bool {
	return r.Status == ApprovalStatusApproved
}

// CanBeExpired reports whether the request may be expired.
func (r *ApprovalRequest) CanBeExpired() bool {
	switch r.Status {
	case ApprovalStatusPending, ApprovalStatusRejected, ApprovalStatusRescheduled:
		return true
	}
	return false
}

// IsTerminal reports whether no further transition is possible.
func (r *ApprovalRequest) IsTerminal() bool {
	switch r.Status {
	case ApprovalStatusExecuted, ApprovalStatusExpired, ApprovalStatusCancelled:
		return true
	}
	return false
}

// --- Guarded transitions ----------------------------------------------------

func (r *ApprovalRequest) invalidTransition(action string) error {
	return NewConflictError(fmt.Sprintf(
		"không thể %s yêu cầu phê duyệt ở trạng thái '%s'", action, r.Status.LabelVN()))
}

// Approve records the approval decision (pending|rescheduled → approved).
func (r *ApprovalRequest) Approve(approverID uint) error {
	if !r.CanBeApproved() {
		return r.invalidTransition("duyệt")
	}
	r.Status = ApprovalStatusApproved
	r.ApproverID = &approverID
	return nil
}

// Reject records a rejection with guidance (pending|rescheduled → rejected).
// The caller is expected to follow with ScheduleRetry or Escalate — reject
// and cancel are DISTINCT: reject means "Thử lại ngày mai".
func (r *ApprovalRequest) Reject(rejecterID uint, note string) error {
	if !r.CanBeRejected() {
		return r.invalidTransition("từ chối")
	}
	r.Status = ApprovalStatusRejected
	r.ApproverID = &rejecterID
	trimmed := strings.TrimSpace(note)
	r.DecisionNote = &trimmed
	return nil
}

// ScheduleRetry records the next retry attempt (rejected|rescheduled →
// rescheduled). attempt is the consumed attempt number (1-based, ==
// retry_count after the call) and scheduledAt is the 08:00 ICT fire time; the
// queue staggers the actual run across minutes to avoid a thundering herd.
func (r *ApprovalRequest) ScheduleRetry(attempt int, scheduledAt time.Time) error {
	if !r.CanBeRescheduled() {
		return r.invalidTransition("lên lịch thử lại")
	}
	if attempt < 1 {
		return NewValidationError("số lần thử lại không hợp lệ")
	}
	r.Status = ApprovalStatusRescheduled
	r.RetryCount = attempt
	at := scheduledAt
	r.ScheduledAt = &at
	return nil
}

// RequeueForReview puts a fired retry back in front of reviewers
// (rescheduled → pending). Unattended runs may only re-file; they never
// approve or send.
func (r *ApprovalRequest) RequeueForReview() error {
	if !r.CanBeRequeued() {
		return r.invalidTransition("đưa trở lại hàng chờ duyệt")
	}
	r.Status = ApprovalStatusPending
	r.ScheduledAt = nil
	return nil
}

// Escalate marks the request as needing human attention
// (rejected|rescheduled → escalated) once the retry budget is spent.
func (r *ApprovalRequest) Escalate() error {
	if !r.CanBeEscalated() {
		return r.invalidTransition("leo thang")
	}
	r.Status = ApprovalStatusEscalated
	r.ScheduledAt = nil
	return nil
}

// Execute records a finished execution handoff (approved → executed) with the
// outcome text the handler produced.
func (r *ApprovalRequest) Execute(outcome string, at time.Time) error {
	if !r.CanBeExecuted() {
		return r.invalidTransition("thực thi")
	}
	r.Status = ApprovalStatusExecuted
	r.Outcome = &outcome
	executedAt := at
	r.ExecutedAt = &executedAt
	r.ScheduledAt = nil
	return nil
}

// Cancel is the terminal "Huỷ / không gửi lại" decision: no retry, no
// escalation, and the (type, cycle) idempotency key stays consumed.
func (r *ApprovalRequest) Cancel(cancellerID uint, note string) error {
	if !r.CanBeCancelled() {
		return r.invalidTransition("huỷ")
	}
	r.Status = ApprovalStatusCancelled
	r.ApproverID = &cancellerID
	if trimmed := strings.TrimSpace(note); trimmed != "" {
		r.DecisionNote = &trimmed
	}
	r.ScheduledAt = nil
	return nil
}

// Expire closes a request that was never acted on in time.
func (r *ApprovalRequest) Expire() error {
	if !r.CanBeExpired() {
		return r.invalidTransition("đánh dấu hết hạn")
	}
	r.Status = ApprovalStatusExpired
	r.ScheduledAt = nil
	return nil
}

// --- Idempotency ------------------------------------------------------------

// BuildApprovalIdempotencyKey derives the unique key for one (type, cycle).
// The key deliberately excludes the requester and any recipient: re-filing the
// same action for the same cycle must collide (409) instead of doubling work —
// the red-team multiplicity trap is that a filer could otherwise fan a request
// out over many recipients/cycles. cycle is the canonical cycle seed produced
// by the type's payload builder (e.g. "report_at_date=2026-11-01").
func BuildApprovalIdempotencyKey(requestType ApprovalRequestType, cycle string) string {
	canonical := strings.ToLower(strings.TrimSpace(cycle))
	sum := sha256.Sum256([]byte(fmt.Sprintf("approval-request|%s|%s", requestType, canonical)))
	return fmt.Sprintf("approval:%s:%x", requestType, sum[:16])
}

// --- Repository contract ----------------------------------------------------

// ApprovalRequestFilters narrows List/Count queries.
type ApprovalRequestFilters struct {
	Status      []ApprovalRequestStatus
	Type        []ApprovalRequestType
	RequesterID *uint
	Limit       int
	Offset      int
	SortBy      string
	SortOrder   string
}

// ApprovalRequestRepository is the persistence port for approval requests.
type ApprovalRequestRepository interface {
	Create(ctx context.Context, request *ApprovalRequest) error
	GetByID(ctx context.Context, id uint) (*ApprovalRequest, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*ApprovalRequest, error)
	List(ctx context.Context, filters ApprovalRequestFilters) ([]*ApprovalRequest, error)
	Count(ctx context.Context, filters ApprovalRequestFilters) (int64, error)
	// SaveTransition persists a guarded transition with a compare-and-swap on
	// the status column: the write only lands when the row still carries one of
	// expectedStatuses. Returns a CONFLICT domain error otherwise, so two
	// concurrent decisions can never both win.
	SaveTransition(ctx context.Context, request *ApprovalRequest, expectedStatuses []ApprovalRequestStatus) error
}
