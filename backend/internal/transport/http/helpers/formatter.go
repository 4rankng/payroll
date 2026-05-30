package helpers

import "time"

// DateToStringPtr converts a time pointer to a string pointer in YYYY-MM-DD format.
// Returns nil if the input time is nil.
func DateToStringPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format("2006-01-02")
	return &str
}

// DateToString converts a time pointer to a string in YYYY-MM-DD format.
// Returns empty string if the input time is nil.
func DateToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// DateTimeToString converts a time pointer to a datetime string in YYYY-MM-DD HH:MM:SS format.
// Returns empty string if the input time is nil.
func DateTimeToString(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}

// StringToDate parses a date string in YYYY-MM-DD format to a time pointer.
// Returns nil if the input string is empty.
// Returns error if the string cannot be parsed.
func StringToDate(dateStr string) (*time.Time, error) {
	if dateStr == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// StringToDateTime parses a datetime string in YYYY-MM-DD HH:MM:SS format to a time pointer.
// Returns nil if the input string is empty.
// Returns error if the string cannot be parsed.
func StringToDateTime(dateTimeStr string) (*time.Time, error) {
	if dateTimeStr == "" {
		return nil, nil
	}
	t, err := time.Parse("2006-01-02 15:04:05", dateTimeStr)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FormatDate formats a time.Time value to YYYY-MM-DD string.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatDateTime formats a time.Time value to YYYY-MM-DD HH:MM:SS string.
func FormatDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
