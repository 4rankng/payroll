---
phase: 1
title: "Timesheet cash-flow cohort + confirmed-payable query"
status: pending
priority: P2
dependencies: []
---

# Phase 1: Timesheet cash-flow cohort + confirmed-payable query

## Overview

Add the one data source that does not exist yet: a **timesheet-accrual cohort** — cumulative approved-pay sampled per cycle-day across N historical pay cycles — to feed the reused Monte-Carlo engine. Confirmed-payable is *not* rebuilt here; it already lives in `GetSummaryStats.PendingPaymentAmount` and is reused as-is in Phase 2.

## Requirements

- Functional: a repository method returning, for each of the last N pay cycles, the cumulative approved timesheet pay at each cycle-day (the shape the MC engine consumes), scoped by the same role/filter semantics as the summary endpoint.
- Non-functional: aggregation done in SQL (`GROUP BY`), timezone-safe day bounds in `time.Local`, no table scan (verify with `EXPLAIN`).

## Architecture

The wallet forecast pivots advance-payment cohorts via `pivotCohort` (`wallet_demand_forecast.go`) into the unexported `cohortSeries` type. We do **not** touch that pipeline. Instead we add a parallel **timesheet** cohort builder in Phase 2 that emits the same `cohortSeries` shape from new repo rows defined here.

<!-- Updated: Validation Session 1 - resolved 4-cycle model + NEW clock helper -->

Cohort definition for timesheets (resolved — global 4-cycle monthly schedule):
- A **cycle (Kỳ)** = one of the four fixed pay periods: Kỳ1 work 1–7/pay 10, Kỳ2 work 8–14/pay 17, Kỳ3 work 15–21/pay 24, Kỳ4 work 22–28/pay day-1-next-month. Same for all employees/projects.
- **cycle-day** = days since the cycle's work-start (calendar day 1, 8, 15, or 22 = cycle-day 1). Sampling runs from cycle-day 1 through the pay date (approval keeps trickling in during the ~3-day work-end→pay gap).
- For each historical cycle, sample **cumulative approved pay** (sum of `amount` for timesheets whose work `date` falls in that cycle's work-days AND `approved_at ≤ that cycle-day`, excluding `payment_status=paid`) at each cycle-day. This is the timesheet analogue of the advance-payment `cumulative[]`.
- **Basis is strong:** ~4 cycles/month → a 6-month lookback ≈ 24 cycle observations, partitioned by Kỳ (1–4) so a Kỳ-1 forecast uses historical Kỳ-1 curves (seasonality by cycle position, not just month).

Confirmed-payable (deterministic, as-of-now): reuse `GetSummaryStats(filters).PendingPaymentAmount` — **do not recompute**. Phase 1 only verifies the status-filter semantics (approved + unpaid + not-transferred) match what the cohort excludes, to prevent double-counting.

## Related Code Files

- Create: `backend/internal/pkg/clock/pay_cycle.go` — the 4-cycle (10/17/24/1) pay-cycle helper (NEW; no existing timesheet cycle helper). Sibling of `advance_payment.go`.
- Modify: `backend/internal/infra/persistence/repositories/timesheet_analytics_repository.go` — add `GetAccrualCohort(ctx, GetAccrualCohortQuery) ([]TimesheetAccrualCycleRow, error)`.
- Modify: `backend/internal/infra/persistence/repositories/timesheet_analytics_repository_test.go` — cohort + timezone-bound regression tests.
- Read-only reference: `backend/internal/transport/http/handlers/timesheet/timesheet_summary_handler.go:52-69` (copy the `time.ParseInLocation` + `loc=Local` pattern verbatim), `backend/internal/app/services/wallet_demand_forecast.go::pivotCohort` (shape to match), `backend/internal/app/services/wallet_demand_forecast_math.go::cohortSeries` (target type).
- Possibly add: one index in `migrations/0NN_*.{up,down}.sql` **only if** `EXPLAIN` shows a scan on the approval-timestamp / status filter. Flag it; do not assume.

## Implementation Steps

1. **Build the pay-cycle clock helper (NEW — resolved Session 1).** Before writing it, grep `listTimesheetsForCycle` for any pre-existing cycle definition to align with. Then add `backend/internal/pkg/clock/pay_cycle.go`: the 4-cycle schedule (pay days 10/17/24/1; work windows 1–7 / 8–14 / 15–21 / 22–28), with `CurrentKy(t)`, `NextPayDate(t)`, `CycleStartDate(t)`, `CycleDay(t)`. This is the timesheet sibling of `advance_payment.go` (which models the *different* advance cycle day 20→9) — do **not** reuse that file's constants. Lead time = 2 (`CASH_FORECAST_LEAD_DAYS`); lookback = 6 months (≈24 cycle observations), partitioned by Kỳ 1–4.
2. Define `TimesheetAccrualCycleRow{ Ky int; ForMonth string; CycleDay int; CumulativeAmount int64; GrandTotal int64; MaxCycleDay int }` (exported; lives next to the repo or in `domain/timesheet`).
3. Write the SQL: per cycle, cumulative sum of approved (not-paid) `amount` ordered by approval date, sampled at each cycle-day. Use `GROUP BY` for the aggregation; build day bounds with `time.Local` (DSN is `loc=Local`). Mirror the cohort SQL's timezone conversion already used elsewhere (obs 42634).
4. Apply role/filter scoping identical to `GetSummary` (partner → `EmployeeCreatedBy`; optional `project_id`/`employee_id`/date range).
5. Run `EXPLAIN`; only add an index if a scan appears. Prefer the existing indexes.
6. Add repo tests: known fixture → expected cumulative curve; **timezone regression** asserting no boundary-day drop (the 65.7M bug class); empty-history → empty slice, no error.

## Success Criteria

- [ ] `GetAccrualCohort` returns N cycles of cumulative approved-pay curves for the admin scope.
- [ ] Confirmed-payable still comes solely from `GetSummaryStats` — cohort query does not duplicate it.
- [ ] Cohort excludes paid/transferred timesheets (no overlap with confirmed-payable status set) — verified by test.
- [ ] No boundary-day drop under `loc=Local` (timezone regression test passes).
- [ ] `EXPLAIN` shows index usage (or a documented, indexed plan); no full scan on the timesheet table.
- [ ] `go test ./internal/infra/persistence/repositories/...` green; `golangci-lint` clean.

## Risk Assessment

- **Kỳ 4 crosses the month boundary** (work 22–28 of month M, paid day 1 of M+1) → month-boundary cycle-day math is the real correctness risk. *Mitigation:* `pay_cycle.go` computes Kỳ from calendar date, not month string; cohort rows carry `{Ky, ForMonth}` so Kỳ-4 rows aren't mis-bucketed into the wrong month. Unit-test the day-1-next-month boundary explicitly.
- **Wrong status filter overlaps confirmed-payable** → double-count. *Mitigation:* derive both from the same status semantics; cross-check test asserting `sum(cohort latest point) ≈ PendingPaymentAmount` within expected variance.
- **Expensive query** → slow admin page. *Mitigation:* `GROUP BY` aggregation, indexed plan, and Phase 3 caching.
