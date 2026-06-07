<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# types — TypeScript Type Definitions

## Purpose

Shared TypeScript type definitions used across the application. Contains interfaces for API responses, domain entities (employees, projects, timesheets, etc.), filter parameters, table configurations, modal configurations, and global type declarations. These types mirror the backend Go struct definitions and ensure end-to-end type safety.

## Key Files

| File | Description |
|------|-------------|
| `api.ts` | Core API response types — `ApiResult<T>`, pagination, error types |
| `user.ts` | User, Role, Permission interfaces |
| `timesheet.ts` | Timesheet, TimesheetEntry, timesheet status enums, filter types |
| `payrates.ts` | Payrate configuration types (flexible and matrix modes) |
| `financial.ts` | Transaction, Wallet, Ledger, Settlement types |
| `partnerProject.ts` | Partner-scoped project view types |
| `approval.ts` | Approval workflow types |
| `filter.types.ts` | Generic filter parameter types for list endpoints |
| `table.types.ts` | DataTable column definition types |
| `modal-config.types.ts` | Modal registry configuration types |
| `modal.types.ts` | Modal state and navigation types |
| `sheetjs.d.ts` | SheetJS type declarations for Excel import/export |
| `global.d.ts` | Global type declarations and augmentations |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `api/` | API-specific type submodules |
| `ext/` | Extended/third-party type augmentations |

## For AI Agents

### Working In This Directory

- Types must match backend API response shapes exactly.
- Use Zod schemas in `src/schemas/` for runtime validation; types here are compile-time only.
- When adding a new API endpoint, add the response type here first.
- Enum values should match backend string constants.

### Testing Requirements

- Run `pnpm type-check` to verify type consistency across the codebase.

### Common Patterns

- **ApiResult<T>**: Wraps all API responses with `{ data: T, message: string }`.
- **PaginatedResponse<T>**: `{ items: T[], total: number, page: number, pageSize: number }`.
- **Status enums**: Union types for entity states (e.g., `TimesheetStatus = 'pending' | 'approved' | 'paid'`).

## Dependencies

### Internal
- Used by `../hooks/`, `../services/`, `../components/`, `../utils/`

### External
- TypeScript built-in types only

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
