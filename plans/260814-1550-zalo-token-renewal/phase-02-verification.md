# Phase 02: Verification and Operational Recovery

## Verification

- Regression test: invalid pasted refresh token fails during save, not about 24 hours later.
- Regression test: valid pasted pair is immediately rotated and the successor pair is stored.
- Race test: two Providers sharing storage and a coordinator cause one OAuth exchange.
- Race test: stale metadata mutation cannot overwrite a rotated pair.
- Existing proactive refresh, forced refresh, `-124` retry, access-only response, and secret-log tests remain green.
- Verify `-14014` is surfaced as an invalid-token replacement action while
  OAuth transport failures remain temporary and retryable.
- Run affected package tests with `-race`, `go vet`, full backend tests, and `make api-test`; classify unrelated failures.

## Production Recovery

- After code deployment, obtain one fresh access/refresh pair for **Payroll's
  own Zalo App ID**. A deployment alone does not validate or supply this pair.
- Save through Payroll; immediate validation must succeed and persist the
  successor pair.
- Send one controlled OTP sample and confirm the saved connection remains
  healthy without `-124` or `-14014`.
- Never paste the same refresh-token chain into a second system.

This is an operator checklist, not a claim that a deployment or live Zalo
validation has been completed.

## Result

- Focused and affected-package race tests: passed.
- Full backend unit suite: passed.
- Full `go vet ./...` and `git diff --check`: passed.
- Full repository race suite: Zalo packages passed; suite failed on the
  pre-existing `internal/infra/events/cache_invalidation_handler_test.go:99`
  mock race.
- `make api-test`: unavailable because no API listened on `localhost:8080`.
- Production deploy and live token validation: not performed.
