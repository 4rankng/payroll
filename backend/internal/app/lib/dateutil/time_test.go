package dateutil

import (
	"testing"
	"time"
)

func TestNow(t *testing.T) {
	now := Now()
	if now.IsZero() {
		t.Error("Now() returned zero time")
	}
}

func TestNowUTC(t *testing.T) {
	now := NowUTC()
	if now.Location() != time.UTC {
		t.Errorf("NowUTC() did not return UTC time, got location: %v", now.Location())
	}
}

func TestNowUnix(t *testing.T) {
	unix := NowUnix()
	if unix == 0 {
		t.Error("NowUnix() returned zero")
	}
}

func TestTodayStart(t *testing.T) {
	start := TodayStart()
	if start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Errorf("TodayStart() did not return start of day: %v", start)
	}
}

func TestStartOfDay(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfDay(testTime)

	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfDay() = %v, want %v", start, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfDay(testTime)

	// Check that it's the last nanosecond of the day
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("EndOfDay() did not return end of day: %v", end)
	}
}

func TestDaysAgo(t *testing.T) {
	sevenDaysAgo := DaysAgo(7)
	diff := time.Since(sevenDaysAgo)

	// Should be approximately 7 days (within a small margin)
	expectedDuration := 7 * 24 * time.Hour
	if diff < expectedDuration-time.Minute || diff > expectedDuration+time.Minute {
		t.Errorf("DaysAgo(7) returned incorrect time, diff: %v", diff)
	}
}

func TestDaysFromNow(t *testing.T) {
	sevenDaysLater := DaysFromNow(7)
	diff := time.Until(sevenDaysLater)

	// Should be approximately 7 days (within a small margin)
	expectedDuration := 7 * 24 * time.Hour
	if diff < expectedDuration-time.Minute || diff > expectedDuration+time.Minute {
		t.Errorf("DaysFromNow(7) returned incorrect time, diff: %v", diff)
	}
}

func TestStartOfWeek(t *testing.T) {
	// March 15, 2024 is a Friday
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfWeek(testTime)

	// Monday March 11, 2024 00:00:00
	expected := time.Date(2024, 3, 11, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfWeek() = %v, want %v", start, expected)
	}
	if start.Weekday() != time.Monday {
		t.Errorf("StartOfWeek() weekday = %v, want Monday", start.Weekday())
	}
}

func TestStartOfMonth(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfMonth(testTime)

	expected := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfMonth() = %v, want %v", start, expected)
	}
}

func TestEndOfMonth(t *testing.T) {
	testTime := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfMonth(testTime)

	// March has 31 days
	if end.Day() != 31 || end.Month() != time.March || end.Year() != 2024 {
		t.Errorf("EndOfMonth() = %v, expected last day of March 2024", end)
	}
}

func TestStartOfYear(t *testing.T) {
	testTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	start := StartOfYear(testTime)

	expected := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !start.Equal(expected) {
		t.Errorf("StartOfYear() = %v, want %v", start, expected)
	}
}

func TestEndOfYear(t *testing.T) {
	testTime := time.Date(2024, 6, 15, 14, 30, 45, 0, time.UTC)
	end := EndOfYear(testTime)

	if end.Day() != 31 || end.Month() != time.December || end.Year() != 2024 {
		t.Errorf("EndOfYear() = %v, expected Dec 31, 2024", end)
	}
}

func TestIsToday(t *testing.T) {
	now := time.Now().UTC()
	if !IsToday(now) {
		t.Error("IsToday(now) should return true")
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	if IsToday(yesterday) {
		t.Error("IsToday(yesterday) should return false")
	}
}

func TestIsPast(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	if !IsPast(past) {
		t.Error("IsPast() should return true for past time")
	}

	future := time.Now().Add(1 * time.Hour)
	if IsPast(future) {
		t.Error("IsPast() should return false for future time")
	}
}

func TestIsFuture(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	if !IsFuture(future) {
		t.Error("IsFuture() should return true for future time")
	}

	past := time.Now().Add(-1 * time.Hour)
	if IsFuture(past) {
		t.Error("IsFuture() should return false for past time")
	}
}

func TestDaysBetween(t *testing.T) {
	t1 := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 3, 8, 0, 0, 0, 0, time.UTC)

	days := DaysBetween(t1, t2)
	if days != 7 {
		t.Errorf("DaysBetween() = %d, want 7", days)
	}

	// Test reverse order (should still be positive)
	days = DaysBetween(t2, t1)
	if days != 7 {
		t.Errorf("DaysBetween() = %d, want 7", days)
	}
}

func TestFormatUnixTimestamp(t *testing.T) {
	// Unix timestamp for 2024-03-15 14:30:45 UTC
	timestamp := int64(1710513045)
	formatted := FormatUnixTimestamp(timestamp)

	expected := "2024-03-15 14:30:45"
	if formatted != expected {
		t.Errorf("FormatUnixTimestamp() = %s, want %s", formatted, expected)
	}
}

func TestParseDate(t *testing.T) {
	dateStr := "2024-03-15"
	parsed, err := ParseDate(dateStr)
	if err != nil {
		t.Errorf("ParseDate() error = %v", err)
	}

	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !parsed.Equal(expected) {
		t.Errorf("ParseDate() = %v, want %v", parsed, expected)
	}
}

func TestParseDateTime(t *testing.T) {
	dateTimeStr := "2024-03-15 14:30:45"
	parsed, err := ParseDateTime(dateTimeStr)
	if err != nil {
		t.Errorf("ParseDateTime() error = %v", err)
	}

	expected := time.Date(2024, 3, 15, 14, 30, 45, 0, time.UTC)
	if !parsed.Equal(expected) {
		t.Errorf("ParseDateTime() = %v, want %v", parsed, expected)
	}
}

func TestMustParseDate(t *testing.T) {
	dateStr := "2024-03-15"
	parsed := MustParseDate(dateStr)

	expected := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)
	if !parsed.Equal(expected) {
		t.Errorf("MustParseDate() = %v, want %v", parsed, expected)
	}
}

func TestMustParseDatePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustParseDate() should panic on invalid date")
		}
	}()

	MustParseDate("invalid-date")
}
