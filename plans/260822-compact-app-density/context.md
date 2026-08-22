# Task Context

## Intent

- **Goal:** Make Admin and Partner operational UI data-dense by removing oversized controls, typography, and vertical spacing visible in the supplied timesheet and dashboard screenshots.
- **Observable success:** Ordinary UI text is 11-12px; desktop controls are 32-36px where appropriate; cards, stats, filters, legends, and table rows use compact vertical spacing; mobile retains 44px interactive targets and no horizontal overflow.
- **In scope:** Shared typography/control/card primitives plus the Admin dashboard, Admin/Partner timesheet desktop paths, and their separate mobile counterparts for parity verification.
- **Out of scope:** Business logic, API/data contracts, deployment, commit, and push. Shared font/control tokens necessarily affect Employee surfaces, but Employee-owned layouts were not otherwise changed.

## Authority

- **Applicable instructions:** Root and frontend AGENTS.md/CLAUDE.md; docs/standards/context-engineering.md; design-data-dense-ui; daisyUI usage/color rules.
- **Product or technical contracts:** Admin and Partner desktop/mobile parity; mobile touch targets at least 44px; Vietnamese labels and permissions unchanged.
- **Explicit user decisions:** Data-dense design; reduce application body/metadata type to 11-12px; remove excessive top/bottom padding from buttons, filters, stats, and cards.
- **User-owned work to preserve:** Worktree was clean at task start. Concurrent backend and check-in settings edits appeared during implementation and were left untouched.

## Active Path

- **Entry point:** Admin `/admin`, Admin `/admin/timesheet`, Partner `/partner/timesheet`, and separate mobile route trees.
- **Owner:** `frontend/src/styles/variables.css`, shared UI primitives, dashboard components, and timesheet shared components.
- **Persisted or public contract:** Presentation-only; filters, status selection, chart filtering, navigation, data, and permissions remain unchanged.
- **Consumers / role / responsive variants:** Admin and Partner desktop; separate Admin and Partner mobile pages/components.
- **Closest valid precedent:** Existing 11px/12px raw Tailwind scale and compact data-table typography; responsive mobile controls already use 44px targets.

## Open Questions

| Question | Why it matters | Status / evidence |
|----------|----------------|-------------------|
| Is the regression shared or page-local? | Determines primitive vs page ownership. | PASS - shared controls are fixed at 44px and Card increases to 24px padding on desktop; feature layers add more spacing. |
| Is the UA knowledge graph current? | Required before impact analysis. | PASS - graph commit equals current HEAD and worktree was clean. |
| Can authenticated rendered QA run? | Required to verify real geometry and overflow. | BLOCKED - neither browser had a Payroll tab and localhost frontend/backend were not running; project rules prohibit auto-starting the frontend. |

## Decisions

| Decision | Evidence | Consequence |
|----------|----------|-------------|
| Keep Employee portal typography unchanged. | It has an explicit mobile readability contract and is not shown in the report. | Scope remains Admin/Partner operational UI. |
| Use 44px below `sm`, compact 32-36px controls at `sm` and above. | Data-dense skill starting points plus project mobile accessibility requirement. | Density improves without shrinking touch targets. |
| Preserve larger titles and primary metrics. | Density is hierarchy, not indiscriminate tiny text. | Ordinary copy becomes 11-12px while key values remain scannable. |

## Progress

| Work item | Status | Evidence / next action |
|-----------|--------|------------------------|
| Locate active path | PASS | Graphify plus source tracing identified shared primitives and the reported dashboard/timesheet owners. |
| Confirm contracts | PASS | Admin/Partner parity and 44px mobile contract confirmed. |
| Implement | PASS | Shared typography/controls/cards plus Admin dashboard and Admin/Partner timesheets compacted; mobile-first 44px targets preserved. |
| Focused verification | PASS | 4 density test files, 9 tests passed, including 11/12px token and responsive control-height guards. |
| Required broader verification | PARTIAL | Frontend lint/type-check/build passed; 115 files/453 tests passed. API integration reached the correct target but localhost:8080 was unavailable. Authenticated responsive browser QA was unavailable. |
| Knowledge graph maintenance | PASS/PENDING COMMIT | `graphify update .` rebuilt 26,725 nodes/55,781 edges and a final refresh reported no topology delta. Understand-Anything remains at HEAD because this task has no authorized commit; its incremental workflow is commit-based. |
| Review completion claim | PASS | Fresh full frontend suite, lint/type-check, production build, focused density tests, and diff whitespace check all passed; blocked checks classified separately. |

## Handoff

- **Changed files:** Shared UI primitives/styles, Admin dashboard components/page, Admin/Partner timesheet components/pages, four density regression tests, and this task packet.
- **Remaining work:** Authenticated rendered QA and API integration when the local app/backend are available.
- **Blockers:** Authenticated responsive browser QA (no Payroll tab/server) and API integration (localhost:8080 refused connections).
- **Latest verified evidence:** Frontend lint/type-check/build passed; full Vitest 115/115 files and 453/453 tests passed; focused density suite 9/9 passed; Graphify refresh completed.
