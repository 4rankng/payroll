---
phase: 1
title: "Parsers keep explicit zero cells"
status: pending
priority: P1
effort: "3h"
dependencies: []
---

# Phase 1: Parsers keep explicit zero cells

## Overview

Make all four BCC parsers emit an entry with `Hours: 0` when a data cell explicitly contains 0,
while blank cells continue to produce no entry. This is the entire root-cause fix; the downstream
pipeline already treats `HoursWorked == 0` as delete-intent.

## Requirements

- Functional: explicit `0` (numeric cell storing 0) → `BCCEntryData{Hours: 0}` appended.
- Functional: blank/empty cell → no entry (unchanged).
- Functional: non-numeric garbage cell → no entry (unchanged).
- Non-functional: draft-row filters (blank STT/code + no entries) must count only non-zero entries,
  so placeholder rows whose cells are all 0 or blank stay filtered out.

## Architecture

Each parser reads day-region cells with `GetCellValue(..., RawCellValue: true)`. Change the
skip conditions so `val == "0"` (or parses to 0.0) no longer `continue`s but appends a zero-hour
entry carrying the same DayNum/ShiftLabel/FullDate as a non-zero entry would. The label/date
resolution is column-based, so zero entries resolve payrate targets identically to non-zero ones.

## Related Code Files

- Modify: `backend/internal/app/services/excel/bcc_parser.go` (`parseEmployees` loop, ~L420-445)
- Modify: `backend/internal/app/services/excel/date_row_bcc_parser.go`
  (`dateRowNumericCell` → also report cell presence; entry loop ~L132-154)
- Modify: `backend/internal/app/services/excel/weekly_bcc_parser.go` (~L228-256)
- Modify: `backend/internal/app/services/excel/weekly_payment_parser.go` (~L331-357)
- Create: zero-cell cases appended to existing parser test files in the same package

## Implementation Steps

1. `bcc_parser.go`: in `parseEmployees`, split "blank" (`val == ""`) from "explicit zero"
   (`val == "0"` or `ParseFloat == 0`); append `{Hours: 0}` entry for the latter. Keep the
   draft-row filter but count entries with `Hours > 0` only.
2. `date_row_bcc_parser.go`: change `dateRowNumericCell` to return `(float64, bool)` (value, cell
   present) or add presence check in the loop; keep blank → skip, explicit 0 → zero entry.
3. `weekly_bcc_parser.go` + `weekly_payment_parser.go`: same blank-vs-zero split; draft filter
   (`empCode == "" && no non-zero entries`) counts non-zero entries only.
4. Add unit tests per parser: explicit `0` → entry `Hours: 0`; blank → no entry; all-zero row keeps
   employee (when STT/code present); placeholder row (blank STT + zeros) still filtered.

## Success Criteria

- [x] `go test ./internal/app/services/excel/...` green with new zero-cell cases.
- [x] Existing parser tests unchanged (except genuinely affected expectations).

## Risk Assessment

Risk: numeric-format cells where Excel stores "0.00" raw — `ParseFloat("0.00") == 0` → treated as
explicit zero, which is correct. Signal if wrong: legacy tests constructing "0"-formatted cells fail.
Response: treat only parseable numerics as explicit, everything else stays skipped.
