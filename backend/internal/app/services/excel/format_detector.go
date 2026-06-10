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
)

// FormatDetectionResult holds the result of format detection.
type FormatDetectionResult struct {
	Format         BCCFormat
	PositionSheets []string // sheet names that are position sheets (for FormatMultiPosition)
}

// DetectFormat determines whether the given Excel file uses the old single-BCC-sheet
// format or the new multi-position format.
//
// Algorithm:
//  1. Iterate all sheets, skip hidden ones (via GetSheetVisible)
//  2. If any visible sheet is named exactly "BCC" → FormatLegacy (BCC wins tiebreaker)
//  3. For each remaining visible sheet (not "STK" case-insensitive):
//     Check row 4 for headers: "STT" AND "Mã nhân viên" AND "Họ và tên"
//     If all found → it's a position sheet (sheet name = position value)
//  4. If position sheets found → FormatMultiPosition
//  5. Otherwise → error
func DetectFormat(f *excelize.File) (*FormatDetectionResult, error) {
	var hasBCCSheet bool
	var positionSheets []string

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

		// Skip STK sheet (case-insensitive)
		if strings.ToUpper(sheetName) == "STK" {
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
