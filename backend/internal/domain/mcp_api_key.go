package domain

import (
	"context"
	"strings"
	"time"
)

// MCPAPIKeyScope bounds what an MCP machine credential may do.
//
// read_only is the GET/HEAD breadth gate; full is every endpoint including
// mutations. The vocabulary is deliberately separate from APIKeyScope: MCP
// keys and vendor integration keys are different credential types and must
// never authenticate each other's surfaces.
type MCPAPIKeyScope string

const (
	// MCPAPIKeyScopeReadOnly permits only read verbs (GET/HEAD) across the
	// authenticated API.
	MCPAPIKeyScopeReadOnly MCPAPIKeyScope = "read_only"
	// MCPAPIKeyScopeFull permits every endpoint, including mutations.
	MCPAPIKeyScopeFull MCPAPIKeyScope = "full"
)

// IsValidMCPAPIKeyScope reports whether s is one of the known scopes.
func IsValidMCPAPIKeyScope(s MCPAPIKeyScope) bool {
	return s == MCPAPIKeyScopeReadOnly || s == MCPAPIKeyScopeFull
}

// MCPAPIKey is a machine credential for the MCP agent surface
// (/api/v1/mcp). Only the SHA-256 hash of the plaintext key is persisted; the
// plaintext is shown to the admin exactly once at creation and is
// unrecoverable afterward. KeyPrefix stores the first 12 characters of the
// plaintext so the admin list can tell keys apart without ever holding the
// secret.
type MCPAPIKey struct {
	ID         uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Name       string         `json:"name" gorm:"type:varchar(100);not null"`
	Scope      MCPAPIKeyScope `json:"scope" gorm:"type:varchar(16);not null"`
	KeyPrefix  string         `json:"key_prefix" gorm:"type:varchar(20);not null"`
	KeyHash    string         `json:"-" gorm:"type:char(64);not null;uniqueIndex:uq_mcp_api_keys_key_hash"`
	CreatedBy  uint           `json:"created_by" gorm:"type:bigint unsigned;not null"`
	LastUsedAt *time.Time     `json:"last_used_at" gorm:"type:datetime(3)"`
	RevokedAt  *time.Time     `json:"revoked_at" gorm:"type:datetime(3);index:idx_mcp_api_keys_revoked_at"`
	// ScopeUpdatedAt records the last admin scope toggle (read_only <->
	// full). NULL means the key still carries its creation-time scope.
	ScopeUpdatedAt *time.Time `json:"scope_updated_at" gorm:"type:datetime(3)"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (MCPAPIKey) TableName() string { return "mcp_api_keys" }

// IsRevoked reports whether the key has been revoked and must no longer
// authenticate.
func (k *MCPAPIKey) IsRevoked() bool { return k.RevokedAt != nil }

// IsReadOnly reports whether the key carries the READ ONLY breadth.
func (k *MCPAPIKey) IsReadOnly() bool { return k.Scope == MCPAPIKeyScopeReadOnly }

// ValidateName enforces the admin-supplied label rules (same bound as vendor
// integration keys).
func (k *MCPAPIKey) ValidateName() error {
	name := strings.TrimSpace(k.Name)
	if name == "" {
		return NewValidationError("Tên khoá API không được để trống")
	}
	if len([]rune(name)) > APIKeyNameMaxLen {
		return NewValidationError("Tên khoá API không được vượt quá 100 ký tự")
	}
	return nil
}

// MCPAPIKeyRepository defines persistence for MCP machine API keys.
type MCPAPIKeyRepository interface {
	Create(ctx context.Context, key *MCPAPIKey) error
	GetByHash(ctx context.Context, keyHash string) (*MCPAPIKey, error)
	GetByID(ctx context.Context, id uint) (*MCPAPIKey, error)
	List(ctx context.Context) ([]*MCPAPIKey, error)
	Revoke(ctx context.Context, id uint, revokedAt time.Time) error
	UpdateLastUsedAt(ctx context.Context, id uint, at time.Time) error
	// UpdateScope persists an admin scope toggle (read_only <-> full) on a
	// live key together with the change timestamp. A revoked key affects
	// zero rows and is reported as not-found.
	UpdateScope(ctx context.Context, id uint, scope MCPAPIKeyScope, updatedAt time.Time) error
}
