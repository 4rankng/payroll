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
- Legacy rows (created before this fix) fall back to a **lazy per-file** fetch instead of one giant union fetch — zero behavior change for old data.
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

No cross-plan dependencies. Phases are independent and can ship in sequence or in parallel branches, but Phase A is the highest-ROI and should land first.

## Expected impact

| Endpoint / path | Before | After |
|---|---|---|
| `/payrolls/bank-transfer-histories` (new data) | 5 batched `SELECT id, date` + month-union fetch | 0 timesheet queries |
| `/payrolls/bank-transfer-histories` (legacy data) | 5 batched queries over 5334-ID union | small per-file lazy fetches (<500 IDs each) |
| 6 `GetByIDs`/`FindByCodes` methods | break / degrade at >1000 IDs | safe at any count |
| `ListWeeklyForWorkMonth` | all-time files with assets | only requested month |
| `visible_t` CTE | unbounded partner history | ~60-day window |

## Verification gates

- `cd backend && go build ./... && go vet ./...` after each phase
- `go test ./internal/app/services/payroll/... ./internal/app/services/dashboard/... ./internal/infra/persistence/... -v` (excluding pre-existing `wallet_payments` migration failures on baseline `main`)
- `make api-test` before deploy (requires live backend)
- Manual: hit `/payrolls/bank-transfer-histories?month=2026-07` and confirm via GORM `[Xms] [rows:Y]` log that no timesheet queries fire for new data
