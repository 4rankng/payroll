---
phase: 3
title: "Hybrid tail parity backend"
status: pending
priority: P1
effort: "3h"
dependencies: []
---

# Phase 3: Hybrid tail parity backend

## Overview

Close the permissiveness hole in the regular FlexPay `CreateRequest`: checkin-enabled (hybrid) employees get the same window rule as everyone else — previous month requestable only during days 1–8 (`RequestCutoffDay`), instead of the current wholesale skip that lets them request July on any day of August.

## Requirements

- Functional: for `hasCheckInEnabled` employees, `CreateRequest` enforces `isRequestMonthAllowed(now, forMonth)` — days 1–8 → previous month only; day 9+ → current month (which the checkin flow handles anyway; the regular endpoint serves the tail).
- Non-functional: no change for non-checkin flexible employees (exact same rule now applies to both); error messages unchanged (reuse cutoff constants).

## Architecture

One-line semantic change at `service.go:245`:

```go
// before
if !eligibility.hasCheckInEnabled && !isRequestMonthAllowed(now, forMonth) {
// after
if !isRequestMonthAllowed(now, forMonth) {
```

The `hasCheckInEnabled` carve-out existed so checkin employees weren't blocked by upload-driven windows; but `isRequestMonthAllowed` only gates WHICH month, and the quota lookup (`GetByEmployeeAndMonth`) + budget check already handle per-month correctness. Removing the carve-out aligns hybrids with the documented tail.

## Related Code Files

- Modify: `backend/internal/app/services/advance_payment/service.go` — `CreateRequest` (~245); update the comment block above it
- Reference: `backend/internal/app/services/advance_payment/month_resolver.go` — `isRequestMonthAllowed` deps (`IsBeforeCutoff`, `RequestCutoffDay = 8`)
- Test: `backend/internal/app/services/advance_payment/` — new/extended service test file (see Phase 5)

## Implementation Steps

1. Remove the `!eligibility.hasCheckInEnabled &&` guard at `service.go:245`.
2. Re-read `CreateRequest` body: the `len(advPayments) == 0` fallback at ~255 also has a `!eligibility.hasCheckInEnabled` carve-out (`IsInLockedGap` message) — keep that one (it only picks a better error message for missing quota, no window semantics); verify with a test.
3. Update the window-rule comment to state hybrids follow the same tail.
4. Check `GetEmployeeAdvanceInfo` (~90-119): for checkin-enabled employees it lists ALL quota months (via `GetMonthsByEmployee`) — that stays (display is fine; the gate is at request time). Optional: none. HOLD scope.
5. `go build ./... && go vet ./internal/app/services/advance_payment/...`.

## Success Criteria

- [x] Hybrid requests July on Aug 5 → accepted (quota from July rows)
- [x] Hybrid requests July on Aug 9 and Aug 31 → rejected with cutoff message
- [x] Hybrid requests August on Aug 5 via regular endpoint → rejected (tail = prev month only)
- [x] Non-checkin flexible employee behavior byte-identical (regression test)
- [x] Unit tests cover the matrix: {day 1–8, day 9–19, day 20–31} × {prev, current} × {hybrid, non-hybrid}

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Over-tightening: some legit flow relied on the skip | Scout found none — the skip's purpose was upload-window relief, and `isRequestMonthAllowed` never blocks for lack of uploads (quota lookup handles that). The `len==0` fallback carve-out stays untouched. |
| Deploys before Phase 4 → hybrids blocked in UI earlier | Deploy Phases 3+4 together (noted in plan.md deployment notes) |
| `GetEmployeeAdvanceInfo` shows months the employee can no longer request from | Display-only (quota facts, not requestability); the form gates by `canRequest` + selected month; acceptable within HOLD scope |
