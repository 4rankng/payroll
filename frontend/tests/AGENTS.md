<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# tests — Playwright E2E Tests

## Purpose

End-to-end test suite using Playwright. Tests cover authentication, employee management, project CRUD, and timesheet workflows. Uses the Page Object Model pattern with shared fixtures, API helpers, and custom assertions.

## Key Files

| File | Description |
|------|-------------|
| `playwright.config.ts` | Playwright configuration (base URL, browser targets, timeouts) |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `e2e/` | E2E test spec files |
| `fixtures/` | Test fixtures (auth, global setup/teardown, test data) |
| `page-objects/` | Page Object Model classes for UI interaction |
| `utils/` | Test utilities (API helpers, assertions) |
| `mcp-config/` | MCP server configuration for test environments |

### e2e/

| File | Description |
|------|-------------|
| `auth.spec.ts` | Login/logout authentication flow tests |
| `employees.spec.ts` | Employee CRUD, import, and list filtering tests |
| `projects.spec.ts` | Project creation, editing, assignment tests |
| `timesheet.spec.ts` | Timesheet entry, approval, and export tests |

### fixtures/

| File | Description |
|------|-------------|
| `auth.fixture.ts` | Authentication fixture providing logged-in context |
| `global-setup.ts` | Global test setup (server health check) |
| `global-teardown.ts` | Global test teardown |
| `test-data.ts` | Shared test data constants |

### page-objects/

| File | Description |
|------|-------------|
| `DashboardPage.ts` | Dashboard page interactions |
| `EmployeesPage.ts` | Employee page interactions |
| `LoginPage.ts` | Login page interactions |

### utils/

| File | Description |
|------|-------------|
| `api-helpers.ts` | API call helpers for test setup/teardown |
| `assertions.ts` | Custom Playwright assertions |

## For AI Agents

### Working In This Directory

- All tests use the Page Object Model — create new page objects in `page-objects/` for new features.
- Use `auth.fixture.ts` for authenticated test contexts.
- Use `api-helpers.ts` for API calls in test setup (creating test data, cleanup).
- Use `assertions.ts` for custom assertions instead of raw Playwright expects.
- Test data goes in `test-data.ts`, not inline in specs.

### Testing Requirements

- Run with `pnpm test:e2e` from `frontend/` root.
- Run with UI: `pnpm test:e2e:ui`.
- Backend must be running on port 8080 before tests execute.
- `global-setup.ts` verifies backend health before running specs.

### Common Patterns

- **Test structure**: `test.describe('Feature', () => { test('does X', async ({ page }) => { ... }) })`.
- **Page Object**: Class with locators and action methods, instantiated in test.
- **Auth fixture**: `test.use({ storageState: 'auth-state.json' })` for authenticated tests.
- **API cleanup**: Use `api-helpers.ts` in `afterEach` to clean up test data.

## Dependencies

### Internal
- Application running at `http://localhost:5173` (dev server)

### External
- Playwright test runner, Chromium/Firefox/WebKit browsers

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
