<!-- Generated: 2026-06-07 | Updated: 2026-07-26 -->

# Payroll

## Purpose
Payroll management system for Vietnamese companies. Monorepo with a Go backend (DDD / Clean Architecture) and a React frontend (Vite + TanStack Query + shadcn/ui). Handles employee timesheets, advance payments (FlexPay), salary disbursements via OnePay/9Pay, double-entry wallet/ledger accounting, and web push notifications. Three user roles: Admin, Partner, Employee (with mobile-first views).

## Key Files
| File | Description |
|------|-------------|
| `CLAUDE.md` | Project-level instructions, integration test patterns, clock system, cache invalidation rules |
| `docs/standards/context-engineering.md` | Task-scoped context selection, state, and evidence contract |
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
- Treat desktop and mobile as one frontend delivery scope for **both Admin and Partner**: every frontend feature or fix must be implemented for both views of every affected role unless the user explicitly requests a narrower exception
- When routes or responsive wrappers render separate desktop and mobile components, trace and update both paths; never assume a desktop change automatically reaches mobile
- Commit format: `<type>(<scope>): <subject>` — e.g. `feat(timesheet): add bulk export`
- `make dev` starts backend (air hot-reload on :8080) and frontend (vite dev on :5173)
- `make deploy` builds and deploys to production server via SSH

### Testing Requirements
- `make api-test` — integration tests against live backend
- Backend unit tests: `cd backend && go test ./... -v -race -cover`
- Frontend: `cd frontend && pnpm lint && pnpm type-check`
- Frontend UI changes: verify authenticated desktop and mobile views for every affected role, including separate Admin and Partner checks; test at least 1280px desktop and 390px mobile, plus 320px when content density, long text, money, tables, dialogs, or sheets may cause overflow
- Desktop/mobile verification must cover feature and action parity, data and permission parity, readable wrapping, no horizontal overflow, accessible controls, and minimum 44px mobile touch targets
- Update `backend/tests/integration` with test scenarios for new features

### Common Patterns
- Backend: Domain events via `EventBus.Publish()`, non-blocking goroutines, outbox pattern
- Frontend: TanStack Query for data fetching, optimistic updates, Vietnamese UI text
- Both: Centralized clock, Redis caching with invalidation after commit, asynq background jobs

### Development Context Engineering
- Start with the user goal, root `AGENTS.md`, `CLAUDE.md`, `README.md`, and the nearest `AGENTS.md` for each target file. Add reference docs only when they govern the task.
- Use `graphify query` to locate the active control flow before broad text search. Then read the owning source, its callers/consumers, and the narrowest relevant tests.
- Retrieve context progressively. Stop when ownership, contract, precedent, verification, and unresolved risks are known; do not preload whole docs trees or `GRAPH_REPORT.md`.
- For cross-module or resumable work, keep a short task packet based on `plans/templates/task-context.md`. Record decisions and evidence, not full tool output.
- Re-check the packet after scope changes or contradictory evidence. Completion claims must name the checks actually run and classify skipped or failing checks.
- The full selection, compression, state, and evaluation contract is in [`docs/standards/context-engineering.md`](docs/standards/context-engineering.md).

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

## Understand-Anything Knowledge Base

This project has an Understand-Anything knowledge graph at `.ua/knowledge-graph.json` (live, LLM-generated; powers the interactive dashboard, `/understand-chat`, `/understand-diff`, `/understand-domain`, and `/understand-onboard`). It is per-developer and gitignored — never commit `.ua/` or `.understand-anything/`.

Rules:
- **Keep the KB current after every task that changes code.** When you finish a code change (feature, fix, refactor), run `/understand` so it does an incremental update against the new commit. `autoUpdate: true` is set in `.ua/config.json`. Do this as a standard completion step, alongside `make api-test` — a stale graph misleads every future session.
- For "how does X flow / where is Y implemented" questions, prefer `/understand-chat` over raw grep once the graph exists.
- Before non-trivial changes, run `/understand-diff` for impact analysis against the existing graph.
- For the initial build or a corrupt/stale graph, run `/understand --full` (full rebuild — slow, token-heavy; use sparingly on this ~2500-file repo).
- The dashboard is launched on demand via `/understand-dashboard`; it is not auto-started.

## Imported Claude Cowork project instructions

<!-- OPENWIKI:START -->

## OpenWiki

This repository has a generated `openwiki/` evidence index. It is optional just-in-time context, not required startup reading.

- Treat source code and tests as authoritative. A brief's unknowns and review items are verification gaps, not automatic requirements.
- Prefer the narrowest quiet validation that proves the changed behavior. Preserve complete failure output.

The scheduled OpenWiki GitHub Actions workflow refreshes the repository wiki. Do not hand-edit generated OpenWiki pages unless explicitly asked; prefer updating source code/docs and letting OpenWiki regenerate.

<!-- OPENWIKI:END -->
