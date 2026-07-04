package otp

import (
	"regexp"
	"testing"
)

// TestGenerateCode_Format verifies every generated code is exactly 6 digits,
// zero-padded, and matches the expected character set.
func TestGenerateCode_Format(t *testing.T) {
	codeRE := regexp.MustCompile(`^[0-9]{6}$`)
	for i := 0; i < 1000; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode returned error: %v", err)
		}
		if !codeRE.MatchString(code) {
			t.Fatalf("code %q does not match ^[0-9]{6}$", code)
		}
	}
}

// TestGenerateCode_Distribution verifies the rejection sampling is unbiased:
// over a large sample, the first digit should be roughly uniform across 0-9
// (a biased generator would over-represent high or low leading digits). This is
// a lightweight check that the modulo-bias fix is actually in effect.
func TestGenerateCode_Distribution(t *testing.T) {
	const sampleN = 60_000
	firstDigit := make(map[byte]int)
	for i := 0; i < sampleN; i++ {
		code, err := GenerateCode()
		if err != nil {
			t.Fatalf("GenerateCode returned error: %v", err)
		}
		firstDigit[code[0]]++
	}
	// With uniform sampling, each leading digit '0'..'9' should appear ~10% of
	// the time. Allow ±20% relative tolerance (8%-12%) to avoid flakiness.
	expected := sampleN / 10
	tolerance := expected / 5 // 20%
	for d := byte('0'); d <= '9'; d++ {
		count := firstDigit[d]
		if count < expected-tolerance || count > expected+tolerance {
			t.Fatalf("leading digit %c: count=%d, expected %d±%d (uniform)", d, count, expected, tolerance)
		}
	}
}

// TestHashCode_RoundTrip verifies a code hashes to a stable value and that
// EqualCodeHash matches the original but not a different code.
func TestHashCode_RoundTrip(t *testing.T) {
	code := "123456"
	h := HashCode(code)
	if len(h) != 8 {
		t.Fatalf("hash length = %d, want 8", len(h))
	}
	if !EqualCodeHash(code, h) {
		t.Fatalf("EqualCodeHash(%q, h) = false, want true", code)
	}
	if EqualCodeHash("999999", h) {
		t.Fatalf("EqualCodeHash different code = true, want false")
	}
	// Empty/nil stored hash never matches.
	if EqualCodeHash(code, nil) {
		t.Fatalf("EqualCodeHash nil stored = true, want false")
	}
	if EqualCodeHash(code, []byte{}) {
		t.Fatalf("EqualCodeHash empty stored = true, want false")
	}
}
