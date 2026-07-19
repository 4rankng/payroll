---
phase: 3
title: Attendance Dock Accessibility and Visual QA
status: completed
priority: P1
dependencies:
  - 1
  - 2
effort: 2-3 days
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
- When today's attendance query is unknown or failed, disable attendance mutation, show `Chưa tải được chấm công`, and provide one guarded retry while retaining safe cached guidance.

### Non-functional

- Retain `role="toolbar"`; do not use daisyUI `dock` navigation semantics.
- Reserve toolbar height plus bottom safe area centrally and ensure anchored content scrolls above sticky/fixed chrome.
- Validate keyboard, screen reader, 200% text, reduced motion, slow/offline network, safe areas, and phone landscape.

## Architecture

`EmployeeCheckInCard` continues to own attendance state and guidance. `EmployeeAttendanceActionDock` remains a thin render-only toolbar. The shell and toolbar share one CSS block-size/reservation contract that supports wrapped or fallback labels at 200% text instead of assuming a fixed pixel height. Existing Radix alerts/sheets and the current user-opened map disclosure remain intact; do not introduce eager third-party tile loading.

## Related Code Files

- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeAttendanceActionDock.tsx` — semantic toolbar, action sizes, states, safe area, focus.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.tsx` — remove mobile CTA duplication, preserve state guidance, anchors and live-region behavior.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeAttendanceActionDock.test.tsx`.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.test.tsx`.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/tests/e2e/employee-portal.spec.ts` — critical regular/flexible/check-in paths and mobile viewport assertions.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/tests/fixtures/test-data.ts` — synthetic employee, bank, payroll, attendance, and location fixtures; keep sensitive-looking values fictitious.
- Create if the spec grows beyond existing helpers: `/Users/dev/Documents/projects/payroll/frontend/tests/page-objects/EmployeePortalPage.ts` — reusable employee portal locators/actions per E2E convention.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/playwright.config.ts` — align the checked-in web-server command and readiness URL with `pnpm` and Vite port 5173, or expose an equivalent verified external-server contract for the employee gate.
- Modify: `/Users/dev/Documents/projects/payroll/.gitignore` — ignore Playwright `test-results/`, `playwright-report/`, and equivalent local artifacts if still absent.

## Implementation Steps

1. Enumerate attendance states: profile loading; today's-attendance loading/error/refetch; outside window; GPS acquiring/inaccurate/denied/unavailable; outside geofence; ready; submitting; checked in; checkout/cooldown; completed; rejected/orphaned; and confirmed without salary.
2. Assert existing labels, callbacks, cancellation confirmation, map disclosure, and salary warnings before changing presentation.
3. Move fixed-toolbar reservation into the shared shell contract; remove page-specific compensation and verify all scroll targets.
4. Polish toolbar actions with semantic employee/button tokens, 44px minimum height, 2:3 width ratio, explicit focus, safe-area padding, and action-specific disabled/loading labels. Share a dynamic/minimum block-size variable with the shell; avoid truncating the only explanation at 200% text.
5. Add polite live announcements only for meaningful GPS/attendance transitions; avoid announcing every location sample.
6. Add an explicit today-query failure/refetch branch and prove no check-in/out mutation is available while server attendance state is unknown.
7. Keep employee E2E fully synthetic: fail on unmatched employee API requests, never target demo/production, use fictitious payroll/bank/location data, ignore local artifacts, and scan retained traces/screenshots/videos for tokens, bank numbers, coordinates, names, and payroll values before upload.
8. Align the Playwright server contract with the documented Vite port 5173, then from `frontend/` run focused Vitest (`pnpm test:run -- <employee test files>`), `pnpm lint`, explicit TypeScript (`pnpm exec tsc -p tsconfig.json --noEmit`), `pnpm build`, and `pnpm test:e2e -- employee-portal.spec.ts`.
9. Execute manual visual/accessibility QA at 320x568, 360x800, 375x667, 390x844, 412x915, 568x320, 768x1024, 1024x768, and 1280x800. Repeat critical phone flows at 200% text, reduced motion, keyboard-only, VoiceOver/TalkBack, slow/offline, long names/banks, and zero/large values; assert actual toolbar/content geometry rather than class names.
10. Compare admin and partner shells for style leakage. From the repo root run `make api-test`, then `graphify update .`.

## Todo

- [ ] Protect the complete attendance state matrix with focused tests.
- [ ] Disable attendance actions during today-query failure/refetch and add guarded recovery.
- [ ] Centralize toolbar reservation and safe-area behavior.
- [ ] Extend employee Playwright coverage.
- [ ] Isolate and redact Playwright artifacts.
- [ ] Complete viewport, accessibility, and cross-role regression QA.

## Success Criteria

- [ ] Fixed actions never cover content, browser chrome, keyboard focus, or anchored sections.
- [ ] At 200% text and 320px/landscape, toolbar labels remain understandable and measured content clearance is at least the rendered toolbar height plus safe area.
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
