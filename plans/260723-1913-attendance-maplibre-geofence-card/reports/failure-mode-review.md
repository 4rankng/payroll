# Failure Mode Review: Attendance MapLibre Geofence Card

Generated: 2026-07-23
Reviewer lens: Failure Mode Analyst

## Scope

- Plan files reviewed:
  - `plans/260723-1913-attendance-maplibre-geofence-card/plan.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-01-map-runtime-and-view-model.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-02-maplibre-geofence-card.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-03-regression-and-mobile-validation.md`
- Evidence files reviewed:
  - `plans/260723-1913-attendance-maplibre-geofence-card/research/researcher-codebase-migration.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/research/researcher-maplibre-platform.md`
  - `frontend/src/components/employees/EmployeeLocationMap.tsx`
  - `frontend/src/components/employees/EmployeeLocationMap.test.tsx`
  - `frontend/src/components/employees/EmployeeCheckInCard.tsx`
  - `frontend/src/components/employees/EmployeeCheckInCard.test.tsx`
  - `frontend/src/utils/checkInGeofenceGuidance.ts`
  - `frontend/src/utils/checkInGeofenceGuidance.test.ts`
  - `frontend/src/styles/base.css`
- Focus: runtime map failures, WebGL unsupported, style/tile network failures, fit bounds, route suppression, geofence polygon math, reduced motion, mobile overlap, and test gaps.

## Overall Assessment

The plan is directionally sound and correctly preserves backend geofence authority, employee-only scope, Leaflet retention for admin maps, lazy disclosure, and Vietnamese fallback copy. It is not yet hard enough for production because several core failure modes remain underspecified: the far-route cutoff is left to implementer judgment, fit-bounds acceptance does not require radius-inclusive bounds, WebGL/style failure tests are not executable enough, and mobile validation does not explicitly cover the map-open state against the fixed attendance dock.

No trust-boundary or backend authorization risk was found because the plan keeps check-in eligibility delegated to existing guidance/backend contracts.

## Critical Issues

None found.

## High Priority

### H1 - Far-route suppression has no deterministic cutoff

Severity: High

Evidence:
- The top-level plan requires far outside-radius samples to suppress the long route, but only says "far" and "local" without a numeric rule: `plan.md:40-41`, `plan.md:60-62`.
- Phase 1 says route is shown only when the employee is "close enough" and any cutoff is "UI-only", but does not define the cutoff or its boundary behavior: `phase-01-map-runtime-and-view-model.md:34-40`.
- Phase 2 repeats "local/near" and "far" without defining either: `phase-02-maplibre-geofence-card.md:39-50`.
- Research explicitly warns that there is no existing far-distance cutoff and that the real geofence threshold is `target.radius_meters`: `researcher-codebase-migration.md:104-116`.
- Current code draws and fits a route for any non-identical sample/gate, regardless of distance: `frontend/src/components/employees/EmployeeLocationMap.tsx:41-44`, `frontend/src/components/employees/EmployeeLocationMap.tsx:133-156`, `frontend/src/components/employees/EmployeeLocationMap.tsx:234-240`.

Failure mode:
- Two implementers can choose different "near" thresholds and both pass the written plan.
- A too-low cutoff suppresses useful nearby guidance; a too-high cutoff preserves the production failure by fitting a city/region-scale route.
- A future refactor may accidentally treat the UI cutoff as geofence eligibility if the helper is not explicitly separated from `getCheckInGeofenceGuidance()`.

Recommended plan fix:
- Add a named view-only helper contract such as `shouldShowEmployeeRoute(guidance, target)` and define exact threshold semantics in the plan.
- Require boundary tests for below, equal, and above the cutoff.
- Require tests proving `guidance.status`, `distanceMeters`, `overByMeters`, and check-in eligibility remain unchanged when only route visibility changes.

### H2 - Fit bounds do not require geofence-radius-inclusive bounds

Severity: High

Evidence:
- Acceptance requires the geofence radius to remain visible: `plan.md:59-61`.
- Phase 2 says to fit "all gates/geofence" when GPS is absent and "gate/geofence only" for far cases: `phase-02-maplibre-geofence-card.md:47-51`.
- Phase 3 test language only says "fit-bounds is called once", not that the bounds include the radius geometry or polygon bbox: `phase-03-regression-and-mobile-validation.md:26-35`.
- Current Leaflet fallback fits only gate coordinates when GPS is absent, which is insufficient for a single-gate radius view: `frontend/src/components/employees/EmployeeLocationMap.tsx:245-247`.

Failure mode:
- Single-gate projects can fit to a degenerate point and rely on `maxZoom`, leaving a 150m geofence clipped or visually misleading on small cards.
- Multi-gate projects can fit gate centers while clipping radius edges.
- Far outside cases can pass "fitBounds called" tests while still failing the user-facing requirement that the office/geofence is legible.

Recommended plan fix:
- Require `getEmployeeMapViewport()` to compute bounds from the geofence polygon bbox, not just gate center points.
- Add tests asserting bounds include north/south/east/west radius extents for a single gate and for multiple gates.
- Add a mobile visual assertion that the geofence ring is fully visible for no-GPS and far-outside cases.

### H3 - WebGL/style failure fallback is specified but not made executable

Severity: High

Evidence:
- The plan requires WebGL/style load failures to show a readable Vietnamese fallback: `plan.md:62-63`.
- Phase 2 says style/source errors and WebGL unsupported/context creation failures should set fallback state: `phase-02-maplibre-geofence-card.md:52-55`.
- Phase 3 includes "map load failure" in manual/mobile checks, but component tests do not explicitly require mocked `onError`, unsupported WebGL, or constructor/context creation failure behavior: `phase-03-regression-and-mobile-validation.md:68-77`.
- Platform research identifies WebGL and external style URL failures as first-order risks: `researcher-maplibre-platform.md:33-38`.
- Current component fallback only handles Leaflet tile errors through `TileLayer.eventHandlers.tileerror`: `frontend/src/components/employees/EmployeeLocationMap.tsx:21-22`, `frontend/src/components/employees/EmployeeLocationMap.tsx:72-76`, `frontend/src/components/employees/EmployeeLocationMap.tsx:93-96`.

Failure mode:
- A MapLibre `error` event may be covered, while WebGL unsupported or context creation failure still throws during mount and breaks the whole check-in card.
- CI can pass jsdom component tests that mock `<Map>` as a harmless div without proving production fallback.
- Style 404/CSP/network failure can leave a blank canvas instead of the Vietnamese fallback while check-in status appears normal.

Recommended plan fix:
- Require component tests for:
  - `react-map-gl/maplibre` `onError` invoking fallback.
  - WebGL unsupported/context creation failure path.
  - fallback preserving status title, radius, nearest gate, and accuracy text.
- Require a browser/manual validation case that blocks the style URL or simulates offline tiles and captures the visible fallback.
- Specify that map runtime errors must be isolated inside `EmployeeLocationMap`, not allowed to escape to the surrounding `EmployeeCheckInCard` suspense boundary.

## Medium Priority

### M1 - Mobile validation does not explicitly cover the map-open card against the fixed dock

Severity: Medium

Evidence:
- Employee views are mobile-first by project convention: `frontend/AGENTS.md:58-64`, `docs/code-standards.md:169-171`.
- Existing dock is fixed at bottom with `z-40`: `frontend/src/components/employees/EmployeeAttendanceActionDock.tsx:69`.
- Current map card relies on an isolated `z-0` wrapper and an internal overlay with `z-[500]`: `frontend/src/components/employees/EmployeeLocationMap.tsx:47-49`, `frontend/src/components/employees/EmployeeLocationMap.tsx:77-83`.
- Phase 3 asks for mobile widths and no overlap but does not explicitly require the disclosure-open employee portal state, fixed dock present, safe-area bottom, or enlarged text case: `phase-03-regression-and-mobile-validation.md:43-49`, `phase-03-regression-and-mobile-validation.md:57`.
- Existing E2E coverage checks the attendance toolbar at 200% text but not the map-open disclosure: `frontend/tests/e2e/employee-portal.spec.ts:313-348`.

Failure mode:
- MapLibre controls/attribution/canvas overlays can visually collide with the fixed attendance dock or card overlays on 320px screens while component tests pass.
- Long Vietnamese status/distance copy can truncate into unreadable state in the exact mobile widths this plan targets.

Recommended plan fix:
- Require Playwright/manual screenshots with the employee map disclosure open at 320px, 375px, and 430px, including the fixed attendance dock.
- Add one 200% text or browser font-size stress case with the map disclosure open.
- Include explicit checks for attribution visibility, no overlap with the dock, readable fallback copy, and no button/text overflow.

### M2 - Geofence polygon helper lacks numeric edge-case requirements

Severity: Medium

Evidence:
- Phase 1 proposes `buildGeofencePolygon(gate, radiusMeters)` and `buildGeofenceFeatureCollection(target)`: `phase-01-map-runtime-and-view-model.md:27-33`.
- Top-level plan requires GeoJSON polygon approximations so the radius remains meter-based: `plan.md:39`.
- Existing geofence authority uses haversine distance and treats invalid target/gates/radius as `no_target`: `frontend/src/utils/checkInGeofenceGuidance.ts:21-35`, `frontend/src/utils/checkInGeofenceGuidance.ts:38-44`.
- The project domain is Vietnam, so antimeridian handling is not a product-critical case, but a generic GeoJSON helper can still produce invalid geometry for malformed coordinates if it lacks validation.

Failure mode:
- A polygon implementation based on naive degree offsets can distort east/west radius, especially if copied later outside Vietnam.
- Invalid lat/lng or non-finite radius can produce invalid GeoJSON and crash MapLibre source parsing instead of falling back to `no_target`.
- Antimeridian crossing is unlikely for current data, but unnormalized longitudes can create self-crossing polygons if bad data ever reaches the UI.

Recommended plan fix:
- Require helper tests for segment count/closed ring, finite coordinates, radius zero/negative behavior, and latitude/longitude validity.
- State that invalid target geometry should avoid mounting MapLibre and use the existing `no_target` status/fallback path.
- Treat antimeridian support as non-blocking for Vietnam-only data, but document whether the helper normalizes longitudes or rejects out-of-range input.

### M3 - Reduced-motion validation is too conditional

Severity: Medium

Evidence:
- Current direction arrow animation explicitly checks `prefers-reduced-motion: reduce`: `frontend/src/components/employees/EmployeeLocationMap.tsx:295-317`.
- CSS disables map/checkpoint animations under reduced motion: `frontend/src/styles/base.css:907-917`.
- Phase 2 only says "If animated direction markers are reintroduced": `phase-02-maplibre-geofence-card.md:56-57`.
- Research recommends preserving reduced-motion behavior or preferring static markers: `researcher-maplibre-platform.md:29-31`.

Failure mode:
- The migration may remove the old arrow component but introduce MapLibre marker/layer transitions or CSS animations that are not covered by reduced-motion tests.
- Users with reduced motion enabled can still get animated route/marker effects because the plan only names direction markers.

Recommended plan fix:
- Require either no map-specific animation at all, or a test/assertion that all route/marker/accent animations are disabled under reduced motion.
- Include reduced-motion mode in the mobile visual checklist.

## Low Priority

### L1 - Attribution compliance is accepted but provider choice is still underdefined

Severity: Low

Evidence:
- Plan requires attribution to remain visible/compliant: `plan.md:41`.
- Phase 2 allows compact/visible attribution "when required by the basemap": `phase-02-maplibre-geofence-card.md:30-35`.
- Platform research suggests CARTO Positron only if attribution requirements are satisfied: `researcher-maplibre-platform.md:40-42`.

Failure mode:
- Implementation can choose a style URL and forget to encode its exact attribution text/control behavior in tests or manual QA.

Recommended plan fix:
- Name the initial style provider and required attribution display in Phase 1's style constant decision.
- Add an assertion or manual QA item that attribution is visible in normal and fallback/noninteractive modes.

## Edge Cases Found

- Nearby outside sample just below the route cutoff.
- Outside sample exactly at the route cutoff.
- Outside sample just above the route cutoff.
- Far outside sample where `distanceMeters` is valid but route is suppressed.
- Single gate with no GPS where bounds must include the full radius, not only the center.
- Multiple gates where each radius contributes to the bbox.
- Style URL blocked/404 while WebGL is otherwise supported.
- WebGL unsupported or context creation failure before MapLibre finishes mounting.
- Map disclosure open with fixed attendance dock at 320px width.
- Map disclosure open with 200% text.
- Reduced-motion mode with route/marker emphasis present.
- Invalid or non-finite target geometry causing `no_target` rather than invalid GeoJSON.
- Antimeridian crossing is not product-critical for Vietnam data but should be consciously rejected or normalized if helper is generic.

## Plan Follow-up Recommendations

1. Define the exact view-only route suppression cutoff and boundary behavior before implementation.
2. Strengthen viewport helper acceptance so fit bounds include geofence polygon extents, not only gate/sample points.
3. Add executable fallback tests for style errors and WebGL/context failure.
4. Expand mobile QA to include the map disclosure open with the fixed attendance dock and high text scale.
5. Add numeric geometry tests for geofence polygon generation and invalid target handling.

## Review Checklist

- Concurrency: no async race in plan scope beyond map load/error ordering; fallback isolation still needs explicit tests.
- Error boundaries: map runtime/style/WebGL failures are acknowledged but not yet fully executable.
- API contracts: backend geofence authority is preserved; view-only cutoff needs stronger separation from eligibility.
- Backwards compatibility: admin Leaflet dependency retention is correctly preserved.
- Input validation: invalid geometry handling is underspecified.
- Auth/authz paths: no sensitive backend operation changes in scope.
- N+1/query efficiency: not applicable; frontend-only map rendering.
- Data leaks: no PII or secret exposure found in plan.
- Fact-checked: cited file paths and current behavior verified against plan, research, graphify query output, and local source.
