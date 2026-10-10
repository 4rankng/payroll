# Phase 1 — Static contrast scanner

## Context

- Token layer (`frontend/src/styles/variables.css`) is already gated by `frontend/src/__tests__/wcag-contrast.test.ts` and is healthy.
- Ad-hoc Tailwind classes (846 instances across 190 files) bypass the token layer and are the primary source of failures.
- Dark mode is present (107 `dark:` rules); each pair must be evaluated against the correct surface set.

## Requirements

- Build a Node script that walks `frontend/src/**/*.{tsx,ts,css}` and extracts:
  - `text-<color>-<step>` classes (including `dark:` variants)
  - `bg-<color>-<step>` and `bg-<color>-<step>/<alpha>` classes
  - inline `color:` and `background:`/`background-color:` hex/rgb values
- Resolve every color to RGB using:
  - Tailwind v3 default palette (mirrored in `tailwind-palette.mjs`)
  - design tokens from `variables.css` (parsed at runtime)
- Compute WCAG 2.x contrast ratio and flag:
  - text < 4.5:1
  - non-text (borders, focus rings, icons carrying information) < 3:1
- Emit a JSON report plus a Markdown summary with `file:line`, current ratio, required ratio, suggested fix.

## Files

- Create: `frontend/scripts/tailwind-palette.mjs`
- Create: `frontend/scripts/contrast-audit.mjs`
- Create: `plans/2026-10-10-wcag-contrast/reports/contrast-audit.md`

## Steps

1. Encode the Tailwind v3 default palette (families × steps 300–700) used in the codebase.
2. Parse `variables.css` into a token → RGB map (HSL triplets and hex both supported).
3. Walk source files with a light regex extractor (do not parse JSX; only class strings and inline styles).
4. Evaluate each pair, categorise by severity and by theme (light / dark).
5. Write JSON + Markdown reports under `plans/2026-10-10-wcag-contrast/reports/`.

## Tests / validation

- `node frontend/scripts/contrast-audit.mjs` exits 0 with a non-empty failure list.
- The Markdown report is ordered worst-first and groups by file.
- Sanity spot-check: the known failing pair `base.css:888` `#0284c7` on white must appear in the report (4.10:1).

## Risks

- Regex extraction is approximate. Class strings embedded in template literals are caught; computed class names at runtime are not (these will be caught in Phase 4 by axe).
- The Tailwind palette is mirrored by hand. If the project is on a custom palette, the audit would under-report. Mitigation: cross-check against `frontend/tailwind.config.ts`.
