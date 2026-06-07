<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-06-07 | Updated: 2026-06-07 -->

# src — Application Source Code

## Purpose

Container directory for all application source code. Entry points (`main.tsx`, `App.tsx`) bootstrap the React app with routing, contexts, and TanStack Query provider. Global styles (`index.css`) define the Tailwind layer with Navy & Gold design tokens.

## Key Files

| File | Description |
|------|-------------|
| `main.tsx` | App entry point — renders `<App />` with providers (QueryClient, AuthContext, etc.) |
| `App.tsx` | Root component with route definitions, sidebar/header layout, and modal system |
| `index.css` | Global styles — Tailwind directives, CSS custom properties, design token overrides |
| `sw.ts` | Service worker for PWA (workbox) with push notification handling |
| `vite-env.d.ts` | Vite client type declarations |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `components/` | Reusable React components (see `components/AGENTS.md`) |
| `pages/` | Page-level route components by role (see `pages/AGENTS.md`) |
| `hooks/` | Custom React hooks (see `hooks/AGENTS.md`) |
| `services/` | API service layer (see `services/AGENTS.md`) |
| `types/` | TypeScript type definitions (see `types/AGENTS.md`) |
| `utils/` | Utility functions (see `utils/AGENTS.md`) |
| `lib/` | Core library utilities (see `lib/AGENTS.md`) |
| `config/` | App configuration and route config (see `config/AGENTS.md`) |
| `constants/` | App-wide constants (branding, email defaults, timesheet constants, modal registry) |
| `contexts/` | React context providers (Auth, AppState, CommandPalette, UserPreferences, etc.) |
| `schemas/` | Zod validation schemas for modals |
| `layouts/` | Layout components (`AdminLayout`, `PartnerLayout`) |
| `styles/` | Additional CSS (react-datepicker styles) |

## For AI Agents

### Working In This Directory

- Path alias `@/` maps to this directory (`src/`).
- `App.tsx` is the routing hub — all page routes and modal routes are defined here.
- `index.css` contains Tailwind `@layer` overrides for the design system; edit tokens there, not in components.
- `sw.ts` handles push notification click events and cache strategies.

### Testing Requirements

- Run `pnpm type-check` from `frontend/` root to validate TypeScript.
- Run `pnpm lint` to check for unused imports and lint errors.

### Common Patterns

- **Provider nesting**: `main.tsx` wraps `App` with `QueryClientProvider`, `AuthProvider`, `AppStateProvider`, etc.
- **Route structure**: Routes split by role (`/admin/*`, `/partner/*`, `/employee/*`) with `ProtectedRoute` guards.
- **Barrel exports**: Most subdirectories export via `index.ts` files.

## Dependencies

### Internal
- `components/ui/` for base UI primitives.
- `hooks/api/` for all data-fetching hooks.
- `services/api/` for Axios client and service modules.
- `lib/` for query keys, cache invalidation, and storage utilities.

### External
- React 18, React Router v6, TanStack Query v5, Axios, Zod, workbox

<!-- MANUAL: Any manually added notes below this line are preserved on regeneration -->
