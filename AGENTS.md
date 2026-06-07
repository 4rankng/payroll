<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# Payroll

## Purpose
Payroll management system for Vietnamese construction companies. Monorepo with a Go backend (DDD / Clean Architecture) and a React frontend (Vite + TanStack Query + shadcn/ui). Handles employee timesheets, advance payments (FlexPay), salary disbursements via OnePay/9Pay, double-entry wallet/ledger accounting, and web push notifications. Three user roles: Admin, Partner, Employee (with mobile-first views).

## Key Files
| File | Description |
|------|-------------|
| `CLAUDE.md` | Project-level instructions, integration test patterns, clock system, cache invalidation rules |
| `Makefile` | Build, test, deploy, and restore commands for the monorepo |
| `docker-compose.yml` | Production Docker Compose (api-server + MySQL + Redis + nginx) |
| `docker-compose.dev.yml` | Development Docker Compose (MySQL + Redis only, backend runs via air) |
| `HANDOFF.md` | Session handoff state (gitignored, auto-updated on session end) |

## Subdirectories
| Directory | Purpose |
|-----------|---------|
| `backend/` | Go API server — DDD with domain, app, infra, and transport layers (see `backend/AGENTS.md`) |
| `frontend/` | React 18 SPA — Vite 6, TanStack Query, shadcn/ui, PWA (see `frontend/AGENTS.md`) |
| `docs/` | Project documentation, ADRs, workflow specs, review notes (see `docs/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- Run `make api-test` after every feature change to catch regressions
- All business time uses `clock.Now()` (Asia/Ho_Chi_Minh) — never `time.Now()` for domain logic
- Production server is **x86_64/amd64** — never build arm64 Docker images
- Production uses **OnePay** (NOT 9Pay); system is provider-agnostic
- Work on main branch directly, no worktrees
- Commit format: `<type>(<scope>): <subject>` — e.g. `feat(timesheet): add bulk export`
- `make dev` starts backend (air hot-reload on :8080) and frontend (vite dev on :5173)
- `make deploy` builds and deploys to production server via SSH

### Testing Requirements
- `make api-test` — integration tests against live backend
- Backend unit tests: `cd backend && go test ./... -v -race -cover`
- Frontend: `cd frontend && pnpm lint && pnpm type-check`
- Update `backend/tests/integration` with test scenarios for new features

### Common Patterns
- Backend: Domain events via `EventBus.Publish()`, non-blocking goroutines, outbox pattern
- Frontend: TanStack Query for data fetching, optimistic updates, Vietnamese UI text
- Both: Centralized clock, Redis caching with invalidation after commit, asynq background jobs

## Dependencies

### Internal
- `backend/internal/domain/` — core domain entities and business rules
- `backend/internal/app/services/` — application service layer
- `frontend/src/components/` — React component library
- `frontend/src/hooks/api/` — TanStack Query hooks wrapping backend APIs

### External
- Go 1.22+ — backend language
- Node 20+ / pnpm — frontend toolchain
- MySQL 8 — primary database
- Redis — caching, event bus, background jobs (asynq)
- Docker / Docker Compose — local development and deployment
- OnePay — production payment disbursement provider
- 9Pay — sandbox/dev payment provider

<!-- MANUAL: Custom project notes can be added below -->

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
