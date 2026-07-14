# Attendance Map Direction Guidance

Status: pending

## Goal

Make the existing employee checkpoint map immediately useful when GPS shows the worker inside the radius but not certainly inside the geofence: open it once for that guidance episode and show a clean straight-line direction toward the nearest gate.

## Phase

1. [Directional map and auto-open lifecycle](./phase-01-directional-map.md) — **pending**

## Dependencies

- Builds on the uncommitted boundary-guidance work in `EmployeeCheckInCard`, `useContinuousLocation`, and `checkInGeofenceGuidance`; those changes must be preserved.
- Reuses React Leaflet, the existing nearest-gate calculation, employee typography/tokens, and reduced-motion rules; no new package is required.
- Backend geofence validation remains authoritative and unchanged.

## Acceptance Criteria

- The `124 m / 150 m / GPS ±48 m` inside-but-uncertain state opens the map once when that state begins.
- Manually closing the map keeps it closed while the same guidance episode remains active; leaving and later re-entering that state may auto-open it again.
- The map shows a sky-blue dashed straight line with a visible arrowhead pointing into the emerald nearest gate.
- A clean overlay reads `Hướng tới <cổng> · <khoảng cách>` and `Đường thẳng tham khảo — không phải lộ trình đường bộ`.
- GPS updates do not continuously reset a worker's manual pan or zoom.
- Poor GPS, outside-geofence, manual map disclosure, and nearest-gate selection retain their existing behavior.
- Motion is one-shot only and disabled under `prefers-reduced-motion`; no looping animation is introduced.
- No backend, API, geofence policy, road routing, or dependency change.

## Completion Gates

- Focused component and geofence-guidance tests cover the acceptance cases.
- Full frontend Vitest suite, lint, type-check, and production build pass.
- `make api-test` passes when the required local backend is available, and `graphify update .` refreshes the project graph after implementation.

