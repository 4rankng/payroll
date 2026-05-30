package scheduler_test

import (
	"testing"
	"time"

	"api-server/internal/app/services/scheduler"
	"api-server/internal/infra/observability"

	"github.com/stretchr/testify/assert"
)

func TestNewScheduler(t *testing.T) {
	logger := observability.GetLogger()
	s := scheduler.NewScheduler(logger, "UTC", true)
	assert.NotNil(t, s)
}

func TestScheduler_AddJob(t *testing.T) {
	logger := observability.GetLogger()
	s := scheduler.NewScheduler(logger, "UTC", true)

	job := scheduler.Job{
		Name:    "test_job",
		Cron:    "* * * * *",
		Enabled: true,
		Handler: func() {
			// Handler function
		},
	}

	s.AddJob(job)
	// We can't easily inspect private fields, but we can verify no panic
	assert.NotNil(t, s)
}

func TestIsLastDayOfMonth(t *testing.T) {
	tests := []struct {
		name     string
		date     time.Time
		expected bool
	}{
		{
			name:     "Last day of January",
			date:     time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Not last day of January",
			date:     time.Date(2023, 1, 30, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Last day of February (non-leap)",
			date:     time.Date(2023, 2, 28, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Last day of February (leap)",
			date:     time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, scheduler.IsLastDayOfMonth(tt.date))
		})
	}
}
