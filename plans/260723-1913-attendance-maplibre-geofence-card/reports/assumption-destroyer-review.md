# Assumption Destroyer Review: Attendance MapLibre Geofence Card

Generated: 2026-07-23

## Code Review Summary

### Scope

- Files reviewed:
  - `plans/260723-1913-attendance-maplibre-geofence-card/plan.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-01-map-runtime-and-view-model.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-02-maplibre-geofence-card.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/phase-03-regression-and-mobile-validation.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/research/researcher-codebase-migration.md`
  - `plans/260723-1913-attendance-maplibre-geofence-card/research/researcher-maplibre-platform.md`
  - `frontend/src/components/employees/EmployeeLocationMap.tsx`
  - `frontend/src/components/employees/EmployeeCheckInCard.tsx`
  - `frontend/src/components/employees/EmployeeLocationMap.test.tsx`
  - `frontend/src/components/employees/EmployeeCheckInCard.test.tsx`
  - `frontend/src/utils/checkInGeofenceGuidance.ts`
  - `frontend/package.json`
- LOC: plan/research/source excerpts reviewed, not implementation diff.
- Focus: plan red-team, Assumption Destroyer lens.
- Scout findings:
  - Graphify confirms `EmployeeLocationMap` is employee-only at runtime, while `LocationMap`, `AttendanceLocationMap`, and `FailedAttemptLocationMap` remain admin Leaflet dependents.
  - The employee map is not a renderer-only component today; it owns status copy, accessibility labels, fallback state, fit-once viewport state, route/arrow behavior, GPS accuracy styling, and reduced-motion behavior.
  - Vitest runs under jsdom with local mocks. It cannot exercise real MapLibre WebGL, style loading, attribution controls, or canvas behavior.

### Overall Assessment

The plan is directionally scoped correctly: employee-only migration, backend geofence authority unchanged, Leaflet retained for admin maps, and no Mapbox token dependency. It is not implementation-ready without tightening several assumptions that can pass component tests but fail in production or regress the employee mobile flow.

Primary risk: the plan under-specifies the boundary between data/view-model logic, real MapLibre runtime behavior, and mocked jsdom tests. That leaves room for a superficially green migration that breaks lazy loading, WebGL fallback, admin Leaflet maps, or far-distance behavior.

### Critical Issues

None found at plan level.

### High Priority

#### 1. Far-distance route suppression has no deterministic threshold or invariant

- Severity: High
- Evidence:
  - Top-level plan says far samples should avoid "a misleading long route" and use fit-bounds only for useful visual comparisons (`plan.md:40-41`, `plan.md:59-62`).
  - Phase 1 says outside status should "show a route only when the employee is close enough" and keep any cutoff "UI-only" (`phase-01-map-runtime-and-view-model.md:34-40`).
  - Phase 2 repeats "local/near cases" and "far cases" without defining either (`phase-02-maplibre-geofence-card.md:39-51`).
  - Research explicitly says there is no existing far-distance cutoff; the real authority is `target.radius_meters`, and adding a second threshold is a product decision unless kept display-only (`researcher-codebase-migration.md:104-116`).
- Impact:
  - Implementers can choose arbitrary values such as 500m, 1km, or viewport-span heuristics and still claim success.
  - Tests will encode the implementer's guess rather than a product invariant.
  - A cutoff too low suppresses useful local guidance; a cutoff too high recreates the continent-scale route problem.
- Required fix:
  - Add a plan decision that defines the display-only suppression rule. Example: "Suppress route when `distanceMeters > max(radiusMeters * 4, 500)`; this affects only route/viewport display and never changes guidance status or submit eligibility."
  - Add boundary tests around exactly-at-threshold and just-over-threshold.
  - Name the helper `shouldShowRouteForDisplay(...)` or equivalent to prevent confusion with geofence authorization.

#### 2. Proposed helper placement conflicts with repo file-responsibility rules

- Severity: High
- Evidence:
  - Frontend project rules say `.tsx` files are UI rendering only; `.ts` files hold business logic, data manipulation, type definitions, and utilities (`frontend/AGENTS.md:23-36`; `docs/code-standards.md:105-112`).
  - Phase 1 allows `buildGeofencePolygon`, `buildGeofenceFeatureCollection`, `buildAccuracyFeature`, `buildRouteFeature`, and `getEmployeeMapViewport` to live "inside `EmployeeLocationMap.tsx`" (`phase-01-map-runtime-and-view-model.md:27-33`).
  - Those helpers are geometry/view-model/data transformation logic, not JSX rendering.
- Impact:
  - This invites a large `.tsx` component that violates local architecture and is harder to test without rendering React.
  - Geometry and viewport decisions become coupled to the MapLibre component, making far-distance and polygon tests brittle.
- Required fix:
  - Change the plan to require a nearby `.ts` helper, for example `frontend/src/components/employees/employee-location-map-model.ts` or `frontend/src/utils/employee-location-map-model.ts`.
  - Keep `EmployeeLocationMap.tsx` as the adapter that renders model output through MapLibre.

#### 3. Lazy-loading preservation is asserted, but the plan does not verify bundle behavior

- Severity: High
- Evidence:
  - Top-level scope explicitly excludes eager map loading before the employee opens the disclosure (`plan.md:30-32`).
  - `EmployeeCheckInCard` currently lazy-loads `EmployeeLocationMap` with `React.lazy` (`EmployeeCheckInCard.tsx:62-64`) and only renders it when `showLocationMap` is true (`EmployeeCheckInCard.tsx:870-899`).
  - Phase 3 checks that `EmployeeLocationMap` mounts only after opening the disclosure (`phase-03-regression-and-mobile-validation.md:35-36`), but existing tests mock the module directly (`EmployeeCheckInCard.test.tsx:35-41`) and do not prove MapLibre remains out of the initial production bundle.
  - Phase 2 imports MapLibre CSS directly from the employee map module (`phase-02-maplibre-geofence-card.md:25-27`), which should be checked in the built assets rather than assumed.
- Impact:
  - The UI test can stay green while `react-map-gl`, `maplibre-gl`, or CSS side effects leak into the initial employee page chunk.
  - Mobile employees pay the bundle cost even when they never open the map, violating the explicit scope.
- Required fix:
  - Add a build-artifact check to Phase 3: after `pnpm build`, inspect Vite output or use a bundle visualizer to confirm MapLibre code is in a lazy chunk and not in the initial employee route chunk.
  - Keep all MapLibre imports confined to the lazy component or a helper imported only by that lazy component. Do not import MapLibre constants/types from `EmployeeCheckInCard` or shared eager modules.

#### 4. WebGL/style fallback cannot be proven by the planned jsdom tests

- Severity: High
- Evidence:
  - Plan acceptance requires WebGL/style load failures to show Vietnamese fallback without blocking the card (`plan.md:61-63`).
  - Phase 2 asks for MapLibre error handling but only names runtime `error` events and fallback text (`phase-02-maplibre-geofence-card.md:52-55`).
  - Phase 3 acknowledges jsdom overfit risk and says to pair tests with visual checks (`phase-03-regression-and-mobile-validation.md:68-72`).
  - Current Vitest config uses jsdom (`frontend/vitest.config.ts:7-11`) and setup only mocks `matchMedia` (`frontend/src/test/setup.ts:3-20`).
- Impact:
  - Map constructor failures, WebGL unsupported states, context loss, and style-fetch failures can still crash or blank the card in real browsers.
  - Mocked `react-map-gl/maplibre` tests will not catch incorrect `onError`, missing `mapLib`, or unsupported WebGL handling.
- Required fix:
  - Add a non-jsdom verification requirement: Playwright or manual browser test that forces one of:
    - invalid style URL,
    - `maplibregl.supported()` false path,
    - WebGL context creation failure or context loss.
  - In the implementation instructions, require an explicit preflight/guard or error boundary around MapLibre construction, not only an `onError` handler after map creation.

### Medium Priority

#### 5. Admin Leaflet retention is stated, but regression verification is optional

- Severity: Medium
- Evidence:
  - Top-level plan says admin migration and full Leaflet removal are out of scope, and admin maps must continue to render through Leaflet (`plan.md:30-32`, `plan.md:57-60`).
  - Research says `LocationMap`, `AttendanceLocationMap`, and `FailedAttemptLocationMap` still depend on Leaflet and have no direct test coverage (`researcher-codebase-migration.md:58-72`, `researcher-codebase-migration.md:98-102`).
  - Phase 3 only says to run an admin smoke check "if practical" (`phase-03-regression-and-mobile-validation.md:70-72`).
- Impact:
  - A dependency or CSS import change can break admin maps while employee tests pass.
  - The plan acceptance criterion cannot be verified if admin smoke remains optional.
- Required fix:
  - Make an admin map smoke check mandatory. At minimum, render `LocationMap` through an existing admin route or add a lightweight mocked component test confirming Leaflet wrappers still import and render.
  - Keep `leaflet`, `react-leaflet`, and `@types/leaflet` lockfile entries explicitly checked after `pnpm add`.

#### 6. Plan metadata says hard-mode reviews are completed before this report exists

- Severity: Medium
- Evidence:
  - `plan.md` says hard-mode plan review is required (`plan.md:70-72`) and then marks status as completed with accepted findings (`plan.md:73-75`).
  - The `reports/` directory was empty before this report was created.
- Impact:
  - The plan overstates review state and can lead the implementation agent to skip unresolved blockers.
  - Accepted/rejected findings cannot be audited back to source reports.
- Required fix:
  - Update plan metadata after all actual red-team reports exist. Link each report and list accepted findings by report, not as unsupported summary text.

#### 7. Dependency assumption should be pinned to verified package metadata, not "latest"

- Severity: Medium
- Evidence:
  - Phase 1 says to run `pnpm add react-map-gl maplibre-gl` without pinning or recording versions (`phase-01-map-runtime-and-view-model.md:25`).
  - Current package metadata checked on 2026-07-23:
    - `react-map-gl@8.1.1` depends on `@vis.gl/react-maplibre@8.1.1`.
    - `react-map-gl@8.1.1` peer-declares `maplibre-gl >=1.13.0`, optional.
    - `@vis.gl/react-maplibre@8.1.1` peer-declares `maplibre-gl >=4.0.0`.
    - `maplibre-gl@6.0.0` requires Node `>=16.14.0`; repo uses Node 20+.
  - `frontend/package.json` currently has no MapLibre or react-map-gl dependency (`frontend/package.json:26-102`).
- Impact:
  - A future install can silently resolve different major versions than the plan/research assumed.
  - Peer compatibility may drift before implementation starts.
- Required fix:
  - Record the verified target versions in the plan, for example `react-map-gl@8.1.1` and `maplibre-gl@6.0.0`, or require implementers to rerun `pnpm view ...` and document the exact versions before adding.
  - Treat lockfile diff review as a required gate.

#### 8. Vietnamese UI and accessibility requirements are underspecified for the new MapLibre canvas

- Severity: Medium
- Evidence:
  - Project requires Vietnamese user-facing text and accessibility/tap-target checks (`frontend/AGENTS.md:58-63`; `docs/standards/review-checklist.md:17-23`, `docs/standards/review-checklist.md:69-72`).
  - Current map has a `role="group"` and Vietnamese aria label derived from route state (`EmployeeLocationMap.tsx:46-55`).
  - Phase 2 says preserve outer structure and status text (`phase-02-maplibre-geofence-card.md:28`) and disable map interactions unless needed for accessibility (`phase-02-maplibre-geofence-card.md:30-34`), but does not specify the screen-reader contract for the MapLibre canvas or attribution links.
- Impact:
  - The map can become visually correct but inaccessible or noisy to screen readers.
  - Required attribution controls can introduce tiny interactive links that fail 44px mobile target expectations or unexpected tab order.
- Required fix:
  - Add acceptance criteria for:
    - Vietnamese `aria-label` / status text equivalent to current route, at-gate, and generic map states.
    - Canvas marked decorative or wrapped so the textual status is the accessible source of truth.
    - Attribution visibility and focus behavior checked on mobile.
    - 320px text overflow check for fallback, outside-distance, nearest-gate, and attribution states.

### Low Priority

#### 9. Basemap provider choice remains open but attribution is an acceptance criterion

- Severity: Low
- Evidence:
  - Plan requires basemap attribution compliance (`plan.md:41`, `phase-02-maplibre-geofence-card.md:30-34`, `phase-02-maplibre-geofence-card.md:79-83`).
  - Research suggests CARTO Positron "only if attribution requirements are satisfied" (`researcher-maplibre-platform.md:40-42`).
- Impact:
  - Implementers may pick any style URL and then discover attribution or availability constraints late.
- Required fix:
  - Select the exact initial style URL and attribution handling in Phase 1, or make "verify provider terms before implementation" a Phase 1 blocking task.

### Edge Cases Found by Scout

- Multiple gates: geofence polygons and viewport bounds must include all gates for no-GPS/gate view, but only nearest gate plus employee for local route view.
- Invalid target: `getCheckInGeofenceGuidance` returns `no_target` when target/gates/radius are missing or invalid; MapLibre should not initialize unnecessarily for this state (`checkInGeofenceGuidance.ts:38-44`).
- Boundary equality: current authority treats `distance + accuracy <= radius` as inside (`checkInGeofenceGuidance.ts:65-72`); new helper tests must preserve equality.
- GPS accuracy styling: current UI uses `<50m` only as badge styling, not authorization (`EmployeeLocationMap.tsx:63-67`; `researcher-codebase-migration.md:104-116`).
- Reduced motion: current animated direction arrow checks `prefers-reduced-motion` and stops animation (`EmployeeLocationMap.tsx:295-317`); a static MapLibre replacement is safer than reintroducing animation.
- Tile/style failure: current Leaflet fallback is `tileerror`-based and narrow (`EmployeeLocationMap.tsx:72-96`); MapLibre failures include style, source, WebGL, and context creation paths.
- Admin retention: admin map wrappers have no direct tests, so dependency retention alone does not prove runtime retention.

### Positive Observations

- The plan correctly preserves backend/API geofence authority and keeps `getCheckInGeofenceGuidance()` as the decision source (`plan.md:24-27`; `phase-01-map-runtime-and-view-model.md:33-40`).
- The plan correctly retains Leaflet packages for admin maps (`plan.md:36-39`; `phase-01-map-runtime-and-view-model.md:25-26`).
- The plan correctly avoids the Mapbox entrypoint and token dependency (`plan.md:36-37`; `researcher-maplibre-platform.md:11-18`).

### Recommended Actions

1. Define the display-only far-route suppression threshold and add boundary tests.
2. Require geometry/view-model helpers to live in a `.ts` helper, not inside `EmployeeLocationMap.tsx`.
3. Add a production build chunk check proving MapLibre stays lazy-loaded behind the disclosure.
4. Add real browser fallback validation for style/WebGL failure, not just jsdom mocks.
5. Make admin Leaflet smoke verification mandatory.
6. Correct the red-team review metadata after all reports exist and are linked.
7. Pin or re-verify exact package versions immediately before dependency install.
8. Add explicit Vietnamese accessibility acceptance criteria for the MapLibre canvas, status text, attribution, and mobile overflow.

### Metrics

- Type Coverage: Not measured in this review.
- Test Coverage: Not measured in this review.
- Linting Issues: Not measured in this review.

### Unresolved Questions

- What exact display-only cutoff should separate "near outside" from "far outside" route rendering?
- Which basemap style URL and attribution text/control will be used for the first implementation?
- Will Phase 3 use Playwright with mocked style failure, manual browser QA, or both for WebGL/style fallback validation?
