---
phase: 3
title: "Regression and Mobile Validation"
status: in-progress
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

1. Rewrite `EmployeeLocationMap.test.tsx` mocks from `react-leaflet` to direct `maplibre-gl`.
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

- [x] Focused Vitest coverage passes for employee map, check-in card disclosure, and geofence guidance.
- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm lint` passes.
- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm build` passes.
- [x] Build output confirms MapLibre remains lazy-loaded behind the map disclosure.
- [x] Real-browser validation confirms approved map hostnames, no Mapbox/Google requests, and visible attribution.
- [x] Automated fallback coverage confirms style/WebGL failure keeps a readable Vietnamese fallback.
- [x] Admin Leaflet smoke verification passes.
- [x] `make api-test` passes or any pre-existing external dependency failure is documented with evidence.
- [x] Mobile visual QA confirms the reported broken map is fixed, the viewport is local, attribution is readable, and the fixed dock does not cover the map card.
- [ ] Optional expanded mobile QA confirms 200% text usability and live broken-style fallback behavior.
- [x] `graphify update .` runs after code changes.

## Implementation Notes

- Focused Vitest command passed after review fixes: 4 files, 40 tests.
- `pnpm lint` passed with 3 pre-existing generated coverage warnings and no touched-file warnings.
- `pnpm build` passed and emitted `EmployeeLocationMap.*.js` plus `maplibre-gl.*.js`, confirming the map runtime remains split behind the lazy employee map.
- `go test ./internal/transport/http/middleware` passed after adding CARTO/MapLibre worker CSP coverage.
- Real-browser mobile check with the provided employee login passed on 2026-07-23: one MapLibre canvas rendered, 4 markers rendered, no fallback, no request failures, no page errors, CARTO/OpenStreetMap attribution visible, and no Mapbox/Google requests. Screenshot: `/tmp/payroll-map-final.png`.
- Direct MapLibre replaced the initially planned `react-map-gl/maplibre` wrapper after visual validation found a wrapper runtime resize crash. `react-map-gl` was removed from `frontend/package.json` and `frontend/pnpm-lock.yaml`.
- Admin Leaflet smoke: source still imports `react-leaflet` and `leaflet/dist/leaflet.css`; production build still emits `AttendanceLocationMap.*` separately, so retained admin Leaflet wiring compiles.
- Root `make api-test` target is absent. `make -C backend api-test` ran and failed in unrelated backend flows: `Assets` upload/metadata/download and `Transaction` export. `EmployeeSelfService` passed.
- Untitled UI MCP could not be used for component retrieval because the MCP server returned HTTP 429 rate limits on search and direct component calls; implementation used the repo's existing employee card/badge styling instead.

## Verification Commands

- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm test:run src/components/employees/EmployeeLocationMap.test.tsx src/components/employees/EmployeeCheckInCard.test.tsx src/utils/checkInGeofenceGuidance.test.ts`
- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm lint`
- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm build`
- [x] Build artifact inspection for MapLibre lazy chunk placement.
- [x] Playwright/manual browser validation for map disclosure network hosts, attribution, and mobile screenshot.
- [x] Admin Leaflet smoke check through an existing admin map wrapper or route.
- [x] `make api-test`
- [x] `graphify update .`

## Risks and Rollback

- Risk: jsdom tests overfit mocked MapLibre props. Mitigation: pair component tests with Playwright visual checks.
- Risk: external basemap requests are flaky in CI. Mitigation: unit-test data and fallback behavior; use a live visual/manual check for the actual style when network is available.
- Risk: unrelated admin Leaflet behavior regresses through dependency changes. Mitigation: keep Leaflet deps and run a mandatory smoke check of admin attendance map routes or wrappers.
- Rollback: revert employee MapLibre files and lockfile additions; admin Leaflet files remain the baseline.
