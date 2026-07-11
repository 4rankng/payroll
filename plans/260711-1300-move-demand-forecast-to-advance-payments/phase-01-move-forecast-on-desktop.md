---
phase: 1
title: "Move forecast on desktop"
status: pending
priority: P2
dependencies: []
---

# Phase 1: Move forecast on desktop

## Overview

Remove the `WalletDemandChart` + `WalletDemandCard` block from the **desktop** `/admin/wallet` page and add it to the **desktop** `/admin/advance-payments` page in a new section placed between the Treasury Hero and the Pipeline + operating-health section. Admin-only (hidden for `adv_partner`).

## Requirements

- Functional: `/admin/advance-payments` shows the demand-forecast chart + card; `/admin/wallet` no longer does.
- Non-functional: same data source (`walletService.getDemandForecast()` + `QueryKeys.wallet.demandForecast()`, 5-min staleTime/refetchInterval). No layout regressions on either page.

## Architecture

The two components are presentational — `data` / `isLoading` flow in via props. The owning page owns the `useQuery`. Desktop layout in the advance-payments page uses the same 3-column `items-stretch` grid as the wallet page did (2 cols chart + 1 col card) so the visual ratio is preserved.

```
 pages/admin/WalletPage/index.tsx
   - drop imports of WalletDemandChart / WalletDemandCard
   - drop the forecast useQuery (optional: keep if other elements use it — they don't, so remove)
   - remove the demand-forecast <section> grid

 pages/admin/AdvancePaymentsPage/index.tsx
   + add useQuery for demand forecast (same key/service as wallet)
   + add imports for WalletDemandChart / WalletDemandCard
   + render new <section> between Treasury Hero (section 2) and Pipeline (section 3)
```

## Related Code Files

- Modify: `frontend/src/pages/admin/WalletPage/index.tsx` (remove imports, query, section)
- Modify: `frontend/src/pages/admin/AdvancePaymentsPage/index.tsx` (add query, imports, section)
- Reference (no change): `frontend/src/components/wallet/WalletDemandChart.tsx`
- Reference (no change): `frontend/src/components/wallet/WalletDemandCard.tsx`
- Reference (no change): `frontend/src/services/api/wallet.service.ts` — `getDemandForecast()`
- Reference (no change): `frontend/src/lib/queryKeys/index.ts` — `QueryKeys.wallet.demandForecast()`
- Reference (no change, **explicit exclusion**): `frontend/src/pages/admin/AdvancePaymentsPage/AdvPartnerView.tsx` — the adv_partner desktop route uses a separate component; forecast is admin-only by design. No change needed there.

## Implementation Steps

### 1. `/admin/wallet` — remove the block

In `frontend/src/pages/admin/WalletPage/index.tsx`:

1. Remove the two imports (lines 25–26):
   - `import { WalletDemandChart } from "@/components/wallet/WalletDemandChart";`
   - `import { WalletDemandCard } from "@/components/wallet/WalletDemandCard";`
2. Remove the now-unused `QueryKeys` import (line 27) **only if** nothing else in the file uses it. Grep confirms `QueryKeys` appears only in the forecast query — remove it.
3. Remove the forecast `useQuery` block (lines 163–168):
   ```ts
   const { data: forecast, isLoading: forecastLoading } = useQuery({
     queryKey: QueryKeys.wallet.demandForecast(),
     queryFn: () => walletService.getDemandForecast(),
     staleTime: 5 * 60_000,
     refetchInterval: 5 * 60_000,
   });
   ```
4. Remove the demand-forecast grid `<div>` (lines 278–283):
   ```tsx
   {/* Demand forecast: cohort chart + prediction card */}
   <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-stretch">
     <div className="md:col-span-2">
       <WalletDemandChart data={forecast} isLoading={forecastLoading} />
     </div>
     <WalletDemandCard data={forecast} />
   </div>
   ```
5. Leave the comment that introduced it removed too (the `{/* Demand forecast ... */}` line).

### 2. `/admin/advance-payments` — add the block

In `frontend/src/pages/admin/AdvancePaymentsPage/index.tsx`:

1. Add imports near the other `@/components/wallet` or `@/components/advance-payment` imports:
   ```ts
   import { useQuery } from "@tanstack/react-query";
   import { WalletDemandChart } from "@/components/wallet/WalletDemandChart";
   import { WalletDemandCard } from "@/components/wallet/WalletDemandCard";
   import { QueryKeys } from "@/lib/queryKeys";
   import { walletService } from "@/services/api/wallet.service";
   ```
   (Check whether `useQuery` / `QueryKeys` are already imported; this page currently does **not** import `useQuery` from `@tanstack/react-query` — add it. `walletService` is also new here.)
2. Inside `AdvancePaymentsPage`, add the query near `page` / `attendancePage`. **Gate with `enabled: !isAdvPartner`** (red-team fix — prevents a 403 storm for adv_partner users, since this component can be reached on mobile via the shared route):
   ```ts
   const { data: demandForecast, isLoading: demandForecastLoading } = useQuery({
     queryKey: QueryKeys.wallet.demandForecast(),
     queryFn: () => walletService.getDemandForecast(),
     enabled: !isAdvPartner,
     staleTime: 5 * 60_000,
     refetchInterval: 5 * 60_000,
   });
   ```
3. Add a new `<section>` **after** the Treasury Hero section (the one closing around line 303 with `</section>`) and **before** the Pipeline section ("Trạng thái xử lý ứng lương", opening around line 306). Wrap in the admin-only guard to match the `WalletBalanceCard` parity already present:
   ```tsx
   {/* ─── Demand forecast: cohort chart + recommendation (admin only) ─── */}
   {!isAdvPartner && (
     <section
       aria-label="Nhu cầu ứng lương"
       className="grid grid-cols-1 gap-4 items-stretch md:grid-cols-3"
     >
       <div className="md:col-span-2">
         <WalletDemandChart data={demandForecast} isLoading={demandForecastLoading} />
       </div>
       <WalletDemandCard data={demandForecast} />
     </section>
   )}
   ```
   - **Use `md:grid-cols-3`** (not `lg:`) — red-team correction. The original chart was designed for the `md:` breakpoint on the wallet page; the destination page uses bespoke grid tracks elsewhere, but no plain `lg:grid-cols-3` convention exists. Keeping `md:` truly preserves the chart's visual ratio (the stated goal) and keeps the chart + card side-by-side from 768px+ rather than collapsing to 1-column until 1024px.
   - **Responsive-page note (red-team):** the desktop `AdvancePaymentsPage` itself uses `useIsMobile()` and renders mobile-style chrome under ~768px. The new section uses `md:` so it gracefully collapses to a single column on narrow widths — both the hybrid-desktop and dedicated-mobile paths are covered. Visual-check at 768px and 1024px+ viewports added to Phase 3.

## Success Criteria

- [ ] `/admin/wallet` desktop renders without the chart/card; hero balance + sync + transactions intact.
- [ ] `/admin/advance-payments` desktop renders chart + card in a new section between Treasury Hero and Pipeline.
- [ ] Section is hidden for `adv_partner`.
- [ ] No unused imports / variables left in either file.
- [ ] `pnpm type-check` clean.

## Risk Assessment

**Low.** Presentational component relocation. Red-team corrections applied:
- The `useQuery` is now gated with `enabled: !isAdvPartner` so adv_partner users do not trigger a forbidden `/wallet/demand-forecast` request (the `AdvancePaymentsPageMobile` component is shared between admin and adv_partner mobile routes — `App.tsx:292`).
- The section uses `md:grid-cols-3` (not `lg:`) to truly preserve the chart's original visual ratio.
- The desktop `AdvancePaymentsPage` is a responsive hybrid (`useIsMobile()`), so the section must look right at all widths; Phase 3 adds explicit 768px and 1024px+ visual checks.
