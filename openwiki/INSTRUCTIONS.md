# OpenWiki Instructions — Payroll

This file is read by OpenWiki on every init/update run. Use it to scope and
prioritize what the wiki documents. OpenWiki never rewrites this file during
normal runs.

## What this project is

Payroll management system for Vietnamese companies. Monorepo with a Go backend
(DDD / Clean Architecture) and a React frontend (Vite + TanStack Query /
shadcn/ui). Handles employee timesheets, advance payments (FlexPay), salary
disbursements via OnePay/9Pay, double-entry wallet/ledger accounting,
attendance with geofencing, and web push notifications. Three roles: Admin,
Partner (project-scoped), Employee (mobile-first). All user-facing text is
Vietnamese.

## What to document

The wiki should explain the systems that future agents need to reason about.
Cover the meaningful subsystems, not a directory inventory.

Priority areas:

1. **Domain layer** (`backend/internal/domain/`) — entities, value objects,
   domain services, specs, wallet aggregate. Pure Go, no framework imports.
2. **Application services** (`backend/internal/app/`) — use cases, DTOs,
   event-driven workflows (transaction-manager + outbox), asynq workers.
3. **Transport / HTTP layer** (`backend/internal/transport/http/`) — handlers,
   middleware (auth, RBAC, rate-limit, security headers), request/response
   contracts.
4. **Persistence** (`backend/internal/infra/`) — GORM repos, event bus,
   storage, email, disbursement adapters (OnePay, 9Pay).
5. **Frontend domain modules** (`frontend/src/`) — components, hooks, pages,
   services, separated by role (admin / partner / employee / mobile).
6. **Cross-cutting** — auth (JWT + Casbin RBAC), caching strategy, webhooks,
   double-entry ledger accounting rules, payment provider abstraction.
7. **Migrations** (`backend/migrations/`) — only the schema evolution stories
   that affect behavior (entity introductions, breaking constraint changes).
8. **Tests** — integration scenarios in `backend/tests/integration/` that
   exercise real flows end-to-end; Playwright E2E suites in
   `frontend/tests/`.
9. **Operational concerns** — `Makefile` commands, `docker-compose*.yml`
   setup, deploy pipeline (`make deploy`), restore/backup flow.

## Existing documentation to honor

`docs/` already contains evergreen reference material (architecture, API,
database, testing, deployment, decisions, lessons). The wiki should **reference
and complement** these — not duplicate them. When a topic is already covered by
a high-quality doc under `docs/`, link to it instead of re-writing the prose.

Existing docs to lean on:

- `docs/system-architecture.md` — overall architecture
- `docs/api.md` — HTTP API route map
- `docs/database.md` — MySQL schema overview
- `docs/code-standards.md` — coding conventions
- `docs/testing.md` — test strategy
- `docs/deployment-guide.md` — deploy + restore flow
- `docs/decisions/` — ADRs (architecture decision records)
- `docs/lessons/` — lessons learned
- `docs/prompt-library/` — reusable agent prompts
- `docs/standards/` — security, performance, UI, review checklists
- `docs/troubleshooting.md` — known issues
- `docs/codebase-summary.md` — "where is X" index
- `backend/CLAUDE.md`, `backend/AGENTS.md` — backend conventions
- `frontend/CLAUDE.md`, `frontend/AGENTS.md` — frontend conventions
- `CLAUDE.md`, `AGENTS.md` — repo-root context

## Non-negotiable invariants the wiki must reflect

These are enforced project rules. They should appear as Claims whenever a wiki
page touches the relevant subsystem.

1. **All business time uses `clock.Now()` (Asia/Ho_Chi_Minh)** — never
   `time.Now()` for domain logic. See ADR-006.
2. **Production server is x86_64/amd64** — never arm64 Docker images.
3. **Production uses OnePay** (NOT 9Pay). System is provider-agnostic via
   ADR-003.
4. **Domain layer has zero framework imports** — no Gin, no GORM in
   `internal/domain/`. See ADR-001.
5. **Cache invalidation happens AFTER transaction commit** — never before.
   See ADR-007.
6. **Commit format** `<type>(<scope>): <subject>` — no AI references.
7. **Work on `main` branch directly** — no worktrees or feature branches by
   convention.
8. **Vietnamese for all user-facing text** — no i18n layer; strings are inline
   in components.
9. **Treat desktop and mobile as one delivery scope** — every frontend change
   must cover both views of every affected role unless explicitly narrowed.

## Language and tone

- Wiki pages are written in **English** for agent memory, even though
  user-facing product text is Vietnamese. Quote Vietnamese strings verbatim
  when they appear in code or UI snippets.
- Keep prose tight. No marketing tone. Prefer code anchors like
  `repo://backend/internal/domain/wallet/aggregate.go#L40-L82` for evidence.
- Skip page padding. A wiki page is not a checklist — if a topic doesn't
  support a meaningful system explanation, it doesn't need its own page.

## Things to skip

- Generated knowledge graphs (`.ua/`, `graphify-out/`, `.codegraph/`,
  `.omc/`, `.agentkit/`, `.opencode/`, `.claude/`, `.codex/`, etc.). These are
  parallel tools, not source.
- Build artifacts, coverage output, scratch QA scripts.
- Repomix snapshots, release manifests.
- Local secrets and database volumes.

The full `.openwikiignore` list is authoritative.

## Page taxonomy hint

Suggested top-level sections for `openwiki/`:

- `quickstart.md` — repository map and entry points (always included).
- `architecture/` — domain, application, transport, infra layers and their
  contracts.
- `features/` — major product capabilities (timesheet, FlexPay, salary
  disbursement, wallet/ledger, attendance, web push).
- `operations/` — config, deploy, backup/restore, observability.
- `integrations/` — OnePay/9Pay, Redis, MySQL, asynq, JWT/Casbin, MCP.
- `testing/` — integration harness, Playwright setup, fixtures.
- `frontend/` — role separation (admin/partner/employee), shared component
  patterns, PWA.

OpenWiki owns the final taxonomy. The above is a starting point.

## OKF provenance notes

- Output language: English (unless overridden with `--language`).
- Producer stamp: `openwiki/<version>` for CLI runs, or the coding-agent host
  (e.g. `opencode`) for host-driven runs.
- Page front-matter must begin with valid OKF v0.2 fields (`type`, `title`,
  `description`, `tags`). Do not author `generated`, `verified`, `sources`,
  or `timestamp` fields — OpenWiki owns those.
