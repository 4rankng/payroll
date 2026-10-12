// Package mcpapikey mints, authenticates, and revokes machine API keys for the
// MCP agent surface (/api/v1/mcp). Keys are 256-bit random values shown once at
// creation; only their SHA-256 hash is stored. A plain SHA-256 (rather than a
// password-style KDF) is sufficient because the key is high-entropy random, not
// a human-chosen secret.
//
// Each key carries a domain.MCPAPIKeyScope (read_only | full) and executes with
// the admin rights of the creator that minted it — the creator is resolved at
// authentication time, so deleting or deactivating the creator disables the
// key.
package mcpapikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// keyPrefix is the plaintext namespace for MCP API keys. Keeping it as a
// literal makes a leaked key identifiable in logs and secret scanners.
const keyPrefix = "mcp_"

// prefixLen is how many leading plaintext characters are stored for display.
const prefixLen = 12

// ErrInvalidMCPAPIKey is returned for an unknown, malformed, revoked, or
// orphaned key. All failure modes are indistinguishable on purpose.
var ErrInvalidMCPAPIKey = errors.New("mcpapikey: invalid or revoked key")

// Service manages the MCP API key lifecycle.
type Service struct {
	repo  domain.MCPAPIKeyRepository
	users domain.UserRepository
	clk   clock.Clock
}

// NewService constructs the service.
func NewService(repo domain.MCPAPIKeyRepository, users domain.UserRepository, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.New()
	}
	return &Service{repo: repo, users: users, clk: clk}
}

// Create mints a new key and returns the record plus the plaintext key — the
// only time the plaintext exists outside the caller's memory.
func (s *Service) Create(ctx context.Context, name string, scope domain.MCPAPIKeyScope, createdBy uint) (*domain.MCPAPIKey, string, error) {
	if !domain.IsValidMCPAPIKeyScope(scope) {
		return nil, "", domain.NewValidationError("Phạm vi khoá API không hợp lệ")
	}
	plaintext, err := newPlaintextKey()
	if err != nil {
		return nil, "", fmt.Errorf("mcpapikey: generate key: %w", err)
	}

	key := &domain.MCPAPIKey{
		Name:      name,
		Scope:     scope,
		KeyPrefix: plaintext[:prefixLen],
		KeyHash:   apikey.HashKey(plaintext),
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

// Authenticate resolves a plaintext key to its record and creating user.
// Unknown, malformed, revoked, or orphaned/deactivated-creator keys all return
// ErrInvalidMCPAPIKey with no distinguishing detail.
func (s *Service) Authenticate(ctx context.Context, plaintext string) (*domain.MCPAPIKey, *domain.User, error) {
	if plaintext == "" {
		return nil, nil, ErrInvalidMCPAPIKey
	}
	key, err := s.repo.GetByHash(ctx, apikey.HashKey(plaintext))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, nil, ErrInvalidMCPAPIKey
		}
		return nil, nil, err
	}
	if key.IsRevoked() {
		return nil, nil, ErrInvalidMCPAPIKey
	}
	user, err := s.users.GetByID(ctx, key.CreatedBy)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, nil, ErrInvalidMCPAPIKey
		}
		return nil, nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, nil, ErrInvalidMCPAPIKey
	}
	return key, user, nil
}

// List returns every key, newest first.
func (s *Service) List(ctx context.Context) ([]*domain.MCPAPIKey, error) {
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

// newPlaintextKey returns "mcp_" + 32 random bytes base64url-encoded.
func newPlaintextKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return keyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}
