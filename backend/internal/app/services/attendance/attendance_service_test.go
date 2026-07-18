package attendance

import (
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestResolveShiftWindowsUsesCheckoutRulesForCrossMidnightShift(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 6, 21, 20, 10, 0, 0, loc)
	flattened := map[string]int{"Công nhân.ngày thường.20:00-04:00": 300000}

	start, end, checkInStart, checkInEnd, checkOutStart, checkOutEnd, ok := ResolveShiftWindows(flattened, "Công nhân", now)
	if !ok {
		t.Fatal("expected resolved cross-midnight shift")
	}
	if got, want := start.Format("15:04"), "20:00"; got != want {
		t.Fatalf("shift start = %s, want %s", got, want)
	}
	if got, want := end.Format("15:04"), "04:00"; got != want {
		t.Fatalf("shift end = %s, want %s", got, want)
	}
	if got, want := checkInStart.Format("15:04"), "19:00"; got != want {
		t.Fatalf("check-in window start = %s, want %s", got, want)
	}
	if got, want := checkInEnd.Format("15:04"), "21:00"; got != want {
		t.Fatalf("check-in window end = %s, want %s", got, want)
	}
	if got, want := checkOutStart.Format("15:04"), "03:00"; got != want {
		t.Fatalf("checkout window start = %s, want %s", got, want)
	}
	if got, want := checkOutEnd.Format("15:04"), "08:00"; got != want {
		t.Fatalf("checkout window end = %s, want %s", got, want)
	}
}

func TestResolveAllShiftWindowsReturnsEveryConfiguredShiftInOrder(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, loc)
	flattened := map[string]int{
		"Công nhân.ngày thường.13:00-22:00": 300000,
		"Công nhân.ngày thường.07:00-12:00": 250000,
	}

	windows := ResolveAllShiftWindows(flattened, "Công nhân", now, nil)
	if len(windows) != 2 {
		t.Fatalf("window count = %d, want 2", len(windows))
	}
	if got, want := windows[0].ShiftStart.Format("15:04"), "07:00"; got != want {
		t.Fatalf("first shift = %s, want %s", got, want)
	}
	if got, want := windows[1].ShiftStart.Format("15:04"), "13:00"; got != want {
		t.Fatalf("second shift = %s, want %s", got, want)
	}
	if got, want := windows[1].CheckOutWindowEnd.Format("15:04"), "02:00"; got != want {
		t.Fatalf("second checkout deadline = %s, want %s", got, want)
	}
}

func TestResolveAllShiftWindowsAttachesAdminNameByRange(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	now := time.Date(2026, 6, 21, 12, 0, 0, 0, loc)
	flattened := map[string]int{
		"Công nhân.ngày thường.09:00-18:00": 300000,
		"Công nhân.ngày thường.21:00-05:00": 350000,
	}
	names := map[string]string{
		"09:00-18:00": "Ca làm",
		"21:00-05:00": "Ca đêm",
	}

	windows := ResolveAllShiftWindows(flattened, "Công nhân", now, names)
	if len(windows) != 2 {
		t.Fatalf("window count = %d, want 2", len(windows))
	}
	byRange := map[string]string{}
	for _, w := range windows {
		byRange[w.ShiftStart.Format("15:04")+"-"+w.ShiftEnd.Format("15:04")] = w.Name
	}
	if got, want := byRange["09:00-18:00"], "Ca làm"; got != want {
		t.Fatalf("day shift name = %q, want %q", got, want)
	}
	if got, want := byRange["21:00-05:00"], "Ca đêm"; got != want {
		t.Fatalf("night shift name = %q, want %q", got, want)
	}

	// nil names -> empty Name on every window (fallback path)
	for _, w := range ResolveAllShiftWindows(flattened, "Công nhân", now, nil) {
		if w.Name != "" {
			t.Fatalf("expected empty name for nil map, got %q", w.Name)
		}
	}
}

func TestExtractShiftRanges(t *testing.T) {
	flattened := map[string]int{
		"Công nhân.ngày thường.09:00-18:00": 300000,
		"Lái xe.ngày thường.08:00-17:00":    250000,
		// duplicate range across positions should be deduped
		"Quản lý.ngày thường.09:00-18:00": 400000,
		// non-shift keys (no HH:MM-HH:MM suffix) must be ignored
		"something.else": 100,
	}
	ranges := ExtractShiftRanges(flattened)
	want := map[string]bool{"09:00-18:00": true, "08:00-17:00": true}
	if len(ranges) != len(want) {
		t.Fatalf("range count = %d (%v), want %d", len(ranges), ranges, len(want))
	}
	for _, r := range ranges {
		if !want[r] {
			t.Fatalf("unexpected range %q", r)
		}
	}
}

func TestResolveShiftWindowsAnchorsActiveOvernightAttendanceToCheckIn(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkInTime := time.Date(2026, 6, 21, 20, 10, 0, 0, loc)
	flattened := map[string]int{"Công nhân.ngày thường.20:00-04:00": 300000}

	_, shiftEnd, _, _, checkOutStart, checkOutEnd, ok := ResolveShiftWindows(flattened, "Công nhân", checkInTime)
	if !ok {
		t.Fatal("expected active overnight attendance shift")
	}
	if got, want := shiftEnd.Format("2006-01-02 15:04"), "2026-06-22 04:00"; got != want {
		t.Fatalf("shift end = %s, want %s", got, want)
	}
	if got, want := checkOutStart.Format("2006-01-02 15:04"), "2026-06-22 03:00"; got != want {
		t.Fatalf("checkout start = %s, want %s", got, want)
	}
	if got, want := checkOutEnd.Format("2006-01-02 15:04"), "2026-06-22 08:00"; got != want {
		t.Fatalf("checkout end = %s, want %s", got, want)
	}
}

func TestValidateCheckOutWindowRejectsBeforeCheckoutWindow(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)  // shift end K for 20:00-04:00
	checkOut := time.Date(2026, 6, 21, 23, 0, 0, 0, loc) // before K-1h

	err := validateCheckOutWindow(&parsedShift{end: shiftEnd}, checkIn, checkOut)

	if err == nil {
		t.Fatal("expected checkout before K-1h to be rejected")
	}
	domainErr, ok := err.(*domain.DomainError)
	if !ok || domainErr.Type != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error, got %T", err)
	}
	expected := "Bạn mới vào làm lúc 20:09. Chỉ có thể tan ca từ 03:00 đến 08:00."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateCheckOutWindowAcceptsInclusiveBounds(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc) // K
	shift := &parsedShift{end: shiftEnd}

	// At K-1h, K, within the window, and K+4h are allowed.
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(-checkOutLowerGrace)); err != nil {
		t.Fatalf("expected checkout exactly at K-1h to be allowed, got %v", err)
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd); err != nil {
		t.Fatalf("expected checkout exactly at K to be allowed, got %v", err)
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(2*time.Hour+30*time.Minute)); err != nil {
		t.Fatalf("expected checkout within [K-1h, K+4h] to be allowed, got %v", err)
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(checkOutUpperGrace)); err != nil {
		t.Fatalf("expected checkout exactly at K+4h to be allowed, got %v", err)
	}

	// Before K-1h and after K+4h are rejected.
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(-checkOutLowerGrace).Add(-time.Nanosecond)); err == nil {
		t.Fatal("expected checkout before K-1h to be rejected")
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(checkOutUpperGrace).Add(30*time.Minute)); err == nil {
		t.Fatal("expected checkout after K+4h to be rejected")
	}
}

func TestValidateCheckOutWindowAcceptsNightShiftCheckoutWithinFourHours(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 22, 6, 30, 0, 0, loc)

	if err := validateCheckOutWindow(&parsedShift{end: shiftEnd}, checkIn, checkOut); err != nil {
		t.Fatalf("expected 06:30 checkout for K=04:00 to be allowed within [K-1h, K+4h], got %v", err)
	}
}

func TestValidateCheckInWindowAcceptsWithinShift(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	shift := &parsedShift{start: time.Date(2026, 6, 21, 8, 0, 0, 0, loc)} // T

	for _, ci := range []time.Time{
		time.Date(2026, 6, 21, 7, 30, 0, 0, loc),
		time.Date(2026, 6, 21, 8, 0, 0, 0, loc),
		time.Date(2026, 6, 21, 8, 30, 0, 0, loc),
	} {
		if err := validateCheckInWindow(shift, ci); err != nil {
			t.Fatalf("expected check-in %v within (T-1h, T+1h) to be allowed, got %v", ci, err)
		}
	}
}

func TestValidateCheckInWindowRejectsOutsideShift(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	shift := &parsedShift{start: time.Date(2026, 6, 21, 8, 0, 0, 0, loc)} // T -> window (07:00, 09:00)

	for _, ci := range []time.Time{
		time.Date(2026, 6, 21, 7, 0, 0, 0, loc),  // exactly T-1h (exclusive)
		time.Date(2026, 6, 21, 6, 59, 0, 0, loc), // before window
		time.Date(2026, 6, 21, 9, 0, 0, 0, loc),  // exactly T+1h (exclusive)
		time.Date(2026, 6, 21, 9, 30, 0, 0, loc), // after window
	} {
		err := validateCheckInWindow(shift, ci)
		if err == nil {
			t.Fatalf("expected check-in %v outside (T-1h, T+1h) to be rejected", ci)
		}
		domainErr, ok := err.(*domain.DomainError)
		if !ok {
			t.Fatalf("expected DomainError, got %T", err)
		}
		if domainErr.Code != attendanceCheckInWindowCode {
			t.Fatalf("expected code %q, got %q", attendanceCheckInWindowCode, domainErr.Code)
		}
		if domainErr.Context["guidance_type"] != "timing" || domainErr.Context["action"] != "check_in" ||
			domainErr.Context["window_start"] != "07:00" || domainErr.Context["window_end"] != "09:00" {
			t.Fatalf("unexpected timing guidance: %#v", domainErr.Context)
		}
	}
}

func TestValidateCheckInWindowAcceptsNightShiftEarlyArrival(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	shift := &parsedShift{start: time.Date(2026, 6, 21, 20, 0, 0, 0, loc)} // night shift T -> window (19:00, 21:00)
	checkIn := time.Date(2026, 6, 21, 19, 50, 0, 0, loc)

	if err := validateCheckInWindow(shift, checkIn); err != nil {
		t.Fatalf("expected early-arrival night-shift check-in to be allowed, got %v", err)
	}
}

func TestResolveShiftUsesConfiguredNightShiftEnd(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)

	shift := service.resolveShift(payrate, "Công nhân", checkIn)

	if shift == nil {
		t.Fatal("expected a resolved shift, got nil")
	}
	expected := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	if !shift.end.Equal(expected) {
		t.Fatalf("expected shift end %v, got %v", expected, shift.end)
	}
}

func TestResolveShiftUsesConfiguredDayShiftEnd(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 7, 55, 0, 0, loc)

	shift := service.resolveShift(payrate, "Công nhân", checkIn)

	if shift == nil {
		t.Fatal("expected a resolved shift, got nil")
	}
	expected := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)
	if !shift.end.Equal(expected) {
		t.Fatalf("expected shift end %v, got %v", expected, shift.end)
	}
}

func TestResolveShiftReturnsNilWhenUnconfigured(t *testing.T) {
	service := &AttendanceService{}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)

	// No payrate configured at all -> no resolvable shift.
	if shift := service.resolveShift(nil, "Công nhân", checkIn); shift != nil {
		t.Fatalf("nil payrate: expected nil shift, got %+v", shift)
	}

	// Payrate exists but the requested position is absent (two configured positions,
	// so no single-position fallback) -> no resolvable shift.
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Bảo vệ":{"ngày thường":{"20:00-04:00":300000}},"Lễ tân":{"ngày thường":{"08:00-17:00":250000}}}`),
	}
	if shift := service.resolveShift(payrate, "Công nhân", checkIn); shift != nil {
		t.Fatalf("missing position: expected nil shift, got %+v", shift)
	}
}

// TestResolveShiftUsesSingleConfiguredPositionAsFallback guards the
// single-position fallback: with exactly one configured position, an
// unconfigured assignment position still resolves a shift.
func TestResolveShiftUsesSingleConfiguredPositionAsFallback(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 7, 55, 0, 0, loc)

	shift := service.resolveShift(payrate, "Lái xe", checkIn)

	if shift == nil {
		t.Fatal("expected single-position fallback to resolve a shift, got nil")
	}
	expected := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)
	if !shift.end.Equal(expected) {
		t.Fatalf("expected shift end %v, got %v", expected, shift.end)
	}
}

// TestResolveShiftAnchorsNightShiftEarlyArrivalToCorrectDay guards the ±1-day
// candidate fix: checking in 10m early for a 20:00-04:00 night shift must anchor
// to TONIGHT's start (2026-06-21 20:00) and end TOMORROW (2026-06-22 04:00). The
// old rollback hack mis-anchored this to yesterday, which would reject a valid
// checkout once the K+4h upper bound is enforced.
func TestResolveShiftAnchorsNightShiftEarlyArrivalToCorrectDay(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 19, 50, 0, 0, loc)

	shift := service.resolveShift(payrate, "Công nhân", checkIn)

	if shift == nil {
		t.Fatal("expected a resolved shift, got nil")
	}
	expectedStart := time.Date(2026, 6, 21, 20, 0, 0, 0, loc)
	expectedEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	if !shift.start.Equal(expectedStart) {
		t.Fatalf("expected shift start %v, got %v", expectedStart, shift.start)
	}
	if !shift.end.Equal(expectedEnd) {
		t.Fatalf("expected shift end %v, got %v", expectedEnd, shift.end)
	}
}

// TestResolveShiftAnchorsPostMidnightCheckIn covers the post-midnight case the
// old rollback hack was written for: checking in at 00:30 for a 20:00-04:00
// night shift must anchor to the previous evening's start and today's end.
func TestResolveShiftAnchorsPostMidnightCheckIn(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 22, 0, 30, 0, 0, loc) // post-midnight

	shift := service.resolveShift(payrate, "Công nhân", checkIn)

	if shift == nil {
		t.Fatal("expected a resolved shift, got nil")
	}
	expectedStart := time.Date(2026, 6, 21, 20, 0, 0, 0, loc)
	expectedEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	if !shift.start.Equal(expectedStart) {
		t.Fatalf("expected shift start %v, got %v", expectedStart, shift.start)
	}
	if !shift.end.Equal(expectedEnd) {
		t.Fatalf("expected shift end %v, got %v", expectedEnd, shift.end)
	}
}

// TestResolveShiftPrefersContainingNightShiftOverShortNeighbor guards the
// "contains ci" preference in closestShift: with a 20:00-04:00 night shift AND a
// 04:00-05:00 morning shift, a 03:55 check-in (inside the night shift, 5m before
// the morning start) must anchor to the night shift (end 04:00), not the morning
// shift (end 05:00) — nearest-start alone would wrongly pick the morning shift.
func TestResolveShiftPrefersContainingNightShiftOverShortNeighbor(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":300000,"04:00-05:00":150000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 22, 3, 55, 0, 0, loc)

	shift := service.resolveShift(payrate, "Công nhân", checkIn)

	if shift == nil {
		t.Fatal("expected a resolved shift, got nil")
	}
	expectedStart := time.Date(2026, 6, 21, 20, 0, 0, 0, loc)
	expectedEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	if !shift.start.Equal(expectedStart) {
		t.Fatalf("expected night shift start %v, got %v (anchor stolen by neighbor?)", expectedStart, shift.start)
	}
	if !shift.end.Equal(expectedEnd) {
		t.Fatalf("expected night shift end %v, got %v (anchor stolen by neighbor?)", expectedEnd, shift.end)
	}
}

func TestCalculateEarningAmountRejectReasonForMissingShift(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 15, 30, 0, 0, loc) // before K-1h

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning amount, got %d", amount)
	}
	expected := "Thời gian vào 08:00 và tan 15:30 không hợp lệ. Bạn phải vào làm từ 07:00 đến 09:00 và tan ca từ 16:00 đến 21:00."
	if reason != expected {
		t.Fatalf("expected invalid shift reason %q, got %q", expected, reason)
	}
}

// TestCalculateEarningAmountRejectsCheckoutAfterWindow verifies earning and the
// checkout gate agree on the upper bound: a check-out after K+4h earns nothing
// (same predicate the gate rejects on).
func TestCalculateEarningAmountRejectsCheckoutAfterWindow(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 21, 30, 0, 0, loc) // 30m after K+4h (21:00)

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning for checkout after K+4h, got %d", amount)
	}
	if !strings.Contains(reason, "tan ca từ 16:00 đến 21:00") {
		t.Fatalf("expected reject reason to state the checkout window, got %q", reason)
	}
}

func TestCalculateEarningAmountRejectReasonForMissingPosition(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}},"Bảo vệ":{"ngày thường":{"08:00-17:00":280000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)

	amount, reason, err := service.calculateEarningAmount(payrate, "Tổ trưởng", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning amount, got %d", amount)
	}
	if !strings.Contains(reason, "Chưa có mức lương cho vị trí") {
		t.Fatalf("expected missing position reason, got %q", reason)
	}
}

func TestCalculateEarningAmountUsesOnlyConfiguredPositionAsFallback(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Phổ Thông":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)

	amount, reason, err := service.calculateEarningAmount(payrate, "Nhân viên sản xuất", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 300000 {
		t.Fatalf("expected earning amount 300000, got %d", amount)
	}
	if reason != "" {
		t.Fatalf("expected no reject reason, got %q", reason)
	}
}

func TestCalculateEarningAmountRecordsConfiguredShift(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 7, 55, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 300000 {
		t.Fatalf("expected earning amount 300000, got %d", amount)
	}
	if reason != "" {
		t.Fatalf("expected no reject reason, got %q", reason)
	}
}

// TestCalculateEarningAmountPaysFullShiftForLateCheckInWithinWindow verifies the
// window-based earning rule: a check-in up to 1h AFTER shift start (within the
// check-in window) still earns the full shift wage, as long as check-out is in
// [K-1h, K+4h].
func TestCalculateEarningAmountPaysFullShiftForLateCheckInWithinWindow(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 30, 0, 0, loc) // 30m late, within (07:00, 09:00)
	checkOut := time.Date(2026, 6, 21, 17, 10, 0, 0, loc)

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 300000 {
		t.Fatalf("expected full earning amount 300000 for late check-in within window, got %d", amount)
	}
	if reason != "" {
		t.Fatalf("expected no reject reason, got %q", reason)
	}
}
