# Research Report: Employee attendance map migration assessment

Generated: 2026-07-23

## Executive Summary

The employee attendance map is still a Leaflet-driven component with a stable public contract: `EmployeeCheckInCard` lazy-loads `EmployeeLocationMap`, passes `CheckInTarget` plus an optional live `LocationSample`, and the map renders geofence state via `getCheckInGeofenceGuidance`. The current behavior is not a generic "distance map"; it is a geofence classifier with one explicit radius boundary and one separate GPS-accuracy styling threshold.

Leaflet cannot be removed outright because the admin attendance maps still depend on it through `LocationMap`, `AttendanceLocationMap`, and `FailedAttemptLocationMap`. The safe migration boundary is employee-only unless the admin maps are also being rewritten. If the goal is a MapLibre swap for the employee card, the smallest credible implementation set is the employee map component, its test file, and only the guidance util if the status semantics change. `EmployeeCheckInCard` does not need to change if the prop contract stays intact.

## Research Methodology

- Sources consulted: 3+ independent local sources plus graph/query output
- Date of evidence: 2026-07-23
- Key terms: `EmployeeLocationMap`, `checkInGeofenceGuidance`, `AttendanceLocationMap`, `FailedAttemptLocationMap`, `leaflet`, `react-leaflet`, `employee-type-*`, `checkpoint-*`

## Key Findings

### 1. Exact Current Behavior

#### Employee attendance map

- `frontend/src/components/employees/EmployeeLocationMap.tsx` uses `react-leaflet` and `leaflet` directly, imports `leaflet/dist/leaflet.css`, and renders a full card with:
  - geofence status title and description,
  - a GPS accuracy badge,
  - a 60vh-ish map area,
  - a route line and direction arrow when location and target gate differ,
  - a circular geofence ring for every gate,
  - a "you are here" marker when sample GPS exists,
  - a bottom summary of radius and nearest gate.
- The map center is:
  - `sample` position when a GPS sample exists,
  - otherwise the first gate.
- `FitLocationBounds` fits the map once for the first usable route, or once to the gates when there is no sample.
- `createDirectionArrow` returns `null` when start and target are identical; otherwise it creates a Leaflet div icon rotated by bearing.
- `AnimatedDirectionArrow` animates the marker unless `prefers-reduced-motion: reduce` is active.
- The map status copy is derived from `statusTitle()` and `statusDescription()` in the same file.

#### Geofence guidance logic

- `frontend/src/utils/checkInGeofenceGuidance.ts` is the canonical decision layer.
- It returns:
  - `no_target` when target/gates/radius are missing or invalid,
  - `no_position` when there is no sample,
  - `inside` when `distance + accuracy <= radius_meters`,
  - `inaccurate` when the coordinate is inside the radius but GPS uncertainty crosses the boundary,
  - `outside` otherwise.
- `distanceMeters` is the nearest-gate straight-line distance, not route distance.
- `overByMeters` is `max(0, nearestDistance - radius_meters)`.
- `getCheckInGeofenceInstruction()` only gives an "outside" instruction, and gives an "inaccurate" instruction only when `accuracyMeters <= radiusMeters`.

#### Employee card integration

- `frontend/src/components/employees/EmployeeCheckInCard.tsx` lazy-loads `EmployeeLocationMap`.
- The disclosure is user-opened; the card passes only `checkInTarget` and `visibleLocationSample`.
- The current card already treats the map as a nested detail view, not a separate route.

#### Admin maps

- `frontend/src/components/admin-dashboard/LocationMap.tsx` is the shared admin Leaflet map shell.
- `AttendanceLocationMap.tsx` and `FailedAttemptLocationMap.tsx` build their own point/checkpoint models and pass them into `LocationMap`.
- `frontend/src/components/attendance/AttendanceMapDialog.tsx` and `frontend/src/components/admin-dashboard/HealthDrilldownSheet.tsx` are the runtime callers for the admin map wrappers.

### 2. Verified Callers and Tests

| Symbol / component | Runtime callers | Test files | Notes |
|---|---:|---:|---|
| `EmployeeLocationMap` | 1 | 1 | Called only from `EmployeeCheckInCard`; tested in `EmployeeLocationMap.test.tsx` |
| `EmployeeCheckInCard` | 2 | 1 | Used by `FlexiblePayEmployeePage` and the employee portal test mock path |
| `AttendanceLocationMap` | 2 | 0 | Called by `AttendanceMapDialog` and `HealthDrilldownSheet`; no direct coverage found |
| `FailedAttemptLocationMap` | 1 | 0 | Called by `HealthDrilldownSheet`; no direct coverage found |
| `LocationMap` | 2 | 0 | Shared admin wrapper only |
| `getCheckInGeofenceGuidance` | 2 direct consumers | 1 | Consumed by `EmployeeLocationMap` and `EmployeeCheckInCard`; tested in `checkInGeofenceGuidance.test.ts` |

### 3. Style Hooks That Matter

- Employee typography hooks are real contract points in `frontend/src/styles/base.css`:
  - `employee-type-card-title`
  - `employee-type-body`
  - `employee-type-body-sm`
  - `employee-type-pill`
- Map animation / emphasis hooks are also real:
  - `gps-accuracy-confirmed`
  - `checkpoint-route-reveal`
  - `checkpoint-marker-emphasis`
- These hooks are covered by the employee map test and by the broader employee styling system.

### 4. Package and Lockfile Impact

- `frontend/package.json` already contains:
  - `leaflet`
  - `react-leaflet`
  - `@types/leaflet`
- `frontend/pnpm-lock.yaml` already resolves those packages.
- Result: an employee-only migration does not have to remove or add Leaflet dependencies unless the new provider is MapLibre-specific.
- If MapLibre is introduced for the employee map, package/lockfile changes are required, but Leaflet still must remain because admin maps depend on it.

### 5. Leaflet Removal Verdict

- Leaflet cannot be removed from the repo as a whole while admin maps still exist.
- It can only be removed if `LocationMap`, `AttendanceLocationMap`, and `FailedAttemptLocationMap` are also migrated or deleted.
- For this plan, Leaflet must remain.

### 6. Far-Distance UX Threshold and Data Logic

- There is no separate "far-distance" cutoff in the employee map logic.
- The real threshold is the geofence radius from `target.radius_meters`.
- Status selection is based on:
  - nearest gate distance,
  - GPS accuracy,
  - whether the coordinate is inside or outside the radius.
- The only extra numeric cutoff in the employee map UI is `sample.accuracy < 50`, which only changes badge styling to `gps-accuracy-confirmed`.
- Therefore:
  - do not invent a second distance threshold,
  - preserve the radius-based boundary,
  - preserve the 50m accuracy styling threshold unless product intent changes.

## Comparative Analysis

### Recommended implementation boundary

1. `frontend/src/components/employees/EmployeeLocationMap.tsx`
2. `frontend/src/components/employees/EmployeeLocationMap.test.tsx`
3. `frontend/src/utils/checkInGeofenceGuidance.ts` only if status or threshold semantics change

### Add only if the migration changes the visible contract

- `frontend/src/components/employees/EmployeeCheckInCard.tsx` if the prop shape, disclosure behavior, or guidance copy changes.
- `frontend/src/styles/base.css` if new motion or class hooks are introduced.
- `frontend/package.json` and `frontend/pnpm-lock.yaml` if MapLibre or another new map stack is actually added.

### Trade-offs

| Option | Performance | Complexity | Maintenance | Risk |
|---|---|---|---|---|
| Keep employee Leaflet behavior unchanged | Best short-term | Lowest | Lowest | Lowest |
| Migrate employee map only, keep admin Leaflet | Good | Medium | Medium | Moderate |
| Migrate employee + admin maps together | Mixed | High | High | High |

### Ranked choice

1. Employee-only migration with Leaflet retained for admin maps.
2. Shared map abstraction only if both employee and admin surfaces are in scope.
3. Full Leaflet removal last, only after all admin wrappers are migrated.

## Implementation Recommendations

### Smallest File Set

If the goal is a provider swap for the employee attendance/geofence card only, the smallest safe file set is:

- `frontend/src/components/employees/EmployeeLocationMap.tsx`
- `frontend/src/components/employees/EmployeeLocationMap.test.tsx`
- `frontend/src/utils/checkInGeofenceGuidance.ts` only if the semantics change

If the provider swap changes class names, animation hooks, or dependency state, expand to:

- `frontend/src/styles/base.css`
- `frontend/package.json`
- `frontend/pnpm-lock.yaml`

`frontend/src/components/employees/EmployeeCheckInCard.tsx` should stay untouched unless the map API changes.

### Proposed Test Matrix

| Level | Case | Expected result |
|---|---|---|
| Unit | no target / no position / inside / inaccurate / outside | Guidance status stays identical |
| Unit | boundary equality | `distance + accuracy == radius` is inside |
| Unit | outside instruction | Uses nearest gate and formatted distance |
| Component | sample absent | Falls back to first gate and no route arrow |
| Component | sample present and gate differs | Route line, arrow, nearest gate emphasis render |
| Component | same sample and gate | Arrow is omitted |
| Component | reduced motion | Arrow animation path is skipped or reduced |
| Integration | `EmployeeCheckInCard` disclosure toggles | Lazy-loaded map still mounts correctly |
| Regression | admin wrappers | Attendance and failed-attempt maps still render through `LocationMap` |

### What this research did not cover

- I did not validate a concrete MapLibre implementation, only the current codebase boundaries and safe migration scope.
- I did not inspect backend geofence contracts beyond the frontend-facing type and helper usage.
- I did not run the frontend test suite or visual QA in this pass.

## Cross-Plan Overlap Assessment

- Overlap with `plans/260719-1310-employee-mobile-polish` is real but narrow:
  - that plan already touched `EmployeeCheckInCard.tsx` and the employee attendance UX,
  - it explicitly kept the map disclosure intact and out of scope for provider changes,
  - its remaining risk surface is the same employee attendance card and its tests.
- For this migration, the conflict risk is low if you stay inside `EmployeeLocationMap.tsx`, `EmployeeLocationMap.test.tsx`, and `checkInGeofenceGuidance.ts`.
- Conflict risk becomes high if you also edit `EmployeeCheckInCard.tsx`, because that file is the same employee-attendance state machine boundary used by the completed polish plan.

## Unresolved Questions

- None blocking. The only open decision is whether the migration should preserve the exact 50m accuracy badge styling or intentionally retune it.
