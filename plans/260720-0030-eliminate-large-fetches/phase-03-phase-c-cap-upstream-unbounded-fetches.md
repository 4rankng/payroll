---
phase: 3
title: 'Phase C: Cap upstream unbounded fetches'
status: completed
priority: P2
effort: M
dependencies: []
---

# Phase C: Cap upstream unbounded fetches

## Overview

Two upstream fetches are unbounded and feed the >=1000-row pattern from the wrong end. `ListWeeklyForWorkMonth` returns all-time files via `OR asset_id IS NOT NULL`. The `visible_t` CTE in `GetPartnerEmployeeStats` materializes a partner's entire timesheet history. We bound each to the window its consumers actually need — with a **hard pre-deploy gate** for the legacy-row problem (red-team F3) and **named placeholders** for arg-ordering safety (red-team F7).

> **Red-team revision (findings F3, F7, F9-limit):**
> - **F3 (Critical):** dropping `OR asset_id IS NOT NULL` silently hides every pre-Phase-A result-upload file (they have nil `from_date`). Partners will believe payments weren't made. **Added a hard pre-deploy COUNT gate** and a **backward-compat clause** for nil-date rows.
> - **F7 (High):** the 60-day CTE bound changes `dropped_employees` semantics (long-churned employees vanish). **Requires product signoff** before ship. Also: use **named placeholders** (`@historyStart`, `@cutoff`) instead of positional `?` to make arg-ordering unbreakable.
> - **LIMIT dropped (red-team F9):** adding `LIMIT/OFFSET` to the file fetch silently undercounts aggregates (`total := len(items)` is computed post-aggregation). The `from_date BETWEEN` bound is sufficient; LIMIT is a footgun with no benefit.

## Background / why

### C1 — `ListWeeklyForWorkMonth`
File: `backend/internal/infra/persistence/bulk_transfer_file_repository.go:140-149`

```go
Where("cycle = ?", "weekly").
Where(r.DB.Where("from_date >= ? AND from_date <= ?", monthStart, monthEnd).
    Or("asset_id IS NOT NULL")).          // ← pulls all-time files with assets
Where("data IS NOT NULL AND data != '' AND data != '{}' ")
```

The `OR asset_id IS NOT NULL` clause was added to catch result-upload files (which historically had nil `from_date`). Combined with no `LIMIT`, this returns every weekly result file ever uploaded. That inflates `transactionCodes`, `timesheetIDSet`, and downstream fetches.

After Phase A, result-upload files' **transaction codes** carry `FromDate` (per-entry), but the `BulkTransferFile` row itself still has nil `from_date` (Phase A no longer denormalizes to the file level — red-team F1). So the `OR asset_id IS NOT NULL` clause is still the only thing surfacing legacy result-upload files. We cannot drop it without a backfill.

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

The 3 outer metrics: `active` (date ≥ now-14d), `dropped` (date < now-14d AND no recent), `paid` (date in selected month). Bounding the CTE to ~60 days covers active + recently-dropped + paid month — but **excludes long-churned employees from the dropped count** (red-team F7). That is a real metric change requiring product signoff.

## Requirements

- **Functional:** `ListWeeklyForWorkMonth` returns only files for the requested month, PLUS legacy nil-date files (via a bounded compat clause). `GetPartnerEmployeeStats` metric values are unchanged for active/paid; `dropped_employees` is redefined to "inactive 14-60 days" (was "inactive >14 days") — **product must sign off**.
- **Non-functional:** both queries bounded; no silent data loss; no silent metric drift.

## Related code files

- Modify: `backend/internal/infra/persistence/bulk_transfer_file_repository.go` — `ListWeeklyForWorkMonth` predicate (compat clause, no LIMIT).
- **NOT modified:** `payroll/service.go:312` — no signature change (no LIMIT/OFFSET params).
- Modify: `backend/internal/infra/persistence/repositories/timesheet_analytics_repository.go` — `GetPartnerEmployeeStats` CTE: named placeholders + 60-day bound.
- **New:** one-time SQL backfill (run manually before deploy, not a code migration) for legacy nil-date `BulkTransferFile` rows.

## Implementation steps

1. **C1 — Pre-deploy COUNT gate (red-team F3, hard requirement):**
   Before deploying C1, run against production:
   ```sql
   SELECT COUNT(*) FROM bulk_transfer_files
   WHERE cycle='weekly' AND from_date IS NULL AND asset_id IS NOT NULL;
   ```
   - If count > 0: **either** backfill those rows' `from_date` (derive from `transaction_codes.Data.WeeklyPay.FromDate` once Phase A is deployed) **or** keep the compat clause (step 2). Do not deploy C1 without one of these.
   - Record the count + decision in the plan's Red Team Review section.

2. **C1 — Bounded compat clause (replaces the broad `OR asset_id IS NOT NULL`):**
   ```go
   Where("cycle = ?", "weekly").
   Where(
       r.DB.Where("from_date BETWEEN ? AND ?", monthStart, monthEnd).
           Or("asset_id IS NOT NULL AND from_date IS NULL"), // legacy compat, bounded by nil-date
   ).
   Where("data IS NOT NULL AND data != '' AND data != '{}' ")
   ```
   - The new `AND from_date IS NULL` qualifier bounds the legacy branch to genuinely-old rows. New-data rows (which Phase A *could* populate on the file if a future phase adds file-level denormalization) are surfaced via the date predicate.
   - Add a `// TODO(backfill): remove this clause once legacy nil-date rows are backfilled` comment with a target date.
   - **Do NOT add LIMIT/OFFSET** (red-team F9): `total := len(items)` at `service.go:509` is computed post-aggregation; truncating files silently undercounts. The `from_date BETWEEN` bound is sufficient.

3. **C2 — Named placeholders + 60-day bound (red-team F7):**
   Rewrite the CTE query in `GetPartnerEmployeeStats` to use GORM named-parameter syntax (`@name`) instead of positional `?`:
   ```go
   historyStart := clock.Now().AddDate(0, 0, -60)
   query := `
       WITH visible_t AS (
           SELECT t.id, t.employee_id, t.date, t.payment_status, t.paid_amount, t.project_id
           FROM timesheets t
           WHERE t.deleted_at IS NULL
             AND (t.employee_id IN (@employee_ids) OR t.project_id IN (@project_ids))
             AND t.date >= @history_start
       )
       SELECT
           (SELECT COUNT(DISTINCT vt.employee_id) FROM visible_t vt WHERE vt.date >= @cutoff) AS active_employees,
           (SELECT COUNT(DISTINCT vt.employee_id) FROM visible_t vt
            WHERE vt.date < @cutoff
              AND NOT EXISTS (SELECT 1 FROM visible_t vt2 WHERE vt2.employee_id = vt.employee_id AND vt2.date >= @cutoff)
           ) AS dropped_employees,
           COUNT(DISTINCT CASE WHEN vt.payment_status = 'paid' THEN vt.employee_id END) AS paid_employees,
           COALESCE(SUM(CASE WHEN vt.payment_status = 'paid' THEN vt.paid_amount ELSE 0 END), 0) AS total_paid_amount
       FROM visible_t vt
       WHERE 1 = 1` + strings.ReplaceAll(paidDateFilter, "t.date", "vt.date")
   ```
   - Pass args as a `map[string]interface{}{"employee_ids": scope.EmployeeIDs, "project_ids": scope.ProjectIDs, "history_start": historyStart, "cutoff": cutoff}` plus the paid args.
   - **Named placeholders make arg-ordering unbreakable** (red-team F7/CRIT-2). A future refactor that adds a third `IN (@x)` cannot shift the binding of `@cutoff`.
   - Verify GORM raw SQL supports `@name` with slice expansion for `IN (@employee_ids)` — if not, keep positional `?` for the IN-lists but use named for `history_start`/`cutoff`. Document the chosen approach.

4. **C2 — Product signoff gate (red-team F7, hard requirement):**
   Before deploying C2, confirm with the product/partner owner: "Employees inactive >60 days will no longer count as 'dropped' in the partner dashboard. They will simply not appear in active/dropped/paid counts. Is this acceptable, or should we widen the window?"
   - Record the decision in the plan.
   - Add an observability metric: `partner_stats_dropped_excluded_by_window` counting employees excluded by the 60-day cutoff, so the definition change is observable over time.

5. **Verify empty-scope short-circuits** in the weekly/monthly partner stats methods still apply (they do, from the prior phase's `PartnerScope.Empty()` guard).

## Success criteria

- [ ] `go build ./... && go vet ./...` clean
- [ ] **F3 pre-deploy gate:** legacy nil-date row count recorded; backfill decision documented
- [ ] **F7 product signoff:** recorded in plan before C2 deploy
- [ ] `ListWeeklyForWorkMonth` test asserts: in-month dated files returned, legacy nil-date files returned (compat clause), out-of-month dated files NOT returned
- [ ] `GetPartnerEmployeeStats` uses named placeholders (arg-ordering unbreakable)
- [ ] **F7 regression test (new):** fixture with employees at now-13d (active), now-20d (dropped), now-70d (excluded by window) — assert counts match the new semantics
- [ ] Manual log check: `/dashboard/partner?month=2026-07` — `visible_t` CTE `[rows:N]` in the low hundreds, not thousands+
- [ ] No LIMIT on `ListWeeklyForWorkMonth` (aggregates not truncated)

## Risk assessment

- **F3 legacy files (Critical — accepted):** the bounded compat clause + pre-deploy COUNT gate ensures no silent data loss. The clause is self-documenting with a TODO and target date.
- **F7 dropped-employees semantics (High — accepted):** the product signoff gate + observability metric make the definition change explicit and reversible. If product rejects, widen the window (the named-placeholder approach makes this a one-line change).
- **F9 aggregate undercounting (Medium — accepted):** dropping LIMIT eliminates this risk entirely. The `from_date BETWEEN` bound is the real cap.
- **F7 arg-ordering (Critical — accepted):** named placeholders eliminate the class of bug. If GORM's `@name` + slice expansion for `IN` is unsupported, the fallback (named for scalars, positional for IN-lists) is documented.
- **Cross-tenant read in `ListWeeklyForWorkMonth` (Security MEDIUM-3 — flagged as follow-up):** the repo returns all partners' files; filtering happens in Go at `service.go:395-399`. This plan does not fix it (out of scope) but acknowledges it as a Phase D candidate.
