package services

import (
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"

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
