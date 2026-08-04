package phone

import (
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidVietnameseMobile = errors.New("invalid Vietnamese mobile number")
	vietnameseMobilePattern    = regexp.MustCompile(`^0(3[2-9]|5[689]|7[06-9]|8[1-689]|9[0-46-9])\d{7}$`)
)

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
