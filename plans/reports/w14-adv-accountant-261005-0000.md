# W14 — Adv-partner + Accountant (UU PRO migration)

Date: 2026-10-05. Owner: w14-adv-accountant. Scope: `frontend/src/pages/adv-partner/**`, `frontend/src/pages/accountant/**`.

## What changed

### `frontend/src/pages/adv-partner/UsersPage/index.tsx` (rebuilt on UU vocabulary)
- lucide → verified `@untitledui/icons` (`SearchLg, ChevronRight, Building02, CreditCard01, User01, Mail01, CheckVerified01`), all decorative icons `aria-hidden="true"`.
- Old tokens → UU bridge families: `text-foreground`→`text-fg-primary`, `text-muted-foreground`→`text-fg-tertiary`, `foreground/80`→`text-fg-secondary`, translucent `border-border/*` + `bg-muted/*` → solid `border-utility-gray-100/200`, `bg-utility-gray-25/50`, containers `rounded-2xl`→`rounded-lg` + `shadow-xs`.
- Raw emerald/blue accents → semantics: project = brand family (`text-fg-brand-secondary`, dot `bg-utility-brand-500`); bank metadata neutralized to `fg-secondary/quaternary` (no blue ladder in the bridge, W2 ruling).
- Search field now the W3 UU `Input` primitive (`type="search"` keeps the `role="searchbox"` + aria-label contract); hand-rolled load-more buttons → `Button variant="outline"`.
- Title → `font-display text-display-xs sm:text-display-sm` (bridge compact display scale); table headers UU sentence case (`text-fg-tertiary` on `bg-utility-gray-50`, uppercase transform dropped — copy identical); row hover `bg-utility-gray-50`; focus rings standardized on the UU `outline-brand` pattern (row name button, scroll region, cards).
- Error alert gains the single `motion-safe:animate-fade-in-up` reveal (W10 idiom).

### `frontend/src/pages/adv-partner/UsersPage/EditAdvPartnerUserSheet.tsx`
- lucide → verified UU icons (`User01, CreditCard01, Building02, Lock01, Eye, EyeOff, CheckCircle, XClose`); section header chips → `bg-utility-gray-100` + `text-fg-tertiary` (uppercase dropped).
- **Error states moved to the UU-native `aria-invalid` channel**: inputs pass `aria-invalid={!!errors.x || undefined}` and the W3 Input primitive renders `border-utility-error-300` + `outline-error`; manual `border-destructive` classes removed. Field error text → `text-fg-error-primary`.
- Surfaces: section bodies `bg-utility-gray-25` + `border-utility-gray-200`; username read-only box, project pill (`bg-utility-brand-50/200`), divider, SheetHeader border all on utility ladders; close + eye toggles get `outline-brand` focus rings; password-save confirmation recolored `text-emerald-700` → `text-fg-success-primary` (success semantics).
- Sheet (Radix) engine, mobile bottom/right side logic, all form state, validation copy, toast copy, invalidation key `['admin','advance-payments','flex-pay-employees']` untouched.

### `frontend/src/pages/accountant/AccountantPage.tsx`
- lucide → verified UU icons (`CalendarCheck01, FileDownload02` [Excel export], `Upload01, FileDownload01, LogOut01`); dropped redundant `mr-1.5` (UU Button has `gap-1.5`); all icons `aria-hidden`.
- Tokens: `text-foreground`→`text-fg-primary`, `text-muted-foreground`→`text-fg-tertiary`, `text-destructive`→`text-fg-error-primary`, `text-warning`→`text-fg-warning-primary`, header border → `border-utility-gray-200`.
- Zero logic deltas: hooks, query options (`pageSize: 500`, sort, filters), selection effect, bulk-approve wiring, dialogs, tabs, logout handler identical.

### `frontend/src/pages/accountant/AccountantPage.test.tsx` (new)
- Thin contract tests (the accountant role has no e2e spec): header logout wiring; select-all/individual tick → `bulkApprove.mutate` receives exactly the ticked ids; week-change clears stale selection + hidden-rows warning renders.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit`: **zero errors in owned files** (grep of output for `adv-partner|accountant` empty). Remaining full-tree errors are in foreign in-flight files (see Concerns).
- Scoped vitest: **3 files / 10 tests passed** (`src/pages/adv-partner` 7 pre-existing — untouched and passing — + `src/pages/accountant` 3 new). 30s timeouts per load-artifact rule.
- Token sweep: no `text-foreground|text-muted-foreground|text-destructive|border-border|text-warning|bg-muted|border-destructive` left in owned files.
- No e2e gate exists for either role (no specs; noted as required by brief). Per-wave full gate (lint/build/test:run/e2e/visual) intentionally not run by this wave per constraints; commit-time combined gate owned by team-lead.

## Design notes / intentional deviations

- **Avatar identity gradients kept verbatim** (teal/sky/emerald/amber/rose/lime). They are per-person identity decoration, not chrome; the bridge has no multi-hue ladder set that would preserve the directory's at-a-glance distinction, and the liked-UI iterate-don't-replace ruling applies.
- Bank info moved from blue to neutral gray: metadata, not status; no blue ladder exists (W2 ruling).
- UU sentence-case table headers (uppercase transform dropped) — styling only; all Vietnamese copy byte-identical.

## Concerns

1. **Foreign tsc errors present in the tree (not mine, do not touch):** `src/components/employees/EmployeeTimesheetPanel.tsx` (`RefreshCw05` unresolved — likely a parallel wave mid-icon-swap) and `src/pages/admin/SettingsPage/index.tsx` + `src/pages/mobile/admin/SettingsPage/index.tsx` (5× `$$typeof` — UU `FunctionComponent` passed where a `LucideIcon` type is expected). These will block the combined gate until their owning wave fixes them.
2. No e2e specs cover either role — the visual-baseline harness presumably has no snapshots for `/accountant` or `/adv-partner` users either; recommend owner manual QA covers both roles at 1280/390/320.
3. One degraded-write incident: the sheet's first Write landed with two corrupted tokens (`fullname: regex-placeholder`, `errs <post-corruption-marker>`); both repaired by targeted edits and verified by grep + tsc + tests. Flagging per the known failure mode.

Status: DONE
Summary: Both single-page roles rebuilt on native UU PRO vocabulary (verified @untitledui/icons, bridge fg/utility/brand tokens, aria-invalid error channel, outline-brand focus rings, compact display scale) with 100% behavior/copy preservation — 10/10 scoped vitest (3 new accountant contract tests), owned files tsc-clean.
Concerns: foreign in-flight tsc errors (admin SettingsPage ×5, EmployeeTimesheetPanel ×1) block the full gate; no e2e coverage for these roles; one repaired degraded-write incident.
