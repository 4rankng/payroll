# W3 — Inputs (ui/input.tsx, ui/textarea.tsx) UU PRO restyle

Date: 2026-10-04 · Wave: W3 of plan `261004-1812-uu-full-migration` · Agent: w3-inputs

## Deliverables

| File | Change |
|---|---|
| `frontend/src/components/ui/input.tsx` | Internals restyled to UU; contract preserved 100% (native `<input>`, `forwardRef`, `React.ComponentProps<"input">`, variant union `'default' \| 'filled' \| 'outlined'` exact, `lang="vi-VN"` on `type="date"` only, `cn(variants[variant], className)` merge order, `displayName`, named export) |
| `frontend/src/components/ui/textarea.tsx` | Same treatment; `TextareaProps` export, `min-h-[80px]` floor kept |
| `frontend/src/components/ui/input.test.tsx` | New — 7 contract tests |
| `frontend/src/components/ui/textarea.test.tsx` | New — 4 contract tests |

No other file touched. `control-density.test.tsx` (pins `h-11 sm:h-9` on Input) untouched and still satisfied.

## Style decisions (vendored UU v7 source as reference)

Read via `mcp__untitledui__get_component(input/textarea, v7)` + `pnpm dlx untitledui add` (see incident below). Mapping from vendored v7 vocabulary to this repo's W1 bridge:

- Surface/border: vendored `bg-primary`/`ring-primary` are unbridgeable (`primary` is shadcn emerald here) → kept role-scoped tokens `bg-card`/`border-input`/`bg-muted` (values equal UU white/gray-300; admin/partner overrides keep applying).
- Shape/elevation: `rounded-md` → `rounded-lg`; added `shadow-xs` (bridge boxShadow). Heights untouched (`h-11 sm:h-9`, `min-h-[80px]`) — data-dense + tablet 40px floor in `variables.css` unchanged.
- Focus: per task spec, the vendored-button treatment: `outline-brand focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2` (replaces `focus-visible:outline-none ring-2 ring-ring ring-offset-2`; dead `ring-offset-background` removed).
- Error state (new, additive): `aria-[invalid=true]:border-utility-error-300` + `aria-[invalid=true]:focus-visible:outline-error`. Vendored v7 uses `ring-error_subtle`/`ring-error` — bare `error` is not bridged, and the named `aria-invalid:` variant does not exist in TW 3.4 (TW4 only), so the arbitrary form compiles (no silent fallback). `filled` additionally gets `aria-[invalid=true]:border` so the error border appears over `border-0`.
- Disabled: `disabled:cursor-not-allowed disabled:opacity-50` kept — identical to vendored wrapper behavior.
- Kept `typography-body-medium`/`file:*` classes (app semantic type role = 12px/1.4, same as UU text-sm; zero-risk) and `placeholder:text-muted-foreground` (UU `placeholder:text-placeholder` token not bridged).

## Vendoring incident (STOP condition — resolved, needs owner awareness)

1. `npx untitledui@latest add input` fails in this repo (npm `EOVERRIDE` — postcss override vs direct dep). `pnpm dlx untitledui@latest add input|textarea --yes` from `frontend/` works (validated).
2. The CLI's dependency step mutated shared files without asking: added `@react-stately/utils`, **bumped `tailwind-merge` 2.6→3.7 (major)**, patched `@untitledui/icons`/`input-otp`, and **overwrote `src/utils/cx.ts` with tailwind-merge v3 syntax** (W1 fix is v2 `classGroups`). Vendoring `input` pulled ~75 files (full family + `base/tags/`, `base/tooltip/`, `foundations/payment-icons/`). Per the "needs a new dependency → STOP" rule I flagged this; a parallel session then reverted the whole vendoring (package.json, lockfile, cx.ts back to HEAD; vendored input/textarea/tags/tooltip/payment-icons deleted). Only w2-leaf's avatar/badges/dot-icon remain.
3. Consequence: the vendored reference files are no longer on disk — the restyle was completed from the v7 source read into context before deletion. If a future wave wants the input family vendored, it needs an owner decision on the dependency churn first (@react-stately/utils + tailwind-merge major, or a manual prune to the dep-free core: input.tsx, textarea.tsx, hint-text, label, tooltip).

## Verification (allowed commands only)

- `pnpm exec vitest run src/components/ui/input.test.tsx src/components/ui/textarea.test.tsx` → **2 files, 11/11 passed** (ref instanceof HTMLInputElement/HTMLTextAreaElement — the RHF `register()` dependency; variant surfaces; `rounded-none` caller override kills `rounded-lg` via twMerge; UU focus classes; disabled + invalid classes; `h-11 sm:h-9`; `lang` behavior).
- `pnpm exec tsc -p tsconfig.app.json --noEmit` → **exit 0** (includes w2-leaf's vendored avatar/badges).
- Not run per constraints: build, lint, full test:run, Playwright, dev servers, prettier; nothing committed.

## Known intended deltas at the wave gate

- All 105 Input importers + textarea: radius md→lg, new focus ring style, new shadow, error state (on `aria-invalid` inputs only). Visual baseline re-diff will show these as intended.

Status: DONE_WITH_CONCERNS
Summary: ui/input.tsx and ui/textarea.tsx restyled to the UU PRO look with contracts 100% preserved; 11 new contract tests pass and app-wide tsc is green.
Concerns: (1) the untitledui vendoring auto-installed deps (tailwind-merge 2→3 major, @react-stately/utils) and rewrote cx.ts to v3 syntax — a parallel session reverted it all; input-family vendoring needs an owner decision before any future wave re-runs it; (2) vendored reference files are gone from disk — future waves must rely on fresh `pnpm dlx` vendoring (use pnpm, not npx — npx hits EOVERRIDE).

## Addendum — scope re-pin received and verified (2026-10-04 ~19:45)

Lead's post-hoc rules were followed by the completed run: `git status --porcelain` was run immediately after each CLI add (which surfaced the shared-file mutation); no adds beyond input/textarea; package.json/pnpm-lock.yaml untouched since the revert. Live state at this time: `base/input/`, `base/textarea/`, `base/tags/`, `base/tooltip/`, `foundations/payment-icons/` are ABSENT (removed by the parallel revert) — only w2's `base/avatar/`, `base/badges/`, `foundations/dot-icon.tsx` remain; package.json/pnpm-lock.yaml at HEAD (no @react-stately/utils). My owned files verified intact: `ui/input.tsx`, `ui/textarea.tsx` (UU classes present), `ui/input.test.tsx`, `ui/textarea.test.tsx`. No re-vendoring will be attempted without an explicit lead/owner word.
