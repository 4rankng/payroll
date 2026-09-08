---
type: architecture
title: Architecture Overview
description: DDD layered architecture for the Go backend, request lifecycle, and the boundary between domain, application, transport, and infrastructure.
tags: [architecture, ddd, clean-architecture, layering, golang]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-0618ef15c5b118d2ad707e7d
    resource: repo://backend/internal/app/bootstrap/container.go
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-f10a0a57b8b00bec9b396088
    resource: repo://docs/decisions/ADR-001-ddd-clean-architecture.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Architecture Overview

The backend follows Domain-Driven Design with Clean Architecture (ADR-001). Dependencies point strictly inward — `transport → app → domain ← infra` — so the domain layer never imports Gin, GORM, or any other infrastructure framework. The frontend is a separate Vite + React SPA that talks to the backend over HTTP/JSON.

The full component diagram and the broader topology live in `docs/system-architecture.md`. This page focuses on what the four layers do, how a single request traverses them, and where the seams live.

## The four layers

### Domain — `backend/internal/domain/`

The innermost layer. Owns the business model: entities (Employee, Project, Timesheet, Wallet, LedgerEntry, Settlement), value objects, domain services, specification objects, the transaction manager abstraction, and the wallet aggregate. The package has zero framework imports — only the standard library and `internal/pkg/clock` / `internal/constants`. New aggregates go in subdirectories with their own repository interfaces (see the wallet aggregate as the reference example).

The `domain/ports/` directory holds repository and service interfaces; `domain/services/` holds pure business logic; `domain/specs/` holds reusable rules (e.g. `WorkingDaysSpec`); `domain/transactions/` owns the unit-of-work abstraction; `domain/wallet/` is the self-contained wallet aggregate.

### Application — `backend/internal/app/`

Orchestrates use cases. App services are the only layer allowed to depend on both domain ports and infrastructure clients, and they translate user intent into domain operations. They never touch GORM models directly — handlers consume domain types. The application layer also owns bootstrap (`app/bootstrap/container.go`), DTOs (`app/dto/`), and asynq background workers (`app/workers/`).

### Transport — `backend/internal/transport/`

The HTTP edge. `transport/http/handlers/` exposes the API grouped by domain area; `transport/http/middleware/` owns the auth → authorization → audit-context chain plus rate-limit, request-timeout, and security-header middleware; `transport/http/validation/` parses and validates incoming requests; `transport/http/response/` formats errors. Handlers depend only on app services and domain types — never on GORM models.

### Infrastructure — `backend/internal/infra/`

The outermost layer. Implements the ports defined in domain: `infra/persistence/` for GORM repositories (50+ implementations mapping GORM models → domain types via `ToDomain()`), `infra/events/` for the event bus, `infra/cache/` for Redis, `infra/asynq/` for background job processing, `infra/disbursement/` for OnePay and 9Pay adapters, `infra/email/` for Resend, `infra/storage/` for file uploads, `infra/observability/` for slog and Prometheus, and `infra/transaction/` for the GORM-backed unit of work.

## Request lifecycle

A single HTTP request flows through the layers as follows:

1. **Gin router** dispatches to the matching route group.
2. **Middleware chain** runs in order: security headers → request timeout → rate limit → JWT auth → Casbin RBAC authorization → audit context → partner scoping.
3. **Handler** parses the request, calls into an application service, and translates the result into a response. Handlers may also publish domain events and emit non-blocking audit logs.
4. **Application service** orchestrates the use case: loads aggregates through domain ports, invokes domain services for rules, persists via repositories, and emits events.
5. **Domain layer** enforces business invariants, runs the wallet state machine, validates accounting balance, applies the transaction manager for atomicity.
6. **Infrastructure repositories** write to MySQL via GORM; the event bus publishes to Redis Streams; asynq enqueues background jobs.
7. **Cache invalidation** runs after the transaction commits (see ADR-007) so reads after the write see the new state.

The component diagram in `docs/system-architecture.md` is the authoritative visual; this prose is the layer-level reading guide.

## What goes where

| Concern | Layer | Example |
|---|---|---|
| Business rule | `domain/` | `domain/services/timesheet_validation_service.go` |
| Persistence interface | `domain/ports/` | `WalletTopupRepository` in `domain/wallet/repository.go` |
| GORM implementation | `infra/persistence/` | `wallet_topup_repository.go` |
| Use case | `app/services/` | `bcc_import_pipeline.go` orchestrates bulk import |
| HTTP handler | `transport/http/handlers/` | `handlers/wallet.go` |
| Middleware | `transport/http/middleware/` | `auth.go`, `authorization.go`, `audit_context.go` |
| Configuration | `internal/config/` | reads env vars |
| External API client | `infra/disbursement/`, `infra/email/` | OnePay, 9Pay, Resend adapters |

## Anti-corruption layer

`internal/adapters/` provides ports for external system contracts — events, cache, notification, services — so the rest of the codebase can refer to stable interfaces rather than third-party API shapes. Adapters wrap SDK calls and translate between wire format and domain types.

## Container wiring

`backend/internal/app/bootstrap/container.go` is the dependency injection root. It builds infrastructure (`bootstrap/infrastructure/`), repositories (`bootstrap/repositories/`), and application services (`bootstrap/services/`), then constructs the HTTP handlers and registers routes (`bootstrap/routes_*.go`). New features typically add wiring in three places: a new `routes_<area>.go` file for the route registration, a new entry in `container.go` to construct the handler, and a `bootstrap/services/` initializer that takes its dependencies from the repository and infrastructure registries.

## Why the layering matters

Without strict layer boundaries, business logic tends to leak into handlers (HTTP concerns) or repositories (database concerns), making it hard to test in isolation and fragile to framework upgrades. The DDD layering is what makes property-based testing of accounting rules (`accounting_rules_test.go`) possible without a database, and what makes the wallet state machine unit-testable without Redis. ADR-001 records the rationale and the rejected alternatives (MVC, pure hexagonal, flat structure).

## Cross-cutting conventions

- **Clock** — all business time uses `clock.Now()` (Asia/Ho_Chi_Minh) via `pkg/clock`. Never `time.Now()` for domain logic.
- **Logging** — `slog` via `infra/observability/`. Never `fmt.Printf`. Log at critical junctions.
- **Errors** — domain errors via `domain.NewNotFoundError`, `NewValidationError`, `NewConflictError`, etc. Transport layer translates them to HTTP status codes.
- **Events** — emit via `EventBus.Publish()`. Service code wraps emission in `go func() { ... }()` for non-blocking dispatch.
- **Audit** — `AuditService.LogFileImport()` / `LogFileExport()` are non-blocking and always include `data_type`, `record_count`, `file_name`.

## Related pages

- [Quickstart](../quickstart.md) — repo map and entry points.
- [Transaction Manager and Outbox](./transaction-manager-and-outbox.md) — how domain operations achieve atomicity.
- [Double-Entry Ledger and Chart of Accounts](./double-entry-ledger.md) — the wallet + ledger surfaces.
- [Auth, RBAC, and Casbin](../operations/auth-rbac-and-casbin.md) — middleware chain details.
