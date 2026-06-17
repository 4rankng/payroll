package excel

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// BCCFormat represents the detected format of a BCC Excel file.
type BCCFormat int

const (
	// FormatLegacy is the old BCC/STK format with a single "BCC" sheet.
	FormatLegacy BCCFormat = iota
	// FormatMultiPosition is the new format with position-named sheets.
	FormatMultiPosition
	// FormatWeeklyBCC is the format with BCC-<shiftType> sheets (e.g. BCC-HC, BCC-OT150).
	FormatWeeklyBCC
)

// FormatDetectionResult holds the result of format detection.
type FormatDetectionResult struct {
	Format          BCCFormat
	PositionSheets  []string // sheet names that are position sheets (for FormatMultiPosition)
	WeeklyBCCSheets []string // sheet names like "BCC-HC", "BCC-OT150" (for FormatWeeklyBCC)
}

// DetectFormat determines whether the given Excel file uses the old single-BCC-sheet
// format, the multi-position format, or the weekly BCC format.
//
// Algorithm:
//  1. Iterate all sheets, skip hidden ones (via GetSheetVisible)
//  2. If any visible sheet is named exactly "BCC" → FormatLegacy (BCC wins tiebreaker)
//  3. For each remaining visible sheet (not "STK" case-insensitive):
//     a. If sheet name has "BCC-" prefix → weekly BCC sheet (shift type from suffix)
//     b. Otherwise check row 4 for headers: "STT" AND "Mã nhân viên" AND "Họ và tên"
//     If all found → it's a position sheet (sheet name = position value)
//  4. Priority: FormatLegacy > FormatWeeklyBCC > FormatMultiPosition
//  5. Otherwise → error
func DetectFormat(f *excelize.File) (*FormatDetectionResult, error) {
	var hasBCCSheet bool
	var positionSheets []string
	var weeklyBCCSheets []string

	for _, sheetName := range f.GetSheetList() {
		// Skip hidden sheets
		visible, err := f.GetSheetVisible(sheetName)
		if err != nil {
			// If visibility check fails, assume visible (don't block processing)
			visible = true
		}
		if !visible {
			continue
		}

		// Check for BCC sheet (exact or trimmed match — some files have trailing spaces)
		if sheetName == "BCC" || strings.TrimSpace(sheetName) == "BCC" {
			hasBCCSheet = true
			// Don't break — we need to check all sheets, but BCC wins as tiebreaker
			continue
		}

		// Skip STK sheet (case-insensitive, whitespace-trimmed — partners export
		// it as "STK " etc.; without the trim the sheet leaks into position-sheet
		// detection and can misroute the upload).
		if strings.EqualFold(strings.TrimSpace(sheetName), "STK") {
			continue
		}

		// Check for weekly BCC sheet (BCC-<shiftType> naming pattern, case-insensitive)
		if strings.HasPrefix(strings.ToUpper(sheetName), "BCC-") {
			weeklyBCCSheets = append(weeklyBCCSheets, sheetName)
			continue
		}

		// Check row 4 for expected header pattern
		if isPositionSheet(f, sheetName) {
			positionSheets = append(positionSheets, sheetName)
		}
	}

	// BCC sheet wins as tiebreaker (even if position sheets also exist)
	if hasBCCSheet {
		return &FormatDetectionResult{
			Format: FormatLegacy,
		}, nil
	}

	// Weekly BCC takes priority over multi-position
	if len(weeklyBCCSheets) > 0 {
		return &FormatDetectionResult{
			Format:          FormatWeeklyBCC,
			WeeklyBCCSheets: weeklyBCCSheets,
		}, nil
	}

	if len(positionSheets) > 0 {
		return &FormatDetectionResult{
			Format:         FormatMultiPosition,
			PositionSheets: positionSheets,
		}, nil
	}

	return nil, fmt.Errorf("không nhận diện được định dạng file BCC")
}

// isPositionSheet checks if a sheet has the expected header pattern in row 4:
// "STT" + "Mã nhân viên" + "Họ và tên"
func isPositionSheet(f *excelize.File, sheetName string) bool {
	rows, err := f.GetRows(sheetName)
	if err != nil || len(rows) < 4 {
		return false
	}

	// Row 4 is index 3
	row := rows[3]
	hasSTT := false
	hasMaNV := false
	hasHoTen := false

	for _, cell := range row {
		val := strings.TrimSpace(cell)
		if val == "STT" {
			hasSTT = true
		}
		if strings.Contains(val, "Mã nhân viên") || strings.Contains(val, "Ma nhan vien") {
			hasMaNV = true
		}
		if strings.Contains(val, "Họ và tên") || strings.Contains(val, "Ho va ten") || strings.Contains(val, "Họ và Tên") {
			hasHoTen = true
		}
	}

	return hasSTT && hasMaNV && hasHoTen
}

// ExtractShiftType extracts the shift type from a weekly BCC sheet name.
// e.g. "BCC-HC" → "HC", "BCC-OT150" → "OT150", "bcc-HC" → "HC"
// Returns empty string if the sheet name doesn't match the BCC-<shiftType> pattern.
func ExtractShiftType(sheetName string) string {
	upper := strings.ToUpper(sheetName)
	if !strings.HasPrefix(upper, "BCC-") {
		return ""
	}
	// Find the prefix position in the original string to preserve shiftType casing.
	idx := strings.Index(sheetName, "-")
	if idx < 0 {
		return ""
	}
	shiftType := sheetName[idx+1:]
	if shiftType == "" {
		return ""
	}
	return shiftType
}
