---
type: operations
title: Deploy, Backup & Restore
description: Production deploy via make deploy, x86_64/amd64 architecture constraint, the docker-compose layout (api-server + MySQL + Redis + nginx), migration discipline, and the backup/restore flow.
tags: [operations, deploy, restore, backup, docker, production, amd64]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-8037e2358a2c4f9b2c722a11
    resource: repo://AGENTS.md
  - id: openwiki-source-0f104d87e52630bb474cdbe0
    resource: repo://backend/internal/domain/wallet/repository.go
  - id: openwiki-source-156ad5a7f27e5758f4a19efd
    resource: repo://backend/migrations/AGENTS.md
  - id: openwiki-source-7865fb2b5570e6ebb2f50ca8
    resource: repo://docs/deployment-guide.md
  - id: openwiki-source-62317b515c31ac5b3e190eb4
    resource: repo://docs/system-architecture.md
  - id: openwiki-source-012f2c78e3b1446dfc35803f
    resource: repo://Makefile
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Operations: Deploy, Backup & Restore

<!-- openwiki: broken internal link [../docs/deployment-guide.md] file "../docs/deployment-guide.md" does not exist. Fix the href or restore the target, then delete this comment. -->
Production deploys the full stack behind nginx on a single x86_64 droplet at `/opt/payroll/`. The deploy pipeline is `make deploy` and the architecture constraint is non-negotiable: **x86_64/amd64 only, never arm64**. The full deployment doc lives in [`docs/deployment-guide.md`](../docs/deployment-guide.md); this page is the focused operations summary.

## Architecture invariant

The production server is `x86_64/amd64`. The Dockerfiles and the docker-compose layout are pinned to amd64. Building arm64 images breaks Redis container compatibility and provider SDK signing paths. Code review rejects any PR that introduces arm64 conditions.

## Topology

```
/opt/payroll/
├── docker-compose.yml         # production-like stack
├── .env                        # secrets (gitignored)
└── Services:
    ├── frontend  (nginx serving the React SPA)
    ├── backend   (Go api-server + embedded asynq worker)
    ├── mysql     (8.x)
    ├── redis
    ├── nginx     (reverse proxy, SSL termination)
    └── adminer   (on-demand, SSH-tunnel only)
```

The demo target (`demo.tingting.vip`) is the same topology on a smaller droplet with `:demo` image tags and a different API URL.

## Deploy pipeline

`make deploy`:

1. SSH to the production host.
2. Pull the latest image tags.
3. Run pending migrations via `go run cmd/migrate/main.go up` (or via the api-server bootstrap).
4. Stop, remove, and re-create the backend container (zero-downtime only if multiple instances; the single-droplet setup tolerates a brief restart).
5. Rebuild and restart the frontend container.
6. Run a smoke check against `/api/v1/health` (or the equivalent readiness endpoint).

`make api-test` runs the integration suite against the running production DB before deploy. `make api-lint` and `make api-build` validate before any container is rebuilt.

## Migration discipline

- Migrations are raw SQL under `backend/migrations/`; the convention is forward-only with idempotent recovery scripts (`backend/migrations/AGENTS.md`).
- A deploy that changes the schema runs the matching migration first; the api-server is restarted only after the schema is current.
- Down migrations exist for some files but not all — assume no downgrade without testing.

## Backup and restore

- **Backup:** periodic MySQL logical dumps via `mysqldump` (or the equivalent) into the host's volume. Frequency is configured at the host level.
- **Restore:** drop the schema, replay the dump, replay any incremental WAL/binlog. Test restore on a clone of production every quarter.
- **Wallet state is part of the backup contract** because IPN records, wallet_payments, and ledger entries are the audit trail. Restoring from a stale backup can produce a replay loop with the payment provider; the IPN handler must be ready to deduplicate via `wallet_ipn.invoice_no` (unique by provider).
- **Redis is is rebuildable from the source of truth (MySQL).** It holds cache + queues + streams + rate-limit counters. After a Redis loss, the system recovers on first use.

## Configuration and secrets

- `.env` carries `ONPAY_*`, `NINEPAY_*`, `ZALO_*` (seed only), JWT signing keys, DB credentials. It is gitignored and never committed. Restore uses the same secret values; rotating the JWT signing key invalidates every issued session.
- `PASSWORD_RESET_ENABLE`, `DISBURSEMENT_FEE_FAIL_OPEN`, `ENABLE_NINEPAY` are the runtime switches that gate features without a redeploy.

## Production-side observability hooks

- Structured logging via slog → log aggregator.
- Prometheus metrics scraped from the api-server `/metrics` endpoint.
- System health and cron health surfaces in the admin UI consume these signals.
- The OpenAI-compatible or Anthropic proxy configuration lives in `~/.openwiki/.env` and is local to the operator's environment, not on the production server.

## Where to read next

<!-- openwiki: broken internal link [../docs/deployment-guide.md] file "../docs/deployment-guide.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- Full deploy reference — [`docs/deployment-guide.md`](../docs/deployment-guide.md).
- Local development — `operations/local-dev.md`.
- Observability — `operations/observability.md`.
- Migrations discipline — `migrations/schema-evolution.md` and `backend/migrations/AGENTS.md`.
- Payment providers — `integrations/payment-providers.md` (the runtime gate env vars).
