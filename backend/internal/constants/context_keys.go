package constants

// Context Keys Constants
const (
	CtxUserID   = "user_id"
	CtxUserRole = "user_role"

	// CtxAPIKeyID / CtxAPIKeyName carry the authenticated machine API key
	// identity for integration routes (no JWT user exists on those requests).
	CtxAPIKeyID   = "api_key_id"
	CtxAPIKeyName = "api_key_name"
	// CtxAPIKeyScope carries the authenticated key's domain.APIKeyScope as a
	// string. RequireAPIKeyScope reads it to refuse a valid key whose scope
	// does not cover the route it is calling.
	CtxAPIKeyScope = "api_key_scope"
	// CtxMCPKeyID carries the authenticated MCP API key id for requests that
	// arrive with an X-API-Key machine credential.
	CtxMCPKeyID = "mcp_key_id"
)

// System User ID - used for automated/system operations
const SystemUserID uint = 49

// AdminUserID is the primary admin user used for system-initiated operations
// that require an admin context (e.g., reconciliation, wallet sync).
const AdminUserID uint = 1
