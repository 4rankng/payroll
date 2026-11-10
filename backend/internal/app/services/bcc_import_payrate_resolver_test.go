package services

import (
	"context"
	"strings"
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"

	"github.com/stretchr/testify/assert"
)

// testPayrate builds a payrate config effective from `from` with the given
// payrate JSON (flattened by newPayrateResolver exactly like production).
func testPayrate(id uint, from time.Time, rates string) *domain.Payrate {
	return &domain.Payrate{
		ID:       id,
		Payrate:  domain.PayrateConfiguration(rates),
		FromDate: from,
	}
}

// mustTestResolver builds the shared per-date payrate resolver from configs.
func mustTestResolver(t *testing.T, loc *time.Location, configs ...*domain.Payrate) *payrateResolver {
	t.Helper()
	r, err := newPayrateResolver(configs, loc)
	if err != nil {
		t.Fatalf("newPayrateResolver: %v", err)
	}
	return r
}

func oct2026(loc *time.Location) (time.Time, time.Time) {
	start := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, -1)
}

// TestPayrateResolverConfigForNearestStartDate pins the owner-mandated
// resolution rule (PAYR-36): the config for a work date is the one with the
// NEAREST from_date at or before that date — never a file-level anchor —
// with same-date ties broken by the higher id (matching the SQL of
// GetActiveByProjectAndDate).
func TestPayrateResolverConfigForNearestStartDate(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	r := mustTestResolver(t, loc,
		testPayrate(1, d(2026, time.September, 1), `{"A.x.y":1}`),
		testPayrate(2, d(2026, time.October, 15), `{"B.x.y":2}`),
		testPayrate(4, d(2026, time.October, 15), `{"D.x.y":4}`), // same-day tie: higher id wins
		testPayrate(3, d(2026, time.November, 1), `{"C.x.y":3}`),
	)
	cases := []struct {
		day    string
		wantID uint
		wantOK bool
	}{
		{"2026-08-25", 0, false}, // before every config
		{"2026-09-05", 1, true},
		{"2026-10-14", 1, true}, // still config A the day before the switch
		{"2026-10-15", 4, true}, // switch day: new config; tie → higher id
		{"2026-10-31", 4, true}, // tied id 4 wins through end of month
		{"2026-11-01", 3, true}, // November config takes over
		{"2027-01-01", 3, true}, // latest config keeps serving
	}
	for _, tc := range cases {
		day, _ := time.Parse("2006-01-02", tc.day)
		got, ok := r.configFor(day)
		if ok != tc.wantOK || (ok && got.id != tc.wantID) {
			t.Errorf("configFor(%s) = id %d ok=%v, want id %d ok=%v", tc.day, got.id, ok, tc.wantID, tc.wantOK)
		}
	}
}

// TestPayrateResolverMonthPositionsUnionMidMonth pins the PAYR-36 union rule:
// the position vocabulary of an import month is the union across the configs
// effective at some day of that month — a position added to a config
// mid-month is valid for a full-month file, a config superseded before the
// month never contributes.
func TestPayrateResolverMonthPositionsUnionMidMonth(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	r := mustTestResolver(t, loc,
		testPayrate(10, d(2026, time.August, 1), `{"Đã xóa.ngày thường.HC":10000}`),
		testPayrate(1, d(2026, time.September, 25), `{"Phổ thông.ngày thường.HC":30000}`),
		testPayrate(2, d(2026, time.October, 15), `{"Phổ thông.ngày thường.HC":30000,"Ca đêm.ngày thường.HC":50000}`),
		testPayrate(3, d(2026, time.November, 1), `{"Phổ thông.ngày thường.HC":32000,"Ca đêm.ngày thường.HC":52000}`),
	)
	monthStart, monthEnd := oct2026(loc)

	avail := r.monthPositions(monthStart, monthEnd)

	caDem, ok := avail[bccNormName("Ca đêm")]
	if !ok {
		t.Fatal("position added to the config mid-month (Ca đêm, effective 15/10) must be in the month union")
	}
	if !caDem.availableFrom.Equal(d(2026, time.October, 15)) {
		t.Errorf("Ca đêm availableFrom = %s, want 15/10 (the config day it became priceable)", caDem.availableFrom.Format("02/01"))
	}
	phoThong, ok := avail[bccNormName("Phổ thông")]
	if !ok {
		t.Fatal("position carried into October by the 25/09 config must be in the month union")
	}
	if !phoThong.availableFrom.Equal(monthStart) {
		t.Errorf("Phổ thông availableFrom = %s, want month start (config began before the month)", phoThong.availableFrom.Format("02/01"))
	}
	if _, ok := avail[bccNormName("Đã xóa")]; ok {
		t.Fatal("position from a config superseded before the month start (Đã xóa) must NOT be in the month union")
	}

	assert.ElementsMatch(t, []string{"Ca đêm", "Phổ thông"}, r.positionNamesForMonth(monthStart, monthEnd))
}

// TestPayrateResolverMissingPositionMessage pins the unified actionable error
// (PAYR-36): the position-missing text fires only when the position is absent
// from EVERY config effective in the month, names the availability date when
// one exists anywhere in the timeline, and always lists the month's available
// positions.
func TestPayrateResolverMissingPositionMessage(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	r := mustTestResolver(t, loc,
		testPayrate(10, d(2026, time.August, 1), `{"Đã xóa.ngày thường.HC":10000}`),
		testPayrate(1, d(2026, time.October, 1), `{"Phổ thông.ngày thường.HC":30000,"Ca đêm.ngày thường.HC":50000}`),
		testPayrate(5, d(2026, time.December, 1), `{"Tương lai.ngày thường.HC":60000}`),
	)
	monthStart, monthEnd := oct2026(loc)

	msg := r.missingPositionMessage("Bốc xếp", monthStart, monthEnd)
	for _, want := range []string{
		`Vị trí "Bốc xếp" chưa có trong bảng lương của dự án`,
		"cần cập nhật bảng lương hiệu lực từ đầu kỳ hoặc import theo kỳ config phù hợp",
		"Các vị trí khả dụng trong tháng 10/2026",
		"Ca đêm, Phổ thông",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
	if strings.Contains(msg, "sẽ có từ") || strings.Contains(msg, "đã có từ") {
		t.Errorf("message for a never-configured position must carry no availability hint: %q", msg)
	}

	if msg := r.missingPositionMessage("Tương lai", monthStart, monthEnd); !strings.Contains(msg, "sẽ có từ 01/12") {
		t.Errorf("future-config position must hint its start date, got %q", msg)
	}
	if msg := r.missingPositionMessage("Đã xóa", monthStart, monthEnd); !strings.Contains(msg, "đã có từ 01/08 nhưng không còn hiệu lực trong tháng 10/2026") {
		t.Errorf("superseded position must say it no longer applies, got %q", msg)
	}
}

// TestBuildMultiPositionEntriesResolvesRatesPerWorkDate pins PAYR-36 on the
// multi-position path: (a) a position added to the config mid-month imports
// fine from a full-month file, and (c) rows before/after the config switch
// are priced by the config active on THEIR OWN work date.
func TestBuildMultiPositionEntriesResolvesRatesPerWorkDate(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	resolver := mustTestResolver(t, loc,
		// Config 01/10: only Phổ thông, priced as "ca ngày".
		testPayrate(1, d(2026, time.October, 1), `{"Phổ thông.ngày thường.ca ngày":30000}`),
		// Config 15/10: adds the Ca đêm position mid-month and re-keys
		// Phổ thông to a different leaf — the day-20 row must follow it.
		testPayrate(2, d(2026, time.October, 15), `{"Phổ thông.ngày thường.ca đêm":32000,"Ca đêm.ngày thường.ca đêm":50000}`),
	)
	parsed := &excelparser.MultiPositionImportData{Sheets: []excelparser.PositionSheetData{
		{Position: "Phổ thông", Employees: []excelparser.PositionEmployeeData{{
			EmployeeCode: "012345678901", FullName: "NV Thường",
			Entries: []excelparser.PositionEntryData{
				{DayNum: 5, RateVND: 30000, Hours: 8},
				{DayNum: 20, RateVND: 32000, Hours: 8},
			},
		}}},
		{Position: "Ca đêm", Employees: []excelparser.PositionEmployeeData{{
			EmployeeCode: "012345678902", FullName: "NV Đêm",
			Entries: []excelparser.PositionEntryData{{DayNum: 20, RateVND: 50000, Hours: 8}},
		}}},
	}}
	byCCCDAndPosition := map[string]*domain.ProjectEmployee{
		"012345678901|phổ thông": {EmployeeID: 1, EmployeeCCCD: "012345678901", EmployeeName: "NV Thường", Position: "Phổ thông", PaymentSchedule: string(domain.PaymentScheduleWeekly)},
		"012345678902|ca đêm":    {EmployeeID: 2, EmployeeCCCD: "012345678902", EmployeeName: "NV Đêm", Position: "Ca đêm", PaymentSchedule: string(domain.PaymentScheduleWeekly)},
	}

	entries, _, totalRows, errs := (&BCCImportService{}).buildMultiPositionEntries(
		context.Background(), parsed, resolver, 2026, time.October, loc,
		map[string]*domain.ProjectEmployee{}, map[string]*domain.ProjectEmployee{}, byCCCDAndPosition,
		7, 9, false)

	assert.Empty(t, errs, "mid-month added position + full-month file must import without errors")
	assert.Equal(t, 2, totalRows)
	if !assert.Len(t, entries, 3) {
		return
	}
	ngayThuong := "ngày thường"
	assert.Equal(t, domainservices.BulkCreateTimesheetEntry{ProjectID: 7, EmployeeID: 1, Date: "2026-10-05", HoursWorked: 8, HourType: "ca ngày", DayType: &ngayThuong}, entries[0])
	assert.Equal(t, domainservices.BulkCreateTimesheetEntry{ProjectID: 7, EmployeeID: 1, Date: "2026-10-20", HoursWorked: 8, HourType: "ca đêm", DayType: &ngayThuong}, entries[1])
	assert.Equal(t, domainservices.BulkCreateTimesheetEntry{ProjectID: 7, EmployeeID: 2, Date: "2026-10-20", HoursWorked: 8, HourType: "ca đêm", DayType: &ngayThuong}, entries[2])
}

// TestWeeklyBCCConfigErrorPerWorkDate pins the weekly path on the same
// per-date rule (PAYR-36): shift validation resolves the config of each
// worked date through the shared resolver — a shift absent on an early
// worked date still fails that date even when a mid-month config adds it,
// and passes once all worked dates carry it.
func TestWeeklyBCCConfigErrorPerWorkDate(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	resolver := mustTestResolver(t, loc,
		testPayrate(1, d(2026, time.October, 1), `{"Phổ thông.ngày thường.NN":30000}`),
		testPayrate(2, d(2026, time.October, 15), `{"Phổ thông.ngày thường.NN":30000,"Phổ thông.ngày thường.HC":50000}`),
	)

	sheet := func(shift string, days ...int) *excelparser.WeeklyBCCImportData {
		entries := make([]excelparser.WeeklyBCCEntryData, 0, len(days))
		for _, day := range days {
			entries = append(entries, excelparser.WeeklyBCCEntryData{Date: d(2026, time.October, day), Hours: 2})
		}
		return &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
			ShiftType: shift,
			Employees: []excelparser.WeeklyBCCEmployeeData{{FullName: "Mai Ngọc Trung", Entries: entries}},
		}}}
	}

	if reason := weeklyBCCConfigError(sheet("HC", 20, 21), 2026, time.October, loc, resolver.ratesFor); reason != "" {
		t.Fatalf("shift configured by every worked date's own config must pass, got: %s", reason)
	}
	reason := weeklyBCCConfigError(sheet("HC", 6), 2026, time.October, loc, resolver.ratesFor)
	if !strings.Contains(reason, `ca làm "HC" không có trong cấu hình lương cho ngày 2026-10-06`) {
		t.Fatalf("shift missing on a worked date must fail that date, got: %s", reason)
	}
}

// TestBuildWeeklyPaymentEntriesRatesFollowWorkDate pins the weekly-payment
// path (PAYR-36): rate resolution follows each row's own work date through
// the shared resolver — the pre-switch date prices under the old config,
// the post-switch date demands the new one.
func TestBuildWeeklyPaymentEntriesRatesFollowWorkDate(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	resolver := mustTestResolver(t, loc,
		testPayrate(1, d(2026, time.October, 1), `{"Lương 520.ngày thường.HC":65000}`),
		testPayrate(2, d(2026, time.October, 15), `{"Lương 520.ngày thường.TCN":97500}`), // HC leaf removed mid-month
	)
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	byCCCD["031207007051"].Position = "Lương 520"
	parsed := &excelparser.WeeklyPaymentImportData{Sheets: []excelparser.WeeklyPaymentSheetData{{
		Position: "Lương 520",
		Employees: []excelparser.WeeklyPaymentEmployeeData{{
			FullName: "Mai Ngọc Trung",
			Entries: []excelparser.WeeklyPaymentEntryData{
				{Day: 6, ShiftKey: "HC", Hours: 2},  // under config 1: HC exists
				{Day: 20, ShiftKey: "HC", Hours: 2}, // under config 2: HC was removed
			},
		}},
	}}}

	entries, _, totalRows, errs := (&BCCImportService{}).buildWeeklyPaymentEntries(
		context.Background(), parsed, 2026, time.October, loc,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		resolver.ratesFor, 7, false)

	assert.Equal(t, 1, totalRows) // one employee row; two entries under it
	if !assert.Len(t, entries, 1) {
		return
	}
	assert.Equal(t, "2026-10-06", entries[0].Date)
	assert.Equal(t, "HC", entries[0].HourType)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, "không tìm thấy mức lương cho ca HC, vị trí Lương 520, ngày 2026-10-20", errs[0].Reason)
	}
}

// TestBCCCrossMonthError pins the one-file-per-month contract (PAYR-36 owner
// ruling): worked full dates outside the import month fail with the
// split-file guidance and the stray days, never with a misleading position
// or rate error.
func TestBCCCrossMonthError(t *testing.T) {
	loc := time.Local
	d := func(y int, m time.Month, day int) time.Time {
		return time.Date(y, m, day, 0, 0, 0, 0, loc)
	}
	year, month := 2026, time.October

	cases := []struct {
		name        string
		dates       []time.Time
		wantEmpty   bool
		wantStrays  []string
		notContains string
	}{
		{
			name:      "all within the import month",
			dates:     []time.Time{d(2026, time.October, 29), d(2026, time.October, 31)},
			wantEmpty: true,
		},
		{
			name:       "payroll cycle straddling into November",
			dates:      []time.Time{d(2026, time.October, 30), d(2026, time.November, 7), d(2026, time.November, 1), d(2026, time.November, 3), d(2026, time.November, 2), d(2026, time.November, 6), d(2026, time.November, 4), d(2026, time.November, 5)},
			wantStrays: []string{"tách file theo từng tháng (29-31/10 một file, 1-7/11 một file)", "1-7/11/2026", "tháng 10/2026"},
		},
		{
			name:       "non-contiguous strays render as separate runs",
			dates:      []time.Time{d(2026, time.November, 5), d(2026, time.November, 1), d(2026, time.November, 3), d(2026, time.November, 2)},
			wantStrays: []string{"1-3/11/2026, 5/11/2026"},
		},
		{
			name:        "no stray list when everything is in month",
			dates:       []time.Time{d(2026, time.October, 5)},
			wantEmpty:   true,
			notContains: "Các ngày bị lẫn",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reason := bccCrossMonthError(tc.dates, year, month, loc)
			if tc.wantEmpty {
				if reason != "" {
					t.Fatalf("expected no error, got: %s", reason)
				}
				return
			}
			if reason == "" {
				t.Fatal("expected split-file error, got empty")
			}
			for _, want := range tc.wantStrays {
				if !strings.Contains(reason, want) {
					t.Errorf("reason %q missing %q", reason, want)
				}
			}
			if tc.notContains != "" && strings.Contains(reason, tc.notContains) {
				t.Errorf("reason %q must not contain %q", reason, tc.notContains)
			}
		})
	}
}
