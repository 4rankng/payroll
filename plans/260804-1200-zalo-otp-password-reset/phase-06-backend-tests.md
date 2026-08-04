---
phase: 6
title: "Backend: Tests & Verification Gate"
status: pending
priority: P1
dependencies: ["1", "2", "3"]
effort: "M"
---

# Phase 6: Backend: Tests & Verification Gate

## Overview

Close the loop on the backend with the test scenarios that the per-phase "Success
Criteria" reference but defer to a centralized gate: anti-enumeration timing,
end-to-end confirm semantics, and the integration scenario that exercises the whole
stack against a live backend. This phase does **not** introduce new production code —
only `*_test.go` files plus any test helpers (fakes, `httptest` Zalo mock).

Per `AGENTS.md`: `make api-test` after every feature change, and integration scenarios
go in `backend/tests/integration`.

## Requirements

- **Functional**
  - Unit tests for: phone normalization, provider (`-124` retry, redact), store (atomic consume, wrong-code survival, TTL), service (request/confirm/resend + role gate + anti-enumeration), rate-limit key-getters (body-restore + phone-norm).
  - Integration test: `/auth/zalo-reset/request` → `/confirm` against the live backend with a stubbed ZNS provider (inject a fake at the bootstrap seam, or use a configurable `ZALO_BASE_URL` env that the provider honors in dev — see Implementation Step 3).
  - **Anti-enumeration timing test** (the headline security assertion): measure wall-clock of `/request` for known vs unknown mobile; assert `< 5ms` delta after warm-up.
- **Non-functional**
  - All new tests `-race` clean.
  - No real network calls to `business.openapi.zalo.me` — every test uses `httptest` or a fake `zalo.Sender`.
  - Coverage ≥ 80% for `internal/infra/zalo/` and `internal/app/services/zaloreset/`.

## Architecture

### Test taxonomy

| Layer | What it asserts | File(s) |
|---|---|---|
| Unit — phone | `NormalizePhone` table-driven | `infra/zalo/phone_test.go` |
| Unit — provider | `-124` single retry; no-retry on `-118`; redact; refresh-token-missing rejection | `infra/zalo/provider_test.go` (httptest) |
| Unit — store | atomic consume; wrong-code survival; TTL; concurrency | `infra/cache/zalo_reset_store_test.go` (miniredis or real redis in dev) |
| Unit — service | role gate, anti-enumeration dummy, async send panic recovery, `UpdatePasswordAndInvalidateSessions` called once, event published | `app/services/zaloreset/service_test.go` |
| Unit — rate-limit | mobile normalization, body restore, fallback to IP | `middleware/rate_limit_test.go` (extend existing) |
| Integration | end-to-end `/request`→`/confirm`, ZNS stub records a send with the right `otp_code` | `tests/integration/zalo_reset_test.go` |

### Anti-enumeration timing harness

```go
// tests/integration/zalo_reset_test.go
func TestZaloResetRequest_NoEnumerationTimingDelta(t *testing.T) {
    // Seed: an employee with mobile "0987654321".
    // Warm-up: 5 calls each to stabilize JIT/cache/Redis conn pool.
    // Measure: 20 calls known vs 20 unknown, alternate to absorb drift.
    // Assert: |meanKnown - meanUnknown| < 5ms AND p99 delta < 15ms.
    //
    // R-Z8: this is the test that proves Create and CreateDummy have the same
    // Redis RTT profile. If it ever flakes, investigate before tuning the threshold —
    // a real timing leak is a security regression, not flakiness.
}
```

## Related Code Files

- **Create:**
  - `backend/internal/infra/zalo/{phone,provider,credential_repo}_test.go` (some may land in Phase 1; this phase completes/extends them).
  - `backend/internal/infra/cache/zalo_reset_store_test.go` (Phase 2 starts; Phase 6 completes).
  - `backend/internal/app/services/zaloreset/service_test.go`.
  - `backend/internal/transport/http/middleware/zalo_rate_limit_test.go` (or extend the existing `rate_limit_test.go`).
  - `backend/tests/integration/zalo_reset_test.go`.
- **Reference (read-only):**
  - `backend/internal/infra/cache/password_reset_token_store_test.go` — store test pattern (atomic GETDEL, TTL).
  - `backend/internal/app/services/passwordreset/service_test.go` — service test pattern (fakes, anti-enumeration).
  - `backend/internal/app/services/otp/*_test.go` — code-gen + hash test pattern.
  - `backend/tests/integration/` — existing integration harness setup.

## Implementation Steps

1. **Audit per-phase tests** — verify every "Success Criteria" checkbox from Phases 1–3 has a corresponding test; fill gaps here.
2. **Service tests** — fake `userRepo`, fake `zalo.Sender` (records the last `template_data["otp_code"]`), spy `store` (records Create/Consume calls), spy `eventBus`. Cover: known-employee, unknown-mobile, admin-mobile (role gate), wrong-code-no-consume, consume-twice, weak-password, panic-in-send-goroutine.
3. **Integration seam for ZNS** — add a `ZALO_BASE_URL` env (default = production endpoint) honored by `Provider`. In integration tests, point it at an `httptest.Server` that returns `{error:0,data:{msg_id:"test"}}` and records the request. This also lets Phase 1's unit tests avoid hard-coded URLs.
4. **Timing test** — implement the harness above; run locally 5× to establish a stable threshold; commit with the threshold documented.
5. **`make api-test`** — run the full backend integration suite; resolve any regressions from the `NewAuthHandler` signature change (Phase 3 R-Z10).
6. **Race + coverage** — `cd backend && go test ./... -race -cover`; confirm the two new packages clear 80%.

## Success Criteria

- [ ] `go test ./internal/infra/zalo/... ./internal/app/services/zaloreset/... ./internal/infra/cache/... -race -cover` green; coverage ≥ 80% on each.
- [ ] Integration test exercises `/request` → `/confirm` end-to-end with a stubbed ZNS endpoint, and the stub records exactly one send carrying the correct 6-digit `otp_code`.
- [ ] Anti-enumeration timing test passes with `< 5ms` mean delta, `< 15ms` p99 delta, consistently across 5 local runs.
- [ ] `make api-test` green — no regressions from the bootstrap constructor change.
- [ ] Every Phase 1–3 "Success Criteria" checkbox is backed by a named test (add a cross-reference comment in each test's docstring).

## Risk Assessment

- **R-Z11 (timing test flakiness on CI):** CI runners are noisy; a 5ms threshold can flake under load. **Mitigation:** run the timing test with `t.Parallel()` disabled, pin to a single-core measurement, and tag it `// +build timing` so it can be skipped on flaky CI runners while still running locally and in nightly. Document the threshold's basis.
- **R-Z12 (integration test needs a real Redis + MySQL):** The existing `make api-test` harness already provides these (dev docker-compose). No new infra; just follow the existing integration-test setup pattern.
