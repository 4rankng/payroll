# W5 — Overlay restyle (dialog, alert-dialog, sheet, popover, tooltip)

Date: 2026-10-04 · Owner: w5-overlays · Plan of record: `plans/261004-1812-uu-full-migration/plan.md`

## Scope delivered

Restyle-in-place of the five overlay primitives to the UU token vocabulary. **Engine stays Radix** (plan engine-keep list); no dependency changes, no CLI adds, nothing vendored.

| File | Change |
|---|---|
| `frontend/src/components/ui/dialog.tsx` | Surface `border` → `border border-utility-gray-200` (W2 card recipe); `bg-card shadow-2xl` and per-side radii kept. Header strips `bg-emerald-950` → `bg-brand-section_subtle` (W1 bridge token added for exactly this strip); on-brand muted copy `text-emerald-200/75` → `text-tertiary_on-brand`. Navy-header close button hover/focus migrate from emerald palette literals to the white-on-brand idiom (`hover:bg-white/20`, `focus:ring-white/30`). All contracts untouched: `useIsMobile` bottom sheet + edge-to-edge style override, `useDialogFocusReturn` wiring, sr-only Title/Description fallbacks, `DialogNavyHeader` CSS-var overrides, `contentPadding`/`mobileOverlay`, exact `Đóng` label. |
| `frontend/src/components/ui/alert-dialog.tsx` | Content/footer borders → `border-utility-gray-200`; `bg-card`/`text-card-foreground` portal-safe tokens kept (daisyUI-portal comments preserved). `AlertDialogAction` restated in the W4 compat-button vocabulary: brand/error solids (`bg-brand-solid`/`bg-error-solid`, `_hover` actives, `shadow-xs-skeumorphic`, inner `before:` white/12 gradient, `ring-1 ring-transparent ring-inset`), UU outline focus ring (`outline-brand`/`outline-error`), UU disabled tokens (`bg-disabled`/`text-fg-disabled`/`ring-disabled_subtle`). App contract kept: `min-h-11` 44px targets, `px-4`, `normal-case`, `disabled:pointer-events-none`. `AlertDialogCancel` = W4 outline idiom on `bg-card`. |
| `frontend/src/components/ui/sheet.tsx` | Surface `bg-card shadow-sm` → `bg-card border border-utility-gray-200 shadow-lg`. Side variants, focus-return, mobile edge-to-edge override, sr-only fallbacks untouched. |
| `frontend/src/components/ui/popover.tsx` | `rounded-md border` → `rounded-xl border border-utility-gray-200`, `shadow-sm` → `shadow-lg`. Positioning (`align`/`sideOffset`), `w-72 p-4`, `outline-none`, animations untouched. |
| `frontend/src/components/ui/tooltip.tsx` | Same recipe at `rounded-lg` (see deviation note), `shadow-lg`. `typography-body-medium`, positioning untouched. |

## Contracts verified preserved

- Portal token rule: every surface uses `bg-card`/`bg-popover`/`text-card-foreground`/`text-popover-foreground` (:root tokens) — zero daisyUI-scoped classes (`ct-*`, `base-100`); alert-dialog's explanatory portal comments preserved verbatim.
- Engine keep: Radix Dialog/AlertDialog/Popover/Tooltip; **note: `sheet.tsx` never used vaul** — it is `@radix-ui/react-dialog` based; the brief's "vaul stays" was moot, engine untouched either way.
- Tailwind 3.4 syntax only; all tokens from the W1 bridge; `cn` as each file already used; Vietnamese strings unchanged (`Đóng` exact).

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit` — exit 0 (run after all edits).
- Vitest, only the 5 owned files: **21/21 passed** (`dialog` 6, `sheet` 5, `alert-dialog` 5, `popover` 1, `tooltip` 1; 11 new contract tests: variants render, `contentPadding`/`overlayClassName`/`mobileOverlay` flow preserved, focus-return smoke, portal-cleanliness assertion, UU surface classes).
- Tailwind emission check (isolated content scan of the 5 files, per the repo's no-silent-fallback bar): every new class resolves — incl. `bg-brand-section_subtle`, `text-tertiary_on-brand`, `border-utility-gray-200`, `bg-brand-solid`/`bg-error-solid` + hover/active, `shadow-xs-skeumorphic`, `disabled:bg-disabled`/`text-fg-disabled`/`ring-disabled_subtle`, `outline-brand`/`outline-error`, `rounded-xl`, `shadow-lg`, `min-h-11`.
- Pre-existing `dialog.test.tsx` + `sheet.test.tsx` green, unmodified assertions.
- Not run (per brief): build, lint, test:run, Playwright, dev servers. No commits made.

## Deviations / notes

1. **Tooltip radius `rounded-lg`, not the recipe's literal `rounded-xl`.** UU's real tooltip is rounded-lg; rounded-xl on a ~24px-tall surface reads pill-ish. Popover took `rounded-xl` as specified. Flagging in case the lead wants strict literalism.
2. **Header strips migrated off Tailwind-palette literals** (`emerald-950`, `emerald-200/75`, `emerald-800`, `emerald-200/50`) to bridge tokens per the no-hardcode rule. Visual delta: strip goes from emerald-tinted near-black to the W1 forest `#032214`; description text from green-tinted to neutral white/72% (UU on-brand). This is the intended identity migration, but it is a visible delta for the visual harness (expected in the W5 re-baseline).
3. **Overlay scrims untouched** (`bg-black/40|80`, `bg-neutral/60 backdrop-blur-[2px]`, `shadow-neutral/20`): `neutral` verified as a real theme token (`#20382d`), and scrims are not UU surfaces.
4. **`DialogNavyHeader`/`DialogContent` close button kept white-on-dark** — it overlays the brand strip; the only headerless `DialogContent` caller is `command.tsx` (W6 scope).
5. **Environment flake (not a code issue):** during verification the machine's load average hit ~50 (other projects' esbuild processes), which blew the default 5s per-test timeout on a pre-existing dialog test (it passed twice earlier at normal load, and the full suite passes 21/21 under `--testTimeout=30000`; no repo config changed). Anyone running wave gates concurrently may see the same spurious timeouts.

## Files touched

Modified: `frontend/src/components/ui/{dialog,alert-dialog,sheet,popover,tooltip}.tsx`, `frontend/src/components/ui/{dialog,sheet}.test.tsx`
Added: `frontend/src/components/ui/{alert-dialog,popover,tooltip}.test.tsx`

Status: DONE
Summary: All five overlay primitives restyled to the UU bridge vocabulary with the engine kept on Radix and every documented contract preserved; tsc green, 21/21 tests in the owned files, full Tailwind emission check clean.
Concerns: Visible deltas from header-strip token migration will need the planned visual re-baseline; tooltip radius deviates from the literal recipe (rationale above); machine load ~50 during the gate window can produce spurious 5s test timeouts.
