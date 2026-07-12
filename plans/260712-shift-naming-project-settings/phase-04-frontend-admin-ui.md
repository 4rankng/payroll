---
phase: 4
title: "Frontend: admin ShiftNamesSection"
status: pending
priority: P1
dependencies: [3]
---

# Phase 4: Frontend — admin "Tên ca làm việc" section

## Overview
Add a new inline-editable section to `ProjectDetailsSheet` (the same sheet that hosts `GeofenceSection`) that lists every shift range detected from the project's payrate and lets the admin type a display name for each. Save via the existing `useUpdateProject` mutation.

## Requirements
- Functional: for an `is_flexible` project, admin sees one row per distinct shift time-range in the payrate, each with a text input for the name; Save persists `shift_names`; Cancel discards.
- Non-functional: reuse the `GeofenceSection` inline-edit pattern (local state → save/cancel buttons → `useUpdateProject`) so the UX matches the rest of the sheet.

## Architecture
New component `ShiftNamesSection.tsx` colocated with `GeofenceSection.tsx`. It reads:
- the configured shift ranges from the project's payrate (`project.payrates` or a dedicated selector — match how `PayrateConfigTab` reads them), and
- the saved `project.shift_names`,
and shows one labelled text input per detected range. The range key displayed is the same `"HH:MM-HH:MM"` string the backend joins on.

The section renders only when `project.is_flexible` and there is at least one payrate shift. `ProjectDetailsSheet.tsx` adds it right after `GeofenceSection` (~line 366).

## Related Code Files
- Create: `frontend/src/components/projects/details/ShiftNamesSection.tsx`
- Modify: `frontend/src/components/sheets/ProjectDetailsSheet.tsx` — import + render `<ShiftNamesSection project={project} />` inside the `project.is_flexible` block near line 364-366.
- Reference (no change): `frontend/src/components/projects/details/GeofenceSection.tsx` — copy the local-state + Save/Cancel + `useUpdateProject` pattern.
- Reference (no change): `frontend/src/components/payrates/components/matrix-editor/FlexibleShiftManager.tsx` — only for how it reads payrate shift ranges; do **not** reuse `PRESET_SHIFTS` labels here.

## Implementation Steps
1. Implement `ShiftNamesSection` with this shape (mirrors `GeofenceSection`):
   - `useMemo` computes `detectedRanges: string[]` from the payrate (flatten each payrate entry's keys, extract the `"HH:MM-HH:MM"` suffix the same way the backend does).
   - local `names` state seeded from `project.shift_names` (keyed by range).
   - for each detected range, render a row: a `<Label>` showing `09:00 - 18:00` (and a small overnight badge when `end ≤ start`), plus a shadcn `<Input>` bound to `names[range].name`.
   - Save → `updateMutation.mutate({ id: project.id, data: { shift_names: detectedRanges.map(r => ({ range: r, name: names[r] ?? "" })) } })`; disable button while pending.
   - Cancel → reset local state.
2. Mount it in `ProjectDetailsSheet` inside the existing `project.is_flexible` guard.
3. Surface server validation errors (the 400 from phase 2) via the sheet's existing toast/error surface.

## Success Criteria
- [ ] Section renders only for `is_flexible` projects with ≥1 payrate shift.
- [ ] Admin can name each range; Save persists; reload shows the names.
- [ ] Empty name rows are sent as `""` (backend treats as "not set" → employee sees fallback label).
- [ ] Overnight range rows show the overnight badge inline.
- [ ] Disabled while `updateMutation.isPending`.

## Risk Assessment
- **Risk:** Reading payrate ranges on the frontend may need the payrate flattened the same way as backend. **Mitigation:** reuse whatever `PayrateConfigTab`/`FlexibleShiftManager` already does to enumerate shifts; if no helper exists, extract the `"HH:MM-HH:MM"` suffix with the same regex used in `ValidateShiftNames`.
- **Risk:** New ranges added to the payrate after names are set leave orphan names. **Mitigation:** the section only renders detected ranges, so orphans are invisible and harmlessly ignored on save; backend validation already rejects ranges not in the payrate.
