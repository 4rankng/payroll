package zalo

import (
	"regexp"
	"strings"
)

// phoneDigits strips every non-digit rune. Port of the PHP preg_replace('/\D+/', ”, ...).
func phoneDigits(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// vnPhoneRe validates the final normalized form: country code 84 + exactly 9
// digits (Vietnamese mobile numbers). Port of PHP '/^84\d{9}$/'.
var vnPhoneRe = regexp.MustCompile(`^84\d{9}$`)

// NormalizePhone converts any common Vietnamese phone representation to the
// `84xxxxxxxxx` form Zalo requires. Accepts domestic ("0987654321"),
// already-international ("84987654321"), prefixed ("+84987654321"), and
// whitespace/punctuated ("0987 654 321") inputs. Returns "" for anything that
// does not normalize to exactly 84 + 9 digits (the caller then short-circuits
// the send, matching PHP's behavior that avoids Zalo error -108).
//
// Port of vfic_zns_normalize_phone in functions-zns.php.
func NormalizePhone(phone string) string {
	if phone == "" {
		return ""
	}
	digits := phoneDigits(phone)
	if digits == "" {
		return ""
	}

	var normalized string
	// Already carries the 84 country code.
	if strings.HasPrefix(digits, "84") {
		rest := strings.TrimLeft(digits[2:], "0")
		normalized = "84" + rest
	} else {
		// Domestic: drop the leading 0 (if any), prepend 84.
		normalized = "84" + strings.TrimLeft(digits, "0")
	}

	if !vnPhoneRe.MatchString(normalized) {
		return ""
	}
	return normalized
}
