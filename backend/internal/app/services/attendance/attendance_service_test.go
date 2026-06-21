package attendance

import (
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestValidateMinimumCheckoutDurationRejectsBeforeFourHours(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 11, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 14, 59, 0, 0, loc)

	err := validateMinimumCheckoutDuration(checkIn, checkOut)

	if err == nil {
		t.Fatal("expected checkout before 4 hours to be rejected")
	}
	domainErr, ok := err.(*domain.DomainError)
	if !ok || domainErr.Type != "VALIDATION_ERROR" {
		t.Fatalf("expected validation error, got %T", err)
	}
	expected := "Bạn mới vào làm lúc 11:00. Chỉ có thể tan ca sau 15:00"
	if err.Error() != expected {
		t.Fatalf("expected %q, got %q", expected, err.Error())
	}
}

func TestValidateMinimumCheckoutDurationAllowsAtFourHours(t *testing.T) {
	loc := time.FixedZone("ICT", 7*60*60)
	checkIn := time.Date(2026, 6, 21, 11, 0, 0, 0, loc)
	checkOut := time.Date(2026, 6, 21, 15, 0, 0, 0, loc)

	if err := validateMinimumCheckoutDuration(checkIn, checkOut); err != nil {
		t.Fatalf("expected checkout at 4 hours to be allowed, got %v", err)
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
