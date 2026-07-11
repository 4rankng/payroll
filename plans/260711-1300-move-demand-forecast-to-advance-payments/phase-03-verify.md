---
phase: 3
title: "Verify"
status: pending
priority: P2
dependencies: [1, 2]
---

# Phase 3: Verify

## Overview

Static + build verification that both relocations are clean: no orphaned imports, no type errors, no lint regressions, and the production bundle still compiles.

## Requirements

- Functional: both pages compile and type-check.
- Non-functional: `pnpm lint` passes; no dead code left behind.

## Implementation Steps

1. **Type-check** (per `backend/AGENTS.md` + `frontend/AGENTS.md` guidance):
   ```bash
   cd frontend && pnpm type-check
   ```
   Expect 0 errors. If errors appear, they will almost certainly be:
   - Unused import left in a wallet page (`QueryKeys`, `WalletDemandChart`, `WalletDemandCard`, `walletService`). Fix by removing.
   - Missing import in an advance-payments page (`useQuery`, `QueryKeys`, `walletService`, the two components). Fix by adding per Phase 1/2 steps.

2. **Lint**:
   ```bash
   cd frontend && pnpm lint
   ```
   Expect clean. The `@typescript-eslint/no-unused-vars` rule will flag any leftover import — caught here if missed above.

3. **Production build** (catches lazy-chunk / import-resolution issues the dev server hides):
   ```bash
   cd frontend && pnpm build
   ```
   Both `AdvancePaymentsPage` and `WalletPage` are `lazy()`-imported in `App.tsx` (lines 22, 25, 41, 48); the build must still split them cleanly.

4. **Manual smoke** (dev server):
   ```bash
   make dev   # or: cd frontend && pnpm dev
   ```
   - Visit `/admin/wallet` → confirm: hero balance, "Đang chi trả" tile, sync button, transactions list. **No** "Nhu cầu ứng lương" chart, **no** "Mức cần giữ trong ví" card.
   - Visit `/admin/advance-payments` (admin) → confirm: Treasury Hero (Wallet | Flow | Fee), **then** the demand chart + card, then Pipeline section, then tabs/table.
   - Visit `/admin/advance-payments` as `adv_partner` (or toggle role) → confirm the demand section is **absent**, matching `WalletBalanceCard` parity.
   - Resize to mobile breakpoint (or use `/mobile` routes) → confirm the mobile layout per Phase 2.

5. **adv_partner network silence (red-team — replaces the vacuous dedupe check)**: log in as `adv_partner`, visit `/adv-partner/advance-payments` on mobile, open DevTools Network and filter `demand-forecast`. Confirm **zero** `demand-forecast` requests fire — not on mount, not after 5 min. This validates the `enabled: !isAdvPartner` gate; without it, a 403 storm would be invisible (errors are silently swallowed). Also do this on desktop `/admin/advance-payments` reached as adv_partner.

6. **Responsive-width visual check (red-team)**: the desktop `AdvancePaymentsPage` is a responsive hybrid (`useIsMobile()`). Check the demand-chart section at three widths: 768px (collapse point — expect single column), 1023px (just under old `lg` — expect single column with `md:` choice), and ≥1024px (side-by-side). Also verify on the dedicated mobile route.

## Success Criteria

- [ ] `pnpm type-check` exits 0.
- [ ] `pnpm lint` exits 0.
- [ ] `pnpm build` succeeds; both lazy chunks present.
- [ ] Manual smoke confirms wallet page is clean and advance-payments page shows the section in the chosen spot, admin-only.
- [ ] **adv_partner network silence:** zero `demand-forecast` requests fired as adv_partner (validates `enabled: !isAdvPartner`).
- [ ] Section looks correct at 768px, 1023px, and ≥1024px on both desktop and mobile routes.

## Risk Assessment

**Very low.** Pure verification step. If any check fails, the fix is local to the two page files touched in Phases 1–2; no backend or shared-contract surface is implicated.
