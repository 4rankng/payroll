# Task Context

## Intent

- **Goal:** Remove inherited oversized typography and give every standard empty state a compact illustrated treatment.
- **Observable success:** Normal copy is 11–12px, unstyled headings do not inflate empty states, and every standalone table/list/card/sheet/dialog empty view shows the generated illustration.
- **In scope:** Frontend Admin, Partner, and Employee shared empty-state paths; desktop and mobile table/list primitives.
- **Out of scope:** Loading, error, permission, and command-menu microcopy states.

## Decisions

| Decision | Evidence | Consequence |
|----------|----------|-------------|
| Remove global heading size inheritance | The reported CheckIn empty `<h2>` has no size utility and inherits `h2 { font-size: var(--text-4xl) }`. | Semantics no longer accidentally change visual density. |
| Use a generated illustration family through a shared component | Empty-state meaning follows the data domain (employees, projects, finance, activity, records, or search). | Every standard empty state receives coherent contextual imagery without per-page asset logic. |

## Progress

| Work item | Status | Evidence / next action |
|----------|--------|------------------------|
| Locate active path | PASS | `CheckInSettingsPage` and global `base.css` identified. |
| Implement | PASS | Added a six-image contextual illustration family, removed global heading-size inheritance, and migrated all audited empty data views to a horizontal image-left/copy-right treatment. |
| Focused verification | PASS | `control-density` and `CheckInSettingsPage` tests: 17/17 passed. |
| Required broader verification | BLOCKED | Lint/typecheck and production build passed; API and authenticated browser QA need local services. |

## Handoff

- **Changed files:** `styles/base.css`, shared/table/list empty-state components, Admin/Partner page empty states, dashboard/sheet/dialog empty states, and the generated illustration family under `public/images/empty-states/`.
- **Blockers:** Ports 3000, 5173, and 8080 are not serving; API integration fails before test execution at admin login.
- **Latest verified evidence:** `pnpm lint`, `pnpm build`, `git diff --check`, and 17 focused tests passed; graphify refreshed at 16:54.
