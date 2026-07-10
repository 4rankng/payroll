---
phase: 3
title: "HTTP endpoint + DI wiring + caching"
status: pending
priority: P2
dependencies: [2]
---

# Phase 3: HTTP endpoint + DI wiring + caching

## Overview

Expose `CashReadinessForecastService` at `GET /api/v1/timesheets/cash-readiness`, wire it in DI, add a short-TTL cache with invalidation on approve/transfer, and gate it with Casbin. Advisory read-only — no money-path surface.

## Requirements

- Functional: `GET /timesheets/cash-readiness?lead_days=&horizon=&project_id=&employee_id=&fromDate=&toDate=` returns `{confirmed_payable, projected_p50, band_lower, band_upper, wallet_available, gap, prepare_by_date, next_pay_date, method, confidence, basis_cycles, generated_at}`.
- Functional: role-scoped — admin/manager/accountant see all (within their data scope); partner sees only own employees (mirror `GetSummary` partner logic at `timesheet_summary_handler.go:27-35`).
- Non-functional: cached ~15s with event-driven invalidation; advisory-only; parses dates with `time.ParseInLocation` (copy the summary handler pattern).

## Architecture

- Handler method on the existing timesheet `Handler`: `GetCashReadiness`. Parses query params identically to `GetSummary` (same `loc=Local` date parsing, same role scoping), calls `cashReadinessSvc.GetCashReadiness`, maps to a response DTO.
- Service wired into the dashboard `Service` struct (`dashboard/service.go`) or the timesheet service — wherever `GetSummaryStats` lives, so they share filter semantics. Add `CashReadinessSvc` field + constructor arg.
- DI: instantiate in `backend/internal/app/bootstrap/services/init.go`, register in `container.go`, inject into the timesheet handler.
- Cache: key `cash-readiness:{role}:{userID}:{filterHash}:{leadDays}:{horizon}`, TTL 15s (mirror the timesheet-summary cache TTL from obs 44465). Invalidate on bulk-approve and bulk-transfer events (the events that change `PendingPaymentAmount`) — recall the stale-summary investigation (obs 44466–44469) where invalidation pattern mismatch hid stale data; reuse the *correct* invalidation path the summary now uses.
- Casbin: add a read policy for the endpoint — admin/manager/accountant `GET /timesheets/cash-readiness`; partner `GET` scoped (same object/permission as summary).

## Related Code Files

- Create: `backend/internal/transport/http/handlers/timesheet/cash_readiness_handler.go` — `GetCashReadiness`.
- Create: `backend/internal/app/dto/cash_readiness.go` — response DTO (+ JSON tags).
- Modify: `backend/internal/app/services/dashboard/service.go` — add `CashReadinessSvc` (or add to the timesheet service that owns `GetSummaryStats`).
- Modify: `backend/internal/app/bootstrap/services/init.go`, `container.go` — instantiate + inject.
- Modify: timesheet routes file (where `GET /summary` is registered) — register `GET /cash-readiness`.
- Modify: `backend/internal/casbin/` policy definitions — add read permission.
- Modify: cache invalidation — extend the existing summary-invalidation hook to also purge `cash-readiness:*` on approve/transfer events.
- Read-only reference: `timesheet_summary_handler.go` (copy date-parse + role-scope + response patterns).

## Implementation Steps

1. Define the response DTO in `dto/cash_readiness.go` (snake_case JSON; include `generated_at` RFC3339).
2. Add `GetCashReadiness` handler: parse params with `time.ParseInLocation("2006-01-02", ..., loc)`; apply partner scope; call service; map to DTO; `response.Success`.
3. Register the route next to `/summary`; add Casbin read policy for admin/manager/accountant + scoped partner.
4. Wire the service in DI (`init.go` + `container.go`); inject into the timesheet handler constructor.
5. Add caching: wrap the service call (or handler) with the same cache service the summary uses; key includes role/user/filters/lead_days/horizon; TTL 15s.
6. Hook invalidation: on bulk-approve / bulk-transfer (and any event that mutates `PendingPaymentAmount`), delete `cash-readiness:*` for the affected scope — mirror the corrected summary invalidation.
7. Handler tests: admin returns full scope; partner scoped; date params parsed in local tz (no boundary drop); cache hit returns fast; response shape contract.

## Success Criteria

- [ ] `GET /api/v1/timesheets/cash-readiness` returns the documented payload for admin; partner sees only own employees.
- [ ] Date params parsed with `loc=Local` (no boundary-day drop — regression-covered).
- [ ] Cache serves repeat reads within TTL; invalidation fires on approve/transfer (verified by test: mutate → next read is fresh).
- [ ] Casbin denies unauthorized roles; allows admin/manager/accountant + scoped partner.
- [ ] Endpoint is read-only — no mutation/disbursement handler is reachable through it.
- [ ] `go test ./internal/transport/http/handlers/timesheet/...` green; `golangci-lint` clean.

## Risk Assessment

- **Stale cache after a bulk-approve** → admin under-prepares. *Mitigation:* event-driven invalidation on every `PendingPaymentAmount`-mutating event + 15s TTL backstop; tested.
- **Partner scope leak** → partner sees other employees' payroll. *Mitigation:* reuse the exact `EmployeeCreatedBy` scoping from `GetSummary`; test the partner path.
- **Casbin over-broad/deny-overrides** → 403 for legit roles (recall the partner-403 lessons). *Mitigation:* mirror the summary endpoint's policy; verify with a per-role matrix test.
