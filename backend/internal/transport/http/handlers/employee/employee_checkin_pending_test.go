package employee

import (
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

// A deferred check-in enable (activation on day 1 of the next month) must reach
// the client. Without these fields the employee detail sheet cannot tell "off"
// apart from "turns on 01/10" and renders the toggle as a plain "off" switch.
func TestBuildProjectInfoFromAssignmentCarriesPendingCheckInEnable(t *testing.T) {
	pending := true
	effective := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	info := buildProjectInfoFromAssignment(domain.ProjectEmployee{
		ProjectID:             58,
		ID:                    1674,
		Position:              "Phổ Thông",
		PaymentSchedule:       string(domain.PaymentScheduleFlexible),
		CheckInEnabled:        false,
		PendingCheckInEnabled: &pending,
		CheckInEffectiveFrom:  &effective,
	})

	require.False(t, info.CheckInEnabled, "not active yet")
	require.NotNil(t, info.PendingCheckInEnabled, "pending flag must survive mapping")
	require.True(t, *info.PendingCheckInEnabled)
	require.NotNil(t, info.CheckInEffectiveFrom, "effective date must survive mapping")
	require.Equal(t, "2026-10-01", *info.CheckInEffectiveFrom)
}

// The no-pending case must keep both pointers nil so the JSON omitempty tags
// drop them and the client renders a normal, immediately-effective switch.
func TestBuildProjectInfoFromAssignmentOmitsPendingWhenAbsent(t *testing.T) {
	info := buildProjectInfoFromAssignment(domain.ProjectEmployee{
		ProjectID:       58,
		CheckInEnabled:  true,
		PaymentSchedule: string(domain.PaymentScheduleWeekly),
	})

	require.True(t, info.CheckInEnabled)
	require.Nil(t, info.PendingCheckInEnabled)
	require.Nil(t, info.CheckInEffectiveFrom)
}

// A cancelled pending enable (user hit "Hủy") is pending=false, not nil. The
// sheet's isPending check is `!enabled && !!pending`, so false must not be
// misread as an active request.
func TestBuildProjectInfoFromAssignmentCarriesCancelledPending(t *testing.T) {
	cancelled := false

	info := buildProjectInfoFromAssignment(domain.ProjectEmployee{
		ProjectID:             58,
		CheckInEnabled:        false,
		PendingCheckInEnabled: &cancelled,
	})

	require.NotNil(t, info.PendingCheckInEnabled)
	require.False(t, *info.PendingCheckInEnabled, "cancelled pending must not read as an active request")
}
