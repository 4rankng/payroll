---
phase: 3
title: "Local live verification with BUMHAN file"
status: pending
priority: P1
effort: "3h"
dependencies: [1, 2]
---

# Phase 3: Local live verification with BUMHAN file

## Overview

Prove the fix end-to-end on local dev (`make dev` stack: payroll-mysql :3306, payroll-redis :6379)
using the real partner file `/Users/dev/Documents/downloads/BCC BUMHAN Thang8.xlsx` — copies with
cells edited to 0. Test matrix is written to `plans/testplan/` BEFORE any upload runs (repo rule).

## Requirements

- Functional: matrix in `plans/testplan/260904-bcc-zero-cell-delete.md` executed in order, each
  step's expectation asserted against the `timesheets` table (not just the API response).
- Non-functional: no prod systems touched; local DB only; admin credentials from local seed.

## Architecture

1. Ensure backend running (air :8080) + migrations current. Start `make dev` if not up.
2. Find the BUMHAN project id + a real employee with pending rows in 2026-08 (import the ORIGINAL
   file first if local DB has no data for it).
3. Build modified copies with a script (python + openpyxl or Go excelize helper): zero one cell,
   zero a whole employee row, zero everything.
4. Login admin → `POST /api/v1/timesheets/partner-import` (multipart: file, project_id, for_month).
5. Assert DB state after each upload via `docker exec payroll-mysql mysql payroll_db`.

## Related Code Files

- Create: `plans/testplan/260904-bcc-zero-cell-delete.md` (matrix, BEFORE verification)
- Create: temp modified xlsx copies under `/tmp` (not committed)
- Read: `backend/internal/transport/http/handlers/timesheet/bcc_import_handler.go` (form fields)

## Implementation Steps

1. Write test matrix file.
2. Start backend; confirm health.
3. Baseline: import original file → note created pending rows (count, one sample key).
4. Upload copy with single cell → 0 → assert that row hard-deleted, others intact.
5. Upload copy with whole employee row → 0 → assert all their pending rows gone.
6. Upload copy with all cells 0 → assert no error, prior deletions persist.
7. Approve one row (DB or API), upload 0 for it → assert untouched + skipped count.
8. Blank-cell copy → assert rows untouched (blank ≠ 0).
9. `make api-test` regression suite.

## Success Criteria

- [x] Every matrix row passes with DB-level evidence (row counts before/after).
- [x] `make api-test` green (or deviations reported honestly).

## Risk Assessment

Risk: local DB lacks BUMHAN data / project — mitigate by importing the original file first (step 3);
it creates employees + pending rows (STK auto-create path). Signal: upload errors "nhân viên không
tìm thấy" → project_id mismatch → resolve via `projects` table by name. Response: pick the right
project_id and re-run matrix.
