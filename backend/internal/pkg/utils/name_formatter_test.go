package utils

import (
	"testing"
)

func TestToVietnameseTitleCase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Basic Vietnamese name",
			input:    "nguyễn văn a",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with extra spaces",
			input:    "  nguyễn   văn   a  ",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with leading/trailing tabs",
			input:    "\tnguyễn văn a\t",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with numbers and special characters",
			input:    "123 nguyễn @# văn !!! a 456",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with leading numbers",
			input:    "999nguyễn văn a",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with trailing special characters",
			input:    "nguyễn văn a!!!@#$",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with mixed case",
			input:    "NGUYỄN VĂN A",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with multiple special characters in between",
			input:    "nguyễn123văn456a",
			expected: "Nguyễnvăna",
		},
		{
			name:     "Already formatted name",
			input:    "Nguyễn Văn A",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with double spaces",
			input:    "nguyễn  văn  a",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Single word name",
			input:    "nguyễn",
			expected: "Nguyễn",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "Only special characters",
			input:    "!@#$%^&*()",
			expected: "",
		},
		{
			name:     "Vietnamese diacritics preserved",
			input:    "đoàn thị hồng",
			expected: "Đoàn Thị Hồng",
		},
		{
			name:     "Name with all Vietnamese tones",
			input:    "trần thị ánh",
			expected: "Trần Thị Ánh",
		},
		{
			name:     "Complex Vietnamese name",
			input:    "  lê   văn   tùng  ",
			expected: "Lê Văn Tùng",
		},
		{
			name:     "Name starting with special character",
			input:    "@#$nguyễn văn a",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name ending with number",
			input:    "nguyễn văn a123",
			expected: "Nguyễn Văn A",
		},
		{
			name:     "Name with only numbers",
			input:    "123456",
			expected: "",
		},
		{
			name:     "Name with lowercase and spaces",
			input:    "nguyễn    thị    hoa",
			expected: "Nguyễn Thị Hoa",
		},
		{
			name:     "Name with tabs and newlines",
			input:    "nguyễn\t\tthị\n\nhoa",
			expected: "Nguyễn Thị Hoa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToVietnameseTitleCase(tt.input)
			if result != tt.expected {
				t.Errorf("ToVietnameseTitleCase(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestToVietnameseTitleCase_OnlyLettersAndSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Name with numbers",
			input: "nguyễn 123 văn a",
		},
		{
			name:  "Name with special characters",
			input: "nguyễn !@# văn a",
		},
		{
			name:  "Name with mixed",
			input: "123 nguyễn @#$ văn !!! a 456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToVietnameseTitleCase(tt.input)

			// Verify result contains only letters and spaces
			for _, r := range result {
				if r != ' ' && !isLetter(r) {
					t.Errorf("ToVietnameseTitleCase(%q) = %q contains non-letter character: %c", tt.input, result, r)
				}
			}
		})
	}
}

func TestToVietnameseTitleCase_StartsAndEndsWithLetter(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Name with leading spaces",
			input: "   nguyễn văn a",
		},
		{
			name:  "Name with trailing spaces",
			input: "nguyễn văn a   ",
		},
		{
			name:  "Name with leading numbers",
			input: "123nguyễn văn a",
		},
		{
			name:  "Name with trailing special chars",
			input: "nguyễn văn a!!!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToVietnameseTitleCase(tt.input)

			if result == "" {
				return // Empty result is acceptable
			}

			// Check first character
			firstChar := []rune(result)[0]
			if !isLetter(firstChar) {
				t.Errorf("ToVietnameseTitleCase(%q) = %q does not start with a letter", tt.input, result)
			}

			// Check last character
			runes := []rune(result)
			lastChar := runes[len(runes)-1]
			if !isLetter(lastChar) {
				t.Errorf("ToVietnameseTitleCase(%q) = %q does not end with a letter", tt.input, result)
			}
		})
	}
}

func TestToVietnameseTitleCase_NoDoubleSpaces(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "Name with double spaces",
			input: "nguyễn  văn  a",
		},
		{
			name:  "Name with triple spaces",
			input: "nguyễn   văn   a",
		},
		{
			name:  "Name with mixed spacing",
			input: "nguyễn    văn a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToVietnameseTitleCase(tt.input)

			// Check for double spaces
			for i := 0; i < len(result)-1; i++ {
				if result[i] == ' ' && result[i+1] == ' ' {
					t.Errorf("ToVietnameseTitleCase(%q) = %q contains double spaces", tt.input, result)
				}
			}
		})
	}
}

// Helper function to check if a rune is a letter
func isLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 127 // Vietnamese characters are > 127 in UTF-8
}
