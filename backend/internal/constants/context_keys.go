package constants

// Context Keys Constants
const (
	CtxUserID   = "user_id"
	CtxUserRole = "user_role"

	// CtxAPIKeyID / CtxAPIKeyName carry the authenticated machine API key
	// identity for integration routes (no JWT user exists on those requests).
	CtxAPIKeyID   = "api_key_id"
	CtxAPIKeyName = "api_key_name"
)

// System User ID - used for automated/system operations
const SystemUserID uint = 49

// AdminUserID is the primary admin user used for system-initiated operations
// that require an admin context (e.g., reconciliation, wallet sync).
const AdminUserID uint = 1
