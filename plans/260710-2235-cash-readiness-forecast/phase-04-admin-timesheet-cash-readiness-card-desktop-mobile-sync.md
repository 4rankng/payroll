---
phase: 4
title: "Admin timesheet cash-readiness card (desktop + mobile sync)"
status: pending
priority: P2
dependencies: [3]
---

# Phase 4: Admin timesheet cash-readiness card (desktop + mobile sync)

## Overview

Surface the forecast as a single **"Chuẩn bị tiền trả"** card on `/admin/timesheet` (desktop) and the mobile admin timesheet page, sharing one component and one data hook. Shows confirmed-payable now, projected p50 with a [p50–p95] band, wallet balance, the **gap to prepare** (highlighted), and a "prepare by \<date\>" line — with a confidence badge.

## Requirements

- Functional: card renders `confirmed_payable`, `projected_p50`, `band [lower, upper]`, `wallet_available`, `gap` (primary figure), `prepare_by_date`, `next_pay_date`, and a confidence badge (high/medium/low).
- Functional: identical component, data, and title on desktop (`pages/admin/TimesheetPage`) and mobile (`pages/mobile/admin/TimesheetPage`).
- Non-functional: no gold/accent-gold tokens; VND formatting (no decimals); loading/error/empty/low-confidence states; respects the user's typography system (no oversized/inconsistent weights).

## Architecture

- One shared component `CashReadinessCard` (desktop + mobile import the same file).
- One hook `useCashReadiness(params)` (TanStack Query) calling `timesheetService.getCashReadiness`; query key includes filters + lead_days + horizon; invalidate alongside `useTimesheetSummary` so approve/transfer refreshes both.
- Service method `getCashReadiness` in `frontend/src/services/api/` (extend the timesheet service or a new `cash-readiness.service.ts`).
- Card placed in the "Cần xử lý" area of `/admin/timesheet` (next to the existing "Chờ thanh toán" stat), and the equivalent slot on mobile.

## Related Code Files

- Create: `frontend/src/components/timesheet/CashReadinessCard.tsx` — shared card.
- Create: `frontend/src/hooks/api/useCashReadiness.ts` (or extend `useTimesheets.ts`) — `useCashReadiness`.
- Create/modify: `frontend/src/services/api/*.ts` — `getCashReadiness`; add endpoint constant to `frontend/src/config/api.config.ts`.
- Create: `frontend/src/types/api/cash-readiness.types.ts` — response type.
- Modify: `frontend/src/pages/admin/TimesheetPage/index.tsx` — mount the card near the "Cần xử lý" group (lines ~128–167).
- Modify: `frontend/src/pages/mobile/admin/TimesheetPage/index.tsx` — mount the same card (mobile+desktop sync).
- Read-only reference: `frontend/src/components/wallet/WalletDemandCard.tsx` (CI/band styling patterns), `frontend/src/hooks/useTimesheetStatsConfig.ts` (filter plumbing).

## Implementation Steps

1. Add the response type + endpoint constant + service method `getCashReadiness`.
2. Implement `useCashReadiness` with a query key scoped to the page filters; share invalidation with `useTimesheetSummary`.
3. Build `CashReadinessCard`: primary figure = `gap` ("Cần chuẩn bị"); secondary row = confirmed now + projected p50; band rendered as a range `[lower–upper]`; wallet line; "Chuẩn bị trước <prepare_by_date>" (for next payment on <next_pay_date>). Confidence badge color-coded but **no gold** — use neutrals + one accent (e.g. emerald/amber/red for high/medium/low) consistent with existing tokens.
4. Handle states: loading skeleton; error (fall back to confirmed-only from the existing summary if the forecast 5xxs); empty (no data); low-confidence (show band + "độ tin cậy thấp" note, never p50 alone).
5. Mount on desktop `/admin/timesheet` in the pending/"Cần xử lý" area; mount the same component on mobile.
6. Verify typography matches the system (no oversized fonts — recall the 2026-07-10 typography overhaul); VND no decimals; tabular-nums for figures.
7. Tests: component renders all fields from a fixture; low-confidence branch; error→confirmed-only fallback; loading skeleton.

## Success Criteria

- [ ] Card shows on `/admin/timesheet` (desktop) and mobile with identical numbers + title.
- [ ] Primary figure is the **gap to prepare**; confirmed + projected + band + wallet all visible.
- [ ] For the 10-July case, confirmed-payable matches the existing "Chờ thanh toán" card exactly.
- [ ] Low-confidence path shows the band + note, never a bare p50.
- [ ] No gold tokens; VND no decimals; typography consistent with the system.
- [ ] Approve/transfer refreshes the card (shared invalidation with summary).
- [ ] `tsc --noEmit` + `npm run build` clean; component test green.

## Risk Assessment

- **False precision** → admin trusts a precise p50. *Mitigation:* band + confidence always shown; gap derived from p50 but band visible alongside.
- **Desktop/mobile drift** → inconsistent numbers. *Mitigation:* one shared component + one hook; tested on both.
- **Gold tokens slip in** → violates user preference. *Mitigation:* reuse existing neutral/accent tokens; review against `feedback_no_gold_design_tokens`.
