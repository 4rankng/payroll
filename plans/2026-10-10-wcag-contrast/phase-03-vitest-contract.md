# Phase 3 — Extend the vitest contrast contract

## Context

`frontend/src/__tests__/wcag-contrast.test.ts:1` currently asserts 4.5:1 for 12 tokens against white only. Real components pair tokens with tinted surfaces (`--muted`, `--background`, `--employee-*-soft`), and ad-hoc Tailwind classes live outside the token layer entirely.

## Requirements

- Lock the floor into CI so regressions fail tests, not users.
- Cover every real surface a token is used against, not just white.
- Cover the Tailwind ad-hoc palette so a future `text-amber-400` reintroduction fails CI.
- Cover non-text contrast for borders and focus rings at 3:1.

## Files

- Extend: `frontend/src/__tests__/wcag-contrast.test.ts`
- Create: `frontend/src/__tests__/wcag-palette.test.ts` (palette-level gate)

## Steps

1. Add a token × surface matrix (every token × every surface in `variables.css`).
2. Add a palette gate that reads the failing classes discovered in Phase 1 and asserts they now resolve to ≥4.5:1 on their surface.
3. Add a non-text 3:1 gate for `--border`, `--input`, `--ring`, `--employee-focus-ring`, `--surface-border` against their backgrounds.
4. Add a `dark:` variant gate that resolves the same tokens under `.dark`.

## Tests / validation

- `pnpm test:run` passes.
- Temporarily reintroduce `text-amber-400` in a scratch file and confirm the palette test fails.

## Risks

- The palette test must not depend on the scanner being run first. It should recompute the palette contract independently from `tailwind-palette.mjs` and `variables.css`.
