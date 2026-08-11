package services

import (
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"

	"github.com/stretchr/testify/assert"
)

func TestGetShiftTypes(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.HC":      30000,
		"pho thong.ngay thuong.OT150":   150000,
		"pho thong.ngay nghi.HC":        45000,
		"co kinh nghiem.ngay thuong.HC": 43750,
	}

	shiftTypes := getShiftTypes(flatRates)
	assert.Contains(t, shiftTypes, "HC")
	assert.Contains(t, shiftTypes, "OT150")
	assert.Len(t, shiftTypes, 2)
}

func TestGetShiftTypes_Empty(t *testing.T) {
	assert.Empty(t, getShiftTypes(map[string]int{}))
}

func TestGetShiftTypes_ShortPaths(t *testing.T) {
	flatRates := map[string]int{
		"HC":     30000,
		"OT.HC":  45000,
		"a.b.HC": 50000,
	}
	shiftTypes := getShiftTypes(flatRates)
	assert.Contains(t, shiftTypes, "HC")
	assert.Len(t, shiftTypes, 1)
}

func TestDetermineDayType(t *testing.T) {
	tests := []struct {
		date     time.Time
		expected string
	}{
		{time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), "ngày thường"}, // Mon
		{time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC), "ngày thường"}, // Tue
		{time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC), "ngày thường"}, // Wed
		{time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC), "ngày thường"}, // Thu
		{time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC), "ngày thường"}, // Fri
		{time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC), "ngày nghỉ"},   // Sat
		{time.Date(2026, 6, 7, 0, 0, 0, 0, time.UTC), "ngày nghỉ"},   // Sun
	}
	for _, tt := range tests {
		t.Run(tt.date.Format("Mon_2006-01-02"), func(t *testing.T) {
			assert.Equal(t, tt.expected, determineDayType(tt.date))
		})
	}
}

func TestFindSTKRow(t *testing.T) {
	rows := []excelparser.STKRow{
		{CCCD: "031086013798", FullName: "Nguyễn Ngọc Minh", BankName: "MBank", BankAccount: "123456"},
		{CCCD: "030207001868", FullName: "Hà Huy Hoàng Dương", BankName: "VCB", BankAccount: "789012"},
	}

	// Found
	row := findSTKRow(rows, "031086013798")
	assert.NotNil(t, row)
	assert.Equal(t, "MBank", row.BankName)
	assert.Equal(t, "123456", row.BankAccount)

	// Second row
	row = findSTKRow(rows, "030207001868")
	assert.NotNil(t, row)
	assert.Equal(t, "VCB", row.BankName)

	// Not found
	row = findSTKRow(rows, "999999999999")
	assert.Nil(t, row)

	// Empty slice
	row = findSTKRow(nil, "031086013798")
	assert.Nil(t, row)
}

func TestWeeklyBCCMissingAssignmentError_SuppressesBlockedEmployee(t *testing.T) {
	blocked := map[string]struct{}{"031082006723": {}}
	reported := make(map[string]struct{})
	assert.True(t, isWeeklyBCCEmployeeBlocked(blocked, "031082006723"))
	assert.False(t, isWeeklyBCCEmployeeBlocked(blocked, "031082006724"))

	for range 5 {
		err := weeklyBCCMissingAssignmentError(
			"031082006723",
			"Nguyễn Văn Hương",
			blocked,
			reported,
		)
		assert.Nil(t, err)
	}
	assert.Empty(t, reported)
}

func TestWeeklyBCCMissingAssignmentError_ReportsUnknownEmployeeOnce(t *testing.T) {
	blocked := make(map[string]struct{})
	reported := make(map[string]struct{})

	first := weeklyBCCMissingAssignmentError(
		"031082006723",
		"Nguyễn Văn Hương",
		blocked,
		reported,
	)
	assert.Equal(t, &domain.ImportError{
		Employee: "Nguyễn Văn Hương",
		Reason:   "không tìm thấy nhân viên với CCCD \"031082006723\" trong dự án",
	}, first)

	second := weeklyBCCMissingAssignmentError(
		"031082006723",
		"Nguyễn Văn Hương",
		blocked,
		reported,
	)
	assert.Nil(t, second)
}

func TestWeeklyPaymentRateLookup_UsesSheetPositionAndShiftCode(t *testing.T) {
	flatRates := map[string]int{
		"Lương 520.ngày thường.HC":  65000,
		"Lương 520.ngày thường.TCN": 97500,
	}

	rates := buildShiftRatesForShift(flatRates, "HC")
	assert.NotEmpty(t, rates, "HC should produce rates")

	key := weeklyPaymentRateKey("Lương 520")
	assert.Equal(t, 65000, rates[key])
	assert.Empty(t, buildShiftRatesForShift(flatRates, "Lương 520HC"))
}

func TestWeeklyPaymentRateLookup_AlwaysUsesNgayThuong(t *testing.T) {
	flatRates := map[string]int{
		"Lương 520.ngày thường.HC":  65000,
		"Lương 520.ngày nghỉ.HC":    130000,
		"Lương 520.ngày thường.TCN": 97500,
	}

	rates := buildShiftRatesForShift(flatRates, "HC")
	assert.Equal(t, 65000, rates[weeklyPaymentRateKey("Lương 520")])
	assert.NotEqual(t, 130000, rates[weeklyPaymentRateKey("Lương 520")])
}

// TestBuildShiftRatesForShift_NoRateForKey tests that missing rates return empty map.
func TestBuildShiftRatesForShift_NoRateForKey(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.HC": 30000,
	}

	rates := buildShiftRatesForShift(flatRates, "TCN")
	assert.Empty(t, rates, "missing rate key should return empty rates map")
}

func TestWeeklyPaymentRateLookup_DistinguishesSheetPositions(t *testing.T) {
	flatRates := map[string]int{
		"Lương 520.ngày thường.HC": 65000,
		"Lương 700.ngày thường.HC": 87500,
	}

	rates := buildShiftRatesForShift(flatRates, "HC")
	assert.Equal(t, 65000, rates[weeklyPaymentRateKey("Lương 520")])
	assert.Equal(t, 87500, rates[weeklyPaymentRateKey("Lương 700")])
}

func TestBuildWeeklyPaymentPositions_UsesOwningSheet(t *testing.T) {
	sheets := []excelparser.WeeklyPaymentSheetData{
		{
			Position: "Lương 520",
			Employees: []excelparser.WeeklyPaymentEmployeeData{
				{EmployeeCode: "CCCD-520", FullName: "Nhân viên 520"},
			},
		},
		{
			Position: "Lương 700",
			Employees: []excelparser.WeeklyPaymentEmployeeData{
				{EmployeeCode: "CCCD-700", FullName: "Nhân viên 700"},
			},
		},
	}

	positions, blocked, errs := buildWeeklyPaymentPositions(sheets)

	assert.Equal(t, "Lương 520", positions["CCCD-520"])
	assert.Equal(t, "Lương 700", positions["CCCD-700"])
	assert.Empty(t, blocked)
	assert.Empty(t, errs)
}

func TestBuildWeeklyPaymentPositions_BlocksEmployeeInMultiplePositionSheets(t *testing.T) {
	sheets := []excelparser.WeeklyPaymentSheetData{
		{
			Position: "Lương 520",
			Employees: []excelparser.WeeklyPaymentEmployeeData{
				{EmployeeCode: "CCCD-DUP", FullName: "Nhân viên trùng"},
			},
		},
		{
			Position: "Lương 700",
			Employees: []excelparser.WeeklyPaymentEmployeeData{
				{EmployeeCode: "CCCD-DUP", FullName: "Nhân viên trùng"},
			},
		},
	}

	positions, blocked, errs := buildWeeklyPaymentPositions(sheets)

	assert.NotContains(t, positions, "CCCD-DUP")
	assert.Contains(t, blocked, "CCCD-DUP")
	if assert.Len(t, errs, 1) {
		assert.Contains(t, errs[0].Reason, "xuất hiện ở nhiều sheet vị trí")
	}
}

func TestPlanWeeklyPaymentPositionCorrections_SkipsExcludedFlexibleEmployees(t *testing.T) {
	assignments := []*domain.ProjectEmployee{
		{ID: 10, EmployeeID: 100, EmployeeCCCD: "CCCD-WEEKLY", EmployeeName: "Nhân viên tuần", Position: "Lương 520", PaymentSchedule: string(domain.PaymentScheduleWeekly)},
		{ID: 20, EmployeeID: 200, EmployeeCCCD: "CCCD-FLEX", EmployeeName: "Nhân viên linh hoạt", Position: "Lương 520", PaymentSchedule: string(domain.PaymentScheduleFlexible)},
	}
	positions := map[string]string{
		"CCCD-WEEKLY": "Lương 700",
		"CCCD-FLEX":   "Lương 750",
	}

	corrections := planWeeklyPaymentPositionCorrections(assignments, positions, false)

	if assert.Len(t, corrections, 1) {
		assert.Equal(t, uint(10), corrections[100].assignmentID)
		assert.Equal(t, "Lương 520", corrections[100].oldPosition)
		assert.Equal(t, "Lương 700", corrections[100].newPosition)
	}
	assert.NotContains(t, corrections, uint(200))
}

func TestSelectWeeklyPaymentPositionCorrections_OnlyEmployeesWithFinalEntries(t *testing.T) {
	planned := map[uint]posCorrection{
		100: {assignmentID: 20, employeeID: 100, oldPosition: "Lương 520", newPosition: "Lương 700"},
		200: {assignmentID: 10, employeeID: 200, oldPosition: "Lương 520", newPosition: "Lương 750"},
	}
	entries := []domainservices.BulkCreateTimesheetEntry{
		{EmployeeID: 100},
		{EmployeeID: 100},
	}

	selected := selectWeeklyPaymentPositionCorrections(entries, planned)

	if assert.Len(t, selected, 1) {
		assert.Equal(t, uint(20), selected[0].assignmentID)
		assert.Equal(t, uint(100), selected[0].employeeID)
	}
}
