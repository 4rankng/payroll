# Phase 01: Token Authority and Persistence

## Requirements

- Introduce a Redis-backed refresh lock using owner-checked release and a bounded TTL.
- Keep the Provider's local mutex and add the distributed lock around every token exchange.
- Re-read credentials only after both locks are held; skip a duplicate exchange when another actor already replaced the rejected/expiring access token.
- When an admin supplies a refresh token, exchange the merged candidate credentials before storing them and persist only the validated successor pair.
- Use optimistic compare-and-swap for settings JSON mutations so stale status/error writes cannot restore a consumed refresh token.
- Classify Zalo `-14014` as an invalid refresh token requiring a fresh pair;
  retain generic OAuth transport and malformed-response failures as temporary
  retryable failures.

## Touchpoints

- `backend/internal/infra/zalo/provider.go`
- `backend/internal/infra/zalo/types.go`
- `backend/internal/infra/zalo/*_test.go`
- `backend/internal/app/services/zaloconnect/service.go`
- `backend/internal/app/services/zaloconnect/service_test.go`
- `backend/internal/infra/persistence/settings_repository.go`
- `backend/internal/app/bootstrap/services/init.go`

## Contracts to Preserve

- Admin credential request/response schema and routes
- `zalo.Sender` and password-reset delivery behavior
- Existing settings key `zalo.credentials`
- Refresh-token retention when Zalo omits a replacement
- Previously stored credentials remain unchanged when validation of a newly
  supplied pair fails

## Rollback

- Remove distributed coordinator wiring and restore local-only locking.
- Restore save-without-validation behavior only if Zalo's endpoint cannot validate newly issued refresh tokens; keep CAS protection independently.
