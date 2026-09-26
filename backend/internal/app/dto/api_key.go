package dto

import "time"

// CreateAPIKeyRequest is the body for POST /admin/api-keys.
type CreateAPIKeyRequest struct {
	Name string `json:"name" binding:"required,max=100"`
}

// APIKeyResponse is the admin view of an API key. Key carries the plaintext
// and is populated ONLY in the create response — never on list.
type APIKeyResponse struct {
	ID         uint       `json:"id"`
	Name       string     `json:"name"`
	Key        string     `json:"key,omitempty"`
	KeyPrefix  string     `json:"key_prefix"`
	CreatedBy  uint       `json:"created_by"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}
