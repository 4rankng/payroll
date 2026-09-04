package excel

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// TestParseBCCData_ZeroCellFixture locks the zero-cell contract against a
// real BUMHAN-shaped workbook (anonymized single-employee copy of the partner
// Thang8 file): an explicit 0 must survive parsing as a zero-hour entry — the
// import pipeline reads it as a deletion request for that day's chờ duyệt
// timesheet — while blank cells produce no entry. The sheet keeps its "BCC"
// name with a date-row layout, so this also guards the strategy router: the
// date-row parser must win over the legacy shape-keyed parser.
func TestParseBCCData_ZeroCellFixture(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	path := filepath.Clean(filepath.Join(cwd, "..", "..", "..", "..", "tests", "fixtures", "bcc-zero", "bumhan-single-employee.xlsx"))
	f, err := excelize.OpenFile(path)
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer func() { _ = f.Close() }()

	data, winner, err := ParseBCCData(f)
	if err != nil {
		t.Fatalf("ParseBCCData: %v", err)
	}
	if winner != FormatDateRow {
		t.Fatalf("winning format = %v, want FormatDateRow (date-row layout under a BCC sheet name)", winner)
	}
	if len(data.Employees) != 1 {
		t.Fatalf("employees = %d, want 1", len(data.Employees))
	}

	emp := data.Employees[0]
	type cell struct {
		day   string // dd/mm as read from the file's date row
		label string
		hours float64
	}
	got := make([]cell, 0, len(emp.Entries))
	for _, e := range emp.Entries {
		if e.FullDate == nil {
			t.Fatalf("entry {day=%d %q} has no FullDate — date-row parser must set it", e.DayNum, e.ShiftLabel)
		}
		got = append(got, cell{e.FullDate.Format("02/01"), e.ShiftLabel, e.Hours})
	}

	// Row layout: 21/08 (NT, OT) blank; 22/08 T7=8, OT T7=0; 23/08 CN=0,
	// OT CN=0; 24/08 NT=9, OT=2; … Blank cells must not appear; the three
	// explicit zeros must.
	want := []cell{
		{"22/08", "T7", 8},
		{"22/08", "OT T7", 0},
		{"23/08", "CN", 0},
		{"23/08", "OT CN", 0},
		{"24/08", "NT", 9},
		{"24/08", "OT", 2},
	}
	if len(got) < len(want) {
		t.Fatalf("entries = %#v, want at least the first six (blanks dropped, zeros kept)", got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("entry %d = {%s %s %.1f}, want {%s %s %.1f}", i, got[i].day, got[i].label, got[i].hours, w.day, w.label, w.hours)
		}
	}
	for _, g := range got {
		if g.day == "21/08" {
			t.Errorf("21/08 entry {%s %.1f} exists — blank cells must not produce entries", g.label, g.hours)
		}
	}
}
