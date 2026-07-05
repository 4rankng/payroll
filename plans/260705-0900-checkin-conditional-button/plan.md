---
title: "Employee check-in: conditional button (GPS + timing) with micro-animations"
description: "The employee check-in button currently shows unconditionally (only disabled during a request). This plan makes it show only when GPS is stable + inside geofence + within the shift check-in window, with micro-animations (GPS-warming pulse, became-ready pop, outside-window hint) to guide the user. Backend exposes the resolved shift window in the employee profile DTO; the frontend derives readiness from GPS state (already available via useContinuousLocation) + the new timing signal + geofence guidance (already available). No new animation library — uses existing tailwindcss-animate + custom keyframes."
status: pending
priority: P2
branch: "main"
tags: [employee, check-in, gps, ux, animation, mobile]
blockedBy: []
blocks: []
created: "2026-07-05T02:33:48.025Z"
createdBy: "ck:plan"
source: skill
---

# Employee check-in: conditional button (GPS + timing) with micro-animations

## Overview

Today the employee check-in button (`EmployeeCheckInCard.tsx:732-747`) is always
visible — it only gets disabled while a check-in request is in flight. All real
gating (geofence, accuracy, shift window ±1h, GPS freshness, per-day idempotency)
happens server-side in `AttendanceService.CheckIn` and the frontend only reacts
to the rejection *after* the tap.

This plan moves GPS + timing readiness to the **frontend**, so the button itself
reflects whether the employee *can* check in right now:

- **GPS state** is already available: `useContinuousLocation` exposes
  `progress.status` (warming/excellent/acceptable/weak) + `isSubmitReady`
  (sample present + geofence inside). Just needs wiring to the button.
- **Timing state** is NOT available — the ±1h check-in window is computed
  server-side from payrate shift keys and never reaches the employee profile.
  Phase 1 adds it to the profile DTO.

Micro-animations (no new library — `tailwindcss-animate` + existing keyframes):
- **GPS-warming pulse**: button breathes gently while GPS is acquiring.
- **Became-ready pop**: when GPS stabilizes + inside geofence + within window,
  button pops green ("ready to check in").
- **Outside-window hint**: when outside the shift window, button is hidden /
  replaced with "Ca làm việc bắt đầu lúc HH:MM" + countdown.

## Phases

| Phase | Name | Effort | Ships independently? |
|-------|------|--------|----------------------|
| 1 | [Backend shift-window DTO](./phase-01-backend-shift-window-dto.md) | S | ✅ (additive field, no breaking change) |
| 2 | [Frontend readiness gate + animations](./phase-02-frontend-readiness-gate-animations.md) | M | Depends on Phase 1's DTO field |

## Critical path

```
1 (backend DTO) ──▶ 2 (frontend gate + animations)
```

Phase 2 depends on Phase 1's new field. Phase 1 is purely additive (new
optional fields on the profile response) and ships safely on its own.

## Key technical decisions (from audit + owner)

- **Backend exposes the resolved shift window** in `EmployeeProfileResponse` —
  the shift start/end times and the valid check-in window bounds. The frontend
  derives "is now within the window" client-side using the device clock.
  Advisory only — the server's `validateCheckInWindow` stays authoritative.
- **GPS state** reuses `useContinuousLocation` (already wired into the card) —
  no new hook needed.
- **Geofence state** reuses `getCheckInGeofenceGuidance` (already available).
- **Animations**: `tailwindcss-animate` + existing keyframes (`pulse-dot`,
  `employee-pay-breathe`, `mobile-stat-pop`). No framer-motion / react-spring.
- **Mobile-first**: the employee portal (`FlexiblePayEmployeePage`) is inherently
  mobile; no separate mobile folder.

## Out of scope

- Changing the backend check-in validation logic (`validateCheckInWindow` stays).
- Checkout button gating (Phase 2 focuses on check-in; checkout is analogous
  and can follow the same pattern later).
- New GPS acquisition logic (the continuous warm-up from commit `00e71d4` stays).
- Admin override UI (commit `d3e2638` stays).
