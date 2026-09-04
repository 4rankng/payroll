package services

import (
	"testing"
	"time"

	excelparser "api-server/internal/app/services/excel"
)

func TestLabelRateTarget(t *testing.T) {
	flatRates := map[string]int{
		"phổ thông.ngày thường.NT":    66666,
		"phổ thông.ngày nghỉ.NT":      100000,
		"kinh nghiệm.ngày thường.NT":  70000,
		"phổ thông.ngày nghỉ.T7":      100000,
		"phổ thông.ngày nghỉ.OT":      0, // zero-rate buckets must not resolve
		"phổ thông.ngày thường.OT T7": 100000,
	}
	got, ok := labelRateTarget(flatRates, "nt")
	if !ok || got.dayType != "ngày thường" || got.hourType != "NT" {
		t.Fatalf("labelRateTarget(nt) = %+v ok=%v, want ngày thường/NT", got, ok)
	}
	got, ok = labelRateTarget(flatRates, "T7")
	if !ok || got.dayType != "ngày nghỉ" || got.hourType != "T7" {
		t.Fatalf("labelRateTarget(T7) = %+v ok=%v, want ngày nghỉ/T7", got, ok)
	}
	got, ok = labelRateTarget(flatRates, "OT T7")
	if !ok || got.hourType != "OT T7" {
		t.Fatalf("labelRateTarget(OT T7) = %+v ok=%v", got, ok)
	}
	if _, ok := labelRateTarget(flatRates, "ZZ"); ok {
		t.Fatal("unknown label must not resolve")
	}
	if _, ok := labelRateTarget(flatRates, "OT"); ok {
		t.Fatal("zero-rate bucket must not resolve")
	}
}

func TestLabelRateTargetDeterministicUnderTies(t *testing.T) {
	// Equal day-type priority across positions: the winner must not depend
	// on map iteration order.
	flatRates := map[string]int{
		"đội a.ngày thường.NT": 100000,
		"đội b.ngày thường.NT": 120000,
		"đội b.thường.NT":      90000,
	}
	want := rateTarget{dayType: "ngày thường", hourType: "NT"}
	for i := 0; i < 50; i++ {
		got, ok := labelRateTarget(flatRates, "NT")
		if !ok || got != want {
			t.Fatalf("iteration %d: labelRateTarget(NT) = %+v ok=%v, want %+v", i, got, ok, want)
		}
	}
}

func TestResolveBCCEntryTarget(t *testing.T) {
	wednesday := time.Date(2026, 8, 5, 0, 0, 0, 0, time.Local)

	// 1. The file's own rate row wins first.
	shiftRates := map[string]int64{"HC": 250000}
	rateToTarget := map[int]rateTarget{250000: {dayType: "ngày thường", hourType: "ca ngày"}}
	got, ok := resolveBCCEntryTarget(shiftRates, rateToTarget, nil, "HC", wednesday, false, false)
	if !ok || got != (rateTarget{"ngày thường", "ca ngày"}) {
		t.Fatalf("rate path = %+v ok=%v", got, ok)
	}

	// 2. Date-row label-keyed resolution: CN names a payrate leaf (Chủ Nhật).
	dateRowRates := map[string]int{"phổ thông.ngày nghỉ.CN": 500000}
	got, ok = resolveBCCEntryTarget(nil, nil, dateRowRates, "CN", wednesday, true, true)
	if !ok || got.dayType != "ngày nghỉ" || got.hourType != "CN" {
		t.Fatalf("label-keyed CN = %+v ok=%v, want ngày nghỉ/CN", got, ok)
	}

	// 3. Legacy rateless files keep the calendar path: their CN means
	// "ca ngày", so a Wednesday CN entry resolves via the calendar — the
	// date-row CN leaf must NOT capture it.
	legacyRates := map[string]int{"phổ thông.ngày nghỉ.CN": 500000, "phổ thông.ngày thường.ca ngày": 200000}
	got, ok = resolveBCCEntryTarget(nil, nil, legacyRates, "CN", wednesday, true, false)
	if !ok || got.dayType != "ngày thường" || got.hourType != "ca ngày" {
		t.Fatalf("legacy CN on Wednesday = %+v ok=%v, want ngày thường/ca ngày", got, ok)
	}

	// 4. Unknown label on a date-row file: no resolution, caller surfaces the
	// missing-rate error (without a VND figure).
	if _, ok := resolveBCCEntryTarget(nil, nil, dateRowRates, "X", wednesday, true, true); ok {
		t.Fatal("unknown label must not resolve")
	}

	// 5. Rate-carrying files never fall back to the calendar path — an
	// unknown rate must fail loudly.
	if _, ok := resolveBCCEntryTarget(shiftRates, rateToTarget, legacyRates, "ZZ", wednesday, false, false); ok {
		t.Fatal("unknown rate on a rate file must not resolve")
	}
}

func TestEarliestInMonthDay(t *testing.T) {
	jul := func(day int) *time.Time { d := time.Date(2026, 7, day, 0, 0, 0, 0, time.Local); return &d }
	aug := func(day int) *time.Time { d := time.Date(2026, 8, day, 0, 0, 0, 0, time.Local); return &d }
	sep := func(day int) *time.Time { d := time.Date(2026, 9, day, 0, 0, 0, 0, time.Local); return &d }

	straddling := []excelparser.BCCEmployeeData{{
		Entries: []excelparser.BCCEntryData{
			{DayNum: 26, FullDate: jul(26)}, {DayNum: 31, FullDate: jul(31)},
			{DayNum: 1, FullDate: aug(1)}, {DayNum: 25, FullDate: aug(25)},
		},
	}}
	if got := earliestInMonthDay(straddling, 2026, time.August); got != 1 {
		t.Errorf("straddling July/August for August = %d, want 1 (first in-month day)", got)
	}
	sepFile := []excelparser.BCCEmployeeData{{
		Entries: []excelparser.BCCEntryData{
			{DayNum: 21, FullDate: aug(21)}, {DayNum: 1, FullDate: sep(1)}, {DayNum: 15, FullDate: sep(15)},
		},
	}}
	if got := earliestInMonthDay(sepFile, 2026, time.September); got != 1 {
		t.Errorf("September file = %d, want 1", got)
	}
	outOfMonthOnly := []excelparser.BCCEmployeeData{{
		Entries: []excelparser.BCCEntryData{{DayNum: 26, FullDate: jul(26)}},
	}}
	if got := earliestInMonthDay(outOfMonthOnly, 2026, time.August); got != 0 {
		t.Errorf("out-of-month-only = %d, want 0", got)
	}
	// Legacy files (no FullDate) keep day numbers as-is.
	legacy := []excelparser.BCCEmployeeData{{
		Entries: []excelparser.BCCEntryData{{DayNum: 3}, {DayNum: 17}},
	}}
	if got := earliestInMonthDay(legacy, 2026, time.August); got != 3 {
		t.Errorf("legacy = %d, want 3", got)
	}
}

func TestMergeSTKRows(t *testing.T) {
	base := []excelparser.STKRow{
		{CCCD: "A", FullName: "Ann", BankAccount: "old", BankName: "VCB"},
		{CCCD: "B", FullName: "Bee", BankAccount: "keep", BankName: "TCB"},
	}
	stk := []excelparser.STKRow{
		{CCCD: "A", FullName: "Ann", BankAccount: "new", BankName: "MB", Mobile: "0900111222"},
		{CCCD: "C", FullName: "Cee", BankAccount: "only", BankName: "ACB"},
	}
	merged := mergeSTKRows(base, stk)

	if len(merged) != 3 {
		t.Fatalf("merged rows = %d, want 3 (STK-only hire appended)", len(merged))
	}
	if merged[0].BankAccount != "new" || merged[0].BankName != "MB" || merged[0].Mobile != "0900111222" {
		t.Errorf("STK must win bank/mobile for the same CCCD: %+v", merged[0])
	}
	if merged[1].BankAccount != "keep" {
		t.Errorf("BCC-only row must keep its bank: %+v", merged[1])
	}
	if merged[2].CCCD != "C" || merged[2].BankAccount != "only" {
		t.Errorf("STK-only row appended: %+v", merged[2])
	}
}
