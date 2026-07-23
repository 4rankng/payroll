---
phase: 2
title: "MapLibre Geofence Card"
status: pending
effort: "L"
---

# Phase 2: MapLibre Geofence Card

## Overview

Replace the employee card's Leaflet renderer with `react-map-gl/maplibre` while preserving the existing component contract: `EmployeeLocationMap({ target, sample })`.

## Related Code Files

- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/employee-location-map-model.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.test.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeCheckInCard.tsx`
- `/Users/dev/Documents/projects/payroll/frontend/src/styles/base.css`
- `/Users/dev/Documents/projects/payroll/frontend/src/utils/checkInGeofenceGuidance.ts`
- `/Users/dev/Documents/projects/payroll/frontend/src/utils/geoDistance.ts`

## Implementation Steps

1. Replace `react-leaflet` imports in `/Users/dev/Documents/projects/payroll/frontend/src/components/employees/EmployeeLocationMap.tsx` with:
   - `Map`, `Marker`, `Source`, and `Layer` from `react-map-gl/maplibre`.
   - `maplibre-gl/dist/maplibre-gl.css`.
2. Preserve the outer card structure, `role="group"`, Vietnamese status title/description, radius summary, nearest gate summary, and GPS accuracy badge behavior.
3. Keep `EmployeeCheckInCard.tsx` unchanged unless a compile-time integration issue requires a narrow import or suspense fallback adjustment. The map should remain user-opened, not eagerly mounted.
4. Render the MapLibre map as a status card:
   - no zoom/pan/navigation controls for normal use,
   - no scroll-wheel zoom,
   - no map interaction unless implementation needs it for accessibility,
   - compact or visible attribution when required by the basemap.
   - accessible text/status outside the canvas remains the screen-reader source of truth; mark the canvas/decorative map internals accordingly where MapLibre allows.
5. Render geofence data with GeoJSON:
   - fill layer for translucent radius area,
   - line layer for radius boundary,
   - point/marker layer or React markers for each gate.
6. Render employee GPS data:
   - employee marker when `sample` exists and the employee has opened the disclosure,
   - accuracy radius as a GeoJSON polygon or MapLibre circle approximation that remains clear at card scale,
   - route line only for local/near cases where both points remain visually meaningful.
   - no visible coordinate text, no viewport/sample logging, and no remote tile loading around far outside samples unless the Phase 1 provider/privacy decision explicitly permits it.
7. Replace the current long dashed route behavior with outside-status behavior for far samples:
   - keep the office/geofence legible,
   - show distance/over-radius in the existing status area or a compact map overlay,
   - avoid implying a navigable route.
8. Fit the viewport after the map loads and when target configuration changes:
   - fit all gate/geofence polygon extents when GPS is absent,
   - fit sample plus nearest gate for local cases,
   - fit gate/geofence polygon extents only for far cases,
   - preserve the current "fit first usable route once" intent so user map state is not repeatedly reset.
9. Add MapLibre error handling:
   - style load/source errors should set the existing map-failed fallback state,
   - WebGL unsupported, context creation failure, or constructor-time failure should show `Không tải được bản đồ.` or equivalent Vietnamese text,
   - map failures should not hide check-in status, radius, or nearest gate data.
   - map runtime errors must stay isolated inside `EmployeeLocationMap`, not escape to the surrounding `EmployeeCheckInCard` suspense boundary.
10. Prefer no map-specific animation. If route, marker, or emphasis animation is added, disable it under `prefers-reduced-motion: reduce` and cover that behavior.
11. Keep `frontend/src/styles/base.css` edits narrow. Reuse existing hooks such as `gps-accuracy-confirmed`, `checkpoint-route-reveal`, and `checkpoint-marker-emphasis` only if they still match the MapLibre markup.
12. Preserve Vietnamese accessibility text:
   - route state: map is heading to the nearest gate with straight-line distance,
   - at-gate state: employee is at the gate,
   - generic/no-GPS state: attendance area map,
   - fallback state: map failed while status/radius/gate text remains readable.

## Success Criteria

- [ ] `EmployeeLocationMap` no longer imports `react-leaflet`, `leaflet`, or `leaflet/dist/leaflet.css`.
- [ ] The employee map renders through `react-map-gl/maplibre`.
- [ ] Nearby check-in cases show employee, nearest gate, geofence radius, and a compact straight-line indicator.
- [ ] Far outside-radius cases keep the geofence visible and do not draw a long route across a wide viewport.
- [ ] Basemap load/WebGL failure leaves a usable Vietnamese status card.
- [ ] Required attribution is visible and does not create broken focus order or mobile text overflow.
- [ ] Admin Leaflet maps are unchanged.

## Verification

- `cd /Users/dev/Documents/projects/payroll/frontend && pnpm test:run src/components/employees/EmployeeLocationMap.test.tsx`
- Manual or Playwright visual check at mobile widths around 320px, 375px, and 430px for:
  - no GPS,
  - inside radius,
  - inaccurate GPS,
  - near outside radius,
  - far outside radius,
  - map load failure.
- Reduced-motion check for route/marker/accent behavior.
- Accessibility check for Vietnamese status labels, canvas/decorative treatment, attribution focus, and 320px text overflow.

## Risks and Rollback

- Risk: MapLibre attribution is accidentally hidden. Mitigation: explicitly verify provider attribution requirements during implementation and keep a compact visible attribution path.
- Risk: WebGL support differs across employee devices. Mitigation: test the fallback path and keep the surrounding check-in status independent of the map canvas.
- Risk: style URL failure makes the card look broken. Mitigation: handle MapLibre `error` events and keep the existing fallback message.
- Risk: exact GPS position is disclosed to a third-party tile provider in far outside cases. Mitigation: keep far mode centered on the gate/geofence unless provider/privacy review allows sample-centered tiles.
- Rollback: restore the Leaflet version of `EmployeeLocationMap.tsx` and remove MapLibre-only helper/test changes; admin maps are unaffected.
