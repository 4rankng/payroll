package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// SettingsValueType represents the type of setting value
type SettingsValueType string

const (
	ValueTypeString  SettingsValueType = "string"
	ValueTypeNumber  SettingsValueType = "number"
	ValueTypeBoolean SettingsValueType = "boolean"
	ValueTypeJSON    SettingsValueType = "json"
)

// Settings represents a system configuration setting
type Settings struct {
	ID        uint              `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Key       string            `json:"key" gorm:"type:varchar(100);uniqueIndex;not null"`
	Value     *string           `json:"value" gorm:"type:text"`
	ValueType SettingsValueType `json:"value_type" gorm:"type:enum('string','number','boolean','json');not null;default:'string'"`
	DeletedAt gorm.DeletedAt    `json:"-" gorm:"index"`
	UpdatedAt time.Time         `json:"updated_at" gorm:"type:timestamp;default:CURRENT_TIMESTAMP;autoUpdateTime"`
}

// SettingsRepository defines the interface for settings persistence operations
type SettingsRepository interface {
	Create(ctx context.Context, settings *Settings) error
	GetByID(ctx context.Context, id uint) (*Settings, error)
	GetByKey(ctx context.Context, key string) (*Settings, error)
	Update(ctx context.Context, settings *Settings) error
	Delete(ctx context.Context, id uint) error
	List(ctx context.Context, filters SettingsFilters) ([]*Settings, error)
	Count(ctx context.Context, filters SettingsFilters) (int64, error)
}

// SettingsFilters represents filtering options for settings queries
type SettingsFilters struct {
	ValueType []SettingsValueType
	Search    string // Search in key
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// ValidateKey validates the setting's key
func (s *Settings) ValidateKey() error {
	if s.Key == "" {
		return NewValidationError("key is required")
	}
	if len(s.Key) > 100 {
		return NewValidationError("key must be less than 100 characters")
	}
	return nil
}

// ValidateValueType validates the setting's value type
func (s *Settings) ValidateValueType() error {
	switch s.ValueType {
	case ValueTypeString, ValueTypeNumber, ValueTypeBoolean, ValueTypeJSON:
		return nil
	default:
		return NewValidationError("value_type must be one of: string, number, boolean, json")
	}
}

// ValidateValue validates the setting's value based on its type
func (s *Settings) ValidateValue() error {
	if s.Value == nil {
		return nil // Null values are allowed
	}

	switch s.ValueType {
	case ValueTypeJSON:
		// Validate that the value is valid JSON
		var js json.RawMessage
		if err := json.Unmarshal([]byte(*s.Value), &js); err != nil {
			return NewValidationError("value must be valid JSON when value_type is 'json'")
		}
	case ValueTypeBoolean:
		// Validate that the value is a valid boolean string
		if *s.Value != "true" && *s.Value != "false" {
			return NewValidationError("value must be 'true' or 'false' when value_type is 'boolean'")
		}
	case ValueTypeNumber:
		// Basic number validation - could be enhanced with proper number parsing
		if *s.Value == "" {
			return NewValidationError("value cannot be empty when value_type is 'number'")
		}
		// Note: Could add more sophisticated number validation here if needed
	}

	return nil
}

// IsValid validates the entire settings entity
func (s *Settings) IsValid() error {
	if err := s.ValidateKey(); err != nil {
		return err
	}
	if err := s.ValidateValueType(); err != nil {
		return err
	}
	if err := s.ValidateValue(); err != nil {
		return err
	}
	return nil
}

// GetStringValue returns the value as string, or empty string if null
func (s *Settings) GetStringValue() string {
	if s.Value == nil {
		return ""
	}
	return *s.Value
}

// GetBoolValue returns the value as boolean, or false if null/invalid
func (s *Settings) GetBoolValue() bool {
	if s.Value == nil || s.ValueType != ValueTypeBoolean {
		return false
	}
	return *s.Value == "true"
}

// GetJSONValue unmarshals the value into the provided interface
func (s *Settings) GetJSONValue(dest interface{}) error {
	if s.Value == nil || s.ValueType != ValueTypeJSON {
		return NewValidationError("setting is not a JSON type or has no value")
	}
	return json.Unmarshal([]byte(*s.Value), dest)
}

// GetFloatValue returns the value as float64, or 0 if null/invalid
func (s *Settings) GetFloatValue() (float64, error) {
	if s.Value == nil {
		return 0, NewValidationError("setting value is null")
	}
	if s.ValueType != ValueTypeNumber {
		return 0, NewValidationError("setting is not a number type")
	}

	var result float64
	if _, err := fmt.Sscanf(*s.Value, "%f", &result); err != nil {
		return 0, NewValidationError("failed to parse number value: " + err.Error())
	}

	return result, nil
}
