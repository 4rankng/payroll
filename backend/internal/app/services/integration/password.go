package integration

import (
	"crypto/rand"
	"math/big"
)

// password classes; the concatenation plus shuffle satisfies the project
// password policy (≥1 upper, ≥1 lower, ≥1 digit, ≥1 special) by construction.
const (
	passwordUpper   = "ABCDEFGHJKLMNPQRSTUVWXYZ" // no I/O (visual ambiguity)
	passwordLower   = "abcdefghijkmnpqrstuvwxyz" // no l/o
	passwordDigits  = "23456789"                 // no 0/1
	passwordSpecial = "!@#$%^&*"
)

// generatedPasswordLen is the fixed length of a server-generated password.
const generatedPasswordLen = 12

// GeneratePassword returns a 12-character random password that satisfies the
// project password policy: 3 upper, 3 lower, 4 digits, 2 specials, shuffled
// with a crypto/rand Fisher–Yates so the mandatory classes are not ordered.
func GeneratePassword() (string, error) {
	chars := make([]byte, 0, generatedPasswordLen)
	var err error
	if chars, err = appendClass(chars, passwordUpper, 3); err != nil {
		return "", err
	}
	if chars, err = appendClass(chars, passwordLower, 3); err != nil {
		return "", err
	}
	if chars, err = appendClass(chars, passwordDigits, 4); err != nil {
		return "", err
	}
	if chars, err = appendClass(chars, passwordSpecial, 2); err != nil {
		return "", err
	}

	if err := secureShuffle(chars); err != nil {
		return "", err
	}
	return string(chars), nil
}

// appendClass draws n characters from set and appends them to dst.
func appendClass(dst []byte, set string, n int) ([]byte, error) {
	setMax := big.NewInt(int64(len(set)))
	for range n {
		idx, err := rand.Int(rand.Reader, setMax)
		if err != nil {
			return nil, err
		}
		dst = append(dst, set[idx.Int64()])
	}
	return dst, nil
}

// secureShuffle performs a crypto/rand Fisher–Yates shuffle in place.
func secureShuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		k := j.Int64()
		b[i], b[k] = b[k], b[i]
	}
	return nil
}
