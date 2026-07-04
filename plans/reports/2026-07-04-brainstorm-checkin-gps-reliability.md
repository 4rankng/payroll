# Brainstorm — Seamless Worker Check-in/out via GPS Reliability

**Date:** 2026-07-04
**Status:** Converged → ready for `/ck:plan`
**Scope:** Frontend-only (`geolocation.ts`, `EmployeeCheckInCard.tsx`)
**Constraint:** Geofence + accuracy + checkout + shift-window gates are HARD constraints — untouched.

---

## Problem statement

Workers standing at the project gate in open sky get their check-in rejected because the phone's first GNSS fix takes 22–28s and the frontend gave up before that. Production evidence (`attendance_failed_attempts` rows with null lat/lng/accuracy, gps_timeout/gps_unavailable) shows the dominant failure is **pure acquisition timeout** — the request never reaches the geofence. The gate is correct; the acquisition layer fails the worker before the gate ever runs.

## Root cause (scout-verified)

`EmployeeCheckInCard.tsx` runs **two disjoint GPS acquisitions**:

1. **Always-on preview** (`useEffect` L203–240): budget **12s** (`timeoutMs: 12000`). Silently gives up on cold fixes. Sets `lastFreshSample` only on success.
2. **Submit path** (`handleAction` L264): budget **30s**. Clears `lastFreshSample` on tap (L249), then **cold-starts from zero** regardless of what the preview acquired.

A cold GNSS fix at a gate takes 22–28s. Preview dies at 12s → worker taps → submit starts cold → 30s spinner → often timeout. Part A (verified today, not yet deployed) raised the budget and allowed preview reuse, but **only when the preview succeeded** — which at a 12s budget it usually doesn't on a cold phone. **The 12s preview budget is the seam.**

## Non-goals (OUT of scope this round)

- Geofence, accuracy-gate, checkout-gate, shift-window logic (frontend + backend) — frozen.
- Proactive device pre-flight / guided retry (Approach 2) — deferred.
- Adaptive budget / stationary clustering (Approach 3) — deferred until production data justifies.
- Native wrapper, true background geolocation (iOS PWA can't), offline queue.
- Workers indoors / under steel with no satellite fix — correct answer is existing admin override (`RecordAdminCheckIn`), not this work.

## Agreed solution — Approach 1: Continuous warm-up + seamless handoff

Unify preview + submit into **one continuous acquisition state machine** owning a single `watchPosition` lifecycle + rolling best-sample cache, running while the card is mounted and in an actionable state. On tap: submit the cached hot sample if fresh + clears the directional accuracy bar; otherwise keep acquiring to a 30s hard cap.

### Design

- **One watch, not two.** A single `watchPosition` runs continuously while the card is mounted and `canPreviewCheckLocation`-equivalent is true. Replaces both the 12s preview effect and the cold submit start.
- **Rolling cache.** Track best-fresh-sample (lat/lng/accuracy/timestamp/sampleCount/elapsedMs), freshest sample, and convergence trend. Never overwrite a better sample with a worse one while the worker is stationary.
- **Instant-submit rule.** On tap, compute guidance via existing `getCheckInGeofenceGuidance(target, cachedSample)`:
  - `status === "inside"` (i.e. `dist + accuracy ≤ radius` — mirrors backend gate) AND sample age ≤ 15s → **submit immediately, no spinner.**
  - Else → keep watching, show converging-accuracy UX, submit when `inside`, hard cap 30s.
- **Map preview reads the same cache** — `EmployeeLocationMap` gets a continuously-fresh sample instead of a 12s-or-stale one.
- **Lifecycle/battery guard.** Clear watch on card unmount and on `visibilitychange → hidden`; re-warm on visible. iOS suspends the PWA anyway, so this is bounded. No watch leak.
- **No new backend contract.** Payload unchanged: `{lat, lng, accuracy, gps_at, gps_sample_count, gps_best_accuracy, gps_elapsed_ms}`. The count/elapsed now reflect the continuous watch — still honest.

### Natural module boundary (DRY)

Extract a `useContinuousLocation(target, options)` hook (or extend `geolocation.ts` with a continuous primitive) that owns the watch + cache + tap-evaluation. Card consumes it; both check-in and check-out paths share it. Removes the duplicated acquisition logic between preview effect and `handleAction`.

## Touchpoints

| File | Change |
|------|--------|
| `frontend/src/utils/geolocation.ts` | Add continuous-acquisition primitive (or mode on `requestBestCurrentLocation`). Stop exposing two-budget split. |
| `frontend/src/components/employees/EmployeeCheckInCard.tsx` | Replace 12s preview effect + cold submit with one shared hook. On-tap instant-submit from cache. |
| `frontend/src/components/employees/EmployeeLocationMap.tsx` | No code change expected — receives a fresher `sample` from the shared cache. |
| `frontend/src/utils/checkInGeofenceGuidance.ts` | Reused as-is — already computes the `inside` check that mirrors the backend gate. |

## Risks & mitigations

| Risk | Mitigation |
|------|-----------|
| Battery drain from continuous watch | Run only while card mounted + visible; clear on `visibilitychange hidden`/unmount. Worker is at the gate <2 min. |
| Stale cached fix submitted after worker moves | Freshness window ≤15s; server gate still validates; continuous watch keeps producing fresh samples while stationary. If newest sample diverges from cache position, prefer newest. |
| Watch leak (double watchPosition) | Unifying removes today's two-watch split; single `watchId` owned by the hook, cleared in cleanup. |
| Worker-perceived regression if "instant" path submits a fix that server then rejects | Server rejection still surfaces via existing error UX (poor-accuracy / outside-geofence banners). Net effect still strictly better than today's cold-start. |

## Acceptance criteria

1. Worker opens app at gate (open sky), waits a few seconds, taps **Vào làm** → submits in <3s from cached warm fix, no 30s spinner.
2. Worker taps immediately on a cold phone → watch keeps running, converging-accuracy UX shown, submits when directional bar clears, hard cap 30s.
3. **Tan ca** (check-out) behaves identically via the same hook.
4. **No gate regression** — worker outside the gate still rejected; `accuracy > radius` still rejected (server gate unchanged; we only stop sending bad fixes, never send one that should pass when it shouldn't).
5. No `watchPosition` leak (verify single watchId, clean cleanup).
6. `pnpm lint` (includes `tsc --noEmit`) + `pnpm build` clean.

## Success metrics (post-deploy)

- Drop in `attendance_failed_attempts` rows with `gps_status ∈ {gps_timeout, gps_unavailable}` (the null-fix failures).
- Median worker-perceived tap→success latency falls from ~25s (cold) to <3s (hot handoff).
- `validateGeofence` rejection rate unchanged (gates untouched — sanity check, not a target).

## Dependencies / pre-conditions

- **Part A must ship first** (30s budget, removed inner watchPosition timeout, `maximumAge: 15s`, preview-sample reuse). Verified lint/build-clean today; blocked only on deploy (Docker SIGSEGV fix for `make push` finished this morning). Approach 1 builds on top of Part A.

## Notes for `/ck:plan`

- Default `/ck:plan` mode fits (new frontend behavior, no critical business-logic refactor, no strong existing test coverage to lock in). `--tdd` is optional — the codebase has minimal FE test coverage today, so TDD would mean writing the harness first; reasonable but not required.
- Suggested phases: (1) extract continuous-acquisition primitive in `geolocation.ts` with unit tests for the cache/freshness/inside-rule; (2) wire into `EmployeeCheckInCard.tsx`, remove 12s preview effect + cold-submit; (3) lifecycle/battery guards; (4) manual + build verification.
- Keep the change contained to the two files (+ the shared hook). Resist pressure to also touch the gates or the map component.
