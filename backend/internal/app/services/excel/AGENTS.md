# AGENTS.md — services/excel (BCC/Excel parsing)

Parser-only package for timesheet workbook imports. Zero persistence, zero
HTTP: bytes in, typed parse results out. Orchestration lives in
`../bcc_import_*.go`; generic primitives live in `internal/pkg/excelkit`.

## Adding a new BCC template family

1. **Fixture** — drop an anonymized real file into `testplan/excelfixture/`
   (env-gated sweep picks it up) and/or commit a copy under
   `backend/tests/fixtures/bcc/<format>/`. Generate its golden:
   `GOLDEN_UPDATE=1 go test ./internal/app/services/excel/ -run TestCharacterization`.
2. **Parser** — new `<name>_parser.go` with a typed entry
   (`Parse<Name>File(f, sheets) (*<Name>ImportData, error)`). Reuse
   `shared.go` helpers (`isSummaryRow`, `headerContainsAny`, `bccCell`,
   `dateRowNumericCell`, `parseExcelDate`); port nothing that already exists.
3. **Fingerprint + registry line** — add a `Match` (fingerprint func) and one
   entry in `bccFormatRegistry` (`bcc_formats.go`) at its documented priority
   position. `DetectFormat` and routing need no other change.
4. **Dispatch** — if the layout parses into `BCCImportData` (like date-row),
   the default branch's strategy list in `bcc_strategy.go` is the only other
   touch. Otherwise add one `case` in `bcc_import_process.go` calling your
   `process<Name>Upload`, built from the shared pipeline helpers
   (`bccFailer`, `prepareImportContext`, `planMonthReplacement`,
   `finishEmptyBCCImport`, `finalizeCreatedBCCImport`).
5. **Tests** — golden parse test + a row in
   `routing_characterization_test.go`'s real-file classification table.

Acceptance: a new template = 1 parser file + 1 registry line + 1 test file +
fixture. If it takes more, the design has drifted — fix the design.

## Invariants (breaking any of these is a production bug)

- **Priority order** in `bccFormatRegistry`: legacy > weekly BCC > weekly
  payment > multi-position > date-row. Reordering changes which files parse.
- **Winner-format contract**: a "BCC"-named sheet can carry a date-row layout
  (T09); `ParseBCCData` overwrites the detected format with the strategy
  winner, and `bcc_import_process.go` keys STK collection, position
  precedence, and label-keyed rate resolution on that winner.
- **Explicit 0-hour cells** are deletion requests — parsers must keep them;
  blank cells are skipped. `dateRowNumericCell` returns `(0, true)` for that.
- **RawCellValue** everywhere hours/dates are read: display formats round
  7.5h to 8 (the Nguyễn Trọng Thắng bug) and shift date serials by one day.
- **Hidden sheets and `"STK "` (trailing space)** are excluded from
  detection; STK is a bank-info sheet, not a data sheet.
- **Two `parseForMonth`s are intentionally different** (here vs
  `services/bcc_import_helpers.go`) — different accepted formats and error
  text; error text is user-visible. Do not merge.
- All goldens under `testdata/goldens/` must stay byte-identical unless a
  behavior change was explicitly approved.

## Layout map

| File | Owns |
|------|------|
| `bcc_formats.go` | sheet classification + ordered detection registry (routing truth) |
| `format_detector.go` | `DetectFormat` loop + sheet fingerprints |
| `bcc_strategy.go` | legacy↔date-row parse arbitration (`ParseBCCData`, Accept verify) |
| `bcc_parser.go`, `date_row_bcc_parser.go`, `weekly_bcc_parser.go`, `weekly_payment_parser.go`, `multi_position_parser.go`, `stk_parser.go` | per-family parsers |
| `shared.go` | in-package shared helpers (thin excelkit delegates + predicates) |
| `timesheet_template_service.go`, `timesheet_import_service.go` | the system-generated template round-trip |
| `export_service.go` | styling toolkit for exports |
