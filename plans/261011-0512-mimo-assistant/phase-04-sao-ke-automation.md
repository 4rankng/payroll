---
title: "Phase 4: Sao Kê Automation"
status: todo
---

# Phase 4: Sao Kê Automation

## Overview

End the manual sao kê email ritual: "Chuẩn bị sao kê" produces LLM-drafted emails as an approval request; approve sends idempotently through the existing email worker; reject-with-note drives the reschedule loop (08:00, max 3, escalate).

## Requirements

- [ ] "Chuẩn bị sao kê" action (admin button this phase; chatbot files the same request in phase 5) runs deterministic data-completeness check → generates report (existing `payrollReportSvc.GenerateExcel`) → LLM `mimo-v2.6-pro` drafts the email body → creates `send_sao_ke_email` approval request with draft + deterministic summary (total amount, fee, timesheet count from `PayrollEmailMetadata`)
- [ ] Approval card shows full draft for review before approve
- [ ] Approve → enqueue existing `email:payroll_report` task (`EnqueuePayrollReportEmail` with idempotency key; worker `payroll_report_email_worker.go` already sends once per key) — the "executed" transition of phase 2
- [ ] Reject(note) → retry next day 08:00: re-check preconditions deterministically, re-draft with the note as drafting guidance, new attempt on the same request (`retry_count++`); after 3 → escalation
- [ ] Unattended retry runs can never send — only re-draft + re-file
- [ ] Manual send path (`email_handler.go` `POST /timesheets/payroll/report/send-email`) remains as fallback

## Implementation Steps

1. `internal/app/services/approval/types/sao_ke.go` — payload schema, precondition (cycle statements complete), execution adapter wrapping `EmailService.SendPayrollReportEmail` semantics.
2. Drafting: `internal/app/services/llm/drafting.go` — pro model call; prompt templates Vietnamese; input = deterministic summary + recipient list; output = subject + body (markdown/plain).
3. Prepare handler + route (admin only): `POST /approvals/sao-ke/prepare` returning the draft + summary preview.
4. Retry loop wiring: phase-2 worker re-invokes prepare logic with the rejection note; `ProcessAt` 08:00 (`clock.Now()`); TaskID dedup per attempt.
5. Frontend: prepare button + preview dialog on the sao kê/email surface (desktop + mobile), approval card renders draft body readably (email preview pane).
6. Tests: `backend/tests/integration/flow_sao_ke_automation.go` — prepare→approve→send-once (double-approve = single email, idempotency evidence); reject→advance clock→retried; 3× reject→escalated; incomplete-data prepare → deterministic error, no request created.

## Related Code Files

- Create: `internal/app/services/approval/types/sao_ke.go`, `internal/app/services/llm/drafting.go`, `backend/tests/integration/flow_sao_ke_automation.go`
- Modify: `internal/app/services/notification/email_service.go` (expose summary/attachment for preview), `internal/workers/payroll_report_email_worker.go` (execute hook), approval service, sao kê UI surfaces (both variants)

## Success Criteria

- [ ] Admin effort per cycle ≤ 2 clicks (prepare + approve)
- [ ] Zero double sends across 3 cycles (log evidence)
- [ ] Reject "gửi mai, hôm nay dữ liệu chưa đủ" → next-day retry only runs when data is complete, otherwise reschedules again (within max 3)
- [ ] `make api-test` green

## Risks

- Draft drift: LLM draft must not alter numbers — numbers appear only from the deterministic summary; drafting prompt receives them as fixed quoted strings; card shows both draft and summary so mismatches are visible to the approver.
- Worker semantics change on the critical email path — keep the manual path untouched and branch on approval-context payload.
