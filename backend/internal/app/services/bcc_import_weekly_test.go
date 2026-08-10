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

// TestBuildShiftRatesForShift_PrefixedKey tests rate lookup with prefixed keys
// for the WeeklyPayment format (e.g., "520HC", "700NN").
func TestBuildShiftRatesForShift_PrefixedKey(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.520HC":  65000,
		"pho thong.ngay thuong.520TCN": 97500,
		"pho thong.ngay nghi.520NN":    130000,
		"pho thong.ngay nghi.520TCNN":  162500,
		"pho thong.ngay thuong.700HC":  87500,
	}

	// Test 520 tier prefix
	rates := buildShiftRatesForShift(flatRates, "520HC")
	assert.NotEmpty(t, rates, "520HC should produce rates")

	// Check that the rate key is formed correctly
	key := wbccRateKey{position: "phổ thông", dayType: "ngày thường"}
	assert.Contains(t, rates, key, "rates should contain position 'phổ thông' with day type 'ngày thường'")
	assert.Equal(t, 65000, rates[key], "520HC rate should be 65000")
}

// TestBuildShiftRatesForShift_WeekendVariant tests weekend shift codes with prefixes.
func TestBuildShiftRatesForShift_WeekendVariant(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay nghi.520NN":   130000,
		"pho thong.ngay nghi.520TCNN": 162500,
		"pho thong.ngay nghi.700NN":   175000,
	}

	// Test NN (weekend normal)
	rates := buildShiftRatesForShift(flatRates, "520NN")
	assert.NotEmpty(t, rates)

	key := wbccRateKey{position: "phổ thông", dayType: "ngày nghỉ"}
	assert.Contains(t, rates, key)
	assert.Equal(t, 130000, rates[key], "520NN rate should be 130000")

	// Test TCNN (weekend overtime)
	rates = buildShiftRatesForShift(flatRates, "520TCNN")
	assert.NotEmpty(t, rates)

	assert.Contains(t, rates, key)
	assert.Equal(t, 162500, rates[key], "520TCNN rate should be 162500")
}

// TestBuildShiftRatesForShift_ShiftLabelDrivesLookup tests that the shift label
// correctly determines the day type for rate lookup.
func TestBuildShiftRatesForShift_ShiftLabelDrivesLookup(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.520HC":  65000,
		"pho thong.ngay nghi.520NN":    130000,
		"pho thong.ngay thuong.520TCN": 97500,
		"pho thong.ngay nghi.520TCNN":  162500,
	}

	testCases := []struct {
		shiftKey      string
		expectedDayType string
		expectedRate   int
	}{
		{"520HC", "ngày thường", 65000},
		{"520TCN", "ngày thường", 97500},
		{"520NN", "ngày nghỉ", 130000},
		{"520TCNN", "ngày nghỉ", 162500},
	}

	for _, tc := range testCases {
		t.Run(tc.shiftKey, func(t *testing.T) {
			rates := buildShiftRatesForShift(flatRates, tc.shiftKey)
			assert.NotEmpty(t, rates)

			key := wbccRateKey{position: "phổ thông", dayType: tc.expectedDayType}
			assert.Contains(t, rates, key, "rates should contain expected day type")
			assert.Equal(t, tc.expectedRate, rates[key], "rate should match expected value")
		})
	}
}

// TestBuildShiftRatesForShift_NoRateForKey tests that missing rates return empty map.
func TestBuildShiftRatesForShift_NoRateForKey(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.HC": 30000,
	}

	// 520HC not in flatRates
	rates := buildShiftRatesForShift(flatRates, "520HC")
	assert.Empty(t, rates, "missing rate key should return empty rates map")
}

// TestBuildShiftRatesForShift_MultiTierTests tests multiple salary tiers.
func TestBuildShiftRatesForShift_MultiTierTests(t *testing.T) {
	flatRates := map[string]int{
		"pho thong.ngay thuong.520HC": 65000,
		"pho thong.ngay thuong.700HC": 87500,
		"pho thong.ngay thuong.750HC": 93750,
		"pho thong.ngay thuong.800HC": 100000,
		"pho thong.ngay thuong.900HC": 112500,
	}

	tiers := []string{"520", "700", "750", "800", "900"}
	expectedRates := []int{65000, 87500, 93750, 100000, 112500}

	for i, tier := range tiers {
		t.Run(tier+"HC", func(t *testing.T) {
			shiftKey := tier + "HC"
			rates := buildShiftRatesForShift(flatRates, shiftKey)
			assert.NotEmpty(t, rates)

			key := wbccRateKey{position: "phổ thông", dayType: "ngày thường"}
			assert.Contains(t, rates, key)
			assert.Equal(t, expectedRates[i], rates[key], "rate should match expected for tier "+tier)
		})
	}
}
