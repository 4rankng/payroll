package constants

// Context Keys Constants
const (
	CtxUserID   = "user_id"
	CtxUserRole = "user_role"
)

// System User ID - used for automated/system operations
const SystemUserID uint = 49

// AdminUserID is the primary admin user used for system-initiated operations
// that require an admin context (e.g., reconciliation, wallet sync).
const AdminUserID uint = 1
