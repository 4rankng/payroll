# Timesheet dropdown visual regression

## Context

The July 4 filter refresh pushed `FilterPill` to a 44px touch target, but the timesheet page never got the same treatment for the project and employee dropdowns. That left the desktop row visually split between a larger shared pill and older 28px triggers, and the mismatch was obvious because `bg-background` blended into the page surface at `#f5f7f9`.

## What happened

The fix aligned the desktop triggers to the same 44px surface and switched them to `bg-card` so they read as deliberate controls instead of floating page chrome. In `TimesheetFilters.tsx`, month and status now use the card surface too, while project and employee get the taller `h-11 min-h-11` treatment with consistent padding and icon sizing.

The regression test was the useful part: it failed before the change because the triggers did not share the same 44px card-like surface, then passed after the class updates landed. The suite finished cleanly with 129 tests passing, plus `lint`, `typecheck`, and `build`.

## Reflection

This was a boring bug in the worst way: not broken logic, just UI drift that made the page feel unfinished. The real mistake was letting one shared control evolve in `FilterPill.tsx` while adjacent timesheet filters stayed on the old sizing and background tokens. That kind of drift is exactly how visual regressions survive code review and turn into user-visible inconsistency.

## Decisions

I chose to standardize the timesheet desktop dropdowns on the shared 44px card surface instead of leaving project and employee at 28px for density. That trades a little compactness for consistency and touchability, which is the right call on this screen. I also kept the fix in component props and class names rather than adding a new styling abstraction, because this was a one-off regression, not a design-system rewrite.

## Next

Keep the regression test in place so this does not silently drift again. If another filter row is introduced, it should inherit the same 44px `bg-card` contract by default instead of rediscovering the old mismatch later.
