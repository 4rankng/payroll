---
title: "Verify and ship"
status: todo
priority: P1
effort: "2h"
dependencies: [phase-02-frontend-filter-row-grouped-history-view]
---

# Phase 3: Verify and ship

## Overview

Prove the feature end-to-end on the dev stack for BOTH roles, run the
regression suite, and land one focused commit. The working tree is clean as
of 2026-08-31 evening (the morning's BCC username fix and Samsung SDS work
are committed and deployed), so no commit-sequencing constraints remain —
but the explicit-file-list discipline stays.

## Requirements

- Functional: all plan-level success criteria demonstrated with evidence
  (command output or screenshots), not assertions.
- Functional: verification covers the trust-boundary behaviors added by this
  feature — search boundary, wildcard-literal matching, partner scoping — not
  just admin happy paths.
- Compatibility: `make api-test` regression suite passes.
- Process: one focused conventional commit.

## Implementation Steps

1. Backend (scoped — the full `./internal/...` run has 7 pre-existing
   date-drift failures in `services` documented on a clean tree
   (`TestMaxCycleDay`, `TestCycleDay*`, `TestWalletDemandForecast*`,
   see plans/260831-1720-bcc-samsung-sds-template/plan.md); they are out of
   scope — do not investigate, do not fix here):
   ```bash
   cd backend && go build ./... \
     && go test ./internal/infra/persistence/... ./internal/transport/http/handlers/timesheet/... -count=1
   ```
2. Frontend: `cd frontend && pnpm type-check && pnpm lint`.
3. Manual QA on the dev stack (`make dev`; QA creds localhost:3000 — admin
   `frankng`, partner `cuongnv`, pw `Admin123`. Local only; never prod UI
   login):
   - ADMIN: search "georim" (lowercase) → only matching uploads; pagination
     totals shrink. Clear → all-projects grouped view with headers.
   - ADMIN: dropdown → single project, flat newest-first.
   - ADMIN: search `100%` and `data_8` → literal matches only, no wildcard
     expansion.
   - PARTNER (cuongnv): history shows the dropdown AND named group headers;
     list contains only that partner's uploads; search works; no
     cross-partner rows.
   - Boundary: 101-character search rejected with 400 (via curl — UI clamps);
     100 accepted.
   - Open from a project page → dropdown hidden, list scoped.
   - Close/reopen sheet → filters cleared.
   - Download + error-details expand still work.
4. `make api-test` — full regression (note: the suite has no GET coverage of
   the list endpoint; the curls above are the real gate for the new SQL).
5. Commit ONLY this feature's files (explicit list, then `git show --stat
   HEAD` to verify):
   `feat(timesheet): search and project grouping in BCC upload history`
   Files: `backend/internal/domain/asset.go`,
   `backend/internal/infra/persistence/asset_repository.go`,
   `backend/internal/transport/http/handlers/timesheet/bcc_import_handler.go`
   (+ any new escape-helper test file),
   `frontend/src/types/api/timesheet.types.ts`,
   `frontend/src/components/timesheet/UploadHistorySheet.tsx`,
   `frontend/src/pages/partner/TimesheetsPage/index.tsx`,
   `frontend/src/pages/mobile/partner/TimesheetsPage/index.tsx`.
   The plan directory ships as a separate `docs(plans)` commit.

## Success Criteria

- [x] Scoped backend build + tests green (5 packages, 0 failures; pre-existing
      services failures not run).
- [x] `pnpm lint` (eslint + `tsc --noEmit`) green.
- [x] API-level QA for BOTH roles on the live dev backend: admin search
      (lowercase georim → 7 rows), wildcard literals, `sort=project`
      ordering, 100/101 boundary; partner `cuongnv` scoped to own 70 uploads
      (single `uploaded_by`), search stays scoped.
- [x] `make api-test` passes: 296 total, 273 passed, 0 failed, 23 skipped.
- [ ] Browser visual pass of the sheet (grouped headers, dropdown, filter
      clear-on-close) — deferred to user; every API behavior it relies on is
      proven, diff reviewed, types green.
- [ ] Single focused feature commit + separate plan commit; `git show --stat`
      verified for both.

## Risk Assessment

- Pre-existing red tests could tempt scope-widening — mitigated by the
  pre-declared exclusion list above.
- Deploy is out of scope for this phase; user decides when to ship
  (babysit-to-completion applies when they do).
