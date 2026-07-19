---
title: "Eliminate >=1000-row fetches: bank-transfer-history + unbatched GetByIDs"
description: "Root-cause fix for the 5,334-row timesheet-date fetch (data-modeling gap in CyclePayData), plus batching for 6 unbatched WHERE id IN (?) repo methods and capping 2 upstream unbounded fetches (ListWeeklyForWorkMonth, visible_t CTE)."
status: pending
priority: P1
branch: "main"
tags: ["performance", "db-query", "bank-transfer-history", "partner-dashboard"]
blockedBy: []
blocks: []
created: "2026-07-19T17:21:47.832Z"
createdBy: "ck:plan"
source: skill
---

# Eliminate >=1000-row fetches: bank-transfer-history + unbatched GetByIDs

## Overview

Triggered by the partner dashboard burst GORM log showing a 5,334-row `SELECT id, date FROM timesheets WHERE id IN (...)` fetch (6 batched queries at 1000/chunk) plus a prior 347ms `/payrolls/bank-transfer-histories` endpoint that loaded full timesheet rows + relations just to read `Date`. An audit found the giant fetch is a **symptom** of a data-modeling gap, not a query to paginate.

**Root cause:** `domain.CyclePayData` (embedded in `TransactionCode.Data` JSON) does not store the cycle's date range. To resolve the weekly cycle (1–4) for each history entry, the service re-derives it by loading every referenced timesheet's date — the union of all timesheet IDs across a whole month's bulk-transfer files. The fix is to persist `FromDate`/`ToDate`/`CycleNum` on `CyclePayData` at creation time, when `plan.FromDate`/`plan.ToDate`/`plan.Cycle` are already in scope.

A parallel audit found **6 unbatched `WHERE id IN (?)` repo methods** that will break at >1000 IDs (MySQL packet limit) and **2 upstream unbounded fetches** (`ListWeeklyForWorkMonth` with `OR asset_id IS NOT NULL` pulls all-time files; the `visible_t` CTE materializes unbounded partner history).

## Goals

- New bulk-transfer data resolves the weekly cycle with **zero timesheet-date queries** (was 5 batched date-only queries after the prior phase, previously 5 `SELECT *` + ~10 relation queries).
- Legacy rows (created before this fix) fall back to the **existing month-level union** fetch (today's behavior) — zero behavior change for old data, no N+1 regression.
- 6 latent production-breaking `GetByIDs`/`FindByCodes` methods become safe at any ID count.
- `ListWeeklyForWorkMonth` returns only the requested month's files (not all-time files with assets).
- `visible_t` CTE bounded to ~60 days (covers active/dropped/paid cutoffs).

## Non-goals (explicitly out of scope)

- One-time SQL backfill of legacy `BulkTransferFile` rows (Phase A's lazy fallback covers them).
- Frontend TanStack Query tuning.
- Remaining `timesheetRepo.GetByIDs` callers in settlement/event handlers — these are per-event/per-upload bounded, not the burst pattern in the log.
- Refactoring `BatchProcessor.ProcessInBatches` itself (works fine; we add a typed helper alongside).

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Phase A: Eliminate timesheet-date fetch at root](./phase-01-phase-a-eliminate-timesheet-date-fetch-at-root.md) | Pending |
| 2 | [Phase B: Batch unbatched GetByIDs/FindByCodes](./phase-02-phase-b-batch-unbatched-getbyids-findbycodes.md) | Pending |
| 3 | [Phase C: Cap upstream unbounded fetches](./phase-03-phase-c-cap-upstream-unbounded-fetches.md) | Pending |

## Dependencies

No cross-plan dependencies. Phases are independent and can ship in sequence or in parallel branches, but Phase A is the highest-ROI and should land first. **Phase C2 depends on product signoff** (red-team F7) — do not deploy C2 until recorded below.

## Expected impact

| Endpoint / path | Before | After |
|---|---|---|
| `/payrolls/bank-transfer-histories` (new data) | 5 batched `SELECT id, date` over month-union | 0 timesheet queries |
| `/payrolls/bank-transfer-histories` (legacy data) | 5 batched queries over 5334-ID union | unchanged (month-level union preserved — no N+1) |
| 6 `GetByIDs`/`FindByCodes` methods | break / degrade at >1000 IDs | safe at any count |
| `ListWeeklyForWorkMonth` | all-time files with assets | requested month + bounded legacy compat clause |
| `visible_t` CTE | unbounded partner history | ~60-day window (semantics change — see F7) |

## Verification gates

- `cd backend && go build ./... && go vet ./...` after each phase
- `go test ./internal/app/services/payroll/... ./internal/app/services/dashboard/... ./internal/infra/persistence/... -v` (excluding pre-existing `wallet_payments` migration failures on baseline `main`)
- `make api-test` before deploy (requires live backend)
- Manual: hit `/payrolls/bank-transfer-histories?month=2026-07` and confirm via GORM `[Xms] [rows:Y]` log that no timesheet queries fire for new data

## Red Team Review

**Date:** 2026-07-20 · **Reviewers:** 3 (Security Adversary, Assumption Destroyer, Failure Mode Analyst) · **Findings:** 15 raw → 9 distinct (all evidence-backed) → **9 accepted, 0 rejected**

| # | Sev | Title | Disposition |
|---|---|---|---|
| F1 | 🔴 Critical | Phase A3 built on non-existent data path (`storeResultMetadata` receives no dates) | ✅ Accept — Phase A rewritten to per-entry resolution, drops file-level denorm |
| F2 | 🔴 Critical | `fixedWeeklyCycle` rejects off-boundary export dates → "0 queries" claim false | ✅ Accept — derive cycle via `KyFromWorkDay` directly, skip `fixedWeeklyCycle` on persisted path |
| F3 | 🔴 Critical | Phase C1 silently hides legacy nil-date files | ✅ Accept — hard pre-deploy COUNT gate + bounded compat clause |
| F4 | 🟠 High | File-level `FromDate` wrong when result file mixes Ky1+Ky2 | ✅ Accept — per-entry resolution (subsumed by F1 fix) |
| F5 | 🟠 High | Legacy path becomes N+1 by file count | ✅ Accept — month-level union for legacy (today's behavior) |
| F6 | 🟠 High | Phase B `Chunk` breaks tx-scoped reads (`transactionRepository.GetByIDs` uses `r.db`) | ✅ Accept — precondition: switch to `getDB(ctx)` before batching |
| F7 | 🟠 High | Phase C2 changes `dropped_employees` semantics + positional-arg fragility | ✅ Accept — named placeholders + product signoff gate + observability metric |
| F8 | 🟡 Medium | B7 signature change breaks `BuildGetByIDsQuery`'s only caller | ✅ Accept — drop B7, keep silent truncation with metrics |
| F9 | 🟡 Medium | Phase A3 must not touch `BulkTransferFile.Cycle` (`*string`) — rollback footgun | ✅ Accept — Phase A no longer modifies any file-level field |

### Open product signoffs (blocking)

- [ ] **F3:** legacy nil-date row count recorded; backfill-vs-compat-clause decision made (Phase C1 precondition)
- [ ] **F7:** product accepts "dropped_employees = inactive 14-60d" (was "inactive >14d") (Phase C2 precondition)

### Whole-Plan Consistency Sweep

Performed 2026-07-20 after applying all 9 findings. Searched all plan files for stale terms from the original design:
- ✅ "file-level denormalization" / "storeResultMetadata picks first non-nil" — removed from Phase A (replaced with per-entry resolution)
- ✅ "lazy per-file fetch" — removed from Phase A (replaced with month-level union for legacy)
- ✅ "B3 / B7 fail loudly" — removed from Phase B (replaced with metrics-only observability)
- ✅ "LIMIT/OFFSET on ListWeeklyForWorkMonth" — removed from Phase C (aggregates would undercount)
- ✅ "positional `?` for CTE bound" — removed from Phase C (replaced with named placeholders)
- ✅ "covers the dropped-employee window" — struck; replaced with explicit semantics-change note + product signoff gate

**Unresolved contradictions:** none. Plan is ready for `/ck:cook`.
