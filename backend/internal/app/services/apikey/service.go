// Package apikey mints, authenticates, and revokes machine API keys for the
// external chatbot integration API. Keys are 256-bit random values shown once
// at creation; only their SHA-256 hash is stored. A plain SHA-256 (rather than
// a password-style KDF) is sufficient because the key is high-entropy random,
// not a human-chosen secret.
package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// keyPrefix is the plaintext namespace for integration API keys. Keeping it as
// a literal makes a leaked key identifiable in logs and secret scanners.
const keyPrefix = "ttk_"

// prefixLen is how many leading plaintext characters are stored for display.
const prefixLen = 12

// ErrInvalidAPIKey is returned for an unknown, malformed, or revoked key.
var ErrInvalidAPIKey = errors.New("apikey: invalid or revoked key")

// Service manages the API key lifecycle.
type Service struct {
	repo domain.APIKeyRepository
	clk  clock.Clock
}

// NewService constructs the service.
func NewService(repo domain.APIKeyRepository, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.New()
	}
	return &Service{repo: repo, clk: clk}
}

// Create mints a new key and returns the record plus the plaintext key — the
// only time the plaintext exists outside the caller's memory.
func (s *Service) Create(ctx context.Context, name string, createdBy uint) (*domain.APIKey, string, error) {
	plaintext, err := newPlaintextKey()
	if err != nil {
		return nil, "", fmt.Errorf("apikey: generate key: %w", err)
	}

	key := &domain.APIKey{
		Name:      name,
		KeyPrefix: plaintext[:prefixLen],
		KeyHash:   HashKey(plaintext),
		CreatedBy: createdBy,
	}
	if err := key.ValidateName(); err != nil {
		return nil, "", err
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return nil, "", err
	}
	return key, plaintext, nil
}

// Authenticate resolves a plaintext key to its record. Unknown, malformed, or
// revoked keys all return ErrInvalidAPIKey with no distinguishing detail.
func (s *Service) Authenticate(ctx context.Context, plaintext string) (*domain.APIKey, error) {
	if plaintext == "" {
		return nil, ErrInvalidAPIKey
	}
	key, err := s.repo.GetByHash(ctx, HashKey(plaintext))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}
	if key.IsRevoked() {
		return nil, ErrInvalidAPIKey
	}
	return key, nil
}

// List returns every key, newest first.
func (s *Service) List(ctx context.Context) ([]*domain.APIKey, error) {
	return s.repo.List(ctx)
}

// Revoke marks a key revoked. An unknown or already-revoked id is not-found.
func (s *Service) Revoke(ctx context.Context, id uint) error {
	return s.repo.Revoke(ctx, id, s.clk.Now())
}

// TouchLastUsed stamps last_used_at. Called best-effort from the middleware.
func (s *Service) TouchLastUsed(ctx context.Context, id uint) error {
	return s.repo.UpdateLastUsedAt(ctx, id, s.clk.Now())
}

// newPlaintextKey returns "ttk_" + 32 random bytes base64url-encoded (46 chars).
func newPlaintextKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return keyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}

// HashKey returns the hex SHA-256 of a plaintext key. Exported so the
// authenticating middleware and tests share one definition.
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}
