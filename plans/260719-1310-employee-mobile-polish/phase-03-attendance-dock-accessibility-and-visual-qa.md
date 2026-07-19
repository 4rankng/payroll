---
phase: 3
title: "Attendance Dock Accessibility and Visual QA"
status: pending
priority: P1
dependencies: [1, 2]
effort: "2-3 days"
---

# Phase 3: Attendance Dock Accessibility and Visual QA

## Overview

Harden the fixed attendance action toolbar, validate the full attendance state machine, and complete automated plus manual accessibility and viewport regression gates.

## Context Links

- Plan: [Employee Mobile Polish](./plan.md)
- Depends on: [Phase 1](./phase-01-employee-design-foundation-and-shell.md), [Phase 2](./phase-02-core-employee-surface-polish.md)

## Requirements

### Functional

- Keep secondary `Ứng lương` and wider primary attendance action (`Vào làm`/`Tan ca`) available for check-in-enabled employees only.
- Keep GPS, geofence, schedule, map disclosure, cancellation, and no-salary guidance inside the attendance card; do not duplicate the large attendance CTA on mobile.
- Preserve all attendance state transitions and disabled labels.

### Non-functional

- Retain `role="toolbar"`; do not use daisyUI `dock` navigation semantics.
- Reserve toolbar height plus bottom safe area centrally and ensure anchored content scrolls above sticky/fixed chrome.
- Validate keyboard, screen reader, 200% text, reduced motion, slow/offline network, safe areas, and phone landscape.

## Architecture

`EmployeeCheckInCard` continues to own attendance state and guidance. `EmployeeAttendanceActionDock` remains a thin render-only toolbar. The shell receives one explicit toolbar-presence contract so pages do not hand-maintain padding. Existing Radix alerts/sheets and map disclosure remain intact.

## Related Code Files

- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeAttendanceActionDock.tsx` — semantic toolbar, action sizes, states, safe area, focus.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.tsx` — remove mobile CTA duplication, preserve state guidance, anchors and live-region behavior.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeAttendanceActionDock.test.tsx`.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.test.tsx`.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/tests/e2e/employee-portal.spec.ts` — critical regular/flexible/check-in paths and mobile viewport assertions.
- Modify if required by verified safe-area gap: `/Users/dev/Documents/projects/payroll/frontend/index.html` — ensure `viewport-fit=cover` without other metadata changes.

## Implementation Steps

1. Enumerate attendance states: profile loading, outside window, GPS acquiring/inaccurate/denied/unavailable, outside geofence, ready, submitting, checked in, checkout/cooldown, completed, rejected/orphaned, and confirmed without salary.
2. Assert existing labels, callbacks, cancellation confirmation, map disclosure, and salary warnings before changing presentation.
3. Move fixed-toolbar reservation into the shared shell contract; remove page-specific compensation and verify all scroll targets.
4. Polish toolbar actions with semantic employee/button tokens, 44px minimum height, 2:3 width ratio, explicit focus, safe-area padding, and action-specific disabled/loading labels.
5. Add polite live announcements only for meaningful GPS/attendance transitions; avoid announcing every location sample.
6. From `frontend/`, run focused Vitest (`pnpm test:run -- <employee test files>`), `pnpm lint`, explicit TypeScript (`pnpm exec tsc -p tsconfig.json --noEmit`), `pnpm build`, and `pnpm test:e2e -- employee-portal.spec.ts`.
7. Execute manual visual/accessibility QA at 320x568, 360x800, 375x667, 390x844, 412x915, 568x320, 768x1024, 1024x768, and 1280x800. Repeat critical phone flows at 200% text, reduced motion, keyboard-only, VoiceOver/TalkBack, slow/offline, long names/banks, and zero/large values.
8. Compare admin and partner shells for style leakage. From the repo root run `make api-test`, then `graphify update .`.

## Todo

- [ ] Protect the complete attendance state matrix with focused tests.
- [ ] Centralize toolbar reservation and safe-area behavior.
- [ ] Extend employee Playwright coverage.
- [ ] Complete viewport, accessibility, and cross-role regression QA.

## Success Criteria

- [ ] Fixed actions never cover content, browser chrome, keyboard focus, or anchored sections.
- [ ] Attendance labels/actions match the existing business state machine and remain screen-reader understandable.
- [ ] Focused Vitest, lint, explicit TypeScript, build, employee Playwright, and `make api-test` gates pass.
- [ ] All QA viewports pass with no horizontal overflow or targets below 44px.
- [ ] Admin and partner presentation remains unchanged; `graphify update .` succeeds.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Toolbar refactor breaks check-in/out | Critical | Keep state logic untouched; strengthen state/callback tests before styling. |
| Fixed chrome hides content or focus | High | Central reservation token, anchor tests, short viewport/landscape/safe-area QA. |
| GPS live regions become noisy | Medium | Announce meaningful state transitions only and test with a screen reader. |
| Visual tests expose real business regressions | High | Stop and diagnose; do not update baselines or weaken assertions. |

## Security Considerations

- Preserve geofence/GPS validation and authorization; styling must never bypass disabled states or mutation guards.
- Do not log precise employee location or include real payroll/location data in test artifacts.

## Next Steps

- After all gates pass, review the plan checklist, update documentation only if a durable employee UI convention changed, and hand off for implementation review.
