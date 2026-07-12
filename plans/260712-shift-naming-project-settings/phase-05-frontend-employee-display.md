---
phase: 5
title: "Frontend: employee attendance card display"
status: pending
priority: P1
dependencies: [2, 3]
---

# Phase 5: Frontend — render named shifts + overnight badge on the employee card

## Overview
Replace the hardcoded `index === 0 ? "Ca ngày" : "Ca đêm"` tab labels in `AttendanceReference` with the admin-configured `shift_name`, keeping today's labels as a fallback. Add a small overnight badge when a shift's time range crosses midnight.

## Requirements
- Functional: when `schedule_window.shift_name` is non-empty, use it as the tab label; otherwise fall back to the current index-based label so nothing regresses.
- Non-functional: no new dependency; reuse shadcn `Badge` / tailwind classes already in the file.

## Architecture
- Extend `AttendanceScheduleWindow` (`frontend/src/types/api/auth.types.ts:91-98`) with `shift_name?: string`.
- In `AttendanceReference` (`EmployeeCheckInCard.tsx:199-358`):
  - line 272 `{index === 0 ? "Ca ngày" : "Ca đêm"}` → `{window.shift_name?.trim() || (index === 0 ? "Ca ngày" : "Ca đêm")}`;
  - add an overnight badge next to the active tab when the window's shift range crosses midnight. Compute overnight by parsing `window.shift_start`/`shift_end` (already RFC 3339 instants) — overnight when `shift_end <= shift_start`.
- Optionally also reflect `shift_name` on the active row label at line 291 (`"Ca làm"` row) when it differs from the generic; keep the row label as-is unless the user wants the named label there too — default: leave row labels, only change tab labels + add badge.

## Related Code Files
- Modify: `frontend/src/types/api/auth.types.ts` — add `shift_name?: string` to `AttendanceScheduleWindow` (line 91-98).
- Modify: `frontend/src/components/employees/EmployeeCheckInCard.tsx` — tab label at line 272; add overnight badge helper + render near the tab and active heading.
- Modify: `frontend/src/components/employees/EmployeeCheckInCard.test.tsx` — update/add cases.

## Implementation Steps
1. Add `shift_name?` to `AttendanceScheduleWindow`.
2. Replace the hardcoded tab-label expression with `window.shift_name?.trim() || fallback`.
3. Add a small `OvernightBadge` (or inline `<Badge variant="outline">Qua đêm</Badge>`) shown when the window is overnight.
4. Update tests:
   - existing `Ca ngày`/`Ca đêm` fallback assertions still pass (windows with no `shift_name`);
   - new case: a window with `shift_name: "Ca làm"` renders that tab label;
   - new case: `21:00-05:00` window shows the overnight badge.

## Success Criteria
- [ ] `pnpm test -- EmployeeCheckInCard` green (existing + new cases).
- [ ] No regression when `shift_name` is absent (fallback to `Ca ngày`/`Ca đêm`).
- [ ] Overnight badge shows only for cross-midnight windows.

## Risk Assessment
- **Risk:** `AttendanceReference` consumes `scheduleWindows` as a `Pick<...Props>` — adding a field must not widen the required props. **Mitigation:** make `shift_name` optional; the parent already maps `profile.schedule_windows`.
