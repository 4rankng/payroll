package text

import (
	"strings"
)

// GenerateUsername generates a username from Vietnamese fullname
// following the pattern: first name initials + last name
// Example: "Nguyen Thi Hong Tham" -> "thamnth"
// Example: "Dang Quan Khai" -> "khaidq"
func GenerateUsername(fullname string) string {
	if fullname == "" {
		return ""
	}

	// Normalize Vietnamese characters to ASCII
	normalized := NormalizeVietnamese(fullname)

	// Split name into parts
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return ""
	}

	// Extract first name (last part) and other parts (middle and last names)
	firstName := parts[len(parts)-1]
	otherParts := parts[:len(parts)-1]

	// Build username: firstname + initials of other parts
	var username strings.Builder
	username.WriteString(strings.ToLower(firstName))

	for _, part := range otherParts {
		if len(part) > 0 {
			username.WriteRune(rune(strings.ToLower(part)[0]))
		}
	}

	return username.String()
}

// EnsureUniqueUsername takes a base username and appends a number if needed
// to ensure uniqueness. The checker function should return true if username exists.
func EnsureUniqueUsername(baseUsername string, checker func(username string) bool) string {
	username := baseUsername
	counter := 1

	// Keep incrementing counter until we find a unique username
	for checker(username) {
		username = baseUsername + string(rune('0'+counter))
		counter++
	}

	return username
}
