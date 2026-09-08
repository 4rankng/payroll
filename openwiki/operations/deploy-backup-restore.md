---
type: operations
title: Deploy, Backup, and Restore
description: Make targets, Docker Compose topology for dev and production, MySQL backup and restore flow, and the production x86_64 invariant.
tags: [deploy, docker, backup, restore, runbooks, x86_64]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-08T09:18:16.270Z
sources:
  - id: openwiki-source-7865fb2b5570e6ebb2f50ca8
    resource: repo://docs/deployment-guide.md
  - id: openwiki-source-012f2c78e3b1446dfc35803f
    resource: repo://Makefile
generated: { by: "opencode", at: "2026-09-08T09:18:16.270Z" }
---

# Deploy, Backup, and Restore

The deploy pipeline targets two hosts: production (`tingting.vip`) and a now-retired demo host (`demo.tingting.vip`, served from the last build before the demo targets were removed). All commands are verified from the root and backend Makefiles. The full step-by-step lives in `docs/deployment-guide.md`; this page is the operator's quick map.

## Production invariants

- **x86_64/amd64 only.** Production images are built and pushed as `linux/amd64`. Never publish an arm64 image to the `:latest` tag; the production host cannot run it. The base images are mirrored into GHCR via `make mirror-bases` and pinned by digest in the Dockerfiles so platform drift cannot sneak in.
- **Production uses OnePay.** Sandbox/development use 9Pay. The active disbursement provider is selected at runtime from `Settings.disbursement_provider`; flipping this in production is an out-of-band incident.
- **All business time is `clock.Now()` (Asia/Ho_Chi_Minh).** Never override the timezone on the production container — the seed data and the historical timesheets assume the canonical TZ.

## Deploy

`make deploy` runs three phases:

1. **Frontend build & push** — `cd frontend && make push` builds the Vite production bundle into a multi-stage `linux/amd64` image and pushes it to GHCR as `:latest`.
2. **Backend build & push** — `cd backend && make push` does the same for the Go binary, tagging both `:latest` and `:GIT_SHA` so the running image is always pinned to a commit.
3. **SSH deploy** — `cd backend && make deploy` SSHs to the production host, pulls the new images, and recreates the `frontend` and `backend` containers with `--force-recreate --no-deps`. MySQL and Redis are not touched.

On the server the equivalent commands are:

```bash
ssh root@tingting.vip
cd /opt/payroll && docker compose pull frontend backend
cd /opt/payroll && docker compose up -d --force-recreate --no-deps frontend backend
```

Database migrations run as a separate step before the backend container starts (see `docs/runbooks/`). Migrations are forward-only and the migration runner is idempotent for replay.

GHCR auth uses `scripts/ghcr-token.sh` which tries `GHCR_TOKEN` → Docker keychain → legacy `~/.zshrc` export in that order, validating each candidate against ghcr.io before accepting it. A rejected candidate is skipped with a warning rather than failing the login — this matters when an old export in `~/.zshrc` is stale and silently invalid.

## Docker Compose topology

| Compose file | Purpose | Components |
|---|---|---|
| `docker-compose.yml` | Production stack | `api-server`, MySQL 8, Redis, nginx reverse proxy |
| `docker-compose.dev.yml` | Local development | MySQL 8, Redis, Adminer (backend runs on host via `air` hot-reload) |

Production runs api-server, MySQL, and Redis behind nginx. Local development keeps the dependencies containerized but runs the backend on the host through `air` so edit-reload cycles are fast. The `mysql_data/` and `redis_data/` directories are gitignored and mounted as Docker volumes.

## Local development

`make dev` starts the local stack — MySQL and Redis from `docker-compose.dev.yml`, the Go backend on `:8080` via `air`, the Vite dev server on `:5173`. The frontend reads `VITE_API_BASE_URL` to talk to the local backend.

`make sandbox` builds and runs the full stack containerized for offline testing. `make adminer` opens Adminer over an SSH tunnel at `http://localhost:18081` (no internet exposure, no UFW 8081).

## Backup

`make backup` runs the backend's `make backup` target. Backups go to OneDrive (mounted or synced on the production host). The dump is a logical MySQL dump plus a snapshot of the uploaded `assets/` directory so proof files survive a restore.

Run backups on a schedule (cron on the production host), and confirm the most recent backup is fresh before any high-risk change (schema migration, bulk import re-run, deploy after a long outage).

## Restore

`make restore` runs the backend's `make restore` target. The runbook restores the latest backup from OneDrive:

1. Stop the api-server container so writes do not race the restore.
2. Apply the logical MySQL dump to the running MySQL volume.
3. Restore the `assets/` directory if it changed.
4. Run migrations forward to the current head — the dump is from a specific migration set; the migration runner advances the schema if needed.
5. Start the api-server container.
6. Run `DELETE /api/v1/cache` (admin-only) so cached aggregates do not serve pre-restore state.

The restore runbook is `docs/runbooks/`. The wallet-bulk-transfer-stuck-completing recovery runbook handles a specific operational edge case where a bulk transfer's asynq task stops mid-flight; it documents how to resume without double-paying.

## Operational runbooks

- `docs/runbooks/wallet-bulk-transfer-stuck-completing-recovery.md` — diagnose and recover a bulk transfer whose task entered `completing` but did not finalize.
- `docs/runbooks/zalo-oa-connect.md` — connect a Zalo Official Account for the ZNS password-reset flow. The admin UI at `/admin/settings?tab=zalo` is the operator surface; env vars are bootstrap-only seed.
- `docs/troubleshooting.md` — symptom → cause → fix for recurring operational issues. Check here first when something looks wrong in production.

## Smoke checks after deploy

After `make deploy` completes, verify:

- Backend health endpoint: `curl https://tingting.vip/api/v1/health` returns 200 with dependency statuses green.
- Login flow works for an admin and a partner account.
- A read-heavy page (Dashboard, Wallet) loads without 5xx — this exercises the cache and the transaction manager.
- The most recent backup is fresh and restorable in a staging slot.

## Related pages

- [Architecture Overview](../architecture/overview.md) — what is being deployed.
- [Auth, RBAC, and Casbin](./auth-rbac-and-casbin.md) — what role gates are live after deploy.
- [Testing: Integration and Playwright](../testing/integration-and-playwright.md) — `make api-test` is the regression gate before any deploy.
