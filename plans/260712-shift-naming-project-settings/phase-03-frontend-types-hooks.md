---
phase: 3
title: "Frontend: types + API hooks"
status: pending
priority: P1
dependencies: [2]
---

# Phase 3: Frontend — types and API plumbing

## Overview
Mirror the backend's new `shift_names` field in the frontend `Project` and `UpdateProjectData` types so the existing `useUpdateProject` mutation carries it without further wiring. No new endpoint.

## Requirements
- Functional: reading a project returns `shift_names`; updating a project sends `shift_names`; both typed end-to-end.
- Non-functional: no new hook or service method — the existing `projectService.updateProject` already spreads `UpdateProjectData` into the PATCH body.

## Architecture
`frontend/src/types/api/project.types.ts` owns the `Project` and `UpdateProjectData` interfaces. Add `ShiftName` and the optional field to both. The `useProjects.ts` `useUpdateProject` mutation (line 195) already forwards `data: UpdateProjectData` to `projectService.updateProject`, which PATCHes it — no change needed there.

## Related Code Files
- Modify: `frontend/src/types/api/project.types.ts` — add `ShiftName` interface (`{ range: string; name: string }`); add `shift_names?: ShiftName[] | null` to `Project` (after `geofence_radius_meters`, ~line 60) and to `UpdateProjectData` (~lines 92-110).
- Verify (no change unless missing): `frontend/src/services/api/project.service.ts` — `updateProject` spreads the body; confirm it doesn't strip unknown keys.

## Implementation Steps
1. Add the `ShiftName` interface next to `GeofenceGate`.
2. Add `shift_names?` to `Project` and `UpdateProjectData`.
3. Grep for any explicit field allow-list in `project.service.ts` and extend it if `shift_names` would be dropped.
4. `pnpm type-check` — should pass with the additive type.

## Success Criteria
- [ ] `pnpm type-check` clean.
- [ ] A `useUpdateProject().mutate({ id, data: { shift_names: [...] } })` call type-checks.
- [ ] `useProject(id).data.shift_names` is typed `ShiftName[] | null | undefined`.

## Risk Assessment
- **Risk:** `UpdateProjectData` may use a strict `Pick<Project, ...>` that excludes new fields. **Mitigation:** if so, add `shift_names` to the Pick union or define it explicitly (the existing `geofence_gates` pattern shows which).
