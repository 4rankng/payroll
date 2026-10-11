package domain

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/pkg/clock"
)

// ApprovalRequestChangedEvent is published for EVERY lifecycle transition of an
// approval request. One event shape keeps the audit pipeline uniform while the
// EventName (see the factories below) still distinguishes each transition for
// event-specific consumers.
//
// Audit messages are built here with fmt.Sprintf instead of via AuditMatrix:
// the matrix lives in audit_templates.go and this engine's verbs have no
// templates there yet. The sentences follow the matrix convention (actor
// first, complete Vietnamese sentence, no placeholders).
type ApprovalRequestChangedEvent struct {
	BaseEvent
	RequestType ApprovalRequestType
	FromStatus  ApprovalRequestStatus
	ToStatus    ApprovalRequestStatus
	Note        string
}

// ApprovalTypeLabelVN returns the Vietnamese label of a request type for
// audit/notification copy.
func ApprovalTypeLabelVN(t ApprovalRequestType) string {
	switch t {
	case ApprovalTypeSendSaoKeEmail:
		return "gửi email sao kê"
	case ApprovalTypeSendPayrollEmail:
		return "gửi email bảng lương"
	case ApprovalTypeSendReminder:
		return "gửi nhắc nhở"
	case ApprovalTypeVoidTimesheet:
		return "đề xuất huỷ bảng công"
	}
	return string(t)
}

// truncateNote bounds free-text decision notes embedded into audit messages.
func truncateNote(note string, max int) string {
	trimmed := strings.TrimSpace(note)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) <= max {
		return trimmed
	}
	return string(runes[:max]) + "…"
}

func approvalEvent(ctx context.Context, eventName string, req *ApprovalRequest,
	actorUserID uint, auditMessage string, action AuditAction, note string) ApprovalRequestChangedEvent {
	return ApprovalRequestChangedEvent{
		BaseEvent: newBaseEventWithActor(ctx, eventName, req.ID, actorUserID,
			action, EntityTypeApprovalRequest, auditMessage),
		RequestType: req.Type,
		Note:        note,
	}
}

// NewApprovalRequestCreatedEvent — a request was filed.
func NewApprovalRequestCreatedEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã tạo yêu cầu phê duyệt %s (mã %d)", actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestCreated", req, actorUserID, msg, AuditActionCreate, "")
	ev.FromStatus = ""
	ev.ToStatus = ApprovalStatusPending
	return ev
}

// NewApprovalRequestApprovedEvent — the approval decision was recorded.
func NewApprovalRequestApprovedEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã duyệt yêu cầu phê duyệt %s (mã %d)", actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestApproved", req, actorUserID, msg, AuditActionApprove, "")
	ev.FromStatus = ApprovalStatusPending
	ev.ToStatus = ApprovalStatusApproved
	return ev
}

// NewApprovalRequestRejectedEvent — "Thử lại ngày mai" with drafting guidance.
func NewApprovalRequestRejectedEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName, note string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã từ chối yêu cầu phê duyệt %s (mã %d), thử lại ngày mai", actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	if short := truncateNote(note, 200); short != "" {
		msg = fmt.Sprintf("%s — ghi chú: %s", msg, short)
	}
	ev := approvalEvent(ctx, "ApprovalRequestRejected", req, actorUserID, msg, AuditActionReject, note)
	ev.FromStatus = ApprovalStatusPending
	ev.ToStatus = ApprovalStatusRejected
	return ev
}

// NewApprovalRequestRescheduledEvent — a retry attempt was scheduled
// (automatic; actor is the system user when driven by the retry worker).
func NewApprovalRequestRescheduledEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string, attempt int, scheduledAt time.Time) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã lên lịch thử lại yêu cầu phê duyệt %s (mã %d) lần %d, hẹn %s",
		actorName, ApprovalTypeLabelVN(req.Type), req.ID, attempt,
		scheduledAt.In(clock.DefaultLocation).Format("02/01/2006 15:04"))
	ev := approvalEvent(ctx, "ApprovalRequestRescheduled", req, actorUserID, msg, AuditActionReschedule, "")
	ev.FromStatus = ApprovalStatusRejected
	ev.ToStatus = ApprovalStatusRescheduled
	return ev
}

// NewApprovalRequestRetryFiredEvent — a scheduled retry put the request back
// in front of reviewers (rescheduled → pending). Unattended re-file only.
func NewApprovalRequestRetryFiredEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("Yêu cầu phê duyệt %s (mã %d) đã được đưa trở lại hàng chờ duyệt sau lần thử lại",
		ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestRetryFired", req, actorUserID, msg, AuditActionUpdate, "")
	ev.FromStatus = ApprovalStatusRescheduled
	ev.ToStatus = ApprovalStatusPending
	return ev
}

// NewApprovalRequestEscalatedEvent — the retry budget is spent.
func NewApprovalRequestEscalatedEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã đưa yêu cầu phê duyệt %s (mã %d) vào danh sách cần xem xét (hết số lần thử lại)",
		actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestEscalated", req, actorUserID, msg, AuditActionEscalate, "")
	ev.FromStatus = ApprovalStatusRescheduled
	ev.ToStatus = ApprovalStatusEscalated
	return ev
}

// NewApprovalRequestExecutedEvent — the execution handoff finished.
func NewApprovalRequestExecutedEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã thực thi yêu cầu phê duyệt %s (mã %d)", actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestExecuted", req, actorUserID, msg, AuditActionExecute, "")
	ev.FromStatus = ApprovalStatusApproved
	ev.ToStatus = ApprovalStatusExecuted
	return ev
}

// NewApprovalRequestCancelledEvent — terminal "Huỷ / không gửi lại".
func NewApprovalRequestCancelledEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName, note string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("%s đã huỷ yêu cầu phê duyệt %s (mã %d), không gửi lại", actorName, ApprovalTypeLabelVN(req.Type), req.ID)
	if short := truncateNote(note, 200); short != "" {
		msg = fmt.Sprintf("%s — ghi chú: %s", msg, short)
	}
	ev := approvalEvent(ctx, "ApprovalRequestCancelled", req, actorUserID, msg, AuditActionCancel, note)
	ev.FromStatus = ApprovalStatusPending
	ev.ToStatus = ApprovalStatusCancelled
	return ev
}

// NewApprovalRequestExpiredEvent — closed without a decision.
func NewApprovalRequestExpiredEvent(ctx context.Context, req *ApprovalRequest, actorUserID uint, actorName string) ApprovalRequestChangedEvent {
	msg := fmt.Sprintf("Yêu cầu phê duyệt %s (mã %d) đã hết hạn mà chưa được xử lý", ApprovalTypeLabelVN(req.Type), req.ID)
	ev := approvalEvent(ctx, "ApprovalRequestExpired", req, actorUserID, msg, AuditActionExpire, "")
	ev.ToStatus = ApprovalStatusExpired
	return ev
}
