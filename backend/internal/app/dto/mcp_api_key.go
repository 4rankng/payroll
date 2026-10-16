package dto

import (
	"time"

	"api-server/internal/domain"
)

// CreateMCPAPIKeyRequest is the body for POST /admin/mcp-api-keys.
type CreateMCPAPIKeyRequest struct {
	Name  string                `json:"name" binding:"required,max=100"`
	Scope domain.MCPAPIKeyScope `json:"scope" binding:"required"`
}

// SetMCPAPIKeyScopeRequest is the body for PUT /admin/mcp-api-keys/:id/scope.
// Only the closed MCP scope vocabulary is accepted (read_only | full) —
// revocation stays on its own endpoint.
type SetMCPAPIKeyScopeRequest struct {
	Scope domain.MCPAPIKeyScope `json:"scope" binding:"required"`
}

// MCPAPIKeyResponse is the admin view of an MCP API key. Key carries the
// plaintext and is populated ONLY in the create response — never on list.
// ScopeUpdatedAt is the last admin scope toggle (null = never toggled).
type MCPAPIKeyResponse struct {
	ID             uint                  `json:"id"`
	Name           string                `json:"name"`
	Scope          domain.MCPAPIKeyScope `json:"scope"`
	Key            string                `json:"key,omitempty"`
	KeyPrefix      string                `json:"key_prefix"`
	CreatedBy      uint                  `json:"created_by"`
	LastUsedAt     *time.Time            `json:"last_used_at"`
	RevokedAt      *time.Time            `json:"revoked_at"`
	ScopeUpdatedAt *time.Time            `json:"scope_updated_at"`
	CreatedAt      time.Time             `json:"created_at"`
}
