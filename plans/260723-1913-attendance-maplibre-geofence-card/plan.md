---
title: "Attendance MapLibre Geofence Card"
description: "Migrate the employee attendance geofence card from Leaflet rendering to react-map-gl with MapLibre while preserving geofence authority and mobile status behavior."
status: pending
priority: P2
branch: "main"
tags:
  - feature
  - frontend
  - employee
  - maplibre
  - geofence
blockedBy: []
blocks: []
created: "2026-07-23T11:27:20.766Z"
createdBy: "ck:plan"
source: skill
---

# Attendance MapLibre Geofence Card

## Overview

Migrate only the employee attendance/geofence map card to `react-map-gl` with MapLibre GL. The current screenshot problem is a viewport and status-presentation issue: the card should keep the office/geofence legible, show the employee position when useful, and avoid drawing a continent-scale route when the employee is far outside the permitted radius.

The implementation must keep backend/API geofence authority unchanged. `frontend/src/utils/checkInGeofenceGuidance.ts` remains advisory/status-only UI guidance for `inside`, `outside`, `inaccurate`, `no_position`, and `no_target`; it must not become the client-side authorization gate for check-in submission.

## Scope

- In: employee-only map rendering, MapLibre runtime setup, map view-model helpers, geofence polygon rendering, markers, near-route display, far-distance outside indicator, WebGL/style failure fallback, component tests, and mobile visual validation.
- Out: admin map migration, full Leaflet removal, backend geofence changes, background location, Google Maps/Mapbox-token dependency, new check-in API fields, and eager map loading before the employee opens the disclosure.
- Preserve: Vietnamese copy, `EmployeeCheckInCard` lazy disclosure, existing geofence guidance statuses, radius-based classification, GPS accuracy display, and admin Leaflet maps.

## Key Decisions

- Add `react-map-gl` and `maplibre-gl` with `pnpm` in `frontend/`; do not use `npm install`. Versions verified on 2026-07-23 were `react-map-gl@8.1.1` and `maplibre-gl@6.0.0`; implementation should pin those or rerun package metadata checks and record the resolved versions before install.
- Import MapLibre through `react-map-gl/maplibre`, not the Mapbox entrypoint.
- Keep `leaflet`, `react-leaflet`, and `@types/leaflet` because `frontend/src/components/admin-dashboard/LocationMap.tsx` and its wrappers still use Leaflet.
- Render geofence radius as GeoJSON polygon approximations, not a visual-only pixel circle, so the 150 m radius stays meter-based.
- Use fit-bounds only for coordinates that should be compared visually. For very far samples, fit the office/geofence and show the employee as an outside-radius status instead of drawing a misleading long route.
- Define "near enough for route display" as `distanceMeters <= max(radiusMeters * 4, 500)`. This cutoff controls only route/viewport display and must not change guidance status, check-in eligibility, or backend submission behavior.
- Keep required basemap attribution visible or otherwise compliant. Hide navigation controls, not attribution required by the basemap provider.
- Before choosing a live style URL, make an explicit provider/privacy decision: allowed style/tile hostnames, attribution terms, whether third-party disclosure of worksite tile coordinates is acceptable, CSP implications, and the fallback if a self-hosted/proxied provider is required.
- Show the exact employee marker only to the logged-in employee after they open the disclosure. Do not render coordinate text, log map viewport/sample coordinates, or pan/load remote tiles around a far outside sample unless that privacy decision explicitly permits it.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Map Runtime and View Model](./phase-01-map-runtime-and-view-model.md) | Pending |
| 2 | [MapLibre Geofence Card](./phase-02-maplibre-geofence-card.md) | Pending |
| 3 | [Regression and Mobile Validation](./phase-03-regression-and-mobile-validation.md) | Pending |

## Dependencies

- No blocking cross-plan dependency.
- Completed plan `plans/260719-1310-employee-mobile-polish/plan.md` is a non-regression boundary: keep the employee map disclosure intact and avoid changing `EmployeeCheckInCard.tsx` unless implementation proves it necessary.
- Phase 2 depends on Phase 1. Phase 3 depends on Phases 1 and 2.

## Acceptance Criteria

- The employee attendance map uses `react-map-gl/maplibre` and `maplibre-gl` while admin attendance maps continue to render through Leaflet.
- Nearby employee and gate coordinates are fit tightly, with the geofence radius visible and no Southeast Asia-scale viewport for local check-in cases.
- Far outside-radius samples do not render as a long dashed navigation route; the card communicates outside status and distance compactly.
- WebGL/style load failures show a readable Vietnamese fallback and do not block the rest of the check-in card.
- The selected basemap provider, attribution display, and network hostnames are verified before release; no Mapbox, Google, or unapproved map hosts are contacted.
- Focused employee map tests, geofence guidance tests, lint/type-check/build, employee mobile visual checks, `make api-test`, and `graphify update .` are accounted for in implementation verification.

## Research

- [Codebase migration assessment](./research/researcher-codebase-migration.md)
- [MapLibre platform notes](./research/researcher-maplibre-platform.md)

## Red Team Review

- Status: completed 2026-07-23.
- Reports:
  - [Security Adversary Review](./reports/security-adversary-review.md)
  - [Assumption Destroyer Review](./reports/assumption-destroyer-review.md)
  - [Failure Mode Review](./reports/failure-mode-review.md)
- Accepted findings:
  - Make remote style/tile privacy and attribution a blocking provider decision.
  - Treat frontend geofence guidance as advisory/status-only while preserving backend authority.
  - Define a deterministic display-only route cutoff and boundary tests.
  - Put geometry/view-model helpers in a `.ts` model file instead of growing `EmployeeLocationMap.tsx`.
  - Compute viewport bounds from geofence polygon extents, not only gate centers.
  - Add real-browser checks for network hosts, attribution, WebGL/style failure, lazy chunks, admin Leaflet smoke, mobile dock overlap, and 200% text.
- Rejected findings: none.

## Whole-Plan Consistency Sweep

- Files reread: `plan.md` and all three phase files.
- Decision deltas checked: employee-only scope, package manager, Leaflet retention, lazy disclosure, advisory frontend guidance, backend authority, far-distance behavior, provider privacy, attribution, fallback, tests, and graph update.
- Unresolved contradictions: 0.
