package domain

import (
	"context"
	"strings"
	"time"
)

// OpsAPIKeyScope bounds what an ops machine credential (pk_ops_*) may do.
//
// read_only is the default at creation and the state every key auto-reverts
// to when its write window lapses; read_write additionally permits mutation
// verbs at the transport gate. The Casbin agent_ops role keeps the mutation
// ceiling at "file an approval request" regardless of scope.
type OpsAPIKeyScope string

const (
	// OpsAPIKeyScopeReadOnly permits only read verbs (GET/HEAD).
	OpsAPIKeyScopeReadOnly OpsAPIKeyScope = "read_only"
	// OpsAPIKeyScopeReadWrite additionally permits mutation verbs, subject to
	// the agent_ops Casbin ceiling and the write window.
	OpsAPIKeyScopeReadWrite OpsAPIKeyScope = "read_write"
)

// IsValidOpsAPIKeyScope reports whether s is one of the known scopes.
func IsValidOpsAPIKeyScope(s OpsAPIKeyScope) bool {
	return s == OpsAPIKeyScopeReadOnly || s == OpsAPIKeyScopeReadWrite
}

// OpsAPIKey is a machine credential for the payroll-ops-mcp agent surface.
// One key per agent (e.g. pk_ops_hermes) so audit rows attribute every call
// to a specific agent and each key can be revoked individually. Only the
// SHA-256 hash of the plaintext is persisted; the plaintext is shown to the
// admin exactly once at creation. The pk_ops_* namespace is deliberately
// separate from the mcp_* chat-integration keys: the two credential types
// must never authenticate each other's surfaces.
type OpsAPIKey struct {
	ID         uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Name       string         `json:"name" gorm:"type:varchar(100);not null"`
	Scope      OpsAPIKeyScope `json:"scope" gorm:"type:varchar(16);not null"`
	KeyPrefix  string         `json:"key_prefix" gorm:"type:varchar(20);not null"`
	KeyHash    string         `json:"-" gorm:"type:char(64);not null;uniqueIndex:uq_ops_api_keys_key_hash"`
	CreatedBy  uint           `json:"created_by" gorm:"type:bigint unsigned;not null"`
	LastUsedAt *time.Time     `json:"last_used_at" gorm:"type:datetime(3)"`
	RevokedAt  *time.Time     `json:"revoked_at" gorm:"type:datetime(3);index:idx_ops_api_keys_revoked_at"`
	// WriteWindowExpiresAt bounds a read_write grant: past this instant the
	// key's effective scope is read_only again (lazily evaluated per request,
	// never cached at authentication).
	WriteWindowExpiresAt *time.Time `json:"write_window_expires_at" gorm:"type:datetime(3)"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (OpsAPIKey) TableName() string { return "ops_api_keys" }

// IsRevoked reports whether the key has been revoked and must no longer
// authenticate.
func (k *OpsAPIKey) IsRevoked() bool { return k.RevokedAt != nil }

// EffectiveScope resolves the scope at instant now: a read_write grant whose
// write window has lapsed behaves as read_only. This is the only way scope
// is read — the stored value is the admin's last decision, not the truth at
// request time (stale-scope red-team constraint).
func (k *OpsAPIKey) EffectiveScope(now time.Time) OpsAPIKeyScope {
	if k.Scope == OpsAPIKeyScopeReadWrite && k.WriteWindowExpiresAt != nil && now.After(*k.WriteWindowExpiresAt) {
		return OpsAPIKeyScopeReadOnly
	}
	return k.Scope
}

// WriteWindowExpired reports whether a read_write grant has lapsed.
func (k *OpsAPIKey) WriteWindowExpired(now time.Time) bool {
	return k.EffectiveScope(now) == OpsAPIKeyScopeReadOnly && k.Scope == OpsAPIKeyScopeReadWrite
}

// ValidateName enforces the admin-supplied label rules. The name doubles as
// the agent identity in audit rows.
func (k *OpsAPIKey) ValidateName() error {
	name := strings.TrimSpace(k.Name)
	if name == "" {
		return NewValidationError("Tên khoá API không được để trống")
	}
	if len([]rune(name)) > APIKeyNameMaxLen {
		return NewValidationError("Tên khoá API không được vượt quá 100 ký tự")
	}
	return nil
}

// OpsAPIKeyRepository defines persistence for ops machine API keys.
type OpsAPIKeyRepository interface {
	Create(ctx context.Context, key *OpsAPIKey) error
	GetByHash(ctx context.Context, keyHash string) (*OpsAPIKey, error)
	GetByID(ctx context.Context, id uint) (*OpsAPIKey, error)
	List(ctx context.Context) ([]*OpsAPIKey, error)
	Revoke(ctx context.Context, id uint, revokedAt time.Time) error
	UpdateLastUsedAt(ctx context.Context, id uint, at time.Time) error
	// UpdateScope persists an admin scope decision together with the write
	// window bound (nil for read_only).
	UpdateScope(ctx context.Context, id uint, scope OpsAPIKeyScope, writeWindowExpiresAt *time.Time) error
	// RevertExpiredWriteWindows flips every lapsed read_write key back to
	// read_only so the stored state converges with the evaluated truth.
	RevertExpiredWriteWindows(ctx context.Context, now time.Time) (int64, error)
}
