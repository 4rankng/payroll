package dto

import (
	"time"

	"api-server/internal/domain"
)

// CreateOpsAPIKeyRequest is the body for POST /admin/ops-api-keys. Scope
// defaults to read_only; a read_write grant opens the (optional) write
// window, DefaultWriteWindowHours when omitted.
type CreateOpsAPIKeyRequest struct {
	Name             string                `json:"name" binding:"required,max=100"`
	Scope            domain.OpsAPIKeyScope `json:"scope"`
	WriteWindowHours *int                  `json:"write_window_hours,omitempty" binding:"omitempty,min=1,max=168"`
}

// SetOpsAPIKeyScopeRequest is the body for PUT /admin/ops-api-keys/:id/scope.
type SetOpsAPIKeyScopeRequest struct {
	Scope            domain.OpsAPIKeyScope `json:"scope" binding:"required"`
	WriteWindowHours *int                  `json:"write_window_hours,omitempty" binding:"omitempty,min=1,max=168"`
}

// OpsAPIKeyResponse is the admin view of an ops key. Key carries the
// plaintext ONLY in the create response — never on list. EffectiveScope is
// the scope at read time (a lapsed write window reads as read_only).
type OpsAPIKeyResponse struct {
	ID                   uint                  `json:"id"`
	Name                 string                `json:"name"`
	Scope                domain.OpsAPIKeyScope `json:"scope"`
	EffectiveScope       domain.OpsAPIKeyScope `json:"effective_scope"`
	WriteWindowExpiresAt *time.Time            `json:"write_window_expires_at"`
	Key                  string                `json:"key,omitempty"`
	KeyPrefix            string                `json:"key_prefix"`
	CreatedBy            uint                  `json:"created_by"`
	LastUsedAt           *time.Time            `json:"last_used_at"`
	RevokedAt            *time.Time            `json:"revoked_at"`
	CreatedAt            time.Time             `json:"created_at"`
}

// OpsKeySelfInfo is the /api/v1/ops/key-info view: what the ops MCP server
// reads at startup and per call to decide which tools to expose. It never
// includes key material.
type OpsKeySelfInfo struct {
	KeyID          uint                  `json:"key_id"`
	AgentName      string                `json:"agent_name"`
	Scope          domain.OpsAPIKeyScope `json:"scope"`
	EffectiveScope domain.OpsAPIKeyScope `json:"effective_scope"`
}
