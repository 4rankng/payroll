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
- Locate the active control flow before broad text search. Then read the owning source, its callers/consumers, and the narrowest relevant tests.
- Retrieve context progressively. Stop when ownership, contract, precedent, verification, and unresolved risks are known; do not preload whole docs trees.
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

## Imported Claude Cowork project instructions

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

## Board tickets — plan.tingting.vip (bắt buộc từ 24/10)
MỌI thay đổi code bắt nguồn từ request/complaint/feedback PHẢI có issue trên board TRƯỚC khi code; commit message cite ticket (vd PAYR-123).

- TICKET = SHIPMATE SPEC (owner 24/10): mở ticket bằng skill **shipmate specify** (`/specify` / `shipmate:specify`) — spec ngắn đã validate (Problem / Objective / Acceptance Criteria / Out of Scope / Technical Notes) qua independent review; `create_issue` dùng description = spec đã approve. Không tự viết ticket spec ngoài luồng shipmate.

- Board: https://plan.tingting.vip/tingtingsoft/PAYR
- Quản lý qua MCP JSON-RPC: `POST https://plan.tingting.vip/mcp` với headers
  `Authorization: Bearer $(cat ~/.hermes/profiles/dev/cache/scratch/ttplan-token.txt)`
  và `Accept: application/json, text/event-stream`.
  Flow: `initialize` (giữ header Mcp-Session-Id từ response) → `notifications/initialized` → `tools/call`.
- Tools hay dùng: `list_issues {projectKey:"PAYR"}` (mỗi call = 1 page — lặp với seen-id set)
  · `create_issue {projectKey, columnId, title, description}`
  · `update_issue {issueId, columnId}` để chuyển cột (KHÔNG dùng move_issue — đó là đổi project)
  · `add_comment {issueId, body}`.
- Cột của PAYR: Backlog=16 · Todo=17 · In Progress=18 · Done=19 · Canceled=20.
- Luật chuyển cột: mọi chuyển PHẢI kèm `add_comment` evidence (Done = có evidence kiểm chứng;
  Canceled = won't-do + ghi rõ ruling/owner decision). Không đổi trạng thái khi thiếu evidence.

## Consultation — webchat (design/UI/UX/SDLC)
Khi gặp quyết định design/UI/UX, lựa chọn kiến trúc/library, hoặc challenge SDLC chưa rõ hướng:
consult webchat TRƯỚC khi tự quyết:
```bash
cd /Volumes/LexarSSD/projects/one-shot
timeout 240 python3 tools/webchat.py gemini send "<câu hỏi + NGỮ CẢNH + số liệu cụ thể>"
sleep 60 && timeout 120 python3 tools/webchat.py gemini capture   # lặp nếu câu trả lời dài
```
- Adapter gemini/deepseek ONLY. Gửi kèm constraint + số đo thật (px, test counts, spec gốc).
- Pattern/verdict trả về phải được dẫn lại trong brief hoặc comment ticket.

## COMPLETION PROTOCOL (lane land xong — bắt buộc 24/10)
Khi công việc đã commit + push thành công:
1. Update ticket trên board: `update_issue {issueId, columnId:<Done>}` + `add_comment` evidence: commit hash, số test pass, gates, screenshot paths.
2. KHÔNG gửi Telegram (owner 24/10 — chat chỉ nhận verdict tổng từ orchestrator). Orchestrator theo dõi board + tự汇总.
3. Chỉ khi ticket đã update xong mới được kết thúc lane. Nếu board/Telegram lỗi → ghi rõ trong HANDOFF.md phần \"COMPLETION BLOCKED\" kèm lý do.

