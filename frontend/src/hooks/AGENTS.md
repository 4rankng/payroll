<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# hooks — Custom React Hooks

## Purpose

Custom React hooks organized into API hooks (TanStack Query), feature-specific hooks, and utility hooks. The `api/` subdirectory contains all data-fetching hooks that wrap TanStack Query's `useQuery` and `useMutation` with service calls. Feature subdirectories contain hooks for specific domains. Root-level hooks are utility hooks used across the application.

## Key Files (Root Level — Utility Hooks)

| File | Description |
|------|-------------|
| `use-infinite-scroll.tsx` | Infinite scroll container hook for virtualized lists |
| `use-media-query.tsx` | CSS media query matching hook |
| `use-mobile.tsx` | Mobile device detection hook |
| `useIsMobile.ts` | Simplified mobile detection (used in most components) |
| `useBreakpoint.ts` | Responsive breakpoint detection |
| `useDebounce.ts` | Generic debounce hook |
| `useDebouncedInput.ts` | Debounced input value with controlled/uncontrolled modes |
| `useModalManager.ts` | Modal lifecycle management |
| `useModalNavigation.ts` | URL-based modal navigation and deep-linking |
| `useModalSystemInit.ts` | Modal system initialization hook |
| `useSecureModal.ts` | Security-checked modal access hook |
| `useDashboardNavigation.ts` | Dashboard navigation state and actions |
| `usePushNotifications.ts` | Web Push notification subscription management (VAPID) |
| `useNotificationToasts.ts` | Toast notifications from push events |
| `useNotificationPageLoad.ts` | Notification fetching on page load |
| `usePageShortcuts.ts` | Keyboard shortcut registration for page actions |
| `useTabDeepLink.ts` | URL-based tab state synchronization |
| `useCountUp.ts` | Animated number counter hook |
| `useAutoResizeTextarea.ts` | Auto-resizing textarea hook |
| `useBellAnimation.ts` | Notification bell animation trigger |
| `useCanEditProject.ts` | Project edit permission check |
| `useProjectCodeGenerator.ts` | Auto-generate project codes |
| `useProjectStatsConfig.ts` | Project statistics column configuration |
| `useEmployeeStatsConfig.ts` | Employee statistics column configuration |
| `useTimesheetStatsConfig.ts` | Timesheet statistics column configuration |
| `useLoanCalculations.ts` | Loan amortization and interest calculations |
| `useLoanValidation.ts` | Loan form validation logic |
| `useSenderProfiles.ts` | Notification sender profile management |
| `useDisbursementSettings.ts` | Disbursement fee configuration hook |
| `useInfiniteScroll.ts` | Infinite scroll pagination hook |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `api/` | TanStack Query API hooks — all data fetching (see `api/AGENTS.md`) |
| `admin/` | Admin-specific hooks |
| `admin-dashboard/` | Admin dashboard data hooks |
| `advance-payment/` | Advance payment hooks |
| `approvals/` | Approval workflow hooks |
| `business/` | Business logic hooks |
| `employees/` | Employee domain hooks |
| `ledger/` | Ledger entry hooks |
| `partner/` | Partner role hooks |
| `partner-dashboard/` | Partner dashboard data hooks |
| `partner-employees/` | Partner employee hooks |
| `partner-timesheet/` | Partner timesheet hooks |
| `projects/` | Project domain hooks |
| `settings/` | Settings hooks |
| `shared/` | Shared cross-domain hooks |
| `timesheet/` | Timesheet domain hooks |
| `transactions/` | Transaction domain hooks |
| `users/` | User management hooks |

## For AI Agents

### Working In This Directory

- **API hooks go in `api/`** — these are the TanStack Query wrappers.
- Feature hooks go in their respective subdirectories.
- Utility hooks (not API-bound) go at the root level.
- All API hooks must use query keys from `src/lib/queryKeys.ts` for cache consistency.
- Mutations must invalidate relevant query keys on success.

### Testing Requirements

- Hooks are tested indirectly through E2E tests and component interaction.
- Run `pnpm type-check` to verify hook type signatures.

### Common Patterns

- **useQuery pattern**: `useQuery({ queryKey: [...], queryFn: () => service.method() })`.
- **useMutation pattern**: `useMutation({ mutationFn, onSuccess: invalidate queries })`.
- **Query key factory**: Use `queryKeys.ts` constants, not inline string arrays.

## Dependencies

### Internal
- `../services/api/` for Axios service modules
- `../lib/queryKeys.ts` for query key constants
- `../types/` for TypeScript interfaces

### External
- TanStack Query v5 (`@tanstack/react-query`)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
