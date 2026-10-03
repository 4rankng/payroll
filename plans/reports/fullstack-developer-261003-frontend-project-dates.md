# Frontend Phase 3 Report — Remove project dates + guarded delete

Plan: `/Users/dev/.claude/plans/i-want-to-remove-robust-harbor.md` (Phase 3)
Date: 2026-10-03
Status: completed

## Files changed (8, all within declared ownership)

| File | Change |
|------|--------|
| `frontend/src/components/sheets/AddProjectSheet.tsx` | Removed "Thời gian dự án" section, date form state/defaults/reset, date validation, and the start-date status derivation — projects now always submit `status: 'active'`. Dropped orphaned `todayDate` memo, `useMemo`, and `Calendar` imports; updated the two layout comments. |
| `frontend/src/components/modals/ModalRouter.tsx` | `effective_from` for payrate creation on project create now falls back to `new Date().toISOString().split('T')[0]` (today, YYYY-MM-DD — same string the old start_date default produced; backend parses `2006-01-02`). |
| `frontend/src/components/sheets/ProjectEditSheet.tsx` | Stripped dates from `ProjectFormData`, initial state, `originalData` prefill, validation, update payload, and JSX. Removed orphaned `formatDateForInput` helper and `formatDateForAPI` import. |
| `frontend/src/components/projects/details/ProjectInfoTab.tsx` | Removed "Thời gian" section and both date Fields; description now renders in its own section (SectionLabel `FileText` + same card styling), hidden when absent. Removed orphaned `formatDateForDisplay` / `formatDate` / `Clock`. |
| `frontend/src/components/projects/details/ProjectHeader.tsx` | Removed the date-range derivation and rendering; dropped `CalendarRange` and `date-fns` `format` imports. Metadata row now shows client only. |
| `frontend/src/components/projects/ProjectMobileList.tsx` | Removed the date span, its Calendar icon, and the adjacent separator (row reads `code · employee badges`). Removed orphaned `formatDateShort` / `formatDate`. `Calendar` import kept — still used by the "Bảng công" button. |
| `frontend/src/types/api/project.types.ts` | `start_date`/`end_date` made optional with `@deprecated backend no longer returns these` JSDoc on `Project`, `CreateProjectData`, `UpdateProjectData`, `ProjectFormData`. Fields not deleted. |
| `frontend/src/components/sheets/ProjectDetailsSheet.tsx` | Added `hasAssignedEmployees = activeEmployeesCount > 0`; Xóa button `disabled={hasAssignedEmployees}`; subtitle conditional ("Dự án đang có nhân viên được giao — kết thúc phân công của tất cả nhân viên trước khi xóa" vs "Hành động này không thể hoàn tác"); confirm-dialog footnote now "Việc xóa sẽ bị từ chối nếu còn nhân viên được giao hoặc có bảng công đã duyệt." `canDeleteProject` visibility and ConfirmDialog onConfirm/confirmText pattern untouched. |

## Gates

- `pnpm lint && pnpm type-check` — PASS (exit 0). Tail of output:

```
> vite_react_shadcn_ts@1.10.0 lint
> eslint . && pnpm type-check

> vite_react_shadcn_ts@1.10.0 type-check
> tsc -p tsconfig.app.json --noEmit && tsc -p tsconfig.node.json --noEmit
[exited with code 0]
```

- Grep for `Ngày bắt đầu|Ngày kết thúc|Thời gian` across `components/sheets/` and `components/projects/`: no project-date UI remains. The only hits are `ProjectAssignmentSheet.tsx:377,389` — the per-employee assignment date inputs ("Phân công dự án" sheet, `project_employees.start_date`/`end_date`), which the plan explicitly keeps (assignment dates are a different concept, out of scope).
- `git status`: my changes are exactly the 8 files above. All other dirty files (backend/*, employee/attendance/config/styles) were already modified by the parallel backend session and the pre-existing uncommitted frontend work; none touched.

## Notes / deviations

- None functional. Two comment-only rewordings to keep comments truthful after removal ("basic info and timeline" → "basic info and salary settings"; "dates+salary+offdays" → "salary+offdays"), consistent with the files' existing comment style.
- The dead-legacy copies (ProjectCreateModal, ProjectEditModal, modals/project-detail/*, partner-project-columns, partner-project-mobile) still compile after the type change — verified by the passing type-check (`tsconfig.app.json` runs with `strict: false`, and the legacy readers guard or `||`-fallback the now-optional fields).
- `effective_from` uses the UTC-based `toISOString()` date, identical to the old `todayDate` default it replaces (per plan: "produce the same string the old start_date default produced").

## Follow-ups for other phases

- Phase 4 integration tests and the live `make api-test` smoke (create arrives `active`, payrate `effective_from` = today) can proceed against this frontend.
