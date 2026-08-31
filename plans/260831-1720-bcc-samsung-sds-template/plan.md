# BCC Samsung SDS template support

Date: 2026-08-31 · Status: in-progress · Mode: /ak-cook --auto

## Outcome

The weekly BCC upload must parse and import the Samsung SDS template
(`~/Downloads/Samsung.xlsx`, project 76 "Sam Sung SDS", 3 failed prod uploads
16:00–16:04 today) — timesheets created from its CB/OT columns, STK names read
from the correct column, 0 spurious errors.

## Diagnosis (evidence: scratch test on real file + code)

Four independent defects, all in the FormatLegacy path (sheet "BCC" →
DetectFormat → ParseBCCFile + ParseSTKSheet):

1. **STK hardcodes column positions.** Samsung STK layout is
   `TÊN | SỐ CCCD | TÊN NGÂN HÀNG | SỐ TÀI KHOẢN`; parser reads FullName from
   col C → employees named "MB BANK"/"VIETCOMBANK"/…; bank name col E empty;
   rows without bank info dropped (col C empty). BCC↔STK name cross-check then
   errors "CCCD … thuộc về MB BANK (theo STK)" → UI fallback
   "Không thể xử lý dòng dữ liệu này" on Dòng 12–15 (rowNum starts at 12).
   Prod bank data verified intact (name-mismatch validation rejected the
   garbage updates; no cleanup needed).
2. **firstDayColumn only accepts ints 1–31.** Aug 1–28 columns are real date
   serials (rejected); trailing stale ints 29/30/31 accepted → day map starts
   col 61 → real data cols (BA/BC/BE) get no day/shift → entries=0.
3. **No rate row** (Samsung carries no VND rates; row 9 is a stale weekday
   row). ShiftRates empty → every entry errors "không tìm thấy mức lương cho
   ca CB (0 VND)".
4. Labels can't be trusted for day type: row 9/10 weekday + CN labels follow a
   stale month pattern (CN on Aug 5 Wed). Only the date row (row 8) is truth.

## Changes

1. `excel/stk_parser.go` — header-label-driven column mapping (account, bank,
   cccd, name, mobile, note; specific labels claimed before generic "tên").
   Fall back to current hardcoded indexes when no header labels found.
2. `excel/bcc_parser.go` —
   - `firstDayColumn`/`buildDayColMap`: accept full date serials via existing
     `parseExcelDate` (year 2020–2040) → day = t.Day(); keep small-int path
     (regression-safe: TestParseBCCFile_DateSerialDayNumbers serials 22–28
     stay day-ints since year < 2020).
   - `buildBCCHeaderMap`: also match "chức danh" → deptCol.
3. `bcc_import_process.go` — rateless-file fallback: when
   `len(parsed.ShiftRates) == 0` and the rate lookup misses, derive
   hourType from the label (contains "OT" → "tăng ca", CB/CN/HC → "ca ngày")
   and dayType from the calendar (`determineDayType`). Files that carry any
   rates keep today's strict behavior unchanged.

## Constraints / non-goals

- No regression: EVA (in-file rates + day-int rows), lgd, multi-position,
  weekly-payment formats untouched. No DB migration. No frontend change.
- Not doing: rowNum display fix (pre-existing off-by-one), holiday detection,
  assignment-position change for rateless files (defaults "phổ thông" for new
  employees; existing Samsung employees unaffected).

## Acceptance

1. Unit tests: STK Samsung layout (names/banks correct, bank-less rows kept);
   BCC serial date columns → correct day numbers; label→hourType mapping;
   rateless fallback path.
2. Existing excel + services tests pass; `go build`/`go vet` clean.
3. E2E local dev: upload Samsung.xlsx to project 76, month 2026-08 →
   completed, 21 entries created (6 employees × Aug 24–27 CB 8h), 0 errors.
4. Commit + push, deploy prod, verify.

## Verification record

- Parser ground truth (scratch run pre-fix): STK 4 rows with FullName="MB
  BANK"/"VIETCOMBANK"/…; BCC 6 employees entries=0; day map startCol=col 61
  (stale tail ints 29/30/31). Post-fix scratch: STK 7 rows correct names/banks;
  6 employees, 20 entries days 24-27 CB 8h; dept="Chia chọn".
- Unit tests: TestParseSTKSheet_SamsungColumnLayout,
  TestParseSTKSheet_ClassicColumnLayout, TestParseBCCFile_SamsungSDSTemplate,
  TestParseBCCFile_DayIntegersNotSerials, TestShiftLabelHourType — all PASS.
  Full excel package PASS. Services package: only the 7 pre-existing
  date-drift failures (TestMaxCycleDay, TestCycleDay*,
  TestWalletDemandForecast*) — present on clean tree before this change.
- E2E local dev (asset 458, project 76, 2026-08): status=completed,
  created_count=20, error_count=0, error_detail NULL. Timesheets: Aug24 ×4,
  Aug25 ×6, Aug26 ×6, Aug27 ×4 — all 8.00h, paytype bucket
  "chia chọn.ngày thường.ca ngày" (position.dayType.hourType resolved
  correctly via label + calendar mapping).
- Prod data check (read-only): no bank-field corruption from the failed
  uploads (name-mismatch validation rejected the garbage updates); 3 failed
  import jobs today (assets 457/458/459 at 16:00-16:04 ICT).
- go build / go vet / gofmt clean on all changed files.

