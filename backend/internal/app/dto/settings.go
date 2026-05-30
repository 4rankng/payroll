package dto

import (
	"time"
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
	ID        uint      `json:"id"`
	Key       string    `json:"key"`
	Value     *string   `json:"value"`
	ValueType string    `json:"value_type"`
	UpdatedAt time.Time `json:"updated_at"`
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
