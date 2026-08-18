---
title: Admin wallet employee bank lookup
description: >-
  Add a read-only Admin employee bank lookup on the wallet page using the active
  OnePay account verifier.
status: completed
priority: P1
branch: main
tags:
  - feature
  - frontend
  - backend
  - api
  - onepay
  - wallet
blockedBy: []
blocks: []
created: '2026-08-18T07:05:11.360Z'
createdBy: 'ck:plan'
source: skill
---

# Admin wallet employee bank lookup

## Overview

Add `Tra cứu tài khoản` to both Admin wallet render paths. The shared dialog searches employees server-side, then calls a new Admin-only endpoint that loads the selected employee's persisted bank tuple and invokes the existing `AccountVerifier` capability. The result compares stored information with the bank-confirmed holder name without updating employee data or initiating a transfer.

## Approved Decisions

- Read-only lookup; no employee update, wallet payment, transfer, or migration.
- Server loads employee and bank data; client sends only `employee_id`.
- Production uses the active OnePay provider; existing generic `/check-account` remains unchanged.
- Missing bank data never calls the provider.
- Provider/configuration failures stay distinct from confirmed invalid or name-mismatch results.
- One shared responsive dialog; both desktop and mobile Admin pages expose the action.
- The dialog also offers a one-off manual mode for bank/SWIFT, account number,
  and account name; custom values use the existing read-only account-check API
  and are never persisted.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Backend employee account lookup](./phase-01-backend-employee-account-lookup.md) | Completed |
| 2 | [Responsive wallet lookup dialog](./phase-02-responsive-wallet-lookup-dialog.md) | Completed |
| 3 | [Validation and release readiness](./phase-03-validation-and-release-readiness.md) | Completed |

## Dependencies

- No blocking plan. `260718-2130-wallet-bulk-transfer-pipeline` is implemented and is only a wallet-page preservation touchpoint.
- Existing dependencies: employee repository, bank relation, disbursement registry, `AccountVerifier`, employee infinite search, wallet dialog primitives.

## Acceptance Criteria

- Admin can find an employee by the existing server-side search and explicitly run a current lookup.
- Request payload contains only `employee_id`; provider input comes from persisted employee/bank data.
- UI shows stored bank, account number/name, provider-confirmed name, and a clear Vietnamese outcome.
- Admin can switch to manual entry and verify custom values without selecting an employee.
- Missing data, invalid account, name mismatch, unavailable verifier, and transport failure are distinguishable.
- Desktop at 1280px and mobile at 390px/320px retain action parity, readable wrapping, no horizontal overflow, and 44px touch targets.
- Focused backend/frontend tests, frontend lint/type-check, backend checks, `make api-test`, and graph updates are classified with evidence.

## Out of Scope

- Editing employee bank information from the dialog.
- Persisting the lookup verdict or validation timestamp.
- Initiating or pre-filling a transfer.
- Partner or Employee role UI.
- Database schema, provider protocol, or existing generic account-check contract changes.
