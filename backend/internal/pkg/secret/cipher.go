// Package secret seals the values of a small, explicitly enumerated set of
// integration settings so a database dump, backup or replica no longer exposes
// third-party credentials in the clear.
//
// The settings table is a generic key/value store; rows such as
// "zalo.credentials" hold a Zalo OA app secret and OAuth tokens verbatim. This
// package seals those values with AES-256-GCM under a key supplied by the
// SETTINGS_SECRET_KEY environment variable (base64, 32 bytes).
//
// The key is optional by design. When it is missing the package degrades to
// pass-through — values are read and written exactly as before — instead of
// failing startup: an unconfigured encryption key must not take the payroll API
// down. Ciphertext carries the "enc:v1:" prefix, so a sealed value is
// recognisable, a legacy plaintext value stays readable forever, and a future
// format change can be introduced without guessing.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	// EnvKey is the environment variable holding the base64-encoded key.
	EnvKey = "SETTINGS_SECRET_KEY"
	// KeySize is the required raw key length in bytes (AES-256).
	KeySize = 32
	// Prefix marks a sealed value. Plaintext values never carry it, and a
	// plaintext value that happened to start with it would still be readable
	// because IsEncrypted only trusts the full prefix.
	Prefix = "enc:v1:"
)

var (
	// ErrInvalidKey reports a key that is not 32 bytes once base64-decoded.
	ErrInvalidKey = errors.New("secret: key must be 32 base64-encoded bytes")
	// ErrDecrypt reports ciphertext that failed authentication (tampered,
	// truncated, or sealed under a different key).
	ErrDecrypt = errors.New("secret: ciphertext failed authentication")
	// ErrKeyUnavailable reports a sealed value read while no key is configured.
	// Reading ciphertext as if it were plaintext would leak the sealed form into
	// services and, worse, get re-written as a "plaintext" credential.
	ErrKeyUnavailable = errors.New("secret: " + EnvKey + " is not configured but a sealed value was read")
)

// Cipher seals and opens setting values. A nil *Cipher is valid and means "no
// key configured": every method becomes a pass-through, matching the behaviour
// the application had before encryption existed. Nil-receiver safety keeps the
// unconfigured case free of branching at every call site.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher builds a cipher from a raw 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("%w (got %d bytes)", ErrInvalidKey, len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: aes: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secret: gcm: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

// NewCipherFromBase64 builds a cipher from a base64-encoded 32-byte key, as
// read from SETTINGS_SECRET_KEY. Both padded and raw (unpadded) standard base64
// are accepted; surrounding whitespace is ignored.
func NewCipherFromBase64(encoded string) (*Cipher, error) {
	trimmed := strings.TrimSpace(encoded)
	if trimmed == "" {
		return nil, fmt.Errorf("%w (empty)", ErrInvalidKey)
	}
	raw, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		raw, err = base64.RawStdEncoding.DecodeString(trimmed)
	}
	if err != nil {
		return nil, fmt.Errorf("%w (not base64)", ErrInvalidKey)
	}
	return NewCipher(raw)
}

// Configured reports whether a key is present. Safe on a nil receiver.
func (c *Cipher) Configured() bool { return c != nil }

// IsEncrypted reports whether a stored value carries the sealed-value prefix.
func IsEncrypted(value string) bool { return strings.HasPrefix(value, Prefix) }

// Encrypt seals plaintext into the "enc:v1:<base64(nonce||ciphertext)>" form.
//
// Without a key (nil receiver) the plaintext is returned unchanged, which is the
// historical behaviour. Empty values and already-sealed values are returned
// as-is so the operation is idempotent.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if c == nil || plaintext == "" || IsEncrypted(plaintext) {
		return plaintext, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secret: nonce: %w", err)
	}
	// Seal appends to nonce, producing nonce||ciphertext in one buffer.
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return Prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a stored value. A value without the sealed prefix is returned
// unchanged: settings written before the key existed (and every unprotected
// key) keep working. A sealed value read without a configured key is an error
// rather than a silent pass-through of the ciphertext.
func (c *Cipher) Decrypt(value string) (string, error) {
	if !IsEncrypted(value) {
		return value, nil
	}
	if c == nil {
		return "", ErrKeyUnavailable
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(value, Prefix))
	if err != nil {
		return "", fmt.Errorf("%w: not base64", ErrDecrypt)
	}
	nonceSize := c.aead.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("%w: truncated payload", ErrDecrypt)
	}
	plaintext, err := c.aead.Open(nil, raw[:nonceSize], raw[nonceSize:], nil)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrDecrypt, err)
	}
	return string(plaintext), nil
}
