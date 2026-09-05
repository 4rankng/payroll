# UI Guidelines

Design system, component conventions, and accessibility rules for the payroll frontend. See [Frontend AGENTS.md](../../frontend/AGENTS.md) for the full DO/DON'T list and [Code Standards](../code-standards.md) for frontend coding conventions.

## Design System: TingTing Emerald

The frontend uses a **TingTing Emerald** design system built on shadcn/ui (primary `#08783e`). Theme tokens are defined in `frontend/tailwind.config.ts` and `frontend/src/styles/variables.css`. Gold accents are reserved for the employee-portal gold card only — never on admin surfaces.

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

### Stat Band (Treasury)

For money-summary stat rows on admin pages. Vocabulary lives in `frontend/src/styles/premium.css` ("TREASURY INSTRUMENT BAND" section); reference implementation is the advance-payments hero band.

**Principles**

1. **One instrument, not a row of cards** — the stat row is a single `treasury-grid` surface; panels are separated by the grid's fading seams, never by boxed cards (max one Card nesting level).
2. **Metric ownership zones** — each panel exclusively owns its metrics. A metric must never appear in two adjacent zones; enforce ownership in a code comment.
3. **Presence = signal** — advisory panels render only in their signal state (e.g. the demand panel appears only on shortfall). No permanently-empty panels.
4. **Color = semantics, not decoration** — tone lands on the value (emerald ok / amber mid / red risk), never on chrome. The `--alert` variant swaps the whole panel wash to rose. Never invent a signal state the data cannot support.
5. **Financial typography** — `font-financial` (JetBrains Mono) + `tabular-nums` for every money value; `~` prefix for derived averages; `—` for empty.
6. **Backend owns derived math** — SQL aggregation on the backend; the frontend renders precomputed values.
7. **KPIs may double as filters** — tile selection syncs with the table's filter state so the two can never disagree.

**Classes**

| Class | Purpose |
|-------|---------|
| `treasury-grid` | Band wrapper; injects fading seams between panels automatically |
| `treasury-panel` (`--dark`, `--alert`) | Panel shell: resting wash + hover light-rail; dark = wallet panel, alert = rose |
| `treasury-mesh` (`--alert`) | Faint dot-grid measurement layer, always `aria-hidden` |
| `treasury-signal` | Pulsing LED status dot on panel labels |
| `treasury-rail` (`--dark`) | Footer divider with accent tick; pin to the shared baseline with `mt-auto` |
| `treasury-value` | Headline value blur-in |
| `treasury-aurora` | Drifting glow — dark panel only, **one per page max** |
| `band-sheen(--glass)`, `led-pulse` | Light sweep / generic LED |

**When to use / not use**

- Use: money-summary bands on admin pages (dashboard strips, payroll control center, wallet hero).
- Don't: CRUD/utility surfaces (audit log, settings, users), the employee portal (it has its own `--employee-*` token system), or a second `treasury-aurora` on any page.
- All effect layers are atmosphere, never information: `aria-hidden`, `pointer-events-none`, `prefers-reduced-motion` guards (already built into `premium.css`).

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

### Employee Action Dock

- Check-in-enabled employee pages keep a compact fixed action dock on mobile with `Ứng lương` as the secondary action and the current attendance action (`Vào làm` / `Tan ca`) as the primary action.
- Keep the dock to a 44px button height with a quiet top divider; reserve the wider slot for attendance and include bottom safe-area padding.
- Attendance status, GPS/geofence guidance, map disclosure, and shift cancellation stay in the attendance card. Do not duplicate the large attendance CTA inside that card on mobile.
- Disabled attendance states must explain themselves in the button label (`Đang kiểm tra GPS…`, `Chưa đến giờ`, `Đã tan ca`, or `Cần kiểm tra`) while leaving `Ứng lương` available.

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
