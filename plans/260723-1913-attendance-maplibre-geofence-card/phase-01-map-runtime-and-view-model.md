---
phase: 1
title: "Map Runtime and View Model"
status: completed
effort: "M"
---

# Phase 1: Map Runtime and View Model

## Overview

Add the MapLibre runtime beside the existing Leaflet stack and extract the map-specific geometry/view decisions needed by the employee card. This phase should not change backend contracts or admin maps.

## Related Code Files

- `/Users/dev/Documents/projects/payroll/frontend/package.json`
- `/Users/dev/Documents/projects/payroll/frontend/pnpm-lock.yaml`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/employee-location-map-model.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.test.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/utils/checkInGeofenceGuidance.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/admin-dashboard/LocationMap.tsx`
- `/Users/dev/Documents/projects/payroll/backend/internal/transport/http/middleware/security_headers.go`
- `/Users/dev/Documents/projects/payroll/backend/internal/transport/http/middleware/security_headers_test.go`

## Implementation Steps

1. Re-verify package metadata immediately before implementation. Target versions verified during planning were `react-map-gl@8.1.1` and `maplibre-gl@6.0.0`; if newer versions are selected, record the reason in the implementation handoff and review the lockfile diff.
2. From `/Users/dev/Documents/projects/payroll/frontend`, add the runtime dependencies with `pnpm add react-map-gl@8.1.1 maplibre-gl@6.0.0` unless the re-verification step intentionally selects newer compatible versions.
3. Confirm `leaflet`, `react-leaflet`, and `@types/leaflet` remain installed because admin attendance maps still use `/Users/dev/Documents/projects/payroll/frontend/src/components/admin-dashboard/LocationMap.tsx`.
4. Create `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/employee-location-map-model.ts` for pure geometry and view-model logic. Keep `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.tsx` focused on UI rendering, matching the frontend rule that `.tsx` files are UI-only and `.ts` files hold data transformation/utilities.
5. Add these model helpers:
   - `buildGeofencePolygon(gate, radiusMeters)`.
   - `buildGeofenceFeatureCollection(target)`.
   - `buildAccuracyFeature(sample)`.
   - `buildRouteFeature(sample, nearestGate, shouldShowRoute)`.
   - `getEmployeeMapViewport(guidance, target, sample)`.
   - `shouldShowEmployeeRouteForDisplay(guidance)`.
6. Use existing `distanceMetersBetween()` from `/Users/dev/Documents/projects/payroll/frontend/src/utils/checkInGeofenceGuidance.ts` for distance decisions where possible. Do not create a second geofence classifier.
7. Define view modes from existing advisory guidance data:
   - `no_position`: fit gate/geofence area.
   - `inside` or `inaccurate`: fit employee plus nearest gate when both are local and useful.
   - `outside`: show a route only when `distanceMeters <= max(radiusMeters * 4, 500)`; otherwise fit the gate/geofence and surface distance in card UI.
   - `no_target`: no map runtime needed beyond fallback/status.
8. Keep the far-distance display cutoff UI-only and documented in code/tests. It must not alter guidance status, check-in eligibility, or server submission behavior.
9. Compute map bounds from geofence polygon bounding boxes, not only gate centers. Single-gate and multi-gate views must include north/south/east/west radius extents.
10. Select the initial style provider in one narrow constant only after verifying privacy and attribution:
   - allowed style/tile hostnames,
   - required attribution display,
   - no `api.mapbox.com`, Google Maps, or token-bearing map requests,
   - CSP `connect-src`/`img-src` implications,
   - self-host/proxy/no-tile fallback if third-party worksite tile disclosure is not acceptable.
11. Add or adjust unit coverage for helper outputs, especially polygon closed rings, finite coordinates, invalid radius/coordinates, route suppression below/equal/above threshold, and radius-inclusive viewport bounds.

## Success Criteria

- [x] `frontend/package.json` and `frontend/pnpm-lock.yaml` include `react-map-gl` and `maplibre-gl`.
- [x] Leaflet dependencies are still present.
- [x] Helper logic is deterministic, unit-testable, and does not depend on browser WebGL.
- [x] Geofence status remains advisory and delegated to `getCheckInGeofenceGuidance()`; backend submission behavior remains authoritative.
- [x] Far-distance route suppression is display-only and cannot affect guidance status, backend check-in authorization, or submission attempts.
- [x] Viewport bounds include geofence radius extents for single-gate and multi-gate cases.
- [x] Provider/privacy decision is documented before a live style URL ships.

## Implementation Notes

- Added `react-map-gl@8.1.1` and `maplibre-gl@6.0.0` with `pnpm`; Leaflet dependencies remain for admin maps.
- Added `employee-location-map-model.ts` with geofence polygon, accuracy polygon, route feature, display cutoff, and viewport helpers.
- Selected CARTO Positron style URL in one constant: `https://basemaps.cartocdn.com/gl/positron-gl-style/style.json`; allowed CARTO hostnames are captured beside the style constant for browser validation.
- Updated production CSP to allow CARTO style/tile fetches and MapLibre worker blobs; middleware tests cover the changed header.

## Verification

- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm test:run src/components/employees/EmployeeLocationMap.test.tsx src/utils/checkInGeofenceGuidance.test.ts`
- [x] `cd /Users/dev/Documents/projects/payroll/frontend && pnpm lint`
- [x] `cd /Users/dev/Documents/projects/payroll/backend && go test ./internal/transport/http/middleware`

## Risks and Rollback

- Risk: new helper code duplicates geofence rules. Mitigation: only reuse existing guidance and distance helpers; tests should assert status behavior remains unchanged.
- Risk: third-party map providers receive sensitive worksite tile requests. Mitigation: make provider/host/privacy review blocking before release.
- Risk: bundle size increases. Mitigation: preserve `EmployeeCheckInCard` lazy disclosure so the map is loaded only after the employee opens it.
- Rollback: remove `react-map-gl`/`maplibre-gl`, revert helper changes, and keep the existing Leaflet employee component.
