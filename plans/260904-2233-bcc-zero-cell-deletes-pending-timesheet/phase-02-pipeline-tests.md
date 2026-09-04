---
phase: 2
title: "Pipeline accounting + unit tests"
status: pending
priority: P1
effort: "2h"
dependencies: [1]
---

# Phase 2: Pipeline accounting + unit tests

## Overview

The shared pipeline needs no behavioral change — zero entries already flow: requested-key marks
pending rows stale → `HardDelete` in `applyTimesheetReplacement` → `BulkCreateTimesheets` silently
skips the non-existent zero row. Only result accounting changes: count zero-hour entries in
`SkippedCount` so the uploader sees how many rows were consumed as deletions instead of creations.

## Requirements

- Functional: zero-hour entries in the final (post-plan) entry list increment `SkippedCount`
  in all four process paths (legacy/date-row shared, weekly BCC, weekly payment, multi-position).
- Functional: all-zero upload that deleted ≥1 pending row completes with status "completed",
  not "không có dữ liệu hợp lệ" (entries non-empty because plan keeps zero entries; verify).
- Non-functional: no change to protection semantics (approved/paid/transaction rows skipped).

## Architecture

Add a small helper `countZeroHourEntries(entries)` next to `planBCCReplacement` and use it in the
`skippedCount :=` expressions of `bcc_import_process.go`, `bcc_import_weekly.go` (×2 paths),
`bcc_import_multi_position.go`. Unit-test `planBCCReplacement` with zero-hour entries: pending →
stale collected + entry kept; approved → protected skip; no existing → entry kept (bulk create no-op).

## Related Code Files

- Modify: `backend/internal/app/services/bcc_import_replacement.go` (helper only)
- Modify: `backend/internal/app/services/bcc_import_process.go` (~L577 skippedCount)
- Modify: `backend/internal/app/services/bcc_import_weekly.go` (both stats blocks)
- Modify: `backend/internal/app/services/bcc_import_multi_position.go` (stats block)
- Modify: `backend/internal/app/services/bcc_import_service_test.go` (zero-entry plan tests)

## Implementation Steps

1. Add `countZeroHourEntries` helper.
2. Wire into the 4 skippedCount expressions.
3. Extend `bcc_import_service_test.go`: zero-entry plan cases (pending/approved/missing).
4. `go build ./... && go test ./internal/app/services/... ./internal/app/services/excel/...`.

## Success Criteria

- [x] Unit tests prove: zero + pending → staleID; zero + approved → protectedSkipped; zero +
      missing → no stale, entry survives plan (silently skipped later).
- [x] Full backend build + services tests green.

## Risk Assessment

Risk: an all-zero file whose rows are ALL protected (approved) → entries empty after plan →
existing "protected skip" completion path handles it (`completeSkippedBCCImport`) — verify with a
test; if it errors instead, that path needs the same guard as non-zero uploads (it already has one).
