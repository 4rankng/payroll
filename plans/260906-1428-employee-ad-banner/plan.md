---
title: "employee-ad-banner"
description: "Project-targeted ad banner for the employee portal with an admin campaign composer and mandatory bounded lifetime."
status: completed
priority: P2
created: 2026-09-06
---

# Employee Ad Banner — Plan Index

Authoritative design: [`spec.md`](./spec.md) (accepted 2026-09-06; corrected after
codebase verification). Execution detail lives in the phase files below.

## Outcome

Admins publish project-targeted ad campaigns (first: LG Display) that employees see
on login — a one-time bottom sheet collapsing into a dismissible card — with mandatory
start/end dates, typed content, tappable CTAs (phone/url) with click counts, all
managed from a new "Quảng cáo" Settings tab.

## Phases

| # | Phase | Status | Gate |
|---|-------|--------|------|
| 1 | [Backend vertical slice](./phase-01-backend.md) | completed | `go build ./... && go test ./... -race -cover` |
| 2 | [Integration flow](./phase-02-integration.md) | completed | `make api-test` (from `backend/`) all green |
| 3 | [Employee portal](./phase-03-employee-portal.md) | completed | vitest + `tsc -p tsconfig.app.json --noEmit` + manual both pages |
| 4 | [Admin settings UI](./phase-04-admin-ui.md) | completed | vitest incl. parity + tsc + lint |
| 5 | [Verification & rollout](./phase-05-verification.md) | completed | testplan matrix + code-review + deploy chain |

Dependencies: 1 → 2 → (3, 4) → 5. Phases 3 and 4 are independent of each other.

## Acceptance criteria

1. Admin can create an ad scoped to selected projects (empty selection = all projects)
   with a mandatory window; `ends_at` is NOT NULL and ≤180 days after `starts_at`.
2. Employee of a targeted project sees the sheet once per campaign version on BOTH
   `EmployeePage` and `FlexiblePayEmployeePage`; dismiss → card; dismiss card → hidden.
3. Editing a campaign (version bump via `updated_at`) re-shows the sheet once.
4. CTA taps record clicks (employee id from auth context); admin list shows counts.
5. Past `ends_at` the campaign resolves for nobody — no cleanup job needed.
6. No regressions: full `go test -race`, integration suite, parity test, tsc, lint.

## Constraints

- Domain layer: zero framework imports (ADR-001); all time via `clock.Now()`.
- Cache invalidation after commit (ADR-007). No DB CHECK constraints. Vietnamese UI.
- Admin-only management (partner access deferred). Text-only banner (no image yet).
- Parallel sessions share this tree: stage explicit paths, never `git add -A`.

## Rollout

Local (mig 106 + `make dev` + full matrix) → demo (mig manual + `make demo` + verify) →
prod (mig manual + `make deploy` from root, amd64, verify tag == HEAD) → create the LG
Display campaign through the admin UI (audit trail). `make deploy` never runs
migrations. Rollback = previous image; the new tables can stay.
