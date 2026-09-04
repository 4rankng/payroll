---
title: "BCC zero cell deletes pending timesheet"
description: "BCC upload with an explicit 0 in a cell must delete the existing chờ duyệt (pending_approval) timesheet row for that employee/date/shift, instead of being silently dropped at parse time."
status: pending
priority: P1
effort: "1d"
tags: [bcc, timesheet, import]
created: 2026-09-04
---

# BCC zero cell deletes pending timesheet

## Overview

Partner uploads a BCC workbook (BUMHAN Tháng 8 file, date-row format) to fix timesheets. Today a
non-zero cell overwrites old data (upsert by employee/date/paytype), but an explicit **0** cell is
dropped by the parser exactly like a blank cell — so the old timesheet row survives and bulk-removing
công requires slow manual clicking. Goal: **0 in a cell = delete the existing pending ("chờ duyệt")
timesheet row** for that key; approved/paid/transaction-linked rows stay protected (skip, never
delete). Blank cells keep today's meaning: no information, row untouched.

## Root Cause (verified by source)

The import pipeline ALREADY supports zero semantics end-to-end:

- `planBCCReplacement` (`bcc_import_replacement.go:44`) hard-deletes existing rows whose key is
  requested by the file — but only when status `pending_approval` and not paid/transaction-linked.
- `BulkCreateTimesheets` (`timesheet_domain_service_bulk_create.go:349,464`) treats
  `HoursWorked == 0` as delete-intent: existing row → DELETE path; missing row → silently skipped.

The gap is upstream: all four BCC parsers drop explicit `0` cells at parse time, so a zero never
becomes an entry and its key is never "requested":

- `excel/bcc_parser.go:430` — `val == "0"` / `hours == 0` → `continue`
- `excel/date_row_bcc_parser.go:143` — `dateRowNumericCell` returns 0 for both blank and explicit 0
- `excel/weekly_bcc_parser.go:237` — same pattern
- `excel/weekly_payment_parser.go:340` — same pattern

Secondary symptom: a file where every in-month cell is 0 parses to zero entries →
"không có dữ liệu hợp lệ để tạo bảng chấm công" error (the "lỗi" the user reported).

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Explicit `0` cell in BCC → delete existing pending timesheet row for that (employee, date, shift) | P1 |
| 2 | Approved / paid / transaction-linked rows are never deleted by a 0 — skipped + counted (existing protection) | P1 |
| 3 | Blank cell behavior unchanged (no entry, existing row untouched) | P1 |
| 4 | Works across all 4 BCC formats: legacy, date-row (BUMHAN), weekly BCC, weekly payment | P1 |
| 5 | All-zero upload no longer errors when it deleted pending rows | P1 |

## Non-Goals

- No DB migration (code-level fix only) — per repo convention.
- No deletion of approved rows (user explicitly scoped to "điều kiện chờ duyệt").
- No frontend changes required (result counts surface via existing import stats).
- Blank ≠ 0 semantics stay distinct; no UI toggle.

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Parsers keep explicit zero cells](./phase-01-parsers-keep-zero.md) | Pending |
| 2 | [Pipeline accounting + unit tests](./phase-02-pipeline-tests.md) | Pending |
| 3 | [Local live verification with BUMHAN file](./phase-03-live-verification.md) | Pending |

## Success Criteria

- [ ] Re-upload BUMHAN Tháng 8 copy with a 0 in a previously non-zero cell → that pending row is
      gone from `timesheets` (hard-deleted), same row not recreated.
- [ ] Non-zero cell in same upload still overwrites (regression guard).
- [ ] Approved/paid row with a 0 cell → row untouched, counted in skipped.
- [ ] Blank cell → row untouched.
- [ ] `go test ./internal/app/services/... ./internal/app/services/excel/...` green.
- [ ] `make api-test` regression suite green (or deviations reported).
- [ ] All-zero employee row → all that employee's pending rows for the month deleted, no error.

## Key Files

- Modify: `backend/internal/app/services/excel/{bcc_parser,date_row_bcc_parser,weekly_bcc_parser,weekly_payment_parser}.go`
- Modify: `backend/internal/app/services/{bcc_import_process,bcc_import_weekly,bcc_import_multi_position}.go` (zero-row skipped-count accounting)
- No change needed: `bcc_import_replacement.go`, `timesheet_domain_service_bulk_create.go`

## Risks

| Risk | Mitigation |
|------|------------|
| Draft/placeholder rows (blank STT) that contain stray 0s now count as employees | Draft-row filter counts NON-ZERO entries only (`bcc_parser.go:443`, weekly `:255`, `:352`) |
| Zero entry with unmapped shift label errors the row ("không tìm thấy mức lương") | Same behavior as non-zero unmapped label — acceptable, existing error surface |
| Duplicate key (0 + non-zero same key in one file) fails whole replacement tx | Pre-existing constraint for duplicate non-zeros; not a regression |
| Phantom data from templates' summary rows | Totals/"Cộng" row filters already run before entry scan |

## Test Evidence

Phase 3 writes `plans/testplan/260904-bcc-zero-cell-delete.md` BEFORE live verification (repo rule).
Live test uses `/Users/dev/Downloads/BCC BUMHAN Thang8.xlsx` (copies modified locally).
