package dto

import (
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/secret"
)

// CreateSettingRequest represents the request to create a new setting
type CreateSettingRequest struct {
	Key       string  `json:"key" binding:"required,max=100"`
	Value     *string `json:"value"`
	ValueType string  `json:"value_type" binding:"required,oneof=string number boolean json"`
}

// UpdateSettingRequest represents the request to update a setting
type UpdateSettingRequest struct {
	Key       *string `json:"key,omitempty" binding:"omitempty,max=100"`
	Value     *string `json:"value,omitempty"`
	ValueType *string `json:"value_type,omitempty" binding:"omitempty,oneof=string number boolean json"`
}

// SettingResponse represents the response containing setting data
type SettingResponse struct {
	ID        uint    `json:"id"`
	Key       string  `json:"key"`
	Value     *string `json:"value"`
	ValueType string  `json:"value_type"`
	// Masked is true when Value was withheld because the key holds a secret;
	// Configured then reports whether a value is stored at all — enough for an
	// admin panel to show connection status without shipping the credential.
	Masked     bool `json:"masked,omitempty"`
	Configured bool `json:"configured,omitempty"`
	// UpdatedAt is when the value was last written.
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSettingResponse builds the API view of a setting. Values of
// secret.ProtectedKeys rows are never returned: the response carries
// masked=true and configured=<a value exists> instead, so
// GET /admin/settings, /admin/settings/{id}, /admin/settings/key/{key} and
// /admin/settings/active cannot leak an integration secret.
func NewSettingResponse(setting *domain.Settings) SettingResponse {
	resp := SettingResponse{
		ID:        setting.ID,
		Key:       setting.Key,
		Value:     setting.Value,
		ValueType: string(setting.ValueType),
		UpdatedAt: setting.UpdatedAt,
	}
	if secret.IsProtectedKey(setting.Key) {
		resp.Masked = true
		resp.Configured = setting.Value != nil && *setting.Value != ""
		resp.Value = nil
	}
	return resp
}

// ListSettingsResponse represents the response for listing settings
type ListSettingsResponse struct {
	Settings []SettingResponse `json:"settings"`
	Total    int64             `json:"total"`
	Limit    int               `json:"limit"`
	Offset   int               `json:"offset"`
}

// GetActiveSettingsResponse represents the response for active settings
type GetActiveSettingsResponse struct {
	Settings []SettingResponse `json:"settings"`
}
