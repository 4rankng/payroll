---
phase: 1
title: "Money integrity core"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Money integrity core

## Overview
Close the three money-path integrity gaps: FlexPay settlement lacks a transaction around the status UPDATE + ledger create (H5), `RetryDisbursement` has no in-flight dedup (H6), and `MarkReconciled` bypasses the wallet FSM (H7). Any of the three can cause double-credit / double-pay or out-of-balance books.

## Requirements
- Functional: a settlement/retry/reconcile that crashes mid-flow must not leave the books inconsistent; a retried disbursement must never double-pay.
- Non-functional: idempotent under retry; ledger errors surface to the caller (not swallowed → `Success:true`).

## Architecture
- **H5:** wrap `ProcessSettlementFile`'s per-request status UPDATE + ledger-entry create in a single `db.Transaction`; add a SHA-256 file-hash idempotency column with a unique index so a re-upload is a no-op; return the ledger error on failure. Mirror the already-fixed timesheet path (`SettlementEventHandler.ApplySettlement` — one tx per record, deadlock-retry, idempotent skip).
- **H6:** before minting a new `disbursementRequestID` + asynq task, query `wallet_payments WHERE advance_request_id = ? AND status NOT IN (terminal)`; if any row exists, return **409 Conflict**.
- **H7:** add a status-allowlist precondition to `MarkReconciled` (`WHERE id = ? AND status IN (...)`) so reconciliation can only override from legal prior states; do not silently regress version semantics.

## Related Code Files
- Modify: `backend/internal/app/services/flex_pay/flex_pay_settlement_service.go` (H5 — `ProcessSettlementFile`, audit-cited 115-188)
- Modify: `backend/internal/transport/http/handlers/advance_payment/retry_disbursement_handler.go` (H6 — insert dedup before Step 6, ~line 67)
- Modify: `backend/internal/infra/persistence/tx_wallet_payment_repository.go` (H7 — `MarkReconciled` 273-292) + repo interface
- Reference: `backend/internal/infra/events/settlement_event_handler.go` (the already-fixed *timesheet* settlement path — `ApplySettlement` is tx + idempotent; mirror its pattern for H5)
- Migration: add `settlement_file_hash` col + unique index (nullable → backfill → unique, or partial `UNIQUE ... WHERE`)
- Tests: `flex_pay_settlement_service_test.go` (tx rollback), `retry_disbursement_handler_test.go` (409 on in-flight), extend wallet-payment repo tests (MarkReconciled precondition)

## Implementation Steps
1. **H5 — confirm + wrap.** Read `ProcessSettlementFile` end-to-end; confirm the audit-cited gap (status UPDATE + ledger create are separate writes, error swallowed → `Success:true`). Wrap both writes in `db.Transaction` mirroring `ApplySettlement`.
2. **H5 — idempotency key.** Add `settlement_file_hash` (SHA-256) + unique index. Re-upload of the same file → detect, skip, return existing result.
3. **H5 — propagate errors.** Remove the swallow; return the ledger error so the upload reports failure.
4. **H6 — dedup query.** Add `HasNonTerminalForAdvanceRequest(ctx, advanceRequestID)` to the wallet-payment repo. In `RetryDisbursement`, call it before Step 6; on true, return 409.
5. **H6 — bind identity.** Ensure the disbursement request is bound to `(advance_request_id, amount)` so a stale payload can't mutate amount mid-flight.
6. **H7 — status precondition.** Change `Where("id = ?", id)` → `Where("id = ? AND status IN ?", id, allowedPriorStates)`. Enumerate allowed transitions with the reconcile-service owner before constraining.
7. **Tests.** Forced-error tx rollback (H5); double-retry → 409 (H6); `MarkReconciled` from a disallowed state → rejected (H7).

## Success Criteria
- [ ] H5: re-upload of the same file is a no-op (idempotency hit); a forced mid-flow crash leaves no orphan ledger entry.
- [ ] H6: two concurrent `POST /:id/retry-disbursement` → one 200, one 409; no second asynq task enqueued.
- [ ] H7: `MarkReconciled` from a terminal/disallowed state is rejected; books stay consistent.
- [ ] All ledger errors propagate (no `Success:true` on failure).

## Risk Assessment
- **H5 migration on a live table:** the file-hash unique index can lock existing rows. *Mitigation:* add col nullable → backfill → add unique index in a follow-up migration (or `UNIQUE ... WHERE` partial index); deploy migration before backend.
- **H6 false-positive 409:** a stuck non-terminal row blocks legit retries. *Mitigation:* document the runbook — retry the *stuck* payment via reconcile, not this endpoint.
- **H7 narrowing allowed states:** reconciliation currently relies on the bare update for legitimate overrides (e.g. failed→completed after IPN contradiction). *Mitigation:* enumerate exact legal transitions from the reconcile service before constraining; coordinate with its owner.
