package attendance

import (
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestValidateCheckOutWindowRejectsBeforeShiftEnd(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc) // shift end K for 20:00-04:00
	checkOut := time.Date(2026, 6, 21, 23, 0, 0, 0, loc) // before K

	err := validateCheckOutWindow(&parsedShift{end: shiftEnd}, checkIn, checkOut)

	if err == nil {
		t.Fatal("expected checkout before shift end to be rejected")
	}
	domainErr, ok := err.(*domain.DomainError)
	if !ok || domainErr.Type != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error, got %T", err)
	}
	expected := "Bạn mới vào làm lúc 20:09. Chỉ có thể tan ca từ 04:00 đến 05:00."
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateCheckOutWindowAcceptsAtEarliestRejectsAtUpperBound(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc) // K
	shift := &parsedShift{end: shiftEnd}

	// At K and within (K, K+1h) are allowed.
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd); err != nil {
		t.Fatalf("expected checkout exactly at K to be allowed, got %v", err)
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(30*time.Minute)); err != nil {
		t.Fatalf("expected checkout within [K, K+1h) to be allowed, got %v", err)
	}

	// At and beyond K+1h are rejected (upper bound is exclusive).
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(checkOutUpperGrace)); err == nil {
		t.Fatal("expected checkout exactly at K+1h to be rejected")
	}
	if err := validateCheckOutWindow(shift, checkIn, shiftEnd.Add(checkOutUpperGrace).Add(30*time.Minute)); err == nil {
		t.Fatal("expected checkout after K+1h to be rejected")
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
		if err := validateCheckInWindow(shift, ci); err == nil {
			t.Fatalf("expected check-in %v outside (T-1h, T+1h) to be rejected", ci)
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
// checkout once the K+1h upper bound is enforced.
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
	checkOut := time.Date(2026, 6, 21, 16, 30, 0, 0, loc) // before shift end K

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning amount, got %d", amount)
	}
	expected := "Thời gian vào 08:00 và tan 16:30 không hợp lệ. Bạn phải vào làm từ 07:00 đến 09:00 và tan ca từ 17:00 đến 18:00."
	if reason != expected {
		t.Fatalf("expected invalid shift reason %q, got %q", expected, reason)
	}
}

// TestCalculateEarningAmountRejectsCheckoutAfterWindow verifies earning and the
// checkout gate agree on the upper bound: a check-out after K+1h earns nothing
// (same predicate the gate rejects on).
func TestCalculateEarningAmountRejectsCheckoutAfterWindow(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 18, 30, 0, 0, loc) // 30m after K+1h (18:00)

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning for checkout after K+1h, got %d", amount)
	}
	if !strings.Contains(reason, "tan ca từ 17:00 đến 18:00") {
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
// [K, K+1h).
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
