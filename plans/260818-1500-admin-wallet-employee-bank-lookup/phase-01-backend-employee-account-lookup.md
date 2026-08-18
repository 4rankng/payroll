---
phase: 1
title: Backend employee account lookup
status: completed
priority: P1
dependencies: []
effort: medium
---

# Phase 1: Backend employee account lookup

## Overview

Add an Admin-only employee account lookup contract beside manual-disbursement account verification. The server owns the bank tuple and performs no persistence or transfer side effects.

## Requirements

- `POST /api/v1/admin/manual-disbursement/employee-account-check` accepts `{ "employee_id": number }`.
- Load `domain.Employee` by ID; the existing query preloads `Bank`.
- Reject missing employee bank ID/relation, SWIFT, account number, or account name before provider resolution.
- Resolve the active provider through the existing registry and require `infrastructure.AccountVerifier`.
- Call with a unique lookup request ID, account type `"0"`, amount `0`, and stored bank/account/name values.
- Never write the employee, create a transaction code, create a wallet payment, or call `InitiateTransfer`.
- Never log full account number or credentials.

## API Contract

Success returns employee identity, stored bank details, `outcome` (`valid`, `invalid`, `name_mismatch`, or `unverified`), and the normalized provider result. Use non-2xx responses for missing employee/data, unavailable verifier, and transport failure. OnePay structured configuration responses `12`, `13`, and `19` map to `unverified`, not confirmed-invalid.

## Related Code Files

- Modify: `backend/internal/transport/http/handlers/disbursement/manual_disbursement_handler.go`
- Modify: `backend/internal/app/bootstrap/container.go`
- Modify: `backend/internal/app/bootstrap/routes_disbursement.go`
- Create or extend: focused handler tests under `backend/internal/transport/http/handlers/disbursement/`
- Extend: `backend/tests/integration/flow_disbursement.go`

## Implementation Steps

1. Inject `domain.EmployeeRepository` into `ManualDisbursementHandler` without changing existing callers' behavior.
2. Define request/response/outcome types and a small classification helper.
3. Implement the server-authoritative lookup with early incomplete-data exits and sanitized logging.
4. Register the route under the authenticated/authorized Admin manual-disbursement group and apply the existing strict limiter.
5. Add spies proving provider input comes from persistence and no provider call occurs on incomplete data.
6. Cover valid, invalid, mismatch, configuration-unverified, unsupported-provider, transport-error, and not-found paths.

## Success Criteria

- [x] Only Admin-authorized callers reach the endpoint.
- [x] The client cannot substitute bank details.
- [x] Lookup has zero persisted or money-moving side effects.
- [x] Outcomes remain semantically distinct and Vietnamese messages are actionable.
- [x] Logs and errors do not expose the complete account number.

## Risk Assessment

- OnePay rejection codes can be mistaken for transport failures. Preserve structured provider answers while separating actual call errors.
- The shared handler constructor has many call sites/tests. Update all compile-time callers and keep dependency optional only where established tests require it.
- Strict rate limiting is per IP and coarse; the provider's existing queued/rate-limited wrapper remains the OnePay TPS authority.
