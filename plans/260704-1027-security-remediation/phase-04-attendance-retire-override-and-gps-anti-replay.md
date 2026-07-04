---
phase: 4
title: "Attendance retire override and GPS anti-replay"
status: pending
priority: P2
dependencies: []
---

# Phase 4: Attendance retire override and GPS anti-replay

## Overview
**M1 (retire):** delete the orphaned admin override route — the UI button is gone but the backend route + handler + `RecordAdminCheckIn` AND three frontend call sites are still live. **H3 (reshaped):** the v1 "±60s on `gps_at`" check was **theater** — `gps_at` is a client-supplied field, so a spoofer sends a fresh timestamp. This phase now does honest input-sanity + a server-side replay guard, and documents real spoofing-defeat (device attestation) as a tracked follow-up. See `## Red Team Corrections`.

## Requirements
- Functional: the override route + all callers are gone; a captured/stale check-in payload can't be replayed; obviously-stale device fixes are rejected.
- Non-functional: legitimate field check-ins (poor GPS, normal clock skew) still pass; real spoofing-defeat is explicitly out of scope (follow-up).

## Architecture
- **M1 (corrected — incl. frontend):** the button is gone from `pages/`/`components/`, but the service/hook/config layer still ships three live callers: `frontend/src/config/api.config.ts:362`, `frontend/src/services/api/dashboard.service.ts:312-314` (`overrideFailedAttempt`), `frontend/src/hooks/api/useDashboard.ts:128-145` (`useOverrideFailedAttempt`). All three must be deleted alongside the backend route + handler (`admin/attendance.go:141-222`) + `RecordAdminCheckIn`, or a future cook re-imports the hook and re-opens the bypass.
- **H3 (reshaped — honest, not theater):**
  - **Replay guard (server-side, real):** persist per-employee shift-window check-in state and reject a duplicate submission (extend the existing per-employee/day idempotency that `RecordAdminCheckIn` already relies on). This stops replaying a captured payload.
  - **Input-sanity (demoted, honest label):** at the **handler layer** (which has `req.GpsAt` at `attendance.go:101`; `validateGeofence` at `attendance_service.go:294` does NOT receive `gps_at` — do NOT change its signature), reject device fixes older than an env-configurable threshold (e.g. 300s) from `clock.Now()`. This catches stale cached/captured fixes; it is **not** anti-replay (a live spoofer sends `now`). Label it accordingly.
  - **`clock.Now()` not `time.Now()`** (`lesson_timezone_loc_local_prod_utc`) — prod scratch runs UTC while DSN forces `time.Local` to Asia/Ho_Chi_Minh.
  - **Spoofing-defeat (follow-up, NOT this plan):** device attestation + payload signing. Tracked in `plan.md` Out of scope.
  - **accuracy-repeat flagging** stays (forensic).

## Related Code Files
- Delete: backend override route registration + `backend/internal/transport/http/handlers/admin/attendance.go:141-222` + `RecordAdminCheckIn` (M1)
- Delete: `frontend/src/config/api.config.ts:362`, `frontend/src/services/api/dashboard.service.ts:312-314`, `frontend/src/hooks/api/useDashboard.ts:128-145` (M1)
- Modify: `backend/internal/transport/http/handlers/attendance/attendance.go` ~101 (H3 — input-sanity at handler; replay guard before `CheckIn`)
- Reference: `feature_attendance_checkout_auto_reject` (auto-reject must still cover failed check-outs once the manual override is gone); `lesson_timezone_loc_local_prod_utc`
- Tests: override route → 404 + frontend type-check clean (M1); stale device fix → rejected; replayed payload → rejected (H3)

## Implementation Steps
1. **M1 — confirm no other callers.** Grep backend + frontend for `RecordAdminCheckIn`, `override`, `failed-attempts/:id/override`, `useOverrideFailedAttempt`.
2. **M1 — delete backend.** Route registration + handler (`admin/attendance.go:141-222`) + `RecordAdminCheckIn` + override-path tests.
3. **M1 — delete frontend.** Remove the three call sites; run `pnpm type-check`.
4. **M1 — auto-reject sanity.** Confirm `feature_attendance_checkout_auto_reject` still covers failed check-outs without the manual override.
5. **H3 — replay guard.** Extend per-employee shift-window idempotency at the handler to reject duplicate check-in submissions.
6. **H3 — input-sanity.** Handler-layer check: reject `gps_at` older than threshold from `clock.Now()`. Env-configurable. Flag (don't hard-reject) borderline cases.
7. **H3 — accuracy flag.** Track recent `accuracy < 5m` readings per employee; alert on repetition.
8. **Tests.** Override route → 404; frontend type-check (M1); stale fix → rejected; replayed payload → rejected (H3).

## Success Criteria
- [ ] M1: `POST /admin/attendances/failed-attempts/:id/override` → 404; `RecordAdminCheckIn` + `useOverrideFailedAttempt` have no callers; `pnpm type-check` clean.
- [ ] H3: a replayed check-in payload is rejected (server-side guard); a stale device fix is rejected; legit skew passes.
- [ ] Auto-reject path still functions without the manual override.

## Risk Assessment
- **M1 removes an operational escape hatch:** user decision (retire) is explicit; auto-reject + re-check-in is the supported path. *Mitigation:* runbook note for admins.
- **H3 ±300s too tight:** real drift. *Mitigation:* env-configurable; flag-over-reject for borderline; rejection-rate monitor.
- **H3 honesty:** this phase does NOT defeat a live spoofer. *Mitigation:* explicitly documented; device attestation is the tracked follow-up.

## Red Team Corrections
- **H3 ±60s was theater (Critical):** `gps_at` is client-controlled (`dto/attendance.go:14`); a spoofer sends a fresh timestamp. Reshaped to (a) server-side replay guard (real), (b) handler-layer input-sanity honestly labeled, (c) `validateGeofence` signature left unchanged, (d) device attestation tracked as follow-up.
- **M1 frontend not removed (High):** three live callers found (`api.config.ts:362`, `dashboard.service.ts:312`, `useDashboard.ts:131`); v1 listed zero frontend files. Now in scope + `pnpm type-check`.
