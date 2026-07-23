## Code Review Summary

### Scope
- Files: `plan.md`, `phase-01-map-runtime-and-view-model.md`, `phase-02-maplibre-geofence-card.md`, `phase-03-regression-and-mobile-validation.md`, plan research notes, and current employee attendance map/geofence code.
- LOC: 380 reviewed lines across the plan and research files, plus targeted source evidence from the frontend and backend attendance/geofence paths.
- Focus: plan-level security, privacy, compliance, and trust-boundary readiness before implementation.
- Scout findings: graphify scoped the affected surface to `EmployeeLocationMap`, `EmployeeCheckInCard`, `useContinuousLocation`, `checkInGeofenceGuidance`, and admin Leaflet wrappers. Current employee Leaflet map disables attribution and calls public OSM tiles; backend remains authoritative for geofence enforcement.

### Overall Assessment
The plan is directionally sound on provider choice, backend authority, attribution, WebGL fallback, admin Leaflet preservation, and lazy loading. It is not ready from a privacy/compliance lens until it explicitly handles remote tile/style leakage and reconciles the frontend advisory geofence model with the backend's broader acceptance rules.

### Critical Issues
None found that would directly create a new auth bypass or data-loss path in the plan as written. The plan keeps backend/API geofence authority out of scope (`plan.md:26`, `plan.md:31`) and the backend still enforces geofence on check-in and checkout (`backend/internal/app/services/attendance/attendance_service.go:521`, `backend/internal/app/services/attendance/attendance_service.go:669`).

### High Priority
- [plans/260723-1913-attendance-maplibre-geofence-card/phase-01-map-runtime-and-view-model.md:40] The plan says to keep the MapLibre style in a narrow constant, and the research recommends CARTO Positron (`researcher-maplibre-platform.md:40-42`), but there is no privacy requirement for remote style/tile requests. Opening the employee map will cause the browser to request map tiles for the employee/gate area; that exposes approximate worksite and live check-in area to the basemap provider via tile coordinates and client IP. This is sensitive workforce location data, not generic UI telemetry. Current code already calls public OSM tiles (`frontend/src/components/employees/EmployeeLocationMap.tsx:93-95`), so the migration is the point to fix the compliance boundary rather than preserve it.
  Fix: add an explicit provider/privacy decision before implementation: allowed provider, attribution/license terms, whether employee GPS/sample tiles may be requested from a third party, CSP/connect-src/img-src requirements, and a self-host/proxy/offline-style fallback if third-party disclosure is unacceptable. Make it an acceptance criterion and a Phase 3 validation item.

- [plans/260723-1913-attendance-maplibre-geofence-card/phase-03-regression-and-mobile-validation.md:26-34] The test plan mocks `react-map-gl/maplibre` and asserts props/GeoJSON, but it does not require any real-browser network inspection for style/tile/attribution behavior. A jsdom mock can pass while production hides attribution, fetches an unexpected Mapbox URL, sends a token, or loads a style that fans out to multiple third-party domains. This is exactly the kind of compliance failure that passes CI.
  Fix: add a Playwright/manual network check for the map disclosure: verify no `api.mapbox.com` or Google requests, enumerate style/tile hostnames, verify attribution is visible in the rendered card, and confirm style load failure uses the Vietnamese fallback without hiding status/radius/gate data.

### Medium Priority
- [plans/260723-1913-attendance-maplibre-geofence-card/plan.md:26] The plan calls `checkInGeofenceGuidance.ts` the source for `inside`, `outside`, and `inaccurate` decisions, but the frontend guidance is not semantically identical to backend authority. Frontend `inside` requires `distance + accuracy <= radius` (`frontend/src/utils/checkInGeofenceGuidance.ts:65-72`); backend also accepts a point in the inner half of the zone when `accuracy <= radius` (`backend/internal/app/services/attendance/attendance_service.go:380-388`). Treating the frontend helper as the decision source can mislabel a backend-acceptable sample as weak/inaccurate and cause future UI gating regressions.
  Fix: reword the plan so frontend guidance is explicitly advisory/status-only. Add a regression case for the backend "inner half with zone-scale accuracy" condition and assert the client still submits when appropriate or, at minimum, does not claim its guidance is the authorization decision.

- [plans/260723-1913-attendance-maplibre-geofence-card/phase-02-maplibre-geofence-card.md:39-42] The plan says to render the employee marker and accuracy radius whenever `sample` exists. That is useful for self-service, but it can display a precise live coordinate in a shared workplace setting and cause remote tile loads around that point. The plan does not define whether the marker should be rounded, hidden in far cases, or suppressed in fallback/privacy modes.
  Fix: define employee-location display policy: exact marker only to the logged-in employee after disclosure, no coordinate text, no logging of map viewport/sample, and far/outside mode should avoid panning or tile-loading around the sample unless product explicitly accepts that disclosure.

### Low Priority
- [plans/260723-1913-attendance-maplibre-geofence-card/plan.md:70-75] The plan already states "Red Team Review" completed before this report exists. That can create false confidence in downstream execution.
  Fix: update plan status outside this review report to link the actual reports and accepted/rejected findings after all requested red-team reviews are complete.

### Edge Cases Found by Scout
- Current `EmployeeLocationMap` disables Leaflet attribution (`frontend/src/components/employees/EmployeeLocationMap.tsx:84-90`) while using OSM tiles (`frontend/src/components/employees/EmployeeLocationMap.tsx:93-95`). The MapLibre plan correctly calls out attribution, but it needs a hard validation gate because this is already broken behavior.
- Backend geofence validation is authoritative and richer than the frontend advisory helper. Check-in validates server-side (`backend/internal/app/services/attendance/attendance_service.go:521-524`), checkout validates server-side (`backend/internal/app/services/attendance/attendance_service.go:669-678`), and the frontend cold path deliberately submits accurate outside samples so the server can reject and audit them (`frontend/src/components/employees/EmployeeCheckInCard.tsx:694-701`).
- The public employee map contract is narrow: `EmployeeCheckInCard` lazy-loads the map (`frontend/src/components/employees/EmployeeCheckInCard.tsx:62-64`) and mounts it only after disclosure (`frontend/src/components/employees/EmployeeCheckInCard.tsx:870-899`). Preserving that is important for bundle size and privacy-by-default.

### Positive Observations
- The plan explicitly avoids Google Maps/Mapbox-token dependency (`plan.md:31`) and requires the MapLibre entrypoint (`plan.md:37`, `phase-02-maplibre-geofence-card.md:25-27`).
- The plan preserves backend geofence authority and excludes backend/API changes (`plan.md:26`, `plan.md:31`).
- The plan includes WebGL/style fallback requirements (`phase-02-maplibre-geofence-card.md:52-55`) and mobile fallback validation (`phase-03-regression-and-mobile-validation.md:71-77`).

### Recommended Actions
1. Add a blocking privacy/provider acceptance criterion covering third-party tile/style requests, attribution/license obligations, and allowed hostnames.
2. Add real-browser network and attribution validation to Phase 3, not only MapLibre component mocks.
3. Reword frontend geofence guidance as advisory/status-only and add tests for backend/frontend semantic drift around zone-scale accuracy.
4. Define a display policy for exact employee marker rendering and far/outside cases so location privacy is intentional.
5. Update the plan's red-team status only after all actual review reports are present.

### Metrics
- Type Coverage: Not measured; no implementation reviewed.
- Test Coverage: Not measured; plan/test matrix reviewed only.
- Linting Issues: Not measured; no code changed.

### Unresolved Questions
- Is third-party disclosure of employee/worksite tile coordinates acceptable under the product's privacy/compliance posture, or should employee attendance maps use self-hosted/proxied tiles?
- Which exact basemap style URL and provider terms will implementation use?
- Should the employee map ever load tiles around a far outside employee sample, or only around configured gates/geofence?
