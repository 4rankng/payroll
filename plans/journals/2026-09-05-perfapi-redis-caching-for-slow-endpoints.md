---
title: "perf(api): redis caching for slow endpoints"
date: 2026-09-05
summary: Cached 4 dashboard analytics + strengthened /projects + grouped timesheets microcache; fixed Search cache-key collision and BulkTransfer invalidation gap
---

# perf(api): redis caching for slow endpoints

## What happened
7-day latency report showed 10 slow endpoints (P95 0.9–3.6s). Scout found the
Redis cache infra + event-driven CacheInvalidationHandler existed, but the four
slowest dashboard analytics endpoints were uncached, /api/v1/projects (2,440
calls) had a 15s microcache that missed most requests plus a count-cache key
that no invalidation pattern ever matched, and /api/v1/timesheets/grouped
(1,101 calls) had no cache at all.

Changes (backend/internal/):
- dashboard/bank_usage.go, project_profitability.go, service.go: 60s cache on
  GetBankUsageAllProjects / GetProjectProfitability / GetProjectWeeklyProfit
  (keyed by days) / GetMonthlyFinancials (keyed by period) — dashboard:* keys,
  cleared by existing event patterns.
- constants/cache.go: ProjectListCacheTTL 15s→60s (project + project-employee
  events already invalidate projects:list*).
- infra/events/cache_invalidation_handler.go: added projects:count* to
  Project/ProjectEmployee cases; added BulkTransfer case (before Transaction —
  BulkTransferTransactionCreated matches both substrings) clearing
  dashboard/timesheets/transactions patterns.
- app/services/timesheet/service.go: 15s microcache for ListGroupedByEmployee
  under timesheets:list:grouped:{filters}.

## Decision
Reuse the existing Get→miss→compute→Set pattern and event-bus invalidation
(ADR-007: events publish post-commit) instead of inventing new invalidation
code paths. Declined caching for /timesheets/export (5 calls/7d) and
/timesheets/cash-readiness (10 calls/day advisory; wallet-balance dependency
has no event invalidation and cache would not help first load — forecast
optimization is the real fix, follow-up).

## Review findings fixed
- BLOCKER: generateSummaryStatsCacheKey omitted filters.Search while
  grouped/list/summary queries honor it → different searches shared one cache
  key within TTL (also a live bug in the pre-existing ListTimesheets
  microcache). Added Search branch; live-verified nguy→20 groups,
  garbage-search→0 groups back-to-back.
- Fixed pre-existing nightly-flaky test
  TestProjectEmployeeRepository_HasActiveFlexiblePaymentScheduleByEmployeeID:
  fixture used local time.Now()-1d against StartOfDay(clock.NowUTC()) —
  SQLite string compare fails 00:00–07:00 +07. Pushed fixture dates 1 more
  day back.

## Verification
go build and go vet clean; unit tests green (dashboard, timesheet, events,
persistence); api-test 304/281/0/23-skipped twice; smoke miss→hit: bank-usage
177ms→3.6ms, profitability 247ms→3.1ms, grouped 226ms→81ms; payloads
byte-identical.

## Next steps
- OPEN: Casbin grants partner GET on /api/v1/dashboard/* (all-projects
  financials) — pre-existing; confirm intended.
- Follow-up: cash-readiness first-load latency (forecast and MC optimization).
- Deploy to prod and re-check the 7-day latency report after a week.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
