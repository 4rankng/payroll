---
title: "Admin attendance review authority"
date: "2026-08-04 23:55"
severity: "High"
component: "Attendance approval and delayed auto-reject"
status: "Resolved"
---

# Admin attendance review authority

## What Happened

We confirmed a real race between manual admin approval and the delayed auto-reject worker. In the bad case, the worker came through after the admin had already approved the attendance and rewrote the row as rejected, leaving `earning_amount` at `0` and `salary_reject_reason` set. The repair path now treats the admin decision as authoritative: if a delayed fallback contradicted an approval, the approval flow repairs the row back to a payable state, restoring the recomputed earning and clearing `salary_reject_reason`.

## The Brutal Truth

This was not a theoretical edge case. We let a delayed fallback overwrite a human decision and then had to build a repair path to undo the damage. That is exactly the kind of race that quietly corrupts payroll data and burns time in the worst possible place: financial correctness.

## Technical Details

- The repository guard for auto-reject now requires `review_action IS NULL`, so a reviewed row is no longer eligible for the delayed worker.
- The admin approve path now repairs legacy rows that were contradicted by the fallback instead of treating every approved row as already clean.
- Regression coverage was added for the backend approval path, the auto-reject sweep, the atomic repository guard, and the desktop/mobile UI label that shows `Duyệt lại` for repair cases.
- The test fixture verifies the repaired financial outcome explicitly: `earning_amount = 300000`, `salary_reject_reason = nil`.

## What We Tried

- Reproduced the stale-row scenario in service-level tests.
- Tightened the persistence guard so the delayed worker cannot win after admin review.
- Added repair-aware approval assertions instead of relying on a one-time happy path.
- Updated the admin UI wording so the operator sees a repair action instead of a normal approve action.

## Root Cause Analysis

The root mistake was treating the delayed auto-reject as if it had equal authority to admin review. It does not. The fallback exists to finalize unresolved rows, not to override a deliberate human approval. The original guard only protected against checkout and prior rejection, so it left a hole for a stale worker to clobber an already-approved row.

## Lessons Learned

Payroll state needs a single authoritative winner. If a manual admin review exists, delayed fallback logic must stop at the persistence boundary, not “fix” the row afterward. Also, repair logic is not optional once a race has shipped; tests need to prove the repair path, not just the ideal path.

## Next Steps

The backend and UI regressions are in place, but local QA is still limited by not exercising the full real async delay under production timing. We need one more end-to-end confirmation on a live-like schedule before calling this completely closed.
