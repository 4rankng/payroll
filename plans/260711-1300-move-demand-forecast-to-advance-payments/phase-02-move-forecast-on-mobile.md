---
phase: 2
title: "Move forecast on mobile"
status: pending
priority: P2
dependencies: [1]
---

# Phase 2: Move forecast on mobile

## Overview

Mirror Phase 1 for the **mobile** variants: remove the demand-forecast section from the mobile `/admin/wallet` page and add it to the mobile `/admin/advance-payments` page, in a stacked section consistent with the mobile page's `MobileSurface` / `space-y-3` rhythm. Admin-only.

## Requirements

- Functional: mobile `/admin/advance-payments` shows the chart then the card (stacked); mobile `/admin/wallet` no longer does.
- Non-functional: same query reuse; mobile tap targets / safe-area insets respected; matches the existing mobile advance-payments page section styling.

## Architecture

The mobile wallet page currently stacks chart + card inside a `<section aria-label="Dự báo nhu cầu ví" className="space-y-3">` (lines 153–156). On the mobile advance-payments page, the natural slot is inside the `<div className="grid gap-3">` block that already holds `WalletBalanceCard`, `AdvPartnerHeroStrip`, and `TreasuryFeePanel` (lines 266–290) — placing the forecast right after the treasury hero group keeps it in the "overview before operations" zone.

```
 pages/mobile/admin/WalletPage/index.tsx
   - drop imports, forecast useQuery, the <section>

 pages/mobile/admin/AdvancePaymentsPage/index.tsx
   + add useQuery for forecast
   + add imports
   + render <WalletDemandChart/> then <WalletDemandCard/> after TreasuryFeePanel, inside the gap-3 grid, admin-only
```

## Related Code Files

- Modify: `frontend/src/pages/mobile/admin/WalletPage/index.tsx`
- Modify: `frontend/src/pages/mobile/admin/AdvancePaymentsPage/index.tsx`
- Reference (no change): the two component files (unchanged from Phase 1).

## Implementation Steps

### 1. Mobile `/admin/wallet` — remove

In `frontend/src/pages/mobile/admin/WalletPage/index.tsx`:

1. Remove imports (lines 10–11):
   - `import { WalletDemandCard } from '@/components/wallet/WalletDemandCard';`
   - `import { WalletDemandChart } from '@/components/wallet/WalletDemandChart';`
2. Remove the forecast `useQuery` (lines 46–51) and the now-unused `QueryKeys` import (line 12) if nothing else references it (grep: only the forecast query uses it here — remove).
3. Remove the `<section aria-label="Dự báo nhu cầu ví">` block (lines 153–156), leaving `<WalletTransactionsList />` as the sole child of the inner panel.

### 2. Mobile `/admin/advance-payments` — add

In `frontend/src/pages/mobile/admin/AdvancePaymentsPage/index.tsx`:

1. Add imports:
   ```ts
   import { useQuery } from "@tanstack/react-query";
   import { WalletDemandChart } from "@/components/wallet/WalletDemandChart";
   import { WalletDemandCard } from "@/components/wallet/WalletDemandCard";
   import { QueryKeys } from "@/lib/queryKeys";
   import { walletService } from "@/services/api/wallet.service";
   ```
   (Check existing imports — this page does not import `useQuery`, `QueryKeys`, or `walletService` today.)
2. Inside `AdvancePaymentsPageMobile`, add. **Gate with `enabled: !isAdvPartner`** (red-team fix — this component is shared between `/admin/advance-payments` and `/adv-partner/advance-payments` mobile routes per `App.tsx:292`; without the gate, every adv_partner visit fires `GET /wallet/demand-forecast` and gets a 403 every 5 min):
   ```ts
   const { data: demandForecast, isLoading: demandForecastLoading } = useQuery({
     queryKey: QueryKeys.wallet.demandForecast(),
     queryFn: () => walletService.getDemandForecast(),
     enabled: !isAdvPartner,
     staleTime: 5 * 60_000,
     refetchInterval: 5 * 60_000,
   });
   ```
3. Inside the `<div className="grid gap-3">` overview block (lines 266–290), after the `<TreasuryFeePanel … />` element (closes around line 288) and before the closing `</div>`, add — guarded admin-only to match `WalletBalanceCard`'s `{!isAdvPartner && …}`:
   ```tsx
   {!isAdvPartner && (
     <>
       <WalletDemandChart data={demandForecast} isLoading={demandForecastLoading} />
       <WalletDemandCard data={demandForecast} />
     </>
   )}
   ```
   - Stacked (chart above card) to match the prior mobile wallet layout order.
   - Both components already render their own `rounded-2xl border … shadow-sm` surface — no extra wrapper needed.
   - **Visual-consistency note (red-team):** the surrounding siblings (`WalletBalanceCard`, `AdvPartnerHeroStrip`, `TreasuryFeePanel`) all use a `compact` prop + `border-[#D8E2EE]` styling; the forecast components have no `compact` variant and use `border-border/60`. The surfaces will intentionally diverge in border color/footprint from the compact tiles. Acceptable per user decision (chart + card together); noted here so the divergence is deliberate, not accidental.
   - **Skeleton note (red-team):** this page has an early-return loading skeleton at ~line 181 (`if (page.summaryLoading && page.requests.length === 0)`) that hardcodes a 3-row placeholder with no forecast slot. After this change, on slow networks the user will see the old skeleton then the forecast surfaces pop in → minor layout shift. Acceptable; document explicitly rather than expand the skeleton.

## Success Criteria

- [ ] Mobile `/admin/wallet` shows hero balance, pending-out tile, sync, transactions — no chart/card.
- [ ] Mobile `/admin/advance-payments` shows chart then card after the treasury hero group, before the tab switcher / lists.
- [ ] Hidden for `adv_partner`.
- [ ] `pnpm type-check` clean.

## Risk Assessment

**Low.** Same presentational-relocation pattern as Phase 1. Red-team corrections applied:
- `useQuery` gated with `enabled: !isAdvPartner` (shared mobile component — load-bearing gate).
- Documented: forecast surfaces diverge in styling from the surrounding compact tiles (deliberate).
- Documented: the page's early-return loading skeleton does not preview the new section (minor layout shift on slow networks, accepted).
- Vertical scroll length increases by two surfaces; acceptable per user decision (chart + card together). No safe-area changes needed (the parent `MobilePageShell` already pads bottom).
