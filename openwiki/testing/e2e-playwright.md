---
type: testing
title: Playwright E2E
description: Frontend Playwright setup, the three viewports (1280/390/320), per-role matrix coverage (Admin / Partner / Employee), the desktop+mobile parity contract, and the testplan scenarios.
tags: [testing, e2e, playwright, viewports, parity, role-matrix]
verified:
  - by: openwiki/0.5.0
    at: 2026-09-19T03:19:10.016Z
sources:
  - id: openwiki-source-aca61ad26f85a3f6b9e0d210
    resource: repo://docs/decisions/ADR-006-clock-injection-pattern.md
  - id: openwiki-source-bf27fb010957bd7d3b81f9b1
    resource: repo://docs/testing.md
  - id: openwiki-source-e483fd3285d99d05c7b265cf
    resource: repo://frontend/AGENTS.md
  - id: openwiki-source-2090dca405aa9c3acd6c7ff8
    resource: repo://frontend/playwright.config.ts
  - id: openwiki-source-eddf02e0f8b63b3cc01e7e90
    resource: repo://frontend/src/pages/admin/list-error-parity.test.tsx
generated: { by: "opencode", at: "2026-09-19T03:19:10.016Z" }
---

# Testing: Playwright E2E

<!-- openwiki: broken internal link [../docs/testing.md] file "../docs/testing.md" does not exist. Fix the href or restore the target, then delete this comment. -->
The frontend test suite uses [Playwright](https://playwright.dev) with TypeScript. The configuration lives in `frontend/playwright.config.ts`; scenarios live in `frontend/tests/`; structured end-to-end cases live in `testplan/`. The full testing strategy is in [`docs/testing.md`](../docs/testing.md); this page is the E2E-specific summary.

## Viewport matrix

Three required viewports, exercised for every shared or role-sensitive change:

| Viewport | Width | Purpose |
|----------|-------|---------|
| Desktop | 1280px | Standard desktop |
| Mobile | 390px | Typical phone |
| Narrow | 320px | Dense content / long Vietnamese text / money / tables / dialogs / sheets |

The 320px breakpoint is required when content density, long Vietnamese text, full VND amounts, tables, dialogs, or sheets could overflow. The mobile parity rule means every change is verified at both 1280px and 390px for Admin and Partner independently; the 320px run is triggered when the surface is overflow-prone.

## Role matrix

| Role | Desktop | Mobile | Narrow |
|------|---------|--------|--------|
| Admin | ✓ | ✓ | when applicable |
| Partner | ✓ | ✓ | when applicable |
| Employee | ✓ (where it exists) | ✓ primary | when applicable |

The employee role is mobile-first; many employee flows do not have a desktop surface and are not tested at 1280px.

## Configuration

`frontend/playwright.config.ts` configures:

- `projects` per browser (Chromium, WebKit, Firefox).
- Per-viewport runs (1280×800, 390×844, 320×568).
- The local api-server URL (`http://localhost:8080/api/v1`).
- The local frontend URL (`http://localhost:5173`).
- Storage state for auth (per-role login fixtures).
- Tracing and screenshots on failure.

`pnpm test:e2e` runs the suite.

## Test layout

```
frontend/tests/
├── e2e/
│   ├── admin/          Admin scenarios
│   ├── partner/        Partner scenarios
│   ├── employee/       Employee scenarios (mobile-first)
│   └── shared/         Cross-role helpers (login, fixtures)
├── fixtures/           Auth, data builders
└── playwright.config.ts
```

Page Object Models are kept lean — one file per role-level page, with shared utilities in `shared/`. The `testplan/` directory at the repo root carries structured scenarios the Playwright suites cover.

## Desktop+mobile parity contract

The contract enforced by tests:

- A page that exists at desktop must have a mobile twin (or an explicit gating record).
- Mutation calls, query keys, and authorization must be identical across both views.
- Filter parity, sort parity, error-state parity are tested.
- Action parity: a button visible on desktop must be reachable on mobile (via sheet, menu, or footer action) within 44px touch targets.

`frontend/src/pages/admin/list-error-parity.test.tsx` is the desktop/mobile list-error parity contract test for the admin role.

## Auth and test data

- Per-role login fixtures (`auth/fixtures.ts`) seed a known user per role.
- The dev DB (`docker-compose.dev.yml`) is the test fixture DB. Tests run against a clean seed and reset between scenarios where the data shape requires.
- Clock control is exercised via `/api/v1/admin/clock/advance` (non-prod only).

## CI and the regression matrix

The Playwright suite is wired to CI. A change that breaks any of:

- A viewport at the role it targets
- The desktop+mobile parity contract
- The role authorization rules

fails the build. The full matrix (3 viewports × 3 roles × ~20 scenarios) runs in CI on every PR.

## Relationships

<!-- openwiki: broken internal link [../docs/testing.md] file "../docs/testing.md" does not exist. Fix the href or restore the target, then delete this comment. -->
- Testing strategy — [`docs/testing.md`](../docs/testing.md).
- Backend integration suite — `testing/integration.md`.
- Admin views — `frontend/admin-views.md`.
- Partner views — `frontend/partner-views.md`.
- Employee views — `frontend/employee-mobile.md`.
