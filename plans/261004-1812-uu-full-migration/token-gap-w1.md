# W1 Token Gap — classes referenced by vendored UU files with no config definition

Derived 2026-10-04 by grepping `src/components/{base,foundations,marketing}` and diffing against `tailwind.config.ts`. Any class below that stays undefined = silent fallback (Tailwind emits no CSS).

## Missing color families (add to `theme.extend.colors`)

- **error family:** `error-solid`, `error-solid_hover`, `error-primary`, `error-primary_hover`, `error-secondary(_hover)`, `error_subtle`, `fg-error-primary`, `fg-error-secondary` (button destructive variants are silently unstyled today). Map to UU v7 error scale; `#b42318`-based (employee theme already uses UU's error-700).
- **warning family:** `warning-solid`, plus `fg-warning-primary`; success family: `success-solid`, `fg-success-primary`. Same ladder pattern.
- **secondary-solid** (used by vendored badge/toggle styles), `secondary_alt` (ring).
- **tertiary:** `tertiary`, `tertiary_hover` (flat, beside existing `tertiary_on-brand`).
- **fg extensions:** `fg-secondary`, `fg-tertiary`, `fg-brand-primary`, `fg-brand-secondary`, `fg-brand-secondary_hover`, `fg-brand-secondary_alt`.
- **utility ladders:** `utility-{gray,brand,error,warning,success}-{50..600}` (only `utility.brand-200` exists today; FeaturedIcon light themes render without borders/fills).
- **featured-icon-light-fg-{brand,gray,error,warning,success}.**
- **button-destructive-primary-icon**, `button-destructive-primary-icon_hover`.

## Missing type scale (add to `theme.extend.fontSize`)

- `display-xs..2xl` mapped to the app's current heading rendering (cx.ts/tailwind-merge already registers `display-*`; values must be calibrated against W0 baselines — never UU's 48–60px).

## Already present (verified in config)

brand.solid/solid_hover/section_subtle/secondary/secondary_hover, brand_alt, primary_on-brand, tertiary_on-brand, primary_hover, secondary_hover, fg.white/quaternary/quaternary_hover/disabled/disabled_subtle, disabled, disabled_subtle, button-primary-icon(_hover), utility.brand-200, outlineColor focus-ring/brand, shadow-xs(_skeumorphic), rounded-xs, half-step spacing, opacity-12, max-w-container, text-md, `transition-inherit-all` plugin utility.

## Dead classes to remove in W1

- All `in-data-input-wrapper:*` / `in-data-leading:` / `in-data-trailing:` variants in `src/components/base/buttons/button.tsx` (TW4.1 `in-*` variant; 0 hits in built CSS). Re-port as `[[data-input-wrapper]_&]:` only if/when input-group is vendored.

## W1 gate

After implementation: `pnpm build`, then grep `dist/assets/*.css` for every class listed above (as escaped selectors) — zero missing; visual diff vs W0 baseline = zero delta.
