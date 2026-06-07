<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# lib — Core Library Utilities

## Purpose

Core infrastructure modules that power the application's data layer, navigation, and security. Contains the TanStack Query key registry (`queryKeys.ts`), cache invalidation service (`cache/`), IndexedDB persistence (`storage/`), authentication utilities (`auth.ts`), the modal navigation system, permissions, typography tokens, and validation helpers.

## Key Files

| File | Description |
|------|-------------|
| `queryKeys.ts` | Centralized TanStack Query key factory — all query keys defined here for cache consistency |
| `auth.ts` | Auth token storage, retrieval, and refresh logic |
| `permissions.ts` | Role-based permission constants and check functions |
| `validation.ts` | Zod-based validation schemas and helpers |
| `modal-registry-auto.ts` | Auto-generated modal registry mapping URL slugs to components |
| `modal-navigation.ts` | URL-based modal navigation (open/close/deep-link) |
| `modal-state-manager.ts` | Modal state persistence and restoration |
| `modal-permissions.ts` | Modal access control by user role |
| `modal-security.ts` | Modal security checks and audit logging |
| `modal-deeplinks.ts` | Modal deep-link configuration and URL generation |
| `realtime.ts` | Real-time event handling (SSE/WebSocket connection) |
| `typography.ts` | Typography token definitions for the design system |
| `responsive-utils.ts` | Responsive breakpoint and layout utility functions |
| `logger.ts` | Structured logging utility |
| `console-filter.ts` | Console log filtering for production |
| `favicon.ts` | Dynamic favicon generation based on notification state |
| `xlsx-loader.ts` | Lazy-loaded xlsx library loader |
| `utils.ts` | shadcn/ui `cn()` utility (clsx + tailwind-merge) |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `cache/` | TanStack Query cache invalidation registry and service |
| `queryKeys/` | Query key factory modules organized by domain |
| `storage/` | IndexedDB storage layer for offline persistence |

### cache/

| File | Description |
|------|-------------|
| `invalidationRegistry.ts` | Registry mapping mutation events to query key invalidation patterns |
| `invalidationService.ts` | Service that executes cache invalidation after mutations |
| `queryPersister.ts` | TanStack Query persister for IndexedDB sync |

### queryKeys/

| File | Description |
|------|-------------|
| `index.ts` | Barrel export of all query key factories |

### storage/

| File | Description |
|------|-------------|
| `indexedDB.ts` | IndexedDB wrapper for offline data persistence |
| `types.ts` | Storage type definitions |

## For AI Agents

### Working In This Directory

- **Query keys**: Always use keys from `queryKeys.ts` — never hard-code query key arrays in hooks.
- **Cache invalidation**: Register new invalidation patterns in `cache/invalidationRegistry.ts` when adding new mutations.
- **Modal system**: New modals must be registered in `modal-registry-auto.ts` for URL deep-linking to work.
- **Permissions**: Check `permissions.ts` before adding role-specific UI logic.

### Testing Requirements

- Run `pnpm type-check` to verify query key types and modal registry consistency.

### Common Patterns

- **Query key factory**: `queryKeys.employees.list(filters)` produces `['employees', 'list', { ...filters }]`.
- **Cache invalidation**: After a mutation, call the invalidation service which uses the registry to find affected queries.
- **Modal deep-link**: `modal-navigation.ts` converts modal state to URL search params.

## Dependencies

### Internal
- `../types/` for type definitions
- `../services/api/` for API client (used by persister)

### External
- TanStack Query, idb-keyval (IndexedDB), clsx, tailwind-merge, Zod

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
