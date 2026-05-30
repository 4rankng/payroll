// Package dateutil provides date helpers.
// All Now() calls delegate to the clock package for testability.
package dateutil

import (
	"time"

	"api-server/internal/pkg/clock"
)

// Now returns the current time in the application timezone.
func Now() time.Time { return clock.Now() }

// NowUTC returns the current time in UTC.
func NowUTC() time.Time { return clock.NowUTC() }

// NowUnix returns the current Unix timestamp in seconds.
func NowUnix() int64 { return clock.Global().UnixNow() }

// TodayStart returns the start of today (00:00:00) in the application timezone.
func TodayStart() time.Time { return clock.Global().TodayStart() }

// TodayEnd returns the end of today (23:59:59.999999999) in the application timezone.
func TodayEnd() time.Time { return clock.Global().TodayEnd() }

// StartOfDay returns the start of the day (00:00:00) for the given time,
// preserving the input's timezone.
func StartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// EndOfDay returns the end of the day (23:59:59.999999999) for the given time.
func EndOfDay(t time.Time) time.Time {
	return StartOfDay(t).Add(24*time.Hour - time.Nanosecond)
}

// StartOfWeek returns 00:00:00 on Monday of the given time's week,
// preserving the input's timezone.
func StartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
}

// StartOfMonth returns 00:00:00 on the 1st of the given time's month,
// preserving the input's timezone.
func StartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// EndOfMonth returns 23:59:59.999999999 on the last day of the given time's month.
func EndOfMonth(t time.Time) time.Time {
	return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// StartOfYear returns 00:00:00 on Jan 1 of the given time's year,
// preserving the input's timezone.
func StartOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

// EndOfYear returns 23:59:59.999999999 on Dec 31 of the given time's year.
func EndOfYear(t time.Time) time.Time {
	return StartOfYear(t).AddDate(1, 0, 0).Add(-time.Nanosecond)
}

// DaysAgo returns the time n days before now.
func DaysAgo(n int) time.Time { return Now().AddDate(0, 0, -n) }

// DaysFromNow returns the time n days after now.
func DaysFromNow(n int) time.Time { return Now().AddDate(0, 0, n) }

// IsToday reports whether t falls on the current calendar day (VNT).
func IsToday(t time.Time) bool {
	now := Now() // clock.Now() — VNT
	t = t.In(now.Location())
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// IsPast reports whether t is before now.
func IsPast(t time.Time) bool { return t.Before(Now()) }

// IsFuture reports whether t is after now.
func IsFuture(t time.Time) bool { return t.After(Now()) }

// DaysBetween returns the absolute number of days between t1 and t2.
func DaysBetween(t1, t2 time.Time) int {
	d := int(t1.Sub(t2).Hours() / 24)
	if d < 0 {
		return -d
	}
	return d
}

// FormatUnixTimestamp formats a Unix timestamp as "2006-01-02 15:04:05" in UTC.
func FormatUnixTimestamp(ts int64) string {
	return time.Unix(ts, 0).UTC().Format("2006-01-02 15:04:05")
}

// ParseDate parses a date string in "2006-01-02" format.
func ParseDate(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, time.UTC)
}

// ParseDateTime parses a datetime string in "2006-01-02 15:04:05" format.
func ParseDateTime(s string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
}

// MustParseDate parses a date string and panics on error.
func MustParseDate(s string) time.Time {
	t, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return t
}
