---
type: quickstart
title: Quickstart
description: Repository map, build/dev commands, Makefile targets, dual Docker Compose layouts, and the production invariants a new agent needs in the first five minutes.
tags: [quickstart, repo-map, makefile, docker, dev, deploy]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-d90fca79839a07479341ddb0
    resource: repo://backend/Makefile
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-2e31c9c6e4f71c40fd02e375
    resource: repo://frontend/Makefile
  - id: openwiki-source-012f2c78e3b1446dfc35803f
    resource: repo://Makefile
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Quickstart

The repository is a monorepo: a Go backend (`backend/`) and a React frontend (`frontend/`), with shared docs at the root (`docs/`). This page is the map; the deep dives live under `architecture/`, `features/`, `integrations/`, `operations/`, and `testing/`.

## Directory map

```
payroll/
├── backend/             Go API server (DDD / Clean Architecture)
├── frontend/            React 18 + Vite + TanStack Query + shadcn/ui (PWA)
├── docs/                Evergreen reference docs (architecture, API, ADRs, lessons)
├── openwiki/            This generated wiki
├── plans/               Timestamped plans and reports
├── scripts/             Local utility scripts
├── testplan/            Structured E2E scenarios
├── qa/                  Project-local QA artifacts
├── Makefile             Top-level build / test / deploy / restore
├── docker-compose.yml        Production-like stack
├── docker-compose.dev.yml    Dev stack (mysql + redis + adminer + provider mocks)
├── Dockerfile, backend/Dockerfile, frontend/Dockerfile
└── AGENTS.md, CLAUDE.md       Repo-root AI agent context
```

The per-directory `AGENTS.md` files are the authoritative entry points for each area:

- `AGENTS.md` (root) — repo overview, testing requirements, common patterns.
- `docs/AGENTS.md` — evergreen doc taxonomy.
- `backend/AGENTS.md` and `backend/internal/*/AGENTS.md` — backend conventions per layer.
- `frontend/AGENTS.md` and `frontend/src/*/AGENTS.md` — frontend conventions per role/area.
- `backend/migrations/AGENTS.md` — migration discipline.

## Makefile targets

| Target | What it does |
|--------|-------------|
| `make dev` | Starts backend (air hot-reload) and frontend (vite) in parallel |
| `make api-test` | Runs backend integration suite against the test DB |
| `make api-lint` | Runs golangci-lint on the backend |
| `make api-build` | Builds the api-server binary |
| `make deploy` | SSH deploys to production (build + migrate + restart) |
| `make restore` | Restore production DB from the latest backup |
| `make backup` | Create a fresh production DB backup |

`backend/Makefile` and `frontend/Makefile` carry the sub-area targets (pnpm scripts, single-test runs, etc.).

## Dual Docker Compose

| File | Purpose |
|------|---------|
| `docker-compose.dev.yml` | Local dev: MySQL + Redis + Adminer + 9Pay mock (9001) + OnePay mock (9002). The backend runs OUTSIDE Docker via `air`; the frontend runs via `vite`. |
| `docker-compose.yml` | Production-like: frontend (nginx) + backend (api-server + asynq) + MySQL + Redis + nginx (reverse proxy) + on-demand adminer. |

The dev compose never builds the backend image. The prod compose is the artifact `make deploy` ships.

## Production invariants

These are non-negotiable:

- **Server architecture is x86_64/amd64** — never arm64 images.
- **Production uses OnePay** (not 9Pay); the system is provider-agnostic via ADR-003 but the prod registry contains only OnePay.
- **All business time uses `clock.Now()` (Asia/Ho_Chi_Minh)** — never `time.Now()` for domain logic (ADR-006).
- **Domain layer has zero framework imports** (ADR-001).
- **Cache invalidation happens AFTER transaction commit** (ADR-007).
- **Work on `main` directly** — no worktrees.
- **Commit format** `<type>(<scope>): <subject>` — no AI references.

## Five-minute start

```bash
git clone <repo>
cd payroll
docker compose -f docker-compose.dev.yml up -d
make dev                 # backend on :8080, frontend on :5173
make api-test            # integration suite against the test DB
```

For a deeper map of "where is X", read [`docs/codebase-summary.md`](../docs/codebase-summary.md). For system architecture, read `architecture/overview.md` (it links to all the deeper pages).

## Where to read next

- Architecture map — `architecture/overview.md`.
- Local development details — `operations/local-dev.md`.
- Deploy + restore — `operations/deploy-restore.md`.
- Feature deep dives — `features/`.
- Backend layer reference — `architecture/domain-layer.md`, `architecture/application-services.md`, `architecture/transport-http.md`, `architecture/infrastructure.md`.
