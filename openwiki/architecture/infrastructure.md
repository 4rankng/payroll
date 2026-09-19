---
type: architecture
title: Backend Infrastructure Adapters
description: How domain ports are implemented — GORM repositories with ToDomain mappers, Redis Streams event bus, cache invalidation after commit, asynq client, OnePay/9Pay disbursement adapters, anti-corruption layer, email and Zalo integrations.
tags: [backend, infrastructure, gorm, redis-streams, asynq, onepay, ninepay, cache]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-0c60b8722ffc081c51df87e3
    resource: repo://backend/internal/AGENTS.md
  - id: openwiki-source-3108763110185ccb9cf1655a
    resource: repo://backend/internal/app/AGENTS.md
  - id: openwiki-source-3d0b00c530be91b05c860949
    resource: repo://backend/internal/domain/transaction_manager.go
  - id: openwiki-source-0e339d0fbfd848d3c007e835
    resource: repo://backend/internal/infra/disbursement/onepay/provider.go
  - id: openwiki-source-e6f70bf8559c1cdc23fc2907
    resource: repo://backend/internal/infra/events/cache_invalidation_handler.go
  - id: openwiki-source-743ab60481dd87f3902f8623
    resource: repo://backend/internal/infra/persistence/base_repository.go
  - id: openwiki-source-49857e2784126dfe96ada34e
    resource: repo://docs/decisions/ADR-003-payment-provider-abstraction.md
  - id: openwiki-source-675cdb8aa785fd3b58b5747f
    resource: repo://docs/decisions/ADR-004-redis-streams-event-bus.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Backend Infrastructure Adapters

`internal/infra` is where the domain ports are realized. Every infrastructure adapter implements a domain interface and is constructed exactly once in `internal/app/bootstrap/`. The dependency rule is strict: infra depends on `domain` ports, never the other way around (ADR-001).

```
internal/infra/
├── asynq/             Background job client, handlers, mux
├── cache/             Redis cache key/value stores (incl. nonce, OTP, password-reset)
├── disbursement/      Payment-provider adapters
│   ├── onepay/        Production provider (OnePay)
│   └── ninepay/       Sandbox/dev provider (9Pay)
├── email/             Email sending (Resend + sandbox)
├── events/            Redis Streams event bus + handlers
├── observability/     slog logging, DB metrics, Prometheus metrics
├── persistence/       GORM repositories (the data layer)
├── storage/           File storage abstraction
├── transaction/       GORM transaction manager / unit of work
└── zalo/              Stateless Zalo ZNS / OAuth client
```

`internal/adapters/` is a separate anti-corruption layer for external-system contracts; details below.

## GORM repositories and ToDomain

Every aggregate has a GORM repository under `internal/infra/persistence/`. Repositories:

- Accept a `context.Context` and detect whether a transaction is active via `domain.GetTransactionFromContext`; if so they join the open `*gorm.DB`, otherwise they use the default connection.
- Hold GORM models and `ToDomain()` mappers; they convert GORM rows into domain types before returning.
- Implement the repository interface declared in `internal/domain/ports/...` (or, for the wallet, in `internal/domain/wallet/repository.go`).

The mapping discipline keeps handlers free of GORM types. A handler calls `service.GetByID(ctx, id)` and receives a domain entity; the GORM `*gorm.DB` and the GORM model stay inside the repository. Migrations live under `backend/migrations/` as numbered `NNN_*.up.sql` / `NNN_*.down.sql` pairs; the discipline is recorded in `backend/migrations/AGENTS.md` and indexed in `docs/database.md`. See `migrations/schema-evolution.md` for the high-impact stories.

The shared `base_repository.go` provides common query helpers and the common `*gorm.DB` plumbing; `common/` holds cross-repo utilities (query builders, filter parsing).

## Redis cache

`internal/infra/cache/` implements `domain.CachePort`. Stores under this tree:

- `nonce_store.go` — request-scoped nonce tracking.
- `otp_pending_store.go` — pending OTP entries (used by Zalo OTP and password reset flows).
- `password_reset_token_store.go` — reset tokens.
- `zalo_reset_store.go` — Zalo-specific reset state.

**Invalidation-after-commit (ADR-007):** cache writes happen inside transactions only when explicitly designed to; invalidations happen after `txManager.RunInTransaction` returns successfully. The `cache_invalidation_handler.go` Redis-stream consumer implements the post-commit path for event-driven invalidation. Code review enforces the rule; integration tests cover the rollback case. Inside a transaction, call `domain.RegisterAfterCommit(ctx, func() { cache.Invalidate(...) })`) — the function will run after a successful commit (or in a goroutine immediately if no transaction is active).

## Event bus (Redis Streams)

Per ADR-004, the event bus is Redis Streams-backed. Files in `internal/infra/events/`:

- `redis_event_bus.go` — production event bus (`Publish` writes to the stream).
- `event_bus.go` — in-memory bus used in tests for determinism.
- `event_bus_with_workers.go` — consumer pool that reads from Redis Streams.
- `event_registry.go` — maps event type strings to handler functions.
- `sequential_handler.go` — orders financial events so settlement cannot overtake a payment.

Domain events live in `internal/domain/events.go` (50+ types) and are produced by per-aggregate factories (`event_factory_*.go`). The publishing pattern is non-blocking and goes through `EventBus.Publish(ctx, event)` — typically registered as an after-commit callback so the publish follows the database write.

Handlers under `internal/infra/events/handlers/`:

- `audit_event_handler.go` — writes audit-log rows from events.
- `settlement_event_handler.go` — processes wallet-payment settlements.
- `cache_invalidation_handler.go` — the cache key driver for the post-commit invalidation rule.
- `employee_user_created_handler.go` — provisions employee users after hire events.
- `cash_forecast_accuracy_handler.go` — keeps forecast accuracy metrics current.

The original database outbox tables were dropped in migration 062; events now flow Redis-only. The financial critical path (wallet payments, ledger entries) is still transactional and does not rely on event delivery for correctness.

## asynq client and worker registration

`internal/infra/asynq/` holds the asynq client/server, the mux, and the inspector wiring. Worker handlers live in `internal/app/workers/` (see `architecture/application-services.md`). Workers are registered in `internal/app/bootstrap/routes.go` and run against the same Redis instance used for caching and streams.

## Payment provider adapters

`internal/infra/disbursement/` is the realization of `domain.PaymentProvider`. Per ADR-003 the system is provider-agnostic; in production it points at OnePay, in sandbox at 9Pay.

```
internal/infra/disbursement/
├── onepay/    Production adapter
│   ├── client.go          HTTP client
│   ├── provider.go        Implements domain.PaymentProvider
│   ├── queue.go           Pending-disbursement queue
│   ├── signing.go         Request signing (RSA / HMAC)
│   ├── types.go           Provider-specific types
│   ├── error_codes.go     Error mapping
│   └── webhook.go         IPN signature verification
└── ninepay/   Sandbox / dev adapter (mirrors onepay structure)
```

Each adapter:

- Implements `domain.PaymentProvider` (Topup, Transfer, StatusInquiry, Cancel, IPNVerification).
- Owns its request signing (`signing.go`) and error vocabulary (`error_codes.go`); `webhook.go` verifies inbound IPN signatures.
- Uses an internal queue (`queue.go`) for in-flight requests, with a separate worker pulling from the queue and writing settlement rows back into the wallet.
- Maps provider errors to domain errors so the wallet state machine can react correctly.

The disbursement service (`internal/app/services/disbursement/`) selects the provider at runtime via `registry_selector.go` reading from `registry.go`; the selection is configured via env (OnePay or 9Pay) and `domain/services/` does not need to know which is active. See `integrations/payment-providers.md` for the operational view.

## Email and Zalo

`internal/infra/email/` provides the email-sending adapter: Resend in production plus a sandbox provider for tests. Templates live in the application layer (`internal/app/services/notification/`, `flex_pay/`); the infra layer is a thin transport.

`internal/infra/zalo/` is a stateless Zalo ZNS / OAuth v4 client, ported from a prior PHP implementation. It is DB-agnostic — credentials are fetched through a `CredentialSource` interface implemented by `internal/app/services/zaloconnect/`, which reads from the `settings` table (`zalo.enabled`, `zalo.credentials`). The OAuth v4 connect flow and hot enable/disable are admin-UI driven; env vars (`ZALO_*`) only seed the initial settings rows on first boot. See `integrations/zalo-otp.md`.

## Anti-corruption layer

`internal/adapters/` is a separate tree for ports that protect the domain from external shapes. Unlike `internal/infra/...` (which implements domain ports), adapters translate foreign shapes into our internal vocabulary without leaking transport concerns upward. Anything that adapts a third-party API (payment-provider SDK, OTP gateway, SMTP) starts here when the surface is large enough to deserve its own package.

## Observability

`internal/infra/observability/` provides slog-backed structured logging, DB metrics, and Prometheus metrics. The convention from `backend/AGENTS.md` is to log at critical junctions, use `slog` (never `fmt.Printf`), and never ignore errors except with `_`. The system-health UI surfaces (`frontend/src/components/system-health/`, `cron-health/`) consume these signals. See `operations/observability.md`.

## Storage

`internal/infra/storage/` abstracts file storage (uploads, exports). Concrete implementations are chosen at bootstrap. The `internal/app/services/db_export/` and `internal/app/services/excel/` packages emit non-blocking audit events for every file import/export.

## Bootstrap wiring

`internal/app/bootstrap/infrastructure/` builds the GORM `*gorm.DB`, Redis client, asynq client/server, and storage adapters. `bootstrap/repositories/` constructs the GORM repositories. The single wiring point is `internal/app/bootstrap/container.go`; everything new goes there exactly once.
