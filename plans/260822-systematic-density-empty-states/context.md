# Task Context

## Intent

- **Goal:** Remove inherited oversized typography and give every standard empty state a compact illustrated treatment.
- **Observable success:** Normal copy is 11–12px, unstyled headings do not inflate empty states, and shared tables/lists/empty-state components show the generated illustration.
- **In scope:** Frontend Admin, Partner, and Employee shared empty-state paths; desktop and mobile table/list primitives.
- **Out of scope:** Loading, error, permission, and command-menu microcopy states.

## Decisions

| Decision | Evidence | Consequence |
|----------|----------|-------------|
| Remove global heading size inheritance | The reported CheckIn empty `<h2>` has no size utility and inherits `h2 { font-size: var(--text-4xl) }`. | Semantics no longer accidentally change visual density. |
| Use one generated raster asset through a shared component | A no-result illustration is usable for employees, projects, timesheets, and financial lists without changing messages. | Every standard empty state has coherent imagery without asset duplication. |

## Progress

| Work item | Status | Evidence / next action |
|----------|--------|------------------------|
| Locate active path | PASS | `CheckInSettingsPage` and global `base.css` identified. |
| Implement | PASS | Added the shared illustration, removed global heading-size inheritance, and updated table/list/page empty paths. |
| Focused verification | PASS | `control-density` and `CheckInSettingsPage` tests: 16/16 passed. |
| Required broader verification | BLOCKED | Lint/typecheck and production build passed; API and authenticated browser QA need local services. |

## Handoff

- **Changed files:** `styles/base.css`, shared/table/list empty-state components, Admin/Partner page empty states, generated asset under `public/images/empty-states/`.
- **Blockers:** Ports 3000, 5173, and 8080 are not serving; API integration fails before test execution at admin login.
- **Latest verified evidence:** `pnpm lint`, `pnpm build`, and 16 focused tests passed; graphify refreshed at 16:35.
