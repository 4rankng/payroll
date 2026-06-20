package advance_payment

import (
	"testing"
	"time"

	"api-server/internal/pkg/clock"
)

func TestIsRequestWindowLocked(t *testing.T) {
	lockedGap := time.Date(2026, 6, 19, 9, 0, 0, 0, clock.DefaultLocation)
	newPeriodStart := time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation)
	openDay := time.Date(2026, 6, 25, 9, 0, 0, 0, clock.DefaultLocation)

	if !isRequestWindowLocked(lockedGap, false) {
		t.Fatalf("expected day 19 without quota to stay locked")
	}
	if isRequestWindowLocked(lockedGap, true) {
		t.Fatalf("expected day 19 with uploaded quota to unlock requests")
	}
	if isRequestWindowLocked(newPeriodStart, false) {
		t.Fatalf("expected day 20 to be treated as new period")
	}
	if isRequestWindowLocked(openDay, false) {
		t.Fatalf("expected day 25 to stay open")
	}
}

func TestIsRequestMonthAllowed(t *testing.T) {
	day5 := time.Date(2026, 6, 5, 9, 0, 0, 0, clock.DefaultLocation)
	day20 := time.Date(2026, 6, 20, 9, 0, 0, 0, clock.DefaultLocation)
	day25 := time.Date(2026, 6, 25, 9, 0, 0, 0, clock.DefaultLocation)

	if !isRequestMonthAllowed(day5, "2026-05") || !isRequestMonthAllowed(day5, "2026-06") {
		t.Fatalf("expected day 5 to allow previous and current calendar month")
	}
	if isRequestMonthAllowed(day5, "2026-04") {
		t.Fatalf("did not expect day 5 to allow unrelated month")
	}

	if !isRequestMonthAllowed(day20, "2026-06") {
		t.Fatalf("expected day 20 to allow current month when quota exists")
	}
	if isRequestMonthAllowed(day20, "2026-05") {
		t.Fatalf("did not expect day 20 to allow previous month")
	}

	if !isRequestMonthAllowed(day25, "2026-06") {
		t.Fatalf("expected day 25 to allow current month")
	}
	if isRequestMonthAllowed(day25, "2026-05") {
		t.Fatalf("did not expect day 25 to allow previous month")
	}
}
