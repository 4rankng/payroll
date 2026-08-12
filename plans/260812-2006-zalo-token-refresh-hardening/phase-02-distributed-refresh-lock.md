---
phase: 2
title: "Distributed Redis refresh lock"
status: pending
priority: P1
effort: "4h"
dependencies: [1]
---

# Phase 2: Distributed Redis refresh lock

## Overview
Replace/augment the process-local `sync.Mutex` in `refreshTokenLocked` with a
Redis `SET NX EX` lock so two processes (or a restart overlap) can never
double-spend the single-use `refresh_token`. Losers re-read the credential row
and reuse the winner's rotated token — exactly ChatBot's pattern.

## Requirements
- Functional: across processes/goroutines, exactly one refresh HTTP call to
  Zalo per rotation; concurrent losers reuse the freshly-rotated token.
- Non-functional: lock auto-expires (TTL) so a crash can't wedge refresh; no
  new domain-layer framework imports (keep the lock in infra).

## Architecture
- Define a tiny port in `internal/infra/zalo` (or `internal/pkg/lock`):
  ```go
  type Locker interface {
      Acquire(ctx context.Context, key string, ttl time.Duration) (Release func(), ok bool, err error)
  }
  ```
- Redis impl reuses the `SetNX` + Lua-release pattern already in
  `internal/app/services/bcc_import_lock.go` (value = random token, release
  only if value matches).
- `Provider` gains a `Locker` (optional — nil = fall back to the existing
  process mutex, preserves unit-test ergonomics).
- `refreshTokenLocked` flow:
  1. Try `Acquire("zalo:token:refresh", 30s)`.
  2. If not acquired: poll `creds.Get` ~2s / 100ms for a changed
     `AccessToken`/`ExpiresAt`; if changed, return the new token; else return
     the last error (do **not** call Zalo).
  3. If acquired: re-read creds (another worker may have rotated while we
     waited), then run the existing `refresh()` + persist; `defer release()`.
- Bootstrap: pass `redis.Client` (via `*persistence.RedisClient`) into
  `zalo.NewProvider` at `internal/app/bootstrap/services/init.go:638`.

## Related Code Files
- Modify: `backend/internal/infra/zalo/provider.go` (Provider struct,
  constructor, `refreshTokenLocked`)
- Modify: `backend/internal/infra/zalo/types.go` (add `Locker` field/ctor param)
- Create: `backend/internal/pkg/lock/redis_lock.go` (or inline in infra/zalo)
- Modify: `backend/internal/app/bootstrap/services/init.go` (inject locker)
- Modify: `backend/internal/infra/zalo/zalo_test.go` (concurrent-refresh test
  with a fake `Locker` that simulates a second process)

## Implementation Steps
1. Confirm prod process topology (api-server only? + separate worker?) —
   grep `cmd/` for a worker entrypoint; check `docker-compose`/deploy notes.
2. Add `Locker` interface + Redis implementation (copy `bcc_import_lock.go`
   acquire/release, generalize key+ttl).
3. Thread `Locker` through `NewProvider`; keep it optional (nil-safe).
4. Rewrite `refreshTokenLocked` to the acquire / poll-reuse / refresh flow.
5. While the file is open: replace `time.Now()` in the refresh path with
   `clock.Now()` (rule #1) — add `clock.Clock` to the Provider.
6. Test: simulate two concurrent refreshers sharing a fake locker where the
   second `Acquire` returns `ok=false`; assert Zalo is hit once and the loser
   returns the winner's token.

## Success Criteria
- [ ] Two concurrent refresh attempts → exactly one Zalo refresh call.
- [ ] Lock-denied loser reuses the rotated token within ~2s; no error surfaced.
- [ ] Lock TTL bounds wedging on crash (release still best-effort on success).
- [ ] No new framework imports in `internal/domain/`.
- [ ] `refresh` path uses `clock.Now()`; new test passes under `-race`.

## Risk Assessment
- **Polling on lock-deny** adds up to ~2s latency to a Send under contention.
  Acceptable: refresh is rare (once per ~24h) and async send paths tolerate it.
- **Lock + clock both plumb through bootstrap** — touch `init.go` once for both.
  Keep the change minimal; don't refactor neighboring wiring.
- **Single-process today**: even if prod is single-process, ship the lock —
  deploy/restart overlap is a real cross-process window.
