---
phase: 3
title: "Testing"
status: completed
priority: P2
dependencies: [2]
---

# Phase 3: Testing

## Overview

Lock the parser, the day-from-weekday resolver, and the end-to-end import with a mix of synthesized unit tests, the real Thai Binh Duong fixture, and an integration flow that exercises the new `FormatWeeklyPayment` branch.

## Test layers

### Layer 1 — Format detector (`format_detector_test.go`)

Add 4 cases that pin the detection signature:

| Case | Input | Expected |
|---|---|---|
| `TestDetectFormat_WeeklyPayment_ThaiBinhDuong` | real fixture, all 5 numeric sheets | `FormatWeeklyPayment`, `WeeklyPaymentSheets = ["520","700","750","800","900"]` |
| `TestDetectFormat_WeeklyPayment_SingleSheet` | synthesized file with one `520` sheet + headers | `FormatWeeklyPayment`, single sheet |
| `TestDetectFormat_WeeklyPayment_NumericButWrongLayout` | synthesized `520` sheet with row 4 headers (not row 8) | falls through to `FormatMultiPosition` |
| `TestDetectFormat_WeeklyPayment_NonNumericName` | synthesized sheet `Phổ thông` with row 8/10 fingerprint | falls through to `FormatMultiPosition` |

### Layer 2 — Parser unit tests (`weekly_payment_parser_test.go`)

| Case | What it pins |
|---|---|
| `TestParseWeeklyPaymentFile_FixtureSheet520` | Use the real Thai Binh Duong file. Assert: 5 sheets parsed, sheet 520 has ≥5 employees, first employee `Phạm Văn Duy` has CCCD `031092018201`, day 4 (L11) has shift `520HC` and 8h, day 6 (P11) has shift `520HC` and 4h. **No duplicate days despite partner's broken cell values.** |
| `TestParseWeeklyPaymentFile_FixtureSheet900` | Same fixture, sheet 900. First column F should be day 4 (not day 1), last column BN should be day 31. |
| `TestParseWeeklyPaymentFile_ForMonthMismatch` | Synthesize a file whose first weekday is T7 (Sat) but call with `forMonth=2026-02` (Feb 1 is Sun). Assert error message about weekday/month mismatch. |
| `TestBuildWeeklyPaymentShiftRow` | Synthesize a sheet with row 10 = `["", "", "", "", "", "HC", "TCN", "NN", "TCNN", ...]`. Assert map keys (1-based col) and values. |
| `TestDayFromColumnIndex` | Table-driven: `firstColDay=4, firstShiftCol=6` → col 6 = day 4, col 8 = day 5, col 10 = day 6. |
| `TestRateKeyFor` | `("520", "HC")` → `"520HC"`; `("700", "TCN")` → `"700TCN"`. |
| `TestDayTypeFor` | `HC,TCN → "ngày thường"`; `NN,TCNN → "ngày nghỉ"`; `unknown → "ngày thường"` (fallback). |
| `TestParseWeeklyPaymentFile_HiddenColumnsSkipped` | Hide col 8 (H) in the fixture, assert no entries fall on day 2. |
| `TestParseWeeklyPaymentFile_TongHopColumnSkipped` | Confirm BP8 "Tổng hợp" is treated as stop col, not as a day cell. |
| `TestParseWeeklyPaymentFile_SummaryRowBreaks` | Confirm row 26 ("Tổng cộng") terminates the employee loop, like in other parsers. |

### Layer 3 — Rate lookup integration (`bcc_import_weekly_test.go`)

Add tests that prove the new flow correctly resolves a `520HC` key against a synthetic payrate:

| Case | What it pins |
|---|---|
| `TestBuildShiftRatesForShift_PrefixedKey` | flatRates with key `phổ thông.ngày thường.520HC → 65000`; `buildShiftRatesForShift(fr, "520HC")` returns the expected rate. |
| `TestBuildShiftRatesForShift_PrefixedKey_WeekendVariant` | flatRates with `phổ thông.ngày nghỉ.520NN → 130000`; `buildShiftRatesForShift(fr, "520NN")` returns 130000. |
| `TestDayTypeFor_ShiftLabelDrivesLookup` | End-to-end: given a payrate, iterate over (cell, row10Label) pairs from the fixture and assert each (employee, date, shift) → (rate, dayType) tuple resolves correctly. |

### Layer 4 — End-to-end integration (`flow_bcc_weekly_payment_import.go`)

Mirror `flow_bcc_weekly_import.go` (`backend/tests/integration/`) but use the new fixture. Concrete scenarios:

| Test | What it pins |
|---|---|
| `flowWeeklyPayment: Upload Thai Binh Duong file` | Partner uploads the sample, status becomes `completed`, 0 errors, expected created count. |
| `flowWeeklyPayment: Per-tier rate resolution` | Verify that one employee in sheet `520` with 8h on a Wednesday gets a timesheet entry with `HourType=HC`, `DayType="ngày thường"`, `HoursWorked=8`. |
| `flowWeeklyPayment: Per-tier weekend rate` | Verify that the same employee with 8h on a Sunday gets a timesheet entry with `HourType=NN`, `DayType="ngày nghỉ"`. |
| `flowWeeklyPayment: STK auto-creates employees` | Employees present in BCC but missing from project get auto-created with bank info from STK. |
| `flowWeeklyPayment: Sheet 520 day uniqueness` | Query the timesheet table for the test project; assert no two entries for the same (employee, date) from the sheet 520 source. |
| `flowWeeklyPayment: Re-upload (latest wins)` | Re-upload the same file; assert stale timesheets are replaced, no duplicates remain. |
| `flowWeeklyPayment: Import with wrong forMonth` | Upload the fixture for `2026-02` instead of `2026-07`; assert the import fails with a clear "weekday/month mismatch" error. |

### Layer 5 — Regression safety

Run the existing suites unchanged and assert they still pass:

- `cd backend && go test ./internal/app/services/excel/... -v -race -cover` (covers all 3 existing parsers + the new one)
- `cd backend && go test ./internal/app/services/... -v -race -cover` (covers the upload flows)
- `cd backend && go test ./tests/integration/...` (after `make db` is up; the existing `flow_bcc_import` and `flow_bcc_weekly_import` flows must still pass with the updated `DetectFormat`)
- `cd frontend && pnpm lint && pnpm type-check` (no frontend change, but verify nothing breaks)

## Test fixtures

- `backend/tests/fixtures/bcc/thai_binh_duong.xlsx` — copy of the attached sample. Located here so the parser test does not depend on the user's drive attachment.
- Synthesized in-memory files (via `excelize.NewFile()`) for negative cases — same pattern as `weekly_bcc_parser_test.go` and `format_detector_test.go`.

## Test gating (Definition of Done)

- [ ] All new unit tests pass
- [ ] `go test ./... -race -cover` ≥ 80% coverage on the new `weekly_payment_parser.go`
- [ ] Integration flow `flow_bcc_weekly_payment_import` passes against a live dev DB
- [ ] `make api-test` green end-to-end
- [ ] No regression in the existing 3 formats (legacy, multi-position, weekly BCC)
- [ ] No regression in the existing STK parser or payrate resolution

## Risks specific to testing

| Risk | Mitigation |
|---|---|
| `make api-test` requires live MySQL/Redis; not always available locally | Layer 1–3 tests are pure unit tests and can run in any environment. Layer 4 only runs when `make db` is up; CI handles it. |
| The fixture has 11 employees × 5 sheets × ~28 day-pair cells = ~1500 cells; tests get slow | Cap column scan to first 200 (same as `weekly_bcc_parser.go`'s `wbccMaxColScan`). The real fixture finishes in < 100ms. |
| The "Tổng hợp" column at BP8 might be misread as day 28 if the loop doesn't stop | Add `TestParseWeeklyPaymentFile_TongHopColumnSkipped` as an explicit guard. |
| Sheet 700/800 have only 1-2 employees in the fixture — easy to over-assert | Use `len(emp.Entries) >= 1` and `assert.NotEmpty` patterns instead of exact counts. |
