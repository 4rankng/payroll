package advance_payment

import (
	"testing"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
)

// The advance request kill switch ("tạm ngừng ứng lương") blocks new advance
// requests when an ACTIVE flexible assignment has AdvanceRequestEnabled=false.
// Ended or non-flexible assignments must never block.

func TestDeriveAdvanceEligibilityKillSwitch(t *testing.T) {
	flex := string(domain.PaymentScheduleFlexible)
	lastDate := time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		assignments  []*domain.ProjectEmployee
		wantFlexible bool
		wantCheckIn  bool
		wantDisabled bool
	}{
		{
			name:         "active flexible enabled",
			assignments:  []*domain.ProjectEmployee{{PaymentSchedule: flex, AdvanceRequestEnabled: true}},
			wantFlexible: true,
		},
		{
			name:         "active flexible disabled blocks",
			assignments:  []*domain.ProjectEmployee{{PaymentSchedule: flex, AdvanceRequestEnabled: false}},
			wantFlexible: true,
			wantDisabled: true,
		},
		{
			name: "ended flexible disabled does not block",
			assignments: []*domain.ProjectEmployee{{
				PaymentSchedule: flex, LastDate: &lastDate, AdvanceRequestEnabled: false,
			}},
		},
		{
			name: "non-flexible disabled does not block",
			assignments: []*domain.ProjectEmployee{{
				PaymentSchedule: "weekly", AdvanceRequestEnabled: false,
			}},
		},
		{
			name: "any active flexible disabled blocks alongside enabled one",
			assignments: []*domain.ProjectEmployee{
				{PaymentSchedule: flex, AdvanceRequestEnabled: true, CheckInEnabled: true},
				{PaymentSchedule: flex, AdvanceRequestEnabled: false},
			},
			wantFlexible: true,
			wantCheckIn:  true,
			wantDisabled: true,
		},
		{
			name: "check-in enabled regression",
			assignments: []*domain.ProjectEmployee{{
				PaymentSchedule: flex, AdvanceRequestEnabled: true, CheckInEnabled: true,
			}},
			wantFlexible: true,
			wantCheckIn:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := deriveAdvanceEligibility(tc.assignments)
			if got.hasFlexible != tc.wantFlexible {
				t.Fatalf("hasFlexible = %v, want %v", got.hasFlexible, tc.wantFlexible)
			}
			if got.hasCheckInEnabled != tc.wantCheckIn {
				t.Fatalf("hasCheckInEnabled = %v, want %v", got.hasCheckInEnabled, tc.wantCheckIn)
			}
			if got.advanceRequestDisabled != tc.wantDisabled {
				t.Fatalf("advanceRequestDisabled = %v, want %v", got.advanceRequestDisabled, tc.wantDisabled)
			}
		})
	}
}

func TestAdvanceRequestGateConstants(t *testing.T) {
	if constants.MsgAdvanceRequestsDisabledVN == "" {
		t.Fatalf("MsgAdvanceRequestsDisabledVN must not be empty (400 rejection message)")
	}
	if constants.MsgAdvanceRequestPausedTitleVN == "" || constants.MsgAdvanceRequestPausedReasonVN == "" {
		t.Fatalf("paused title/reason constants must not be empty (employee notice)")
	}
}
