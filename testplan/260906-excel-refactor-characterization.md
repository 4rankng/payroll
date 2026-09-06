# Test Matrix — Excel Parsing Refactor Characterization (Phase 0)

Rule: this matrix is written BEFORE any verification run. It defines every
characterization case the Phase 0 safety net must lock. Golden outputs are
generated once from CURRENT code and then frozen; any later diff = regression.

## A. Parse goldens (real fixtures, all 5 formats)

| # | Fixture | Parser entry | Assert (golden JSON) |
|---|---------|--------------|----------------------|
| A1 | `fixtures/bcc/legacy/eva_t08.xlsx` (real EVA) | `ParseBCCFile` | Employees, Entries (date/day/shift-label/hours), ShiftRates, month |
| A2 | `fixtures/bcc/date_row/bumhan_t08.xlsx` (real, "BCC" sheet) | `ParseDateRowBCCFile(f, sheets)` | Employees + Entries with exact dates, bank/mobile/position columns, rateless (empty ShiftRates) |
| A3 | `fixtures/bcc/weekly_bcc/lgd_weekly.xlsx` (real LGD) | `ParseWeeklyBCCFile` | Per-sheet (BCC-HC…OT390) employees + entries, STK rows parsed |
| A4 | `fixtures/bcc/weekly_payment/tbd_weekly_payment.xlsx` (real Thái Bình Dương) | `ParseWeeklyPaymentFile` | Tier sheets (Lương 500–900) employees, tier amount, shift codes, STK |
| A5 | `fixtures/bcc/multi_position/pqc_haiphong.xlsx` (real PQC) | `ParseMultiPositionFile` | Position sheets → positions, employees, entries, rates |

Golden mechanics: marshal parsed struct → JSON → compare vs committed
`*.golden.json` (maps sort keys; slice order = sheet/row order ⇒ deterministic).
`GOLDEN_UPDATE=1` regenerates.

## B. Routing table (DetectFormat → ParseBCCData winner)

| # | Input | Expected DetectFormat | Expected route/winner |
|---|-------|----------------------|----------------------|
| B1 | A1 file | FormatLegacy, no sheets | default branch → ParseBCCData → legacy wins, Format=FormatLegacy |
| B2 | A2 file ("BCC"-named date-row, T09) | FormatLegacy (sheet name!) | default branch → legacy strategy fails Accept → dateRow wins, **Format=FormatDateRow** (winner contract) |
| B3 | A3 file | FormatWeeklyBCC + WeeklyBCCSheets | weekly branch |
| B4 | A4 file | FormatWeeklyPayment + sheets | weekly payment branch |
| B5 | A5 file | FormatMultiPosition + PositionSheets | multi branch |
| B6 | Synthetic: sheet named `"STK "` (trailing space) only | error "không nhận diện được" | STK-trim skip proves sheet not misrouted |
| B7 | Synthetic: hidden BCC sheet + visible position sheet | FormatMultiPosition (hidden skipped) | |
| B8 | Synthetic: empty workbook | DetectFormat error | fail path "không nhận diện được định dạng file" |
| B9 | Synthetic: unknown sheet ("Foo", no fingerprints) | DetectFormat error; ParseBCCData error | unknown-template failure surfaced |
| B10 | Synthetic: date-row sheet alongside weekly-BCC sheets | FormatWeeklyBCC (priority order) | date-row never wins vs named patterns |

## C. STK + strategy Accept behavior

| # | Case | Assert |
|---|------|--------|
| C1 | A3/A4 STK sheets | `ParseSTKSheet` columns resolved by label incl. reordered columns |
| C2 | Synthetic numeric-only "BCC" sheet | legacy Accept=false (pure-digit labels) → falls to dateRow or error |
| C3 | Real A2 | `ParseBCCData` returns FormatDateRow + data (C→B2 duplicates at unit level; keep both) |

## D. Employee import characterization (zero coverage today)

| # | Case | Assert |
|---|------|--------|
| D1 | Synthetic workbook, valid rows B..I | parsed rows: name, CCCD, DOB (4 layouts), phone, bank fields |
| D2 | Row with invalid/missing CCCD | silently skipped (current behavior — locked, not fixed) |
| D3 | Rows with empty optional cells | zero-value fields, no error |

## E. Fixed invariants (assert explicitly, not only via goldens)

| # | Invariant | Where |
|---|-----------|-------|
| E1 | Explicit-0 hour cell → pending timesheet delete (not skip) | A1/A2 goldens must contain 0-hour entries |
| E2 | Day-number-only date semantics (no tz drift) | golden dates are calendar-exact |
| E3 | Winner-format contract | B2/B10 |
| E4 | Hidden sheets skipped; `"STK "` trimmed | B6/B7 |
| E5 | Unknown template errors (not silent import) | B8/B9 |

## F. Suites + order

1. `go test ./internal/app/services/excel/...` (A–C, E)
2. `go test ./internal/app/services/employee/...` (D)
3. `go test ./...` full
4. `make api-test` baseline (flow_bcc_import, flow_bcc_weekly_import, flow_bcc_weekly_payment_import, flow_flexpay_import)

Env-gated: `BCC_REAL_FILE_DIR` (default `testplan/excelfixture/`) — files found
there but not in fixtures are ALSO parsed against the routing table (skip-if-absent).

## Results (fill after runs)

| Suite | Run at | Result |
|-------|--------|--------|
| excel characterization | 2026-09-06 | PASS — 7 goldens, all 5 real formats (incl. BUMHAN T09 winner=FormatDateRow) |
| employee characterization | 2026-09-06 | PASS — 2 tests (parse + empty-workbook) |
| go test ./... | 2026-09-06 | 0 failures at every phase gate |
| make api-test baseline | 2026-09-06 | 281 passed / 0 failed / 23 skipped — identical at every phase gate |

Goldens stayed byte-identical through Phases 2–6 (registry rewrite, dispatch
collapse, weekly split, satellite adoption, upload guard).
