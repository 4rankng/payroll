<!-- Generated: 2026-07-04 | Init via /docs init -->

# docs/

Project documentation for the payroll monorepo. Evergreen reference docs live directly under `docs/`; asset and reference subdirectories are grouped by topic.

## Evergreen Docs

| File | Purpose |
|------|---------|
| `project-overview-pdr.md` | Product overview, roles/personas, functional scope, domain glossary |
| `codebase-summary.md` | High-level map: where things live, LOC by area, "where do I find X" index |
| `code-standards.md` | Conventions observed in backend and frontend code |
| `system-architecture.md` | Component diagram, request/event/job flows, deployment topology |
| `project-roadmap.md` | Direction inferred from recent commit history and codebase signals |
| `deployment-guide.md` | Prod/demo targets, Docker constraints, migrations, backup/restore |

The project front door is the root [`README.md`](../README.md). Per-area developer context lives in [`backend/AGENTS.md`](../backend/AGENTS.md) and [`frontend/AGENTS.md`](../frontend/AGENTS.md).

## Asset & Reference Subdirectories

| Directory | Contents |
|-----------|----------|
| `checkinout/` | Attendance check-in/check-out reference material |
| `LGD Files/` | Vietnamese bank/banking reference files (Lệnh giao dịch) |
| `onepay/` | OnePay (production disbursement provider) integration reference |
| `qa/` | QA scripts and review notes |
| `superpowers/` | Skill specs and plans |
| `WeeklyBCC/` | Weekly BCC (timesheet) upload reference files |

## Conventions

- Evergreen docs use kebab-case slugs (e.g. `system-architecture.md`).
- Timestamped content (plans, agent reports, session journals) does NOT live here — it goes under `plans/{YYMMDD-HHmm}-{slug}/` and `plans/reports/`. See root `AGENTS.md`.
- Keep each doc under 800 LOC; split into `{name}-{variant}.md` and cross-link when a topic outgrows that.
- Docs are written in English prose; user-facing role names and provider names stay in the original Vietnamese where that is the in-product term.
