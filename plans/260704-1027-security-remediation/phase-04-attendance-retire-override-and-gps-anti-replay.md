---
phase: 4
title: "Attendance retire override and GPS anti-replay"
status: pending
priority: P2
dependencies: []
---

# Phase 4: Attendance retire override and GPS anti-replay

## Overview
Two attendance items. **M1:** retire the now-orphaned admin override route — UI button already removed, but the backend route + `RecordAdminCheckIn` are still live and bypass geofence/window/`CheckInEnabled`. **H3:** add server-side anti-replay to check-in — the accuracy/checkout gates (`a3f892c`/`bd2b9d1`) are deployed (verified, see plan.md Baseline), but reported coords are still trusted with no `gps_at` freshness check, so a plausible inside-gate coord from elsewhere still earns a shift.

## Requirements
- Functional: the override route is gone (no bypass); a check-in with `gps_at` outside a tight server-clock window is rejected/flagged; repeated `accuracy < 5m` readings are flagged.
- Non-functional: legitimate field check-ins (real device clock skew, poor GPS) still pass; anti-replay is defense-in-depth, not a fraud silver-bullet.

## Architecture
- **M1 (retire):** delete the route registration, the handler block (`admin/attendance.go` ~141-222), and `attendanceService.RecordAdminCheckIn`. Frontend button already removed — confirm no remaining callers.
- **H3 (anti-replay):** in the check-in path (`attendance_service.go` 294-320 + handler `attendance.go` 153-212), cross-check `gps_at` against `clock.Now()` within ±60s; flag/reject if outside. Log repeated sub-5m-accuracy readings per employee as suspicious. Long-term: sign the check-in payload + device attestation (documented as follow-up, out of scope here).

## Related Code Files
- Delete/modify: `backend/internal/transport/http/handlers/admin/attendance.go` (M1 — override route + handler 141-222)
- Delete/modify: `backend/internal/app/services/attendance/attendance_service.go` (M1 — `RecordAdminCheckIn`)
- Delete: route registration for `/admin/attendances/failed-attempts/:id/override`
- Modify: `attendance_service.go` 294-320 + `attendance.go` 153-212 (H3 — `gps_at` freshness, accuracy-repeat flag)
- Reference: `feature_attendance_checkout_auto_reject` (auto-reject must still cover failed check-outs once the manual override is gone); `lesson_timezone_loc_local_prod_utc` (use `clock.Now()`, not `time.Now()`)
- Tests: override route → 404 (M1); check-in with stale/future `gps_at` rejected (H3)

## Implementation Steps
1. **M1 — confirm no callers.** Grep `RecordAdminCheckIn` + the override route across backend + frontend; confirm the only caller was the removed button.
2. **M1 — delete.** Remove route registration, handler method, and `RecordAdminCheckIn` service method. Remove override-path tests.
3. **M1 — auto-reject sanity.** Confirm the auto-reject path (`feature_attendance_checkout_auto_reject`) still covers failed check-outs now that the manual override is gone.
4. **H3 — `gps_at` freshness.** In check-in validation, compare reported `gps_at` to `clock.Now()` (Asia/Ho_Chi_Minh); reject if `|delta| > 60s`. Threshold env-configurable.
5. **H3 — accuracy-repeat flag.** Track recent `accuracy < 5m` readings per employee; alert on repetition (flag, don't auto-reject — false-positive risk).
6. **Tests.** Override route → 404 (M1); stale `gps_at` → rejected (H3); fresh `gps_at` → accepted.

## Success Criteria
- [ ] M1: `POST /admin/attendances/failed-attempts/:id/override` → 404; `RecordAdminCheckIn` has no callers; `grep` clean.
- [ ] H3: a check-in with `gps_at` > 60s from server clock is rejected/flagged; legitimate skew passes.
- [ ] Auto-reject path still functions without the manual override.

## Risk Assessment
- **M1 removes an in-use escape hatch:** if operations relied on the override for legitimate device-failure cases. *Mitigation:* user decision was explicit (retire); the auto-reject + re-check-in flow is the supported path. Document the runbook change for admins.
- **H3 ±60s too tight:** real device clock drift > 60s would reject legit check-ins. *Mitigation:* env-configurable threshold; start at 60s with a rejection-rate monitor; prefer flag-over-reject for borderline cases.
- **H3 doesn't defeat determined spoofing:** a device with NTP-synced clock + plausibly-spoofed coords still passes. *Mitigation:* documented as defense-in-depth; payload signing + device attestation is the long-term follow-up.
