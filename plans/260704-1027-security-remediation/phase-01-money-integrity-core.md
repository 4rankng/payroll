---
phase: 1
title: "Money integrity core"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Money integrity core

## Overview
Close three money-path integrity gaps. **H5:** `FlexPaySettlementService.ProcessSettlementFile` swallows ledger errors and returns `Success:true`, with no idempotency. **H6:** `RetryDisbursement` mints a fresh asynq task per click (residual = duplicate tasks, NOT double-pay — the worker is already idempotent). **H7:** `MarkReconciled` bypasses the wallet FSM. Red-team corrected the original design assumptions; see `## Red Team Corrections`.

## Requirements
- Functional: a settlement that crashes mid-flow must not leave the books inconsistent or report success; reconcile overrides must not break the documented `failed→completed` flip.
- Non-functional: idempotent under re-upload; ledger errors propagate; H6's guarantee is restated honestly (no double-pay — already enforced by the worker).

## Architecture
- **H5 (reshaped):** `ProcessSettlementFile` loops over many transactions (`for txnID, tx := range txnMap`). Wrap **per-iteration** in `db.Transaction` (one tx per record + deadlock-retry + idempotent skip) — NOT a single tx wrapping the whole loop (would roll back all valid settlements on one stale ID). The "mirror `ApplySettlement`" analogy in v1 was false: `ApplySettlement` is the single-row *timesheet* path; this service writes `advance_payment_requests`+`transactions`+`ledger` and needs a per-row tx.
  - `FlexPaySettlementService` has **no `*gorm.DB`** today — add it to the struct, constructor, and bootstrap wiring (`init.go:318`, `container.go:337`).
  - Replace **both** swallow paths (`continue` at 134-136; no-return at 159-162) with `return err`.
  - Use `GetByIDForUpdate` (row-locked), not `GetByID` — avoids write skew under concurrent settlement.
  - Idempotency: new **`settlement_uploads(id, file_hash, uploaded_at, request_ids_json, settled_count)`** audit table with a plain `UNIQUE(file_hash)` (MySQL-compatible — do NOT use `UNIQUE ... WHERE`, which MySQL doesn't support). Check `SELECT id FROM settlement_uploads WHERE file_hash = ?` BEFORE touching `transactions`/ledger; on hit, skip + return prior result.
- **H6 (reshaped — severity dropped to Medium):** the execute worker (`WalletPaymentService.Initiate`) is **already idempotent** — `HasPendingForRecipient` + `request_id` unique index + `ErrDuplicatePaymentInProgress` (treated as terminal by `disbursement_execute_worker.go:109-122`). Double-pay is already prevented; H6's residual is duplicate asynq tasks (log noise). Use **asynq `UniqueTTL`** bound to `(advance_request_id, amount)` so a second enqueue is deduped at the queue. The handler-level guard uses the **correct column `entity_id`** (NOT `advance_request_id`, which does not exist on `wallet_payments`; the link is `wallet_payments.entity_id = advance_request_id`).
- **H7:** add a status-allowlist precondition `WHERE id = ? AND status IN ?`. Allowlist sourced from the existing `ReconcilePayment` switch (`wallet_payment_service.go:719-777`): `{StateFailed, StateCompleted, StateAuthorised, StateVerified}`. `StatePending`/`StateReversed` fall through (rejected).

## Related Code Files
- Modify: `backend/internal/app/services/flex_pay/flex_pay_settlement_service.go` (H5 — struct + `ProcessSettlementFile` 53-188; swallow paths 134-136, 159-162, `Success:true` 181)
- Modify: `backend/internal/app/bootstrap/services/init.go:318` + `container.go:337` (H5 — inject `*gorm.DB`)
- Modify: `backend/internal/transport/http/handlers/advance_payment/retry_disbursement_handler.go` (H6 — `UniqueTTL` on enqueue ~line 86; dedup query on `entity_id`)
- Modify: `backend/internal/infra/persistence/tx_wallet_payment_repository.go` (H7 — `MarkReconciled` 273-292)
- Migration: new `settlement_uploads` table (paired `.up.sql`/`.down.sql`, idempotent) — see Deploy & Rollback in `plan.md`
- Reference: `backend/internal/infra/events/settlement_event_handler.go` (`ApplySettlement` — per-record tx + `GetByIDForUpdate` pattern to mirror); `wallet_payment_service.go:163-209, 719-777` (existing dedup + reconcile switch)
- Tests: `flex_pay_settlement_service_test.go` (forced-error per-iteration rollback; idempotent re-upload), retry handler test (duplicate enqueue deduped), `MarkReconciled` regression test per transition

## Implementation Steps
0. **Confirm prod volume** on `FlexPaySettlementService.ProcessSettlementFile` (called from `recon_settlement_handler.go:58`). If it's dead/legacy in practice, re-scope H5 — but wiring confirms it's live for FlexPay recon.
1. **H5 — struct + wiring.** Add `db *gorm.DB` to `FlexPaySettlementService`; update constructor + `init.go:318` + `container.go:337`.
2. **H5 — per-iteration tx.** Wrap each `txnMap` iteration in `db.Transaction`; mirror `ApplySettlement`'s deadlock-retry + idempotent skip. Replace both swallow paths with `return err`.
3. **H5 — row lock.** Switch the per-tx lookup to `GetByIDForUpdate`.
4. **H5 — idempotency table.** Create `settlement_uploads` (migration). Compute file SHA-256; check existence before any write; insert on success.
5. **H6 — asynq UniqueTTL.** Bind task uniqueness to `(advance_request_id, amount)` with `asynq.UniqueTTL(...)`. Add a handler-level `HasNonTerminalByEntityID(ctx, advanceRequestID)` query (column `entity_id`) → 409 if a non-terminal row exists. (Belt-and-suspenders; the worker is the real guard.)
6. **H7 — precondition.** Change `Where("id = ?", id)` → `Where("id = ? AND status IN ?", id, []State{Failed, Completed, Authorised, Verified})`. No RowsAffected → `ErrNotFound`.
7. **Tests.** Per-iteration forced-error rollback (H5); re-upload idempotency no-op (H5); duplicate retry → one task via UniqueTTL (H6); `MarkReconciled` regression per transition incl. `failed→completed` (H7).

## Success Criteria
- [ ] H5: re-upload of the same file is a no-op (`settlement_uploads.file_hash` hit); a forced mid-iteration error rolls back that record and propagates (no `Success:true` on failure); other records in the batch still settle.
- [ ] H6: duplicate `POST /:id/retry-disbursement` does not double-pay (worker idempotency asserted by test); second enqueue deduped by `UniqueTTL`.
- [ ] H7: `MarkReconciled` rejects `StatePending`/`StateReversed`; `failed→completed` reconcile override still succeeds; regression test per transition green.
- [ ] All ledger errors propagate (no swallow → `Success:true`).

## Risk Assessment
- **H5 migration on live table:** new `settlement_uploads` table is additive (no lock risk on existing tables). Deploy migration before backend; paired `.down.sql`; manual DDL runbook for demo/prod (`schema_migrations` is dump-seeded, empty). *Mitigation:* see Deploy & Rollback in `plan.md`.
- **H6 false-positive 409:** a stuck non-terminal row blocks retries. *Mitigation:* runbook — clear stuck payments via reconcile, not this endpoint; 409 message names the stuck `entity_id`.
- **H7 under-/over-constrain:** allowlist is sourced from the live `ReconcilePayment` switch (not guessed), so the documented overrides are covered.

## Red Team Corrections
- **H5 design was wrong (Critical):** "mirror `ApplySettlement`" is a false analogy — different tables, multi-row loop, no `*gorm.DB`. Reshaped to per-iteration tx + new `settlement_uploads` audit table (not `transactions`) + explicit bootstrap wiring.
- **H5 migration was wrong (Critical):** table unspecified, MySQL `UNIQUE ... WHERE` invalid, no `.down.sql`. Reshaped to additive audit table + plain UNIQUE + paired down migration.
- **H6 severity overstated (High→Medium):** worker is already idempotent; column was `advance_request_id` (nonexistent, real name `entity_id`). Reshaped to `UniqueTTL` + honest success criterion.
- **H7 allowlist under-specified (High):** was "coordinate with owner" — now baked from `ReconcilePayment` switch: `{Failed, Completed, Authorised, Verified}`.
