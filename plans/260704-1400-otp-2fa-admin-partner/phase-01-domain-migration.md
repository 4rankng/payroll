---
phase: 1
title: "Lockout store & Redis pending-session"
status: pending
priority: P1
effort: "S (≈0.5d)"
dependencies: []
---

# Phase 1: Lockout store & Redis pending-session

## Overview

Email-OTP needs **no persistent per-user secret** (the code is generated per-login
and lives only in Redis). What it DOES need is per-account brute-force/lockout
state that survives Redis eviction and restarts. This phase adds two columns to
`users` (mirroring the `tokens_invalid_before` pattern from migration 086 /
commit `619f9cd`) and a small Redis helper package for pending-session storage.

## Requirements

- **Functional:** store per-account `otp_failed_attempts` and `otp_locked_until`
  on `users`; provide Redis helpers to create/load/consume/delete pending OTP
  sessions keyed by an opaque id and bound to a user_id + IP + UA.
- **Non-functional:** columns are additive (safe on a running DB); Redis helpers
  are TTL'd (5-min) and use high-entropy keys (non-enumerable).

## Architecture

### Lockout columns on `users` (migration 087)
```
ALTER TABLE users
  ADD COLUMN otp_failed_attempts INT NOT NULL DEFAULT 0
    COMMENT 'consecutive failed OTP verifies; reset on success',
  ADD COLUMN otp_locked_until DATETIME(3) NULL
    COMMENT 'when set, OTP-gated login is refused until this instant'
  AFTER tokens_invalid_before;
```
Why on `users` (not a separate table): the red-team retired RT-M6's
`user_mfas` table — email-OTP has no secret to store. Lockout is the only
persistent state, and it mirrors the adjacent `tokens_invalid_before` column
(086) exactly. One small migration, one repo method extension.

### Redis pending-session layout
- Key: `otp:pending:<opaque-id>` — id = 32 bytes `crypto/rand`, base64url (~192 bits).
- Value: JSON `{user_id, code_hash, ip, ua, attempts, created_at}`. TTL 5 min.
- Secondary index: `otp:user:<user_id>` → the current opaque-id (RT-H1: **one
  live pending session per user** — creating a new one overwrites/invalidates the
  prior). This defeats the "1000 sessions × 5 attempts each" brute-force amplifier.
- `code_hash`: SHA-256 (or Argon2id — see note) of the 6-digit code. Compared
  constant-time. **Note:** hashing 6-digit codes adds limited offline resistance
  (tiny keyspace) — its value is log hygiene + preventing a Redis-reader from
  seeing the plaintext code. SHA-256 is sufficient; Argon2id is overkill here.
  The real brute-force defense is the per-account attempt cap (Phase 6), not the hash.

## Related Code Files

- **Create:** `migrations/087_add_user_otp_lockout.up.sql` + `.down.sql`
- **Modify:** `internal/domain/user.go` — add `OTPFailedAttempts int` and
  `OTPLockedUntil *time.Time` to the `User` struct (GORM auto-maps; mirror the
  `TokensInvalidBefore` field added in 086).
- **Modify:** `internal/domain/user.go` `UserRepository` interface — add
  `UpdateOTPLockout(ctx, userID, failedAttempts int, lockedUntil *time.Time) error`.
- **Modify:** `internal/infra/persistence/user_repository.go` — GORM impl of the
  above (mirror `UpdateTokensInvalidBefore`).
- **Create:** `internal/infra/cache/otp_pending_store.go` — Redis helpers:
  `CreateSession(ctx, userID, codeHash, ip, ua) (sessionID string, err error)`,
  `GetSession(ctx, sessionID) (*OTPPendingSession, error)`,
  `DeleteSession(ctx, sessionID) error`,
  `DeleteSessionsForUser(ctx, userID) error` (RT-M4),
  `IncrementSessionAttempts(ctx, sessionID) (int, error)`.
- **Modify:** `internal/app/bootstrap/infrastructure/init.go` — construct +
  inject `OTPPendingStore` (reuse the shared Redis client at `infra/persistence/redis.go:20`).

## Implementation Steps

1. **Migration 087** — `ALTER TABLE users ADD COLUMN ... AFTER tokens_invalid_before`
   (matches 086's placement + comment style). Down migration drops both columns.
2. **`domain/user.go`** — add the two fields with GORM tags (`otp_failed_attempts`,
   `otp_locked_until`); add `UpdateOTPLockout` to the `UserRepository` interface.
3. **`user_repository.go`** — impl `UpdateOTPLockout` using
   `Updates(map[string]interface{}{...})` (handles the nullable `locked_until`).
4. **`otp_pending_store.go`** — Redis helpers using `go-redis/v9` (already a dep).
   `CreateSession` writes both keys in a pipeline: `SET otp:pending:<id> <json> EX 300`
   and `SET otp:user:<userID> <id> EX 300` (overwriting any prior id → RT-H1).
5. **DI wiring** — pass the shared `*redis.Client` into `OTPPendingStore`; inject
   into `OTPService` (Phase 2).

## Success Criteria

- [ ] Migration 087 applies cleanly (additive) and rolls back without affecting
      existing rows or the `tokens_invalid_before` column.
- [ ] `go build ./...` passes with the new field + repo method.
- [ ] `OTPPendingStore.CreateSession` overwrites a prior session for the same
      user (RT-H1 verified at the storage layer).
- [ ] `GetSession` returns not-found after TTL expiry.
- [ ] `UpdateOTPLockout` round-trips both fields (incl. `locked_until = NULL` reset).

## Risk Assessment

- **Risk:** forgetting to reset `otp_failed_attempts = 0` on a successful verify
  (an old failure count blocks the next login). **Mitigation:** Phase 2's verify
  success path explicitly resets both fields; covered in Phase 6 tests.
- **Risk:** the `otp:user:<userID>` secondary key and the `otp:pending:<id>` key
  drift out of sync (TTL skew). Acceptable — worst case a stale secondary key
  points to an expired pending key → `GetSession` returns not-found → user
  re-logs in. No security impact.
