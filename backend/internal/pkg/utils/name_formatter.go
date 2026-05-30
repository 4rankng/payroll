package utils

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ToVietnameseTitleCase converts a Vietnamese name to title case format.
// Each word's first letter is capitalized while preserving Vietnamese diacritics.
// Leading and trailing spaces are trimmed, and multiple spaces are normalized to single spaces.
// Names must start and end with a letter and contain only letters and spaces.
// All non-letter characters (numbers, special characters, etc.) are removed.
// Example: "  đoàn  thị  hồng  " -> "Đoàn Thị Hồng"
// Example: "  123 đoàn thị hồng !@# " -> "Đoàn Thị Hồng"
func ToVietnameseTitleCase(name string) string {
	// Trim leading and trailing spaces
	name = strings.TrimSpace(name)

	if name == "" {
		return name
	}

	// Remove all non-letter characters except spaces
	// This ensures the name contains only letters and spaces
	var builder strings.Builder
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsSpace(r) {
			builder.WriteRune(r)
		}
	}
	name = builder.String()

	// Trim leading and trailing spaces again after filtering
	name = strings.TrimSpace(name)

	if name == "" {
		return name
	}

	// Split by whitespace (Fields automatically handles multiple spaces)
	words := strings.Fields(name)
	if len(words) == 0 {
		return name
	}

	// Capitalize first letter of each word
	for i, word := range words {
		if word == "" {
			continue
		}

		// Get the first rune and capitalize it
		firstRune, size := utf8.DecodeRuneInString(word)
		if size == 0 {
			continue
		}

		// Capitalize the first rune and lowercase the rest
		capitalizedFirst := string(unicode.ToUpper(firstRune))
		rest := strings.ToLower(word[size:])
		words[i] = capitalizedFirst + rest
	}

	return strings.Join(words, " ")
}
