# UI Guidelines

Design system, component conventions, and accessibility rules for the payroll frontend. See [Frontend AGENTS.md](../../frontend/AGENTS.md) for the full DO/DON'T list and [Code Standards](../code-standards.md) for frontend coding conventions.

## Design System: Navy & Gold

The frontend uses a **Navy & Gold** design system built on shadcn/ui. Theme tokens are defined in `frontend/tailwind.config.ts`.

### Philosophy

1. **Composition over configuration** — Build with primitives, not heavy abstractions.
2. **Consistency first** — Typography, spacing, and theme tokens must align with the system.
3. **Type-safety at all costs** — Strict TypeScript everywhere.
4. **Accessibility as a feature** — Build accessible-first.
5. **Avoid over-abstraction** — Keep utilities transparent and minimal.

## File Responsibility Rule (NON-NEGOTIABLE)

- **`.tsx` files** — UI/UX rendering only. No business logic, no API calls, no data transformation.
- **`.ts` files** — Business logic, data manipulation, API calls, type definitions, utilities.

## Component Patterns

### shadcn/ui

- Use shadcn/ui primitives from `@/components/ui/` — small, composable, accessible-first.
- Domain components live in `src/components/<domain>/` (e.g., `timesheet/`, `wallet/`, `ledger/`, `employees/`).
- **Barrel exports**: Every component directory has an `index.ts` re-exporting public API.
- **No duplicate components** — Don't maintain multiple versions. Don't prefix with "Modern", "Improved", "Enhanced". Keep only the best version.

### Naming

| Element | Convention | Example |
|---------|-----------|---------|
| Handlers | `handleXxx` | `handleSubmit`, `handleEmployeeSelect` |
| Render helpers | `renderXxx` | `renderEmployeeRow`, `renderEmptyState` |
| Components | PascalCase | `EmployeeTable`, `WalletBalanceCard` |
| Files | kebab-case | `employee-table-desktop.tsx`, `wallet-balance-card.tsx` |

### Mobile Variants

Mobile-specific components exist alongside desktop variants:
```
components/employees/
  employee-table-desktop.tsx
  employee-table-mobile.tsx
  index.ts
```

### Modal System

Centralized modal registry in `src/lib/modal-registry-auto.ts` with deep-link support. Use the registry rather than ad-hoc modal state.

## TanStack Query

All API data fetching goes through hooks in `src/hooks/api/`:

```typescript
// Query hook
export function useEmployees(filters: EmployeeFilters) {
  return useQuery({
    queryKey: ['employees', filters],
    queryFn: () => api.employees.list(filters),
  });
}

// Mutation with cache invalidation
export function useCreateEmployee() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (data: CreateEmployeeInput) => api.employees.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['employees'] });
    },
  });
}
```

- **Optimistic updates**: Mutations update cache immediately, rollback on error.
- **Query key conventions**: `['entity', filters]`, `['entity', 'detail', id]`.

## Forms

`react-hook-form` with `zod` validation via `@hookform/resolvers/zod`:

```typescript
const schema = z.object({
  name: z.string().min(1, 'Tên không được để trống'),
  phone: z.string().regex(/^[0-9]{10}$/, 'Số điện thoại không hợp lệ'),
});

const form = useForm({ resolver: zodResolver(schema) });
```

## Imports

Order: React → UI libs → icons → hooks → local components → types. Use `@/` path aliases.

## Vietnamese UI Text

All user-facing text is in Vietnamese. No i18n layer — strings are inline in components.

| English | Vietnamese |
|---------|-----------|
| Employee | Nhân viên |
| Timesheet | Bảng chấm công |
| Payroll | Bảng lương |
| Wallet | Ví |
| Advance Payment | Ứng lương (FlexPay) |
| Bulk Transfer | Chuyển khoản hàng loạt |
| Pending Payment | Chờ thanh toán |
| Confirmed | Đã chốt |

## Mobile-First

Employee views are mobile-first — they are the primary users on mobile devices.

### Hooks

- `useIsMobile` — Returns `true` on mobile viewport.
- `useBreakpoint` — Returns current breakpoint.

### Responsive Patterns

- Use `MobileShell` wrapper for employee mobile pages.
- Mobile-specific components: `WalletHero`, `EmployeeHomeModel`.
- Desktop layouts use a sidebar + content area; mobile uses a bottom nav + stacked content.

### Touch Targets

Minimum **44px** tap targets for all interactive elements on mobile.

## Accessibility

- **ARIA roles** — Interactive elements have appropriate `role`, `aria-label`, `aria-describedby`.
- **Focus states** — Focus is visible. Tab order follows visual order.
- **Keyboard navigation** — All interactive elements are operable via keyboard.
- **Color contrast** — Meets WebAIM Contrast Checker standards.
- **Screen reader** — Test with VoiceOver (macOS/iOS) and TalkBack (Android).

## PWA

- **Workbox** service worker for asset caching.
- **PWA manifest** in `frontend/public/`.
- **Web Push notifications** via VAPID (`webpush-go` on backend, browser Push API on frontend).
- **Push subscriptions** managed via `/api/v1/push` endpoints.

## React Best Practices

1. Always include dependency arrays in `useEffect`.
2. Memoize functions/objects/elements with `useMemo`/`useCallback` to prevent infinite loops.
3. Add cleanup functions in `useEffect` when updating parent state.
4. Split effects by concern — keep dependencies minimal.
5. Use `React.memo` / selective updates to skip unnecessary renders.
6. Memoize React elements passed to parents via callbacks.

### Don'ts

- **DON'T** auto-run prettier or `lint:fix` without user control.
- **DON'T** depend on and update the same state in a single `useEffect` without functional updates.
- **DON'T** use fallback values that hide real issues.
- **DON'T** pass new React elements in `useEffect` without memoization.
- **DON'T** hard-code components in pages — always create reusable components in the component library.
- **DON'T** run the frontend dev server automatically.
