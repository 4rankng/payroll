---
type: architecture
title: Architecture Overview
description: The dependency-direction diagram (transport → app → domain ← infra), the frontend role separation, the three async surfaces (domain events via Redis Streams, asynq background jobs, payment-provider IPN webhooks), and the desktop+mobile parity rule.
tags: [architecture, ddd, redis-streams, asynq, ipn, frontend, mobile, parity]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-0c60b8722ffc081c51df87e3
    resource: repo://backend/internal/AGENTS.md
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-f10a0a57b8b00bec9b396088
    resource: repo://docs/decisions/ADR-001-ddd-clean-architecture.md
  - id: openwiki-source-49857e2784126dfe96ada34e
    resource: repo://docs/decisions/ADR-003-payment-provider-abstraction.md
  - id: openwiki-source-675cdb8aa785fd3b58b5747f
    resource: repo://docs/decisions/ADR-004-redis-streams-event-bus.md
  - id: openwiki-source-28d4ebbc4a3f54433e091cbe
    resource: repo://docs/decisions/ADR-005-asynq-background-jobs.md
  - id: openwiki-source-291b71381ac4ab544f08ee9c
    resource: repo://docs/decisions/ADR-007-transaction-manager-unit-of-work.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-454c9bcdde0b77b35e0fc994
    resource: repo://frontend/src/App.tsx
  - id: openwiki-source-58e80110e3c82917d0ff4e0c
    resource: repo://frontend/src/components/ResponsivePage.tsx
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Architecture Overview

<!-- openwiki: broken internal link [../docs/system-architecture.md] file "../docs/system-architecture.md" does not exist. Fix the href or restore the target, then delete this comment. -->
This page is the map. The component diagram and request/event/job flows are documented in detail in [`docs/system-architecture.md`](../docs/system-architecture.md); this page focuses on the rules and relationships that connect the pieces.

## Dependency direction (backend)

```
transport → app → domain ← infra
```

The domain layer has zero framework imports. `transport` (HTTP handlers, middleware, validation) depends on `app` services and `domain` types. `app` (services, DTOs, workers, bootstrap) orchestrates use cases against `domain` ports and `infra` adapters. `infra` (persistence, event bus, cache, disbursement, asynq, Zalo, email, storage) implements the `domain` ports. `domain` is the innermost ring — see ADR-001 and `architecture/domain-layer.md`.

Anything new touches this layout in order: define the entity and rule in `domain/`; declare a port in `domain/ports/`; implement the port in `infra/`; expose a service in `app/services/`; wire a handler in `transport/http/handlers/`; mount it in `app/bootstrap/routes.go`. Each step is reviewable on its own.

## Frontend role separation

```
frontend/src/
├── App.tsx                 Router + auth provider
├── pages/
│   ├── admin/              Admin role (desktop + mobile/admin)
│   ├── partner/            Partner role (project-scoped)
│   ├── adv-partner/        Advanced-partner role
│   ├── employee/           Employee role (mobile-first)
│   ├── accountant/         Accountant role
│   ├── mobile/             Mobile wrappers (admin/, partner/)
│   ├── Login.tsx / OTPLogin.tsx / ForgotPassword.tsx / ResetPassword.tsx / ZaloResetPassword.tsx
│   └── NotFound.tsx
├── components/             Role-grouped + shared UI
│   ├── ui/                 shadcn/ui primitives
│   ├── shared/             Cross-role components
│   ├── admin/ partner/ employee/  Role-grouped UI
│   ├── ResponsivePage.tsx  Desktop/mobile dual layout wrapper
│   ├── AdminSidebar.tsx / PartnerSidebar.tsx / MobileBottomNav.tsx
│   └── …feature components (attendance, advance-payment, payroll, …)
├── hooks/                  TanStack Query hooks
├── services/               API clients
├── schemas/                Zod request/response schemas
├── contexts/               Auth, theme, preferences
├── sw.ts                   PWA service worker
└── …
```

The **desktop + mobile parity rule** applies to every frontend change for every role: a feature must work in both desktop and mobile viewports of every affected role unless explicitly narrowed. The two layouts are not a fallback — `ResponsivePage.tsx` plus role-specific pages (`pages/mobile/admin/`, `pages/mobile/partner/`) ship side by side, and `MobileBottomNav.tsx` is the mobile navigation surface. Mobile breakpoints to verify: 390px (typical phone) and 320px (narrow phone / content density edge cases). Touch targets must be ≥44px.

All UI text is Vietnamese; there is no i18n layer. Strings live inline in components or in DTO tables for server-supplied labels. See `frontend/CLAUDE.md` for the full frontend conventions and `frontend/admin-views.md`, `frontend/partner-views.md`, `frontend/employee-mobile.md`, and `frontend/shared-platform.md` for role-specific detail.

## Three async surfaces

The system has three distinct asynchronous paths. They share Redis as the runtime substrate but serve different purposes.

1. **Domain events (Redis Streams)** — Used for cross-aggregate side effects. A change in one aggregate publishes an event; multiple handlers react (audit log, cache invalidation, settlement, notification). Events are produced by per-aggregate factories and published non-blockingly. The financial critical path does not depend on event delivery for correctness; events are post-commit. See ADR-004 and `integrations/event-bus.md`.

2. **asynq background jobs (Redis queues)** — Used for work that is too long, expensive, or risky to do inline: bulk transfers to a payment provider, BCC weekly import processing, attendance auto-reject sweeps, payroll report emails, recovery sweepers for stuck payments. Tasks are enqueued via the asynq client and run by handlers registered in `bootstrap/routes.go`. See ADR-005 and `integrations/background-jobs.md`.

3. **Payment provider IPN webhooks (HTTP)** — Used for asynchronous confirmation that a payment has settled, failed, or been reversed at the bank. The provider POSTs to a webhook endpoint; an `ipn:process` worker verifies the signature, looks up the wallet payment, and feeds the matching `Trigger` into the wallet state machine (`ipn_completed`, `ipn_failed`, `ipn_reversed`). IPN processing is the only path that drives wallet payments to a terminal state in production.

These three surfaces are intentionally separate so that a backlog in one (e.g., a slow 9Pay API) does not block the others. They share infrastructure (Redis, MySQL, asynq client) but not semantics.

## Transaction discipline

Multi-aggregate writes go through `domain.TransactionManager.RunInTransaction`; cache invalidation and event publish are deferred to `domain.RegisterAfterCommit` callbacks that fire only after the transaction commits. See ADR-007 and `architecture/application-services.md` for the contract and the after-commit hook APIs.

## Cross-cutting invariants

These are project-wide rules that the wiki should reflect wherever it touches the relevant subsystem:

- **Clock**: every business time value goes through `internal/pkg/clock` (Asia/Ho_Chi_Minh). Admin clock endpoints exist only in non-prod builds. See ADR-006 and `architecture/domain-layer.md`.
- **Architecture**: server is x86_64/amd64; Docker images are amd64-only. Production uses OnePay, never 9Pay. The system is provider-agnostic via ADR-003.
- **Domain purity**: no Gin, no GORM, no Redis, no asynq in `internal/domain/`. See ADR-001.
- **Cache invalidation**: after commit, never inside a transaction. See ADR-007.
- **Frontend parity**: every change covers desktop and mobile for every affected role.
- **Commit format**: `<type>(<scope>): <subject>` — no AI references; work happens on `main`.

## Where to read next

| Topic | Page |
|-------|------|
| Domain entities, ports, wallet aggregate | `architecture/domain-layer.md` |
| Application services, transaction manager, workers | `architecture/application-services.md` |
| HTTP handlers, middleware, contracts | `architecture/transport-http.md` |
| GORM repos, Redis Streams, OnePay/9Pay adapters | `architecture/infrastructure.md` |
<!-- openwiki: broken internal link [../docs/system-architecture.md] file "../docs/system-architecture.md" does not exist. Fix the href or restore the target, then delete this comment. -->
| Detailed component diagram and request flow | [`docs/system-architecture.md`](../docs/system-architecture.md) |
<!-- openwiki: broken internal link [../docs/api.md] file "../docs/api.md" does not exist. Fix the href or restore the target, then delete this comment. -->
| API route map | [`docs/api.md`](../docs/api.md) |
<!-- openwiki: broken internal link [../docs/database.md] file "../docs/database.md" does not exist. Fix the href or restore the target, then delete this comment. -->
| Database schema overview | [`docs/database.md`](../docs/database.md) |
| Decisions (ADRs) | [`docs/decisions/`](../docs/decisions/) |
| Frontend role-specific layouts | `frontend/admin-views.md`, `frontend/partner-views.md`, `frontend/employee-mobile.md`, `frontend/shared-platform.md` |
