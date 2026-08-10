---
phase: 2
title: "Implementation"
status: completed
priority: P2
dependencies: [1]
---

# Phase 2: Implementation

## Overview

Wire `FormatWeeklyPayment` into the detection/parse/upload pipeline so the new template behaves like the other 3 formats end-to-end (accept → enqueue → parse → import → commit).

## Step-by-step

### 2.1 — Add the format constant + detector

`backend/internal/app/services/excel/format_detector.go`

```go
const (
    FormatLegacy BCCFormat = iota
    FormatMultiPosition
    FormatWeeklyBCC
    FormatWeeklyPayment  // NEW
)

type FormatDetectionResult struct {
    Format                 BCCFormat
    PositionSheets         []string
    WeeklyBCCSheets        []string
    WeeklyPaymentSheets    []string // NEW
}

// In DetectFormat's per-sheet loop, before the isPositionSheet fallback:
if isWeeklyPaymentSheet(f, sheetName) {
    weeklyPaymentSheets = append(weeklyPaymentSheets, sheetName)
    continue
}

// New: dedicated check (row 8 fingerprint + row 10 shift label + numeric name)
func isWeeklyPaymentSheet(f *excelize.File, sheetName string) bool {
    if !isNumericSheetName(sheetName) {
        return false
    }
    return hasWeeklyPaymentHeader(f, sheetName) && hasWeeklyPaymentShiftRow(f, sheetName)
}
```

`hasWeeklyPaymentHeader` reads row 8 (index 7) and verifies `STT`+`Mã nhân viên`+`Họ và tên`+`Bộ phận`+`Lương 8h` are all present (same trim/contains rules as `isPositionSheet` but pointed at row 8). `hasWeeklyPaymentShiftRow` reads row 10 (index 9) and checks for any of `HC`, `TCN`, `NN`, `TCNN` (substring match, case-insensitive — partners may use lowercase variants).

Priority update in `DetectFormat`:
```go
if len(weeklyBCCSheets) > 0 { return ...FormatWeeklyBCC }
if len(weeklyPaymentSheets) > 0 { return ...FormatWeeklyPayment } // NEW, before multi-position
if len(positionSheets) > 0     { return ...FormatMultiPosition }
```

### 2.2 — Build the parser

`backend/internal/app/services/excel/weekly_payment_parser.go` (new file, ~250 LOC)

Public types:
```go
type WeeklyPaymentImportData struct {
    Sheets []WeeklyPaymentSheetData
}

type WeeklyPaymentSheetData struct {
    Prefix    string                          // "520", "700", ...
    ShiftRows WeeklyPaymentShiftRow           // row 10 cells (col index → shift code)
    Employees []WeeklyPaymentEmployeeData
}

type WeeklyPaymentShiftRow struct {
    ByCol map[int]string // 1-based col → "HC" | "TCN" | "NN" | "TCNN" | ...
}

type WeeklyPaymentEmployeeData struct {
    EmployeeCode string // col B = CCCD
    FullName     string // col C
    Department   string // col D
    SalaryTier   int    // col E = "Lương 8h" (informational; not used for rate lookup)
    Entries      []WeeklyPaymentEntryData
}

type WeeklyPaymentEntryData struct {
    Day      int       // 1..31, derived from column position (NOT from row 8 cell value)
    ShiftKey string    // e.g. "520HC" — sheet prefix + row 10 label, used to look up rate
    Hours    float64
}
```

Public functions:
- `ParseWeeklyPaymentFile(f *excelize.File, sheetNames []string, forMonth string) (*WeeklyPaymentImportData, error)` — iterates sheets, calls `parseWeeklyPaymentSheet` for each
- `ExtractWeekdayFromR9(f *excelize.File, sheet string) (string, error)` — first non-empty cell in row 9 (case-insensitive, trim) used as the starting weekday

Helpers (private):
- `parseWeeklyPaymentSheet(f, sheetName, firstColumnDay int) (*WeeklyPaymentSheetData, error)` — entry point for one sheet
- `buildWeeklyPaymentShiftRow(f, sheet string) (map[int]string, int, error)` — reads row 10 cells, returns `col→shift` map and the first col index (1-based) that has a shift label
- `buildWeeklyPaymentEmployeeRows(f, sheet string, firstShiftCol int, firstColumnDay int) ([]WeeklyPaymentEmployeeData, error)` — reads rows 11+
- `dayFromColumnIndex(firstColDay, firstShiftCol, currentCol int) int` — `firstColDay + (currentCol - firstShiftCol) / 2`
- `rateKeyFor(prefix, shiftCode string) string` — `prefix + shiftCode`
- `dayTypeFor(shiftCode string) string` — `HC|TCN → "ngày thường"`, `NN|TCNN → "ngày nghỉ"`, fallback to weekday via `determineDayType` (already in `bcc_import_weekly.go`)

### 2.3 — Wire into the import service

`backend/internal/app/services/bcc_import_service.go` (in `processAssetData`):

```go
switch formatResult.Format {
case excelparser.FormatMultiPosition:
    return s.processMultiPositionUpload(...)
case excelparser.FormatWeeklyBCC:
    return s.processWeeklyBCCUpload(...)
case excelparser.FormatWeeklyPayment:  // NEW
    return s.processWeeklyPaymentUpload(...)
default:
    // legacy
}
```

`backend/internal/app/services/bcc_import_weekly.go` (new method, mirrors `processWeeklyBCCUpload`):

```go
func (s *BCCImportService) processWeeklyPaymentUpload(
    ctx context.Context,
    xf *excelize.File,
    formatResult *excelparser.FormatDetectionResult,
    filename string,
    projectID uint,
    uploaderID uint,
    uploaderRole string,
    createdAsset *domain.Asset,
    effectiveMonth string,
    includeFlexibleEmployees bool,
) (*BCCImportResult, error) { ... }
```

Key differences from `processWeeklyBCCUpload`:

| Step | WeeklyBCC | WeeklyPayment |
|---|---|---|
| Sheet name | one shift type per sheet | one salary tier per sheet (`520`, `700`, ...) |
| Shift per employee/date | implied by sheet name | taken from `Entry.ShiftKey` (= `prefix + row10Label`) |
| Rate lookup key | bare shift (`HC`, `OT150`) | composite (`520HC`, `520NN`) |
| STK cross-check | by CCCD | by CCCD (same) |
| Auto-create employees from STK | yes | yes (same) |
| Auto-create employees from BCC sheets missing in project | yes | yes (same — `byCCCD[emp.EmployeeCode]` lookup) |
| Default position | first position in payrate | first position in payrate (only 1 exists per partner) |
| Position validation against payrate | not needed (sheet = shift) | validate `<prefix>` is a known tier via a new helper or simply rely on rate lookup failing |

### 2.4 — Rate-key resolution

Reuse `buildShiftRatesForShift` with a per-cell shift key. Concretely, in the entry loop:

```go
for _, sheet := range parsed.Sheets {
    shiftRatesByDate := make(map[string]map[wbccRateKey]int)
    for _, emp := range sheet.Employees {
        for _, entry := range emp.Entries {
            realDate := time.Date(year, month, entry.Day, 0, 0, 0, 0, loc)
            if realDate.Month() != month { continue }

            shiftRates := shiftRatesByDate[realDate.Format("2006-01-02")]
            if shiftRates == nil {
                shiftRates = buildShiftRatesForShift(flatRatesFor(realDate), entry.ShiftKey)
                shiftRatesByDate[realDate.Format("2006-01-02")] = shiftRates
            }

            dayType := dayTypeFor(entry.ShiftKey[len(sheet.Prefix):]) // "HC","TCN","NN","TCNN"
            key := wbccRateKey{position: empPosition, dayType: dayType}
            if _, ok := shiftRates[key]; !ok {
                // log+record import error
                continue
            }
            entries = append(entries, BulkCreateTimesheetEntry{...})
        }
    }
}
```

### 2.5 — STK flow

No code change — `ParseSTKSheet` already handles the 5-column layout (`STT | Mã NV | Tên | STK | Ngân hàng`). The existing `bcc_import_weekly.go` STK auto-creation block can be lifted wholesale into `processWeeklyPaymentUpload`. The "auto-create employees from BCC sheets missing in project" block (the one keyed on `emp.EmployeeCode` rather than the STK row) also carries over unchanged.

### 2.6 — Day-from-weekday resolution (the central design choice)

`parseWeeklyPaymentSheet` is given `firstColumnDay` (computed once in `ParseWeeklyPaymentFile` per sheet). Computation:

```go
firstWeekday, err := ExtractWeekdayFromR9(f, sheetName)
if err != nil { return nil, err }

monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
monthStartWeekday := vietnameseWeekdayCode(monthStart) // "T2".."CN"

// offset = (firstWeekday - monthStartWeekday) mod 7
offset := weekdayOffset(firstWeekday, monthStartWeekday)
firstColumnDay := 1 + offset
```

`vietnameseWeekdayCode(t)` maps `time.Weekday()` → `T2/T3/T4/T5/T6/T7/CN` (Mon..Sun). `weekdayOffset(a, b)` returns the number of days from `b` to `a` (always `0..6`).

This makes sheet 520 (first weekday T4) start at day 1 (since July 1 is Wed = T4) and sheet 900 (first weekday T7) start at day 4 (since July 1 is Wed, the next Sat is day 4).

### 2.7 — Validation: stop column

The "Tổng hợp" summary column (BP8 in the sample) must be excluded. Implementation:

```go
// In buildWeeklyPaymentShiftRow: stop at the column whose row 8 value is a string
// containing "Tổng" or "Tổng hợp" (case-insensitive). Same approach as
// weekly_bcc_parser.go:148-152.
```

## Related Code Files

- **Create:**
  - `backend/internal/app/services/excel/weekly_payment_parser.go`
  - `backend/internal/app/services/excel/weekly_payment_parser_test.go`
  - `backend/tests/fixtures/bcc/thai_binh_duong.xlsx` (copy of the attached sample)
  - `backend/tests/integration/flow_bcc_weekly_payment_import.go`
- **Modify:**
  - `backend/internal/app/services/excel/format_detector.go` — add `FormatWeeklyPayment` + `isWeeklyPaymentSheet`
  - `backend/internal/app/services/excel/format_detector_test.go` — add 4 detector cases
  - `backend/internal/app/services/bcc_import_service.go` — add the `case` in the switch
  - `backend/internal/app/services/bcc_import_weekly.go` — add `processWeeklyPaymentUpload`
  - `backend/internal/app/services/bcc_import_weekly_test.go` — add payrate/rate-key helper tests
  - `backend/tests/integration/main.go` — register the new flow

## Architecture

```
                          ┌──────────────────────┐
  uploaded xlsx ────────► │  excelize.OpenReader  │
                          └──────────┬───────────┘
                                     ▼
                          ┌──────────────────────┐
                          │   DetectFormat()      │  ── FormatWeeklyPayment
                          │   isWeeklyPaymentSheet│     if numeric name + row8 fingerprint + row10 shift
                          └──────────┬───────────┘
                                     ▼
                          ┌──────────────────────┐
                          │  processWeeklyPayment │
                          │  Upload()             │
                          └──────────┬───────────┘
                                     ▼
              ┌──────────────────────┴──────────────────────┐
              ▼                                             ▼
   ┌──────────────────────┐                    ┌────────────────────────┐
   │ ParseWeeklyPayment   │                    │ ParseSTKSheet (existing)│
   │ File()               │                    └────────────┬───────────┘
   │  - read row 8/9/10   │                                 ▼
   │  - per-cell shift    │                    ┌────────────────────────┐
   │  - day = col-derived │                    │ STK auto-create        │
   └──────────┬───────────┘                    │ (existing helpers)     │
              ▼                                └────────────────────────┘
   ┌──────────────────────┐
   │ Rate lookup:         │
   │  buildShiftRatesForShift(flatRates, "<prefix><shift>")
   │  → dayType from shift label
   └──────────┬───────────┘
              ▼
   ┌──────────────────────┐
   │ BulkCreateTimesheets │
   │ InTransaction        │
   └──────────────────────┘
```

## Success Criteria

- [ ] `FormatWeeklyPayment` detected, `WeeklyPaymentSheets` populated
- [ ] Sheet 520's day numbers are 1..28 with no duplicates despite the partner's broken row 8 cell values
- [ ] Sheet 900 starts at day 4 (Saturday) and ends at day 31
- [ ] Rate key `520HC` is looked up in `flatRates` and resolved to a `BulkCreateTimesheetEntry` with `HourType="HC"`, `DayType="ngày thường"`
- [ ] STK rows create employees with bank info
- [ ] Existing 3 formats' tests still pass (no regression in `format_detector_test.go`)
- [ ] `go build ./...` and `cd backend && go test ./... -v -race -cover` green
