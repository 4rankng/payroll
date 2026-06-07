<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# sheets — Sheet/Slide-Over Panel Components

## Purpose

Slide-over panel (Sheet) components for CRUD operations across all major entities. Each sheet is a form for creating or editing an entity, opened from table rows or action buttons. These are the primary data entry forms in the application.

## Key Files

| File | Description |
|------|-------------|
| `AddEmployeeSheet.tsx` | Create/edit employee form sheet |
| `AddProjectSheet.tsx` | Create new project form sheet |
| `ProjectEditSheet.tsx` | Edit existing project form sheet |
| `ProjectDetailsSheet.tsx` | Read-only project details view |
| `ProjectAssignmentSheet.tsx` | Employee-project assignment management |
| `AddUserSheet.tsx` | Create new user form sheet |
| `EditUserSheet.tsx` | Edit user form sheet |
| `UserDetailsSheet.tsx` | Read-only user details view |
| `UserDetailsSheetContainer.tsx` | Container wrapper for user details |
| `UserProfileSheet.tsx` | Current user profile editor |
| `EmployeeDetailsSheet.tsx` | Read-only employee details view |
| `TimesheetEntrySheet.tsx` | Timesheet entry create/edit form |
| `TimesheetEntrySheetWrapper.tsx` | Wrapper providing context for timesheet entry sheet |
| `AddTransactionSheet.tsx` | Manual transaction entry form |
| `TransactionDetailsSheet.tsx` | Transaction detail view |
| `AdvancePaymentSheet.tsx` | Advance payment processing sheet |
| `LoanDetailsSheet.tsx` | Loan detail view with repayment schedule |
| `AddLoanSheet.tsx` | Create new loan form |
| `AddLenderSheet.tsx` | Create new lender form |
| `EditLenderSheet.tsx` | Edit lender form |
| `LenderManagementSheet.tsx` | Lender management panel |
| `LoanTypeSelector.tsx` | Loan type selection component |
| `AutoInterestFields.tsx` | Auto-interest configuration fields |
| `CustomScheduleFields.tsx` | Custom repayment schedule fields |
| `PrincipalDateFields.tsx` | Principal and date fields for loans |
| `ScheduleDisplay.tsx` | Repayment schedule display |
| `LenderDisbursementSection.tsx` | Lender disbursement section |
| `FormField.tsx` | Reusable form field wrapper component |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `templates/` | Reusable sheet template components |
| `timesheet-entry/` | Timesheet entry sheet sub-components |

## For AI Agents

### Working In This Directory

- Each sheet handles both create and edit modes (detected by presence of initial data).
- Forms use react-hook-form with Zod validation schemas.
- Sheet open/close is managed by the modal navigation system in `src/lib/`.
- After successful mutation, sheets invalidate relevant TanStack Query caches.
- Use `FormField.tsx` for consistent form field styling.

### Testing Requirements

- Run `pnpm type-check` after changes.
- Form validation is tested via E2E tests.

### Common Patterns

- **Sheet form pattern**: `useForm` with Zod resolver -> render fields -> `useMutation` on submit -> close sheet on success.
- **Create/Edit detection**: If `initialData` prop is provided, form pre-fills for editing.
- **Cache invalidation**: `queryClient.invalidateQueries({ queryKey: [...] })` on mutation success.

## Dependencies

### Internal
- `../../hooks/api/` for mutation hooks
- `../../lib/queryKeys.ts` for cache key constants
- `../../schemas/` for Zod validation schemas
- `../ui/` for base form and sheet components

### External
- react-hook-form, @hookform/resolvers, Zod

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
