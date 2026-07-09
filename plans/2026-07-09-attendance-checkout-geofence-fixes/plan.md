# Fix attendance check-out failure issues (demo root-cause)

**Status:** implemented (pending code review)
**Date:** 2026-07-09
**Branch:** main
**Source investigation:** `root@demo.tingting.vip` — "Lần chấm thất bại — Check-out 2026-07" (24 records; all 7 `gps_inaccurate` rows belong to employee #918 Bùi Nguyễn Duy Anh)

## Results
- **Phase 1 (geofence):** done. `go test ./internal/app/services/attendance/` — all 9 tests pass (6 existing + 2 new + classifier).
- **Phase 2 (dedup):** done. `go build ./...` + `go vet` clean; handler + admin handler tests pass. `ExistsRecent` override added to test fake.
- **Phase 3 (frontend cooldown):** done. `tsc -p tsconfig.json --noEmit` clean (exit 0).
- **Phase 4 (timezone):** **no bug — dropped.** DSN uses `loc=Local` (GORM reads the +07-naive `datetime` as `+07:00`); frontend renders via `parseISO` + `date-fns format` in the viewer tz. The observed 1h offset is the investigator's own GMT+8 browser viewing a GMT+7 business event — data is correct and consistent across failed-attempts and attendance tables. No code change.
- **Pre-existing failures (not mine):** `TestCreateWithBudgetCheck_*` in `advance_payment_request_repository_test.go` fail with my changes stashed too (confirmed). Unrelated.
- **Out of scope:** 07-08 attendance #28 (employee never checked out → auto-rejected → 0 pay) is a payroll/admin-approval decision, not a code fix.
- **Note:** the working tree also contains unrelated in-flight `wallet_demand_forecast*` / `config.go` / wallet-frontend changes that are NOT part of this task; any commit must stage only the 7 files below.

## Confirmed root causes

1. **Edge-case geofence rejection.** Employee 50.08 m from a 150 m gate rejected because `dist(50.08) + accuracy(100) = 150.08 > 150` → `insideButUncertain` → `gps_inaccurate`. They are deep inside the zone but blocked by a 0.08 m GPS-uncertainty overshoot. (`attendance_service.go:299` `validateGeofence`)
2. **Misleading failure reason.** The accuracy guard `accuracy > radius` fires *before* the distance loop, so a reading 5.9 km away (±200 m, radius 150 m) is reported as "GPS không chính xác" instead of "outside geofence".
3. **No failed-attempt de-dup.** 6 byte-identical rows created over 42 s of patient re-taps (the in-flight double-click guard does not cover re-taps). `recordFailedAttempt` inserts on every call.

## Out of scope (operational, not code)

- **07-08 attendance #28**: employee #918 checked in 07:58, never checked out → auto-rejected at 21:00 → earning 0. Restoring pay is a payroll decision (admin `Approve`); not automated and not part of this change.

## Design note (geofence relaxation is constrained by existing tests)

Existing tests pin two rejections that any relaxation must preserve:
- `TestValidateGeofenceRejectsPoorAccuracyEvenOnGate`: 50 m, ±800 m, 100 m radius → reject.
- `TestValidateGeofenceRejectsWhenUncertaintyCrossesRadius`: 80 m, ±30 m, 100 m radius → reject.

These rule out "trust point estimate whenever `dist≤radius ∧ acc≤radius`" (passes the 80 m/±30 m boundary case) and rule out pure inner-half leniency (passes the 50 m/±800 m anti-spoof case). The combined rule below is the minimal logic that fixes the BA case while keeping both tests red→green.

## Phases

### Phase 1 — Geofence 3-way reclassification + inner-half relaxation (backend)
File: `internal/app/services/attendance/attendance_service.go` (`validateGeofence`, ~line 299).

Per gate, classify the reading:
- **pass** if `dist + accuracy ≤ radius` (worst-case inside) **OR** `accuracy > 0 ∧ accuracy ≤ radius ∧ dist ≤ radius/2` (point estimate comfortably inside with zone-scale GPS).
- **definitelyOutside** if `dist - accuracy > radius` (even best case is outside).
- else **straddles** (uncertainty circle crosses the boundary, or accuracy so large it is untrustworthy).

Aggregate: any pass → return gate name; else any straddles → `gps_inaccurate`; else → `geofence_outside`.

Removes the standalone `accuracy > radius` short-circuit (subsumed: near + bad accuracy → straddles → `gps_inaccurate`; far + bad accuracy → `definitelyOutside` → `geofence_outside`). `accuracy == 0` legacy on-gate still passes via `dist + 0 ≤ radius`. User-facing message strings unchanged.

Tests (`geofence_test.go`): keep all 6 existing tests green; add:
- 50 m / ±100 m / 150 m radius → **PASS** (BA case).
- 5869 m / ±200 m / 150 m radius → **geofence_outside** (mislabel fix).
- 50 m / ±800 m / 100 m radius → **gps_inaccurate** (anti-spoof preserved — already exists, keep).

### Phase 2 — Failed-attempt de-dup (backend)
Files: `domain/attendance.go`, `infra/persistence/attendance_failed_attempt_repository.go`, `transport/http/handlers/attendance/attendance.go`.

Add repo method `ExistsRecent(ctx, employeeID, attemptType, reasonCategory, projectID, within time.Duration) (bool, error)` — true if a row with the same `(employee_id, attempt_type, reason_category, project_id)` exists within `within` of now. In `recordFailedAttempt`'s goroutine, skip `Create` when `ExistsRecent(..., 60s)` is true. Net effect: the 42 s retry cluster collapses to one record; the earlier 5.9 km record (different time window) is retained.

### Phase 3 — Frontend post-failure cooldown (frontend)
File: `frontend/src/components/employees/EmployeeCheckInCard.tsx`.

After a failed check-out whose reason is GPS/geofence, disable the "Tan ca" button for a short cooldown (≈10 s) with an inline hint, spacing retries so GPS can improve. Reuse existing `isPending` plus a cooldown-timestamp state. Lower priority — Phase 2 already fixes the dashboard count.

### Phase 4 — Timezone verification (verify-only)
Confirm whether the failed-attempts view and attendance timestamps render with an offset. DB evidence shows `attendance_failed_attempts.created_at` and `attendances.check_in_time/check_out_time` are both stored +07-naive (GORM sets `created_at` from the +07 backend container; MySQL `DEFAULT CURRENT_TIMESTAMP` is never used). Inspect `dto/dashboard.go` `AdminFailedAttempt.created_at` serialization + the frontend formatter. **Fix only if a concrete rendering mismatch is found**; otherwise drop with rationale.

## Acceptance criteria
- `go test ./internal/app/services/attendance/...` passes (existing 6 + new geofence tests).
- `go build ./...` and `go vet ./...` clean.
- Dedup collapses rapid identical failures to one row (unit-testable).
- Frontend builds (`yarn build` / tsc) if Phase 3 included.
- No public-contract change: error message wording unchanged; only the `gps_inaccurate` vs `geofence_outside` categorization shifts for far + inaccurate reads. No DB migration.

## Risks / rollback
- Edge anti-spoof slightly relaxed: an attacker within ~radius with `accuracy ≤ radius` could pass. Accepted tradeoff; the ±800 m anti-spoof test still holds.
- Code-level only, no migration, no schema change. Rollback = revert.
