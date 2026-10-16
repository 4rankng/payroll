// Package mcpapikey mints, authenticates, toggles, and revokes machine API
// keys for the MCP agent surface (/api/v1/mcp). Keys are 256-bit random
// values shown once at creation; only their SHA-256 hash is stored. A plain
// SHA-256 (rather than a password-style KDF) is sufficient because the key is
// high-entropy random, not a human-chosen secret.
//
// Each key carries a domain.MCPAPIKeyScope (read_only | full) and executes with
// the admin rights of the creator that minted it — the creator is resolved at
// authentication time, so deleting or deactivating the creator disables the
// key. The scope is re-read from the database on every request, so an admin
// toggle (read_only <-> full) applies to the very next call with no cache to
// invalidate; every toggle is audit-logged with actor and old/new scope.
package mcpapikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/pkg/clock"
)

// keyPrefix is the plaintext namespace for MCP API keys. Keeping it as a
// literal makes a leaked key identifiable in logs and secret scanners.
const keyPrefix = "mcp_"

// prefixLen is how many leading plaintext characters are stored for display.
const prefixLen = 12

// Audit action names (audit_logs.action).
const (
	ActionKeyScopeChange = "MCP_KEY_SCOPE_CHANGED"
)

// entityType is the audit_logs.entity_type for every mcp-key row.
const entityType = "mcp_api_key"

// ErrInvalidMCPAPIKey is returned for an unknown, malformed, revoked, or
// orphaned key. All failure modes are indistinguishable on purpose.
var ErrInvalidMCPAPIKey = errors.New("mcpapikey: invalid or revoked key")

// Service manages the MCP API key lifecycle.
type Service struct {
	repo  domain.MCPAPIKeyRepository
	users domain.UserRepository
	audit asynqinfra.AuditEnqueuer
	clk   clock.Clock
}

// NewService constructs the service. audit may be nil (auditing is then a
// no-op — unit tests); clk defaults to the real clock.
func NewService(repo domain.MCPAPIKeyRepository, users domain.UserRepository, audit asynqinfra.AuditEnqueuer, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.New()
	}
	return &Service{repo: repo, users: users, audit: audit, clk: clk}
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

// SetScope applies an admin scope toggle on a live key (read_only <-> full —
// the closed MCP scope enum; anything else is a validation error, so
// revocation can never sneak through this path). The change is audit-logged
// with the actor and the old/new scope. Because the transport gate re-reads
// the scope from the database on every request, the new scope applies to the
// very next call — there is no cache to invalidate.
func (s *Service) SetScope(ctx context.Context, id uint, scope domain.MCPAPIKeyScope, actorID uint) (*domain.MCPAPIKey, error) {
	if !domain.IsValidMCPAPIKeyScope(scope) {
		return nil, domain.NewValidationError("Phạm vi khoá API MCP không hợp lệ. Chỉ hỗ trợ: read_only, full")
	}
	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if current.IsRevoked() {
		// A revoked key is terminal: its scope must not move anymore.
		return nil, domain.NewNotFoundError("mcp api key not found")
	}
	now := s.clk.Now()
	if err := s.repo.UpdateScope(ctx, id, scope, now); err != nil {
		return nil, err
	}
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enqueueAudit(ctx, key, actorID, current.Scope, key.Scope)
	return key, nil
}

// enqueueAudit publishes one scope-change row onto the audit:log:write
// pipeline (actor + old/new scope). Failures are logged, never propagated —
// auditing must not block the request path.
func (s *Service) enqueueAudit(ctx context.Context, key *domain.MCPAPIKey, actorID uint, oldScope, newScope domain.MCPAPIKeyScope) {
	if s.audit == nil {
		return
	}
	entityID := key.ID
	err := s.audit.EnqueueAuditLogWrite(asynqinfra.AuditLogWritePayload{
		UserID:     actorID,
		Action:     ActionKeyScopeChange,
		EntityType: entityType,
		EntityID:   &entityID,
		Message: fmt.Sprintf("scope=mcp đổi phạm vi key=%d name=%s old_scope=%s new_scope=%s",
			key.ID, key.Name, oldScope, newScope),
		CreatedAt: s.clk.Now(),
	})
	if err != nil {
		slog.Default().Warn("mcpapikey: enqueue audit failed", "error", err, "action", ActionKeyScopeChange, "mcp_key_id", key.ID)
	}
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
