---
phase: 3
title: "Phase C: Cap upstream unbounded fetches"
status: pending
priority: P2
effort: "S"
dependencies: []
---

# Phase C: Cap upstream unbounded fetches

## Overview

Two upstream fetches are unbounded and feed the >=1000-row pattern from the wrong end. `ListWeeklyForWorkMonth` returns all-time files via `OR asset_id IS NOT NULL`. The `visible_t` CTE in `GetPartnerEmployeeStats` materializes a partner's entire timesheet history. We bound each to the window its consumers actually need.

## Background / why

### C1 — `ListWeeklyForWorkMonth`
File: `backend/internal/infra/persistence/bulk_transfer_file_repository.go:140-149`

```go
Where("cycle = ?", "weekly").
Where(r.DB.Where("from_date >= ? AND from_date <= ?", monthStart, monthEnd).
    Or("asset_id IS NOT NULL")).          // ← pulls all-time files with assets
Where("data IS NOT NULL AND data != '' AND data != '{}' ")
```

The `OR asset_id IS NOT NULL` clause was added to catch result-upload files (which historically had nil `from_date`). Combined with no `LIMIT`, this returns every weekly result file the partner ever uploaded. That inflates `transactionCodes`, `timesheetIDSet`, and downstream fetches.

After Phase A, result-upload files **will** have `from_date` populated, so the `OR asset_id IS NOT NULL` escape hatch is no longer needed for new data. For legacy files, the date predicate should be `from_date BETWEEN monthStart AND monthEnd` and we accept that legacy nil-date files are not surfaced by this query (they are reachable via other history paths if needed).

### C2 — `visible_t` CTE
File: `backend/internal/infra/persistence/repositories/timesheet_analytics_repository.go:665-690`

```sql
WITH visible_t AS (
  SELECT t.id, t.employee_id, t.date, t.payment_status, t.paid_amount, t.project_id
  FROM timesheets t
  WHERE t.deleted_at IS NULL
    AND (t.employee_id IN (...) OR t.project_id IN (...))   -- ← no date bound
)
```

The 3 outer metrics against `visible_t`:
- `active_employees`: `date >= now-14d`
- `dropped_employees`: `date < now-14d AND NOT EXISTS(... date >= now-14d)`
- `paid_employees` / `total_paid_amount`: `date` in the selected month

The union of these needs at most ~60 days of history (14d active window + 14d-30d dropped window + current month). Materializing years of history for a large partner is pure waste.

## Requirements

- **Functional:** `ListWeeklyForWorkMonth` returns only files whose `from_date` falls in the requested month. `GetPartnerEmployeeStats` returns identical metric values for any window ≥ 60 days (the dropped-employee logic needs the 14-day-ago boundary, so we must not exclude it).
- **Non-functional:** both queries become bounded — file count per month (~tens), timesheet rows per partner per ~60 days (~hundreds-to-low-thousands).

## Related code files

- Modify: `backend/internal/infra/persistence/bulk_transfer_file_repository.go` — `ListWeeklyForWorkMonth` predicate + optional `LIMIT`/`OFFSET`.
- Modify: `backend/internal/app/services/payroll/service.go:312` — pass pagination if signature changes.
- Modify: `backend/internal/infra/persistence/repositories/timesheet_analytics_repository.go` — `GetPartnerEmployeeStats` CTE gains a `t.date >= ?` bound.

## Implementation steps

1. **C1 — Tighten `ListWeeklyForWorkMonth`:**
   - Replace the nested `Where(... .Or("asset_id IS NOT NULL"))` with a simple `Where("from_date BETWEEN ? AND ?", monthStart, monthEnd)`.
   - Decide on `LIMIT`/`OFFSET`: the history endpoint already paginates *aggregates*, but the file fetch is per-month. Add optional `limit, offset int` params; default to a sane cap (e.g. 500) when unset. Update the call site in `payroll/service.go:312`.
   - **Document the legacy-file caveat** in the function comment: nil-`from_date` files created before Phase A are no longer surfaced here; if business needs them, add a separate explicit path (or a backfill — see plan non-goals).

2. **C2 — Bound the `visible_t` CTE:**
   - Compute `historyStart := clock.Now().AddDate(0, 0, -60)` inside `GetPartnerEmployeeStats`.
   - Add `AND t.date >= ?` to the CTE's WHERE, with `historyStart` appended to `args` **before** the existing cutoff args (preserve positional arg order — the accessCond args come first, then historyStart, then the 3 cutoff args, then paidArgs).
   - Verify the `dropped_employees` subquery still works: it filters `vt.date < cutoff` (now-14d) with `NOT EXISTS(... vt2.date >= cutoff)`. Since the CTE now starts at now-60d, any employee whose last timesheet is older than 60 days is simply not in `visible_t` — they would have been counted as dropped only if they had a pre-14d timesheet. Decide:
     - **Option A (recommended):** accept the slight semantics change — employees inactive for >60 days are excluded from "dropped" (they are effectively "churned", not "dropped this period"). This matches typical retention-window definitions.
     - **Option B:** widen the window to cover the longest plausible "dropped then churned" gap. Avoid — defeats the bound.

3. **Verify empty-scope short-circuits** in the weekly/monthly partner stats methods still apply (they do, from the prior phase's `PartnerScope.Empty()` guard).

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] `ListWeeklyForWorkMonth` unit/integration test asserts only in-month files are returned
- [ ] Manual log check: `/dashboard/partner?month=2026-07` — the `visible_t` CTE query shows a `[rows:N]` count in the low hundreds, not thousands+
- [ ] Manual: partner dashboard metric values unchanged for active/dropped/paid counts (Option A) — confirm with the partner user

## Risk assessment

- **C1 legacy files:** dropping `OR asset_id IS NOT NULL` means nil-`from_date` files vanish from the monthly history view. After Phase A, new files have dates; the only exposure is legacy nil-date files that *also* have an asset. Mitigation: confirm with a data query how many such legacy rows exist; if material, run a one-time backfill (out of scope here) before deploying C1. If immaterial, ship as-is.
- **C2 "dropped" semantics:** Option A changes the definition slightly. Confirm the product expectation with the partner before shipping, or pick a wider window. The `active` and `paid` metrics are unaffected.
- **Arg ordering in `GetPartnerEmployeeStats`:** the raw SQL uses positional `?` placeholders; inserting `historyStart` in the wrong position would corrupt the query. The implementation step calls out the exact position.
