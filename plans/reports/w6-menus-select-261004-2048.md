# W6 — Menus + Select restyle (dropdown-menu, select)

Delivered 2026-10-04 ~21:20. Agent: w6-menus. Plan of record: `plans/261004-1812-uu-full-migration/plan.md` (engine-keep ruling: Radix stays; restyle-in-place).

## Files touched (exclusively)

- `frontend/src/components/ui/dropdown-menu.tsx` (modified)
- `frontend/src/components/ui/select.tsx` (modified)
- `frontend/src/components/ui/dropdown-menu.test.tsx` (new, 6 contract tests)
- `frontend/src/components/ui/select.test.tsx` (new, 5 contract tests)

## What changed

**Both files — popper/content surfaces:** `rounded-md border` → `rounded-lg ring-1 ring-utility-gray-200` (ring replaces border per the wave recipe; composes with the existing `shadow-sm`; `bg-popover` / `text-popover-foreground` kept). All `data-[state=*]` / `data-[side=*]` tailwindcss-animate hooks byte-identical. Separators: `bg-muted` → `bg-utility-gray-200` (matches landed W2 separator.tsx).

**Items (DropdownMenuItem, CheckboxItem, RadioItem, SubTrigger, SelectItem):** `focus:bg-accent focus:text-accent-foreground` → `focus:bg-utility-gray-100`. The `focus:text-accent-foreground` pairing is dropped — the new surface is near-white so foreground copy stays readable; this also lets call-site destructive classes (`text-destructive focus:text-destructive`) win cleanly. SubTrigger gained `transition-colors` (its sibling Item already had it). SubTrigger open state: `data-[state=open]:bg-utility-gray-100`.

**SelectTrigger:** restated to the landed W3 input vocabulary — `rounded-lg border-input bg-card shadow-xs`, legacy `focus:ring-2 ring-ring ring-offset-2`/`ring-offset-background` replaced by `outline-brand transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2`. Trailing chevron `opacity-50` → `text-fg-quaternary` (UU icon token). `h-11 / sm:h-9 / [&>span]:line-clamp-1` untouched.

**Untouched by design:** labels (`typography-body-medium`, inherited color — UserAvatarDropdown renders user info inside it; recoloring would leak into a file I don't own), shortcut slot, scroll buttons, all structural spans/indicator slots, every prop signature and export.

## Contracts verified preserved

- `--radix-select-trigger-width` / `--radix-select-trigger-height` viewport sizing (popper default `position="popper"`) — test-pinned.
- `onScroll` pass-through, scroll up/down buttons, SelectPrimitive.Icon chevron slot.
- `forceMount` / `sideOffset` / `align` / `avoidCollisions` forwarded via `{...props}`.
- Checkbox/radio `ItemIndicator` structure; `inset` prop; `data-[disabled]` hooks; caller class overrides win through tailwind-merge (test-pinned).
- Note: `src/components/shared/FilterPill.tsx` composes `@radix-ui/react-select` directly, so it does not consume `ui/select.tsx` (plan brief listed it as a consumer — it exercises the raw primitive only).

## Deliberate divergences / findings

1. **Hover vs focus collapse:** the recipe's gray-50 hover / gray-100 focus distinction collapses in Radix menus — pointer hover DOM-focuses items, so one surface (`focus:bg-utility-gray-100`) serves both. Documented in the file header comment.
2. **No in-file destructive item exists** in either component; destructive styling is call-site-driven (e.g. `UserAvatarDropdown` logout: `text-destructive focus:text-destructive`). Per file ownership + no speculative API, no `destructive` variant was added; the new focus surface pairs fine with call-site red text (#b42318 on #f2f4f7 ≈ 5.9:1). The recipe's destructive mapping is a no-op for W6 unless a later wave adds the variant.
3. **Radix forceMount behavior:** in the installed Radix version, Content with `forceMount` is NOT mounted while the root is closed (the prop is forwarded only). The test pins this actual behavior rather than the assumed one.
4. Two medium-size Writes landed corrupted (degraded-write hazard): `data-[side=left]:in-something-left-2`, `data-[side=directionless]`, `</SelectPrimitive.SelectItem>`, and a stray `focus-visible:outline-utility-gray-400`. All caught by full `git diff` review + small-Edit repair before verification. Final diff audited line-by-line; clean.

## Verification

- `pnpm exec tsc -p tsconfig.app.json --noEmit` → exit 0 (ran against the shared tree including concurrent W5/W7 in-flight edits).
- `pnpm exec vitest run` on my 2 new files + 6 consumer/control-density files (`control-density`, `searchable-select`, `searchable-dropdown`, `bank-selector`, `employee-multi-selector`, `EmployeeSingleSelector`) → **8 files / 31 tests passed** (11 new). Control-density's pinned `h-11 sm:h-9` trigger classes hold.
- Tailwind CLI emission check (content = my 2 files): `ring-utility-gray-200`, `focus:bg-utility-gray-100`, `bg-utility-gray-200`, `text-fg-quaternary`, `outline-brand`, `focus-visible:outline-2`, `shadow-xs`, `rounded-lg` — all emitted, no silent token fallback.
- Not run (per wave rules): build, lint, test:run, Playwright, dev servers, prettier, commit.

Status: DONE
Summary: dropdown-menu + select restyled in place to UU vocabulary (ring-utility-gray-200 popper surfaces, utility-gray-100 item focus, W3-input trigger vocabulary); all Radix contracts preserved and test-pinned; 31/31 tests, tsc green, emission check clean.
Concerns: none blocking. For the record: FilterPill does not consume ui/select; no in-file destructive item existed for the recipe's destructive mapping; three concurrent agents are mid-edit elsewhere in ui/ (tsc green includes their state as of ~21:16).
