---
title: "Frontend: filter row + grouped history view"
status: todo
priority: P1
effort: "4h"
dependencies: [phase-01-backend-filename-search-project-ordering]
---

# Phase 2: Frontend: filter row + grouped history view

## Overview

Give `UploadHistorySheet` a filter row (file-name search input + project
dropdown) and render grouped sections under project headers in the
all-projects view. All four mount points (admin/partner × desktop/mobile)
inherit the shared-component change; the two PARTNER mounts additionally need
the `projects` prop wired (they don't pass it today).

## Requirements

- Functional: debounced (≈350 ms) search input filters by file name
  server-side; typing resets page to 1.
- Functional: project dropdown lists "Tất cả dự án" + `projects` prop;
  hidden only when the sheet is already project-scoped (`projectId` prop set).
- Functional: BOTH partner mounts pass `projects` (they currently pass only a
  conditional `projectId` — `pages/partner/TimesheetsPage/index.tsx:538`,
  `pages/mobile/partner/TimesheetsPage/index.tsx:342`). Without this, the
  dropdown and named headers are invisible to partners — the primary BCC
  uploader role. The pages already hold the data
  (`timesheetManagement.projects` is used by the admin mounts the same way).
- Functional: in the all-projects view, items render under project-name
  headers (project → newest-first via backend `sort=project`); selecting one
  project shows a flat newest-first list without headers. Headers render on
  EVERY page while the all-projects view is active — including continuation
  pages that contain a single project (otherwise the user loses track of which
  project they are scrolling). Headers carry NO counts: page-local counts
  would read as project totals and contradict the pagination footer.
- Functional: search input clears (and project filter resets to "all") when
  the sheet closes — the component stays mounted at all 4 sites, so state
  survives close; a stale filter hiding a fresh upload recreates the
  duplicate-upload loop this feature exists to fix.
- Non-functional: the debounced value sent to the API is clamped to 100
  runes (`slice(0, 100)`) so the backend 400 is unreachable from the UI (the
  sheet has no error state; a 400 would render as "Chưa có tệp nào được tải
  lên" — a lie).
- Non-functional: 44px (h-11) touch targets; Vietnamese labels; filter row
  wraps on mobile; existing tokens/style (`bg-card`, `text-muted-foreground`,
  shadcn `Select`).
- Compatibility: download button, `ErrorDetail` expand, status badges, 2 s
  polling while pending/processing, and the upload modal's
  `['partner-imports']` invalidation all keep working.

## Related Code Files

- Modify: `frontend/src/types/api/timesheet.types.ts` (PartnerImportListParams
  += `search?: string; sort?: string`)
- Modify: `frontend/src/components/timesheet/UploadHistorySheet.tsx`
- Modify: `frontend/src/pages/partner/TimesheetsPage/index.tsx` (pass
  `projects`)
- Modify: `frontend/src/pages/mobile/partner/TimesheetsPage/index.tsx` (pass
  `projects`)
- Reference (no change): `frontend/src/services/api/timesheet.service.ts`
  (`buildQueryString` already serializes new keys and skips empties),
  `frontend/src/hooks/useDebounce.ts`, the admin mounts (already pass
  `projects`).

## Implementation Steps

1. Types: add `search?: string; sort?: string` to `PartnerImportListParams`.
2. `UploadHistorySheet.tsx` state + derived params:
   ```ts
   const [searchInput, setSearchInput] = useState('');
   const debouncedSearch = useDebounce(searchInput, 350);
   const [projectFilter, setProjectFilter] = useState('all'); // 'all' | '<id>'
   const showProjectFilter = !projectId && !!projects && projects.length > 0;
   const effectiveProjectId = projectFilter !== 'all' ? Number(projectFilter) : projectId;
   const clampedSearch = debouncedSearch.trim().slice(0, 100);
   ```
   Reset page on filter change AND clear filters when the sheet closes:
   ```ts
   useEffect(() => { setPage(1); }, [open, projectId, clampedSearch, projectFilter]);
   useEffect(() => { if (!open) { setSearchInput(''); setProjectFilter('all'); } }, [open]);
   ```
   Query params: `{ project_id: effectiveProjectId, search: clampedSearch ||
   undefined, sort: effectiveProjectId ? undefined : 'project', page,
   page_size: pageSize }`.
3. Filter row JSX above the list: search input (Search icon, clear-X button,
   placeholder "Tìm theo tên tệp…") + shadcn `Select` for the project when
   `showProjectFilter`. h-11 controls, `flex flex-wrap gap-2`.
4. Grouped rendering:
   ```ts
   const groups = useMemo(() => { /* Map<project_id, items[]> in server order */ }, [items]);
   const isGrouped = effectiveProjectId == null && groups.length >= 1; // >=1, NOT >1
   ```
   When `isGrouped`, render a header row per group (project name via
   `projectMap`; keep the existing `Dự án #<id>` fallback for unmapped ids)
   above its cards — header text only, no count. Otherwise today's flat list.
5. Empty states: filtered-but-empty → "Không tìm thấy tệp phù hợp" with
   "Thử từ khóa khác hoặc xóa bộ lọc."; unfiltered empty keeps current copy.
6. Partner mounts: add `projects={timesheetManagement.projects}` to
   `UploadHistorySheet` at
   `pages/partner/TimesheetsPage/index.tsx:538` and
   `pages/mobile/partner/TimesheetsPage/index.tsx:342` (mirror the admin
   mounts, e.g. `pages/admin/TimesheetPage/index.tsx:461`). Confirm
   `timesheetManagement.projects` is populated in partner context (the
   selected-project selector there uses the same source).

## Success Criteria

- [x] `pnpm lint` (eslint + `tsc --noEmit`) passes — no separate type-check
      script exists in this repo's frontend.
- [ ] Typing "georim" (any case) narrows the list across pages; totals +
      pagination reflect the filtered count; the clear-X restores the full
      list.
- [ ] All-projects view shows project headers (e.g. "Georim"), including on
      continuation pages; picking a project from the dropdown shows only that
      project, flat, newest-first.
- [ ] PARTNER pages (desktop + mobile) show the dropdown and NAMED group
      headers — not `Dự án #<id>`.
- [ ] Closing and reopening the sheet starts with a cleared search and
      "Tất cả dự án".
- [ ] Download, error expand, badges, polling behave as before; sheet opened
      from a project page hides the dropdown and stays scoped.

## Risk Assessment

- Debounce + TanStack polling: independent (poll only fires while items are
  pending/processing); no interaction expected. If refetch storms appear,
  raise debounce to 500 ms.
- `timesheetManagement.projects` may differ in shape/availability between
  admin and partner pages — verify at implementation; if the partner hook
  lacks it, fetch/derive the project list from the page's existing project
  selector source rather than dropping the prop.
