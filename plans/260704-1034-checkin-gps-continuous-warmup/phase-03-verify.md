---
phase: 3
title: Verify
status: completed
effort: S
dependencies:
  - 2
---

# Phase 3: Verify

## Overview

Static gates (lint + build) plus a manual smoke matrix that proves the seamless-warm path works AND the frozen gates still reject bad fixes. No automated integration test — the acquisition layer depends on real GNSS + permissions that a headless test can't honestly reproduce; the unit tests in Phase 1 cover the pure logic.

## Requirements

- **Functional:** every smoke scenario below passes on a real phone browser (and desktop Chrome with DevTools location override where useful).
- **Non-functional:** `pnpm lint` and `pnpm build` clean; no new console warnings/errors during acquisition.

## Architecture

Verification only — no code design. Runs against the dev server (`make dev` from repo root, or `pnpm dev` in `frontend/`) pointed at the real backend.

## Related Code Files

- **Read-only:** `frontend/src/utils/geolocation.ts`, `frontend/src/hooks/useContinuousLocation.ts`, `frontend/src/components/employees/EmployeeCheckInCard.tsx`.

## Implementation Steps

1. `cd frontend && pnpm lint` (includes `tsc --noEmit`).
2. `cd frontend && pnpm build` (Vite production build + PWA SW).
3. Start dev env, log in as a flexible employee with `check_in_enabled` and a project that has geofence gates configured.
4. Run the smoke matrix (Success Criteria). Use Chrome DevTools → Sensors → Location to simulate "inside gate" and "outside gate"; use a real phone for the cold-start GNSS timing case (the whole point of this plan).
5. Watch-leak check: in DevTools, run `navigator.geolocation.getCurrentPosition = (...)=>{ throw 'should not be called; we use watchPosition' }` is NOT valid here — instead confirm via the Phase-1 lifecycle unit test and by checking `watchPosition` call count via a spy during the manual run (or trust the unit test). Navigate away from the card and back 3×; confirm no degradation/warning.
6. Confirm no regression in the admin failed-attempt path: deliberately fail (deny permission) once and confirm a row appears in the admin Check-in Health dashboard.

## Success Criteria

- [ ] `pnpm lint` clean.
- [ ] `pnpm build` clean (SW precache entries unchanged in count materially).
- [ ] **Warm tap:** open app at gate, wait ~5–10s for preview to warm, tap **Vào làm** → submits in <3s, no 30s spinner.
- [ ] **Cold tap:** kill GPS/restart browser, tap immediately → converging-accuracy spinner, submits when `inside`, hard cap 30s.
- [ ] **Outside gate:** DevTools location set outside radius → server rejects with "ngoài khu vực chấm công"; recovery banner shows; no submit.
- [ ] **Poor accuracy:** simulate accuracy > radius → server rejects with "Tín hiệu GPS không đủ chính xác"; recovery banner shows.
- [ ] **Checkout parity:** checked_in → tap **Tan ca** → same instant/continuous behavior; `confirm_no_salary` dialog still reachable when checking out outside the window.
- [ ] **Visibility:** background the tab (or lock phone) → watch pauses; foreground → resumes; check-in still works after.
- [ ] **No watch leak:** navigate away and back multiple times; no stacked watches (Phase-1 lifecycle test green + no runtime warnings).
- [ ] **Admin failed-attempt path intact:** denying permission produces a row in the admin Check-in Health dashboard.
- [ ] Gates unchanged: confirm `attendance_service.go` `validateGeofence`, accuracy gate, checkout gate, shift-window code have **zero diff** in this plan's PR.

## Risk Assessment

- **Manual-only verification of GNSS timing** → the core value (cold-fix no longer times out) can only be honestly proven on a real device in open sky. Reproduce the original failure mode (pre-Part-A) mentally and confirm the new path would have submitted. The Phase-1 unit tests lock the cache/freshness logic; the smoke test locks the UX + gate parity.
- **DevTools location can't simulate poor accuracy well** → for the poor-accuracy case, prefer a real device in a known bad-signal spot, or stub the accuracy value via a temporary debug hook if needed (remove before commit).
