---
title: "Admin-named shifts in project settings (check-in mode)"
description: "Let admin assign display names (e.g. 'Ca làm', 'Ca đêm') to the shift time-ranges already configured in the payrate, for flexible/check-in-enabled projects. Payrate stays the single source of truth for shift times. Names surface on the employee attendance card with an overnight badge."
status: pending
priority: P2
branch: main
tags: [attendance, admin, project-settings, frontend, backend, shift]
blockedBy: []
blocks: []
created: "2026-07-12T00:00:00.000Z"
createdBy: "ck:plan"
source: skill
related: [260712-attendance-reference-image-to-code, 260705-0900-checkin-conditional-button]
---

# Admin-named shifts in project settings (check-in mode)

## Overview

Today the employee attendance card (`AttendanceReference`, `EmployeeCheckInCard.tsx:272`) labels every shift with a hardcoded fallback — `index === 0 ? "Ca ngày" : "Ca đêm"` — because the advisory `ScheduleWindow` DTO (`dto.ScheduleWindowInfo`, `backend/internal/app/dto/employee_profile.go:44`) carries no name. Admin has no way to communicate the real shift names ("Ca làm", "Ca đêm", "Hành chính") to workers.

This plan adds a **`shift_names`** JSON column to `projects` (mirroring the existing `geofence_gates` pattern) that maps a payrate time-range (`"09:00-18:00"`) to an admin-chosen display name. Payrate remains the single source of truth for times and pricing — admin only names the ranges the payrate already defines. The names propagate through the existing advisory chain (`ResolveAllShiftWindows` → `ShiftWindow.Name` → `ScheduleWindowInfo.ShiftName` → `dto.ScheduleWindowInfo.ShiftName`) and render on the employee card with a small "qua đêm" overnight badge.

## Decision (locked from user interview)

- **Shift source = name existing payrate shifts.** Admin sees the shift ranges detected from the project's payrate and types a display name for each. Payrate is not touched; it stays the single source of truth for shift times.
- **Overnight display = name + badge.** Show the admin-given name, plus a small overnight badge when `end ≤ start` (shift crosses midnight).

## Why this shape

- `geofence_gates` already proves the `[]struct{...}` GORM `serializer:json` column works on `projects` with full validation, transport DTO, and per-section UI (`GeofenceSection.tsx`). Copying it is the lowest-risk path and keeps the migration to one new JSON column.
- `resolveShifts`/`ResolveAllShiftWindows` already build a `"HH:MM-HH:MM"` time-range key (`attendance_service.go:1375`). That key is the natural join between the admin's name config and the resolved shift windows — attach the name where the key is constructed, no matching ambiguity.
- The advisory (`schedule_windows`) chain is display-only, so adding a field is a non-breaking additive change. No validation logic moves.

## Scope

**In scope**
- New `ShiftNames []ShiftName` on `domain.Project` + JSON column.
- Validation: name required when present, ≤50 chars, range must match a payrate-configured shift, no duplicate ranges, ≤20 entries, gated on `is_flexible`.
- Admin: `CreateProjectRequest`/`UpdateProjectRequest` fields + handler wiring (same shape as `GeofenceGates`).
- Admin UI: new "Tên ca làm việc" section in `ProjectDetailsSheet`, gated on `project.is_flexible`, using the `GeofenceSection` inline-edit pattern and `FlexibleShiftManager`'s `type="time"` precedent only as reference.
- Advisory propagation: `ShiftWindow.Name` → `ScheduleWindowInfo.ShiftName` → `dto.ScheduleWindowInfo.ShiftName`.
- Frontend types + `AttendanceReference` rendering (name + overnight badge), with fallback to today's index-based labels when no name is configured.

**Out of scope**
- Editing payrate shift times or pricing (payrate is untouched).
- Per-employee shift assignment (still position/project-wide).
- The `PRESET_SHIFTS` label arrays in `FlexibleShiftManager.tsx` / `FlexiblePayrateEditor.tsx` — those label payrate hour-types, a separate concern.
- i18n framework (codebase has none; Vietnamese stays inline).

## Data model

```go
// domain/project.go (new struct + field, mirrors GeofenceGate)
type ShiftName struct {
    Range string `json:"range"`  // "HH:MM-HH:MM" 24h, e.g. "09:00-18:00"; must match a payrate shift
    Name  string `json:"name"`   // admin-chosen display name, e.g. "Ca làm"
}

// added to Project
ShiftNames []ShiftName `json:"shift_names" gorm:"type:json;serializer:json"`
```

```sql
-- migrations/089_add_project_shift_names.up.sql
ALTER TABLE projects ADD COLUMN shift_names JSON NULL;
-- down.sql
ALTER TABLE projects DROP COLUMN shift_names;
```

## Phases

1. **Backend: domain + migration + validation** — `ShiftName` struct, `Project.ShiftNames` field, `ValidateShiftNames`, migration `089`.
2. **Backend: transport + advisory propagation** — extend `Create/UpdateProjectRequest`, wire handler (mirror `GeofenceGates`), attach `Name` in `ResolveAllShiftWindows`, add `ShiftName` to `ScheduleWindowInfo` + DTO + mapper.
3. **Frontend: types + hooks** — `ShiftName` type on `Project`/`UpdateProjectData`, ensure `useUpdateProject` carries it.
4. **Frontend: admin UI** — new `ShiftNamesSection` in `ProjectDetailsSheet`, inline-edit pattern from `GeofenceSection`, detects ranges from payrate, names each.
5. **Frontend: employee display** — render `schedule_window.shift_name` with overnight badge in `AttendanceReference`, fallback to current index labels.
6. **Tests + build gates** — backend unit tests, frontend component tests, lint/type-check, build.

See `phase-01-*.md` … `phase-06-*.md` for detail.

## Validation

- `go test ./internal/app/services/attendance/... ./internal/domain/... ./internal/transport/http/handlers/project/...` green.
- New unit tests: `ValidateShiftNames` (required/length/dup/range-mismatch/limit), `ResolveAllShiftWindows` attaches names by range key.
- `cd frontend && pnpm lint && pnpm type-check` clean.
- `AttendanceReference` test updated: named shift renders admin name; unnamed falls back to `Ca ngày`/`Ca đêm`; overnight badge shows on `21:00-05:00`.
- Production build (`make api-test` / `pnpm build`) passes.

## Open questions

None — both forks resolved in interview (name existing payrate shifts; name + overnight badge).
