---
title: "Move wallet demand forecast (chart + card) from /admin/wallet to /admin/advance-payments"
description: "Relocate the WalletDemandChart ('Nhu cầu ứng lương còn lại theo kỳ') and WalletDemandCard ('Mức cần giữ trong ví / Đủ chi trả') block from /admin/wallet to /admin/advance-payments, on both desktop and mobile. The two components are presentational (data via props); only the pages that import + query them change. The wallet page keeps its hero balance, sync, transactions; the advance-payments page gains a new forecast section placed below the treasury hero and above the pipeline/metrics section. Both components stay importable from @/components/wallet/* (no rename) — this is a placement move, not a relocation of the component files. The /wallet/demand-forecast API and the shared query key are reused as-is; TanStack Query dedupes if both pages are ever mounted."
status: pending
priority: P2
branch: "main"
tags: [frontend, wallet, advance-payment, forecast, refactor, ui-relocation]
blockedBy: []
blocks: []
created: "2026-07-11T04:59:05.257Z"
createdBy: "ck:plan"
source: skill
---

# Move wallet demand forecast (chart + card) from /admin/wallet to /admin/advance-payments

## Overview

### What moves

The demand-forecast block currently rendered on `/admin/wallet` (desktop + mobile):

- **`WalletDemandChart`** — cohort line chart titled **"Nhu cầu ứng lương còn lại theo kỳ"** (`components/wallet/WalletDemandChart.tsx`). Subtitle "Theo số tiền yêu cầu sau phí, 3 kỳ gần nhất". One `<Line>` per period (current forecast + last 2 completed).
- **`WalletDemandCard`** — advisory card titled **"Mức cần giữ trong ví"** (`components/wallet/WalletDemandCard.tsx`). Shows recommended balance, current wallet balance or shortfall, and a "Đủ chi trả / Chưa đủ" badge.

Both are rendered together in a 2:1 grid (desktop) / stacked section (mobile).

### Destination

`/admin/advance-payments` (desktop `pages/admin/AdvancePaymentsPage/index.tsx`, mobile `pages/mobile/admin/AdvancePaymentsPage/index.tsx`). New section placed **below the Treasury Hero** ("Tổng quan kỳ ứng lương" — Wallet | Flow | Fee) and **above the Pipeline + operating-health section** ("Trạng thái xử lý ứng lương").

### Why this is conceptually right

The chart and card are advisory *advance-payment demand* projections — they tell the admin "how much advance-pay cash you'll need per cycle." That belongs on the advance-payments page where the admin manages those requests, not on the wallet page whose job is balance / sync / reconciliation. The user confirmed moving **both** as one block (answer to move-scope question).

### Why this is low-risk (with caveats surfaced by red-team)

- The two components are **purely presentational** — they take `data?: WalletDemandForecastResponse` / `isLoading?: boolean` via props and render. No data fetching inside.
- The only data dependency is `GET /wallet/demand-forecast`, served by `walletService.getDemandForecast()` + `QueryKeys.wallet.demandForecast()`. Same call reused on the new page. (Note: `/admin/wallet` and `/admin/advance-payments` are sibling React Router routes under `<AdminLayout>` — only one mounts at a time, so the TanStack "dedupe" property is not load-bearing here. What actually happens: navigation unmounts one page's `useQuery` and remounts the other's; the `['wallet','demand-forecast']` cache entry survives and is reused. Because the key is a child of `['wallet']`, sync/adjust on the wallet page correctly invalidates the forecast.)
- Only **4 files** change (2 page removes + 2 page additions). The component files in `components/wallet/` are **not moved or renamed** — keeping the import path `@/components/wallet/WalletDemandChart` avoids churn in the graph, AGENTS.md docs, and any external references. (A rename is a YAGNI complication for now; documented as a non-goal.)
- No backend, route, query-key, or type changes.

### Security considerations (added post red-team)

The destination page `/admin/advance-payments` is, on mobile, served to **both** admin and `adv_partner` roles via the shared `AdvancePaymentsPageMobile` component (`App.tsx:292`). The demand-forecast payload is treasury-sensitive: `WalletDemandForecastResponse` includes `recommended_balance`, `current_available` wallet balance, `shortfall`, and newsvendor quantiles (`wallet.types.ts:151-184`). Three layers defend it:

1. **Client render guard** — `{!isAdvPartner && ...}` around the JSX (this plan).
2. **Client query gate** — `enabled: !isAdvPartner` on the `useQuery` (red-team fix; without it the request fires and 403s every 5 min for every adv_partner session).
3. **Backend deny-by-default** — `backend/configs/casbin_policy.csv` has **no** `wallet` rule for `adv_partner`; the `keyMatch2` matcher (`casbin_model.conf:14`) returns 403 unless an allow rule matches (`authorization.go:85`). This is correct today but incidental — a future catch-all `adv_partner` allow rule could regress it. Defense-in-depth via layers 1+2 is the durable control.

`WalletDemandCard` surfaces the admin's actual wallet balance via `prediction.current_available` (sourced from `walletSvc.GetBalance` on the backend). The `isAdvPartner` guard MUST cover the network request (layer 2), not just the render — otherwise the balance leaks via the 403-rejected response path or a future refactor.

## Decision (locked)

| Question | Decision | Rationale |
|----------|----------|-----------|
| What moves | Chart **+** Card together | User answer; they read as one block |
| Component file location | Stay in `components/wallet/` | Presentational components; moving the dir is churn with zero behavioral benefit (YAGNI) |
| Placement on advance-payments page | New section between Treasury Hero and Pipeline | User answer ("Below Treasury Hero, above Pipeline") |
| Query strategy on new page | Reuse `walletService.getDemandForecast()` + `QueryKeys.wallet.demandForecast()` with the same `staleTime` / `refetchInterval` as the wallet page (5 min) | Keeps a single source of truth; TanStack dedupes |
| `isAdvPartner` gating | Show the section only for **admin** (not `adv_partner`) | Matches existing parity: `WalletBalanceCard` is already admin-only on this page; adv_partner has no wallet context |

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Move forecast on desktop](./phase-01-move-forecast-on-desktop.md) | Pending |
| 2 | [Move forecast on mobile](./phase-02-move-forecast-on-mobile.md) | Pending |
| 3 | [Verify](./phase-03-verify.md) | Pending |
| 4 | [Graph update & docs](./phase-04-graph-update-docs.md) | Pending |

## Dependencies

None. No relation to the open `260710-2235-cash-readiness-forecast` or `260711-0930-trend-aware-cash-forecast` plans — those touch the **timesheet** cash-readiness forecast on `/admin/timesheet`, a different feature surface backed by a different cohort/service. This plan only relocates the already-shipped **advance-payment** wallet-demand components between two existing admin pages.

## Validation Log

### Verification Results
- **Tier:** Standard (4 phases)
- **Claims checked:** 8
- **Verified:** 8 | **Failed:** 0 | **Unverified:** 0

#### Verified claims (sample)
1. [Contract Verifier] `WalletDemandChart` / `WalletDemandCard` consumers — exactly 2: `src/pages/admin/WalletPage/index.tsx`, `src/pages/mobile/admin/WalletPage/index.tsx`. VERIFIED (grep, no others).
2. [Fact Checker] Desktop `AdvancePaymentsPage` does NOT import `useQuery`, `QueryKeys`, or `walletService` — confirmed; all must be added. VERIFIED.
3. [Fact Checker] Mobile `AdvancePaymentsPage` does NOT import `useQuery`, `QueryKeys`, or `walletService` — confirmed. VERIFIED.
4. [Fact Checker] `QueryKeys` on both wallet pages is used ONLY by the demand-forecast query → safe to remove the import. VERIFIED (desktop line 164, mobile line 47).
5. [Contract Verifier] `walletService` on both wallet pages is used by `getBalance` / `syncBalance` / `adjustBalance` (in addition to `getDemandForecast`) → import MUST stay. VERIFIED (desktop lines 160, 180, 199).
6. [Fact Checker] `isAdvPartner` exists in both `AdvancePaymentsPage` (desktop L95) and `AdvancePaymentsPageMobile` (mobile L103). VERIFIED.
7. [Flow Tracer] Route structure: `/admin/advance-payments` → `AdvancePaymentsPage` (desktop) / `AdvancePaymentsPageMobile` (mobile); `/adv-partner/advance-payments` → `AdvPartnerView` (desktop) / **same** `AdvancePaymentsPageMobile` (mobile). This confirms the mobile `{!isAdvPartner}` guard is load-bearing (the mobile component is shared between roles). VERIFIED (`App.tsx` L258, L291-292).
8. [Fact Checker] Desktop `AdvPartnerView.tsx` does NOT reference the forecast or `WalletBalanceCard` → no change needed there. VERIFIED.

### Whole-Plan Consistency Sweep
- Files reread: plan.md, phase-01…phase-04
- Decision deltas checked: 0 (no changes this session — verification only)
- Reconciled stale references: 0
- Unresolved contradictions: 0

**Outcome:** plan is eligible for implementation. No interview questions — all decision points (move scope, placement) were resolved during planning via `AskUserQuestion`, and verification surfaced no failures.

## Red Team Review

### Session — 2026-07-11
**Reviewers:** Security Adversary, Failure Mode Analyst, Assumption Destroyer (Standard tier — Fact Checker + Contract Verifier)
**Findings:** 12 unique (after dedup of ~24 raw findings across 3 reviewers)
**Accepted:** 12 | **Rejected:** 0
**Severity breakdown:** 0 Critical, 4 High, 7 Medium, 1 Low

The three reviewers independently converged on one **High** finding: the proposed unconditional `useQuery` fires `GET /wallet/demand-forecast` for `adv_partner` users because the mobile `AdvancePaymentsPageMobile` component is shared between admin and adv_partner routes (`App.tsx:292`). The `{!isAdvPartner}` JSX guard does not cover the hook. Fix: `enabled: !isAdvPartner`.

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | Unconditional `useQuery` → 403 storm for adv_partner (shared mobile route) | High | Accept | phase-01, phase-02 (added `enabled: !isAdvPartner`) |
| 2 | `lg:grid-cols-3` "convention" claim fabricated — destination uses bespoke tracks | High | Accept | phase-01 (switched to `md:grid-cols-3`) |
| 3 | Desktop `AdvancePaymentsPage` is a responsive hybrid (`useIsMobile`) — section must work at all widths | Medium | Accept | phase-01 note, phase-03 width checks |
| 4 | "TanStack dedupes if both pages mounted" justification is vacuous (routes are mutually exclusive) | Medium | Accept | plan.md "Why low-risk" rewritten |
| 5 | Phase 3 "no double-fetch sanity" check is unfalsifiable | Medium | Accept | phase-03 replaced with adv_partner network-silence check |
| 6 | Mobile forecast surfaces diverge in styling from surrounding compact tiles | Medium | Accept | phase-02 documented as deliberate |
| 7 | Backend deny-by-default via casbin absence is correct but undocumented | Medium | Accept | plan.md "Security considerations" added |
| 8 | `WalletDemandCard` exposes admin wallet balance — guard must cover network, not just render | Medium | Accept | plan.md "Security considerations" |
| 9 | sessionStorage persistence restores stale forecast (existing behavior) | Low | Accept | noted; out of scope to change |
| 10 | Single-commit across 4 surfaces hurts bisect | Medium | Accept | phase-04 split into 2 commits |
| 11 | Verification log said "10 claims" but listed 8 | Low | Accept | plan.md count corrected to 8 |
| 12 | Mobile early-return skeleton doesn't preview forecast slot | Medium | Accept | phase-02 documented; accepted layout shift |

### Whole-Plan Consistency Sweep (post red-team)
- Files reread: plan.md, phase-01…phase-04
- Decision deltas checked: 6 (`enabled` gate, `md:` breakpoint, dedupe rewrite, security section, 2-commit split, skeleton note)
- Reconciled stale references: 3 (removed `lg:grid-cols-3` from phase-01 success-criteria context; removed "Single commit" from phase-04; removed dedupe claim from plan.md)
- Unresolved contradictions: 0

**Outcome:** plan is ready for implementation. The single material code change from red-team is the `enabled: !isAdvPartner` gate; everything else is documentation/verification strengthening.

## Out of scope (non-goals)

- Renaming `WalletDemandChart` / `WalletDemandCard` or relocating their files under `components/advance-payment/`. The `wallet/` import path stays.
- Any backend, route, or type change.
- The `260710-2235-cash-readiness-forecast` (timesheet) work.
- Redesigning the chart or card visuals.

## Acceptance criteria

- [ ] `/admin/wallet` (desktop + mobile) no longer renders the demand chart or card; the page still shows hero balance, sync, mismatch dialog, and transactions exactly as before.
- [ ] `/admin/advance-payments` (desktop + mobile) renders the chart + card in a new section below the Treasury Hero and above the Pipeline section, **admin only** (hidden for `adv_partner`).
- [ ] The forecast query is fetched once via `QueryKeys.wallet.demandForecast()` with 5-min `staleTime` / `refetchInterval`.
- [ ] `pnpm type-check` passes; `pnpm lint` passes.
- [ ] No orphaned imports in either wallet page.
- [ ] `graphify-out/` regenerated (`graphify update .`) so import edges reflect the new consumers.
