---
phase: 4
title: "Visual QA and Verification"
status: pending
effort: ""
priority: P1
dependencies: [1, 2, 3]
---

# Phase 4: Visual QA and Verification

## Overview

Prove the redesign is complete, responsive, accessible, behavior-preserving, and isolated to admin surfaces.

## Requirements

- Functional: authenticated route sweep and representative interaction checks for navigation, filters, details, tabs, dialogs, and financial actions up to—but not including—irreversible confirmation.
- Non-functional: no new lint/type/build errors, no critical console/network errors, no overflow/clipping, and graph updated.

## Related Code Files

- Modify only defects found during verification.
- Update tests under `frontend/src/**` or `frontend/tests/**` when changed shared behavior requires regression coverage.
- Update docs only if the admin design-system contract materially changes maintainer guidance.

## Implementation Steps

1. Run focused component tests, `pnpm lint`, TypeScript checks, and production build.
2. Use authenticated Playwright QA at 390, 768, and 1440px for every reachable `/admin` route.
3. Check overflow, touch targets, focus visibility, status labels, reduced motion, loading/empty/error states, and console/network errors.
4. Exercise representative route state: filters/sorting/pagination, settings tabs, modal deep links/close, sidebar/dock navigation, and mobile-only subpages.
5. Verify high-risk actions through confirmation preview without submitting destructive or money-moving operations.
6. Capture before/after representative screenshots and compare non-admin isolation.
7. Run adversarial code review and tester/debugger agents; fix verified regressions.
8. Run `graphify update .`, synchronize plan status, and update warranted documentation.

## Success Criteria

- [ ] Focused tests, lint, TypeScript, and build pass.
- [ ] Every route renders at 390/768/1440px without critical visual defects.
- [ ] No critical console/network errors are introduced.
- [ ] Navigation, filters, modal deep links, tabs, and high-risk confirmation previews behave as before.
- [ ] Partner/employee representative routes remain visually unchanged.
- [ ] Reviewer and tester gates pass; graph is current.

## Risk Assessment

Live data can make some states intermittent. Use deterministic DOM checks where possible, avoid submitting irreversible actions, and record any environment-only failures separately from regressions.
