package services

import (
	"context"
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"
	"api-server/internal/domain"
	domainservices "api-server/internal/domain/services"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/unicode/norm"
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
			false,
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
		false,
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
		false,
		blocked,
		reported,
	)
	assert.Nil(t, second)
}

// A row whose "Mã nhân viên" cell is blank names the blank cell in its reason
// instead of quoting an empty CCCD; ambiguous names and missing names read
// differently, and each name-only employee is reported exactly once.
func TestWeeklyBCCMissingAssignmentError_BlankCCCDNamesTheCell(t *testing.T) {
	blocked := make(map[string]struct{})
	reported := make(map[string]struct{})

	unknown := weeklyBCCMissingAssignmentError("", "Mai Ngọc Trung", false, blocked, reported)
	assert.Equal(t, &domain.ImportError{
		Employee: "Mai Ngọc Trung",
		Reason:   "dòng thiếu mã CCCD và không có nhân viên tên \"Mai Ngọc Trung\" trong dự án",
	}, unknown)

	// Second row for the same name: deduped.
	assert.Nil(t, weeklyBCCMissingAssignmentError("", "Mai Ngọc Trung", false, blocked, reported))

	// A different name-only employee must still be reported.
	ambiguous := weeklyBCCMissingAssignmentError("", "Nguyễn Văn An", true, blocked, reported)
	assert.Equal(t, &domain.ImportError{
		Employee: "Nguyễn Văn An",
		Reason:   "dòng thiếu mã CCCD và có nhiều nhân viên tên \"Nguyễn Văn An\" trong dự án",
	}, ambiguous)

	// Row with neither identifier.
	blank := weeklyBCCMissingAssignmentError("", "   ", false, blocked, reported)
	assert.Equal(t, &domain.ImportError{
		Employee: "   ",
		Reason:   "dòng thiếu cả mã CCCD và tên nhân viên",
	}, blank)
}

func TestBCCNameIndex_Lookup(t *testing.T) {
	assignments := []*domain.ProjectEmployee{
		{EmployeeID: 1, EmployeeCCCD: "031207007051", EmployeeName: "Mai Ngọc Trung"},
		{EmployeeID: 2, EmployeeCCCD: "031207009984", EmployeeName: "  nguyễn đức phúc  "},
		{EmployeeID: 3, EmployeeCCCD: "000000000001", EmployeeName: "Nguyễn Văn An"},
		{EmployeeID: 4, EmployeeCCCD: "000000000002", EmployeeName: "Nguyễn Văn An"},
		{EmployeeID: 5, EmployeeCCCD: "000000000003", EmployeeName: ""},
	}
	idx := newBCCNameIndex(assignments)

	// NFD input from macOS Excel must match the NFC name stored by the app.
	nfd := norm.NFD.String("Mai Ngọc Trung")
	got, ambiguous := idx.lookup(nfd)
	assert.False(t, ambiguous)
	if assert.NotNil(t, got) {
		assert.Equal(t, uint(1), got.EmployeeID)
	}

	// Case/whitespace-insensitive.
	got, ambiguous = idx.lookup("NGUYỄN ĐỨC PHÚC")
	assert.False(t, ambiguous)
	if assert.NotNil(t, got) {
		assert.Equal(t, uint(2), got.EmployeeID)
	}

	// Two employees share the name: never guess.
	got, ambiguous = idx.lookup("Nguyễn Văn An")
	assert.Nil(t, got)
	assert.True(t, ambiguous)

	// Unknown and blank names resolve to nothing without flagging ambiguity.
	got, ambiguous = idx.lookup("Trần Văn Lạ")
	assert.Nil(t, got)
	assert.False(t, ambiguous)
	got, ambiguous = idx.lookup("")
	assert.Nil(t, got)
	assert.False(t, ambiguous)
}

func weeklyBCCAssignmentFixtures() (byCCCD map[string]*domain.ProjectEmployee, assignments []*domain.ProjectEmployee, empNames map[uint]string) {
	assignments = []*domain.ProjectEmployee{
		{ID: 11, EmployeeID: 101, EmployeeCCCD: "031207009984", EmployeeName: "Nguyễn Đức Phúc", Position: "phổ thông"},
		{ID: 12, EmployeeID: 102, EmployeeCCCD: "031207007051", EmployeeName: "Mai Ngọc Trung", Position: "phổ thông"},
	}
	byCCCD = make(map[string]*domain.ProjectEmployee, len(assignments))
	empNames = make(map[uint]string, len(assignments))
	for _, a := range assignments {
		byCCCD[a.EmployeeCCCD] = a
		empNames[a.EmployeeID] = a.EmployeeName
	}
	return byCCCD, assignments, empNames
}

func weeklyBCCFlatRates() map[string]int {
	return map[string]int{"phổ thông.ngày thường.HC": 50000}
}

// Regression: a partner row whose "Mã nhân viên" (CCCD) cell is blank must
// resolve through the employee name when exactly one active assignment in the
// project carries it — the app used to report "Không tìm thấy nhân viên" for an
// employee who plainly exists in the project.
func TestBuildWeeklyBCCEntries_ResolvesNameOnlyRow(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{EmployeeCode: "031207009984", FullName: "Nguyễn Đức Phúc", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 1), Hours: 8}}},
			{FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyBCCEntryData{
				{Date: dateOf(2026, time.October, 6), Hours: 2},
				{Date: dateOf(2026, time.October, 7), Hours: 2},
			}},
		},
	}}}

	entries, _, totalRows, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, errs)
	assert.Equal(t, 2, totalRows)
	if assert.Len(t, entries, 3) {
		assert.Equal(t, uint(102), entries[1].EmployeeID)
		assert.Equal(t, "2026-10-06", entries[1].Date)
		assert.Equal(t, float64(2), entries[1].HoursWorked)
		assert.Equal(t, "HC", entries[1].HourType)
		assert.Equal(t, uint(102), entries[2].EmployeeID)
		assert.Equal(t, "2026-10-07", entries[2].Date)
	}
}

// A name-only row that matches two employees must not be guessed at, and each
// such employee is reported once even across sheets.
func TestBuildWeeklyBCCEntries_NameOnlyRowAmbiguous(t *testing.T) {
	shared := &domain.ProjectEmployee{EmployeeID: 201, EmployeeCCCD: "000000000001", EmployeeName: "Nguyễn Văn An", Position: "phổ thông"}
	other := &domain.ProjectEmployee{EmployeeID: 202, EmployeeCCCD: "000000000002", EmployeeName: "Nguyễn Văn An", Position: "phổ thông"}
	byCCCD := map[string]*domain.ProjectEmployee{"000000000001": shared, "000000000002": other}
	assignments := []*domain.ProjectEmployee{shared, other}
	empNames := map[uint]string{201: "Nguyễn Văn An", 202: "Nguyễn Văn An"}

	row := excelparser.WeeklyBCCEmployeeData{FullName: "Nguyễn Văn An", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}}
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{
		{ShiftType: "HC", Employees: []excelparser.WeeklyBCCEmployeeData{row}},
		{ShiftType: "OT150", Employees: []excelparser.WeeklyBCCEmployeeData{row}},
	}}

	entries, _, totalRows, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	assert.Equal(t, 2, totalRows)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, &domain.ImportError{
			Employee: "Nguyễn Văn An",
			Reason:   "dòng thiếu mã CCCD và có nhiều nhân viên tên \"Nguyễn Văn An\" trong dự án",
		}, &errs[0])
	}
}

// A name-only row that matches nobody keeps failing, but the reason names the
// blank identifier cell the partner has to fill in.
func TestBuildWeeklyBCCEntries_NameOnlyRowNotFound(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Trần Văn Lạ", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, &domain.ImportError{
			Employee: "Trần Văn Lạ",
			Reason:   "dòng thiếu mã CCCD và không có nhân viên tên \"Trần Văn Lạ\" trong dự án",
		}, &errs[0])
	}
}

// Name resolution stays guarded by the STK cross-check: when the resolved
// employee's CCCD maps to a different person in the STK sheet, the row fails
// instead of posting hours against the wrong profile.
func TestBuildWeeklyBCCEntries_NameOnlyRowCrossCheckedAgainstSTK(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}
	stkNameByCCCD := map[string]string{"031207007051": "Nguyễn Văn Khác"}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, stkNameByCCCD, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, "Mai Ngọc Trung", errs[0].Employee)
		assert.Contains(t, errs[0].Reason, "tên BCC (Mai Ngọc Trung) và tên STK (Nguyễn Văn Khác) khác nhau cho cùng CCCD 031207007051")
	}
}

// A name-resolved row must not resurrect an employee whose STK processing
// already failed (the blocked set is keyed by CCCD).
func TestBuildWeeklyBCCEntries_NameOnlyRowSkipsBlockedEmployee(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}
	blocked := map[string]struct{}{"031207007051": {}}

	entries, _, totalRows, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, blocked,
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	assert.Empty(t, errs)
	assert.Equal(t, 1, totalRows)
}

func TestBuildWeeklyPaymentEntries_ResolvesNameOnlyRow(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	byCCCD["031207007051"].Position = "Lương 520"
	parsed := &excelparser.WeeklyPaymentImportData{Sheets: []excelparser.WeeklyPaymentSheetData{{
		Position: "Lương 520",
		Employees: []excelparser.WeeklyPaymentEmployeeData{
			{FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyPaymentEntryData{{Day: 6, ShiftKey: "HC", Hours: 2}}},
		},
	}}}
	flatRates := map[string]int{"Lương 520.ngày thường.HC": 65000}

	entries, _, totalRows, errs := (&BCCImportService{}).buildWeeklyPaymentEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return flatRates }, 7, false)

	assert.Empty(t, errs)
	assert.Equal(t, 1, totalRows)
	if assert.Len(t, entries, 1) {
		assert.Equal(t, uint(102), entries[0].EmployeeID)
		assert.Equal(t, "2026-10-06", entries[0].Date)
		assert.Equal(t, "HC", entries[0].HourType)
	}
}

func dateOf(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// A project assignment without a CCCD must never become the catch-all for
// every blank-code row (the pre-fix lookup ran byCCCD[""]).
func TestBuildWeeklyBCCEntries_BlankCodeNeverMatchesCCCDLessAssignment(t *testing.T) {
	cccdLess := &domain.ProjectEmployee{ID: 21, EmployeeID: 301, EmployeeName: "Nguyễn Không Có CCCD", Position: "phổ thông"}
	catchAll := &domain.ProjectEmployee{ID: 22, EmployeeID: 302, EmployeeCCCD: "031207007051", EmployeeName: "Mai Ngọc Trung", Position: "phổ thông"}
	byCCCD := map[string]*domain.ProjectEmployee{"": cccdLess, "031207007051": catchAll}
	assignments := []*domain.ProjectEmployee{cccdLess, catchAll}
	empNames := map[uint]string{301: cccdLess.EmployeeName, 302: catchAll.EmployeeName}

	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Trần Văn Lạ", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	assert.Len(t, errs, 1)
}

// A row that carries a CCCD which matches nobody stays an error even when the
// name matches an assignment: a wrong identifier must be corrected in the file,
// not silently re-pointed at another employee.
func TestBuildWeeklyBCCEntries_UnknownCodeDoesNotFallBackToName(t *testing.T) {
	byCCCD, assignments, empNames := weeklyBCCAssignmentFixtures()
	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{EmployeeCode: "999999999999", FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, nil), empNames, nil, map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, "không tìm thấy nhân viên với CCCD \"999999999999\" trong dự án", errs[0].Reason)
	}
}

func TestWeeklySTKCCCDLookup_UniqueNamesOnly(t *testing.T) {
	rows := []excelparser.STKRow{
		{CCCD: "000000000001", FullName: "Mai Ngọc Trung"},
		{CCCD: "000000000002", FullName: "Nguyễn Văn An"},
		{CCCD: "000000000003", FullName: "Nguyễn Văn An"}, // duplicated name
		{CCCD: "", FullName: "Thiếu CCCD"},
		{CCCD: "000000000004", FullName: ""},
	}
	lookup := weeklySTKCCCDLookup(rows)

	assert.Equal(t, "000000000001", lookup["mai ngọc trung"])
	assert.Equal(t, "000000000001", lookup[bccNormName(norm.NFD.String("Mai Ngọc Trung"))])
	assert.NotContains(t, lookup, "nguyễn văn an") // duplicates must not resolve
	assert.Len(t, lookup, 1)
}

// The row's identifier comes from the workbook before its name: an assignment
// whose stored snapshot name drifted from the sheet (typo, accent, rename)
// still resolves through the CCCD the STK sheet carries.
func TestBuildWeeklyBCCEntries_ResolvesBlankCodeFromSTKSheet(t *testing.T) {
	// Snapshot name on the assignment carries no diacritics, so the name index
	// cannot match the sheet's "Mai Ngọc Trung".
	assignment := &domain.ProjectEmployee{ID: 12, EmployeeID: 102, EmployeeCCCD: "031207007051", EmployeeName: "Mai Ngoc Trung", Position: "phổ thông"}
	byCCCD := map[string]*domain.ProjectEmployee{"031207007051": assignment}
	assignments := []*domain.ProjectEmployee{assignment}
	empNames := map[uint]string{102: assignment.EmployeeName}
	stkRows := []excelparser.STKRow{{CCCD: "031207007051", FullName: "Mai Ngọc Trung"}}

	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Mai Ngọc Trung", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, stkRows), empNames,
		weeklySTKNameLookup(stkRows), map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, errs)
	if assert.Len(t, entries, 1) {
		assert.Equal(t, uint(102), entries[0].EmployeeID)
	}
}

// A name listed twice in the STK sheet identifies nobody, so it must not be
// used to pick an employee — with two same-name assignments in the project the
// row stays an error instead of posting hours against the wrong person.
func TestBuildWeeklyBCCEntries_DuplicateNameInSTKDoesNotResolve(t *testing.T) {
	first := &domain.ProjectEmployee{ID: 31, EmployeeID: 401, EmployeeCCCD: "000000000001", EmployeeName: "Nguyễn Văn An", Position: "phổ thông"}
	second := &domain.ProjectEmployee{ID: 32, EmployeeID: 402, EmployeeCCCD: "000000000002", EmployeeName: "Nguyễn Văn An", Position: "phổ thông"}
	byCCCD := map[string]*domain.ProjectEmployee{"000000000001": first, "000000000002": second}
	assignments := []*domain.ProjectEmployee{first, second}
	empNames := map[uint]string{401: first.EmployeeName, 402: second.EmployeeName}
	stkRows := []excelparser.STKRow{
		{CCCD: "000000000001", FullName: "Nguyễn Văn An"},
		{CCCD: "000000000002", FullName: "Nguyễn Văn An"},
	}

	parsed := &excelparser.WeeklyBCCImportData{Sheets: []excelparser.WeeklyBCCSheetData{{
		ShiftType: "HC",
		Employees: []excelparser.WeeklyBCCEmployeeData{
			{FullName: "Nguyễn Văn An", Entries: []excelparser.WeeklyBCCEntryData{{Date: dateOf(2026, time.October, 6), Hours: 2}}},
		},
	}}}

	entries, _, _, errs := (&BCCImportService{}).buildWeeklyBCCEntries(
		context.Background(), parsed, 2026, time.October, time.UTC,
		newWeeklyRowResolver(byCCCD, assignments, stkRows), empNames,
		weeklySTKNameLookup(stkRows), map[string]struct{}{},
		func(time.Time) map[string]int { return weeklyBCCFlatRates() }, 7, false)

	assert.Empty(t, entries)
	if assert.Len(t, errs, 1) {
		assert.Equal(t, "dòng thiếu mã CCCD và có nhiều nhân viên tên \"Nguyễn Văn An\" trong dự án", errs[0].Reason)
	}
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
