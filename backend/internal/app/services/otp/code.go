package otp

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"fmt"
	"hash/fnv"
)

// codeModulo is the 6-digit code space [0, 999999].
const codeModulo = 1_000_000

// codeFormatWidth is the zero-padded width of the numeric string form.
const codeFormatWidth = 6

// maxUnbiased is the largest multiple of codeModulo that fits in a uint32.
// Values drawn in [0, maxUnbiased) map uniformly into the code space; values
// >= maxUnbiased are rejected to eliminate modulo bias.
var maxUnbiased = uint32(0xFFFFFFFF/codeModulo) * codeModulo

// GenerateCode returns a cryptographically-unbiased 6-digit numeric code as a
// zero-padded string (e.g. "004281"). It uses rejection sampling over
// crypto/rand to eliminate the modulo bias a naive `rand % 1e6` would introduce.
func GenerateCode() (string, error) {
	for i := 0; i < 16; i++ {
		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("otp: read random bytes: %w", err)
		}
		v := binary.BigEndian.Uint32(buf)
		// Rejection sampling: only accept values that map cleanly into the
		// modulo range, so every code in [0, 999999] is equally likely.
		if v < maxUnbiased {
			n := int(v % codeModulo)
			return fmt.Sprintf("%0*d", codeFormatWidth, n), nil
		}
	}
	// Practically unreachable — 16 rejection rounds with a ~0.0002% reject rate.
	return "", fmt.Errorf("otp: exhausted rejection-sampling attempts")
}

// HashCode returns a fast, non-cryptographic 64-bit fingerprint of the code,
// suitable for storing in the Redis pending session. A 6-digit code has too
// small a keyspace for hashing to add offline-crack resistance; the real
// brute-force defense is the per-account attempt cap. Hashing here is for log
// hygiene and to avoid storing/transmitting the plaintext code unnecessarily.
//
// Compared constant-time via EqualCodeHash to avoid timing oracles on the compare.
func HashCode(code string) []byte {
	h := fnv.New64a()
	h.Write([]byte(code))
	out := make([]byte, 8)
	binary.BigEndian.PutUint64(out, h.Sum64())
	return out
}

// EqualCodeHash compares a code against a stored hash in constant time. Returns
// true on match. An empty/nil stored hash never matches.
func EqualCodeHash(code string, stored []byte) bool {
	if len(stored) == 0 {
		return false
	}
	got := HashCode(code)
	return subtle.ConstantTimeCompare(got, stored) == 1
}
