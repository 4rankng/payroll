---
type: operations
title: Local Development
description: Bring-up the dev stack — docker-compose.dev.yml (MySQL + Redis + adminer + 9Pay mock + OnePay mock), air hot-reload for the backend, vite for the frontend, ports, and admin tooling.
tags: [operations, local-dev, docker-compose, air, vite, adminer, ports]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-410ab36600fefdde765cb24e
    resource: repo://backend/AGENTS.md
  - id: openwiki-source-aca61ad26f85a3f6b9e0d210
    resource: repo://docs/decisions/ADR-006-clock-injection-pattern.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Operations: Local Development

The local stack runs MySQL + Redis + Adminer + the 9Pay and OnePay mocks via `docker-compose.dev.yml`. The backend runs outside Docker via `air` (hot reload) so a code change reflects in a few seconds. The frontend runs via Vite. No real provider keys are needed — the mocks cover the full IPN flow.

## Ports

| Service | Port | Source |
|---------|------|--------|
| Backend (air) | 8080 | `backend/.air.toml` |
| Frontend (vite) | 5173 | `frontend/vite.config.ts` |
| MySQL | 3306 | `docker-compose.dev.yml` |
| Redis | 6379 | `docker-compose.dev.yml` |
| Adminer | 8081 | `docker-compose.dev.yml` |
| 9Pay mock | 9001 | `docker-compose.dev.yml` |
| OnePay mock | 9002 | `docker-compose.dev.yml` |

## Bring-up

From the repo root:

```bash
docker compose -f docker-compose.dev.yml up -d
make dev          # starts backend (air) and frontend (vite) in parallel
```

`make dev` runs the two dev processes in parallel using the project's task runner. Stop with `Ctrl-C`; tear down DB+Redis with `docker compose -f docker-compose.dev.yml down`.

## Backend

- `air` watches `backend/` and rebuilds on change. Configured by `backend/.air.toml`.
- Hot reload is the canonical dev path — `backend/AGENTS.md` says "never run the backend directly".
- Migrations apply via `cd backend && go run cmd/migrate/main.go up`.
- Tests: `cd backend && go test ./... -v -race -cover` (unit), `make api-test` (integration, requires the dev DB).

## Frontend

- `pnpm dev` (Vite) for hot reload at `http://localhost:5173`.
- The Axios client defaults to `http://localhost:8080/api/v1` in dev.
- The PWA service worker registers on the first load.

## Admin tooling

- **Adminer** at `http://localhost:8081` — DB browser for the dev MySQL.
- `backend/adminer.sh` — convenience wrapper that opens Adminer with the right credentials.
- System health and cron health at `/admin/system-health` and `/admin/cron-health` consume the dev api-server's metrics.

## Env files

- `backend/.env.example` — non-secret defaults. Copy to `.env` for the backend.
- `frontend/.env.example` — same for the frontend.
- Real secrets never go in the example files; production is its own `.env`.

## Test data

The dev DB is reseeded by `cd backend && go run cmd/seed/main.go` (if present) or by manual fixtures in the integration tests. The BCC import tests reference example workbooks under `docs/WeeklyBCC/`.

## Conventions enforced locally

- Branch: `main`. No worktrees.
- Commit format: `<type>(<scope>): <subject>`.
- `make api-test` after any feature change.
- `pnpm lint && pnpm type-check` after any frontend change.
- Clock endpoints (`/api/v1/admin/clock/*`) are mounted in dev so integration tests can advance time. They are NOT mounted in production.

## Relationships

- Deploy reference — `operations/deploy-restore.md`.
- Quickstart — `quickstart.md`.
- Integration tests — `testing/integration.md`.
