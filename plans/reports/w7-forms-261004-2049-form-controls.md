# W7 — Form Controls Restyle (UU PRO)

Date: 2026-10-04. Agent: w7-forms. Plan of record: `plans/261004-1812-uu-full-migration/plan.md`.

## Scope delivered

Restyle-in-place of the seven W7 compat components. Radix engines untouched
(accordion stays Radix per the engine-keep ruling — UU ships no accordion).
100% contract preservation: forwardRef, displayName, aria wiring, prop/variant
unions, Radix data-state hooks, size floors, and caller `className` override
paths all unchanged.

| File | Change |
|---|---|
| `checkbox.tsx` | Rest state: white surface (`bg-card`) + `border-utility-gray-300` + `shadow-xs`. Checked: `bg-brand-solid` + `border-brand-solid`, check in white. Focus: UU `outline-brand` treatment replaces ring-offset ring. Kept `peer`, `h-4 w-4`, `rounded-sm`, disabled pair. |
| `radio-group.tsx` | Item mirrors checkbox: gray-300 border at rest; selected fills brand-solid with white dot (`fill-current` inherits the checked `text-white`). Root (`grid gap-2`) unchanged. |
| `switch.tsx` | On `bg-brand-solid`, off `bg-utility-gray-200`; thumb `bg-white shadow-xs` (was `shadow-sm`). Focus → `outline-brand`. Pill geometry, 44px hit-area pseudo-element, transitions kept. |
| `slider.tsx` | Track `bg-utility-gray-200`, range `bg-brand-solid`, thumb `border-2 border-brand-solid bg-brand-solid` (brand-solid per brief; `border-2` kept so thumb geometry is byte-identical). Focus → `outline-brand`. ARIA label pass-through untouched. |
| `progress.tsx` | Track `bg-utility-gray-100`, indicator `bg-brand-solid`. Height stays caller-overridable (`h-1`/`h-1.5` upload bars). |
| `tabs.tsx` | Existing pill idiom kept and recolored to UU neutrals: list `bg-utility-gray-100 rounded-lg text-fg-tertiary`; active `bg-card + text-fg-primary + shadow-xs`; inactive `text-fg-tertiary hover:bg-utility-gray-50`. Focus → `outline-brand`. `min-h-11` floors kept. |
| `accordion.tsx` | Divider `border-utility-gray-200`; trigger `font-semibold text-fg-primary`; chevron `text-fg-tertiary`. `animate-accordion-up/down` hooks untouched. |

## Contract decisions worth the reviewer's eye

1. **Border, not ring, on checkbox/radio.** The brief said "ring-utility-gray-300",
   but `BCCUploadModal.tsx:573` tints the checkbox via `border-amber-600` — a
   ring base would leave that caller's border-width at 0 and silently drop the
   amber tint. The gray border is expressed as `border-utility-gray-300` (1px,
   same visual weight as UU's inset ring); the merge test proves
   `border-amber-600` still replaces it.
2. **Switch focus.** The brief pinned the outline treatment for checkbox/radio;
   switch/slider/tabs/accordion had ring or no focus states. All W7 controls now
   share the landed `outline-brand focus-visible:outline-2` treatment for
   consistency (matches button/input from W3/W4).
3. **Tabs direction.** Of the two offered active treatments, kept the file's
   existing pill idiom recolored (option b) rather than switching to the
   brand-underline style — smallest structural delta for 13 importer surfaces;
   UU's own pill tabs are neutral the same way.
4. **Disabled states** kept as-is (`disabled:cursor-not-allowed disabled:opacity-50`)
   on checkbox/radio/switch/slider/tabs — the brief said keep disabled states;
   button-style explicit disabled tokens were not prescribed for these controls.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit` — full project, exit 0 (run
  twice; second run after final test-file edits, see Status note below).
- Vitest, my paths + the three consumer tests that exercise Switch/Checkbox
  class overrides end-to-end: **10 files / 44 tests passed**:
  - new contract tests: `checkbox.test.tsx`, `radio-group.test.tsx`,
    `switch.test.tsx`, `slider.test.tsx`, `progress.test.tsx`,
    `tabs.test.tsx` (render + state classes + forwardRef + className
    precedence, per the wave checklist), and a `Accordion UU contract (W7)`
    block appended to the existing `accordion.test.tsx` (keyboard-semantics
    test untouched and still green).
  - existing consumer tests still green: `BCCUploadModal.test.tsx`,
    `CheckInToggle.test.tsx`, `AdvanceRequestToggle.test.tsx`.
- Not run (per instructions): lint, build, test:run, Playwright, dev servers,
  prettier. No commits made. No kanban cards touched.

## Deviations / notes

- Tool-write corruption hit 5 of the new test files mid-flight (known degraded-
  write mode); each was caught by read-back/tsc/vitest and repaired. Final
  files are verified by the green run above.
- Uncommitted; ready for the wave gate and the combined W5–W7 commit sequence.
