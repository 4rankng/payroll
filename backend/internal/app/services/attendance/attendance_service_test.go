package attendance

import (
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestValidateEarliestCheckoutRejectsBeforeShiftEnd(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc) // night shift 20:00-04:00
	checkOut := time.Date(2026, 6, 21, 23, 0, 0, 0, loc)

	err := validateEarliestCheckout(checkIn, shiftEnd, checkOut)

	if err == nil {
		t.Fatal("expected checkout before shift end to be rejected")
	}
	domainErr, ok := err.(*domain.DomainError)
	if !ok || domainErr.Type != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error, got %T", err)
	}
	expected := "Bạn mới vào làm lúc 20:09. Chỉ có thể tan ca sau 04:00"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateEarliestCheckoutAllowsAtAndAfterShiftEnd(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	shiftEnd := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)

	if err := validateEarliestCheckout(checkIn, shiftEnd, shiftEnd); err != nil {
		t.Fatalf("expected checkout exactly at shift end to be allowed, got %v", err)
	}
	if err := validateEarliestCheckout(checkIn, shiftEnd, shiftEnd.Add(time.Hour)); err != nil {
		t.Fatalf("expected checkout after shift end to be allowed, got %v", err)
	}
}

func TestResolveEarliestCheckoutUsesConfiguredNightShiftEnd(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"20:00-04:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)

	earliest := service.resolveEarliestCheckout(payrate, "Công nhân", checkIn)

	expected := time.Date(2026, 6, 22, 4, 0, 0, 0, loc)
	if !earliest.Equal(expected) {
		t.Fatalf("expected earliest checkout %v, got %v", expected, earliest)
	}
}

func TestResolveEarliestCheckoutUsesConfiguredDayShiftEnd(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 7, 55, 0, 0, loc)

	earliest := service.resolveEarliestCheckout(payrate, "Công nhân", checkIn)

	expected := time.Date(2026, 6, 21, 17, 0, 0, 0, loc)
	if !earliest.Equal(expected) {
		t.Fatalf("expected earliest checkout %v, got %v", expected, earliest)
	}
}

func TestResolveEarliestCheckoutFallsBackToMinimumWhenUnconfigured(t *testing.T) {
	service := &AttendanceService{}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 20, 9, 0, 0, loc)
	fallback := checkIn.Add(4 * time.Hour)

	// No payrate configured at all.
	if got := service.resolveEarliestCheckout(nil, "Công nhân", checkIn); !got.Equal(fallback) {
		t.Fatalf("nil payrate: expected %v, got %v", fallback, got)
	}

	// Payrate exists but the requested position is absent (two configured positions,
	// so no single-position fallback) -> no resolvable shift -> minimum duration.
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Bảo vệ":{"ngày thường":{"20:00-04:00":300000}},"Lễ tân":{"ngày thường":{"08:00-17:00":250000}}}`),
	}
	if got := service.resolveEarliestCheckout(payrate, "Công nhân", checkIn); !got.Equal(fallback) {
		t.Fatalf("missing position: expected %v, got %v", fallback, got)
	}
}

func TestCalculateEarningAmountRejectReasonForMissingShift(t *testing.T) {
	service := &AttendanceService{}
	payrate := &domain.Payrate{
		Payrate: domain.PayrateConfiguration(`{"Công nhân":{"ngày thường":{"08:00-17:00":300000}}}`),
	}
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 8, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 16, 30, 0, 0, loc)

	amount, reason, err := service.calculateEarningAmount(payrate, "Công nhân", checkIn, checkOut)

	if err != nil {
		t.Fatalf("calculateEarningAmount returned error: %v", err)
	}
	if amount != 0 {
		t.Fatalf("expected no earning amount, got %d", amount)
	}
	expected := "Thời gian vào 08:00 và tan 16:30 không hợp lệ, bạn phải vào làm trước 08:00 và tan ca sau 17:00."
	if reason != expected {
		t.Fatalf("expected invalid shift reason %q, got %q", expected, reason)
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
