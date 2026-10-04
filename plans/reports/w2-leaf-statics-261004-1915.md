# W2 — Leaf Statics (badge, avatar, card, separator, skeleton, label)

Wave W2 of the UU PRO migration. Executed 2026-10-04 against `main` @ `d9ba00a9` (+ uncommitted W1/W3/W4 work in tree).

## What changed

### Vendored via UU CLI (v7, PRO license from MCP `get_component`)

- `frontend/src/components/base/badges/` (`badges.tsx`, `badge-types.ts`) — 2 files
- `frontend/src/components/base/avatar/` (`avatar.tsx`, `avatar-label-group.tsx`, `avatar-profile-photo.tsx`, `utils.ts`, `base-components/`) — 8 files
- `frontend/src/components/foundations/dot-icon.tsx` — transitive dependency of vendored badges (hard import)

### Vendored-file adaptations (TW4→TW3 ports + token-collision ports, comment-documented)

- badges.tsx: `size-max`→`w-max`; `pl-0.75`/`pr-0.75`→`[3px]`; `p-1.25`→`[5px]`; `stroke-3`→`stroke-[3]`; UU `bg-primary`/`text-secondary`/`ring-primary` (modern gray) → `bg-white`/`text-fg-secondary`/`ring-utility-gray-200` (shadcn name collision, banner precedent); `text-gray-500`→`text-utility-gray-500`.
- avatar.tsx: `bg-avatar-bg`→`bg-utility-gray-100`; `avatar-contrast-border`→`outline-utility-gray-200`; `group-outline-focus-ring` → TW3 `group-focus-visible:outline …outline-focus-ring` (incl. `outline` style class); `text-quaternary`→`text-fg-quaternary`.
- avatar-profile-photo.tsx: `size-18`→`size-[4.5rem]`; `p-0.75/1.25/1.75`→`[3px]/[5px]/[7px]`; `bg-primary`→`bg-white`; `bg-tertiary`→`bg-utility-gray-100` (featured-icon precedent); `text-quaternary`→`text-fg-quaternary`; `outline` style class added so contrast rings paint.
- base-components: `ring-bg-primary`→`ring-white` (light-only app); `bg-fg-success-secondary`→`bg-success-secondary` (no fg.success.secondary in bridge); VerifiedTick `text-utility-blue-500`→`text-focus-ring` (same #2e90fa value, no blue ladder in bridge); fixed upstream typo `size-[4.38px` missing `]`.
- All documented with adaptation comments in-file, matching banner/featured-icon style.

### ui/ compat rewrites (contracts preserved 100%)

| File | UU restyle | Contract kept |
|---|---|---|
| `ui/badge.tsx` | Soft UU chips on utility ladders (50/700/200 per family) for default/secondary/destructive/success/warning/outline; role variants kept as cva solid entries; `role` → UU gray (`bg-utility-gray-600`, same hex as old slate-600); `partner` → `bg-brand-solid` | Badge, badgeVariants (cva), BadgeProps, forwardRef, displayName, all 11 variant names — no call-site edits (119 importers) |
| `ui/avatar.tsx` | utility-gray surface + UU inner contrast outline; `object-cover` added to AvatarImage | Avatar/AvatarImage/AvatarFallback compound + Radix refs/displayNames exact |
| `ui/card.tsx` | `border-utility-gray-200/300`, `shadow-xs`, filled → `bg-utility-gray-50 text-fg-secondary`; CardTitle/Description → UU fg tokens at identical px (title 14px/600 = old typography-title-large; desc 12px = typography-body-medium) | Card + 5 compound exports, `variant` union default/elevated/outlined/filled unchanged |
| `ui/separator.tsx` | `bg-utility-gray-200` | Separator, Radix shape exact |
| `ui/skeleton.tsx` | `bg-utility-... ` utility-gray-100 | Skeleton function shape unchanged (no forwardRef, as before) |
| `ui/label.tsx` | 11px/500/leading-none preserved via compact scale; `text-fg-primary`; tracking-wide kept | Label, Radix label, forwardRef, displayName |

No hex hardcoded anywhere; every class resolves in the W1 token bridge.

## CLI incident (resolved, nothing installed)

`npx untitledui add` failed under the repo's pnpm `overrides` (npm EOVERRIDE at npx bootstrap). Worked around by global install (`npm i -g untitledui@0.1.69`) + run from `frontend/`. The CLI then (a) added ~6 extra component families and (b) edited `package.json` (+`@react-stately/utils`, bumped `tailwind-merge` 2→3.7.0, `@untitledui/icons`, `input-otp`) and `pnpm-lock.yaml`, and mutated `node_modules` before its own npm step failed. **All reverted**: `git checkout -- package.json pnpm-lock.yaml` + `pnpm install --frozen-lockfile` (back to tailwind-merge 2.6.1, icons 0.0.22, input-otp 1.4.2, no @react-stately/utils). Deleted the extra CLI byproduct families (`base/input`, `base/textarea`, `base/tooltip`, future waves re-vendor fresh), trimmed `avatar-add-button.tsx` + its index entry (its tooltip/input chain needed the removed deps). `base/tags` also removed. Kept `dot-icon.tsx` (hard dep of badges). W3/W5 re-vendor cleanly; `untitledui add` overwrite warnings expected.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit` — **exit 0** (whole project, incl. parallel W3/W4 edits + my test files).
- 6 new contract test files (dialog.test.tsx pattern), **19/19 pass**, run twice (mid + final shared tree):
  `badge/card/avatar/separator/skeleton/label.test.tsx` — every variant renders its signature class, call-site className wins merge (twMerge conflict dropped), forwardRef passes through.
- **CSS emission check** (TW3 CLI compile of my files + vendored dirs): every UU token class emits, incl. `bg-utility-brand-50`, `outline-utility-gray-200`, `text-focus-ring`, `stroke-[3]`, `w-max`, hover variants. No silent token fallbacks.
- WCAG check on new pairs: all soft chips (utility-{family}-700 on -50) ≥ ~5:1; `role`/`partner` solids ≥ 5.4:1; `outline` variant fg-primary on transparent = on-card ratio. `info` (sky-600, 4.1:1) and `manager` (teal-600, 3.8:1) — pre-existing legacy palette, unchanged by me, flagged below.

## Concerns / owner decisions

1. **Bridge gap: no utility-blue ladder** — `info` and `admin` badge variants keep the legacy tailwind-default palette (sky-600/blue-600). In UU terms they'd want `utility-blue-*` (and `admin` is a role color, arguably fine legacy). Flag for token owner (W1) to add ladders or accept legacy.
2. **Pre-existing WCAG gaps in legacy role variants** (`info` 4.1:1, `manager` 3.8:1) — unchanged by this wave (would only worsen the migration diff if "fixed" silently); needs owner call vs the 4.5:1 release gate.
3. **Removed vendored families**: base/input, base/textarea, base/tags, base/tooltip were CLI byproducts; if W3/W5 prefer them over re-vendoring, they exist in git history? No — untracked, now deleted. Re-vendor cost is one CLI command each.
4. Extra vendored file `avatar-add-button` removed; if the avatar family is extended later, re-vendor will restore it (needs base/input + base/tooltip + `@react-stately/utils` then).
5. Visual delta: badge soft chips (50/700/200) replace solid badges for the six status variants — intended UU look; visual-baseline teammate will see deltas on any page rendering badges (dashboards, tables).

## Not done (out of scope)

- No call-site edits; no git commit; no build/lint:full/test:run full suites (per wave constraints); no tailwind.config.ts edits (not my file).
