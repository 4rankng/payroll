package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSettings_ValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "app_setting",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "key too long",
			key:     string(make([]byte, 101)),
			wantErr: true,
		},
		{
			name:    "key at boundary (100 chars)",
			key:     string(make([]byte, 100)),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Key: tt.key}
			err := s.ValidateKey()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettings_ValidateValueType(t *testing.T) {
	tests := []struct {
		name      string
		valueType SettingsValueType
		wantErr   bool
	}{
		{
			name:      "string type",
			valueType: ValueTypeString,
			wantErr:   false,
		},
		{
			name:      "number type",
			valueType: ValueTypeNumber,
			wantErr:   false,
		},
		{
			name:      "boolean type",
			valueType: ValueTypeBoolean,
			wantErr:   false,
		},
		{
			name:      "json type",
			valueType: ValueTypeJSON,
			wantErr:   false,
		},
		{
			name:      "invalid type",
			valueType: "invalid",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{ValueType: tt.valueType}
			err := s.ValidateValueType()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettings_ValidateValue(t *testing.T) {
	jsonValue := `{"key": "value"}`
	invalidJSON := `{invalid json`
	trueValue := "true"
	falseValue := "false"
	invalidBool := "yes"
	numberValue := "123.45"
	emptyValue := ""

	tests := []struct {
		name      string
		value     *string
		valueType SettingsValueType
		wantErr   bool
	}{
		{
			name:      "null value",
			value:     nil,
			valueType: ValueTypeString,
			wantErr:   false,
		},
		{
			name:      "valid JSON",
			value:     &jsonValue,
			valueType: ValueTypeJSON,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			value:     &invalidJSON,
			valueType: ValueTypeJSON,
			wantErr:   true,
		},
		{
			name:      "valid boolean true",
			value:     &trueValue,
			valueType: ValueTypeBoolean,
			wantErr:   false,
		},
		{
			name:      "valid boolean false",
			value:     &falseValue,
			valueType: ValueTypeBoolean,
			wantErr:   false,
		},
		{
			name:      "invalid boolean",
			value:     &invalidBool,
			valueType: ValueTypeBoolean,
			wantErr:   true,
		},
		{
			name:      "valid number",
			value:     &numberValue,
			valueType: ValueTypeNumber,
			wantErr:   false,
		},
		{
			name:      "empty number",
			value:     &emptyValue,
			valueType: ValueTypeNumber,
			wantErr:   true,
		},
		{
			name:      "valid string",
			value:     &jsonValue,
			valueType: ValueTypeString,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Value: tt.value, ValueType: tt.valueType}
			err := s.ValidateValue()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettings_IsValid(t *testing.T) {
	validValue := "test"

	tests := []struct {
		name     string
		settings Settings
		wantErr  bool
	}{
		{
			name: "valid settings",
			settings: Settings{
				Key:       "test_key",
				Value:     &validValue,
				ValueType: ValueTypeString,
			},
			wantErr: false,
		},
		{
			name: "invalid key",
			settings: Settings{
				Key:       "",
				Value:     &validValue,
				ValueType: ValueTypeString,
			},
			wantErr: true,
		},
		{
			name: "invalid value type",
			settings: Settings{
				Key:       "test_key",
				Value:     &validValue,
				ValueType: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settings.IsValid()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettings_GetStringValue(t *testing.T) {
	value := "test_value"

	tests := []struct {
		name  string
		value *string
		want  string
	}{
		{
			name:  "non-null value",
			value: &value,
			want:  "test_value",
		},
		{
			name:  "null value",
			value: nil,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Value: tt.value}
			got := s.GetStringValue()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSettings_GetBoolValue(t *testing.T) {
	trueValue := "true"
	falseValue := "false"
	invalidValue := "yes"

	tests := []struct {
		name      string
		value     *string
		valueType SettingsValueType
		want      bool
	}{
		{
			name:      "true boolean",
			value:     &trueValue,
			valueType: ValueTypeBoolean,
			want:      true,
		},
		{
			name:      "false boolean",
			value:     &falseValue,
			valueType: ValueTypeBoolean,
			want:      false,
		},
		{
			name:      "null value",
			value:     nil,
			valueType: ValueTypeBoolean,
			want:      false,
		},
		{
			name:      "wrong type",
			value:     &trueValue,
			valueType: ValueTypeString,
			want:      false,
		},
		{
			name:      "invalid boolean string",
			value:     &invalidValue,
			valueType: ValueTypeBoolean,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Value: tt.value, ValueType: tt.valueType}
			got := s.GetBoolValue()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSettings_GetFloatValue(t *testing.T) {
	validNumber := "123.45"
	invalidNumber := "abc"
	emptyValue := ""

	tests := []struct {
		name      string
		value     *string
		valueType SettingsValueType
		want      float64
		wantErr   bool
	}{
		{
			name:      "valid float",
			value:     &validNumber,
			valueType: ValueTypeNumber,
			want:      123.45,
			wantErr:   false,
		},
		{
			name:      "null value",
			value:     nil,
			valueType: ValueTypeNumber,
			want:      0,
			wantErr:   true,
		},
		{
			name:      "wrong type",
			value:     &validNumber,
			valueType: ValueTypeString,
			want:      0,
			wantErr:   true,
		},
		{
			name:      "invalid number",
			value:     &invalidNumber,
			valueType: ValueTypeNumber,
			want:      0,
			wantErr:   true,
		},
		{
			name:      "empty string",
			value:     &emptyValue,
			valueType: ValueTypeNumber,
			want:      0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Value: tt.value, ValueType: tt.valueType}
			got, err := s.GetFloatValue()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestSettings_GetJSONValue(t *testing.T) {
	validJSON := `{"name": "test", "count": 42}`
	invalidJSON := `{invalid}`
	stringValue := "not json type"

	tests := []struct {
		name      string
		value     *string
		valueType SettingsValueType
		wantErr   bool
	}{
		{
			name:      "valid JSON",
			value:     &validJSON,
			valueType: ValueTypeJSON,
			wantErr:   false,
		},
		{
			name:      "invalid JSON",
			value:     &invalidJSON,
			valueType: ValueTypeJSON,
			wantErr:   true,
		},
		{
			name:      "null value",
			value:     nil,
			valueType: ValueTypeJSON,
			wantErr:   true,
		},
		{
			name:      "wrong type",
			value:     &stringValue,
			valueType: ValueTypeString,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Settings{Value: tt.value, ValueType: tt.valueType}
			var dest map[string]interface{}
			err := s.GetJSONValue(&dest)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "test", dest["name"])
				assert.Equal(t, float64(42), dest["count"])
			}
		})
	}
}
