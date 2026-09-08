---
type: "Reference"
title: "Quickstart"
openwiki_generated: true
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-0c60b8722ffc081c51df87e3
    resource: repo://backend/internal/AGENTS.md
  - id: openwiki-source-50c7d39e20d2fbd3fc8c2e0c
    resource: repo://backend/internal/domain/transactions/state_machine.go
  - id: openwiki-source-4691894e593eb5f2f5eb82b7
    resource: repo://frontend/src/AGENTS.md
  - id: openwiki-source-454c9bcdde0b77b35e0fc994
    resource: repo://frontend/src/App.tsx
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---


# Quickstart

Payroll is a Vietnamese payroll management monorepo. A Go backend (DDD / Clean Architecture) speaks HTTP/JSON to a React 18 + Vite SPA that runs Admin, Partner, and Employee surfaces across desktop and mobile. Timesheets feed FlexPay draws and bulk-transfer disbursements; a wallet aggregate records funds; a double-entry ledger is the accounting system of record.

For the visual map of components, request flow, and infrastructure topology, see [`docs/system-architecture.md`](../docs/system-architecture.md). For "where do I find X", see [`docs/codebase-summary.md`](../docs/codebase-summary.md). This page is the 60-second tour.

## Top-level layout

| Area | What lives here | When to start here |
|---|---|---|
| `backend/` | Go API server (DDD layered), MySQL migrations, asynq workers, integration tests | Backend changes — entity, service, handler, migration |
| `frontend/` | React 18 + Vite SPA, shadcn/ui components, TanStack Query, Playwright E2E | Frontend changes — pages, components, hooks |
| `docs/` | Evergreen reference docs, ADRs, lessons, runbooks, standards | Architectural questions, decisions, incident postmortems |
| `openwiki/` | Generated code wiki (this directory) | What does this codebase do, where, why |
| `plans/` | Timestamped plans, phase files, reports | Active multi-phase work |
| `Makefile` | Top-level build, dev, test, deploy targets | Local commands |

The project front door is `README.md`. Per-area agent context lives in `CLAUDE.md` and `AGENTS.md` at the root, `backend/`, and `frontend/`.

## Backend at a glance

`backend/internal/` is the layered source tree:

- `domain/` — entities, value objects, domain services, ports, specs, wallet aggregate. Zero framework imports. The transactional wallet state machine lives at `domain/transactions/state_machine.go`.
- `app/` — `bootstrap/` (DI container and routes), `services/` (use-case orchestration, 26+ service packages), `dto/`, `workers/` (asynq task handlers).
- `transport/http/` — handlers grouped by domain, middleware (auth → Casbin RBAC → audit context → rate-limit), response helpers, validation.
- `infra/` — `persistence/` (GORM repositories), `events/` (Redis Streams event bus + handlers), `disbursement/` (OnePay/9Pay adapters), `email/`, `storage/`, `cache/`, `asynq/`, `transaction/` (unit-of-work), `observability/`.
- `pkg/clock` — centralized business clock. All business time goes through `clock.Now()` (Asia/Ho_Chi_Minh).
- `migrations/` — 80+ SQL files, applied by `go run cmd/migrate/main.go up`. Forward-only by convention; recovery scripts are idempotent.

Start there: `backend/internal/domain/AGENTS.md` for the domain layer conventions, `backend/internal/app/AGENTS.md` for the application layer, and `backend/AGENTS.md` for backend-wide rules.

## Frontend at a glance

`frontend/src/` is the SPA source tree:

- `App.tsx` — root component with route definitions and lazy-loaded pages.
- `pages/` — page-level components split by role and viewport:
  - `admin/`, `partner/`, `adv-partner/`, `accountant/`, `employee/` — desktop.
  - `mobile/admin/`, `mobile/partner/`, `mobile/` — mobile variants.
  - `Login`, `OTPLogin`, `ForgotPassword`, `ResetPassword`, `ZaloResetPassword` — auth flows.
- `layouts/` — `AdminLayout`, `PartnerLayout` for desktop chrome.
- `components/` — `ui/` (shadcn primitives) and feature components.
- `hooks/api/` — TanStack Query hooks for data fetching.
- `services/api/` — Axios client and per-domain API modules.
- `lib/`, `utils/`, `constants/`, `contexts/`, `schemas/`, `types/` — supporting concerns.
- `sw.ts` — PWA service worker with push notification handling.

Start there: `frontend/src/App.tsx` for routing, `frontend/src/AGENTS.md` for the file responsibility rule (.tsx for UI only, .ts for logic), and `frontend/AGENTS.md` for desktop/mobile parity.

## Non-negotiable invariants

These apply to every change. They appear as Claims on the wiki pages that touch the relevant subsystem.

| Invariant | Why |
|---|---|
| All business time uses `clock.Now()` (Asia/Ho_Chi_Minh) — never `time.Now()` for domain logic. | Timezone correctness across MySQL, asynq, and the wallet FSM. ADR-006. |
| Production server is x86_64/amd64 — never arm64 Docker images. | The production droplet cannot run arm64. |
| Production uses OnePay (NOT 9Pay). System is provider-agnostic via ADR-003. | OnePay is the production disbursement provider. |
| Domain layer has zero framework imports — no Gin, no GORM in `internal/domain/`. | Keeps the domain testable in isolation. ADR-001. |
| Cache invalidation happens AFTER transaction commit — never before. | Prevents phantom-cache state on rollback. ADR-007. |
| Commit format `<type>(<scope>): <subject>` — no AI references. | Project convention for log hygiene. |
| Work on `main` branch directly — no worktrees or feature branches. | Project convention. |
| Vietnamese for all user-facing text — strings inlined in components. | No i18n layer. |
| Desktop and mobile are one delivery scope for every frontend change. | Per-role parity is enforced in code review. |

## Local commands

From the repo root:

```bash
make dev              # MySQL + Redis (docker-compose) + backend (air on :8080) + frontend (vite on :5173)
make api-test         # backend integration tests + go test ./... -race -cover
make backup           # logical MySQL dump to OneDrive
make restore          # restore latest backup
make deploy           # build + push amd64 images, SSH deploy to production
```

From `frontend/`:

```bash
pnpm install
pnpm dev              # dev server at :5173
pnpm lint
pnpm type-check
pnpm test:e2e         # Playwright
```

From `backend/`:

```bash
make lint
go test ./... -v -race -cover
```

## Where to go next

- New to the backend → [Architecture Overview](./architecture/overview.md) for the layered model and request lifecycle.
- Touching the ledger or wallet → [Double-Entry Ledger and Chart of Accounts](./architecture/double-entry-ledger.md).
- Adding a mutation that must invalidate caches or publish events → [Transaction Manager and Outbox](./architecture/transaction-manager-and-outbox.md).
- Touching auth or roles → [Auth, RBAC, and Casbin](./operations/auth-rbac-and-casbin.md).
- Frontend page change → [Frontend Role and View Architecture](./frontend/role-and-view-architecture.md) first.
- Shipping to production → [Deploy, Backup, and Restore](./operations/deploy-backup-restore.md).
- Adding a new test scenario → [Testing: Integration and Playwright](./testing/integration-and-playwright.md).
