---
title: Seamless worker check-in/out via continuous GPS warm-up
description: >-
  Unify the 12s preview + 30s cold-submit GPS acquisitions in
  EmployeeCheckInCard into one continuous watchPosition with a rolling
  best-sample cache, so a worker who opened the app at the gate taps and submits
  instantly from the warm fix. Pure frontend; geofence/accuracy/checkout/shift
  gates frozen.
status: completed
priority: P2
branch: main
tags:
  - attendance
  - frontend
  - gps
  - pwa
blockedBy: []
blocks: []
created: '2026-07-04T02:34:18.000Z'
createdBy: 'ck:plan'
source: skill
---

# Seamless worker check-in/out via continuous GPS warm-up

## Overview

Workers standing at the project gate in open sky get check-in rejected because the phone's first GNSS fix takes 22–28s and the frontend's acquisition layer gives up first. Production evidence (`attendance_failed_attempts` rows with null lat/lng/accuracy, `gps_timeout`/`gps_unavailable`) shows the dominant failure is **pure acquisition timeout** — the request never reaches the geofence.

The card today runs **two disjoint acquisitions**: an always-on **12s** preview (`EmployeeCheckInCard.tsx` L203–240) that silently dies on cold fixes, and a **30s** submit path (L264) that clears the preview's sample on tap (L249) and **cold-starts from zero**. Part A (already in the codebase: 30s budget, no inner `watchPosition` timeout, `maximumAge: 15s`, preview-sample reuse) raised the budgets but only helps when the preview *succeeded* — at a 12s budget it usually doesn't on a cold phone. **The 12s preview budget is the seam.**

This plan unifies them into **one continuous `watchPosition`** with a rolling best-sample cache, running while the card is mounted and actionable. On tap: submit the cached warm fix instantly if it clears the existing `getCheckInGeofenceGuidance` "inside" check (which already mirrors the backend `validateGeofence` certain-path: `dist + accuracy ≤ radius`); otherwise keep acquiring to a 30s hard cap.

**Constraint (hard):** geofence, accuracy gate, checkout geofence, and shift windows are frozen — frontend + backend. This plan does not touch any gate. It only changes *which fix gets submitted and when*, never whether a submitted fix should pass.

**Brainstorm report:** `plans/reports/2026-07-04-brainstorm-checkin-gps-reliability.md`

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Geolocation primitive](./phase-01-geolocation-primitive.md) | Completed |
| 2 | [React integration](./phase-02-react-integration.md) | Completed |
| 3 | [Verify](./phase-03-verify.md) | Completed |

## Dependencies

- **Cross-plan:** none. The two other unfinished plans (`260703-2023-attendance-open-row-unique`, `260704-1027-security-remediation`) are backend / security — zero file overlap.
- **Pre-condition for production effect (not a code dependency):** Part A must be deployed. It is already merged into `geolocation.ts` (verified: `timeoutMs: 30000`, no inner `watchPosition` timeout, `maximumAge: 15000`). Deploy is an ops task outside this plan.
- **Floor:** current `geolocation.ts` + `EmployeeCheckInCard.tsx` on `main`.

## Success Criteria (whole plan)

- Worker opens app at gate (open sky), waits a few seconds, taps **Vào làm** → submits in <3s from cached warm fix, no 30s spinner.
- Worker taps immediately on a cold phone → continuous watch keeps running, converging-accuracy UX shown, submits when the directional bar clears, hard cap 30s.
- **Tan ca** (check-out) behaves identically via the same hook.
- **No gate regression:** worker outside the gate still rejected; `accuracy > radius` still rejected (server gates unchanged).
- No `watchPosition` leak (single `watchId`, clean cleanup on unmount/`visibilitychange hidden`).
- `pnpm lint` (includes `tsc --noEmit`) + `pnpm build` clean.

## Out of scope

- Geofence / accuracy / checkout / shift-window logic — frozen.
- Proactive device pre-flight & guided retry (brainstorm Approach 2) — deferred.
- Adaptive budget & stationary clustering (brainstorm Approach 3) — deferred until production data justifies.
- Native wrapper, true background geolocation (iOS PWA can't), offline queue.
- Backend contract changes / migrations.
