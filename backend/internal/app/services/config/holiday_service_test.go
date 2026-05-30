package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestHolidayService_IsHoliday(t *testing.T) {
	service := NewHolidayService()

	tests := []struct {
		name string
		date string
		want bool
	}{
		{
			name: "New Year 2026",
			date: "2026-01-01",
			want: true,
		},
		{
			name: "Tet 2026 - Day 1",
			date: "2026-02-16",
			want: true,
		},
		{
			name: "Tet 2026 - Day 2",
			date: "2026-02-17",
			want: true,
		},
		{
			name: "Hung Kings 2026",
			date: "2026-04-26",
			want: true,
		},
		{
			name: "Reunification Day",
			date: "2026-04-30",
			want: true,
		},
		{
			name: "Labor Day",
			date: "2026-05-01",
			want: true,
		},
		{
			name: "National Day 2026 - Day 1",
			date: "2026-09-01",
			want: true,
		},
		{
			name: "National Day 2026 - Day 2",
			date: "2026-09-02",
			want: true,
		},
		{
			name: "Regular day",
			date: "2026-03-15",
			want: false,
		},
		{
			name: "Different year",
			date: "2025-01-01",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsHoliday(tt.date)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHolidayService_GetDayType(t *testing.T) {
	service := NewHolidayService()

	tests := []struct {
		name string
		date time.Time
		want string
	}{
		{
			name: "Sunday",
			date: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC), // Sunday
			want: "Ngày nghỉ",
		},
		{
			name: "Saturday",
			date: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), // Saturday
			want: "Ngày nghỉ",
		},
		{
			name: "Monday (weekday)",
			date: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), // Monday
			want: "Ngày thường",
		},
		{
			name: "Tuesday (weekday)",
			date: time.Date(2026, 1, 6, 0, 0, 0, 0, time.UTC), // Tuesday
			want: "Ngày thường",
		},
		{
			name: "Wednesday (weekday)",
			date: time.Date(2026, 1, 7, 0, 0, 0, 0, time.UTC), // Wednesday
			want: "Ngày thường",
		},
		{
			name: "Thursday (weekday)",
			date: time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC), // Thursday
			want: "Ngày thường",
		},
		{
			name: "Friday (weekday)",
			date: time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC), // Friday
			want: "Ngày thường",
		},
		{
			name: "New Year (public holiday)",
			date: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			want: "Ngày lễ",
		},
		{
			name: "Labor Day (public holiday)",
			date: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			want: "Ngày lễ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.GetDayType(tt.date)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHolidayService_IsSaturday(t *testing.T) {
	service := NewHolidayService()

	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{
			name: "Saturday",
			date: time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
			want: true,
		},
		{
			name: "Sunday",
			date: time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "Monday",
			date: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC),
			want: false,
		},
		{
			name: "Friday",
			date: time.Date(2026, 1, 9, 0, 0, 0, 0, time.UTC),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := service.IsSaturday(tt.date)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewHolidayService(t *testing.T) {
	service := NewHolidayService()
	assert.NotNil(t, service)
}
