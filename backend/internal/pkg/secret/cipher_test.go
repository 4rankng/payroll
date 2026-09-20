package secret

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"api-server/internal/app/services/zaloconnect"
)

func testCipher(t *testing.T, seed byte) *Cipher {
	t.Helper()
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = seed + byte(i)
	}
	c, err := NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return c
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	c := testCipher(t, 1)
	plaintext := `{"app_id":"123","secret_key":"s3cr3t","refresh_token":"tok"}`

	sealed, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !IsEncrypted(sealed) {
		t.Fatalf("sealed value %q does not carry the %q prefix", sealed, Prefix)
	}
	if strings.Contains(sealed, "s3cr3t") {
		t.Fatal("sealed value still contains the plaintext secret")
	}

	opened, err := c.Decrypt(sealed)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if opened != plaintext {
		t.Fatalf("Decrypt = %q, want %q", opened, plaintext)
	}
}

func TestEncryptUsesFreshNonceAndIsIdempotentOnSealedInput(t *testing.T) {
	c := testCipher(t, 2)
	plaintext := "token"

	first, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	second, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if first == second {
		t.Fatal("two encryptions of the same plaintext produced identical ciphertext; a nonce is being reused")
	}

	again, err := c.Encrypt(first)
	if err != nil {
		t.Fatalf("Encrypt(sealed): %v", err)
	}
	if again != first {
		t.Fatalf("Encrypt re-sealed an already sealed value: %q", again)
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	c := testCipher(t, 3)
	sealed, err := c.Encrypt(`{"refresh_token":"tok"}`)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	flipped := []byte(sealed)
	// Flip one character of the base64 payload, keeping the prefix intact.
	pos := len(Prefix) + 6
	if flipped[pos] == 'A' {
		flipped[pos] = 'B'
	} else {
		flipped[pos] = 'A'
	}

	if _, err := c.Decrypt(string(flipped)); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("Decrypt(tampered) error = %v, want ErrDecrypt", err)
	}
	if _, err := c.Decrypt(Prefix + "!!not-base64!!"); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("Decrypt(malformed) error = %v, want ErrDecrypt", err)
	}
	if _, err := c.Decrypt(Prefix + "AAAA"); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("Decrypt(truncated) error = %v, want ErrDecrypt", err)
	}
}

func TestDecryptRejectsCiphertextFromAnotherKey(t *testing.T) {
	sealer := testCipher(t, 4)
	other := testCipher(t, 200)

	sealed, err := sealer.Encrypt("token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := other.Decrypt(sealed); !errors.Is(err, ErrDecrypt) {
		t.Fatalf("Decrypt with a different key error = %v, want ErrDecrypt", err)
	}
}

func TestDecryptReturnsLegacyPlaintextUnchanged(t *testing.T) {
	c := testCipher(t, 5)
	legacy := `{"app_id":"123","access_token":"plain"}`

	opened, err := c.Decrypt(legacy)
	if err != nil {
		t.Fatalf("Decrypt(legacy plaintext): %v", err)
	}
	if opened != legacy {
		t.Fatalf("Decrypt(legacy) = %q, want %q", opened, legacy)
	}
}

func TestNilCipherIsPlaintextPassThrough(t *testing.T) {
	var c *Cipher

	if c.Configured() {
		t.Fatal("nil cipher reports itself configured")
	}
	plaintext := `{"app_id":"123"}`
	unchanged, err := c.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt without a key: %v", err)
	}
	if unchanged != plaintext {
		t.Fatalf("Encrypt without a key = %q, want the plaintext back", unchanged)
	}
	if opened, err := c.Decrypt(plaintext); err != nil || opened != plaintext {
		t.Fatalf("Decrypt(plaintext) without a key = %q, %v; want the plaintext back", opened, err)
	}
}

func TestNilCipherRefusesSealedValue(t *testing.T) {
	sealed, err := testCipher(t, 6).Encrypt("token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	var c *Cipher
	if _, err := c.Decrypt(sealed); !errors.Is(err, ErrKeyUnavailable) {
		t.Fatalf("Decrypt without a key error = %v, want ErrKeyUnavailable", err)
	}
}

func TestNewCipherFromBase64(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i * 7)
	}

	c, err := NewCipherFromBase64(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatalf("padded base64 key: %v", err)
	}
	if !c.Configured() {
		t.Fatal("cipher from a valid key is not configured")
	}
	if _, err := NewCipherFromBase64(base64.RawStdEncoding.EncodeToString(key)); err != nil {
		t.Fatalf("raw base64 key: %v", err)
	}
	if _, err := NewCipherFromBase64("  " + base64.StdEncoding.EncodeToString(key) + "\n"); err != nil {
		t.Fatalf("key with surrounding whitespace: %v", err)
	}

	for _, bad := range []string{"", "   ", "not-base64!!", base64.StdEncoding.EncodeToString(key[:16])} {
		if _, err := NewCipherFromBase64(bad); !errors.Is(err, ErrInvalidKey) {
			t.Fatalf("NewCipherFromBase64(%q) error = %v, want ErrInvalidKey", bad, err)
		}
	}
	if _, err := NewCipher(key[:31]); !errors.Is(err, ErrInvalidKey) {
		t.Fatalf("NewCipher(31 bytes) error = %v, want ErrInvalidKey", err)
	}
}

func TestIsProtectedKey(t *testing.T) {
	if !IsProtectedKey(ZaloCredentialsKey) {
		t.Fatalf("%q is not protected", ZaloCredentialsKey)
	}
	if IsProtectedKey("zalo.enabled") || IsProtectedKey("ZALO.CREDENTIALS") {
		t.Fatal("unprotected key reported as protected")
	}
}

// TestProtectedKeysMatchZaloconnectCredentials pins the protected-key literal to
// the constant the Zalo service actually writes: a rename there would silently
// stop sealing the credential.
func TestProtectedKeysMatchZaloconnectCredentials(t *testing.T) {
	if ZaloCredentialsKey != zaloconnect.KeyCredentials {
		t.Fatalf("secret.ZaloCredentialsKey = %q, zaloconnect.KeyCredentials = %q", ZaloCredentialsKey, zaloconnect.KeyCredentials)
	}
}

func TestDefaultResolvesOnceAndWarnsWhenKeyMissing(t *testing.T) {
	logged := 0
	originalWarn := logWarn
	logWarn = func(string, ...any) { logged++ }
	t.Cleanup(func() { logWarn = originalWarn })

	t.Setenv(EnvKey, "")
	resetDefaultForTest()
	t.Cleanup(resetDefaultForTest)

	if c := Default(); c != nil {
		t.Fatal("Default() returned a cipher without a configured key")
	}
	if c := Default(); c != nil {
		t.Fatal("Default() returned a cipher without a configured key")
	}
	if logged != 1 {
		t.Fatalf("degraded-mode warning logged %d times, want exactly 1", logged)
	}
}

func TestDefaultLoadsKeyFromEnv(t *testing.T) {
	originalInfo := logInfo
	logInfo = func(string, ...any) {}
	t.Cleanup(func() { logInfo = originalInfo })

	key := make([]byte, KeySize)
	key[0] = 99
	t.Setenv(EnvKey, base64.StdEncoding.EncodeToString(key))
	resetDefaultForTest()
	t.Cleanup(resetDefaultForTest)

	c := Default()
	if c == nil {
		t.Fatal("Default() returned nil with a valid key in the environment")
	}
	sealed, err := c.Encrypt("token")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if opened, err := Default().Decrypt(sealed); err != nil || opened != "token" {
		t.Fatalf("Decrypt via Default() = %q, %v", opened, err)
	}
}

func TestDefaultIgnoresMalformedKey(t *testing.T) {
	logged := 0
	originalWarn := logWarn
	logWarn = func(string, ...any) { logged++ }
	t.Cleanup(func() { logWarn = originalWarn })

	t.Setenv(EnvKey, "tooshort")
	resetDefaultForTest()
	t.Cleanup(resetDefaultForTest)

	if c := Default(); c != nil {
		t.Fatal("Default() accepted a malformed key")
	}
	if logged != 1 {
		t.Fatalf("malformed-key warning logged %d times, want exactly 1", logged)
	}
}

func TestSetDefaultOverridesEnvironment(t *testing.T) {
	t.Setenv(EnvKey, "")
	resetDefaultForTest()
	t.Cleanup(resetDefaultForTest)

	c := testCipher(t, 9)
	SetDefault(c)
	if Default() != c {
		t.Fatal("SetDefault did not install the supplied cipher")
	}
	SetDefault(nil)
	if Default() != nil {
		t.Fatal("SetDefault(nil) did not disable encryption")
	}
}
