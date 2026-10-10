# Phase 2 — Token-first palette fixes

## Context

Phase 1 produces a list of failing pairs. The fix policy (user-confirmed): prefer design tokens; darken the offending Tailwind step by one when no token fits.

## Requirements

- Every normal-text pair ≥4.5:1 against its actual surface (light and dark).
- Every non-text pair (borders, focus rings, info-carrying icons) ≥3:1.
- Preserve semantic hue (red = destructive, green = success, amber = warning).
- Preserve visual hierarchy — do not flatten all text to one color.

## Files

- `frontend/src/styles/base.css` — replace `#0284c7` (sky-600) with `var(--info)`.
- `frontend/src/styles/variables.css` — only if a token needs to shift (expected: none).
- Ad-hoc TSX files listed in `reports/contrast-audit.md` — swap `text-<family>-<step>` classes.

## Steps

1. Start with `base.css` (single known hex, deterministic fix).
2. For each failing TSX class:
   a. Try semantic token first (`text-muted-foreground`, `text-success`, `text-warning`, `text-destructive`, `text-info`).
   b. If no token fits, darken one step (500 → 600, 600 → 700, 400 → 500 → 600) until ≥4.5:1 on the actual surface.
   c. If the class is under `dark:`, verify it passes on the dark surface; leave it if so.
3. Re-run the scanner; the failure list must shrink to zero.

## Tests / validation

- `node frontend/scripts/contrast-audit.mjs` exits with zero failures.
- `pnpm test:run` passes (existing suite).
- `pnpm type-check && pnpm lint` pass.

## Risks

- Dense money tables can regress visually if a color is darkened too aggressively. Mitigation: screenshot before/after per affected page in Phase 5; revert individual files if the hierarchy is lost.
- Template-literal class names that the scanner cannot see are not fixed here. They surface in Phase 4 and are patched individually.
