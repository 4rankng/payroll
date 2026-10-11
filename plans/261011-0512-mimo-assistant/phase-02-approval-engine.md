---
title: "Phase 2: Approval Engine"
status: todo
---

# Phase 2: Approval Engine

## Overview

Durable approval-request state machine with reject-with-note → scheduled retry (08:00 ICT, max 3, then escalation). Domain + API + RBAC + audit only; UI cards are phase 3.

## Requirements

- [ ] `approval_requests` table + domain entity with guarded transitions: `pending → approved → executed | rejected → rescheduled | escalated | expired | cancelled`
- [ ] Fields: type (`send_sao_ke_email` | `send_payroll_email` | `send_reminder` | `void_timesheet` | extensible), payload JSON, requester/approver, decision note, `scheduled_at`, `retry_count`, `idempotency_key`, timestamps
- [ ] Per-type **deterministic** precondition checks (e.g. "all statements of cycle generated") evaluated before every retry — the LLM never judges data completeness
- [ ] Idempotency: unique key per (type, cycle, recipient) enforced at creation and at execution (BCC dedup precedent, `bcc_import_dedup_test.go`)
- [ ] Reject(note) → asynq task `ProcessAt` next day 08:00 (`EnqueueAutoRejectCheckout` precedent in `internal/infra/asynq/client.go`); `retry_count++`; after 3 → `escalated` + notification. Retry TaskID is **attempt-scoped** (`approval-retry:{id}:{attempt}`) — constant TaskIDs collide with retained completed tasks and silently swallow retries (`client.go:93-105` documents this); retries staggered across minutes (N requests × Excel gen + pro LLM call in one 08:00 burst is a self-inflicted thundering herd)
- [ ] Casbin rows in `configs/casbin_policy.csv` — **spelled out in this phase, not deferred to a PR note**: explicit deny rows `p, partner|adv_partner|employee|accountant, /api/v1/approvals/:id/approve|reject|cancel, *, deny` (keyMatch2 widening is documented in the CSV itself, lines ~171–175; wallet-movements precedent, lines ~216–222); allow only `POST /approvals` (filers) + `GET /approvals` (own rows for non-admin)
- [ ] Payload safety: `send_*` request payloads are **server-derived** — recipients resolved from cycle data, content assembled server-side; the filer chooses the action type from a per-role allowlist (partner may file none of the `send_*` types at MVP — sao kê prepare is admin-only); creation rate limit wired to `POST /approvals` and the chat tool
- [ ] **Reject vs cancel are distinct semantics, both on the card**: "Thử lại ngày mai" (reject + reschedule, note = drafting guidance) and "Huỷ / không gửi lại" (cancel, terminal — no retry, no escalation). Default button label must not say "Từ chối" alone — the repo's reject precedent (attendance review) is terminal, so silent rescheduling violates the admin's mental model
- [ ] Every transition published as domain event → `audit_logs` (event factory pattern, `domain/event_factory_*.go`)
- [ ] Ownership: `GET /approvals` returns own rows for non-admin requesters (handler-level, not just Casbin path rules)
- [ ] Integration tests cover the full lifecycle incl. RBAC negatives and retry scheduling

## Implementation Steps

1. Domain: `internal/domain/approval_request.go` — status enum + `CanBeApproved/CanBeRejected/CanBeRescheduled` guards + transition methods (pattern: `timesheet_edit_request.go`; stateless-machine option: `domain/transactions/state_machine.go` `qmuntal/stateless` — prefer plain enum+guards, KISS).
2. Persistence: GORM model + repository in `internal/infra/persistence/approval_request_repository.go`; migration via the repo's existing mechanism.
3. Service: `internal/app/services/approval/service.go` — Create, Approve (→ execution handoff hook consumed in phase 4), Reject(note), Cancel, retry-handler invoked by worker; precondition registry per type.
4. Worker: `internal/app/workers/approval_retry_worker.go` + task constant in `infra/asynq/handlers.go`, mux registration in `mux.go`; enqueue helper `EnqueueApprovalRetry` with `TaskID` dedup + `ProcessAt`.
5. API: `internal/transport/http/handlers/approval/` + `routes_approvals.go` — `GET /approvals` (filters: status, type, requester), `POST /approvals/:id/approve|reject|cancel`.
6. RBAC: casbin rows + negative tests in `authorization_service_test.go` style.
7. Tests: `backend/tests/integration/flow_approval.go` (`runApprovalsTests` registered in `main.go`) — create→approve→executed; create→reject(note)→rescheduled at 08:00 (advance clock via `clock_helper.go`); 3 rejections→escalated; partner 403 on approve; duplicate idempotency key rejected.

## Related Code Files

- Create: `internal/domain/approval_request.go`, `internal/app/services/approval/`, `internal/app/workers/approval_retry_worker.go`, `internal/transport/http/handlers/approval/`, `internal/app/bootstrap/routes_approvals.go`, `backend/tests/integration/flow_approval.go`
- Modify: `internal/infra/asynq/{client,handlers,mux}.go`, `configs/casbin_policy.csv`, `internal/app/bootstrap/container.go`, `internal/app/services/init.go`, `tests/integration/main.go`

## Success Criteria

- [ ] All transitions guarded — invalid transitions return domain errors (property/unit tests)
- [ ] Retry fires at 08:00 ICT next day in tests (clock advanced); escalation after 3
- [ ] Partner cannot approve (403) — Casbin verified
- [ ] `make api-test` green for `flow_approval.go`

## Risks

- Double execution on approve-click races → execution handoff keyed by idempotency key; approve is a single DB transition under tx.
- Casbin CSV is startup-loaded: policy changes need redeploy — document new rows in the phase PR.
