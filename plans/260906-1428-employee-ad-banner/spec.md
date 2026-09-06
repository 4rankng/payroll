---
title: "employee-ad-banner"
description: "Project-targeted advertising banner shown to employees in the mobile portal, with an admin composer that scopes each campaign to projects and enforces a bounded lifetime."
status: accepted
priority: P2
effort: "6 phases (estimated)"
tags: [feature, employee-portal, admin-settings, backend, frontend]
created: 2026-09-06
---

# Employee Ad Banner — Design Spec

## Overview

Admins need to promote TingTing services to the workers of a specific project. The
first campaign targets LG Display employees and advertises automatic attendance plus
same-day advance payments, closing on a hotline number and a Zalo support group.

Today there is no announcement surface at all. Notifications exist but are an inbox
(bell icon, push channel) with recipient targeting by role — `to_all_employees` — and
no concept of a project scope or a display window. They are the wrong shape: an ad is
ambient and dismissible, a notification is addressed and persistent.

This spec adds a small campaign subsystem: `ad_banners` rows resolved per employee by
their active project assignment, rendered in the mobile portal as a one-time sheet
that collapses into a dismissible card, and configured from a new Settings tab.

**Decisions locked with the user (2026-09-06):** dedicated table with JSON project
targeting (not a `settings` blob, not a 3-table campaign system); hybrid sheet→card
presentation; typed content fields (no admin-authored rich text); mandatory bounded
lifetime.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | An admin can publish an ad visible only to the projects they select | P1 |
| 2 | Every campaign has a mandatory end date; no ad can run forever | P1 |
| 3 | The ad reaches employees on **both** employee home pages | P1 |
| 4 | Content is typed (title/body/bullets/CTAs) and renders inside the portal type scale | P1 |
| 5 | An employee can dismiss the ad and not be nagged for that campaign version | P1 |
| 6 | Admin can see how many employees tapped each CTA | P2 |
| 7 | Multiple campaigns can be scheduled and prioritised concurrently | P2 |

## Non-Goals

- Per-employee server-side dismissal state (localStorage is sufficient; a dismissal
  row per employee per campaign across thousands of employees buys cross-device
  memory that nobody asked for).
- Impression counting. Reach is already derivable from `project_employees`; per-view
  rows would grow without bound for a number that is knowable by query.
- Targeting on anything other than project (no payment-schedule, position, or
  tenure filters until a campaign actually needs one).
- Partner-authored ads. Admin-only for this iteration — see Open Questions.
- Push delivery. The ad is a portal surface, not a notification.

## Data Model

Migration `106_add_ad_banners`. Two tables.

### `ad_banners`

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGINT UNSIGNED` PK | |
| `title` | `VARCHAR(255) NOT NULL` | Headline |
| `body` | `TEXT NULL` | Lead paragraph |
| `bullets` | `JSON NULL` | `["Chấm công tự động, chính xác từng ca làm", ...]`, max 6 |
| `ctas` | `JSON NULL` | `[{"label":"Gọi hotline","type":"phone","value":"0914827988"}]`, max 3 |
| `footer` | `VARCHAR(255) NULL` | Sign-off line |
| `target_project_ids` | `JSON NULL` | `[12]`. `NULL` or `[]` means every project |
| `priority` | `INT NOT NULL DEFAULT 0` | Higher wins when several campaigns are live |
| `starts_at` | `TIMESTAMP NOT NULL` | |
| `ends_at` | `TIMESTAMP NOT NULL` | **Not nullable — this is what bounds the lifetime** |
| `is_active` | `TINYINT(1) NOT NULL DEFAULT 1` | Manual pause without deleting |
| `created_by` | `BIGINT UNSIGNED NOT NULL` | |
| `created_at` / `updated_at` | `TIMESTAMP` | `updated_at` doubles as the campaign version |
| `deleted_at` | `TIMESTAMP NULL` | Soft delete, indexed |

Index: `idx_ad_banners_active_window (is_active, starts_at, ends_at)`.

No CHECK constraints — business rules are validated in domain code, per project
convention.

### `ad_banner_cta_clicks`

| Column | Type | Notes |
|---|---|---|
| `id` | `BIGINT UNSIGNED` PK | |
| `banner_id` | `BIGINT UNSIGNED NOT NULL` | Indexed |
| `cta_index` | `INT NOT NULL` | Position in the banner's `ctas` array |
| `employee_id` | `BIGINT UNSIGNED NOT NULL` | |
| `clicked_at` | `TIMESTAMP NOT NULL` | |

Append-only. The admin click count is `GROUP BY banner_id, cta_index` — aggregation in
SQL, consistent with how dashboard stats are computed elsewhere in this codebase.
Rows are rare (a tap, not a page view), so the table stays small.

### Why `ends_at` is NOT NULL

A nullable end date reintroduces the exact problem this feature must avoid: an ad
created in a hurry with no end date runs until someone notices. Making the column
non-nullable means an endless campaign is unrepresentable in the database, not merely
discouraged by the UI.

## Domain Layer

`backend/internal/domain/ad_banner.go` — zero framework imports (ADR-001).

```go
type AdBannerCTAType string // "phone" | "url"

type AdBannerCTA struct {
    Label string
    Type  AdBannerCTAType
    Value string
}

type AdBanner struct { /* fields per schema above */ }

type AdBannerRepository interface {
    Create(ctx, *AdBanner) error
    GetByID(ctx, id uint) (*AdBanner, error)
    Update(ctx, *AdBanner) error
    Delete(ctx, id uint) error
    List(ctx, AdBannerFilters) ([]*AdBanner, error)
    ListLiveAt(ctx, t time.Time) ([]*AdBanner, error)
    RecordCTAClick(ctx, bannerID uint, ctaIndex int, employeeID uint) error
    CTAClickCounts(ctx, bannerIDs []uint) (map[uint]map[int]int64, error)
}
```

Two pure predicates carry the behaviour worth testing:

- `IsLiveAt(t time.Time) bool` — `is_active && !t.Before(starts_at) && t.Before(ends_at)`.
  Half-open interval: a campaign ending 2026-10-06 stops being live the instant that
  timestamp is reached, with no off-by-one on the final day.
- `TargetsProject(projectID uint) bool` — `true` when `target_project_ids` is empty
  (broadcast) or contains the id.

`Validate()` enforces:

| Rule | Reason |
|---|---|
| `title` non-empty, ≤255 chars | Column width |
| `bullets` ≤ 6, each ≤ 200 chars | Mobile card cannot render more legibly |
| `ctas` ≤ 3, each label ≤ 40 chars | Button row width at 375px |
| CTA `type` in {phone, url} | Determines `tel:` vs `window.open` |
| CTA `value` matches type shape (digits for phone, `https://` for url) | Prevents a broken or `javascript:` link on a surface every worker sees |
| `ends_at` after `starts_at` | Empty window would be silently invisible |
| `ends_at - starts_at` ≤ 180 days | Ceiling so a typo'd year cannot recreate the forever case |
| `target_project_ids` entries are distinct, non-zero | Data hygiene |

All timestamp comparisons and defaults use `clock.Now()` (Asia/Ho_Chi_Minh), never
`time.Now()`.

## Application Service

`backend/internal/app/services/ad_banner/service.go` — package `ad_banner` (snake_case, matching `advance_payment` / `flex_pay`).

**`ResolveForEmployee(ctx, employeeID) (*AdBanner, error)`**

1. Load the employee's active project assignments (`project_employees` where
   `last_date IS NULL`).
2. Load banners live at `clock.Now()`.
3. Keep those where `TargetsProject` matches any of the employee's projects.
4. Order by `priority DESC, created_at DESC`; return the first, or `nil`.

Returning at most one banner is deliberate — the portal shows a single ad slot, and
resolving the winner server-side keeps the precedence rule in one tested place rather
than duplicated in two React pages.

An employee with no active project assignment receives `nil`, not a broadcast ad: they
are between assignments and targeting them is meaningless.

**Caching.** Live banners are cached for 60s keyed by the sorted project-id set.
Invalidation happens **after** transaction commit on any create/update/delete
(ADR-007) — never before.

**`RecordCTAClick`** writes an append-only row. Failures here are logged and swallowed
at the handler boundary: a click-tracking outage must never block a worker from
reaching the hotline.

Admin CRUD (`List` with click counts joined, `Create`, `Update`, `Delete`) emits audit
events through the existing `event_factory_*` pattern.

## HTTP API

Employee self-service lives under `/api/v1/me` in this codebase.

| Method | Path | Role | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/me/ad-banner` | employee | Resolved banner or `null` |
| `POST` | `/api/v1/me/ad-banner/:id/click` | employee | Body `{"cta_index": 0}` |
| `GET` | `/api/v1/ad-banners` | admin | List with click counts and computed status |
| `POST` | `/api/v1/ad-banners` | admin | Create |
| `PUT` | `/api/v1/ad-banners/:id` | admin | Update |
| `DELETE` | `/api/v1/ad-banners/:id` | admin | Soft delete |

Casbin additions to `backend/configs/casbin_policy.csv` — exactly **one** new
line is required:

```
p, employee, /api/v1/me/ad-banner/*, POST, allow
```

Everything else is already covered: `p, admin, /api/*, *, allow` (line 2)
covers the full admin CRUD, and the existing
`p, employee, /api/v1/me/*, GET, allow` wildcard covers the employee GET.

The click endpoint takes the banner id from the path and the employee id from the
authenticated context — never from the request body — so one employee cannot
attribute clicks to another.

## Employee Portal

Two components in `frontend/src/components/employees/`:

- **`EmployeeAdSheet.tsx`** — bottom sheet with the full campaign: title, body,
  bullets, CTA buttons, footer.
- **`EmployeeAdBanner.tsx`** — compact card: title, first bullet, "Xem chi tiết".

Data hook `frontend/src/hooks/api/useEmployeeAdBanner.ts` (TanStack Query + click
mutation).

**Mounting.** As the first content child of **both** `EmployeePage` and
`FlexiblePayEmployeePage`. LG Display workers are on the flexpay/self-checkin page;
shipping to only the regular page would deliver a banner the target audience never
sees. This is the single highest-risk omission in the feature. On
`FlexiblePayEmployeePage` the banner mounts as a sibling **above** the main
content grid, not inside it — that grid pins its children with explicit
`lg:col-start/row-start` placement and an in-grid banner would collide.

**Lifecycle per employee**, keyed on `localStorage["employee_ad_state"]` storing
`{bannerId, version, stage}` where `version` is the banner's `updated_at`:

1. No stored state for this `bannerId:version` → sheet opens automatically.
2. Dismiss sheet → `stage: "card"`; compact card remains in the feed.
3. Dismiss card → `stage: "hidden"`; nothing further for this version.
4. Admin edits the campaign → `updated_at` changes → key no longer matches → cycle
   restarts. That is a republish, and intentional.

**CTA behaviour.** Fire the click mutation, then act: `phone` → `tel:` href (native
dialer), `url` → `window.open(value, "_blank", "noopener,noreferrer")`. Navigation is
not gated on the mutation resolving.

**Styling.** `employee-type-*` semantic classes and `--employee-*` tokens only, no
arbitrary pixel sizes. Full screen width. One Card nesting level. Use the existing
`--employee-accent` dark-emerald family (`frontend/src/styles/variables.css:74-86`) —
the employee theme has no gold token; amber is reserved for actionable warnings.

## Admin UI

New Settings tab `ads` (label **"Quảng cáo"**, icon `Megaphone`) in
`frontend/src/components/settings/SettingsTabList.tsx`, rendering
`frontend/src/components/settings/AdBannerSection.tsx`.

> Implementation notes: (1) `SettingsTabList` uses a `grid-cols-6` mobile layout whose
> `mobileSpan` values currently sum to 12 — rebalance to six × `col-span-2`, not just
> append an entry. (2) The mobile admin SettingsPage
> (`frontend/src/pages/mobile/admin/SettingsPage/index.tsx`) renders its own
> `TabsContent` per tab and `settings-page-parity.test.tsx` renders BOTH pages — add
> the tab to both and add a `vi.mock` for `AdBannerSection` in that test, following
> the `ZaloConnectionSection` mock.

**Layout.** Campaign list on the left, composer and live preview on the right.

**List columns.** Title, targeted projects (names, or "Tất cả dự án"), status chip,
CTA click counts.

**Status chips**, computed from the window:

| Condition | Chip |
|---|---|
| `is_active` and now inside window | `Còn N ngày` |
| `is_active` and now before `starts_at` | `Hẹn lịch` |
| now at or after `ends_at` | `Hết hạn` |
| `!is_active` | `Tạm dừng` |

**Composer fields.** Title, body, bullets (add/remove, max 6), CTAs (label + type +
value, max 3), project multi-select with a "Tất cả dự án" option, priority, active
toggle, and the lifetime control.

**Lifetime control.** Duration is the primary input, not a raw date pair: presets 7 /
14 / 30 / 60 ngày, default 30, with a custom date picker behind "Tùy chọn". The
computed end date is displayed live ("Kết thúc: 06/10/2026") so the admin always sees
what they are committing to before saving.

**Expired campaigns** stay in the list, greyed, grouped under "Hết hạn", because their
click counts are the record of how that run performed. Renewal is a **"Gia hạn"**
action that clones the campaign with a fresh window rather than editing a dead one —
historical stats stay attached to the run that produced them.

**Preview.** A 375px-wide frame rendering the *same* `EmployeeAdSheet` component the
employee sees. Rebuilding the markup for preview guarantees drift; reusing it makes
the preview correct by construction.

The project multi-select must close on click-outside (universal rule for every
dropdown in this codebase).

## Testing

**Domain (unit).** `IsLiveAt` at both window edges and the half-open boundary;
`TargetsProject` with empty, matching, and non-matching target lists; every `Validate`
rule including the 180-day ceiling and CTA value shapes.

**Service (unit, mocked repo).** Highest priority wins among several live campaigns;
expired campaign excluded; campaign targeting another project excluded; employee with
no active assignment gets `nil`; broadcast campaign reaches an employee in any
project; cache invalidation fires after commit, not before.

**Integration.** A flow file in `backend/tests/integration/` following the existing 30
files: admin creates a project-targeted campaign → targeted employee's
`/api/v1/me/ad-banner` returns it → untargeted employee gets `null` → click endpoint
records a row → admin list reflects the count → campaign past `ends_at` disappears
from the employee response. The flow must be registered in
`backend/tests/integration/main.go` (beside `runEmployeeSelfServiceTests`) or it
never runs; `make api-test` is invoked from `backend/` (the root Makefile has no
such target).

**Frontend.** Component tests for sheet and card render, the three-stage dismissal
state machine, and version-bump re-display. Type gate is
`npx tsc -p tsconfig.app.json --noEmit` — `pnpm lint` compiles zero files in this repo
and is not a real gate.

## Rollout

1. Migration 106 on local, then demo (`make demo`), then production. Additive
   `CREATE TABLE` only — no changes to existing tables, no data migration.
2. Backend and frontend deploy together (`make deploy`, amd64).
3. Verify on demo end to end before the production deploy.
4. Create the LG Display campaign **through the admin UI**, not seeded SQL, so it
   lands in the audit log with a real author.
5. `make api-test` (from `backend/`) after the change.

Rollback is a deploy of the previous image; the tables can stay (nothing reads them
when the code is gone).

## Risks

| Risk | Mitigation |
|---|---|
| Banner mounted on only one employee page → target audience never sees it | Explicit acceptance criterion; integration test covers both entry points |
| Admin pastes a malformed or hostile CTA URL | Domain validation restricts to `tel:`-safe digits and `https://` URLs; content is typed, never rendered as HTML |
| Ad feels like spam and erodes trust in the portal | Three-stage dismissal, single ad slot, mandatory end date |
| Wrong project selected → ad shown to the wrong workforce | Live preview plus project names (not ids) in the list; targeting is visible at a glance |
| `SettingsTabList` mobile grid breaks with a sixth tab | Called out above; rebalance spans as part of the frontend phase |

## Open Questions

1. **Which project ID is LG Display?** Needed to create the first campaign. Resolvable
   from the projects list at publish time; does not block implementation.
2. **Should Partners see or manage ads for their own projects?** This spec assumes
   admin-only. Adding partner scope later means a Casbin entry plus a project-scope
   filter on the admin list — no schema change.
3. **Is an image/logo needed in the banner?** The current copy is text-only. The
   `assets` + `file_storage` pipeline exists if this is wanted later; adding an
   `image_asset_id` column is additive.
