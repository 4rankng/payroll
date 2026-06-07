<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# api — TanStack Query API Hooks

## Purpose

All data-fetching hooks using TanStack Query (React Query v5). Each hook wraps `useQuery` or `useMutation` with a service call from `services/api/` and proper query keys from `lib/queryKeys.ts`. These are the primary interface between React components and the backend API.

## Key Files

| File | Description |
|------|-------------|
| `useTimesheets.ts` | Timesheet CRUD, approval, import, export hooks |
| `useEmployees.ts` | Employee CRUD, import, status change hooks |
| `useProjects.ts` | Project CRUD, assignment, statistics hooks |
| `useProjectEmployees.ts` | Project-employee assignment hooks |
| `usePayRates.ts` | Payrate CRUD and configuration hooks |
| `useAdvancePayments.ts` | Advance payment request, approval, transfer hooks |
| `useLoans.ts` | Loan CRUD, repayment, disbursement hooks |
| `usePayrolls.ts` | Payroll processing hooks |
| `useAuth.ts` | Authentication (login, logout, token refresh) hooks |
| `useUsers.ts` | User management CRUD hooks |
| `useDashboard.ts` | Dashboard statistics and widget data hooks |
| `useLedger.ts` | Ledger entry query and reversal hooks |
| `useAudit.ts` | Audit log query hooks |
| `useAuditLogs.ts` | Audit log list hooks |
| `useAssets.ts` | Asset management hooks |
| `useBanks.ts` | Bank list query hooks |
| `useBankMutations.ts` | Bank CRUD mutation hooks |
| `useNotifications.tsx` | Notification list and push subscription hooks |
| `useProfile.ts` | User profile query and update hooks |
| `useSettings.ts` | Settings query and update hooks |
| `useEmails.ts` | Email sending and history hooks |
| `useCronHealth.ts` | Cron health status hooks |
| `useSystemHealth.ts` | System health monitoring hooks |
| `useAttendance.ts` | Attendance query hooks |
| `useAdminAttendance.ts` | Admin attendance management hooks |
| `useManualDisbursement.ts` | Manual disbursement creation hooks |
| `useEmployeePortal.ts` | Employee portal data hooks |
| `useAdvancePaymentFeeSchedules.ts` | Advance payment fee schedule hooks |
| `useAdvancePaymentReconciliation.ts` | Advance payment reconciliation hooks |
| `useDisbursementFeeSchedules.ts` | Disbursement fee schedule hooks |
| `useTimesheetEditRequests.ts` | Timesheet edit request hooks |

## For AI Agents

### Working In This Directory

- **Query hooks** return `{ data, isLoading, error, refetch }` from `useQuery`.
- **Mutation hooks** return `{ mutate, mutateAsync, isPending }` from `useMutation`.
- All query keys come from `src/lib/queryKeys.ts` — never hard-code arrays.
- Mutations must invalidate relevant query caches in `onSuccess` callback.
- Use optimistic updates for instant UI feedback on mutations.

### Testing Requirements

- Run `pnpm type-check` to verify hook types.
- Hooks are tested via E2E tests through the components that use them.

### Common Patterns

- **Query hook**:
  ```ts
  export function useEmployees(filters: EmployeeFilters) {
    return useQuery({
      queryKey: queryKeys.employees.list(filters),
      queryFn: () => employeeService.list(filters),
    })
  }
  ```
- **Mutation hook with invalidation**:
  ```ts
  export function useCreateEmployee() {
    const queryClient = useQueryClient()
    return useMutation({
      mutationFn: employeeService.create,
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: queryKeys.employees.all })
      },
    })
  }
  ```

## Dependencies

### Internal
- `../../services/api/` for Axios service modules
- `../../lib/queryKeys.ts` for query key constants
- `../../types/` for TypeScript interfaces

### External
- `@tanstack/react-query` (useQuery, useMutation, useQueryClient, useInfiniteQuery)

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
