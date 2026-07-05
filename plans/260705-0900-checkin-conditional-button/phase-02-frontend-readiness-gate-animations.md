---
phase: 2
title: "Frontend readiness gate + micro-animations"
status: pending
priority: P1
effort: "M (1d)"
dependencies: [1]
---

# Phase 2: Frontend readiness gate + micro-animations

## Overview
Wire the GPS state + geofence guidance + the new shift-window signal (Phase 1)
into a single "check-in readiness" predicate that controls the button's
visibility, and add three micro-animations (GPS-warming pulse, became-ready pop,
outside-window hint) using existing animation utilities.

## Requirements
- Functional: the check-in button shows **only when** GPS is stable
  (`isSubmitReady`) + inside geofence + within the shift window. When any
  condition is unmet, the button is replaced by an appropriate status/hint.
- Non-functional: no new animation library. Uses `tailwindcss-animate` +
  existing keyframes. Mobile-first. Smooth transitions, no layout jank.

## Architecture

### Readiness predicate
A derived boolean combining three signals the card already has (or will have
after Phase 1):

```ts
const withinWindow = useMemo(() => {
  // Phase 1 fields on the profile
  const winStart = profile?.check_in_window_start;  // "07:00"
  const winEnd = profile?.check_in_window_end;       // "09:00"
  if (!winStart || !winEnd) return true;             // no shift configured → no timing gate
  const now = new Date();
  const nowMin = now.getHours() * 60 + now.getMinutes();
  const [sh, sm] = winStart.split(":").map(Number);
  const [eh, em] = winEnd.split(":").map(Number);
  return nowMin >= sh * 60 + sm && nowMin <= eh * 60 + em;
}, [profile]);

const gpsReady = isSubmitReady;  // from useContinuousLocation (GPS stable + inside geofence)
const gpsAcquiring = isWatching && !gpsReady && !fatalError;

const canShowButton = withinWindow && gpsReady;
```

### Button states → UI

| State | Condition | UI |
|-------|-----------|-----|
| **Ready** | `canShowButton` | Button visible, **green**, with a "became-ready" pop animation on transition into this state |
| **GPS warming** | `withinWindow && gpsAcquiring` | Button visible but **pulsing** (breathing animation), disabled, with status text "Đang xác định vị trí..." |
| **GPS failed** | `withinWindow && fatalError` | Recovery panel (existing) — permission denied / unavailable |
| **Outside geofence** | `withinWindow && guidance.status === "outside"` | Button hidden; hint "Bạn đang ngoài khu vực chấm công (cách Nm)" |
| **Outside window** | `!withinWindow` | Button hidden; hint "Ca làm việc bắt đầu lúc HH:MM" + countdown to window start |
| **No shift configured** | `!winStart && !winEnd` | Button shows when GPS ready (no timing gate) — backward compatible |

### Micro-animations (existing utilities, no new library)

1. **GPS-warming pulse** — apply the existing `employee-pay-breathe` keyframe
   (`base.css:355`) or `animate-pulse` (tailwindcss-animate) to the button while
   `gpsAcquiring`. A gentle scale + opacity breathing to signal "waiting".
   ```css
   /* reuse or add to base.css */
   @keyframes check-in-warming {
     0%, 100% { transform: scale(1); opacity: 0.85; }
     50% { transform: scale(1.03); opacity: 1; }
   }
   .check-in-warming { animation: check-in-warming 1.8s ease-in-out infinite; }
   ```

2. **Became-ready pop** — when the button transitions from warming/hidden → ready,
   fire a one-shot pop. Reuse the existing `mobile-stat-pop` keyframe
   (`premium.css:222`) or add a small bounce:
   ```css
   @keyframes check-in-ready-pop {
     0% { transform: scale(0.9); }
     60% { transform: scale(1.08); }
     100% { transform: scale(1); }
   }
   .check-in-ready-pop { animation: check-in-ready-pop 0.4s ease-out; }
   ```
   Trigger via a `useRef` prev-state tracker + a transient CSS class.

3. **Outside-window hint** — no animation; a static info card with the shift time
   + a live countdown ("Còn 45 phút nữa đến giờ chấm công"). Uses
   `setInterval(60s)` to update the countdown text. Fade-in via
   `animate-in fade-in` (tailwindcss-animate).

## Related Code Files
- **Modify:** `frontend/src/components/employees/EmployeeCheckInCard.tsx` — the
  main change. Add the readiness predicate, button-state branching, and
  animation classes. Touches the default branch (`:716-749`) and the completed
  branch (`:630-650`).
- **Modify:** `frontend/src/types/api/auth.types.ts` — add Phase 1's shift-window
  fields to `EmployeeProfile`.
- **Modify:** `frontend/src/styles/base.css` (or `premium.css`) — add the
  `check-in-warming` + `check-in-ready-pop` keyframes.
- **Read-only:** `frontend/src/hooks/useContinuousLocation.ts` (already exposes
  `isSubmitReady`, `progress.status`, `isWatching`, `fatalError`).
- **Read-only:** `frontend/src/utils/checkInGeofenceGuidance.ts` (already
  exposes `guidance.status`).

## Implementation Steps
1. **Add the shift-window type fields** to `EmployeeProfile`
   (`auth.types.ts`) — mirror Phase 1's DTO.
2. **Add the readiness predicate** in `EmployeeCheckInCard.tsx`:
   - `withinWindow` (from profile shift fields + device clock).
   - Read `isSubmitReady`, `progress.status`, `isWatching`, `fatalError` from
     the existing `useContinuousLocation` destructure.
   - Read `guidance.status` from the existing `getCheckInGeofenceGuidance` call.
3. **Add the animation keyframes** to `base.css` (`check-in-warming`,
   `check-in-ready-pop`). Keep them subtle (mobile, professional).
4. **Rewrite the button render branches** in the default (`:716-749`) and
   completed (`:630-650`) sections:
   - `canShowButton` → green button with `check-in-ready-pop` on transition.
   - `gpsAcquiring` → button with `check-in-warming` pulse + disabled + status text.
   - `!withinWindow` → replace button with the outside-window hint card
     (shift time + countdown).
   - `guidance.status === "outside"` → replace with geofence-outside hint.
5. **Add the became-ready transition tracker** — a `useRef<prevReady>` that
   applies the pop class for 400ms when `canShowButton` transitions false→true.
6. **Add the countdown** for the outside-window hint — `useEffect` with a 60s
   interval updating "Còn Nm nữa".
7. **Test manually** on mobile: GPS warming → ready pop, outside window → hint,
   inside window + GPS stable → green button.

## Success Criteria
- [ ] Button is hidden when outside the shift window; replaced by
      "Ca làm việc bắt đầu lúc HH:MM" + countdown.
- [ ] Button pulses (breathing) while GPS is acquiring; pops green when GPS
      becomes stable + inside geofence + within window.
- [ ] Button is green and tappable only when all three conditions are met.
- [ ] No new animation library installed.
- [ ] `pnpm exec tsc --noEmit` clean.
- [ ] Manual mobile test: warming → ready pop is smooth, no layout jank.
- [ ] Backward compatible: employee with no shift configured (null fields) →
      button shows when GPS ready (no timing gate).

## Risk Assessment
- **Risk:** device clock drift causes the frontend "withinWindow" to disagree
  with the server's `validateCheckInWindow`. **Mitigation:** the server remains
  authoritative; the frontend gate is advisory. A mismatch means the button
  shows but the server rejects — same as today's behavior, just less frequent.
- **Risk:** animation jank on low-end Android. **Mitigation:** keep animations
  GPU-friendly (`transform`/`opacity` only, no layout properties); respect
  `prefers-reduced-motion` (disable warming pulse if set).
- **Risk:** the `EmployeeCheckInCard` is already large (~750 lines). Adding
  branching increases complexity. **Mitigation:** extract the readiness
  predicate + status derivation into a small `useCheckInReadiness` hook to keep
  the card readable.
