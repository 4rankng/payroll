---
phase: 1
title: "Research"
status: completed
priority: P2
dependencies: []
---

# Phase 1: Research

## Overview

Verify the weekly payment template's exact structure against the partner's real file, confirm the rate-key encoding, and lock the format-detection signature so we can build a parser that never collides with the other 3 formats.

## Concrete findings (from the attached `BẢNG CHẤM CÔNG THÁI BÌNH DƯƠNG THÁNG 08.2026.xlsx`)

### Sheets in the sample

| Sheet | Tier (kVND/8h) | First weekday | First col day (if forMonth=2026-07) | Day cells in row 8 (raw `<v>`) |
|---|---|---|---|---|
| `520` | 520 | T4 (Wed) | day 1 | `1, 2, 3, 1, 2, 3, 4, 5, ..., 28` ⚠️ counter resets |
| `700` | 700 | T7 (Sat) | day 4 | `1, 2, 3, ..., 31` |
| `750` | 750 | T7 (Sat) | day 4 | `1, 2, 3, ..., 31` |
| `800` | 800 | T7 (Sat) | day 4 | `1, 2, 3, ..., 31` |
| `900` | 900 | T7 (Sat) | day 4 | `1, 2, 3, ..., 31` |
| `STK` | (employee bank detail) | — | — | — |

### Row 8/9/10 layout (sheet 520, cols F–U)

```
F8=01  F9=T4  F10=HC
G8=··  G9=··  G10=TCN
H8=02  H9=T5  H10=HC
I8=··  I9=··  I10=TCN
J8=03  J9=T6  J10=HC
K8=··  K9=··  K10=TCN
L8=01  L9=T7  L10=HC    ← day cell value 01, but weekday T7 = day 4 of July
M8=··  M9=··  M10=TCN
N8=02  N9=CN  N10=NN
O8=··  O9=··  O10=TCNN
P8=03  P9=T2  P10=HC
Q8=··  Q9=··  Q10=TCN
R8=04  R9=T3  R10=HC
S8=··  S9=··  S10=TCN
T8=05  T9=T4  T10=HC
```

Two structural truths hold across all 5 sheets:

1. **Row 9 weekday labels are continuous and correct.** They cycle `T4 → T5 → T6 → T7 → CN → T2 → T3 → T4 → ...` and never restart. This is the canonical source of "which day of the month is this column".
2. **Row 8 day values are unreliable.** Sheet 520's counter resets at the start of the second week group. The displayed "01" in L8 is wrong; the actual day is 4 (Wed→Sat = 3 days later).

### Rate lookup encoding

| Cell | Row 10 label | Rate key | Path in `flatRates` |
|---|---|---|---|
| F (col 6) | `HC` | `520HC` | `phổ thông.ngày thường.520HC` |
| G (col 7) | `TCN` | `520TCN` | `phổ thông.ngày thường.520TCN` |
| L (col 12) | `HC` | `520HC` | `phổ thông.ngày thường.520HC` |
| N (col 14) | `NN` | `520NN` | `phổ thông.ngày nghỉ.520NN` |
| O (col 15) | `TCNN` | `520TCNN` | `phổ thông.ngày nghỉ.520TCNN` |

Day type is fully encoded in the row 10 label, so the lookup can use the same `buildShiftRatesForShift(flatRates, "<prefix><shiftCode>")` helper that `processWeeklyBCCUpload` already uses — no new payrate-resolver needed.

### Why this is a *new* format (not a variant of WeeklyBCC)

| Property | WeeklyBCC | WeeklyPayment (this plan) |
|---|---|---|
| Sheet name | `BCC-HC`, `BCC-OT150` (one sheet per shift) | `520`, `700`, ... (one sheet per salary tier) |
| Row 1 | `STT, Mã NV, Họ tên, Dự án?, date1, date2, ...` | company info, blank, blank, blank, month title, blank, blank, `STT, Mã NV, Họ tên, Dự án, Lương 8h, date1, date2, ...` |
| Shift code location | implicit in sheet name | explicit in row 10 (per cell) |
| Cells per day | 1 | 2 (primary + overtime) |
| Header row | row 1 | row 8 |
| Rate key | `<shift>` (e.g. `HC`) | `<prefix><shift>` (e.g. `520HC`) |

The detection signature is therefore:

```
FormatWeeklyPayment ⇔
  Sheet name parses as integer (e.g. "520", "700", "900")
  AND row 8 contains "STT", "Mã nhân viên", "Họ và tên", "Bộ phận", "Lương 8h"
  AND row 10 contains at least one of {HC, TCN, NN, TCNN, ...}
```

### STK sheet layout (sheet 7)

```
A1="SỐ TÀI KHOẢN" (title)
A2="Stt" B2="Mã nhân viên" C2="Tên" D2="Stk" E2="Ngân Hàng"
A3..A14 = STT, B3..B14 = CCCD, C3..C14 = name, D3..D14 = account, E3..E14 = bank name
```

This is the same 5-column STK layout the existing `ParseSTKSheet` already handles. The header row is on row 2, not row 3 (the legacy template used 3). The auto-detect-header logic in `stk_parser.go:54-76` already finds it correctly because it scans up to row 10.

## Format-detection priority lock

After this research, the priority order in `DetectFormat` becomes:

```
1. FormatLegacy            (sheet name "BCC")
2. FormatWeeklyBCC         (sheet name prefix "BCC-")
3. FormatWeeklyPayment     (numeric sheet + row 8 fingerprint + row 10 shifts)   ← NEW
4. FormatMultiPosition     (row 4 fingerprint)
```

Numeric sheet names *without* the row 8/10 fingerprint fall through to `FormatMultiPosition`. This protects against the (unlikely) case where a partner names a position sheet `520` in the multi-position template.

## Done when

- [x] Sheet signature documented (5 columns in row 8, row 10 holds shifts, cells paired)
- [x] Day-number gotcha confirmed and day-from-weekday resolution chosen
- [x] Rate-key encoding verified (`<prefix><shiftCode>`)
- [x] Format-detection signature disambiguated from existing 3 formats
- [x] STK sheet confirmed compatible with existing `ParseSTKSheet`
- [x] Sample file copied to `backend/tests/fixtures/bcc/thai_binh_duong.xlsx` so the parser can be tested without depending on the user's drive attachment
