# Deployment Guide

Targets, Docker build constraints, database migrations, and backup/restore procedures. All commands are verified from the root and backend Makefiles.

## Deployment Targets

| Target | Host | Image Tag | Notes |
|--------|------|-----------|-------|
| Production | `tingting.vip` | `:latest` | x86_64/amd64 only |
| Demo | `demo.tingting.vip` | `:demo` | 1GB droplet, requires 2GB swap |

Both targets use SSH deploy: images are built/pushed to **GHCR** (`ghcr.io/4rankng/...`), then pulled on the server via `docker compose`.

## Common Prerequisites

- GHCR credentials in `~/.zshrc` (`GHCR_TOKEN` = a classic PAT with `write:packages`) and `GHCR_OWNER` in `.env`
- Base images vendored into GHCR via `make mirror-bases` (run once per base-image bump; pinned by digest in the Dockerfiles)
- SSH access to target servers (root)
- Production DB credentials in `.env` (`MYSQL_ROOT_PASSWORD_PROD`, `DB_DSN`, etc.)
- `make` available locally

## Production Deploy

```bash
make deploy
```

This runs (from root Makefile):
1. `cd frontend && make push` -- build + push amd64 frontend image (`:latest`)
2. `cd backend && make push` -- build + push amd64 backend image (`:latest`, plus `:GIT_SHA` tag)
3. `cd backend && make deploy` -- SSH to `tingting.vip`, pull images, recreate containers

On the server:
```bash
ssh root@tingting.vip
cd /opt/payroll && docker compose pull frontend backend
cd /opt/payroll && docker compose up -d --force-recreate --no-deps frontend backend
```

## Demo Deploy

```bash
make demo
```

This runs:
1. `cd frontend && make push-demo` -- build + push amd64 frontend image (`:demo`), API URL = `https://demo.tingting.vip`
2. `cd backend && make push-demo` -- build + push amd64 backend image (`:demo`)
3. `cd backend && make deploy-demo` -- SSH to `demo.tingting.vip`, pull `:demo` images, recreate containers

Separate image tags keep demo independent of production.

## Demo Database Reload

```bash
make demo-db
```

Dumps local dev MySQL and restores to demo server. Also resets all demo user passwords to `Admin123`.

**This does NOT touch demo Docker images** -- only the database contents.

## Docker Build Constraints

### Platform: amd64 only

Both frontend and backend images are built with `--platform linux/amd64`. Production server is x86_64. Never build arm64.

Backend push (from backend Makefile):
```bash
docker buildx build --platform linux/amd64 --tag <image> --push .
```

Frontend push (from frontend Makefile):
```bash
docker buildx build --platform linux/amd64 --tag <image> --push .
```

### Build Cache

- **Backend**: Uses `--cache-from type=registry,ref=<image>:latest` (registry manifest cache)
- **Frontend**: Uses `--cache-from type=local,src=$BUILD_CACHE_DIR` (local buildx cache at `~/.cache/payroll-frontend-buildx/`)

The frontend uses local cache to avoid the ~100s registry manifest round-trip that occurs with registry-based cache on a docker-container buildx driver.

### Buildx Builder

Both frontend and backend share a persistent docker-container buildx builder named `bb` (configurable via `BUILDER` env var). Created automatically on first use.

### Dual Lockfile Pitfall

Frontend has both `yarn.lock` (used by Docker build) and `pnpm-lock.yaml` (used by local dev). Running `pnpm add` locally updates `pnpm-lock.yaml` but leaves `yarn.lock` stale, which breaks `make demo` / `make deploy`. To regenerate `yarn.lock`, work in an isolated temp directory to avoid `node_modules` conflicts.

## Database Migrations

103 SQL migration files in `backend/migrations/` (`.up.sql` naming; the latest is migration 103).

### Applying Migrations to Demo/Prod

When a new migration references columns that the deployed backend code expects, the migration must be applied **before** the backend deploy. Migration 103 also backfills loan repayment allocations and repairs related loan aggregates and ledger entries, so apply the complete migration before deploying the matching backend. It refuses to guess where a loan's existing schedule total is lower than its principal; repair those schedules first, then rerun the migration. Two approaches:

1. **Via `make demo-db`**: Dumps local dev DB (which already has migrations applied) and restores to demo. Works when local dev is up to date.

2. **Direct DML application** (for single migrations): Apply the `.up.sql` file directly on the server:
```bash
# Extract password from server DB_DSN
sed -n 's/.*:\(.*\)@.*/\1/p' /opt/payroll/.env

# Apply migration via docker exec
cat migration_name.up.sql | ssh root@demo.tingting.vip \
  "docker exec -i payroll-mysql mysql -u root -p'<password>' payroll_db"
```

Note: `schema_migrations` table has been reported empty on some deployments (dump-seeded environments). Manual DDL application does not require a version row.

### Local Development Migrations

The development workflow applies migrations via the local dev database. `make db` starts MySQL + Redis containers. Migrations are typically applied as part of the local DB setup.

## Backup & Restore

### Backup Production DB to OneDrive

```bash
make backup
```

Flow: SSH to production -> `mysqldump` via docker exec -> gzip -> download to local OneDrive path (`~/Library/CloudStorage/OneDrive-Personal/backup/payroll_mysql_backup/`). File named `payroll_mysql_backup_YYYY-MM-DD_HHMMSS.sql.gz`.

### Restore Latest Backup to Local Dev

```bash
make restore
```

Flow: Finds latest `.sql.gz` in OneDrive backup dir -> drops local `payroll_db` -> recreates -> pipes decompressed dump into local MySQL container.

**Warning:** This completely replaces the local development database with the production backup.

## Adminer (Database UI)

```bash
make adminer
```

Opens Adminer over SSH tunnel at `http://localhost:18081`. No public exposure -- the tunnel forwards localhost:18081 to the production server's loopback. Connection details:
- Server: `payroll-mysql`
- Credentials: from `DB_DSN` in `/opt/payroll/.env` on the production server

Press Ctrl-C to close the tunnel.

## Local Development Setup

```bash
# 1. Start infrastructure (MySQL + Redis + Adminer + sandbox mocks)
cd backend && make db
# Services: MySQL :3306, Redis :6379, Adminer :8081, 9Pay mock :9001, OnePay mock :9002

# 2. Copy and configure .env files
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env

# 3. Start both backend and frontend
make dev
# Backend: air hot-reload on :8080
# Frontend: vite dev on :5173 (started via yarn dev)
```

## Environment Files

| File | Location | Purpose |
|------|----------|---------|
| `.env` | `backend/` | Backend config: DB_DSN, Redis URL, JWT secret, payment provider keys, GHCR_OWNER |
| `.env` | `frontend/` | Frontend build args: API base URL, Google Client ID |
| `.env` | `/opt/payroll/` (prod server) | Production runtime config |
| `.env.example` | `backend/`, `frontend/` | Template with all required variables |
