package domain

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestApprovalRequest(status ApprovalRequestStatus) *ApprovalRequest {
	note := "thử lại với số liệu đã chốt"
	return &ApprovalRequest{
		ID:             7,
		Type:           ApprovalTypeSendSaoKeEmail,
		Status:         status,
		Payload:        `{"reportAtDate":"2026-11-01"}`,
		IdempotencyKey: "approval:send_sao_ke_email:abc",
		RequesterID:    1,
		DecisionNote:   &note,
		RetryCount:     1,
	}
}

// TestApprovalRequestGuards pins the guarded-transition matrix: every action
// must reject with a domain error outside its allowed source states and never
// mutate the entity on failure.
func TestApprovalRequestGuards(t *testing.T) {
	type action struct {
		name      string
		allowed   []ApprovalRequestStatus
		apply     func(*ApprovalRequest) error
		wantState ApprovalRequestStatus // state after a successful transition
	}

	all := ValidApprovalRequestStatuses
	actions := []action{
		{
			name:    "approve",
			allowed: []ApprovalRequestStatus{ApprovalStatusPending, ApprovalStatusRescheduled},
			apply:   func(r *ApprovalRequest) error { return r.Approve(9) },
			// approve leaves the row at 'approved'; the execute handoff follows.
			wantState: ApprovalStatusApproved,
		},
		{
			name:    "reject",
			allowed: []ApprovalRequestStatus{ApprovalStatusPending, ApprovalStatusRescheduled},
			apply:   func(r *ApprovalRequest) error { return r.Reject(9, "gợi ý") },
			// reject is the decision step; scheduling the retry follows.
			wantState: ApprovalStatusRejected,
		},
		{
			name:    "schedule-retry",
			allowed: []ApprovalRequestStatus{ApprovalStatusRejected, ApprovalStatusRescheduled},
			apply: func(r *ApprovalRequest) error {
				return r.ScheduleRetry(2, time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC))
			},
			wantState: ApprovalStatusRescheduled,
		},
		{
			name:      "requeue-for-review",
			allowed:   []ApprovalRequestStatus{ApprovalStatusRescheduled},
			apply:     func(r *ApprovalRequest) error { return r.RequeueForReview() },
			wantState: ApprovalStatusPending,
		},
		{
			name:      "escalate",
			allowed:   []ApprovalRequestStatus{ApprovalStatusRejected, ApprovalStatusRescheduled},
			apply:     func(r *ApprovalRequest) error { return r.Escalate() },
			wantState: ApprovalStatusEscalated,
		},
		{
			name:      "execute",
			allowed:   []ApprovalRequestStatus{ApprovalStatusApproved},
			apply:     func(r *ApprovalRequest) error { return r.Execute("ok", time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)) },
			wantState: ApprovalStatusExecuted,
		},
		{
			name:      "cancel",
			allowed:   []ApprovalRequestStatus{ApprovalStatusPending, ApprovalStatusRejected, ApprovalStatusRescheduled, ApprovalStatusEscalated},
			apply:     func(r *ApprovalRequest) error { return r.Cancel(9, "huỷ") },
			wantState: ApprovalStatusCancelled,
		},
		{
			name:      "expire",
			allowed:   []ApprovalRequestStatus{ApprovalStatusPending, ApprovalStatusRejected, ApprovalStatusRescheduled},
			apply:     func(r *ApprovalRequest) error { return r.Expire() },
			wantState: ApprovalStatusExpired,
		},
	}

	for _, act := range actions {
		allowedSet := map[ApprovalRequestStatus]bool{}
		for _, s := range act.allowed {
			allowedSet[s] = true
		}
		for _, from := range all {
			t.Run(act.name+"_from_"+string(from), func(t *testing.T) {
				req := newTestApprovalRequest(from)
				err := act.apply(req)
				if allowedSet[from] {
					if err != nil {
						t.Fatalf("transition %s from %s must succeed, got error: %v", act.name, from, err)
					}
					if req.Status != act.wantState {
						t.Fatalf("after %s from %s status = %s, want %s", act.name, from, req.Status, act.wantState)
					}
					return
				}
				if err == nil {
					t.Fatalf("transition %s from %s must be rejected", act.name, from)
				}
				var domainErr *DomainError
				if !errors.As(err, &domainErr) {
					t.Fatalf("transition %s from %s must return a domain error, got %T", act.name, from, err)
				}
				if domainErr.Type != "CONFLICT" && domainErr.Type != "VALIDATION_ERROR" {
					t.Fatalf("transition %s from %s returned error type %s", act.name, from, domainErr.Type)
				}
				if req.Status != from {
					t.Fatalf("failed transition %s from %s mutated status to %s", act.name, from, req.Status)
				}
			})
		}
	}
}

// TestApprovalRequestRejectVsCancelSemantics pins the red-team distinction:
// reject means "Thử lại ngày mai" (note captured, request continues), cancel
// means "Huỷ / không gửi lại" (terminal, scheduled retry wiped).
func TestApprovalRequestRejectVsCancelSemantics(t *testing.T) {
	req := newTestApprovalRequest(ApprovalStatusPending)
	if err := req.Reject(3, "  dữ liệu hôm nay chưa đủ  "); err != nil {
		t.Fatalf("reject must succeed: %v", err)
	}
	if req.DecisionNote == nil || *req.DecisionNote != "dữ liệu hôm nay chưa đủ" {
		t.Fatalf("reject must capture the note (trimmed), got %v", req.DecisionNote)
	}
	if req.Status != ApprovalStatusRejected {
		t.Fatalf("reject status = %s", req.Status)
	}
	scheduled := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)
	if err := req.ScheduleRetry(1, scheduled); err != nil {
		t.Fatalf("schedule retry must succeed: %v", err)
	}
	if req.ScheduledAt == nil || !req.ScheduledAt.Equal(scheduled) {
		t.Fatalf("scheduled_at = %v, want %v", req.ScheduledAt, scheduled)
	}

	if err := req.Cancel(3, "không gửi lại"); err != nil {
		t.Fatalf("cancel must succeed: %v", err)
	}
	if req.ScheduledAt != nil {
		t.Fatalf("cancel must clear scheduled_at, got %v", req.ScheduledAt)
	}
	if !req.IsTerminal() {
		t.Fatal("cancelled must be terminal")
	}
	if err := req.RequeueForReview(); err == nil {
		t.Fatal("a cancelled request must never reschedule")
	}
	if err := req.ScheduleRetry(2, scheduled); err == nil {
		t.Fatal("a cancelled request must never schedule another retry")
	}
}

// TestApprovalRequestRetryBudget pins "3 rejects → escalated": each rejection
// consumes one attempt; the third escalates instead of rescheduling.
func TestApprovalRequestRetryBudget(t *testing.T) {
	req := newTestApprovalRequest(ApprovalStatusPending)
	nextDay := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)

	for attempt := 1; attempt <= MaxApprovalRetryAttempts; attempt++ {
		if err := req.Reject(2, "thử lại"); err != nil {
			t.Fatalf("reject %d must succeed: %v", attempt, err)
		}
		if attempt < MaxApprovalRetryAttempts {
			if err := req.ScheduleRetry(attempt, nextDay); err != nil {
				t.Fatalf("attempt %d must schedule: %v", attempt, err)
			}
			if req.Status != ApprovalStatusRescheduled || req.RetryCount != attempt {
				t.Fatalf("attempt %d: status=%s retry_count=%d", attempt, req.Status, req.RetryCount)
			}
			// The scheduled retry fires and re-files the request.
			if err := req.RequeueForReview(); err != nil {
				t.Fatalf("attempt %d must requeue: %v", attempt, err)
			}
			continue
		}
		// Third attempt: escalate instead of rescheduling.
		if err := req.Escalate(); err != nil {
			t.Fatalf("third rejection must escalate: %v", err)
		}
		if req.Status != ApprovalStatusEscalated {
			t.Fatalf("third rejection status = %s, want escalated", req.Status)
		}
	}
}

func TestBuildApprovalIdempotencyKey(t *testing.T) {
	key := BuildApprovalIdempotencyKey(ApprovalTypeSendSaoKeEmail, "report_at_date=2026-11-01")
	again := BuildApprovalIdempotencyKey(ApprovalTypeSendSaoKeEmail, "  report_at_date=2026-11-01  ")
	if key != again {
		t.Fatalf("key must be canonical: %q vs %q", key, again)
	}
	if !strings.HasPrefix(key, "approval:send_sao_ke_email:") {
		t.Fatalf("key %q must carry its type prefix", key)
	}
	otherType := BuildApprovalIdempotencyKey(ApprovalTypeSendPayrollEmail, "report_at_date=2026-11-01")
	if key == otherType {
		t.Fatal("different types for the same cycle must not collide")
	}
	otherCycle := BuildApprovalIdempotencyKey(ApprovalTypeSendSaoKeEmail, "report_at_date=2026-12-01")
	if key == otherCycle {
		t.Fatal("different cycles must not collide")
	}
}

// TestApprovalRequestFactoryMessages pins that every transition factory emits a
// complete Vietnamese audit sentence (the audit handler drops empty messages).
func TestApprovalRequestFactoryMessages(t *testing.T) {
	ctx := context.Background()
	req := newTestApprovalRequest(ApprovalStatusRescheduled)
	at := time.Date(2026, 11, 2, 8, 0, 0, 0, time.UTC)

	events := []ApprovalRequestChangedEvent{
		NewApprovalRequestCreatedEvent(ctx, req, 1, "Quản trị"),
		NewApprovalRequestApprovedEvent(ctx, req, 1, "Quản trị"),
		NewApprovalRequestRejectedEvent(ctx, req, 1, "Quản trị", "thử lại với số liệu chốt"),
		NewApprovalRequestRescheduledEvent(ctx, req, 49, "Hệ thống", 1, at),
		NewApprovalRequestRetryFiredEvent(ctx, req, 49, "Hệ thống"),
		NewApprovalRequestEscalatedEvent(ctx, req, 49, "Hệ thống"),
		NewApprovalRequestExecutedEvent(ctx, req, 1, "Quản trị"),
		NewApprovalRequestCancelledEvent(ctx, req, 1, "Quản trị", "huỷ"),
		NewApprovalRequestExpiredEvent(ctx, req, 49, "Hệ thống"),
	}
	for _, ev := range events {
		if ev.GetAuditMessage() == "" {
			t.Fatalf("%s produced an empty audit message", ev.EventType())
		}
		if strings.Contains(ev.GetAuditMessage(), "%!") {
			t.Fatalf("%s audit message has formatting residue: %q", ev.EventType(), ev.GetAuditMessage())
		}
		if ev.GetEntityType() != EntityTypeApprovalRequest {
			t.Fatalf("%s entity type = %s", ev.EventType(), ev.GetEntityType())
		}
	}
}
