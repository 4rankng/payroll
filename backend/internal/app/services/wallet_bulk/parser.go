package wallet_bulk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ExpectedSheet is the workbook sheet name written by the OnePay exporter.
const ExpectedSheet = "eMB_BulkPayment"

// unzipSizeLimit caps excelize's total unzip output (default is 16GB — a
// zip bomb would OOM us). 50 MiB is plenty for a 5,000-row workbook.
const unzipSizeLimit = 50 << 20

// unzipXMLSizeLimit caps each individual XML stream inside the zip.
const unzipXMLSizeLimit = 10 << 20

// Header row 2 is 1-indexed in Excel; data starts at row 3.
const (
	headerRowIndex = 2 // 1-indexed
	firstDataRow   = 3 // 1-indexed
	maxEmptyStreak = 2 // 2 consecutive empty rows = end of data
)

// Header aliases — match by name, not offset, so column reordering is
// tolerated. Each header cell is trimmed + lowercased before lookup.
var headerAliases = map[string]string{
	// OrderNo
	"stt":        "order_no",
	"ord. no.":   "order_no",
	"ord.no":     "order_no",
	"(ord. no.)": "order_no",
	"(ord.no)":   "order_no",
	// AccountNo
	"số tài khoản":  "account_no",
	"so tai khoan":  "account_no",
	"account no.":   "account_no",
	"account no":    "account_no",
	"(account no.)": "account_no",
	"(account no)":  "account_no",
	// AccountName
	"tên người thụ hưởng": "account_name",
	"ten nguoi thu huong": "account_name",
	"beneficiary":         "account_name",
	"(beneficiary)":       "account_name",
	// Bank (Vietnamese display name)
	"ngân hàng thụ hưởng/chi nhánh": "bank",
	"ngan hang thu huong/chi nhanh": "bank",
	"beneficiary bank":              "bank",
	"beneficiary bank / branch":     "bank",
	"(beneficiary bank)":            "bank",
	// SwiftCode
	"mã swift":     "swift_code",
	"ma swift":     "swift_code",
	"mã swift/bic": "swift_code",
	"ma swift/bic": "swift_code",
	"swift code":   "swift_code",
	"swift/bic":    "swift_code",
	"bic":          "swift_code",
	"(swift code)": "swift_code",
	"(swift/bic)":  "swift_code",
	// Amount
	"số tiền":  "amount",
	"so tien":  "amount",
	"amount":   "amount",
	"(amount)": "amount",
	// PaymentDetail
	"nội dung chuyển khoản": "payment_detail",
	"noi dung chuyen khoan": "payment_detail",
	"payment detail":        "payment_detail",
	"(payment detail)":      "payment_detail",
	"description":           "payment_detail",
}

// canonicalFields lists every header alias value we MUST resolve before we
// can parse a row. swift_code is in this set so a missing SWIFT column is
// caught as ErrMissingSwiftColumn with a helpful message (rather than a
// generic ErrUnknownExcelHeader).
var canonicalFields = map[string]bool{
	"order_no":       true,
	"account_no":     true,
	"account_name":   true,
	"bank":           true,
	"swift_code":     true,
	"amount":         true,
	"payment_detail": true,
}

// vficRe extracts the VFIC transaction code from PaymentDetail.
// Verified against production: VFIC + first 8 hex chars of UUID, e.g. VFIC3ba3ec31.
var vficRe = regexp.MustCompile(`(?i)VFIC[0-9a-f]+`)

// swiftRe validates ISO 9362 format: 4 letters (bank) + 2 letters (country)
// + 2 alphanumerics (location) + optional 3 alphanumerics (branch).
var swiftRe = regexp.MustCompile(`^[A-Z]{4}[A-Z]{2}[A-Z0-9]{2}([A-Z0-9]{3})?$`)

// YeuCauChuyenTienParser parses the OnePay "Yêu cầu chuyển tiền" .xlsx
// exported by /admin/timesheet → "Chuyển OnePay". It does NOT resolve bank
// names — SWIFT codes are read directly from the input file's "Mã SWIFT"
// column (the exporter pre-resolves them at Stage 1).
type YeuCauChuyenTienParser struct {
	logger *slog.Logger
}

// NewYeuCauChuyenTienParser constructs a parser. logger may be nil — a
// default is substituted.
func NewYeuCauChuyenTienParser(logger *slog.Logger) *YeuCauChuyenTienParser {
	if logger == nil {
		logger = slog.Default()
	}
	return &YeuCauChuyenTienParser{logger: logger}
}

// Parse reads the workbook from r and returns the normalized rows.
//
// Format contract:
//   - Sheet name must be `eMB_BulkPayment`.
//   - Header row 2 carries the column names (Vietnamese or English aliases).
//   - Data rows start at row 3 and end at the first 2 consecutive empty rows.
//   - Row count is capped at MaxRows (5,000) to defend against zip bombs.
//
// Returns one of the sentinel errors from types.go when validation fails.
func (p *YeuCauChuyenTienParser) Parse(ctx context.Context, r io.Reader) ([]BulkTransferRow, error) {
	f, err := excelize.OpenReader(r, excelize.Options{
		UnzipSizeLimit:    unzipSizeLimit,
		UnzipXMLSizeLimit: unzipXMLSizeLimit,
		MaxCalcIterations: 1,
		ShortDatePattern:  "yyyy-mm-dd",
		LongDatePattern:   "yyyy-mm-dd",
	})
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	index, err := f.GetSheetIndex(ExpectedSheet)
	if err != nil || index < 0 {
		return nil, fmt.Errorf("%w: sheet %q not found", ErrInvalidSheet, ExpectedSheet)
	}

	colMap, err := p.readHeader(f)
	if err != nil {
		return nil, err
	}

	rows, err := f.GetRows(ExpectedSheet)
	if err != nil {
		return nil, fmt.Errorf("read rows: %w", err)
	}
	if len(rows) < firstDataRow {
		return nil, ErrEmptyFile
	}

	parsed := make([]BulkTransferRow, 0, 32)
	emptyStreak := 0
	for excelRow := firstDataRow; excelRow <= len(rows); excelRow++ {
		if emptyStreak >= maxEmptyStreak {
			break
		}
		raw := rows[excelRow-1]
		if p.rowIsEmpty(raw) {
			emptyStreak++
			continue
		}
		emptyStreak = 0

		if len(parsed) >= MaxRows {
			return nil, fmt.Errorf("%w: %d rows", ErrTooManyRows, MaxRows)
		}

		row, perr := p.parseRow(raw, colMap, excelRow)
		if perr != nil {
			return nil, perr
		}
		parsed = append(parsed, row)
	}

	if len(parsed) == 0 {
		return nil, ErrEmptyFile
	}
	return parsed, nil
}

// resolveHeader maps one raw header cell to its canonical field name.
// Tolerates multi-line headers (e.g. "STT\n(Ord. No.)") by trying each
// newline-separated segment, then the full collapsed string. Returns
// (canonical, true) on the first alias match.
func resolveHeader(cell string) (string, bool) {
	// Try the whole cell first (single-line headers like "Beneficiary").
	if canonical, ok := lookupAlias(cell); ok {
		return canonical, true
	}
	// Split on newlines for bilingual multi-line headers — each segment
	// is a valid alias on its own ("STT" or "(Ord. No.)").
	for _, line := range strings.Split(cell, "\n") {
		if canonical, ok := lookupAlias(line); ok {
			return canonical, true
		}
	}
	// Collapse internal whitespace and try once more (handles "Số  tài khoản").
	collapsed := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(cell)), " "))
	if canonical, ok := headerAliases[collapsed]; ok {
		return canonical, true
	}
	return "", false
}

// lookupAlias trims + lowercases s and looks it up in headerAliases.
func lookupAlias(s string) (string, bool) {
	key := strings.ToLower(strings.TrimSpace(s))
	canonical, ok := headerAliases[key]
	return canonical, ok
}

// readHeader reads row 2 and returns a map: canonical_field_name → 0-based
// column index. Returns ErrMissingSwiftColumn when the SWIFT column is absent;
// ErrUnknownExcelHeader for any unrecognized header cell.
func (p *YeuCauChuyenTienParser) readHeader(f *excelize.File) (map[string]int, error) {
	cols, err := f.GetRows(ExpectedSheet)
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	if len(cols) < headerRowIndex {
		return nil, ErrEmptyFile
	}
	header := cols[headerRowIndex-1]

	colMap := make(map[string]int, len(header))
	for i, cell := range header {
		canonical, ok := resolveHeader(cell)
		if !ok {
			return nil, fmt.Errorf("%w: %q at column %d", ErrUnknownExcelHeader, cell, i+1)
		}
		// First occurrence wins (matches production exporter output).
		if _, exists := colMap[canonical]; !exists {
			colMap[canonical] = i
		}
	}

	for field := range canonicalFields {
		if _, ok := colMap[field]; !ok {
			if field == "swift_code" {
				return nil, fmt.Errorf("%w: header has no Mã SWIFT column", ErrMissingSwiftColumn)
			}
			return nil, fmt.Errorf("%w: %q", ErrUnknownExcelHeader, field)
		}
	}
	return colMap, nil
}

// parseRow maps one raw row to a BulkTransferRow.
func (p *YeuCauChuyenTienParser) parseRow(raw []string, colMap map[string]int, excelRow int) (BulkTransferRow, error) {
	get := func(field string) string {
		idx, ok := colMap[field]
		if !ok || idx >= len(raw) {
			return ""
		}
		return strings.TrimSpace(raw[idx])
	}

	row := BulkTransferRow{
		AccountNo:     get("account_no"),
		AccountName:   get("account_name"),
		Bank:          get("bank"),
		SwiftCode:     strings.ToUpper(get("swift_code")),
		PaymentDetail: get("payment_detail"),
	}

	// OrderNo is optional in input — derive from row position so we don't
	// depend on the producer having stamped it.
	sttStr := get("order_no")
	if sttStr != "" {
		n, err := strconv.Atoi(sttStr)
		if err == nil && n > 0 {
			row.OrderNo = n
		}
	}
	if row.OrderNo == 0 {
		row.OrderNo = excelRow - firstDataRow + 1
	}

	// Amount: tolerate numeric strings ("123456") or formatted ("1.234.567").
	amountStr := strings.ReplaceAll(get("amount"), ".", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amount, err := strconv.ParseInt(strings.TrimSpace(amountStr), 10, 64)
	if err != nil || amount < MinAmountVND {
		return row, fmt.Errorf("%w: row %d amount %q", ErrInvalidAmount, excelRow, get("amount"))
	}
	row.Amount = amount

	// SWIFT format validation (ISO 9362).
	if !swiftRe.MatchString(row.SwiftCode) {
		return row, fmt.Errorf("%w: row %d swift %q", ErrInvalidSwift, excelRow, row.SwiftCode)
	}

	// VFIC extraction from PaymentDetail.
	match := vficRe.FindString(row.PaymentDetail)
	if match == "" {
		return row, fmt.Errorf("%w: row %d", ErrMissingVFICCode, excelRow)
	}
	row.VFICCode = match

	return row, nil
}

// rowIsEmpty reports whether a raw cell slice is visually empty.
func (p *YeuCauChuyenTienParser) rowIsEmpty(raw []string) bool {
	for _, cell := range raw {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
