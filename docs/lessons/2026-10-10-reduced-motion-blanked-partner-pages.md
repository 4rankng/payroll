# Reduced-motion users saw blank partner pages (2026-10-10)

**Date:** 2026-10-10
**Source:** Partner report (Lê Thị Hòa / `hoalt`) — "cannot see the employee list", `tingting.vip/partner/timesheet` blank
**Tags:** [frontend, accessibility, css, partner, pwa]

## Context

A partner on Windows reported that `/partner/timesheet` and `/partner/employees`
rendered an empty content area: the sidebar, nav and version footer painted, but
everything inside `#main-content` was invisible. Same account worked on another
machine; `Ctrl+Shift+R` did not help. Prod nginx logs showed **zero 4xx/5xx** —
`index.html`, `/sw.js` and every hashed chunk returned 200 — and the service
worker had precached the current build, so it was not a stale-chunk/deploy
problem. The DOM was fully rendered (`#main-content` text was present) but
painted at `opacity: 0`.

## Root cause

Both pages gated every content block on an entrance animation:

```
opacity-0 motion-safe:animate-fade-in-up [animation-delay:Nms] [animation-fill-mode:forwards]
```

`motion-safe:` compiles to `@media (prefers-reduced-motion: no-preference)`. On a
machine that reports `prefers-reduced-motion: reduce` (Windows *Settings →
Accessibility → Visual effects → Animation effects* off, a power/IT policy, or a
reduced-motion extension) the animation never applies, so the `opacity-0` base
style is never overridden and the content stays permanently invisible. 9 blocks
across 2 files (`partner/EmployeesPage` ×4, `partner/TimesheetsPage` ×5).

Verified by emulating the media feature in Chrome against the local dev server
with a real partner session:

| `prefers-reduced-motion` | computed opacity | animation-name | `checkVisibility()` |
|---|---|---|---|
| `no-preference` (before) | 1 | `fade-in-up` | true |
| `reduce` (before) | **0** | **none** | **false** |
| `no-preference` (after) | 1 | `fade-in-up` | true |
| `reduce` (after) | 1 | none | true |

## Decision / Outcome

Removed the `opacity-0` base and switched the fill mode to `both`, so the 0%
keyframe (which is what hides the element during the stagger delay) is applied
*by the animation itself*:

```
motion-safe:animate-fade-in-up [animation-delay:Nms] [animation-fill-mode:both]
```

This matches the pattern the rest of the app already uses
(`partner/DashboardPage`, `partner/ProjectsPage`, `Login`, `adv-partner/UsersPage`
all animate without an `opacity-0` base) and keeps the staggered reveal
identical for users who allow motion.

## Lesson

**Never make a base style depend on an animation running to restore
visibility.** `opacity-0` / `invisible` + `motion-safe:animate-*` is a trap: the
variant silently inverts under reduced motion, and the failure mode is a blank
page with a clean network tab and no console error — the hardest kind of report
to reproduce on a dev machine that has animations on. If an element must start
hidden, let the keyframe hide it (`animation-fill-mode: both`) instead of the
base utility.

**Diagnosis pattern that worked:** "works on my machine / fails on hers" +
zero 4xx in the nginx log + DOM present ⇒ stop looking at deploy/cache and check
paint-time CSS (media queries, `prefers-*`, animations). Emulating
`prefers-reduced-motion` in DevTools takes one line.

## References

- `frontend/src/pages/partner/EmployeesPage/index.tsx`
- `frontend/src/pages/partner/TimesheetsPage/index.tsx`
- `frontend/tailwind.config.ts` (`fade-in-up` keyframes / `animation` map)
