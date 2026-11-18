package excel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseWeeklyBlockFixture parses the committed weekly block fixture through
// the weekly BCC entry point, exactly as the import service calls it.
func parseWeeklyBlockFixture(t *testing.T) map[string]WeeklyBCCSheetData {
	t.Helper()
	f := openBCCFixture(t, "weekly_bcc", "lear_monthly.xlsx")

	data, err := ParseWeeklyBCCFile(f, []string{"BCC "})
	require.NoError(t, err)

	byShift := make(map[string]WeeklyBCCSheetData, len(data.Sheets))
	for _, s := range data.Sheets {
		_, dup := byShift[s.ShiftType]
		require.False(t, dup, "duplicate sheet group for label %q", s.ShiftType)
		byShift[s.ShiftType] = s
	}
	return byShift
}

// Label hour totals must equal the file's own row-37 summary goldens.
// CN 270% / Lễ 300% / Lễ 390% produce no group because every cell under them
// is blank — blank cells carry no entry (an explicit 0 would appear with a 0
// total), so those labels are absent from the output.
func TestParseWeeklyBlockBCC_LabelTotalsMatchFileSummary(t *testing.T) {
	byShift := parseWeeklyBlockFixture(t)

	require.Len(t, byShift, 5)
	want := map[string]float64{
		"HC 100%":  932,
		"OT 150%":  329,
		"Đêm 130%": 244,
		"Đêm 200%": 272,
		"CN 200%":  132,
	}
	got := make(map[string]float64, len(byShift))
	for shift, group := range byShift {
		var sum float64
		for _, emp := range group.Employees {
			for _, e := range emp.Entries {
				sum += e.Hours
			}
		}
		got[shift] = sum
	}
	assert.Equal(t, want, got)
}

// The "Mã nhân viên" column carries the citizen ID; it lands in
// WeeklyBCCEmployeeData.EmployeeCode and every entry date falls inside the
// sheet's calendar period.
func TestParseWeeklyBlockBCC_EmployeeIdentityAndDates(t *testing.T) {
	byShift := parseWeeklyBlockFixture(t)

	hc := byShift["HC 100%"]
	require.NotEmpty(t, hc.Employees)
	first := hc.Employees[0]
	assert.Equal(t, "025000000001", first.EmployeeCode)
	assert.Equal(t, "Nhan vien 01", first.FullName)

	novStart := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
	novEnd := time.Date(2026, time.November, 30, 23, 59, 59, 0, time.UTC)
	entryCount := 0
	for _, group := range byShift {
		for _, emp := range group.Employees {
			for _, e := range emp.Entries {
				entryCount++
				assert.False(t, e.Date.Before(novStart) || e.Date.After(novEnd),
					"entry date %s outside the sheet period", e.Date)
			}
		}
	}
	assert.Greater(t, entryCount, 0)
}

// Enhanced Saturdays ("T7 TC") carry OT 150% / Đêm 200% in their 2nd/4th
// sub-columns with the 1st/3rd cells unlabeled — hours there are skipped by
// design, and the labeled sub-columns' dates must still resolve to the
// Saturday via the block anchor.
func TestParseWeeklyBlockBCC_T7TCBlocks(t *testing.T) {
	byShift := parseWeeklyBlockFixture(t)

	ot150, dem200 := 0.0, 0.0
	for _, emp := range byShift["OT 150%"].Employees {
		for _, e := range emp.Entries {
			if e.Date.Day() == 14 && e.Date.Month() == time.November {
				ot150 += e.Hours
			}
		}
	}
	for _, emp := range byShift["Đêm 200%"].Employees {
		for _, e := range emp.Entries {
			if e.Date.Day() == 14 && e.Date.Month() == time.November {
				dem200 += e.Hours
			}
		}
	}
	assert.Greater(t, ot150, 0.0, "14/11 T7 TC block must carry OT 150% hours")
	assert.Greater(t, dem200, 0.0, "14/11 T7 TC block must carry Đêm 200% hours")
}
