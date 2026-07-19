---
phase: 2
title: Core Employee Surface Polish
status: completed
priority: P1
dependencies:
  - 1
effort: 3-4 days
---

# Phase 2: Core Employee Surface Polish

## Overview

Apply the shared visual language to the wage, timesheet, advance, history, and bank surfaces while preserving every calculation, formatter, disclosure, and mutation contract.

## Context Links

- Plan: [Employee Mobile Polish](./plan.md)
- Depends on: [Phase 1](./phase-01-employee-design-foundation-and-shell.md)
- Visual direction: [Employee mobile concept](./assets/employee-mobile-concept.png)

## Requirements

### Functional

- Regular hierarchy: header, wage hero, quick actions, month, grouped timesheets, bank.
- Flexible hierarchy: header, month, advance request, request history, bank; check-in employees also retain attendance guidance/history and the action toolbar.
- Preserve all advance states: available, awaiting payroll, previous month, missing bank, exhausted, pending, fee loading, validation/provider error, cancellation, and confirmation.
- Propagate the cancelling request ID/pending state so rapid reopen or double-confirm produces one cancellation request and affected controls remain locked through settlement.

### Non-functional

- Use employee-owned semantic classes patterned after `card/card-body`, compact `stats`, `list/list-row`, intentional `avatar`, and `btn`; never apply unprefixed or `ct-*` daisyUI classes to employee markup.
- Exactly one 32px primary money value per screen. Use tabular figures for money, hours, dates, times, account numbers, and counts.
- Status requires text or icon; long Vietnamese names, banks, and values must wrap before truncation.

## Architecture

Keep `.tsx` components render-only and retain the existing `mobileHome.ts`, grouping, payment-status, formatting, and API hooks. Consolidate visual variants through small local presentation helpers or semantic classes; do not create a new configuration framework or duplicate components.

## Related Code Files

- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeWalletHero.tsx` — pay-first card, compact metrics/actions, focus-visible states.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeTimesheetPanel.tsx` — responsive list rows and non-color payment status.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeBankInfoCard.tsx` — consistent destination surface and copy action.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeAttendanceHistoryCard.tsx` — compact status/history rows and disclosure behavior.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/advance-payment/AdvancePaymentRequestForm.tsx` — quota/request hierarchy and all validation/mutation states.
- Modify: `/Users/dev/Documents/projects/payroll/frontend/src/components/advance-payment/AdvancePaymentHistoryCard.tsx` — semantic status list and cancellation affordance.
- Modify: existing co-located tests; create tests for `EmployeeWalletHero` and `EmployeeTimesheetPanel` if absent.

## Implementation Steps

1. Record current props, accessible names, state branches, formatters, callbacks, and query/mutation behavior as regression contracts.
2. Polish `EmployeeWalletHero`: one dominant amount, 2-4 compact metrics, restrained quick actions/nudges, visible focus, and no invented profile/avatar data.
3. Convert timesheet, attendance, and request-history presentation to consistent list rows while retaining semantic articles/buttons, disclosures, and infinite loading.
4. Normalize bank and advance-request surfaces to the employee tokens; keep fee calculation, missing-bank guidance, confirmation, and validation unchanged.
5. Make dense three-column summaries collapse or stack below 375px when values collide; avoid nested vertical scrolling.
6. Align loading, empty, offline/error, pending, success, cancelled, and destructive-confirmation visuals across modules.
7. Add a per-request cancellation-pending contract and same-call-stack guard; keep the confirmation open/busy through settlement, restore focus afterward, and recover cleanly on error.
8. Expand component tests across long text, zero/large currency, status labels/icons, keyboard actions, pending mutation, rapid double-confirm, retained error data, and focus restoration.

## Todo

- [ ] Polish regular wage and timesheet flow.
- [ ] Polish flexible advance and history flow.
- [ ] Normalize bank and attendance-history surfaces.
- [ ] Cover state matrices and long-content behavior.
- [ ] Guard cancellation against rapid reopen/double-submit.

## Success Criteria

- [ ] Regular and flexible variants use the planned hierarchy without route or business-rule changes.
- [ ] All financial and attendance states remain understandable without color alone.
- [ ] 320px layouts have no clipped metrics, actions, status pills, or currency values.
- [ ] Existing interaction tests and new presentation-state tests pass.
- [ ] Rapid cancellation activation produces one API call; failure re-enables the correct request without losing context.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Visual refactor changes advance behavior | High | Do not move calculations/API calls; assert callbacks and mutation states before and after. |
| Dense values overflow at small widths | High | Test 320px with Vietnamese long text and extreme numbers; stack summaries when needed. |
| Large component rewrite creates duplicate versions | Medium | Refine existing components in place; add only small shared presentation primitives with multiple consumers. |

## Security Considerations

- Preserve the current self-service display/copy contract and never expose bank or payroll data in any additional surface or artifact. Bank masking is a separate product/privacy decision.
- Error and offline states must not leak provider diagnostics or authorization details.

## Next Steps

- Phase 3 hardens the behavior-sensitive attendance toolbar and performs whole-portal QA.
