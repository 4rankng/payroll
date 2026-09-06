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
	// FormatWeeklyPayment is the format with numeric salary tier sheets (e.g. 520, 700, 750)
	// where row 10 contains per-cell shift codes (HC, TCN, NN, TCNN).
	FormatWeeklyPayment
	// FormatDateRow is the partner layout with a full-date header row (one
	// merged date cell per two-sub-column day) and per-sub-column shift codes
	// below it (e.g. BUMHAN "M1" sheets: NT, OT, T7, OT T7, CN, OT CN).
	FormatDateRow
)

// FormatDetectionResult carries the detected format plus its sheet list.
type FormatDetectionResult struct {
	Format              BCCFormat
	PositionSheets      []string // sheet names that are position sheets (for FormatMultiPosition)
	WeeklyBCCSheets     []string // sheet names like "BCC-HC", "BCC-OT150" (for FormatWeeklyBCC)
	WeeklyPaymentSheets []string // sheet names like "520", "700", "750" (for FormatWeeklyPayment)
	DateRowSheets       []string // sheets with a full-date header row (for FormatDateRow)
}

// DetectFormat determines which BCC format family a workbook belongs to by
// running the ordered registry in bcc_formats.go over a single sheet
// classification pass. Priority: Legacy > WeeklyBCC > WeeklyPayment >
// MultiPosition > DateRow (see bccFormatRegistry).
//
// The registry is the routing truth — strategies and this detector share the
// same fingerprints, and adding a template family means adding a registry
// entry, not editing this loop.
func DetectFormat(f *excelize.File) (*FormatDetectionResult, error) {
	classification := classifyBCCSheets(f)
	for _, entry := range bccFormatRegistry {
		if sheets := entry.sheets(classification); len(sheets) > 0 {
			return formatDetectionResult(entry.format, sheets), nil
		}
	}
	return nil, fmt.Errorf(unknownBCCFormatMsg)
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
		if headerContainsAny(val, "Mã nhân viên", "Ma nhan vien") {
			hasMaNV = true
		}
		if headerContainsAny(val, "Họ và tên", "Ho va ten", "Họ và Tên") {
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

// isWeeklyPaymentSheet checks if a sheet matches the weekly payment format:
// - Row 8 has headers: STT, Mã nhân viên, Họ và tên, Bộ phận, Lương 8h
// - Row 10 has shift codes: HC, TCN, NN, or TCNN
// Sheet name can be any value (not restricted to numeric).
func isWeeklyPaymentSheet(f *excelize.File, sheetName string) bool {
	return hasWeeklyPaymentHeader(f, sheetName) && hasWeeklyPaymentShiftRow(f, sheetName)
}

// hasWeeklyPaymentHeader checks if row 8 contains the expected headers:
// STT, Mã nhân viên, Họ và tên, Bộ phận, Lương 8h
func hasWeeklyPaymentHeader(f *excelize.File, sheet string) bool {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 8 {
		return false
	}

	// Row 8 is index 7
	row := rows[7]
	hasSTT := false
	hasMaNV := false
	hasHoTen := false
	hasBoPhan := false
	hasLuong8h := false

	for _, cell := range row {
		val := strings.TrimSpace(cell)
		if val == "STT" {
			hasSTT = true
		}
		if headerContainsAny(val, "Mã nhân viên", "Ma nhan vien") {
			hasMaNV = true
		}
		if headerContainsAny(val, "Họ và tên", "Ho va ten", "Họ và Tên") {
			hasHoTen = true
		}
		if headerContainsAny(val, "Bộ phận", "Bo phan") {
			hasBoPhan = true
		}
		if headerContainsAny(val, "Lương 8h", "Luong 8h") {
			hasLuong8h = true
		}
	}

	return hasSTT && hasMaNV && hasHoTen && hasBoPhan && hasLuong8h
}

// hasWeeklyPaymentShiftRow checks if row 10 contains any of the shift codes:
// HC, TCN, NN, TCNN (case-insensitive substring match)
func hasWeeklyPaymentShiftRow(f *excelize.File, sheet string) bool {
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) < 10 {
		return false
	}

	// Row 10 is index 9
	row := rows[9]
	shiftCodes := []string{"HC", "TCN", "NN", "TCNN"}

	for _, cell := range row {
		val := strings.ToUpper(strings.TrimSpace(cell))
		for _, code := range shiftCodes {
			if strings.Contains(val, code) {
				return true
			}
		}
	}

	return false
}
