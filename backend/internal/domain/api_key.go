package domain

import (
	"context"
	"strings"
	"time"
)

// APIKeyNameMaxLen bounds the admin-supplied label.
const APIKeyNameMaxLen = 100

// APIKeyScope bounds what a machine credential may authenticate.
//
// It is a closed string enum rather than a bitmask: exactly two call sites
// compare it, and a typo must fail closed (authenticate nothing) rather than
// silently widen a grant. Adding a scope here is a deliberate security change
// and must be mirrored by a RequireAPIKeyScope guard on the routes it covers.
type APIKeyScope string

const (
	// APIKeyScopeTimesheetRead is the vendor read-only timesheet surface:
	// listing projects, listing employees, and querying timesheets. It covers
	// no mutating route anywhere in the system.
	APIKeyScopeTimesheetRead APIKeyScope = "timesheet_read"
)

// ValidAPIKeyScopes lists every accepted scope value, for admin validation.
var ValidAPIKeyScopes = []APIKeyScope{APIKeyScopeTimesheetRead}

// IsValidAPIKeyScope reports whether s is one of the known scopes.
func IsValidAPIKeyScope(s APIKeyScope) bool {
	for _, v := range ValidAPIKeyScopes {
		if s == v {
			return true
		}
	}
	return false
}

// APIKey is a machine credential for an external integration API.
// Only the SHA-256 hash of the plaintext key is persisted; the plaintext is
// shown to the admin exactly once at creation and is unrecoverable afterward.
// KeyPrefix stores the first 12 characters of the plaintext so the admin list
// can tell keys apart without ever holding the secret.
type APIKey struct {
	ID         uint        `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Name       string      `json:"name" gorm:"type:varchar(100);not null"`
	Scope      APIKeyScope `json:"scope" gorm:"type:varchar(32);not null"`
	KeyPrefix  string      `json:"key_prefix" gorm:"type:varchar(20);not null"`
	KeyHash    string      `json:"-" gorm:"type:char(64);not null;uniqueIndex"`
	CreatedBy  uint        `json:"created_by" gorm:"type:bigint unsigned;not null"`
	LastUsedAt *time.Time  `json:"last_used_at" gorm:"type:datetime(3)"`
	RevokedAt  *time.Time  `json:"revoked_at" gorm:"type:datetime(3);index"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

func (APIKey) TableName() string { return "api_keys" }

// IsRevoked reports whether the key has been revoked and must no longer
// authenticate.
func (k *APIKey) IsRevoked() bool { return k.RevokedAt != nil }

// HasScope reports whether this key may authenticate routes guarded by
// required. An empty or unknown stored scope matches nothing — fail closed.
func (k *APIKey) HasScope(required APIKeyScope) bool {
	return k.Scope == required
}

// ValidateName enforces the admin-supplied label rules.
func (k *APIKey) ValidateName() error {
	name := strings.TrimSpace(k.Name)
	if name == "" {
		return NewValidationError("Tên khoá API không được để trống")
	}
	if len([]rune(name)) > APIKeyNameMaxLen {
		return NewValidationError("Tên khoá API không được vượt quá 100 ký tự")
	}
	return nil
}

// ValidateScope rejects a scope outside the closed enum. An empty value is
// rejected too: with no default scope, a key must be explicitly bound to a
// surface or it would silently match nothing.
func (k *APIKey) ValidateScope() error {
	if !IsValidAPIKeyScope(k.Scope) {
		return NewValidationError("Phạm vi khoá API không hợp lệ. Chỉ hỗ trợ: timesheet_read")
	}
	return nil
}

// APIKeyRepository defines persistence for machine API keys.
type APIKeyRepository interface {
	Create(ctx context.Context, key *APIKey) error
	GetByHash(ctx context.Context, keyHash string) (*APIKey, error)
	List(ctx context.Context) ([]*APIKey, error)
	Revoke(ctx context.Context, id uint, revokedAt time.Time) error
	UpdateLastUsedAt(ctx context.Context, id uint, at time.Time) error
}
