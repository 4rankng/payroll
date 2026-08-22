package project_employee

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveCheckInConfigurationMonth(t *testing.T) {
	now := time.Date(2026, time.August, 22, 10, 0, 0, 0, time.FixedZone("ICT", 7*60*60))

	t.Run("uses the current month when the query is empty", func(t *testing.T) {
		month, err := resolveCheckInConfigurationMonth("", now)

		require.NoError(t, err)
		require.Equal(t, "2026-08", month.Format("2006-01"))
	})

	t.Run("accepts a historical month", func(t *testing.T) {
		month, err := resolveCheckInConfigurationMonth("2026-07", now)

		require.NoError(t, err)
		require.Equal(t, "2026-07", month.Format("2006-01"))
		require.Equal(t, now.Location(), month.Location())
	})

	for _, value := range []string{"2026-8", "not-a-month", "2026-09", "2024-07"} {
		t.Run("rejects "+value, func(t *testing.T) {
			_, err := resolveCheckInConfigurationMonth(value, now)
			require.Error(t, err)
		})
	}
}
