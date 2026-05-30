package config

import (
	"time"
)

// HolidayService handles Vietnam public holiday detection
type HolidayService struct{}

// NewHolidayService creates a new holiday service
func NewHolidayService() *HolidayService {
	return &HolidayService{}
}

// Vietnam public holidays in YYYY-MM-DD format
var holidays = []string{
	"2026-01-01",
	"2026-02-16", "2026-02-17", "2026-02-18", "2026-02-19", "2026-02-20",
	"2026-04-26", "2026-04-27",
	"2026-04-30",
	"2026-05-01",
	"2026-09-01", "2026-09-02",
}

// IsHoliday checks if the given date is a Vietnam public holiday
func (h *HolidayService) IsHoliday(date string) bool {
	for _, holiday := range holidays {
		if holiday == date {
			return true
		}
	}
	return false
}

// GetDayType determines the day type based on Vietnam calendar rules
func (h *HolidayService) GetDayType(date time.Time) string {
	// Format date as YYYY-MM-DD
	dateStr := date.Format("2006-01-02")

	// Check if it's a Vietnam public holiday
	if h.IsHoliday(dateStr) {
		return "Ngày lễ"
	}

	// Check day of week
	weekday := date.Weekday()
	switch weekday {
	case time.Sunday:
		return "Ngày nghỉ"
	case time.Saturday:
		// Saturday defaults to "ngày nghỉ" if not specified by user
		return "Ngày nghỉ"
	default:
		return "Ngày thường"
	}
}

// IsSaturday checks if the given date is a Saturday
func (h *HolidayService) IsSaturday(date time.Time) bool {
	return date.Weekday() == time.Saturday
}
