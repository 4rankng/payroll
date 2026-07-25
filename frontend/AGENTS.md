<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-07-25 -->

# Frontend — Payroll Management System

## Purpose

React 18 + TypeScript + Vite 6 frontend for a Vietnamese payroll management system. Three user roles (Admin, Partner, Employee) each have dedicated page sets. The app uses a Navy & Gold design system built on shadcn/ui, TanStack Query for data fetching, and is a PWA with web push notifications. All user-facing text is in Vietnamese.

## Key Files

| File | Description |
|------|-------------|
| `package.json` | Dependencies and scripts (pnpm) |
| `vite.config.ts` | Vite build config with path aliases |
| `tailwind.config.ts` | Tailwind CSS theme (Navy & Gold tokens) |
| `tsconfig.json` | TypeScript strict mode config |
| `index.html` | SPA entry HTML |
| `CLAUDE.md` | Detailed style guide, patterns, and command reference |
| `.env.example` | Environment variables template |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `src/` | Application source code (see `src/AGENTS.md`) |
| `tests/` | Playwright E2E tests (see `tests/AGENTS.md`) |
| `public/` | Static assets, PWA manifest, service worker |

## For AI Agents

### File Responsibility Rule (NON-NEGOTIABLE)

- **`.tsx` files** — UI/UX rendering only. No business logic, no API calls, no data transformation.
- **`.ts` files** — Business logic, data manipulation, API calls, type definitions, utilities.

### Desktop and Mobile Parity (NON-NEGOTIABLE)

- Develop every frontend feature and bug fix for desktop and mobile in the same task for both Admin and Partner. A change is incomplete until both views of every affected role are implemented and verified.
- Audit Admin and Partner independently. Desktop/mobile parity is required within each role, while role-specific capabilities and permissions must remain intentionally distinct between roles.
- Before editing, identify whether the route uses shared responsive markup or swaps to dedicated desktop/mobile pages, components, headers, tables, cards, dialogs, sheets, or navigation. Update every active render path.
- Keep feature capabilities aligned across views: actions, buttons, filters, states, data, validation, permissions, error handling, and workflow outcomes must remain equivalent unless the user explicitly approves a platform-specific difference.
- Mobile may use a different presentation—cards, bottom sheets, overflow menus, stacked controls—but must not silently omit desktop functionality. Put space-constrained actions in an accessible menu or sheet.
- Share hooks, services, mutations, query keys, authorization rules, and domain behavior wherever possible so desktop and mobile cannot drift independently.
- Add regression coverage when desktop and mobile use separate components or callback wiring. Tests must fail if an Admin or Partner workflow disappears from either view.
- Verify authenticated Admin and Partner desktop views at 1280px or wider and their mobile views at 390px for every shared or role-sensitive change. Also test 320px for dense content, long Vietnamese text, full currency values, tables, dialogs, sheets, and other overflow-prone UI.
- Browser QA must confirm feature/action parity, correct data and permissions, readable wrapping, no horizontal scrolling, keyboard and screen-reader semantics, visible focus, and touch targets of at least 44px.

### DOs

**Code & Architecture**
1. Follow shadcn/ui philosophy: small, composable, accessible-first components.
2. Enforce TypeScript strict typing everywhere.
3. Keep separation of concerns: UI in `.tsx`, logic in `.ts`.
4. Use `@/` path aliases for clean imports.
5. Group imports: React -> UI libs -> icons -> hooks -> local components -> types.
6. Extract helpers (title mapping, filtering) into utility functions in `src/utils/`.
7. Split large JSX blocks into smaller render functions or sub-components.
8. Consistent naming: `handleXxx` for handlers, `renderXxx` for render helpers.
9. Refer to API spec at `/Users/dev/Documents/clients/payroll-backend/docs/api/` when unsure.

**React & Hooks**
10. Always include dependency arrays in `useEffect`.
11. Memoize functions/objects/elements with `useMemo`/`useCallback` to prevent infinite loops.
12. Add cleanup functions in `useEffect` when updating parent state.
13. Split effects by concern — keep dependencies minimal.
14. Use `React.memo` / selective updates to skip unnecessary renders.
15. Memoize React elements passed to parents via callbacks.

**UI/UX**
16. Vietnamese for all user-facing text.
17. Ensure color contrast (WebAIM Contrast Checker).
18. ARIA roles, focus states, keyboard navigation.
19. Minimum 44px tap targets for interactive elements.
20. Do optimistic updates for mutations.

### DON'Ts

1. **DON'T** auto-run prettier or `lint:fix` without user control.
2. **DON'T** depend on and update the same state in a single `useEffect` without functional updates.
3. **DON'T** use fallback values that hide real issues.
4. **DON'T** compromise quality with "quick fixes".
5. **DON'T** prefix components with "Modern", "Improved", "Enhanced" — keep only the best version.
6. **DON'T** maintain multiple versions of the same component.
7. **DON'T** build for legacy support.
8. **DON'T** pass new React elements in `useEffect` without memoization.
9. **DON'T** depend on non-memoized functions in `useEffect`.
10. **DON'T** forget cleanup functions when `useEffect` updates parent state.
11. **DON'T** run the frontend dev server automatically.
12. **DON'T** implement nice-to-have features.
13. **DON'T** hard-code components in pages — always create reusable components in the component library.

### Project Philosophy

- **Composition over configuration** — build with primitives, not heavy abstractions.
- **Consistency first** — typography, spacing, theme tokens must align with the system.
- **Type-safety at all costs** — strict TypeScript everywhere.
- **Accessibility as a feature** — build accessible-first.
- **Avoid over-abstraction** — keep utilities transparent and minimal.

### Development Commands

```bash
pnpm install          # Install dependencies
pnpm dev              # Dev server at http://localhost:5173
pnpm build            # Production build
pnpm lint             # ESLint check
pnpm lint:fix         # ESLint auto-fix (with user approval)
pnpm type-check       # TypeScript compiler check
pnpm test:e2e         # Playwright E2E tests
```

### Testing Requirements

- E2E tests use Playwright (see `tests/AGENTS.md`).
- No unit tests currently; Vitest setup planned.
- Run `pnpm type-check` and `pnpm lint` before committing.

### Common Patterns

- **TanStack Query**: All API data fetching via `useQuery`/`useMutation` hooks in `src/hooks/api/`.
- **Barrel exports**: Every directory has an `index.ts` re-exporting public API.
- **Optimistic updates**: Mutations update cache immediately, rollback on error.
- **Modal system**: Centralized modal registry in `src/lib/modal-registry-auto.ts` with deep-link support.
- **Responsive**: Mobile-first components with `useIsMobile` hook and mobile-specific variants.

## Dependencies

### Internal
- Backend API at `http://localhost:8080/api/v1` (dev mode)
- API client configured in `src/services/api/client.ts`

### External
- **React 18** + **React Router v6**
- **TanStack Query** (React Query v5)
- **shadcn/ui** + **Radix UI** primitives
- **Tailwind CSS** v3
- **Vite 6** build tool
- **Axios** HTTP client
- **Zod** schema validation
- **react-hook-form** form management
- **date-fns** date utilities
- **Recharts** charting library
- **workbox** PWA service worker

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->


QA Testing

localhost:3000

admin login
frankng
Admin123

partner login
cuongnv
Admin123

all accounts have password Admin123
