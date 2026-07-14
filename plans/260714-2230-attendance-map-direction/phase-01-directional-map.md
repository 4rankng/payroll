# Phase 1 — Directional Map and Auto-open Lifecycle

Status: pending

## Context

- `EmployeeCheckInCard.tsx` owns `showLocationMap`; backend GPS/geofence failures already open and scroll to the disclosure.
- The new boundary guidance classifies `124 m / 150 m / ±48 m` as actionable `inaccurate` guidance and names the nearest gate.
- `EmployeeLocationMap.tsx` already renders a dashed line from the GPS sample to that gate, but has no arrowhead, labels it as distance only, and currently calls `fitBounds` again as reactive sample objects change.
- The worktree contains related uncommitted edits, especially in `EmployeeCheckInCard`, `useContinuousLocation`, and `checkInGeofenceGuidance`; implementation must extend rather than overwrite them.

## Files

- Modify: `frontend/src/components/employees/EmployeeCheckInCard.tsx`
- Modify: `frontend/src/components/employees/EmployeeCheckInCard.test.tsx`
- Modify: `frontend/src/components/employees/EmployeeLocationMap.tsx`
- Modify: `frontend/src/components/employees/EmployeeLocationMap.test.tsx`
- Modify only if a dedicated map class/reduced-motion override is needed: `frontend/src/styles/base.css`
- Verify, without changing contracts: `frontend/src/utils/checkInGeofenceGuidance.ts`, `frontend/src/utils/checkInGeofenceGuidance.test.ts`, and `frontend/src/hooks/useContinuousLocation.ts`

## Implementation

1. Derive an explicit `isActionableBoundaryGuidance` condition from the existing inside-but-uncertain guidance, rather than treating all `inaccurate` or `outside` states alike.
2. Add an edge-triggered effect in `EmployeeCheckInCard`: on `false → true`, open the map once; do not reopen after a manual close until the condition first becomes false and later true again. Keep the existing error-driven open-and-scroll path unchanged and avoid automatic scrolling for passive GPS transitions.
3. Keep the disclosure button accessible (`aria-expanded`/`aria-controls`) and preserve manual open/close in every other attendance state.
4. In `EmployeeLocationMap`, keep the route a straight sky-blue dashed line and add a small, fixed-size directional arrowhead whose tip targets the computed nearest gate. Use existing Leaflet/SVG primitives and a local pure geometry helper; do not add a routing or arrow plugin. Give route and arrow stable semantic class names so their direction/target can be asserted.
5. Replace the current bottom distance chip with a restrained two-line overlay: primary `Hướng tới <gate> · <distance>` with a direction icon, secondary `Đường thẳng tham khảo — không phải lộ trình đường bộ`. Keep the project chip, GPS badge, emerald gate/circle, sufficient contrast, and non-interactive overlays (`pointer-events-none`).
6. Change viewport fitting to initialize the gate view once and, if the map initially has no location, fit the first available user-to-nearest-gate route once. Do not refit for subsequent GPS sample updates, so user pan/zoom remains intact; reset only when the component remounts or the target project changes.
7. Reuse the existing short route-reveal treatment only as a one-shot entrance. Ensure route/arrow styles have no infinite animation and are covered by `prefers-reduced-motion: reduce`.

## Tests

Add or extend focused tests for:

- Exact `124 m / 150 m / GPS ±48 m` state: inward guidance appears and the map auto-opens.
- Auto-open lifecycle: manual close is respected while guidance remains active; a later exit and re-entry can auto-open again.
- Multiple gates: overlay, route endpoint, arrow tip, and highlighted checkpoint all use the nearest gate.
- Arrow semantics: the dashed route and fixed-size arrowhead are present, point at the nearest gate, and the visible copy states that this is a straight-line reference rather than a road route.
- Poor GPS (for example ±800 m) retains recovery guidance rather than inward-direction auto-open.
- Outside-geofence behavior remains outside guidance and retains its existing error/manual map behavior.
- `FitLocationBounds` does not run again for ordinary sample-coordinate updates after its initial route fit.
- Reduced-motion CSS disables any one-shot route/arrow entrance motion.

Run:

```bash
cd frontend
pnpm test:run -- src/components/employees/EmployeeCheckInCard.test.tsx src/components/employees/EmployeeLocationMap.test.tsx src/utils/checkInGeofenceGuidance.test.ts
pnpm test:run
pnpm lint
pnpm type-check
pnpm build
cd ..
make api-test
graphify update .
```

If the live backend required by `make api-test` is unavailable, record that environmental limitation explicitly; frontend gates remain mandatory.

## Visual Review

- Inspect the actionable state on a narrow employee viewport and desktop width: overlay does not cover the user/gate endpoints, Vietnamese copy wraps cleanly, and map controls remain draggable/zoomable.
- Confirm the arrow is legible over both light and dark map tiles, terminates at the emerald nearest gate, and does not resemble turn-by-turn navigation.
- Confirm closing/reopening feels stable with no scroll jump or viewport snap during subsequent GPS updates.

## Risks and Mitigations

- **State loop reopens a map the worker closed:** edge-trigger on guidance entry and retain the previous-state ref independently of `showLocationMap`.
- **Live GPS updates fight map gestures:** fit only on initial mount/first usable route, not on every sample object.
- **Arrow scales poorly at very short or long distances:** use fixed visual size and point it at the gate, rather than scaling it as a percentage of route length.
- **Overlay obscures map content:** keep it compact, anchored with safe inset, and include its footprint in visual review.
- **Dirty-worktree regression:** inspect the diff before and after each edit and preserve all existing `useContinuousLocation` and boundary-guidance changes.

## Rollback

Revert only the card's edge-triggered auto-open state/effect, the map arrow/overlay/one-time fitting changes, their focused tests, and any dedicated CSS classes. The prior manual disclosure, dashed line, GPS guidance, backend contracts, and existing uncommitted boundary fix remain intact.

Status: DONE
Summary: Implementation plan prepared for a clean, nearest-gate directional map with controlled auto-open lifecycle and stable map interaction.
Concerns/Blockers: None; `make api-test` requires a running local backend during implementation verification.
