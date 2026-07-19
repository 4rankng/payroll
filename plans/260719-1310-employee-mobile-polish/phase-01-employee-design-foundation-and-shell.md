---
phase: 1
title: "Employee Design Foundation and Shell"
status: pending
priority: P1
dependencies: []
effort: "2-3 days"
---

# Phase 1: Employee Design Foundation and Shell

## Overview

Create the employee-only visual foundation and one reusable shell for ready, loading, and error states. Remove page-specific safe-area and skeleton drift before polishing individual cards.

## Context Links

- Plan: [Employee Mobile Polish](./plan.md)
- Visual direction: [Employee mobile concept](./assets/employee-mobile-concept.png)
- Rules: `frontend/CLAUDE.md`, `docs/standards/ui-guidelines.md`

## Requirements

### Functional

- Preserve `EmployeeRouter` routing by payment schedule and all existing header actions.
- Reuse one shell contract for regular, flexible, loading, and recoverable error states.
- Keep regular employee smooth-scroll actions, adding reduced-motion behavior.

### Non-functional

- Scope styles to `.employee-mobile-page` or an employee data attribute; no admin/partner leakage.
- Keep Tailwind 3.4, daisyUI 4.12.24, `ct-` prefix, and existing admin theme root unchanged unless a verified isolated configuration is required.
- Maintain 16px phone gutters, 24px section rhythm, 44px controls, safe-area insets, and one readable scroll container.

## Architecture

`EmployeeRouter` selects a page; both pages render through `EmployeeMobileShell`. Extend that shell with reusable chrome/state slots rather than duplicating markup. Map daisyUI concepts to employee-scoped presentation utilities (`card`, `btn`, optional compact `stats`) while keeping the current semantic CSS variables and shadcn components as the implementation base.

## Related Code Files

- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/styles/variables.css` — complete semantic employee surface, radius, focus, and toolbar tokens only where missing.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/styles/base.css` — employee-scoped component/focus/reduced-motion rules.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeMobileShell.tsx` — canonical canvas, content width, safe-area and toolbar-reservation contract.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeePortalHeader.tsx` — consistent identity/actions, wrapping and focus states.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/pages/employee/EmployeeRouter/index.tsx` — shared loading/error chrome.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/pages/employee/EmployeePage/index.tsx` — remove duplicate loading shell and honor reduced motion.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/pages/employee/FlexiblePayEmployeePage/index.tsx` — remove duplicate loading shell and page-local toolbar padding.
- Create: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeMobileShell.test.tsx`.
- Create: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeePortalHeader.test.tsx`.
- Create: `/Users/dev/Documents/projects/payroll/frontend/src/pages/employee/EmployeeRouter/index.test.tsx`.

## Implementation Steps

1. Inventory hard-coded employee colors, radii, shadows, z-indexes, and spacing; map each repeated value to an existing or justified `--employee-*` token.
2. Define a narrow employee presentation contract for surface/card/list/stat/button states. Do not expose or repurpose admin `ct-*` selectors.
3. Refactor `EmployeeMobileShell` to own max width, gutters, header/content offsets, bottom safe area, and optional action-toolbar reservation.
4. Extract reusable loading and recoverable-error chrome that shape-matches the final layout and uses Vietnamese action-specific labels.
5. Replace the three duplicated loading/error page shells while preserving query/refetch behavior and retained data.
6. Normalize header wrapping, icon-only accessible names, focus-visible rings, notification count, and account-menu trigger behavior.
7. Add tests for shell variants, header actions, router loading/error/refetch, safe-area classes, and regular-page reduced-motion scrolling.

## Todo

- [ ] Freeze employee token and component mapping.
- [ ] Unify shell, header, loading, and error chrome.
- [ ] Add focused state and accessibility tests.
- [ ] Confirm admin and partner routes receive no employee selectors.

## Success Criteria

- [ ] All employee page variants render through one shell contract without changing data hooks or routes.
- [ ] No duplicated loading-page structure remains in the two employee pages.
- [ ] Keyboard focus, safe-area padding, and reduced-motion behavior are explicit and tested.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Employee CSS leaks into other roles | High | Require employee-root scoping and screenshot/DOM comparison of admin and partner shells. |
| Shared loading refactor changes query behavior | Medium | Move presentation only; retain existing hooks, enabled flags, refetch callbacks, and boundaries. |
| Token cleanup causes contrast regression | Medium | Verify semantic pairs and WCAG contrast before replacing hard-coded values. |

## Security Considerations

- Do not add employee data to CSS, logs, test screenshots, or generated fixtures.
- Preserve protected routing, logout/token removal, and notification/account authorization behavior.

## Next Steps

- Phase 2 consumes the stabilized shell and employee presentation contract.
