---
phase: 5
title: "Tests + timezone safety + accuracy instrumentation"
status: pending
priority: P2
dependencies: [1, 2, 3, 4]
---

# Phase 5: Tests + timezone safety + accuracy instrumentation

## Overview

Close the loop: the unit/integration tests (woven through Phases 1–4) plus the one thing that makes the statistical-vs-ML decision data-driven — **accuracy instrumentation** that records p50-vs-actual each pay cycle, and a backtest comparing the model to recent actuals. This is what tells us whether to ever escalate to ML.

## Requirements

- Functional: every public path covered — repo cohort (incl. tz regression), service math + advisory invariant, provider determinism, handler role/cache, frontend card branches.
- Functional: after each pay cycle, persist/log `{cycle, confirmed_payable_at_forecast, projected_p50, projected_p95, actual_total_paid, abs_pct_error_p50, band_captured_actual}`.
- Functional: a backtest harness that replays the last N cycles and reports mean/median abs % error + band-coverage rate.
- Non-functional: no new infra for logging (use `slog` + existing audit/log path); backtest is a CLI/script (`qa/scripts/`), not a hot path.

## Architecture

- **Accuracy record:** a lightweight, idempotent write fired once per pay cycle (e.g., on the bulk-transfer event) capturing the forecast that was *shown* for that cycle vs. the *actual* transferred total. Store either as a dedicated log line (structured `slog`) or reuse an existing audit table — prefer no new migration; if a table is needed, flag it.
- **Backtest:** `qa/scripts/cash-readiness-backtest.{sh,go}` that calls the repo for the last N cycles, runs the provider, and prints error stats. Mirrors the existing `qa/scripts/orphan-timesheet-check.sh` pattern (a money-loss detector).

## Related Code Files

- Add/extend tests (per earlier phases): `timesheet_analytics_repository_test.go`, `cash_readiness_forecast_test.go`, `cash_readiness_handler_test.go`, `CashReadinessCard.test.tsx`.
- Create: instrumentation hook — emit the accuracy record on the bulk-transfer event (in the bulk-transfer service, alongside existing audit emission; non-blocking goroutine, mirrors the audit pattern in `backend/CLAUDE.md`).
- Create: `qa/scripts/cash-readiness-backtest.go` (or `.sh`) — offline accuracy report.
- Optional docs: append a short section to `docs/flows/` or a runbook on how to read the accuracy log + when to consider ML.

## Implementation Steps

1. Audit test coverage from Phases 1–4; fill gaps: cohort tz regression, confirmed=sum exactness, band monotonicity, no-history fallback, invariant spy, determinism, handler role matrix, cache invalidation, frontend branches.
2. Implement the accuracy record: on bulk-transfer completion, write `{cycle, ts, confirmed_at_forecast, projected_p50, projected_p95, actual_total}`. Idempotent on cycle (re-running overwrites the same cycle row / dedups).
3. Implement the backtest CLI: fetch last N cycles, run the provider at each cycle's forecast point, compute abs % error vs actual + band-coverage; print a table.
4. Run the backtest on demo (has real history) to get a baseline error figure; record it in the plan/runbook.
5. Lint/build gates: `golangci-lint run ./...`, `go test ./...` (relevant packages), frontend `tsc --noEmit` + `npm run build`.
6. Decision hook: document the escalation threshold — *if median abs % error > ~12% over ≥4 cycles AND seasonal swings are evident, evaluate a Prophet/LightGBM provider behind `ForecastProvider`.* Otherwise keep statistical.

## Success Criteria

- [ ] All Phase 1–4 unit/integration tests green; `golangci-lint` clean; frontend build clean.
- [ ] Timezone regression test proves no boundary-day drop under `loc=Local`.
- [ ] Advisory-invariant spy test proves zero `SyncBalance`/`CreateTopup` calls across all service paths.
- [ ] Accuracy record writes once per pay cycle (idempotent); backtest CLI runs and prints error stats.
- [ ] Baseline accuracy number captured from demo; escalation threshold documented.
- [ ] No new migration merged unless explicitly approved (instrumentation uses existing log/audit path).

## Risk Assessment

- **Instrumentation itself mutates state / costs money** → *Mitigation:* read-only + log-only; non-blocking; no disbursement/top-up handle; idempotent.
- **Backtest overfits to a short window** → misleading "ML is better" signal. *Mitigation:* require ≥4 cycles and visible seasonality before escalating; report band-coverage, not just point error.
- **Accuracy log grows unbounded** → *Mitigation:* one row per cycle (cycle-keyed upsert) or rotated structured logs; negligible volume either way.
