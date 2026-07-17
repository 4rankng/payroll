package timesheet

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"api-server/internal/domain"
)

func TestToCashReadinessResponse_UsesTargetKyObservedField(t *testing.T) {
	cash := &domain.CashReadiness{
		ObservedApproved:  120_000_000,
		ProjectedExpected: 80_000_000,
		ExpectedTotal:     200_000_000,
		BandLower:         190_000_000,
		BandUpper:         230_000_000,
		NextPayDate:       time.Date(2026, time.July, 24, 0, 0, 0, 0, time.Local),
		PrepareByDate:     time.Date(2026, time.July, 22, 0, 0, 0, 0, time.Local),
	}

	response := toCashReadinessResponse(cash)
	if response.ObservedApproved != cash.ObservedApproved {
		t.Fatalf("ObservedApproved = %d, want %d", response.ObservedApproved, cash.ObservedApproved)
	}

	payload, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal response: %v", err)
	}
	jsonText := string(payload)
	if !strings.Contains(jsonText, `"observed_approved":120000000`) {
		t.Errorf("response missing target-Ky observed field: %s", jsonText)
	}
	if strings.Contains(jsonText, "confirmed_payable") {
		t.Errorf("response still exposes obsolete outstanding-payment field: %s", jsonText)
	}
}
