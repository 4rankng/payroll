# Wallet bulk-transfer pipeline: 6 CRITICAL money-moving bugs caught at the gate

**Date**: 2026-07-19
**Severity**: High (caught pre-ship; migrations deployed, app code held)
**Component**: `wallet_bulk` (backend Go) + `/admin/wallet` + `/admin/timesheet` (frontend) + migrations `092`/`093`
**Status**: Resolved at code level; app deployment pending user go-ahead

## What Happened

Built and verified the two-stage Wallet Bulk Transfer Pipeline: admins export an OnePay-API-compatible `.xlsx` from `/admin/timesheet` (eMB_BulkPayment, 7 columns, fresh VFIC per row, Mã SWIFT resolved from each employee's bank record), then upload it on `/admin/wallet` where the backend parses, dup-checks by content hash + per-row VFIC, and enqueues one asynq task per row. The row worker mirrors `disbursement_execute_worker.go` line-for-line — the canonical 5-step money flow with explicit Accepted flag, FeeWaived on preflight rejections, `ErrFeeResolution → SkipRetry`. On batch completion, one aggregate Expense transaction books `sum(fee)` via the 3-return idempotent `CreateTransaction`.

Surface area: 13 new backend files + 11 modified, new `wallet_bulk` package (parser / service / queue / KQ generator / sync-dispatcher test helper), new `OnePayExporter` in `bulktransfer`, new domain entity `BulkTransferBatch`, migrations `092` (`bulk_transfer_batches`) and `093` (5 new `wallet_payments` columns + 2 indexes), 3 new React components, 2 page integrations, Vietnamese UI throughout, and two runbooks (smoke test + stuck-completing recovery).

The pipeline went through `ck:cook` green: backend built, 36 unit tests passed, frontend `lint` + `build` clean, migrations verified on both prod and demo. Then the mandatory `code-reviewer` subagent pass found 6 CRITICAL money-moving safety bugs the unit tests structurally could not catch. We fixed all 6 before any app code shipped. Migrations are on prod; app code is not.

## The Brutal Truth

The exhausting reality is that the pipeline was "done" three times today. First time it built. Second time the tests went green. Third time `ck:cook` cleared. None of those meant it was safe — they meant it ran. The reviewer's pass is what made it safe, and every one of the six findings was a real failure mode I had not modeled: a deadlock, a double-book, a missing audit row, a phantom test that lied about coverage, an idempotency-leaking state flip, and a fee-burn on an empty wallet.

What makes this particularly painful is that five of the six were consequences of branches the unit tests already exercised. C1's error branches were tested — what wasn't tested was that returning `ErrDuplicatePaymentInProgress` before inserting a `wallet_payments` row meant that row never counted toward batch completion. The test asserted the error. It did not assert the batch could ever finish. That gap is the whole story.

The decision point that mattered: migrations were already live on prod, but no app code had shipped. We had a clean window to fix without rollback complexity. Asking "fix all 6 criticals + key highs" instead of "ship as-is" was obviously right in hindsight — but it cost a second full pass, and the urge to call it done after `ck:cook` was real.

## Technical Details

The 6 CRITICAL findings, in the order they would have burned us:

- **C1 — Batch strands in `processing` on terminal errors.** When `Initiate` returned `ErrDuplicatePaymentInProgress` or `ErrFeeResolution`, no `wallet_payments` row was inserted. Batch completion is computed as `processed_count == total_count`; skipped rows never incremented either side. Batch deadlocks silently in `processing` forever. **Fix**: new `DecrementTotalCount` repo method; worker atomically subtracts skipped rows from the total so the math still closes.

- **C2 — Double-booked Expense transactions.** Window between `CreateTransaction` committing and `batchRepo.Update(ledger_txn_id)` committing. The `completing`-recovery cron could fire in that gap, see no `ledger_txn_id`, and insert a second Expense row for the same batch. **Fix**: both writes wrapped in a single DB transaction via `TransactionManager.WithTransactionResult`.

- **C3 — No audit trail on upload.** Money-moving upload handler emitted no audit event; every peer handler (`disbursement_execute`, `bulk_transfer_export`, etc.) does. **Fix**: added `AuditEventEmitter` port + audit row on every upload.

- **C4 — Phantom test helper.** `SyncDispatcher` declared interface methods nothing implemented — `ProcessRowTask` vs `ProcessJob` mismatch. Tests using it compiled against the wrong surface and the integration coverage it advertised could not actually run. **Fix**: renamed to match the real dispatcher.

- **C5 — `enqueue_state` flip on partial failure.** On partial enqueue failure the code unconditionally flipped `enqueue_state='enqueued'`, hiding unenqueued rows from the stale-enqueue sweeper. They'd never be retried. **Fix**: only flip on full success; partial failures leave `enqueue_state='pending'` so the sweeper re-enqueues everything (asynq `TaskID` dedup absorbs the duplicates).

- **C6 — No Step-0 pre-flight balance check.** Worker skipped the canonical balance guard that `disbursement_execute_worker` runs as Step 0. A 5,000-row batch against an empty wallet would have burned ~19M VND of fees (5,000 × 3,850 VND per the DB config) for zero successful transfers. **Fix**: re-added the Step-0 guard; underfunded rows are skipped without charging a fee.

Reviewer's analysis was correct on all 6 — verified line numbers and failure modes. Not paranoia.

## What We Tried

- **First**: ship after `ck:cook` green. Rejected when reviewer returned 6 CRITICALs.
- **Then**: fix all 6 criticals + key highs in one pass, re-verify. Accepted. Cost a second full build/test/lint cycle but left the pipeline actually safe.
- **Decision held in tension**: migrations already deployed to prod + demo before the review pass. Could have rolled back schema, but the schema changes (new table + new columns + 2 indexes) are purely additive and the table is empty until app code ships, so leaving them deployed was strictly safer than rolling back.

## Root Cause Analysis

Five of the six findings share one root cause: **unit tests verified what a function returned, not what the system did as a consequence.** C1's branches were tested for their return values. C2's `CreateTransaction` and `Update` were each tested. C5's enqueue path was tested for the success branch. None of those tests could see the batch-accounting or cron-recovery or sweeper consequences because those consequences live in other files running on other schedules.

C6 is a different failure: the worker was specified as "mirror `disbursement_execute_worker.go` line-for-line" and the Step-0 guard was simply dropped during implementation. No test caught it because no test asserted "if balance is zero, no fee is charged." That's a spec-compliance gap, not a coverage gap.

C4 is the most embarrassing: a test helper that didn't implement its own interface. The compiler caught the mismatch only because of how the helper was wired — the tests using it ran against a different surface than they advertised. Coverage numbers lied.

The unifying lesson: for money-moving code, the unit test boundary is the function. The failure boundary is the system. Those are different boundaries.

## Lessons Learned

- **`ck:cook` green is "it runs," not "it's safe."** The code-reviewer pass is mandatory for money-moving features specifically because unit tests can't see cross-schedule consequences (cron recovery, sweepers, batch accounting).
- **Batch accounting requires monotone bookkeeping.** Any code path that skips inserting a row must also adjust the denominator, or completion can never fire. Test for "can the batch ever finish?" not just "does this branch return the right error?"
- **Two-step ledger writes (create txn → record txn id) are a double-book hazard.** Wrap them in a single DB transaction or accept that a cron can land in the gap.
- **Every money-moving handler emits an audit event.** No exceptions. This is now a checklist item, not a judgment call.
- **Sync test helpers must implement the same interface as the production dispatcher — and the compiler must enforce it.** A manually-wired helper can silently drift.
- **State flips on partial failure are idempotency leaks.** Only flip state on full success; let sweepers re-drive from the previous state.
- **Specified as "mirror X line-for-line" means diff against X.** Don't trust the spec translation; trust the diff.
- **Additive schema migrations on an empty table are safe to deploy ahead of app code.** This gave us a clean fix window without rollback risk. Worth remembering as a deliberate deploy strategy.

## Next Steps

- **App deployment**: user runs `make deploy` when ready to enable the feature flag. Set `OTP_ENABLE=true` in production at the same time. Owner: user. When: user's call.
- **Smoke test before going live**: follow `docs/runbooks/wallet-bulk-transfer-smoke-test.md` against demo first. Owner: user + backend on call.
- **Stuck-completing recovery**: `docs/runbooks/wallet-bulk-transfer-stuck-completing-recovery.md` documents the manual SQL recovery path if C1's class of bug ever recurs in production (it shouldn't, but the runbook exists because the failure mode is silent).
- **C4 follow-up**: add a compile-time assertion that `SyncDispatcher` satisfies the real dispatcher interface, so the helper can't drift again. Low priority — owner: backend, next time the package is touched.
- **C6 follow-up**: add a regression test "empty wallet + N rows ⇒ zero fees charged." Owner: backend, this cycle.
