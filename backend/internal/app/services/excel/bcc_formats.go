package excel

import (
	"strings"

	"github.com/xuri/excelize/v2"
)

// unknownBCCFormatMsg is the single source of the no-format-matched error,
// shared by DetectFormat and ParseBCCData so callers see one message shape.
const unknownBCCFormatMsg = "không nhận diện được định dạng file BCC"

// bccSheetClassification is one pass over a workbook's sheets, collecting the
// sheets carrying each format family's fingerprint. Detection then picks a
// winner by registry priority — a sheet that matches several families is only
// ever reported through its highest-priority family.
type bccSheetClassification struct {
	bccSheet      string // sheet named "BCC" (trimmed); "" when absent
	weeklyBCC     []string
	weeklyPayment []string
	position      []string
	dateRow       []string
}

// classifyBCCSheets walks every visible sheet once and classifies it. Sheet
// rules, in order: hidden sheets are skipped; a "BCC"-named sheet is the
// legacy candidate; "STK" (case-insensitive, whitespace-trimmed — partners
// export "STK " etc.) is skipped so the bank sheet never leaks into data
// detection; "BCC-"-prefixed sheets are weekly BCC; remaining sheets are
// fingerprinted as weekly payment, then position, then date-row.
//
// Unlike the pre-registry detector, date-row sheets are collected
// unconditionally — priority selection makes the old workbook-level gate
// redundant (a gate-closed workbook could never select date-row anyway), and
// collecting is what keeps this pass order-independent.
func classifyBCCSheets(f *excelize.File) bccSheetClassification {
	var c bccSheetClassification
	for _, sheetName := range f.GetSheetList() {
		visible, err := f.GetSheetVisible(sheetName)
		if err != nil {
			// If visibility check fails, assume visible (don't block processing).
			visible = true
		}
		if !visible {
			continue
		}
		if isBCCSheetName(sheetName) {
			c.bccSheet = sheetName
			continue
		}
		if isSTKSheetName(sheetName) {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(sheetName), "BCC-") {
			c.weeklyBCC = append(c.weeklyBCC, sheetName)
			continue
		}
		if isWeeklyPaymentSheet(f, sheetName) {
			c.weeklyPayment = append(c.weeklyPayment, sheetName)
			continue
		}
		if isPositionSheet(f, sheetName) {
			c.position = append(c.position, sheetName)
			continue
		}
		if isDateRowBCCSheet(f, sheetName) {
			c.dateRow = append(c.dateRow, sheetName)
		}
	}
	return c
}

// bccFormatEntry is one row of the detection registry: which sheets carry
// this format family, given a classified workbook.
type bccFormatEntry struct {
	name   string
	format BCCFormat
	sheets func(bccSheetClassification) []string
}

// bccFormatRegistry is the single routing truth for BCC workbook detection,
// in priority order: legacy > weekly BCC > weekly payment > multi-position >
// date-row. Legacy leads because it is the strictest shape (pricing from a
// dedicated VND rate row); date-row trails because its fingerprint is the
// loosest (any full-date header row).
//
// Adding a new template family: add a fingerprint, then one entry here at its
// documented priority position. If the family parses into BCCImportData (like
// date-row), that is the whole integration; otherwise the typed parser is
// dispatched by the import service on the detected Format.
var bccFormatRegistry = []bccFormatEntry{
	{
		name:   "legacy",
		format: FormatLegacy,
		sheets: func(c bccSheetClassification) []string {
			if c.bccSheet != "" {
				return []string{c.bccSheet}
			}
			return nil
		},
	},
	{
		name:   "weekly-bcc",
		format: FormatWeeklyBCC,
		sheets: func(c bccSheetClassification) []string { return c.weeklyBCC },
	},
	{
		name:   "weekly-payment",
		format: FormatWeeklyPayment,
		sheets: func(c bccSheetClassification) []string { return c.weeklyPayment },
	},
	{
		name:   "multi-position",
		format: FormatMultiPosition,
		sheets: func(c bccSheetClassification) []string { return c.position },
	},
	{
		name:   "date-row",
		format: FormatDateRow,
		sheets: func(c bccSheetClassification) []string { return c.dateRow },
	},
}

// formatDetectionResult fills the format-specific sheet list field. Legacy
// carries no sheet list in the result — its sheet is resolved again at parse
// time by the strategy router.
func formatDetectionResult(format BCCFormat, sheets []string) *FormatDetectionResult {
	res := &FormatDetectionResult{Format: format}
	switch format {
	case FormatMultiPosition:
		res.PositionSheets = sheets
	case FormatWeeklyBCC:
		res.WeeklyBCCSheets = sheets
	case FormatWeeklyPayment:
		res.WeeklyPaymentSheets = sheets
	case FormatDateRow:
		res.DateRowSheets = sheets
	}
	return res
}
