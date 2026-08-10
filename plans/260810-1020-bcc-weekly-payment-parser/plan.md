---
title: "BCC Weekly Payment Parser (4th Template — Thai Binh Duong Format)"
description: "Add a 4th BCC parsing strategy for weekly payment files where sheet name is the salary prefix (520/700/750/800/900) and row 10 holds per-cell shift codes (HC/TCN/NN/TCNN). Combined key = prefix + shiftCode, looked up in the project payrate (single position)."
status: completed
priority: P2
branch: "main"
tags: [bcc, parsing, excel, weekly-payment, thai-binh-duong]
blockedBy: []
blocks: []
created: "2026-08-10T03:20:11.230Z"
createdBy: "ck:plan"
source: skill
---

# BCC Weekly Payment Parser (4th Template)

## Overview

Add a 4th BCC Excel parsing strategy for partner "Thái Bình Dương" weekly payment files. The template lays out the full month in a single wide row per employee, with per-cell shift codes — distinct from the existing 3 formats in two ways:

1. **Sheet name can be any value** - the detection relies on row 8/10 fingerprint, not sheet naming. In the sample file, sheets are named `520`, `700`, `750`, `800`, `900` (kVND per 8h), but any name works as long as the layout matches.
2. **Shift code is per-cell in row 10**, not per-sheet. Each date occupies a *pair* of columns — first cell = primary shift (`HC` for weekday / `NN` for weekend), second cell = overtime (`TCN` / `TCNN`). This means a single (employee, date) can produce 0, 1, or 2 timesheet entries depending on which cells are non-zero.

Rate lookup key = **`<sheetName><shiftCode>`** (e.g., `520HC`, `520TCN`, `520NN`, `520TCNN`). The project has a single position, so the existing position-agnostic `buildShiftRatesForShift` helper in `bcc_import_weekly.go` already supports this lookup pattern — we just feed it a different shift string per cell.

## File Layout (concrete — verified against `BẢNG CHẤM CÔNG THÁI BÌNH DƯƠNG THÁNG 08.2026.xlsx`)

```
Sheet name: "520" | "700" | "750" | "800" | "900"  (5 tiers in the sample)
Title cell: A5 = "BẢNG CHẤM CÔNG THÁNG MM/YYYY"    (informational, parser uses user forMonth)
Header row 8:    A8=STT B8=Mã nhân viên C8=Họ và tên D8=Bộ phận E8=Lương 8h
                 F8..BN8 = day numbers (serial 1..31, formatted 'dd' — see "Day number gotcha" below)
                 BP8 = "Tổng hợp" (summary, skipped)
                 BT8 = "Suất ăn tăng ca" (meal allowance, skipped)
Weekday row 9:   F9=T4  H9=T5  J9=T6  L9=T7  N9=CN  P9=T2  R9=T3  T9=T4  ... (Vietnamese T2–T7 = Mon–Sat, CN = Sun)
Shift row 10:    F10=HC G10=TCN H10=HC I10=TCN J10=HC K10=TCN L10=HC M10=TCN
                 N10=NN O10=TCNN P10=HC Q10=TCN R10=HC S10=TCN T10=HC ...
Data rows 11+:   A=STT B=CCCD C=Họ tên D=Bộ phận E=Lương 8h (tier VND)
                 F..BO = hours (numeric, can be empty / 0)
                 BP = "Tổng hợp" sum column (skip)
                 BT = meal overtime cells (skip)
                 Row 26 is "Tổng cộng" footer (stop)
Empty rows 18+ on every sheet in the sample — stop on 5 consecutive blank rows, same as WeeklyBCC parser.
```

### Rate key encoding (per cell)

| Cell column | Row 10 label | Combined key | Day type (from label) | Hour type |
|---|---|---|---|---|
| F (weekday primary) | `HC` | `<prefix>HC` (e.g. `520HC`) | `ngày thường` | `HC` |
| G (weekday overtime) | `TCN` | `<prefix>TCN` | `ngày thường` | `TCN` |
| N (weekend primary) | `NN` | `<prefix>NN` | `ngày nghỉ` | `NN` |
| O (weekend overtime) | `TCNN` | `<prefix>TCNN` | `ngày nghỉ` | `TCNN` |

### Day number gotcha (verified — the partner's template is buggy)

Row 8 cell values in the sample (read from raw XML `<v>...</v>`) are NOT always `1, 2, 3, ..., 31`. Sheet `520` has values `1, 2, 3, 1, 2, 3, 4, 5, ..., 28` — the counter *restarts* at the start of the second "week group" because the partner's copy-pasted template formula was broken. Sheet `900` happens to be `1, 2, 3, ..., 31` because it starts on Saturday so the counter doesn't visibly clash with the displayed day.

**Decision: derive day from column position + first weekday in file, not from row 8 cell value.**

- For sheet 520: first weekday is T4 (Wed). If `forMonth = 2026-07`, then July 1 2026 is Wed = day 1. F=day 1, H=day 2, J=day 3, L=day 4 (not day 1 as the broken cell claims), N=day 5, ..., BN=day 28.
- For sheet 900: first weekday is T7 (Sat). July 1 is Wed, so first column F = day 4, H = day 5, ..., BN = day 31.
- This matches the partner's *intent* (weekday labels in row 9 are continuous and correct across all 5 sheets) and avoids the duplicate-day problem on sheet 520.

The first weekday in the file is read from row 9 (leftmost non-empty cell). The mapping uses a `weekday → day-of-month` table built from the user-supplied `forMonth`:

```
ForMonth: "2026-07" → 2026-07-01 is a Wednesday (T4)
First weekday in sheet 520 = T4 → offset = 0 → F=day 1
First weekday in sheet 900 = T7 → offset = 3 → F=day 4
```

If the title text and the first weekday cannot be reconciled with the user-supplied `forMonth` (e.g., the partner's file says "Tháng 02/2026" but the first weekday is T4 and the user uploaded for July), surface this as a clear import error rather than silently producing wrong days.

## Format priority order

| Priority | Format | Sheet signal |
|---|---|---|
| 1 | `FormatLegacy` | sheet named exactly `BCC` (wins tiebreaker) |
| 2 | `FormatWeeklyBCC` | sheet name starts with `BCC-` |
| 3 | **`FormatWeeklyPayment`** (NEW) | row 8 has `STT`+`Mã nhân viên`+`Họ và tên`+`Bộ phận`+`Lương 8h` AND row 10 has any of `HC/TCN/NN/TCNN` (sheet name can be any value) |
| 4 | `FormatMultiPosition` | row 4 has `STT`+`Mã nhân viên`+`Họ và tên` |

Sheets with row 4 headers fall through to multi-position detection. The row-based signature (row 8 vs row 4) keeps the two formats disjoint.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Research](./phase-01-research.md) | completed (verified against real file) |
| 2 | [Implementation](./phase-02-implementation.md) | completed ✅ |
| 3 | [Testing](./phase-03-testing.md) | pending (requires Go environment) |

## Files to Create

- `backend/internal/app/services/excel/weekly_payment_parser.go` — format constant `FormatWeeklyPayment`, `ParseWeeklyPaymentFile`, header/employee parsers, day-from-weekday resolver, rate-key builder
- `backend/internal/app/services/excel/weekly_payment_parser_test.go` — unit tests (synthesized file + fixture from the attached xlsx)
- `backend/tests/fixtures/bcc/thai_binh_duong.xlsx` — copy of the attached sample (use the original to avoid dependency on the user's drive)

## Files to Modify

- `backend/internal/app/services/excel/format_detector.go` — add `FormatWeeklyPayment` constant + `isWeeklyPaymentSheet()` check
- `backend/internal/app/services/excel/format_detector_test.go` — add `TestDetectFormat_WeeklyPayment` cases
- `backend/internal/app/services/excel/stk_parser.go` — already handles 4-col STK (`STT | Mã NV | Tên | STK | Ngân hàng`); the new template's STK sheet has the same layout — no change needed
- `backend/internal/app/services/bcc_import_service.go` — add `case excelparser.FormatWeeklyPayment: → processWeeklyPaymentUpload(...)` in `processAssetData`
- `backend/internal/app/services/bcc_import_weekly.go` — add `processWeeklyPaymentUpload()` (mirrors `processWeeklyBCCUpload` but takes one prefix per sheet and reads shift type from row 10 cells instead of sheet name)
- `backend/internal/app/services/bcc_import_weekly_test.go` — add payrate/rate-key helper tests
- `backend/tests/integration/flow_bcc_weekly_payment_import.go` — new end-to-end flow (mirror of `flow_bcc_weekly_import.go` but using the Thai Binh Duong fixture)
- `backend/tests/integration/main.go` — register the new flow

## Files to NOT Touch (verified compatible)

- `internal/app/services/excel/stk_parser.go` — already auto-detects the 5-column STK layout used by this template
- `internal/app/services/bcc_import_helpers.go` — `applySTKBankFields`, `buildSTKBankUpdates`, `findSTKRow` all work on `STKRow` regardless of source format
- `internal/app/services/employee/bank*.go` — same bank name resolution pipeline
- `internal/app/services/timesheet/service.go` — `BulkCreateTimesheetsInTransaction` is format-agnostic

## Acceptance Criteria

- [ ] `FormatWeeklyPayment` is detected when row 8 has the 5-column header AND row 10 contains `HC`/`TCN`/`NN`/`TCNN` (sheet name can be any value)
- [ ] Sheets with row 4 headers (not row 8) fall through to `FormatMultiPosition` (no false positives)
- [ ] Each non-zero cell in rows 11+ becomes exactly one timesheet entry
- [ ] Cell (col, row) is mapped to the correct date: `dayOfMonth = firstColumnDay + (pairIndex)`, where `firstColumnDay` is derived from the user-supplied `forMonth` and the first non-empty weekday in row 9
- [ ] Rate lookup uses key `<sheetName><row10Label>` and reuses `buildShiftRatesForShift`
- [ ] Day type is derived from the row 10 label: `HC/TCN → "ngày thường"`, `NN/TCNN → "ngày nghỉ"` (label takes priority over weekday — matches the existing `determineDayType` for the same shifts)
- [ ] Hour type column on `BulkCreateTimesheetEntry` is the bare shift code (e.g., `HC`), matching the WeeklyBCC convention
- [ ] Sheet 520 sample (with broken day-number cells) produces 1–31 unique days, not duplicates on days 1/2/3
- [ ] Sheet 900 sample (where the first weekday is Saturday) correctly offsets to start at day 4 of July 2026
- [ ] STK sheet's bank/mobile fields are persisted to auto-created employees (same as WeeklyBCC)
- [ ] STK ↔ BCC name cross-check fires on CCCD mismatch (same as other formats)
- [ ] `make api-test` green; `go test ./... -v -race -cover` green for `services` and `excel` packages
- [ ] `pnpm lint && pnpm type-check` green (no frontend changes)
- [ ] No regression in the existing 3 formats — the new branch in `DetectFormat` runs *before* the multi-position sheet-name heuristic and short-circuits

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Partner renames a sheet to something non-numeric | Detection is signature-based (row 8 layout + row 10 shifts), not just name. If signature doesn't match, treat as multi-position |
| Partner uses different row 10 labels (e.g., `HC1`/`TCN1`/`NN1`/`TCNN1`) | Parser only emits entries for cells whose row 10 label appears in `getShiftTypes(flatRates)`. Unknown labels are silently dropped from the output and logged at `info` level so the partner can see them in the import result |
| Rate for a key is 0 in the config | Reuse `buildShiftRatesForShift` which already skips zero rates → cell becomes "no matching rate" import error (same UX as WeeklyBCC) |
| File is for a different month than `forMonth` (partner re-uses an old file) | Day-from-weekday resolver catches the mismatch: if the first weekday in the file doesn't match what `forMonth` predicts for day 1, fail with `"ngày đầu tiên trong file (T4) không khớp với Tháng 2026-07 (T4 Thứ Tư) — kiểm tra lại tháng đã chọn"` |
| Tổng hợp (summary) column at BP8 is misread as a day | Stop column is determined by reading `Tổng hợp` text or by hitting col 73 (BU) max — same pattern as `isProjectColumn` in weekly_bcc_parser.go |
| Hidden columns break the column count | `GetColVisible` check (same as `weekly_bcc_parser.go:134`) before treating a column as a date column |
