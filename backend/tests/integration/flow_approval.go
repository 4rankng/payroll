package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const flowApprovals = "Approvals"

// approvalFlowRequest mirrors the approval API response slice used by the flow.
type approvalFlowRequest struct {
	ID           uint           `json:"id"`
	Type         string         `json:"type"`
	Status       string         `json:"status"`
	RetryCount   int            `json:"retry_count"`
	ScheduledAt  *time.Time     `json:"scheduled_at"`
	DecisionNote *string        `json:"decision_note"`
	Outcome      *string        `json:"outcome"`
	Payload      map[string]any `json:"payload"`
	RequesterID  uint           `json:"requester_id"`
}

// voidableTimesheetIDs fetches unpaid timesheet ids for void filings.
func voidableTimesheetIDs(n int) []uint {
	// Excludes timesheets already referenced by an approval request: void
	// idempotency keys are consumed forever (even after cancel), so prior runs
	// permanently shrink an unfiltered pool.
	raw, err := queryPayrollScalar(
		fmt.Sprintf("SELECT GROUP_CONCAT(id ORDER BY id DESC) FROM (SELECT t.id FROM timesheets t WHERE t.payment_status NOT IN ('paid','failed','cancelled') AND t.id NOT IN (SELECT CAST(JSON_EXTRACT(payload,'$.timesheetId') AS UNSIGNED) FROM approval_requests WHERE JSON_EXTRACT(payload,'$.timesheetId') IS NOT NULL) ORDER BY t.id DESC LIMIT %d) x", n))
	if err != nil || raw == "" {
		return nil
	}
	var out []uint
	for _, part := range splitCommaRaw(raw) {
		if id, err := strconv.ParseUint(part, 10, 64); err == nil {
			out = append(out, uint(id))
		}
	}
	return out
}

func splitCommaRaw(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

// fileApprovalFirstAvailable tries each candidate timesheet until one files
// (skipping ids whose (type, cycle) idempotency key is already consumed by
// earlier runs — executed/cancelled rows keep their key forever — and ids
// that no longer resolve on a shared dev DB, where prior suite runs may have
// removed or re-homed fixtures between the pool snapshot and the file call).
func fileApprovalFirstAvailable(client *APIClient, ids []uint, from *int, reason string) (*approvalFlowRequest, error) {
	for *from < len(ids) {
		id := ids[*from]
		*from++
		req, err := fileApproval(client, id, reason)
		if err == nil {
			return req, nil
		}
		msg := err.Error()
		if !strings.Contains(msg, "409") && !strings.Contains(msg, "không tìm thấy bảng công") {
			return nil, err
		}
	}
	return nil, fmt.Errorf("no unclaimed voidable timesheet left")
}

func fileApproval(client *APIClient, tsID uint, reason string) (*approvalFlowRequest, error) {
	var created approvalFlowRequest
	_, err := client.PostInto("/api/v1/approvals", map[string]any{
		"type":   "void_timesheet",
		"params": map[string]any{"timesheetId": tsID, "reason": reason},
	}, &created)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func runApprovalsTests(client *APIClient, data *TestData, reporter *Reporter, _ *TestConfig) {
	reporter.PrintSection("FLOW: Approvals")

	if len(data.Partners) == 0 {
		reporter.Skip(flowApprovals, "All tests", "no partner account available")
		return
	}
	partner := client.WithToken(data.Partners[0].Token)
	// Larger pool: shared dev DBs accumulate consumed void keys across runs,
	// and candidates that no longer resolve are skipped at file time.
	tsIDs := voidableTimesheetIDs(32)
	if len(tsIDs) < 3 {
		reporter.Skip(flowApprovals, "All tests", "not enough voidable timesheets in dev data")
		return
	}

	var approveTarget, retryTarget *approvalFlowRequest
	var claimedTSID any
	idx := 0
	defer func() {
		// Terminal cleanup: executed/cancelled/escalated rows are closed history;
		// cancel anything still open so dev data stays tidy.
		for _, r := range []*approvalFlowRequest{approveTarget, retryTarget} {
			if r != nil && (r.Status == "pending" || r.Status == "rescheduled") {
				_, _, _ = client.Post(fmt.Sprintf("/api/v1/approvals/%d/cancel", r.ID), map[string]string{"note": "dọn dẹp sau kiểm thử"})
			}
		}
	}()

	reporter.RunTest(flowApprovals, "Partner files void_timesheet within allowlist", func() error {
		created, err := fileApprovalFirstAvailable(partner, tsIDs, &idx, "[ITEST] nhầm ngày công")
		if err != nil {
			return fmt.Errorf("file: %w", err)
		}
		approveTarget = created
		if err := AssertEqual("status", "pending", created.Status); err != nil {
			return err
		}
		if created.Payload["timesheetId"] == nil {
			return fmt.Errorf("payload missing server-derived timesheet snapshot")
		}
		claimedTSID = created.Payload["timesheetId"]
		return AssertEqual("requester", data.Partners[0].ID, created.RequesterID)
	})

	reporter.RunTest(flowApprovals, "Partner cannot file send_sao_ke_email (allowlist)", func() error {
		_, status, err := partner.PostExpectError("/api/v1/approvals", map[string]any{
			"type":   "send_sao_ke_email",
			"params": map[string]any{"reportAtDate": "2026-10-05"},
		})
		if err != nil {
			return fmt.Errorf("transport error: %v", err)
		}
		return AssertEqual("status", 403, status)
	})

	reporter.RunTest(flowApprovals, "Partner cannot approve (Casbin deny + service wall)", func() error {
		_, status, err := partner.PostExpectError(fmt.Sprintf("/api/v1/approvals/%d/approve", approveTarget.ID), map[string]string{})
		if err != nil {
			return fmt.Errorf("transport error: %v", err)
		}
		return AssertEqual("status", 403, status)
	})

	reporter.RunTest(flowApprovals, "Admin approves and outcome is recorded", func() error {
		var decided approvalFlowRequest
		if _, err := client.PostInto(fmt.Sprintf("/api/v1/approvals/%d/approve", approveTarget.ID), map[string]string{}, &decided); err != nil {
			return fmt.Errorf("approve: %w", err)
		}
		if err := AssertEqual("status", "executed", decided.Status); err != nil {
			return err
		}
		return AssertNotNil("outcome", decided.Outcome)
	})

	reporter.RunTest(flowApprovals, "Duplicate (type, cycle) file is rejected 409", func() error {
		_, status, err := client.PostExpectError("/api/v1/approvals", map[string]any{
			"type":   "void_timesheet",
			"params": map[string]any{"timesheetId": claimedTSID, "reason": "[ITEST] trùng"},
		})
		if err != nil {
			return fmt.Errorf("transport error: %v", err)
		}
		return AssertEqual("status", 409, status)
	})

	reporter.RunTest(flowApprovals, "Reject-with-note reschedules at 08:00 next day", func() error {
		created, err := fileApprovalFirstAvailable(client, tsIDs, &idx, "[ITEST] từ chối thử")
		if err != nil {
			return fmt.Errorf("file: %w", err)
		}
		retryTarget = created
		var rejected approvalFlowRequest
		if _, err := client.PostInto(fmt.Sprintf("/api/v1/approvals/%d/reject", created.ID), map[string]string{"note": "gửi mai, dữ liệu chưa đủ"}, &rejected); err != nil {
			return fmt.Errorf("reject: %w", err)
		}
		if err := AssertEqual("status", "rescheduled", rejected.Status); err != nil {
			return err
		}
		if err := AssertEqual("retry_count", 1, rejected.RetryCount); err != nil {
			return err
		}
		if rejected.ScheduledAt == nil {
			return fmt.Errorf("scheduled_at missing")
		}
		if rejected.ScheduledAt.Hour() != 8 {
			return fmt.Errorf("scheduled hour = %d, want 8 (08:00 ICT)", rejected.ScheduledAt.Hour())
		}
		if rejected.DecisionNote == nil || *rejected.DecisionNote != "gửi mai, dữ liệu chưa đủ" {
			return fmt.Errorf("decision note not preserved")
		}
		return nil
	})

	reporter.RunTest(flowApprovals, "Retry budget exhaustion escalates", func() error {
		var cur approvalFlowRequest
		for i := 2; i <= 3; i++ {
			if _, err := client.PostInto(fmt.Sprintf("/api/v1/approvals/%d/reject", retryTarget.ID), map[string]string{"note": fmt.Sprintf("note %d", i)}, &cur); err != nil {
				return fmt.Errorf("reject %d: %w", i, err)
			}
		}
		if err := AssertEqual("retry_count after 3 rejects", 3, cur.RetryCount); err != nil {
			return err
		}
		if _, err := client.PostInto(fmt.Sprintf("/api/v1/approvals/%d/reject", retryTarget.ID), map[string]string{"note": "lần cuối"}, &cur); err != nil {
			return fmt.Errorf("final reject: %w", err)
		}
		retryTarget.Status = cur.Status
		return AssertEqual("status", "escalated", cur.Status)
	})

	reporter.RunTest(flowApprovals, "Partner list shows only own requests", func() error {
		var items []approvalFlowRequest
		if _, err := partner.GetInto("/api/v1/approvals?page=1&pageSize=50", &items); err != nil {
			return fmt.Errorf("list: %w", err)
		}
		for _, r := range items {
			if r.RequesterID != data.Partners[0].ID {
				return fmt.Errorf("partner saw foreign request %d (requester %d)", r.ID, r.RequesterID)
			}
		}
		return nil
	})

	reporter.RunTest(flowApprovals, "Cancel is terminal", func() error {
		created, err := fileApprovalFirstAvailable(client, tsIDs, &idx, "[ITEST] huỷ")
		if err != nil {
			return fmt.Errorf("file: %w", err)
		}
		var cancelled approvalFlowRequest
		if _, err := client.PostInto(fmt.Sprintf("/api/v1/approvals/%d/cancel", created.ID), map[string]string{"note": "huỷ, không gửi lại"}, &cancelled); err != nil {
			return fmt.Errorf("cancel: %w", err)
		}
		if err := AssertEqual("status", "cancelled", cancelled.Status); err != nil {
			return err
		}
		_, status, err := client.PostExpectError(fmt.Sprintf("/api/v1/approvals/%d/approve", created.ID), map[string]string{})
		if err != nil {
			return fmt.Errorf("transport error: %v", err)
		}
		return AssertEqual("post-cancel approve status", 400, status)
	})
}
