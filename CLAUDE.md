# CLAUDE.md — Payroll Monorepo

> **Long-term memory for AI coding agents.** This file gives Claude Code, Codex, and Gemini CLI the context they need to work effectively in this repo. For subdirectory-level detail, see [`backend/CLAUDE.md`](backend/CLAUDE.md) and [`frontend/CLAUDE.md`](frontend/CLAUDE.md).

---

## Project Summary

Payroll management system for Vietnamese companies. Monorepo with a Go backend (DDD / Clean Architecture) and a React frontend (Vite + TanStack Query + shadcn/ui). Handles employee timesheets, advance payments (FlexPay), salary disbursements via OnePay/9Pay, double-entry wallet/ledger accounting, attendance with geofencing, and web push notifications.

**Three roles:** Admin (full access), Partner (project-scoped), Employee (mobile-first). All user-facing text is in Vietnamese.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.26, Gin, GORM, MySQL 8, Redis, asynq |
| Frontend | React 18.3, Vite 6.4, TypeScript 5.8, TanStack Query 5, shadcn/ui, Tailwind 3.4, PWA |
| Auth | JWT + Casbin RBAC (Admin / Partner / Employee / adv_partner) |
| Payments | OnePay (production), 9Pay (sandbox/dev) |
| Infrastructure | Docker, Docker Compose, SSH deploy to `tingting.vip` |

## Repo Layout

```
payroll/
  backend/           Go API server (DDD / Clean Architecture)
    cmd/             Entry points: api-server, hashpw, seed-temp, migrate
    configs/         Casbin RBAC policy (model + CSV)
    internal/
      domain/        Entities, value objects, domain services, ports, specs, wallet aggregate
      app/           Application services, bootstrap/DI, DTOs, asynq workers
      adapters/      Anti-corruption layer for external contracts
      infra/         GORM repos, event bus, asynq, email, disbursement adapters, storage
      pkg/           clock, db, geo, retry, httputils, scopes, validation
      transport/http/ Handlers, middleware (auth, RBAC, rate-limit, security headers)
    migrations/      88 SQL .up.sql files
    tests/integration/ 30 flow test files + HTML report generator
  frontend/          React 18 SPA
    src/
      components/    Domain components (timesheet, wallet, ledger, employees, ...)
      hooks/api/     TanStack Query hooks wrapping backend APIs
      pages/         admin/, partner/, employee/, mobile/
      services/      API client (Axios)
      lib/           Utilities, modal registry
    tests/           Playwright E2E tests
  docs/              Evergreen reference docs, ADRs, standards, lessons
  plans/             Timestamped plan directories + templates + reports
```

## Non-Negotiable Rules

1. **All business time uses `clock.Now()` (Asia/Ho_Chi_Minh)** — never `time.Now()` for domain logic. See [ADR-006](docs/decisions/ADR-006-clock-injection-pattern.md).
2. **Production server is x86_64/amd64** — never build arm64 Docker images.
3. **Production uses OnePay** (NOT 9Pay) — system is provider-agnostic via [ADR-003](docs/decisions/ADR-003-payment-provider-abstraction.md).
4. **Domain layer has zero framework imports** — no Gin, no GORM in `internal/domain/`. See [ADR-001](docs/decisions/ADR-001-ddd-clean-architecture.md).
5. **Cache invalidation happens AFTER transaction commit** — never before. See [ADR-007](docs/decisions/ADR-007-transaction-manager-unit-of-work.md).
6. **Commit format:** `<type>(<scope>): <subject>` — e.g. `feat(timesheet): add bulk export`. No AI references in commit messages.
7. **Work on `main` branch directly** — no worktrees or feature branches by convention.
8. **Vietnamese for all user-facing text** — no i18n layer; strings are inline in components.

## Key Commands

```bash
# Development
make dev              # Start backend (air hot-reload :8080) + frontend (vite :5173)
make db               # Start MySQL + Redis + Adminer + sandbox mocks

# Testing
make api-test         # Integration tests against live backend (30 flow files)
cd backend && go test ./... -v -race -cover   # Backend unit tests
cd frontend && pnpm lint && pnpm type-check    # Frontend checks
cd frontend && pnpm test:e2e                    # Playwright E2E

# Deployment
make deploy           # Build + push amd64 images + SSH deploy to production
make backup           # Backup production DB to OneDrive
make restore          # Restore latest backup to local dev
make adminer          # SSH tunnel to production DB UI at localhost:18081
```

## Where to Find Things

| Need | Look at |
|------|---------|
| Coding conventions | [`docs/code-standards.md`](docs/code-standards.md) |
| Architecture diagram | [`docs/system-architecture.md`](docs/system-architecture.md) |
| "Where is X" index | [`docs/codebase-summary.md`](docs/codebase-summary.md) |
| API route map | [`docs/api.md`](docs/api.md) |
| Database schema | [`docs/database.md`](docs/database.md) |
| Testing strategy | [`docs/testing.md`](docs/testing.md) |
| Known issues | [`docs/troubleshooting.md`](docs/troubleshooting.md) |
| Deployment | [`docs/deployment-guide.md`](docs/deployment-guide.md) |
| Architecture decisions | [`docs/decisions/`](docs/decisions/README.md) |
| Standards (security, perf, UI, review) | [`docs/standards/`](docs/standards/README.md) |
| Reusable prompts | [`docs/prompt-library/`](docs/prompt-library/README.md) |
| Lessons learned | [`docs/lessons/`](docs/lessons/README.md) |
| Definition of Done | [`docs/definition-of-done.md`](docs/definition-of-done.md) |
| Backend detail | [`backend/CLAUDE.md`](backend/CLAUDE.md) + [`backend/AGENTS.md`](backend/AGENTS.md) |
| Frontend detail | [`frontend/CLAUDE.md`](frontend/CLAUDE.md) + [`frontend/AGENTS.md`](frontend/AGENTS.md) |

## Working Effectively

- **Scout before coding.** Read the relevant `AGENTS.md` file in the target directory — every major module has one.
- **Load context progressively.** Follow [Context Engineering](docs/standards/context-engineering.md): establish a small task packet, retrieve the owning path and contracts first, and stop loading when the task can be executed and verified.
- **Follow the Locate → Repair → Validate loop** instead of one massive prompt. See [prompt-library/bug-fix.md](docs/prompt-library/bug-fix.md).
- **Run `make api-test` after every feature change** to catch regressions.
- **Review agent output at architectural boundaries**, not line by line.
- **Use the [review checklist](docs/standards/review-checklist.md) before submitting changes.**
- **Graphify is available:** `graphify query "<question>"` for codebase navigation when `graphify-out/graph.json` exists.

<!-- OPENWIKI:START -->

## OpenWiki

See [AGENTS.md](AGENTS.md) for OpenWiki agent instructions.

<!-- OPENWIKI:END -->

<!-- code-review-graph MCP tools -->
## MCP Tools: code-review-graph

**This project has a knowledge graph. Start with the code-review-graph
MCP tools to narrow scope, then read the source.** The graph is cheaper than scanning files and
gives you structural context (callers, dependents, test coverage) that file search cannot.

### When to use graph tools FIRST

- **Exploring code**: `semantic_search_nodes_tool` or `query_graph_tool` instead of Grep
- **Understanding impact**: `get_impact_radius_tool` instead of manually tracing imports
- **Code review**: `detect_changes_tool` + `get_review_context_tool` instead of reading entire files
- **Finding relationships**: `query_graph_tool` with callers_of/callees_of/imports_of/tests_for
- **Architecture questions**: `get_architecture_overview_tool` + `list_communities_tool`

### Verify in the source

- Narrow scope with the graph, then read the source. Do not change code from graph output alone.
- For any non-trivial change, read the implementation and the relevant tests before concluding.
- Verify the exact source when touching behavior, database logic, migrations, retries, fallbacks,
  recovery, or compatibility code.
- When the graph and the source disagree, the source wins. The graph may be stale or may not
  model that relationship.
- An empty graph result can mean "not indexed" or "not statically visible", not "does not exist".

### Key Tools

| Tool | Use when |
| ------ | ---------- |
| `detect_changes_tool` | Reviewing code changes — gives risk-scored analysis |
| `get_review_context_tool` | Need source snippets for review — token-efficient |
| `get_impact_radius_tool` | Understanding blast radius of a change |
| `get_affected_flows_tool` | Finding which execution paths are impacted |
| `query_graph_tool` | Tracing callers, callees, imports, tests, dependencies |
| `semantic_search_nodes_tool` | Finding functions/classes by name or keyword |
| `get_architecture_overview_tool` | Understanding high-level codebase structure |
| `refactor_tool` | Planning renames, finding dead code |

### Workflow

1. The graph auto-updates on file changes (via hooks).
2. Use `detect_changes_tool` for code review.
3. Use `get_affected_flows_tool` to understand impact.
4. Use `query_graph_tool` pattern="tests_for" to check coverage.
<!-- /code-review-graph MCP tools -->

## Agent skills

### Issue tracker

Issues and specs live as GitHub issues in `4rankng/payroll`, managed via the `gh` CLI. See [`docs/agents/issue-tracker.md`](docs/agents/issue-tracker.md).

### Triage labels

Five canonical triage roles use the default label strings (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See [`docs/agents/triage-labels.md`](docs/agents/triage-labels.md).

### Domain docs

Single-context layout: root `CONTEXT.md` (created lazily by `/domain-modeling`) with ADRs in `docs/decisions/`. See [`docs/agents/domain.md`](docs/agents/domain.md).
