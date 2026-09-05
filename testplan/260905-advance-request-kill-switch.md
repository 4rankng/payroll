# Test Plan — Per-Employee Advance Request Kill-Switch (Tạm ngừng ứng lương)

Date: 2026-09-05. Feature: `project_employees.advance_request_enabled` — per-employee block on NEW advance payment requests (regular + self-check-in flows). Existing pending/approved requests untouched.

## 1. Backend unit (`go test ./internal/app/services/advance_payment/...`)

| # | Case | Expected |
|---|------|----------|
| U1 | active flexible assignment, enabled | hasFlexible=true, disabled=false |
| U2 | active flexible assignment, disabled | disabled=true |
| U3 | ended (LastDate set) flexible, disabled | disabled=false (active-only semantics) |
| U4 | non-flexible (weekly), disabled | disabled=false |
| U5 | two active flexible rows, one disabled | disabled=true, hasCheckIn preserved |
| U6 | check-in enabled regression | hasFlexible + hasCheckInEnabled both set |
| U7 | constants non-empty | MsgAdvanceRequestsDisabledVN / PausedTitle / PausedReason |

## 2. Integration (`make api-test`, live local backend; flow_advance_request_kill_switch.go)

| # | Case | Expected |
|---|------|----------|
| I1 | Admin GET /advance-payments/employees | row for test employee found; project id captured |
| I2 | Admin PATCH disable `advance_request_enabled:false` | 200 |
| I3 | Employee GET /me/advance-payment | canRequest=false, canRequestTitle="Tạm ngừng ứng lương", reason non-empty |
| I4 | Employee POST /me/advance-payment/request | 4xx, message contains "tạm ngừng" |
| I5 | Employee GET /me/check-in-advance | canRequestTitle="Tạm ngừng ứng lương" (gate precedes window/not-enabled branches) |
| I6 | Employee-token PATCH toggle | rejected (401/403) |
| I7 | Idempotent: repeat disable PATCH | 200 no-op |
| I8 | Admin PATCH re-enable → info title no longer paused; state restored for later flows | 200 |

## 3. Frontend

| # | Case | Expected |
|---|------|----------|
| F1 | `npx tsc -p tsconfig.app.json --noEmit` | no new errors vs baseline |
| F2 | `pnpm lint` | pass |
| F3 | `pnpm test` (AdvanceRequestToggle.test.tsx) | confirm dialog on disable; immediate fire on enable; stopPropagation |
| F4 | Manual: admin desktop 1280px → Ứng lương column toggle | disable→confirm→row shows "Đang tạm ngừng"; enable immediate |
| F5 | Manual: adv_partner desktop (AdvPartnerView) | same column visible/works |
| F6 | Manual: mobile 390px admin+adv_partner employee list | toggle row works, tap on row doesn't fire toggle |
| F7 | Employee portal (mobile): after disable | request button disabled + "Tạm ngừng ứng lương" notice (server-blocked variant); after re-enable restorable |

## 4. Prod rollout

| # | Case | Expected |
|---|------|----------|
| P1 | Apply 105 up.sql on prod via SSH (established raw-SQL recipe) | information_schema shows column, default 1 |
| P2 | `make deploy` (root Makefile; babysit to completion) | new backend+frontend images running, tag == HEAD |
| P3 | Smoke via admin UI | disable a test employee → employee sees notice → re-enable |

Rollback: 105 down.sql + previous image.

## Results

- **U1–U7 (unit):** PASS — `go test ./... -race` full backend suite exit 0 (2026-09-05).
- **Migration local:** applied to dev `payroll-mysql`; column verified `default=1`.
- **I1–I8 (integration):** PASS — `make api-test` Total 304 | Passed 281 | Failed 0 | Skipped 23 (first run had 1 FAIL = test-side swapped AssertContains args; fixed, behavior was correct).
- **F1 (tsc):** 0 errors in touched files (134 pre-existing backlog unchanged; older test files lack vitest global imports — not mine).
- **F2 (lint):** exit 0.
- **F3 (vitest component):** 5/5 pass (confirm-on-disable, cancel, immediate-enable, labels, stopPropagation).
- **P1–P3 (prod):** migration 105 applied pre-deploy (column verified); deployed 18:31 — image Created postdates commits; containers recreated.

## Post-deploy adversarial review round (`3c96640d`)

Findings verified in source, then fixed:
- **AssignEmployee Create→Save auto-pause** (GORM zero-value full-row Save) — CONFIRMED + fixed with explicit `AdvanceRequestEnabled: true` at all 9 assignment-creation sites (assign handler, employee import, FlexPay import, BCC weekly/multi/process imports); integration canary added (freshly re-added employee must start enabled) — PASS `employee 439 → project 58 (LGD), advance_request_enabled=true`.
- **Paused info returned zeroed quota** — fixed: both info endpoints populate real data; pause notice overrides at the end (employee sees frozen quota + "Tạm ngừng ứng lương", not zeros).
- **Stats dedup MIN/MAX cross-pairing** — fixed with COALESCE active-row preference; legacy fallback keeps ended-only employees listed.
- Double-fire on confirm dialog (pending guard), access-helper duplication (shared requireProjectModifyAccess), predicate→prefix invalidation, missing-project PATCH guard (shared renderer), EmployeeCard guard merge, unused `disabled` prop.
- Sibling-wave UI regressions (7831419d): '—' placeholders restored in TreasuryFeePanel, 0 ₫ amounts shown again in status overview, misleading progressbar aria removed, 44px wallet-sync target restored.
- Dismissed as stale/false: "missing 105 down.sql" (committed, verified in ls-tree); "deploy-order hazard" (migration applied pre-deploy); "global-block semantics" (deliberate strict-by-default on a money gate; user confirmed 1 employee = 1 project).
- Re-verification: go unit ok; api-test 304/281/0; vitest 52/52; tsc baseline 134 unchanged; lint 0.
