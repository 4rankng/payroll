# Test Plan — BCC date-row import fixes (review round on aed8a007)

Date: 2026-09-04 · Scope: all findings from the adversarial review of commit
`aed8a007` (date-row BCC template), owner session exited — fixes taken over here.
Ground truth verified against the real template
`~/Downloads/BCC BUMHAN T09.2026 thợ phụ chốt ứng lương - Copy.xlsx` (sheet "M1"):

- date row 6: **52 full-date anchors, cols 9–60** (21/08→15/09, both sub-cols dated)
- cols 61–78: **bare day numbers 16..24** (Sep 16–24 continuation, both sub-cols)
- cols 79+: summary headers ("Tổng giờ công chính thức", "OT 150%", …) + money totals 87–90
- roster rows 9–34 (26 employees), totals row 35, blanks 36–37, footnote row 38
  ("* Chú ý:" in the CCCD column)
- employee headers row 5: STT(1) Mã NV(2) Họ và tên(3) TK Ngân hàng(4) Ngân hàng(5)

Legend: [A]=automated unit · [F]=real-file fixture test · status ☑/⌀/✗

## Findings under test

| ID | Finding (verified against code) | Fix |
|----|----------------------------------|-----|
| C1 | Day region truncated at col 60: real file continues with day-number pairs 61–78 (Sep 16–24 silently dropped); naive cap-raise would ingest summary/money cols 79+ via forward-fill | Day-number continuation off the last full date (validated day = prev+1); walk bounded by last day column; paired-subcol dating replaces forward-fill |
| C2 | First blank CCCD+name row ends the roster (spacer row would cut it) — but naive skip would ingest the "* Chú ý:" footnote | Skip a blank row only when a row within the next 3 has CCCD/name AND day-region hours; totals break kept |
| C3 | Red test at HEAD: RealFile expects 27/301 (stale — footnote row no longer ingested); parser correctly yields 26/288 pre-fix | Reconcile to 26 employees + post-C1 entry count (computed from fixed parser), add Sep-continuation spot-check |
| D1 | Catch-all fingerprint can parse unknown layouts by positional guess | Fold shift-row requirement into findDateRowAndCols (D12) — a sheet with stray dates above the real header now parses the REAL header instead of failing; unknown sheets still fail loudly at DetectFormat |
| D2 | FormatDateRow never reads the STK sheet (bank/mobile refresh + STK-only hires skipped) | Parse STK when present; merge by CCCD, STK wins bank/mobile; STK-only rows appended |
| D3 | labelRateTarget tie-break nondeterministic on equal dayType priority (map order) | Deterministic: among equal priority prefer the smaller dayType string (benign for pricing — target identical — but stable) |
| D4 | Label-first resolution fires for ALL rateless files; legacy 'CN'(ca ngày) collides with BUMHAN 'CN'(Chủ Nhật) leaves → legacy files silently mispriced | Label-first gated to FormatDateRow only; legacy rateless keeps shiftLabelHourType + calendar |
| D5 | Missing-rate error prints "(0 VND)" for rateless files | Omit the VND clause when the file carries no rate for the label |
| D6 | Date forward-fill attributes a blank-anchor day's hours to the previous day | Paired-subcol dating: date = anchor[col] else anchor[col−1] else skip (no cross-day inheritance) |
| D7 | earliestDay computed from raw DayNum before the in-month filter → payrate probe can target an out-of-month day | Compute earliest from in-month entries only (month parsed before prepareImportContext) |
| D8 | Date-row employees never populate Mobile even when an SĐT column exists | Detect phone column ("sđt"/"điện thoại"); plumb through STKRow.Mobile |
| D9 | Employee-header scan window (dateRow−4) may miss deeper header stacks | Widen to dateRow−6 |
| D10 | multi_position.go declares local dayTypePriority/rateTarget shadowing package-level | Delete locals, use shared decls |
| D11 | flatRates bucket scan duplicated 3× | Extract shared iterator only if behavior-identical; else keep (documented) |
| D12 | isDateRowBCCSheet re-implements findDateRowAndCols with ANY-row semantics vs parser's FIRST-row → detect-then-parse-fail | Fold shift-row requirement into findDateRowAndCols; detector = parser helper |
| D13 | DetectFormat probes date-row fingerprint on every sheet even when a higher-priority format already matched | Skip probe once any higher-priority sheet list is non-empty |
| D15 | dateRowEmployeeCols.sttCol detected but never read | Delete field + detection |

Reviewer claims overturned by scouting (no fix): angle-a's "anchors at cols 57–75"
(imprecise — region is 9–60 full dates + 61–78 day numbers; conclusion stood);
D3 pricing nondeterminism (target identical across ties; rate resolved downstream
per assignment).

## 1. Day-region coverage (C1, D6)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 1.1 | Real T09 file | Entries parsed for every day 21/08→24/09 incl. Sep 16–24; zero entries with summary labels ("Tổng giờ công chính thức" etc.) or from money cols 87–90 | F | ☑ |
| 1.2 | Fixture: full dates 3 days then day-number continuation | Continuation days dated correctly (prev+1, month rollover safe) | A | ☑ |
| 1.3 | Fixture: continuation cell wrong (day skips 2) | Continuation stops at mismatch; hours under it dropped (not mis-dated) | A | ☑ |
| 1.4 | Fixture: summary label columns right of day region | Never ingested (walk bounded by last day col) | A | ☑ |
| 1.5 | Merged-style anchors (one date per day pair — existing fixture) | Sub-col without anchor dated via col−1 | A | ☑ |
| 1.6 | Day whose anchor is blank/text (D6) | That day's sub-columns skipped — NOT attributed to previous day | A | ☑ |

## 2. Roster integrity (C2)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 2.1 | Real file: totals row 35 then blanks then footnote 38 | 26 employees; totals + footnote never ingested | F | ☑ |
| 2.2 | Fixture: blank spacer between two active employees | Both sides parsed (look-ahead continues) | A | ☑ |
| 2.3 | Fixture: blank rows then footnote-style row (no day hours) | Roster ends (no ingestion) | A | ☑ |
| 2.4 | Fixture: "Tổng cộng" row mid-roster | Break immediately (unchanged) | A | ☑ |

## 3. Rate resolution (D3, D4)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 3.1 | FormatDateRow + label NT with leaves under 2 day types | ngày thường wins (priority), stable across repeated calls | A | ☑ |
| 3.2 | Equal-priority tie (2 positions, same dayType class) | Deterministic smaller-dayType winner across map-order permutations | A | ☑ |
| 3.3 | LEGACY rateless file, entry CN on a Wednesday, config has ngày nghỉ.CN leaf | Resolves via shiftLabelHourType+calendar → (ngày thường, ca ngày) — NOT label-first to ngày nghỉ | A | ☑ |
| 3.4 | FormatDateRow, label X matching no leaf | Missing-rate error (5.x message shape, no VND) | A | ☑ |

## 4. Import context + STK (D7, D2, D5)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 4.1 | Straddling file (Jul 26–Aug 25) for month 08, payrate from Aug 20 | Rejected at payrate gate (probe = first in-month day, Aug 1) — no retroactive pricing | A | ☑ |
| 4.2 | Date-row file + STK sheet with corrected bank + STK-only hire | STK bank applied; STK-only employee created + assigned | A | ☑ |
| 4.3 | Rateless missing-rate error text | "không tìm thấy mức lương cho ca X" — no "(0 VND)" | A | ☑ |
| 4.4 | Date-row file without STK sheet | Unchanged (BCC-derived rows only) | A | ☑ |

## 5. Detection (D12, D13, D1)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 5.1 | Sheet with 3 stray dates in row 3, real header row 6 | Detector AND parser both use row 6 (unified) — no detect-then-parse-fail | A | ☑ |
| 5.2 | Workbook with position sheets + a date-row-looking sheet | FormatMultiPosition wins; date-row probe skipped (guard) — result unchanged from today | A | ☑ |
| 5.3 | Unknown workbook (no fingerprint) | Loud "không nhận diện được định dạng file" (unchanged) | A | ☑ |
| 5.4 | Real T09 detection | FormatDateRow, sheets [M1] (hidden sheets excluded) | F | ☑ |

## 6. Employee columns (D8, D9, D15)

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 6.1 | Fixture: headers 6 rows above date row | Bank/phone cols still detected (widened window) | A | ☑ |
| 6.2 | Fixture with SĐT column | Mobile captured + flows to STK row | A | ☑ |
| 6.3 | No bank columns present | Empty bank fields, no error (legit variant) | A | ☑ |
| 6.4 | sttCol removed from struct | Build passes; no behavior change | A | ☑ |

## 7. Red-test reconciliation (C3) + regression gates

| # | Case | Expected | Type | Status |
|---|------|----------|------|--------|
| 7.1 | RealFile test | 26 employees; entries = post-fix count (computed); Aug 22 T7=9/OT T7=1 spot-check kept; NEW Sep-continuation spot-check | F | ☑ |
| 7.2 | `go build ./...` + `go vet` touched pkgs | clean | A | ☑ |
| 7.3 | `go test ./internal/app/services/... ./internal/app/services/excel/` | all green (incl. legacy/weekly/multi-position/stk suites) | A | ☑ |
| 7.4 | Integration suite | 275/0/21 baseline | A | ☑ |

## Execution notes

- 1.1/2.1/7.1 run against the real file via the existing skip-if-absent pattern;
  expected numbers computed by running the FIXED parser once, then locked into
  the test (26 employees is pre-verified by probe).
- Scratch probe file (zz_scratch_probe_test.go) is deleted before commit.
- No frontend changes in this round.

## Results (executed 2026-09-04 against the real T09 template + fixtures)

- All matrix sections ☑ (1.1–1.6, 2.1–2.4, 3.1–3.4, 4.1–4.4, 5.1–5.4, 6.1–6.4, 7.1–7.4).
- **1.1 nuance discovered during execution**: the T09 file's Sep 16–24
  continuation columns hold ZERO hours (settlement-cutoff file; data ends
  15/09). The truncation was latent, not active: the parser now spans the
  full grid (locked structurally — day map ends col 78 / 2026-09-24) and the
  entry count stays 288 on this file. The original 27/301 expectations were
  stale for a different reason (footnote-row ingestion, pre-totals-break
  parser); reconciled to 26/288.
- **D11 deliberately not unified**: the three flatRates scans differ in
  canonicalization semantics (raw vs canonical segments); a shared iterator
  would change resolution behavior. D10 (shadowing locals) removed instead.
- Gates: `go build` ✓ · `go vet` (services + excel) ✓ · excel package fully
  green (8 date-row tests + legacy/weekly/multi-position/stk suites) ✓ ·
  services package green except 7 PRE-EXISTING wallet-forecast failures
  (documented unfinished cutoff 21→20 work, files untouched here) ·
  integration 275/0/21 ✓ · no frontend changes.
