package excel

import (
	"errors"
	"strings"

	"github.com/xuri/excelize/v2"
)

// unknownBCCFormatMsg is the single source of the no-format-matched error,
// shared by DetectFormat and ParseBCCData so callers see one message shape.
const unknownBCCFormatMsg = "không nhận diện được định dạng file BCC"

// ErrUnknownBCCFormat is the sentinel for "no supported template matched".
// Callers use errors.Is on it to replace the internal wording with the
// uploader-facing rejection message.
var ErrUnknownBCCFormat = errors.New(unknownBCCFormatMsg)

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
// rules, in order: hidden sheets are skipped; "STK" (case-insensitive,
// whitespace-trimmed — partners export "STK " etc.) is skipped so the bank
// sheet never leaks into data detection; weekly BCC claims "BCC-"-prefixed
// sheets AND sheets carrying the weekly block content fingerprint — the
// fingerprint test must precede the name rule because block-layout sheets
// ship under a plain "BCC " name (trailing space) that the legacy rule would
// otherwise claim; a remaining "BCC"-named sheet is the legacy candidate;
// after that, sheets are fingerprinted as weekly payment, then position. The
// date-row fingerprint runs in a second phase, only when no higher-priority
// family claimed the workbook: it is the loosest (any full-date header row)
// and most expensive test, and by registry priority it can only win when
// everything else is empty — so skipping it otherwise is outcome-identical.
func classifyBCCSheets(f *excelize.File) bccSheetClassification {
	var c bccSheetClassification
	var unmatched []string
	for _, sheetName := range f.GetSheetList() {
		visible, err := f.GetSheetVisible(sheetName)
		if err != nil {
			// If visibility check fails, assume visible (don't block processing).
			visible = true
		}
		if !visible {
			continue
		}
		if isSTKSheetName(sheetName) {
			continue
		}
		// The weekly BCC claim must precede the BCC-name rule: block-layout
		// sheets ship under a plain "BCC " name (trailing space) and the
		// loose sheet-name rule would otherwise claim them as legacy before
		// the content fingerprint gets a chance.
		if strings.HasPrefix(strings.ToUpper(sheetName), "BCC-") || isWeeklyBlockBCCSheet(f, sheetName) {
			c.weeklyBCC = append(c.weeklyBCC, sheetName)
			continue
		}
		if isBCCSheetName(sheetName) {
			c.bccSheet = sheetName
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
		unmatched = append(unmatched, sheetName)
	}
	if c.bccSheet == "" && len(c.weeklyBCC) == 0 && len(c.weeklyPayment) == 0 && len(c.position) == 0 {
		for _, sheetName := range unmatched {
			if isDateRowBCCSheet(f, sheetName) {
				c.dateRow = append(c.dateRow, sheetName)
			}
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
