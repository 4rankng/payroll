---
phase: 2
title: React integration
status: completed
effort: M
dependencies:
  - 1
---

# Phase 2: React integration

## Overview

Introduce a `useContinuousLocation` hook that owns the Phase-1 primitive's lifecycle (start on mount when actionable, stop on unmount/`visibilitychange hidden`) and exposes the rolling best-sample cache + an `isSubmitReady` flag. Rewire `EmployeeCheckInCard.tsx` to consume it: delete the 12s preview `useEffect` (L203–240) and the cold-start inside `handleAction` (L264); on tap, submit the cached warm fix instantly when `getCheckInGeofenceGuidance(target, sample).status === "inside"`, else wait for `isSubmitReady` with a 30s hard cap. Both check-in and check-out share the hook.

## Requirements

- **Functional**
  - `useContinuousLocation({ target, radius, actionType, enabled })` returns `{ bestFreshSample, progress, isAcquiring, isSubmitReady, error }`.
  - `isSubmitReady === true` iff a fresh cached sample exists whose `getCheckInGeofenceGuidance(target, sample).status === "inside"` (mirrors backend `validateGeofence` certain-path).
  - Watch starts when `enabled` (card mounted + actionable state, mirroring today's `canPreviewCheckLocation`), stops on unmount and on `document.visibilityState === "hidden"`, resumes on `visible`.
  - On `handleAction("check_in"|"check_out")`:
    - If `isSubmitReady` → submit immediately from `bestFreshSample` (no spinner).
    - Else → flip `isLocating`, show the existing converging-accuracy UX, submit the moment `isSubmitReady` flips true, hard-cap 30s then surface the poor-accuracy recovery panel.
  - Payload to the backend **unchanged**: `{lat, lng, accuracy, gps_at, gps_sample_count, gps_best_accuracy, gps_elapsed_ms}` (+ `confirm_no_salary` for checkout).
  - Map preview reads `bestFreshSample` (replaces `visibleLocationSample = locationProgress?.bestFreshSample ?? lastFreshSample`).
  - Keep the `submittingRef` double-tap guard (L182) — still needed because React render lag.
- **Non-functional**
  - No backend contract change. No migration. Gates frozen.
  - Battery: watch runs only while card is visible + actionable.

## Architecture

```
EmployeeCheckInCard
  └─ useContinuousLocation({ target, radius, actionType, enabled })
        └─ watchContinuousLocation()  (Phase 1 primitive)
              └─ rolling cache ──► getCheckInGeofenceGuidance(target, sample) ──► isSubmitReady
                                                                              │
        onAction: ── if isSubmitReady ──► submit(sample)  (instant)
                   ── else ────────────► wait for isSubmitReady | 30s cap ─► recovery panel
```

- The hook computes `isSubmitReady` on every cache update using the **existing** `getCheckInGeofenceGuidance` — no new geofence math, no duplication of the backend gate.
- `actionType` is informational only (the watch behavior is identical for check-in/out); kept in the API so future per-action tuning has a seam without a redesign (YAGNI boundary: do not branch on it now).

## Related Code Files

- **Create:** `frontend/src/hooks/useContinuousLocation.ts`.
- **Modify:** `frontend/src/components/employees/EmployeeCheckInCard.tsx` — delete L203–240 preview effect; rework `handleAction` (L242–365) to read from the hook; both action branches share it; map preview reads `bestFreshSample`.
- **Read-only reuse:** `frontend/src/utils/geolocation.ts` (Phase 1 primitive), `frontend/src/utils/checkInGeofenceGuidance.ts` (guidance oracle), `frontend/src/components/employees/EmployeeLocationMap.tsx` (no change — receives a fresher `sample`).

## Implementation Steps

1. Create `useContinuousLocation.ts`: instantiate `watchContinuousLocation` in a `useRef`, start/stop via `useEffect` keyed on `enabled` + `document.visibilityState`, expose `{ bestFreshSample, progress, isAcquiring, isSubmitReady, error }`. Compute `isSubmitReady` with `getCheckInGeofenceGuidance` in a `useMemo` over `bestFreshSample + target`.
2. In `EmployeeCheckInCard.tsx`, replace the preview `useEffect` (L203–240) with `const location = useContinuousLocation({ target: checkInTarget, radius: checkInTarget?.radius_meters, actionType, enabled: canPreviewCheckLocation })`.
3. Rework `handleAction`:
   - Drop `setLastFreshSample(null)` (L249) and the local `requestBestCurrentLocation` call (L264).
   - If `location.isSubmitReady && location.bestFreshSample` → build payload from the cached sample, `await mutateAsync(payload)` immediately.
   - Else → `setIsLocating(true)`, `await` a promise that resolves when `location.isSubmitReady` becomes true or a 30s `setTimeout` rejects (poor-accuracy path). Use the existing `locationProgress` UI for convergence feedback.
   - Preserve all existing error classification (`isPoorLocationAccuracyMessage`, `isGeofenceOutsideMessage`, `canConfirmNoSalaryCheckout`, `getLocationPermissionIssue`) — only the *acquisition source* changes, not the error UX.
4. Map preview: set `visibleLocationSample = location.bestFreshSample`.
5. Verify the `submittingRef` guard still wraps the new instant-submit branch.
6. Run `pnpm lint`.

## Success Criteria

- [ ] `useContinuousLocation` hook created with the documented return shape and lifecycle (start on `enabled`, stop on unmount/hidden).
- [ ] 12s preview `useEffect` and the in-`handleAction` `requestBestCurrentLocation` call both deleted (grep clean).
- [ ] Warm tap (cached inside-sample present) submits with no spinner; cold tap shows converging UX then submits on `isSubmitReady` or 30s cap.
- [ ] Both check-in and check-out share the hook; `confirm_no_salary` checkout flow intact.
- [ ] Backend payload shape unchanged.
- [ ] `pnpm lint` clean.

## Risk Assessment

- **Stale cached fix submitted after worker walks away from the gate** → freshness window (15s) + `getCheckInGeofenceGuidance` re-check at tap time + server gate backstop. If the worker moved, the cache's "moved position" rule (Phase 1) produces a current sample; if that sample is outside, `isSubmitReady` is false and we keep acquiring.
- **`isSubmitReady` flapping** (accuracy oscillating around the bar) → accept the first `true` edge during an in-flight action (resolve-once). Do not submit twice.
- **iOS PWA suspension** → watch stops on suspend regardless; on resume, `visibilitychange visible` re-warms. Acceptable.
- **Double-tap race on the instant path** → `submittingRef` already covers it; explicitly verify the new branch is inside the guard.
- **Forgetting an error path** → the recovery panel + toast + admin-failed-attempt log paths must all still fire; only the acquisition source changes. Re-walk every `setLocationIssue` call site.
