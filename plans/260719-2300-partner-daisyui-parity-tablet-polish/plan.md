---
title: Partner Surface Polish v2 — daisyUI Parity + Tablet + Token Cleanup
status: completed
priority: P2
branch: main
tags: [frontend, partner, ui, daisyui, tablet, tokens, responsive]
created: '2026-07-19T23:00:00.000Z'
createdBy: 'ck:cook'
predecessor: plans/260719-1700-partner-surface-polish-tailkit
---

# Partner Surface Polish v2 — daisyUI Parity + Tablet + Token Cleanup

## Objective

Polish the partner frontend (web + tablet + mobile) to visual parity with admin by:
1. Extending daisyUI theme scope so partner reuses the existing `congtruong` theme admin uses (no new theme — name is pre-existing debt, out of scope to rename).
2. Adding `--partner-*` token definitions + override block mirroring admin's `[data-admin-ui]` block.
3. Populating `partner.css` with `partner-*` utility classes mirroring `admin-daisy.css`.
4. Token cleanup of 56 pre-existing Tailkit literal classes in D1–D4.
5. Tablet-specific polish: new `useIsTablet` hook + tuned layouts for 768–1023px range.
6. Shell polish: PartnerSidebar, MobileBottomNav partner branch, SidebarToggle.

## Key Decisions

1. **Reuse `congtruong` theme** (not define new). User: "since it is just wrong theme name, just use congtruong theme for partner (admin and partner use same theme)".
2. **All-at-once single commit** (user choice for sequencing).
3. **Additive-only daisyUI themeRoot extension** — admin's `[data-admin-ui]` selector unchanged.
4. **Token override block mirrors admin's exactly** — `[data-partner-ui], html.partner-route-active { ... }`.
5. **MobileBottomNav class rename keeps admin aliases** for backward compat.

See full plan details in commit message and reports/implementation-log.md (to be written).
