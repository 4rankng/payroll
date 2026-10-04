# ADR-012: Untitled UI PRO Frontend Migration

Date: 2026-10-05 (migration executed 2026-10-04 → 2026-10-05)
Status: Accepted

## Context

The frontend was built on shadcn/ui + Radix primitives with a daisyUI layer (role-scoped `ct-` classes) — three overlapping systems with divergent tokens. The owner directed a full migration to Untitled UI (UU) PRO with "superb, elegant UI/UX," accepting theme drift while preserving the overall feeling (forest-green brand identity, compact data-dense practicality, role structure, all Vietnamese copy) and the repo's UI laws (WCAG 4.5:1, desktop/mobile parity at 1280/390/320, 44px touch targets, one-row controls).

## Decision

**Three-layer migration, executed in 15 waves (W0–W14 + W-final), one commit-series per wave:**

1. **Token bridge** (`tailwind.config.ts`): the full UU v7 token set in Tailwind 3.4 syntax with UU values as source of truth; the green brand + a new `utility-blue` ladder mapped onto UU color roles. All role overrides are defined at `:root` and paired with `html.{admin,partner,employee}-route-active` selectors — **never `[data-*-ui]` alone**, because Radix portals render outside those wrappers (the transparent-portal bug documented in the old alert-dialog).
2. **Vendored UU components** (`src/components/base|foundations|marketing/`), pinned to UU **v7**, sourced via the Untitled UI MCP/CLI, ported TW4→TW3 (`data-[x]:`, `[&>[attr]]:`, `!`-prefix), Vietnamese labels threaded at vendoring time.
3. **Compatibility layer** (`src/components/ui/*`): every primitive's internals restated in UU vocabulary while preserving its TypeScript contract 100% (513 importing files never changed). App-specific variants with no UU equivalent (button success/info/warning/monochrome, badge role variants) remain as cva entries on bridge tokens.

**Engine-keep exceptions (deliberate, permanent):** `dialog`/`sheet`, `select`, `dropdown-menu`, the date-picker stack, `accordion` (UU ships none), `sonner`, `command`/cmdk, and `sidebar` keep their Radix/cmdk/vaul/sonner engines under UU styling. Swapping them to React Aria Components was evaluated and rejected per family: bespoke mobile bottom-sheet + focus-return hooks, popper/scroll-button contracts, a different date value model, and missing UU equivalents. A full RAC swap remains a possible follow-up, not part of this migration.

**Rulings recorded during execution:**
- **tailwind-merge stays v2.** Upstream UU pairs with v3; the vendor CLI auto-bumps it (major). Vendored files are cx-ported to the repo v2 `cx`; vendored dirs are style references, never runtime dependencies of the compat layer.
- **`untitledui add` CLI** is used only with per-family approval, followed immediately by `git status` and revert of out-of-scope changes (it rewrites `cx.ts`, `is-react-component.ts`, and existing `base/**` files). Use `pnpm dlx` — plain `npx` fails with npm EOVERRIDE here.
- **Compat layer stays native-first where contracts demand it:** inputs/textarea keep native elements + `forwardRef` (react-hook-form `register()`); error states use `aria-[invalid=true]:` (TW3 has no `aria-invalid:` variant).
- **Icons** migrate lucide → `@untitledui/icons` per surface with free glyph choice. Shared icon-slot props are widened to `ComponentType<SVGProps<SVGSVGElement>>` (backward compatible), closing the lucide/`LucideIcon`-typed seam codebase-wide.
- **WCAG 4.5:1 release gate** enforced during migration: the info/admin/manager solid accents were darkened to 700-level palette steps (sky-600 measured 4.1:1, teal-600 3.8:1); a `utility-blue` ladder now exists in the bridge.

**Consequences:** daisyUI is retired per-surface (employee, auth, and partner surfaces are `ct-`-free; the remainder is tracked in the W-final sweep). Radix and react-aria-components coexist until the retirement audit removes zero-importer packages. The visual-regression harness (`playwright.visual.config.ts`, origin `localhost:3000` — the only origin the backend CORS whitelist allows) certifies every wave with deterministic pinned fixtures; its baselines are re-baselined per-surface with wave/commit attribution.

## Verification

Per wave: `pnpm lint` → `pnpm build` → `pnpm test:run` → targeted/full Playwright → certified visual diff (intended deltas re-baselined with attribution) → owner manual QA (kanban `DEV_COMPLETED` → `QA_TESTED`). Final gates at time of writing: 174 files / 814 vitest tests, auth e2e 18/18, employee-portal e2e 14/14, visual 24/24 (42/42 with auth riders) — three consecutive certified runs.
