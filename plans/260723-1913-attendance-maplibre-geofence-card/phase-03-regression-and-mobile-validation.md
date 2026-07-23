---
phase: 3
title: "Regression and Mobile Validation"
status: pending
effort: "M"
---

# Phase 3: Regression and Mobile Validation

## Overview

Lock down the migration with focused tests, build checks, and mobile visual validation. This phase verifies the new renderer without weakening existing geofence, check-in, or admin-map contracts.

## Related Code Files

- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.test.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/employee-location-map-model.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.test.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/utils/checkInGeofenceGuidance.test.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/admin-dashboard/LocationMap.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/package.json`
- `/Users/dev/Documents/projects/payroll/Makefile`
- `/Users/dev/Documents/projects/payroll/graphify-out/graph.json`

## Implementation Steps

1. Rewrite `EmployeeLocationMap.test.tsx` mocks from `react-leaflet` to `react-map-gl/maplibre`.
2. Assert the behavioral contract instead of MapLibre internals:
   - stacking context remains below the fixed attendance dock,
   - map uses the configured light style,
   - gate/geofence GeoJSON is produced,
   - employee marker and accuracy geometry render when `sample` exists,
   - nearby route appears with correct coordinates,
   - far outside route is suppressed at `distanceMeters > max(radiusMeters * 4, 500)` and outside status text remains visible,
   - route display boundary cases below, equal to, and above the cutoff,
   - viewport bounds include geofence polygon extents for single-gate and multi-gate targets,
   - fit-bounds is called once per target configuration/first usable route.
3. Add fallback tests for MapLibre runtime paths:
   - mocked `onError` event invokes fallback,
   - unsupported WebGL/context creation path invokes fallback,
   - fallback preserves status title, radius, nearest gate, and accuracy text.
4. Keep or add `checkInGeofenceGuidance.test.ts` coverage for no target, no position, inside, inaccurate, outside, boundary equality, and the fact that frontend guidance is advisory while server submission remains authoritative.
5. Keep `EmployeeCheckInCard.test.tsx` coverage for the lazy map disclosure. It should still mount `EmployeeLocationMap` only after the employee opens the map.
6. Run focused tests first, then broaden:
   - employee map component test,
   - check-in card test,
   - geofence guidance test,
   - frontend lint/type check,
   - frontend build.
7. After `pnpm build`, inspect Vite output to confirm MapLibre code stays in the lazy employee map chunk and does not move into the initial employee route bundle. Keep all MapLibre imports confined to the lazy map component and its lazy-only `.ts` model helper.
8. Run mobile visual QA through Playwright or the repo's existing preview process with the map disclosure open and the fixed attendance dock present. Capture at least:
   - 320px mobile,
   - common iPhone-width mobile,
   - desktop/card layout.
9. Include a 200% text or browser font-size stress case with the map disclosure open.
10. Run a real-browser network/attribution validation when the map disclosure opens:
   - enumerate style/tile hostnames,
   - verify no `api.mapbox.com`, Google Maps, or unapproved map hosts,
   - verify attribution is visible and focusable/readable as appropriate,
   - block or break the style URL and verify the Vietnamese fallback.
11. Make admin Leaflet verification mandatory. At minimum, render or smoke-check the admin `LocationMap`/attendance wrappers enough to prove the retained Leaflet dependency still works after the lockfile change.
12. Run `make api-test` after the frontend migration because the project requires it after feature changes, even though backend behavior should remain unchanged.
13. After code changes pass, run `graphify update .` from `/Users/dev/Documents/projects/payroll`.
14. Record any unavoidable residual risk in the implementation handoff, especially if live map tile/style loading cannot be exercised in CI.

## Success Criteria

- [ ] Focused Vitest coverage passes for employee map, check-in card disclosure, and geofence guidance.
- [ ] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm lint` passes.
- [ ] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm build` passes.
- [ ] Build output confirms MapLibre remains lazy-loaded behind the map disclosure.
- [ ] Real-browser validation confirms approved map hostnames, no Mapbox/Google requests, visible attribution, and fallback behavior for style/WebGL failure.
- [ ] Admin Leaflet smoke verification passes.
- [ ] `make api-test` passes or any pre-existing external dependency failure is documented with evidence.
- [ ] Mobile visual QA confirms no overlapping text, no wide-region viewport for nearby coordinates, no misleading far-distance route, readable fallback state, fixed-dock compatibility, and 200% text usability.
- [ ] `graphify update .` runs after code changes.

## Verification Commands

- `cd /Users/dev/Documents/projects/payroll/frontend && pnpm test:run src/components/employees/EmployeeLocationMap.test.tsx src/components/employees/EmployeeCheckInCard.test.tsx src/utils/checkInGeofenceGuidance.test.ts`
- `cd /Users/dev/Documents/projects/payroll/frontend && pnpm lint`
- `cd /Users/dev/Documents/projects/payroll/frontend && pnpm build`
- Build artifact inspection for MapLibre lazy chunk placement.
- Playwright/manual browser validation for map disclosure network hosts, attribution, blocked style URL, and mobile screenshots.
- Admin Leaflet smoke check through an existing admin map wrapper or route.
- `make api-test`
- `graphify update .`

## Risks and Rollback

- Risk: jsdom tests overfit mocked MapLibre props. Mitigation: pair component tests with Playwright visual checks.
- Risk: external basemap requests are flaky in CI. Mitigation: unit-test data and fallback behavior; use a live visual/manual check for the actual style when network is available.
- Risk: unrelated admin Leaflet behavior regresses through dependency changes. Mitigation: keep Leaflet deps and run a mandatory smoke check of admin attendance map routes or wrappers.
- Rollback: revert employee MapLibre files and lockfile additions; admin Leaflet files remain the baseline.
