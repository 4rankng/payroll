<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# services — API Service Layer

## Purpose

Axios-based API service modules that communicate with the backend REST API. Each service file encapsulates HTTP calls for a specific domain (employees, timesheets, projects, etc.). The `api/` subdirectory contains the Axios client configuration with interceptors for auth tokens and error handling.

## Key Files (Root Level — Type Definitions)

| File | Description |
|------|-------------|
| `api.types.ts` | Base API response types (pagination, APIResult wrapper) |
| `filter.types.ts` | Filter parameter types for list queries |
| `modal-config.types.ts` | Modal configuration type definitions |
| `modal.types.ts` | Modal state type definitions |
| `table.types.ts` | Table column and data types |
| `user.ts` | User-related type definitions |
| `timesheet.ts` | Timesheet type definitions (status enums, filter types) |
| `payrates.ts` | Payrate type definitions |
| `financial.ts` | Financial type definitions |
| `partnerProject.ts` | Partner project type definitions |
| `approval.ts` | Approval type definitions |
| `global.d.ts` | Global type declarations |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `api/` | Axios client and per-domain service modules |
| `attendance/` | Attendance tracking service |

## For AI Agents

### Working In This Directory

- **Never call Axios directly** in hooks or components — always go through a service module.
- Each service file corresponds to a backend domain module.
- Service methods return typed responses using the types in `api.types.ts`.
- Error handling is centralized in the Axios client interceptor (`api/client.ts`).
- The base URL and auth token injection are configured in `api/client.ts`.

### Testing Requirements

- Services are tested via E2E tests through the Playwright API helpers.
- Run `pnpm type-check` to verify service method signatures match types.

### Common Patterns

- **Service module pattern**: Export an object with methods (list, get, create, update, delete).
- **API response type**: `ApiResult<T>` wraps backend responses with success/error data.
- **Pagination**: List methods accept `page`, `pageSize`, and filter parameters.

## Dependencies

### Internal
- `../types/` for domain type definitions
- `../lib/auth.ts` for auth token retrieval

### External
- Axios HTTP client

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
