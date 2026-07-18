---
phase: 2
title: "Shared Admin Primitives"
status: complete
effort: ""
priority: P1
dependencies: [1]
---

# Phase 2: Shared Admin Primitives

## Overview

Unify the reusable admin presentation layer so route migrations are mostly composition rather than bespoke restyling.

## Requirements

- Functional: preserve all component props, callbacks, Radix focus behavior, responsive switching, table sorting, pagination, and URL-modal close semantics.
- Non-functional: consistent visual hierarchy, semantic statuses, tablet-aware tables, compact desktop density, and accessible mobile touch targets.

## Architecture

Build thin admin composites over existing shadcn/Radix primitives and reuse daisyUI class patterns for cards/stats/tables/badges/filters. Avoid a second parallel component library: improve current shared components and introduce only boundaries that remove repeated route markup.

## Related Code Files

- Modify: `frontend/src/components/shared/PageHeader.tsx`, mobile header/shell/section components, `FilterBar.tsx`, stats and empty-state components.
- Modify: `frontend/src/components/ui/responsive-table.tsx`, `data-table.tsx`, `mobile-table.tsx`, pagination, card, button, badge, tabs, dialog, sheet, skeleton.
- Modify: feature header/stat/filter components under users, projects, employees, timesheet, transactions, ledger, advance-payment, payroll, and observability only where shared primitives cannot cover them.

## Implementation Steps

1. Standardize the page-header composition and action behavior for desktop/mobile.
2. Create ledger-style summary, filter/action bar, data surface, mobile record card, and status patterns.
3. Upgrade responsive tables to preserve scan-friendly tablet layouts where feasible and consistent mobile cards elsewhere.
4. Standardize tabs, form fields, dialogs/sheets, confirmation recaps, loading skeletons, empty states, and error alerts.
5. Add focus-visible, semantic text/icon status, tabular-numeral, reduced-motion, and touch-target rules.
6. Keep public props stable and add focused tests only where behavior-bearing markup changes.

## Success Criteria

- [ ] Shared headers, cards/stats, filters, tables/lists, tabs, overlays, and states use one admin visual language.
- [ ] Existing callbacks, props, focus management, sorting, pagination, and modal semantics are unchanged.
- [ ] Mobile controls meet 44px minimum and no status is color-only.
- [ ] Shared changes do not regress non-admin consumers.

## Risk Assessment

Shared primitives have non-admin consumers. Prefer admin-scope selectors/variants, retain public APIs, and test representative partner/employee rendering before accepting broad primitive changes.
