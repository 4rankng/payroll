---
type: architecture
title: Backend HTTP Transport
description: Handlers, middleware chain, request/response contracts, RBAC enforcement, and the conventions that keep the HTTP layer thin and free of GORM types.
tags: [backend, http, middleware, rbac, handlers, dto, validation]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-0c60b8722ffc081c51df87e3
    resource: repo://backend/internal/AGENTS.md
  - id: openwiki-source-bb85a585365bad2d8b04f798
    resource: repo://backend/internal/app/dto/AGENTS.md
  - id: openwiki-source-86f697faa586533d24f62d85
    resource: repo://backend/internal/transport/http/middleware/security_headers.go
  - id: openwiki-source-80c58d83d1feff25104094b3
    resource: repo://backend/internal/transport/http/response/error_translator.go
  - id: openwiki-source-44e4834b3240a239158799aa
    resource: repo://backend/internal/transport/http/validation/request_validator.go
  - id: openwiki-source-98b4fef5bee3b5a0d880f16b
    resource: repo://docs/api.md
  - id: openwiki-source-aca61ad26f85a3f6b9e0d210
    resource: repo://docs/decisions/ADR-006-clock-injection-pattern.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Backend HTTP Transport

`internal/transport/http` is the boundary between the world's HTTP and the application's use cases. It owns parsing, validation, RBAC enforcement, error translation, and response shaping. It does **not** own business logic — handlers delegate to application services and translate domain errors into HTTP responses.

```
internal/transport/http/
├── handlers/          HTTP handlers (grouped by domain)
├── middleware/        request id, recovery, security headers, auth, RBAC,
│                      rate limit, partner scoping, IP whitelist, audit context
├── helpers/           context, request parsing, pagination, formatting
├── response/          error translator + response envelope
├── validation/        request validator (validator tags, custom rules)
├── uploadguard/       upload size + mime guards
└── routes             wired by internal/app/bootstrap/routes.go
```

<!-- openwiki: broken internal link [../docs/api.md] file "../docs/api.md" does not exist. Fix the href or restore the target, then delete this comment. -->
The complete route inventory is in [`docs/api.md`](../docs/api.md). This page covers the patterns, not the route list.

## Middleware chain

The chain is mounted in `internal/app/bootstrap/routes.go` and applied per-route group. The order matters:

1. **Security headers** (`middleware/security_headers.go`) — XSS protection, content-type options, frame deny, referrer policy.
2. **Request timeout** (`middleware/request_timeout.go`) — caps request duration so a slow downstream cannot exhaust the server.
3. **Rate limiting** (`middleware/rate_limit.go`) — Redis-backed per-endpoint counters.
4. **Auth (JWT)** (`middleware/auth.go`) — extracts and validates the bearer token; injects user ID and role into the Gin context. Self-service password reset is gated by `PASSWORD_RESET_ENABLE`.
5. **Authorization (Casbin RBAC)** (`middleware/authorization.go`, ADR-008) — checks the actor against `backend/configs/casbin_policy.csv`.
6. **Audit context** (`middleware/audit_context.go`) — sets up the per-request audit logging context for non-blocking emit.
7. **Partner scoping** — for `partner`/`adv-partner` roles, enforces project-level access (handlers consult `isScopedPartnerRole` plus `CanUserAccessProject` / `CanUserAccessEmployee`).
8. **Tenant semaphore** (`middleware/tenant_semaphore.go`) — limits concurrent heavy operations per tenant where appropriate.
9. **IP whitelist** (`middleware/ip_whitelist.go`) — enforces allow-listed source IPs for sensitive webhooks (notably OnePay/9Pay IPN endpoints).
10. **API metrics** (`middleware/api_metrics.go`) — Prometheus counters/timers for each route.

Recovery middleware (`middleware/`) is mounted globally to catch panics and return a 500 with a request id.

## Handlers

Handlers in `internal/transport/http/handlers/` are grouped by domain — `admin/`, `partner/`, `employee/`, `attendance/`, `advance_payment/`, `disbursement/`, `wallet/`, `bank/`, `audit/`, etc. Each handler:

- Receives services via constructor injection, wired in `internal/app/bootstrap/container.go`.
- Parses the request via `helpers/request.go` and validates via `validation/request_validator.go`.
- Calls an application service and translates its result (success or domain error) to an HTTP response.
- Emits non-blocking audit events when handling imports/exports.
- Never imports `internal/infra/persistence` and never sees a GORM model. The handler talks to services; services talk to repositories.

```go
type MyHandler struct {
    myService    *services.MyService
    auditService *infrastructure.AuditService
    clock        clock.Clock
}
```

## Request conventions

- **DTOs live in `internal/app/dto/`** — one file per capability, with Vietnamese labels baked in.
- **Validation tags** are consumed by `validation/request_validator.go` (validator/v10 with custom rules in Vietnamese where helpful).
- **Pagination, sorting, filtering** — `helpers/pagination.go` and `helpers/request.go` produce typed query helpers; handlers reuse them rather than parsing `c.Query()` ad-hoc.
- **Upload guards** — `uploadguard/` enforces size and mime limits before files reach services.
- **Audit context** — `middleware/audit_context.go` exposes an actor + request id that handlers can attach to audit entries.

## Response conventions

`internal/transport/http/response/` owns the response shape.

- A standard envelope: `{ "data": ..., "error": ... }` with localized Vietnamese error messages.
- `error_translator.go` is the **only** place that maps `domain.*Error` types to HTTP status codes and user-facing strings. Code elsewhere must use the helper rather than translate ad-hoc.
- `response.go` provides success helpers that handlers call instead of writing raw JSON.
- Pagination responses embed `page`, `pageSize`, `total`, and `data`.

The error code vocabulary (`backend/internal/domain/error_codes.go`) is the contract that ties domain errors to HTTP messages. The translation layer is the only place that touches both sides.

## Partner scoping

Two layers enforce partner scope:

1. **Casbin policy** — endpoint-level allow/deny by role.
2. **Handler-level checks** — handlers call `isScopedPartnerRole(ctx)` and the project/employee access predicates from the user service before returning data.

Both layers are required: a partner without an endpoint deny rule could still call a wildcard endpoint; an endpoint deny rule cannot know which project the request refers to. `docs/api.md` documents the partner-aware endpoints; new partner endpoints must include both checks.

## Webhooks and IP whitelisting

The OnePay and 9Pay IPN endpoints are mounted under `/api/v1/webhooks/` with `IP whitelist` middleware (`middleware/ip_whitelist.go`) so only the provider IPs can post. The handler verifies the signature, enqueues an `ipn:process` asynq task, and returns 200 quickly. See `integrations/payment-providers.md` for the IPN contract and `integrations/background-jobs.md` for the worker.

## Admin clock endpoints (non-prod only)

`/api/v1/admin/clock/{time,set,advance,reset}` exist to drive integration tests but are not mounted in production builds. They let a test harness freeze, set, or advance server time without monkey-patching. See ADR-006 and `architecture/domain-layer.md`.

## Conventions summary

- Handlers never touch GORM or domain internals beyond the service interface.
- Services never read `c.Request` or set status codes; that is the handler's job.
- Errors from services are typed; `response/error_translator.go` is the only place they become HTTP.
- Vietnamese UI strings are written once — in DTOs, error translators, or component templates.
- Every endpoint that is partner-accessible must apply both Casbin and project-scoped checks.
