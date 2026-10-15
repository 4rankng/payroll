// Package opsapikey mints, authenticates, toggles, and revokes machine API
// keys for the payroll-ops-mcp agent surface (pk_ops_* namespace).
//
// Keys are 256-bit random values shown once at creation; only their SHA-256
// hash is stored. Each key carries a domain.OpsAPIKeyScope (read_only |
// read_write) plus an optional write-window bound: a read_write grant lapses
// back to read_only at window expiry, evaluated per request so an admin flip
// or an expired window blocks in-flight work within one request cycle.
//
// Isolation from the chat-integration keys (mcp_*) is absolute: different
// table, different prefix, different scope vocabulary, and the two services
// never authenticate each other's keys.
//
// Every authenticated call is audit-logged (one row per call: key id, agent
// identity, scope-at-time, tool, payload hash) onto the existing
// audit:log:write pipeline.
package opsapikey

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/pkg/clock"
)

// KeyPrefix is the plaintext namespace for ops keys. Keeping it as a literal
// makes a leaked key identifiable in logs and secret scanners, and lets the
// auth middleware route the credential to the right surface.
const KeyPrefix = "pk_ops_"

// prefixLen is how many leading plaintext characters are stored for display.
const prefixLen = 12

// DefaultWriteWindowHours is the default time-box for a read_write grant.
const DefaultWriteWindowHours = 8

// Audit action names (audit_logs.action).
const (
	ActionKeyCreated     = "OPS_KEY_CREATED"
	ActionKeyRevoked     = "OPS_KEY_REVOKED"
	ActionKeyScopeChange = "OPS_KEY_SCOPE_CHANGED"
	ActionCall           = "OPS_MCP_CALL"
	ActionScopeProbe     = "OPS_MCP_SCOPE_PROBE"
)

// entityType is the audit_logs.entity_type for every ops-key row.
const entityType = "ops_api_key"

// ErrInvalidOpsAPIKey is returned for an unknown, malformed, revoked, or
// orphaned key. All failure modes are indistinguishable on purpose.
var ErrInvalidOpsAPIKey = errors.New("opsapikey: invalid or revoked key")

// Service manages the ops API key lifecycle and call auditing.
type Service struct {
	repo  domain.OpsAPIKeyRepository
	users domain.UserRepository
	audit asynqinfra.AuditEnqueuer
	clk   clock.Clock
}

// NewService constructs the service. audit may be nil (auditing is then a
// no-op — unit tests); clk defaults to the real clock.
func NewService(repo domain.OpsAPIKeyRepository, users domain.UserRepository, audit asynqinfra.AuditEnqueuer, clk clock.Clock) *Service {
	if clk == nil {
		clk = clock.New()
	}
	return &Service{repo: repo, users: users, audit: audit, clk: clk}
}

// Create mints a new key and returns the record plus the plaintext key — the
// only time the plaintext exists outside the caller's memory. scope defaults
// to read_only when empty. A read_write grant carries writeWindowHours
// (DefaultWriteWindowHours when nil).
func (s *Service) Create(ctx context.Context, name string, scope domain.OpsAPIKeyScope, createdBy uint, writeWindowHours *int) (*domain.OpsAPIKey, string, error) {
	if scope == "" {
		scope = domain.OpsAPIKeyScopeReadOnly
	}
	if !domain.IsValidOpsAPIKeyScope(scope) {
		return nil, "", domain.NewValidationError("Phạm vi khoá API không hợp lệ")
	}
	plaintext, err := newPlaintextKey()
	if err != nil {
		return nil, "", fmt.Errorf("opsapikey: generate key: %w", err)
	}

	key := &domain.OpsAPIKey{
		Name:      name,
		Scope:     scope,
		KeyPrefix: plaintext[:prefixLen],
		KeyHash:   apikey.HashKey(plaintext),
		CreatedBy: createdBy,
	}
	if scope == domain.OpsAPIKeyScopeReadWrite {
		hours := DefaultWriteWindowHours
		if writeWindowHours != nil && *writeWindowHours > 0 {
			hours = *writeWindowHours
		}
		exp := s.clk.Now().Add(time.Duration(hours) * time.Hour)
		key.WriteWindowExpiresAt = &exp
	}
	if err := key.ValidateName(); err != nil {
		return nil, "", err
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return nil, "", err
	}
	s.enqueueAudit(ctx, key, ActionKeyCreated, createdBy,
		fmt.Sprintf("scope=ops tạo khoá agent=%s scope_at_time=%s", key.Name, key.Scope))
	return key, plaintext, nil
}

// Authenticate resolves a plaintext key to its record and creating user.
// Unknown, malformed, revoked, or orphaned/deactivated-creator keys all
// return ErrInvalidOpsAPIKey with no distinguishing detail.
func (s *Service) Authenticate(ctx context.Context, plaintext string) (*domain.OpsAPIKey, *domain.User, error) {
	if plaintext == "" {
		return nil, nil, ErrInvalidOpsAPIKey
	}
	key, err := s.repo.GetByHash(ctx, apikey.HashKey(plaintext))
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, nil, ErrInvalidOpsAPIKey
		}
		return nil, nil, err
	}
	if key.IsRevoked() {
		return nil, nil, ErrInvalidOpsAPIKey
	}
	user, err := s.users.GetByID(ctx, key.CreatedBy)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, nil, ErrInvalidOpsAPIKey
		}
		return nil, nil, err
	}
	if user == nil || !user.IsActive() {
		return nil, nil, ErrInvalidOpsAPIKey
	}
	return key, user, nil
}

// List returns every key, newest first.
func (s *Service) List(ctx context.Context) ([]*domain.OpsAPIKey, error) {
	return s.repo.List(ctx)
}

// GetByID loads one key by id (admin console + key introspection).
func (s *Service) GetByID(ctx context.Context, id uint) (*domain.OpsAPIKey, error) {
	return s.repo.GetByID(ctx, id)
}

// Revoke marks a key revoked. An unknown or already-revoked id is not-found.
func (s *Service) Revoke(ctx context.Context, id uint, actorID uint) error {
	if err := s.repo.Revoke(ctx, id, s.clk.Now()); err != nil {
		return err
	}
	if key, err := s.repo.GetByID(ctx, id); err == nil {
		s.enqueueAudit(ctx, key, ActionKeyRevoked, actorID, fmt.Sprintf("scope=ops thu hồi khoá agent=%s", key.Name))
	}
	return nil
}

// SetScope applies an admin scope toggle. A read_write grant opens (or
// renews) the write window; read_only clears it. Enforcement is per request,
// so the new scope applies to the very next call.
func (s *Service) SetScope(ctx context.Context, id uint, scope domain.OpsAPIKeyScope, writeWindowHours *int, actorID uint) (*domain.OpsAPIKey, error) {
	if !domain.IsValidOpsAPIKeyScope(scope) {
		return nil, domain.NewValidationError("Phạm vi khoá API không hợp lệ")
	}
	var window *time.Time
	if scope == domain.OpsAPIKeyScopeReadWrite {
		hours := DefaultWriteWindowHours
		if writeWindowHours != nil && *writeWindowHours > 0 {
			hours = *writeWindowHours
		}
		exp := s.clk.Now().Add(time.Duration(hours) * time.Hour)
		window = &exp
	}
	if err := s.repo.UpdateScope(ctx, id, scope, window); err != nil {
		return nil, err
	}
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.enqueueAudit(ctx, key, ActionKeyScopeChange, actorID,
		fmt.Sprintf("scope=ops đổi phạm vi agent=%s scope_at_time=%s", key.Name, scope))
	return key, nil
}

// RevertExpiredWriteWindows converges lapsed read_write grants to read_only.
// Called best-effort from the authentication path; evaluation in
// EffectiveScope is authoritative either way.
func (s *Service) RevertExpiredWriteWindows(ctx context.Context) {
	if _, err := s.repo.RevertExpiredWriteWindows(ctx, s.clk.Now()); err != nil {
		slog.Default().Warn("opsapikey: revert expired write windows failed", "error", err)
	}
}

// TouchLastUsed stamps last_used_at. Called best-effort from the middleware.
func (s *Service) TouchLastUsed(ctx context.Context, id uint) error {
	return s.repo.UpdateLastUsedAt(ctx, id, s.clk.Now())
}

// CallAudit captures one authenticated machine call for the audit trail.
type CallAudit struct {
	KeyID       uint
	AgentName   string
	CreatorID   uint
	ScopeAtTime domain.OpsAPIKeyScope
	Method      string
	Path        string
	Tool        string // X-MCP-Tool header when the caller is the ops MCP server
	PayloadHash string // "sha256:<hex-prefix>" of the request body ("" when none)
	IPAddress   string
	UserAgent   string
}

// LogCall writes exactly one audit row per authenticated ops call.
func (s *Service) LogCall(ctx context.Context, e CallAudit) {
	msg := fmt.Sprintf("scope=ops key=%d agent=%s scope_at_time=%s %s %s",
		e.KeyID, e.AgentName, e.ScopeAtTime, e.Method, e.Path)
	if e.Tool != "" {
		msg += " tool=" + e.Tool
	}
	if e.PayloadHash != "" {
		msg += " payload=" + e.PayloadHash
	}
	s.enqueueAudit(ctx, &domain.OpsAPIKey{ID: e.KeyID, Name: e.AgentName}, ActionCall, e.CreatorID, msg)
}

// LogScopeProbe records a wrong-scope attempt: a pk_ops_* key presented on a
// surface it must never reach (scope-probing signal).
func (s *Service) LogScopeProbe(ctx context.Context, plaintextPrefix, method, path, ip, userAgent string) {
	s.enqueueAudit(ctx, &domain.OpsAPIKey{}, ActionScopeProbe, 0,
		fmt.Sprintf("scope=ops key_prefix=%s wrong-surface probe %s %s", plaintextPrefix, method, path))
}

// enqueueAudit publishes one row onto the audit:log:write pipeline. Failures
// are logged, never propagated — auditing must not block the request path.
func (s *Service) enqueueAudit(ctx context.Context, key *domain.OpsAPIKey, action string, actorID uint, message string) {
	if s.audit == nil {
		return
	}
	entityID := key.ID
	err := s.audit.EnqueueAuditLogWrite(asynqinfra.AuditLogWritePayload{
		UserID:     actorID,
		Action:     action,
		EntityType: entityType,
		EntityID:   &entityID,
		Message:    message,
		CreatedAt:  s.clk.Now(),
	})
	if err != nil {
		slog.Default().Warn("opsapikey: enqueue audit failed", "error", err, "action", action, "ops_key_id", key.ID)
	}
}

// newPlaintextKey returns "pk_ops_" + 32 random bytes base64url-encoded.
func newPlaintextKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return KeyPrefix + base64.RawURLEncoding.EncodeToString(raw), nil
}
