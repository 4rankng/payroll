package helpers

import (
	"testing"
	"time"
)

func TestDateToStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		expected *string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name: "valid date",
			input: func() *time.Time {
				t := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				return &t
			}(),
			expected: func() *string {
				s := "2024-01-15"
				return &s
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DateToStringPtr(tt.input)
			if (result == nil && tt.expected != nil) || (result != nil && tt.expected == nil) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
			if result != nil && tt.expected != nil && *result != *tt.expected {
				t.Errorf("expected %s, got %s", *tt.expected, *result)
			}
		})
	}
}

func TestDateToString(t *testing.T) {
	tests := []struct {
		name     string
		input    *time.Time
		expected string
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: "",
		},
		{
			name: "valid date",
			input: func() *time.Time {
				t := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
				return &t
			}(),
			expected: "2024-01-15",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DateToString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestStringToDate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectNil bool
		expectErr bool
	}{
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
			expectErr: false,
		},
		{
			name:      "valid date",
			input:     "2024-01-15",
			expectNil: false,
			expectErr: false,
		},
		{
			name:      "invalid date format",
			input:     "15-01-2024",
			expectNil: false,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := StringToDate(tt.input)

			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if tt.expectNil && result != nil {
				t.Errorf("expected nil but got %v", result)
			}
			if !tt.expectNil && !tt.expectErr && result == nil {
				t.Errorf("expected non-nil result but got nil")
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	result := FormatDate(date)
	expected := "2024-01-15"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFormatDateTime(t *testing.T) {
	date := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	result := FormatDateTime(date)
	expected := "2024-01-15 10:30:45"

	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}
