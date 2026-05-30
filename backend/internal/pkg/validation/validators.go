package validation

import (
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"
)

var (
	// ErrInvalidEmail is returned when an email address is invalid
	ErrInvalidEmail = errors.New("invalid email address")
	// ErrInvalidID is returned when an ID is invalid
	ErrInvalidID = errors.New("invalid ID")
	// ErrEmptyValue is returned when a required value is empty
	ErrEmptyValue = errors.New("value cannot be empty")
)

// ValidateEmail validates an email address format.
// Returns nil if valid, error otherwise.
func ValidateEmail(email string) error {
	if email == "" {
		return ErrEmptyValue
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}

	return nil
}

// IsValidEmail checks if an email address is valid.
// Returns true if valid, false otherwise.
func IsValidEmail(email string) bool {
	return ValidateEmail(email) == nil
}

// ParseUint parses a string to uint with validation.
// Returns error if the string is empty, not a valid number, or negative.
func ParseUint(str string) (uint, error) {
	if str == "" {
		return 0, ErrEmptyValue
	}

	val, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidID, str)
	}

	if val == 0 {
		return 0, fmt.Errorf("%w: ID cannot be zero", ErrInvalidID)
	}

	return uint(val), nil
}

// ParseUintOrDefault parses a string to uint, returning a default value if parsing fails.
func ParseUintOrDefault(str string, defaultVal uint) uint {
	val, err := ParseUint(str)
	if err != nil {
		return defaultVal
	}
	return val
}

// ParseIDs parses a comma-separated string of IDs into a slice of uints.
// Invalid IDs are skipped. Returns empty slice if input is empty or all IDs are invalid.
func ParseIDs(idsStr string) []uint {
	if idsStr == "" {
		return nil
	}

	parts := strings.Split(idsStr, ",")
	ids := make([]uint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := ParseUint(part)
		if err == nil {
			ids = append(ids, id)
		}
	}

	return ids
}

// ValidateRequired checks if a string value is not empty.
// Returns error if empty, nil otherwise.
func ValidateRequired(fieldName, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", fieldName)
	}
	return nil
}

// ValidateMinLength checks if a string has at least the specified minimum length.
func ValidateMinLength(fieldName, value string, minLength int) error {
	if len(value) < minLength {
		return fmt.Errorf("%s must be at least %d characters", fieldName, minLength)
	}
	return nil
}

// ValidateMaxLength checks if a string does not exceed the specified maximum length.
func ValidateMaxLength(fieldName, value string, maxLength int) error {
	if len(value) > maxLength {
		return fmt.Errorf("%s must not exceed %d characters", fieldName, maxLength)
	}
	return nil
}

// ValidateRange checks if a numeric value is within the specified range (inclusive).
func ValidateRange(fieldName string, value, min, max int64) error {
	if value < min || value > max {
		return fmt.Errorf("%s must be between %d and %d", fieldName, min, max)
	}
	return nil
}

// IsPositive checks if a number is greater than zero.
func IsPositive(value int64) bool {
	return value > 0
}

// IsNonNegative checks if a number is greater than or equal to zero.
func IsNonNegative(value int64) bool {
	return value >= 0
}
