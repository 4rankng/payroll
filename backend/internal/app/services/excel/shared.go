package excel

import (
	"strings"
	"time"

	"api-server/internal/pkg/excelkit"
	"github.com/xuri/excelize/v2"
)

// Shared, verified-identical helpers for the BCC parser family. Everything
// here is a semantics-frozen port: the per-parser goldens fail if matching
// behavior drifts. Divergent look-alikes (the two parseForMonth variants, the
// four stop-column variants) deliberately stay local to their owners.

// isSummaryRow reports whether an employee data row is actually the
// workbook's summary row — "tổng cộng" in the name or identity column.
// Parsers stop reading the roster there.
func isSummaryRow(name, code string) bool {
	return strings.Contains(strings.ToLower(name), "tổng cộng") ||
		strings.Contains(strings.ToLower(code), "tổng cộng")
}

// headerContainsAny reports whether a trimmed header cell contains any of the
// given variants. The Contains-variant idiom used by the multi-position
// parser and the detector's sheet fingerprints.
func headerContainsAny(val string, variants ...string) bool {
	for _, v := range variants {
		if strings.Contains(val, v) {
			return true
		}
	}
	return false
}

// isBCCSheetName matches the legacy "BCC" sheet exactly or after trimming —
// some files carry trailing spaces. Shared by the detector's classifier and
// the legacy parse strategy so both route the same sheets.
func isBCCSheetName(name string) bool {
	return name == "BCC" || strings.TrimSpace(name) == "BCC"
}

// isSTKSheetName matches the bank sheet case-insensitively after trimming —
// partners export "STK ", " stk", etc.
func isSTKSheetName(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), "STK")
}

// normHeader normalizes a header cell for matching: trim, NFC unicode,
// lower-case. NFC prevents macOS-exported NFD text from missing "họ và tên".
// Delegate to the shared excelkit primitive (verbatim semantics).
func normHeader(s string) string {
	return excelkit.NormalizeHeader(s)
}

// bccCell reads one cell's formatted string value by 1-based coordinates,
// trimmed; read errors surface as "". Delegate to excelkit.
func bccCell(f *excelize.File, sheet string, col, row int) string {
	return excelkit.Cell(f, sheet, col, row)
}

// dateRowNumericCell reads a data cell as a raw, format-proof number and
// reports whether the cell carried an explicit value. Blank and non-numeric
// cells return (0, false); an explicit 0 returns (0, true) — the import reads
// that as a deletion request for the day. Delegate to excelkit.
func dateRowNumericCell(f *excelize.File, sheet string, col, row int) (float64, bool) {
	return excelkit.NumericCell(f, sheet, col, row)
}

// parseExcelDate parses an Excel date serial cell value (fraction floored,
// 2020–2040 guard). Delegate to excelkit; the returned time carries no
// location — business dates derive from the clock package, never time.Local.
func parseExcelDate(val string) (time.Time, bool) {
	return excelkit.ExcelDate(val)
}
