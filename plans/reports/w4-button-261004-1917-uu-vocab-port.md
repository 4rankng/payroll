# W4 — Button compat-layer migration (255 importers)

2026-10-04 · Wave W4 of `plans/261004-1812-uu-full-migration/plan.md` · File owned: `frontend/src/components/ui/button.tsx` (+ new `button.test.tsx`)

## What changed

`ui/button.tsx` internals rewritten to the vendored UU v7 button vocabulary while
preserving the TypeScript contract 100%: same exports (`Button`, `buttonVariants`,
`ButtonProps`), same 11 variants × 5 sizes, `asChild` via `@radix-ui/react-slot`,
`forwardRef<HTMLButtonElement>`, zero call-site edits, zero removed props.

## Method and two structural decisions

1. **Native `<button>`/`Slot` element kept; vendored component NOT rendered internally.**
   The vendored UU Button is a react-aria-components control that force-defaults
   `type="button"` (breaks ~implicit-submit buttons in forms) and wraps children in a
   `data-text` span (DOM change ×255 call sites). Rendering it would break both.
   Importing just its `styles` export would evaluate the module and risk pulling
   react-aria-components into the entry chunk (main chunk is already 1.46 MB).
   → The UU **style vocabulary** is restated as the cva source (with a sync-note
   header pointing at `components/base/buttons/button.tsx` as source of truth).
2. **Neutral surfaces translated onto non-colliding bridge tokens.**
   Verified ground truth (dist CSS + config): `bg-primary`/`bg-primary_hover` are
   shadcn green (#08783e/#066632), so UU surface strings can't be reused verbatim.
   Mapping used: solids verbatim (`bg-brand-solid`, `bg-error-solid`, `success-solid`,
   `warning-solid`), neutral fills `utility-gray-*`, hover surface `bg-secondary_hover`
   (#e4e7ec = UU gray-200), text `text-tertiary`, disabled `bg-disabled`/`fg-disabled`/
   `ring-disabled_subtle` (UU explicit disabled tokens replace the old opacity-40).

## Variant → token map (all verified emitting in dist CSS)

| App variant | UU source | Result |
|---|---|---|
| default | primary | `bg-brand-solid` verbatim, before-gradient, skeumorphic shadow |
| destructive | primary-destructive | `bg-error-solid` + `outline-error` focus |
| secondary | — | `utility-gray-100/300/700` ladder (today's slate-100 look, UU tokens) |
| outline | secondary-gray | kept `border border-border bg-card` **border idiom** — many call sites override `border-*` (ApiKeysSection, dashboards, AdvancePaymentEmailDialog `border-dashed`) |
| ghost | tertiary | `text-tertiary hover:bg-secondary_hover hover:text-tertiary_hover` (fixed hover surface) |
| success/info/warning | app-specific | UU solid pattern on success/warning families; info stays `bg-sky-600` (no blue in bridge) |
| monochrome(-outline) | app-specific | `utility-gray-700` solid / border idiom |
| link | link-color | `text-brand-secondary` + underline idiom adapted (no data-text wrapper) |

Sizes: heights/paddings **verbatim** (compact scale is pinned by tests); base gains
UU radius (`rounded-lg`), weight (`font-semibold`), focus ring (`outline-brand`,
outline-offset-2), `h-max` (size heights override via tw-merge), transition.

## Verification (commands allowed by the task)

- `pnpm exec tsc -p tsconfig.app.json --noEmit` → exit 0 (255-importer gate)
- New `src/components/ui/button.test.tsx`: 8/8 — variant matrix signatures, native
  button + type/form/name/value passthrough, **no default `type` attr** (implicit
  submit preserved), forwardRef focus, native disabled + UU tokens, asChild/Slot
  merge order, call-site override (h-7 w-7 replaces h-11), `buttonVariants({variant,size,className})` callable
- Existing Button-pinned tests: 8 files / 44 tests green — control-density
  (`h-11 sm:h-9 text-sm`), SettingCard (`h-11`), CheckInSettingsPage, EmployeeAttendanceActionDock
  (`min-h-11 h-auto whitespace-normal`), ProjectMobileList, EmployeeMobileCard,
  TransactionPageHeader, EmployeePortalHeader

## Concerns / follow-ups (amended at acceptance, 2026-10-04)

1. **W1 bridge collision (affects future compat waves):** UU surface tokens
   (`bg-primary`, `bg-primary_hover`) resolve to this app's green. Future compat
   restatements must use the unambiguous families (`secondary_hover`, `utility-*`,
   `fg-*`, solid families), as W4 did. Correction at acceptance: the vendored banner
   is NOT affected — its secondary CTA already carries an explicit
   `bg-white text-foreground` override with a comment documenting this exact
   collision (banner-dual-action-brand-full-width.tsx:40-44).
2. **info variant**: no blue family in the bridge; kept Tailwind palette `bg-sky-600`
   (7 call sites) as interim — W7+ will decide between a UU-blue bridge family or
   keeping palette classes (noted in plan).
3. **Correction at acceptance:** `outline-error` resolves via the W1 token bridge —
   `tailwind.config.ts` outlineColor gained `error: '#b42318'` (working tree). The
   earlier daisyUI note came from checking the stale W0 baseline dist build; verify
   exact emission against the next dist build at commit time.
4. **Bundle**: RAC kept out of the entry chunk by restating the vocabulary; costs a
   sync obligation between base/buttons/button.tsx and ui/button.tsx (header comment
   records it).

## Scope re-pin (team-lead notice, post-CLI-overwrite incident)

- Style source re-read after the incident: `base/buttons/button.tsx` on disk is the
  cleaned W1 version and is byte-identical to what this port was derived from
  (no `in-data-input-wrapper:*`, sm/md single strings) — zero drift, no rework needed.
- `npx untitledui add` was never run in this wave (only tsc/vitest/grep/sed).
  Nothing outside `ui/button.tsx` + `ui/button.test.tsx` was touched here; other
  modified files in `git status` belong to W2/W3/visual-baseline/main.
- Gates re-run against the current tree at 19:48: tsc exit 0, button.test.tsx 8/8.

Status: DONE_WITH_CONCERNS
Summary: ui/button.tsx now renders the vendored UU v7 vocabulary on a native button/Slot element with 100% contract preservation (tsc gate green, 8 new contract tests + 44 existing pinned tests pass); UU surface tokens were translated to non-colliding bridge tokens because bg-primary/primary_hover are green shadcn roles here.
Concerns: W1 bridge collision (UU bg-primary = green) affects the vendored banner and every future compat wave; info variant lacks a UU blue ladder; outline-error depends on daisyUI color injection.
