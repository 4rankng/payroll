# Phase 03: Tolerant OAuth Expiry Decoding

## Context

Production confirmed that a newly supplied access token is valid through
Zalo's read-only OA profile endpoint, while Payroll rejected the save after the
OAuth endpoint returned HTTP 200. The strict `int64` decoder treats a quoted
or otherwise non-canonical `expires_in` value as a failure of the whole token
response before the valid successor pair can be persisted.

## Implementation

- Decode `expires_in` independently from the required token fields.
- Accept integer JSON numbers and quoted base-10 integers.
- Treat missing, `null`, non-positive, malformed, or unsafe expiry values as
  unknown and use the existing one-hour fallback.
- Preserve existing access-token success, refresh-token rotation/retention,
  empty-body, malformed whole-body, non-2xx, `-14014`, and `-124` behavior.
- Never log response bodies, credentials, or token fragments.

## Files

- `backend/internal/infra/zalo/provider.go`
- `backend/internal/infra/zalo/zalo_test.go`
- `backend/internal/app/services/zaloconnect/service_test.go`

## Verification

- Provider refresh accepts numeric and quoted numeric expiry.
- Missing, `null`, non-positive, and malformed expiry still persist the valid
  successor pair with a one-hour fallback.
- Malformed whole JSON/HTML remains a refresh failure and preserves stored
  credentials.
- Save-time validation persists the successor pair when expiry is quoted.
- Run focused race tests, affected persistence tests, vet/lint, broad backend
  tests, `git diff --check`, and `graphify update .`; classify unavailable
  integration checks explicitly.

## Scope Boundary

- No public API, database schema, environment-variable, or UI changes.
- No live refresh-token exchange, commit, push, or deployment in this phase.
