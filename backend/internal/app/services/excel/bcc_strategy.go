package excel

import (
	"fmt"
	"log/slog"
	"unicode"

	"github.com/xuri/excelize/v2"
)

// BCCParseStrategy parses one BCC workbook layout family into BCCImportData.
// New partner templates are supported by adding a strategy implementation and
// registering it in bccDataStrategies — the router then handles the rest.
type BCCParseStrategy interface {
	// Name identifies the strategy in logs and error paths.
	Name() string
	// Format is the BCCFormat this strategy reports to the import pipeline,
	// which keys post-parse behavior (label-keyed rate resolution, STK merge,
	// position handling) off it.
	Format() BCCFormat
	// Match reports the workbook's sheets carrying this strategy's layout.
	Match(f *excelize.File) ([]string, bool)
	// Parse converts matched sheets into import data.
	Parse(f *excelize.File, sheets []string) (*BCCImportData, error)
	// Accept judges a successful parse result as part of routing: when a
	// fingerprint matched but the parse produced no data, the router falls
	// through to the next strategy. Fingerprints are layout heuristics and can
	// collide across template families — a date-row layout can ship under the
	// legacy "BCC" sheet name — so routing verifies with the parser itself.
	Accept(data *BCCImportData) bool
}

// bccDataStrategies routes BCC workbook parsing in priority order. Legacy
// first: it is the strictest shape (its pricing comes from a dedicated VND
// rate row), so a workbook it can actually read must not be re-interpreted by
// the looser date-row strategy. Date-row takes whatever legacy rejects.
//
// Relationship to bccFormatRegistry (bcc_formats.go): the registry decides
// which FORMAT family a workbook belongs to for dispatch; this list then
// arbitrates the one ambiguous case — a "BCC"-named sheet whose content is
// date-row (partner T09) — by parse-then-verify. Both share their sheet-name
// and fingerprint helpers via shared.go, so there is one definition of each
// layout test.
var bccDataStrategies = []BCCParseStrategy{
	legacyBCCStrategy{},
	dateRowBCCStrategy{},
}

// ParseBCCData routes a workbook to the first strategy that matches its layout
// AND yields usable data. A strategy whose parse fails or produces nothing is
// skipped so the next one runs; only when no strategy can read the file does
// routing fail. Returns the winning strategy's format so the caller keeps its
// post-parse behavior keyed to the parser that actually ran.
func ParseBCCData(f *excelize.File) (data *BCCImportData, format BCCFormat, err error) {
	var lastErr error
	for _, strategy := range bccDataStrategies {
		sheets, ok := strategy.Match(f)
		if !ok {
			continue
		}
		parsed, perr := strategy.Parse(f, sheets)
		if perr == nil && strategy.Accept(parsed) {
			return parsed, strategy.Format(), nil
		}
		if perr != nil && lastErr == nil {
			lastErr = perr
			slog.Warn("ParseBCCData: strategy failed, trying next",
				"strategy", strategy.Name(), "sheets", sheets, "error", perr)
		}
	}
	if lastErr != nil {
		return nil, 0, lastErr
	}
	return nil, 0, fmt.Errorf(unknownBCCFormatMsg)
}

// legacyBCCStrategy parses the original single-"BCC"-sheet format: day-number
// grid, shift-label and VND rate rows, employee rows below.
type legacyBCCStrategy struct{}

func (legacyBCCStrategy) Name() string      { return "legacy" }
func (legacyBCCStrategy) Format() BCCFormat { return FormatLegacy }

func (legacyBCCStrategy) Match(f *excelize.File) ([]string, bool) {
	for _, sheet := range f.GetSheetList() {
		if isBCCSheetName(sheet) {
			return []string{sheet}, true
		}
	}
	return nil, false
}

func (legacyBCCStrategy) Parse(f *excelize.File, _ []string) (*BCCImportData, error) {
	return ParseBCCFile(f)
}

// Accept requires evidence the parser actually read the legacy shape: at
// least one alphabetic shift label among the rates/entries. The label-row
// fallback can otherwise manufacture labels (and phantom rates) from employee
// hour cells — pure-digit "labels" that no legacy template produces, but any
// numeric sheet does.
func (legacyBCCStrategy) Accept(data *BCCImportData) bool {
	if data == nil {
		return false
	}
	labels := make(map[string]bool)
	for label := range data.ShiftRates {
		labels[label] = true
	}
	for _, emp := range data.Employees {
		for _, entry := range emp.Entries {
			labels[entry.ShiftLabel] = true
		}
	}
	for label := range labels {
		for _, r := range label {
			if unicode.IsLetter(r) {
				return true
			}
		}
	}
	return false
}

// dateRowBCCStrategy parses partner sheets headed by a full-date header row
// (BUMHAN "M1"-style: two sub-columns per day, per-column shift codes).
type dateRowBCCStrategy struct{}

func (dateRowBCCStrategy) Name() string      { return "date-row" }
func (dateRowBCCStrategy) Format() BCCFormat { return FormatDateRow }

// Match collects every visible sheet carrying the date-row fingerprint. The
// layout test is shared with parsing (isDateRowBCCSheet → findDateRowAndCols),
// so a sheet accepted here always parses.
func (dateRowBCCStrategy) Match(f *excelize.File) ([]string, bool) {
	var sheets []string
	for _, sheet := range f.GetSheetList() {
		visible, err := f.GetSheetVisible(sheet)
		if err != nil {
			// Mirror DetectFormat: on a visibility error, assume visible.
			visible = true
		}
		if !visible {
			continue
		}
		if isDateRowBCCSheet(f, sheet) {
			sheets = append(sheets, sheet)
		}
	}
	return sheets, len(sheets) > 0
}

func (dateRowBCCStrategy) Parse(f *excelize.File, sheets []string) (*BCCImportData, error) {
	return ParseDateRowBCCFile(f, sheets)
}

// Accept mirrors ParseDateRowBCCFile's own contract: any parsed employee
// counts as usable; entries may legitimately be zero (blank month) and the
// empty-roster case errors out of Parse itself.
func (dateRowBCCStrategy) Accept(data *BCCImportData) bool {
	return data != nil && len(data.Employees) > 0
}
