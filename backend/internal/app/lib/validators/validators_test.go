package validators

import (
	"testing"
)

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		expectErr bool
	}{
		{"valid email", "test@example.com", false},
		{"valid email with subdomain", "user@mail.example.com", false},
		{"empty email", "", true},
		{"invalid email - no @", "testexample.com", true},
		{"invalid email - no domain", "test@", true},
		{"invalid email - no user", "@example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseUint(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint
		expectErr bool
	}{
		{"valid number", "123", 123, false},
		{"large number", "999999", 999999, false},
		{"empty string", "", 0, true},
		{"zero", "0", 0, true},
		{"negative number", "-5", 0, true},
		{"invalid string", "abc", 0, true},
		{"decimal number", "12.5", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseUint(tt.input)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !tt.expectErr && result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseUintOrDefault(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultVal uint
		expected   uint
	}{
		{"valid number", "123", 10, 123},
		{"invalid uses default", "abc", 10, 10},
		{"empty uses default", "", 10, 10},
		{"zero uses default", "0", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseUintOrDefault(tt.input, tt.defaultVal)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []uint
	}{
		{"valid single ID", "123", []uint{123}},
		{"valid multiple IDs", "1,2,3,4", []uint{1, 2, 3, 4}},
		{"IDs with spaces", "1, 2, 3", []uint{1, 2, 3}},
		{"empty string", "", nil},
		{"only commas", ",,,", nil},
		{"mixed valid and invalid", "1,abc,3", []uint{1, 3}},
		{"skip zero", "1,0,3", []uint{1, 3}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseIDs(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
				return
			}
			for i, val := range result {
				if val != tt.expected[i] {
					t.Errorf("at index %d: expected %d, got %d", i, tt.expected[i], val)
				}
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		value     string
		expectErr bool
	}{
		{"valid value", "username", "john", false},
		{"empty value", "username", "", true},
		{"whitespace only", "username", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.fieldName, tt.value)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateMinLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		minLength int
		expectErr bool
	}{
		{"valid length", "password", 8, false},
		{"exact minimum", "12345678", 8, false},
		{"too short", "pass", 8, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinLength("password", tt.value, tt.minLength)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateMaxLength(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxLength int
		expectErr bool
	}{
		{"valid length", "test", 10, false},
		{"exact maximum", "1234567890", 10, false},
		{"too long", "12345678901", 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxLength("field", tt.value, tt.maxLength)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateRange(t *testing.T) {
	tests := []struct {
		name      string
		value     int64
		min       int64
		max       int64
		expectErr bool
	}{
		{"within range", 5, 1, 10, false},
		{"at minimum", 1, 1, 10, false},
		{"at maximum", 10, 1, 10, false},
		{"below minimum", 0, 1, 10, true},
		{"above maximum", 11, 1, 10, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRange("value", tt.value, tt.min, tt.max)
			if tt.expectErr && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsPositive(t *testing.T) {
	tests := []struct {
		value    int64
		expected bool
	}{
		{1, true},
		{100, true},
		{0, false},
		{-1, false},
	}

	for _, tt := range tests {
		result := IsPositive(tt.value)
		if result != tt.expected {
			t.Errorf("IsPositive(%d): expected %v, got %v", tt.value, tt.expected, result)
		}
	}
}

func TestIsNonNegative(t *testing.T) {
	tests := []struct {
		value    int64
		expected bool
	}{
		{1, true},
		{0, true},
		{-1, false},
	}

	for _, tt := range tests {
		result := IsNonNegative(tt.value)
		if result != tt.expected {
			t.Errorf("IsNonNegative(%d): expected %v, got %v", tt.value, tt.expected, result)
		}
	}
}
