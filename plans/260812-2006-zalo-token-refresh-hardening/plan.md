---
title: "Zalo token refresh hardening (ChatBot parity)"
description: "Stop the single-use rotating refresh_token from being double-spent/stuck; finish the in-flight rotation edits; deploy. Matches ChatBot's durable Zalo token handling."
status: pending
priority: P1
effort: "1-2d"
tags: [zalo, infra, concurrency]
created: 2026-08-12
related: [260806-1719-zalo-domain-errors]
---

# Zalo token refresh hardening (ChatBot parity)

## Why this plan

Prod returns:
```json
{ "message": "zalo: token refresh failed: missing access_token or refresh_token",
  "http_status": 500, "code": "INTERNAL_ERROR" }
```
User asks why ChatBot ("enter token once, never expires") can't behave the same here.

**Answer: it can.** Both projects call the same endpoint
(`oauth.zaloapp.com/v4/oa/access_token`) and both use Zalo's **single-use,
rotating `refresh_token`**. The difference is how the refresh is guarded,
persisted, and diagnosed.

## Root cause (evidence-grounded)

| Concern | ChatBot (works) | Payroll (current source) | Gap |
|---|---|---|---|
| Refresh trigger | Reactive (on `-216`/`-213`) | Proactive (`expires_at` within 2h) | **Keep proactive** (decision) |
| Concurrency guard | Redis dist lock `SET NX EX 30` | `sync.Mutex` (process-local only) | **Missing — core gap** |
| Persist rotated tokens | One DB commit before cache evict | `mutateCredentials` load-then-write (non-tx) | **Non-atomic** |
| Parse Zalo error body | Reads `error`/`message` | Done locally, **not deployed** | Deploy |
| On refresh failure | Self-heals next send | Logs only; dead token retained → silent loop | **No self-heal/visibility** |

**Most probable trigger of the prod error:** a deploy/restart overlap, or
api-server + worker as separate processes, lets two refreshes race past the
process-local mutex. The first refresh rotates the single-use `refresh_token`;
the second hits Zalo with the now-dead token and gets an error body with no
token fields → old code printed "missing access_token or refresh_token" and
**retained the dead token**, so every later Send re-failed.

## State of the working tree (read before planning)

`provider.go`, `service.go`, `zalo_test.go` are **modified, uncommitted**. A
prior session already wrote the "diagnose" half:
- `oauthTokenResponse` now parses `error` + `message`.
- `refresh()` keeps the old `refresh_token` when Zalo doesn't rotate it (the
  old hard-reject of access-token-only responses is gone).
- `Credentials` has a `LastError` field; `SaveCredentials`/`Update` clear it.

The error message you pasted comes from the **deployed binary** (old code), not
this source. Half the fix exists locally; it needs the lock + atomic persist +
self-heal to be complete, then commit + deploy.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | A single-use `refresh_token` can never be double-spent across processes/restart-overlap | P1 |
| 2 | Rotated tokens persist atomically (no lost update) | P1 |
| 3 | A failed refresh surfaces a real reason (`LastError` + Zalo code) and stops looping silently | P2 |
| 4 | Prod ZNS sends keep working across token-expiry boundaries with no admin intervention | P1 |

## Non-Goals

- **HTTP error → domain-error mapping** (500 → 400/422). Owned by
  [`260806-1719-zalo-domain-errors`](../260806-1719-zalo-domain-errors/plan.md).
  Coordinate only — shared file `service.go`.
- **AES-GCM encryption-at-rest parity** (ChatBot encrypts the row; payroll
  stores plaintext JSON in `settings`). Separate security task.
- **Switching to reactive refresh.** We keep proactive (user-approved).

## Phases

| # | Phase | Status | Effort |
|---|-------|--------|--------|
| 1 | [Land in-flight rotation edits](./phase-01-land-inflight-edits.md) | pending | 2h |
| 2 | [Distributed Redis refresh lock](./phase-02-distributed-refresh-lock.md) | pending | 4h |
| 3 | [Atomic persist + failure self-heal](./phase-03-atomic-persist-selfheal.md) | pending | 3h |
| 4 | [Deploy + recovery + verify](./phase-04-deploy-recovery-verify.md) | pending | 2h |

## Decisions

- **Full parity, keep proactive refresh** (user-approved 2026-08-12). Lowest
  churn, ChatBot-equivalent reliability. The distributed lock removes the
  double-spend; reactive trigger is not required.

## Related plans / coordination

- [`260806-1719-zalo-domain-errors`](../260806-1719-zalo-domain-errors/plan.md)
  (In Progress) — maps Zalo errors to domain errors in `service.go`. Land
  independently. **Both touch `service.go`** — sequence the commits, don't
  edit-conflict. This plan does not change HTTP status semantics.

## Risks

- **Scoping the commit/deploy.** The working tree also has *unrelated* dirty
  frontend files (`serve.json`, `App.tsx`, `ErrorBoundary.tsx`, `main.tsx`,
  `chunk-reload.*` — a separate HMR/chunk-reload effort). **Do not sweep them
  into the zalo commit or deploy.** Stage only the three zalo files.
- **Dead token already in prod.** Regardless of fix, the stored
  `refresh_token` is consumed. Admin must re-paste one fresh pair after deploy.
- **`time.Now()` in `provider.go`** (lines 137, 143, 205, 252, 254) instead of
  `clock.Now()` — violates rule #1. Pre-existing; fold into Phase 2 while the
  file is open (Provider needs a `clock.Clock` added via bootstrap).

## Open questions

1. Is prod single-process (api-server only) or multi-process (api-server + a
   separate asynq worker)? Determines whether the lock is essential or
   belt-and-suspenders. Confirm in Phase 2. (Either way we ship it.)
2. What exact Zalo error code does the refresh return? Phase 1's logged
   `zalo_error` will tell us; it drives the retain-vs-clear decision in Phase 3.
3. Encryption-at-rest parity wanted? Default non-goal unless requested.

## Verification (whole plan)

- `go test ./internal/infra/zalo/... -race` and
  `go test ./internal/app/services/zaloconnect/... -race` pass.
- Concurrent-refresh test proves only one Zalo refresh fires; losers reuse the
  winner's token.
- Prod ZNS test send succeeds after admin re-paste; a forced expiry-boundary
  refresh does not 500.
- `make api-test` regression-green.
