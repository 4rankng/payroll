---
phase: 1
title: Geolocation primitive
status: completed
effort: M
dependencies: []
---

# Phase 1: Geolocation primitive

## Overview

Add a continuous-acquisition primitive to `frontend/src/utils/geolocation.ts` that owns a single `watchPosition` lifecycle and a rolling best-sample cache, plus small pure helpers (freshness window, cache update, convergence/stall detection). Keep the existing one-shot `requestBestCurrentLocation` intact (still useful, and the floor this plan builds on). Unit-test the pure logic without DOM mocking; add one lifecycle test with a stub `navigator.geolocation`.

## Requirements

- **Functional**
  - New `watchContinuousLocation(options)` exposes: `unsubscribe()`, `getBestFreshSample()`, and an `onUpdate` callback receiving a `LocationAcquisitionProgress`-shaped payload (sample count, best accuracy, freshest sample, elapsed, status).
  - Rolling cache: keep the freshest accurate sample (`freshMaxAgeMs`, default 15s) with the **best** accuracy among fresh samples; never overwrite a better sample with a worse one while the worker is stationary, BUT prefer a newer sample whose *position* diverges (worker moved).
  - No `watchPosition` leak: a single `watchId` per primitive instance; `unsubscribe()` calls `clearWatch` + clears the outer timeout and any warmup timer.
  - Reuse the existing `enableHighAccuracy: true`, `maximumAge: 15000` (Part A) watch options; no per-attempt `timeout` (Part A removed it — outer budget governs).
- **Non-functional**
  - Pure helpers are exported and unit-testable with no DOM.
  - No backend contract change.

## Architecture

```
watchContinuousLocation(options) ──┬── watchPosition callback ──► updateCache() ──► onUpdate(progress)
                                   │                              └── getBestFreshSample() (sync read)
                                   └── outer setTimeout(budgetMs) ──► onTimeout (no reject; just stops best-effort)
```

- `updateCache(prev, incoming)` — pure: returns next cache state. Rules: incoming must be fresh (`now - timestamp ≤ freshMaxAgeMs`); keep the sample with min `accuracy`; if incoming accuracy worse but position moved > N m (use `distanceMetersBetween` from `checkInGeofenceGuidance.ts`), treat as a new position and accept it as the new "best for this spot" only if it's the freshest.
- `isSampleFresh(sample, now, maxAgeMs)` — pure boolean.
- `classifyStatus(bestAccuracy, options)` — reuse the existing `getAccuracyStatus` logic (excellent/acceptable/weak/warming); extract or call it.

## Related Code Files

- **Modify:** `frontend/src/utils/geolocation.ts` — add `watchContinuousLocation` + pure helpers; refactor existing `getAccuracyStatus` to be shared/exportable if needed.
- **Reuse (no change):** `frontend/src/utils/checkInGeofenceGuidance.ts` (`distanceMetersBetween`, `getCheckInGeofenceGuidance`) — the "inside" oracle used at tap time lives here already; do NOT duplicate.
- **Create:** `frontend/src/utils/geolocation.continuous.test.ts` — Vitest unit tests for pure helpers + one lifecycle test with a stub `navigator.geolocation`.

## Implementation Steps

1. Extract `getAccuracyStatus` (currently file-private) to an exported helper if `watchContinuousLocation` needs it; otherwise add a thin shared classifier.
2. Add pure helpers: `isSampleFresh(sample, now, maxAgeMs)`, `updateLocationCache(prev, incoming, options)` returning `{ bestFreshSample, sampleCount, freshSampleCount, bestAccuracy }`.
3. Add `watchContinuousLocation(options): { unsubscribe, getBestFreshSample }` modeled on the watchPosition block of `requestBestCurrentLocation` (L251–294) but **non-terminating** — it does not resolve/reject on a good sample; it keeps the cache warm and emits `onUpdate` until `unsubscribe()`.
4. Accept a `budgetMs` (default 30000) that, when elapsing, emits a `status: "weak"`/stalled signal but does NOT auto-unsubscribe (the React hook decides when to stop). Provide an `onStalled` callback hook for the UX layer.
5. Write `geolocation.continuous.test.ts`:
   - `isSampleFresh` boundary (==maxAge fresh, maxAge+1 stale).
   - `updateLocationCache`: better accuracy replaces; worse accuracy at same spot retained as fallback; moved position handled.
   - `watchContinuousLocation` lifecycle with stub geolocation: single `watchPosition` call, `clearWatch` on `unsubscribe()`, no double-watch, `onUpdate` fires with correct shape.
6. Run `pnpm lint` + `pnpm test --run geolocation.continuous`.

## Success Criteria

- [ ] `watchContinuousLocation` exported with `{ unsubscribe, getBestFreshSample }` and an `onUpdate`/`onStalled` callback contract.
- [ ] Pure helpers `isSampleFresh`, `updateLocationCache` exported and unit-tested.
- [ ] Single `watchPosition` per instance; `clearWatch` + timer cleanup verified by lifecycle test (no leak).
- [ ] Existing `requestBestCurrentLocation` untouched (diff confirms no behavioral change to the one-shot path).
- [ ] `geolocation.continuous.test.ts` green.
- [ ] `pnpm lint` clean.

## Risk Assessment

- **Over-sharing watchPosition across the one-shot and continuous paths** → tempting DRY trap. Keep them as two thin entry points over a *small* shared inner (watch+cache) only if it cleanly falls out; otherwise accept two code paths. Don't pre-refactor `requestBestCurrentLocation`.
- **Stale-fix risk in the cache** → mitigated by `freshMaxAgeMs` (15s) and the "moved position" rule; the server gate is the final backstop.
- **Test brittleness from DOM mocking** → keep tests on pure helpers; the single lifecycle test uses a minimal hand-rolled `navigator.geolocation` stub, no `jsdom` env dependency beyond what Vitest already provides.
