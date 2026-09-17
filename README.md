# Payroll

Payroll management system for Vietnamese companies. Handles employee timesheets, advance payments (FlexPay), salary disbursements via OnePay, double-entry wallet/ledger accounting, attendance check-in/out with geofencing, and web push notifications.

Three roles: **Admin** (full access), **Partner** (project-scoped), **Employee** (mobile-first views). All user-facing text is in Vietnamese.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, Gin, GORM, MySQL 8, Redis, asynq |
| Frontend | React 18.3, Vite 6.4, TypeScript 5.8, TanStack Query 5, shadcn/ui, Tailwind 3.4, Zustand 5, PWA |
| Auth | JWT + Casbin RBAC (Admin / Partner / Employee) |
| Payments | OnePay (production), 9Pay (sandbox/dev) |
| Infrastructure | Docker, Docker Compose, SSH deploy |

## Repo Layout

```
payroll/
  backend/           Go API server (DDD / Clean Architecture)
    cmd/             Entry points: api-server, hashpw, seed-temp
    configs/         Casbin RBAC policy (model + CSV)
    internal/
      domain/        Entities, value objects, domain services, ports, specs, wallet aggregate
      app/           Application services, bootstrap/DI, DTOs, asynq workers
      adapters/      Anti-corruption layer for external contracts
      infra/         GORM repos, event bus (in-memory), asynq, email, disbursement adapters, storage
      pkg/            clock, db, geo, retry, httputils, scopes, validation, ...
      transport/http/ Handlers, middleware (auth, RBAC, rate-limit, security headers)
    migrations/       83 SQL migration files (.up.sql)
  frontend/          React SPA
    src/
      components/    UI library grouped by domain (admin, attendance, payroll, wallet, partner, ...)
      hooks/          TanStack Query wrappers + custom hooks (including hooks/api/)
      contexts/       AuthContext, AppStateContext, BottomNavContext, ...
      pages/          Route pages (admin, partner, employee/mobile)
      config/        API config, constants
  docs/              Documentation (this README's siblings live here)
  Makefile            Monorepo-level build, test, deploy commands
```

## Quick Start

### Prerequisites

- Go 1.26+, Node 20+, pnpm, Docker
- `.env` files in `backend/` and `frontend/` (see `.env.example` in each)

### Local Development

```bash
# Start infrastructure (MySQL + Redis + Adminer + sandbox mocks)
cd backend && make db

# Start backend (air hot-reload on :8080) and frontend (vite dev on :5173)
make dev
```

Sandbox mode serves both the 9Pay and OnePay mocks on localhost:9001. Email is captured in-process, not sent.

Fresh local MySQL volumes use `backend/scripts/init-local-db.sh`. It applies
only forward schema migrations, excludes the production-specific transaction
repair, and reconciles historical schema omissions before dependent migrations.
It refuses to run against a non-empty database. Existing local volumes are
preserved; migrations are never applied automatically to them or to production.
Run `make -C backend db-bootstrap-test` to verify a fresh schema in a separate
Docker container without published ports.

### Testing

```bash
# Backend unit tests
cd backend && go test ./... -v -race -cover

# Frontend lint + type-check
cd frontend && pnpm lint && pnpm type-check

# Integration tests (requires running backend)
make api-test
```

Use synthetic local accounts through `PAYROLL_ADMIN_USER`, `PAYROLL_ADMIN_PASS`,
`PAYROLL_PARTNER_USERS`, and `PAYROLL_COMMON_PASS`. `PAYROLL_TEST_DB_DSN` can point
repository unit tests at a separate local schema to keep their cleanup isolated.
Import fixtures can be overridden with `PAYROLL_BCC_FIXTURE` and
`PAYROLL_FLEXPAY_FIXTURE`. To run the wallet payment happy path, set
`PAYROLL_WALLET_BULK_FIXTURE` to a fresh OnePay export and
`PAYROLL_WALLET_BULK_TIMESHEET_IDS` to its comma-separated timesheet IDs. That test
requires the local provider and verifies completion, ledger linkage, recorded
payment totals, and the KQ workbook; without a fixture it reports an explicit skip.

### Deployment

```bash
make deploy          # Production (tingting.vip) — builds + pushes amd64 images, SSH deploy
make demo           # Demo (demo.tingting.vip) — separate :demo image tag + API URL
make demo-db        # Reload demo MySQL from local dev DB (does not touch images)
make backup          # Backup production DB to OneDrive
make restore         # Restore latest OneDrive backup
make adminer         # Open Adminer over SSH tunnel → localhost:18081
```

## Key Conventions

| Convention | Detail |
|-----------|--------|
| **Time** | All business logic uses `clock.Now()` (Asia/Ho_Chi_Minh). Never `time.Now()` for domain logic. |
| **Platform** | Production server is x86_64/amd64. Never build arm64 Docker images. |
| **Payment provider** | Production = OnePay; sandbox/dev = 9Pay. System is provider-agnostic. |
| **Branch** | Work on `main` directly. No worktrees or feature branches by convention. |
| **Commits** | `<type>(<scope>): <subject>` — e.g. `feat(attendance): add geofence validation` |
| **Language** | Backend: snake_case files, PascalCase types, camelCase functions. Frontend: PascalCase components, camelCase utilities. |
| **Events** | Domain events via `EventBus.Publish()`, non-blocking goroutines. |
| **Database** | GORM for all access. Repository pattern. Domain types in handlers, not GORM models. |

## Documentation

| Document | Description |
|----------|-------------|
| [Project Overview & PDR](docs/project-overview-pdr.md) | Purpose, roles, functional scope, domain glossary |
| [Codebase Summary](docs/codebase-summary.md) | LOC by area, layer map, "where do I find X" index |
| [Code Standards](docs/code-standards.md) | Backend/frontend/shared conventions and patterns |
| [System Architecture](docs/system-architecture.md) | Component diagrams, request/event/job flows |
| [Project Roadmap](docs/project-roadmap.md) | Recent themes and inferred direction |
| [Deployment Guide](docs/deployment-guide.md) | Targets, Docker build, migration, backup/restore |
