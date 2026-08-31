package excel

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/xuri/excelize/v2"
)

// STKRow represents parsed bank account information for an employee from the STK sheet
type STKRow struct {
	CCCD        string
	FullName    string
	BankAccount string
	BankName    string
	Note        string
	Mobile      string
}

// ParseSTKSheet searches for and parses the STK sheet in the provided Excel file.
// Returns a slice of STKRow. If the sheet is not found, returns (nil, nil).
func ParseSTKSheet(f *excelize.File) ([]STKRow, error) {
	// Match case-insensitively after trimming: partners frequently export the
	// STK sheet with surrounding whitespace (e.g. "STK ", " stk"), which would
	// otherwise miss the exact match and silently yield zero rows — so no
	// employee gets created from STK and the matching BCC rows report
	// "nhân viên không tìm thấy trong hệ thống".
	sheetName := ""
	for _, sheet := range f.GetSheetList() {
		if strings.EqualFold(strings.TrimSpace(sheet), "STK") {
			sheetName = sheet
			break
		}
	}
	if sheetName == "" {
		return nil, nil
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from STK sheet: %w", err)
	}

	if len(rows) < 2 {
		return nil, nil
	}

	// STK sheet layouts vary: some have a merged title on row 1 (so the column
	// header lands on row 2 and data on row 3), others place the header on
	// row 3 with data from row 4. Column ORDER varies too: the classic layout
	// is ID | Tên | Stk | Tên Ngân hàng | …, while e.g. the Samsung SDS
	// template ships TÊN | SỐ CCCD | TÊN NGÂN HÀNG | SỐ TÀI KHOẢN — reading
	// fixed positions there stores the bank name as the employee's name.
	// Detect the header row AND map each column from its label; fall back to
	// the historical fixed layout when no labelled header is found.
	headerRow := -1
	var cols stkColumnMap
	scanLimit := min(len(rows), 10)
	for i := range scanLimit {
		if c := detectSTKHeaderColumns(rows[i]); c != nil {
			headerRow = i
			cols = *c
			break
		}
	}

	var dataStart int
	if headerRow >= 0 {
		dataStart = headerRow + 1
	} else if loose := detectSTKHeaderRowLoose(rows); loose >= 0 {
		// Partially-labelled header (e.g. only "ID" present): keep the classic
		// fixed column order but honour the detected header position.
		dataStart = loose + 1
		cols = stkColumnMap{cccd: 1, name: 2, bankAccount: 3, bankName: 4, note: 5, mobile: 6}
	} else {
		// No header labels at all: preserve the historical assumption of 3
		// header rows (data from row 4) with the classic fixed column order.
		dataStart = 3
		cols = stkColumnMap{cccd: 1, name: 2, bankAccount: 3, bankName: 4, note: 5, mobile: 6}
	}

	var stkRows []STKRow
	for i := dataStart; i < len(rows); i++ {
		row := rows[i]
		// Interior blank spacer rows must not terminate parsing — partner
		// sheets use them as visual separators, and stopping here would
		// silently drop every row below. Rows carrying only a stray leading
		// cell (e.g. an orphan STT) are treated as spacers too.
		if len(row) <= 1 {
			continue
		}
		cccd := stkCell(row, cols.cccd)
		fullName := stkCell(row, cols.name)

		// Stop when both CCCD and Name are empty
		if cccd == "" && fullName == "" {
			break
		}
		if cccd == "" {
			continue
		}

		stkRows = append(stkRows, STKRow{
			CCCD:        cccd,
			FullName:    fullName,
			BankAccount: stkCell(row, cols.bankAccount),
			BankName:    stkCell(row, cols.bankName),
			Note:        stkCell(row, cols.note),
			Mobile:      sanitizeMobile(stkCell(row, cols.mobile)),
		})
	}

	return stkRows, nil
}

// detectSTKHeaderRowLoose replicates the original header detection: a row is a
// header when ANY ONE of the identity / name / account labels appears in
// columns B, C, or D. Used when full label mapping failed, so partially
// labelled sheets keep their detected data-start row instead of falling back
// to the fixed row-3 assumption.
func detectSTKHeaderRowLoose(rows [][]string) int {
	scanLimit := min(len(rows), 10)
	for i := range scanLimit {
		row := rows[i]
		colB, colC, colD := "", "", ""
		if len(row) > 1 {
			colB = normHeader(row[1])
		}
		if len(row) > 2 {
			colC = normHeader(row[2])
		}
		if len(row) > 3 {
			colD = normHeader(row[3])
		}
		isIDCol := colB == "id" || colB == "cccd" || strings.Contains(colB, "mã nhân viên")
		isNameCol := colC == "tên" || strings.Contains(colC, "họ tên") || strings.Contains(colC, "họ và tên")
		isSTKCol := colD == "stk" || colD == "số tk" || strings.Contains(colD, "số tài khoản") || strings.Contains(colD, "so tai khoan")
		if isIDCol || isNameCol || isSTKCol {
			return i
		}
	}
	return -1
}

// stkColumnMap holds 0-based column indexes into an STK sheet row.
type stkColumnMap struct {
	cccd        int
	name        int
	bankAccount int
	bankName    int
	note        int
	mobile      int
}

// stkCell reads one trimmed cell by index, tolerating short rows.
func stkCell(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

// detectSTKHeaderColumns maps a row to column positions by its header labels.
// Specific labels are claimed before generic ones so "tên ngân hàng" resolves
// to the bank column instead of the employee-name column. Returns nil when the
// row is not an STK header (missing identity or name column).
func detectSTKHeaderColumns(row []string) *stkColumnMap {
	cols := stkColumnMap{cccd: -1, name: -1, bankAccount: -1, bankName: -1, note: -1, mobile: -1}
	claimed := make(map[int]bool)

	claim := func(idx int, target *int) {
		if !claimed[idx] && *target == -1 {
			claimed[idx] = true
			*target = idx
		}
	}

	// Pass 1 — specific labels.
	for idx, cell := range row {
		v := normHeader(cell)
		if v == "" {
			continue
		}
		isAccount := v == "stk" || v == "số tk" || v == "so tk" ||
			strings.Contains(v, "số tài khoản") || strings.Contains(v, "so tai khoan") ||
			strings.Contains(v, "tài khoản") || strings.Contains(v, "tai khoan")
		isBank := strings.Contains(v, "ngân hàng") || strings.Contains(v, "ngan hang") ||
			v == "bank" || strings.Contains(v, "chi nhánh") || strings.Contains(v, "chi nhanh")
		if isAccount {
			claim(idx, &cols.bankAccount)
		} else if isBank {
			claim(idx, &cols.bankName)
		}
	}

	// Pass 2 — identity, then name, then optional columns.
	for idx, cell := range row {
		v := normHeader(cell)
		if v == "" || claimed[idx] {
			continue
		}
		isCCCD := v == "cccd" || v == "số cccd" || v == "so cccd" || v == "cmnd" ||
			v == "id" || strings.Contains(v, "mã nhân viên") || strings.Contains(v, "ma nhan vien")
		if isCCCD {
			claim(idx, &cols.cccd)
			continue
		}
		isName := v == "tên" || v == "ten" || v == "họ tên" || v == "ho ten" ||
			strings.Contains(v, "họ và tên") || strings.Contains(v, "ho va ten")
		if isName {
			claim(idx, &cols.name)
			continue
		}
		isMobile := v == "sdt" || v == "sđt" || v == "số đt" || v == "phone" ||
			strings.Contains(v, "điện thoại") || strings.Contains(v, "dien thoai") || strings.Contains(v, "số phone")
		if isMobile {
			claim(idx, &cols.mobile)
			continue
		}
		isNote := strings.Contains(v, "ghi chú") || strings.Contains(v, "ghi chu") || v == "note"
		if isNote {
			claim(idx, &cols.note)
		}
	}

	if cols.cccd == -1 || cols.name == -1 {
		return nil
	}
	return &cols
}

// sanitizeMobile strips non-digit characters (spaces, +, -, parens) so the
// value satisfies Employee.ValidateMobile (digits-only) regardless of how the
// partner formatted the phone in column G.
func sanitizeMobile(s string) string {
	digitsOnly := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, strings.TrimSpace(s))
	if len(digitsOnly) > 50 {
		slog.Warn("sanitizeMobile: phone exceeds employees.mobile varchar(50), dropping",
			"length", len(digitsOnly), "original", s)
		return ""
	}
	return digitsOnly
}
