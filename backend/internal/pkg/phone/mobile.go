package phone

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidVietnameseMobile = errors.New("invalid Vietnamese mobile number")
	vietnameseMobilePattern    = regexp.MustCompile(`^0(3[2-9]|5[689]|7[06-9]|8[1-689]|9[0-46-9])\d{7}$`)
	cccdPattern                = regexp.MustCompile(`^\d{9}(\d{3})?$`)
	cccdCardPattern            = regexp.MustCompile(`^\d{12}$`)
)

// IsCCCDCard reports whether value is a 12-digit CCCD (citizen identity card
// issued since 2016). A 12-digit number can never be a Vietnamese mobile: a
// domestic mobile is 010 + 8 digits, and even the 84 country-code form is 11
// digits. Used to reject a CCCD typed into a phone field.
//
// Deliberately narrower than IsCCCDFormat: 9-digit values are ambiguous
// (legacy 9-digit CMND vs a mobile written without its leading zero), so they
// are NOT treated as CCCD here.
func IsCCCDCard(value string) bool {
	return cccdCardPattern.MatchString(strings.TrimSpace(value))
}

// IsCCCDFormat reports whether the identifier is shaped like a Vietnamese
// national ID (9-digit CMND or 12-digit CCCD). Deliberately false for
// 10-digit strings: those are mobile numbers, so a mobile value wrongly
// stored in users.cccd must never match a phone-number login identifier.
func IsCCCDFormat(value string) bool {
	return cccdPattern.MatchString(value)
}

// NormalizeVietnameseMobile converts common Vietnamese mobile representations
// to the domestic 0XXXXXXXXX form used for exact-match login and uniqueness.
func NormalizeVietnameseMobile(value string) (string, error) {
	cleaned := strings.NewReplacer(
		" ", "",
		"-", "",
		".", "",
		"(", "",
		")", "",
	).Replace(strings.TrimSpace(value))

	switch {
	case strings.HasPrefix(cleaned, "+84"):
		cleaned = "0" + strings.TrimPrefix(cleaned, "+84")
	case strings.HasPrefix(cleaned, "84"):
		cleaned = "0" + strings.TrimPrefix(cleaned, "84")
	}

	if !vietnameseMobilePattern.MatchString(cleaned) {
		return "", ErrInvalidVietnameseMobile
	}

	return cleaned, nil
}
