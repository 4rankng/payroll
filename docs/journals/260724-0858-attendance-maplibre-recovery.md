---
title: "Attendance MapLibre recovery"
date: "2026-07-24 08:58"
severity: "High"
component: "frontend/src/components/employees/EmployeeLocationMap.tsx"
status: "Resolved"
---

## Context

The employee attendance map had two bad assumptions baked into its lifecycle. GPS/sample updates were being treated like map reconstruction work instead of data updates into one mounted MapLibre instance, and a generic async map error could latch the card into its fallback state permanently.

## What Happened

A transient MapLibre failure was enough to leave the employee card stuck on `Không tải được bản đồ.` until the page was refreshed. That was the wrong failure mode: the map was not dead, it just needed a clean remount. The frustrating part is that repeated location samples also exercised the lifecycle too hard when they should have only synchronized the mounted map.

## Decision

We kept one live MapLibre instance and pushed GPS/sample changes into that mounted map. When MapLibre reports an error, the card now shows the fallback with an explicit retry action instead of silently staying wedged. Retry remounts the map through a fresh instance, and the old instance is removed before the new one takes over.

## Verification

- `EmployeeLocationMap.test.tsx` now covers one-instance lifecycle behavior while repeated location attempts update the sample.
- The fallback test proves status details stay visible when MapLibre emits an error.
- The retry test proves the old instance is cleaned up (`remove()` called once) and a second instance is mounted after clicking `Thử tải lại bản đồ`.
- The render-failure test still isolates the map crash inside the attendance card instead of taking down the rest of the page.

## Lessons Learned

- A map canvas is stateful infrastructure, not a disposable render target for every GPS update.
- Generic async errors need an explicit recovery path; otherwise they become dead ends.
- Cleanup and remount behavior must be tested together, or the fallback will look “handled” while remaining unrecoverable.

## Next Steps

No follow-up beyond keeping these regression tests in CI and watching for future MapLibre wrapper or lifecycle regressions.
