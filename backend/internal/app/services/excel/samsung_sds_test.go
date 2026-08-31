package excel

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

// The Samsung SDS weekly template (T8.2026) broke parsing in three ways this
// file locks in:
//   - STK sheet ships TÊN | SỐ CCCD | TÊN NGÂN HÀNG | SỐ TÀI KHOẢN — column
//     order differs from the classic layout, so columns must be mapped from
//     header labels, not fixed positions (otherwise the bank name becomes the
//     employee name).
//   - BCC date row holds real Excel date serials (2026-08-24 → 46258) followed
//     by stale plain integers 29/30/31 in the tail; serials must resolve to
//     their day of month and must win over the Atoi path.
//   - No VND rate row exists anywhere (row 9 is a weekday row), so
//     ShiftRates stays empty.
const samsungAug24Serial = 46258 // 2026-08-24

// newSamsungSDSWorkbook builds a synthetic workbook with the structural shape
// of the Samsung SDS template (junk company rows 1-7, header row 8, weekday
// row 9, shift-label row 10, data rows 11+).
func newSamsungSDSWorkbook(t *testing.T) *excelize.File {
	t.Helper()
	f := excelize.NewFile()
	const sheet = "BCC"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		t.Fatalf("new sheet: %v", err)
	}
	f.SetActiveSheet(idx)
	_ = f.DeleteSheet("Sheet1")

	set := func(cell, val string) {
		t.Helper()
		if err := f.SetSheetRow(sheet, cell, &[]any{val}); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}

	// Header row 8: STT | Mã nhân viên (=CCCD) | Họ và tên | Chức danh | dates…
	set("A8", "STT")
	set("B8", "Mã nhân viên")
	set("C8", "Họ và tên")
	set("D8", "Chức danh")
	// Dates as full serials, one per 2-column span (CB|OT); tail columns hold
	// stale plain integers 29/30 as sometimes left by the partner's Excel.
	for i, day := 0, 0; day < 3; day++ {
		col, _ := excelize.CoordinatesToCellName(5+i, 8)
		if err := f.SetSheetRow(sheet, col, &[]any{float64(samsungAug24Serial + day)}); err != nil {
			t.Fatalf("set date %s: %v", col, err)
		}
		i += 2
	}
	staleCol, _ := excelize.CoordinatesToCellName(11, 8)
	if err := f.SetSheetRow(sheet, staleCol, &[]any{float64(29)}); err != nil {
		t.Fatalf("set stale tail: %v", err)
	}
	// Weekday row 9 (stale labels — must not be trusted for day type).
	set("E9", "T4")
	// Shift labels row 10: CB|OT per date, then CN|OT CN on the stale tail.
	set("E10", "CB")
	set("F10", "OT")
	set("G10", "CB")
	set("H10", "OT")
	set("I10", "CB")
	set("J10", "OT")
	set("K10", "CN")
	set("L10", "OT CN")
	// Data rows: CCCD in col B, hours in the CB columns.
	set("B11", "031082017000")
	set("C11", "Đào Ngọc Tiến")
	set("D11", "Chia chọn")
	_ = f.SetSheetRow(sheet, "E11", &[]any{nil, nil, float64(8), nil, float64(8)})
	set("B12", "031083004278")
	set("C12", "Trần Quang Huy")
	set("D12", "Chia chọn")
	_ = f.SetSheetRow(sheet, "I12", &[]any{float64(8)})

	// STK sheet with the Samsung column order.
	setStk := func(cell, val string) {
		t.Helper()
		if err := f.SetSheetRow("STK", cell, &[]any{val}); err != nil {
			t.Fatalf("set STK %s: %v", cell, err)
		}
	}
	if _, err := f.NewSheet("STK"); err != nil {
		t.Fatalf("new STK sheet: %v", err)
	}
	setStk("A1", "TÊN")
	setStk("B1", "SỐ CCCD")
	setStk("C1", "TÊN NGÂN HÀNG")
	setStk("D1", "SỐ TÀI KHOẢN")
	setStk("A2", "Đào Ngọc Tiến")
	setStk("B2", "031082017000")
	setStk("C2", "MB BANK")
	setStk("D2", "0936855779")
	setStk("A3", "Trần Quang Huy")
	setStk("B3", "031083004278")
	// Row 3 has no bank info — must still parse as a person.

	return f
}

func TestParseSTKSheet_SamsungColumnLayout(t *testing.T) {
	f := newSamsungSDSWorkbook(t)
	t.Cleanup(func() { _ = f.Close() })

	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 STK rows, got %d: %+v", len(rows), rows)
	}

	first := rows[0]
	if first.FullName != "Đào Ngọc Tiến" {
		t.Errorf("FullName = %q, want employee name from col A (not the bank name)", first.FullName)
	}
	if first.CCCD != "031082017000" {
		t.Errorf("CCCD = %q, want value from SỐ CCCD col B", first.CCCD)
	}
	if first.BankName != "MB BANK" {
		t.Errorf("BankName = %q, want %q from col C", first.BankName, "MB BANK")
	}
	if first.BankAccount != "0936855779" {
		t.Errorf("BankAccount = %q, want %q from col D", first.BankAccount, "0936855779")
	}

	second := rows[1]
	if second.FullName != "Trần Quang Huy" {
		t.Errorf("bank-less row lost: FullName = %q, want Trần Quang Huy", second.FullName)
	}
	if second.BankAccount != "" || second.BankName != "" {
		t.Errorf("bank-less row should have empty bank fields, got %+v", second)
	}
}

func TestParseSTKSheet_ClassicColumnLayout(t *testing.T) {
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	if _, err := f.NewSheet("STK"); err != nil {
		t.Fatalf("new STK sheet: %v", err)
	}
	_ = f.DeleteSheet("Sheet1")
	header := []any{"STT", "ID", "Tên", "Stk", "Tên Ngân hàng", "Ghi chú", "SDT"}
	data := []any{"1", "031082017000", "Đào Ngọc Tiến", "0936855779", "MB", "", "0936855779"}
	if err := f.SetSheetRow("STK", "A3", &header); err != nil {
		t.Fatalf("header: %v", err)
	}
	if err := f.SetSheetRow("STK", "A4", &data); err != nil {
		t.Fatalf("data: %v", err)
	}

	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.CCCD != "031082017000" || r.FullName != "Đào Ngọc Tiến" ||
		r.BankAccount != "0936855779" || r.BankName != "MB" || r.Mobile != "0936855779" {
		t.Errorf("classic layout misparsed: %+v", r)
	}
}

// TestParseSTKSheet_BlankSpacerRow keeps parsing past an interior blank row:
// partner sheets use blank separators, and stopping at the first one silently
// drops every row below (the "nhân viên không tìm thấy" failure class).
func TestParseSTKSheet_BlankSpacerRow(t *testing.T) {
	f := newSamsungSDSWorkbook(t)
	t.Cleanup(func() { _ = f.Close() })

	// Append a third data row AFTER a blank spacer row.
	if err := f.SetSheetRow("STK", "A5", &[]any{nil, nil, nil, nil}); err != nil {
		t.Fatalf("spacer: %v", err)
	}
	if err := f.SetSheetRow("STK", "A6", &[]any{"Bùi Văn Vịnh", "031205016269", "MB BANK", "123456789"}); err != nil {
		t.Fatalf("row: %v", err)
	}

	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows (spacer must not terminate parsing), got %d: %+v", len(rows), rows)
	}
	if rows[2].FullName != "Bùi Văn Vịnh" || rows[2].CCCD != "031205016269" {
		t.Errorf("row after spacer misparsed: %+v", rows[2])
	}
}

// TestParseSTKSheet_PartialHeaderKeepsDataStart guards partially-labelled
// headers: a lone recognizable label must still position the data start (as
// legacy detection did) instead of falling back to the fixed row-3 assumption.
func TestParseSTKSheet_PartialHeaderKeepsDataStart(t *testing.T) {
	f := excelize.NewFile()
	t.Cleanup(func() { _ = f.Close() })
	if _, err := f.NewSheet("STK"); err != nil {
		t.Fatalf("new STK sheet: %v", err)
	}
	_ = f.DeleteSheet("Sheet1")
	// Header at row 2 with only the ID label; classic column order below it.
	if err := f.SetSheetRow("STK", "A2", &[]any{"STT", "ID"}); err != nil {
		t.Fatalf("header: %v", err)
	}
	if err := f.SetSheetRow("STK", "A3", &[]any{"1", "031082017000", "Đào Ngọc Tiến", "0936855779", "MB"}); err != nil {
		t.Fatalf("data: %v", err)
	}

	rows, err := ParseSTKSheet(f)
	if err != nil {
		t.Fatalf("ParseSTKSheet: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d: %+v", len(rows), rows)
	}
	if rows[0].FullName != "Đào Ngọc Tiến" || rows[0].CCCD != "031082017000" {
		t.Errorf("partial-header sheet misparsed: %+v", rows[0])
	}
}

func TestParseBCCFile_SamsungSDSTemplate(t *testing.T) {
	f := newSamsungSDSWorkbook(t)
	t.Cleanup(func() { _ = f.Close() })

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}

	if len(data.ShiftRates) != 0 {
		t.Errorf("template has no rate row; ShiftRates = %v", data.ShiftRates)
	}
	if len(data.Employees) != 2 {
		t.Fatalf("expected 2 employees, got %d", len(data.Employees))
	}

	first := data.Employees[0]
	if first.CCCD != "031082017000" {
		t.Errorf("CCCD = %q (must come from Mã nhân viên col B)", first.CCCD)
	}
	if first.FullName != "Đào Ngọc Tiến" {
		t.Errorf("FullName = %q", first.FullName)
	}
	if first.Department != "Chia chọn" {
		t.Errorf("Department = %q, want chức danh value", first.Department)
	}
	// Hours sit in the CB columns of Aug 25 (G11) and Aug 26 (I11).
	gotDays := map[int]float64{}
	for _, e := range first.Entries {
		gotDays[e.DayNum] += e.Hours
	}
	if len(first.Entries) != 2 || gotDays[25] != 8 || gotDays[26] != 8 {
		t.Errorf("employee 0 entries wrong: %+v (days map %v)", first.Entries, gotDays)
	}
	for _, e := range first.Entries {
		if e.ShiftLabel != "CB" {
			t.Errorf("entry day %d label = %q, want CB", e.DayNum, e.ShiftLabel)
		}
	}

	second := data.Employees[1]
	if len(second.Entries) != 1 || second.Entries[0].DayNum != 26 {
		t.Errorf("employee 1 entries wrong: %+v", second.Entries)
	}
}

// TestParseBCCFile_DayIntegersNotSerials guards the serial-vs-day-int order:
// a raw 22 must stay day 22 (legacy dd-formatted day cells), while a real
// 2026 serial must resolve to its day of month.
func TestParseBCCFile_DayIntegersNotSerials(t *testing.T) {
	f := excelize.NewFile()
	const sheet = "BCC"
	if _, err := f.NewSheet(sheet); err != nil {
		t.Fatalf("new sheet: %v", err)
	}
	_ = f.DeleteSheet("Sheet1")
	_ = f.SetSheetRow(sheet, "A8", &[]any{"STT", "Mã NV", "Họ và tên"})
	_ = f.SetSheetRow(sheet, "E8", &[]any{float64(22), nil, float64(46259)})
	_ = f.SetSheetRow(sheet, "E10", &[]any{"CB", "OT", "CB", "OT"})
	_ = f.SetCellValue(sheet, "C11", "Người Một")
	_ = f.SetSheetRow(sheet, "E11", &[]any{float64(8), nil, float64(8)})
	t.Cleanup(func() { _ = f.Close() })

	data, err := ParseBCCFile(f)
	if err != nil {
		t.Fatalf("ParseBCCFile: %v", err)
	}
	if len(data.Employees) != 1 {
		t.Fatalf("employees: %d", len(data.Employees))
	}
	days := map[int]bool{}
	for _, e := range data.Employees[0].Entries {
		days[e.DayNum] = true
	}
	// 22 is a plain day number; serial 46259 = 2026-08-25 → day 25.
	if !days[22] {
		t.Errorf("day 22 (plain int) missing, got %v", days)
	}
	if !days[25] {
		t.Errorf("day 25 (from date serial 46259) missing, got %v", days)
	}
	if days[46259] || days[59] {
		t.Errorf("serial leaked into day number: %v", days)
	}
}
