package password

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPasswordValidator(t *testing.T) {
	validator := NewPasswordValidator()
	assert.NotNil(t, validator)
	assert.Len(t, validator.rules, 6)
}

func TestPasswordValidator_Validate(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Valid password with all requirements",
			password: "Test123!@#",
			wantErr:  false,
		},
		{
			name:     "Password too short",
			password: "Test1!",
			wantErr:  true,
		},
		{
			name:     "Password missing uppercase",
			password: "test123!@#",
			wantErr:  true,
		},
		{
			name:     "Password missing lowercase",
			password: "TEST123!@#",
			wantErr:  true,
		},
		{
			name:     "Password missing digit",
			password: "TestTest!@#",
			wantErr:  true,
		},
		{
			name:     "Password missing special character",
			password: "Test123456",
			wantErr:  true,
		},
		{
			name:     "Strong valid password",
			password: "MySecure!Pass123",
			wantErr:  false,
		},
		{
			name:     "Password too long (over 72 chars)",
			password: "Test123!@#Test123!@#Test123!@#Test123!@#Test123!@#Test123!@#Test123!@#Test123!@#",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPasswordValidator_ValidatePasswordStrength(t *testing.T) {
	validator := NewPasswordValidator()

	tests := []struct {
		name        string
		password    string
		minScore    int
		maxScore    int
		hasFeedback bool
	}{
		{
			name:        "Strong password",
			password:    "MySecure!Pass123",
			minScore:    90,
			maxScore:    100,
			hasFeedback: false,
		},
		{
			name:        "Medium password",
			password:    "Test1234!",
			minScore:    75,
			maxScore:    100,
			hasFeedback: false,
		},
		{
			name:        "Weak password - short",
			password:    "Test1!",
			minScore:    0,
			maxScore:    70,
			hasFeedback: true,
		},
		{
			name:        "No uppercase",
			password:    "test1234!",
			minScore:    0,
			maxScore:    85,
			hasFeedback: true,
		},
		{
			name:        "No special char",
			password:    "Test123456",
			minScore:    0,
			maxScore:    85,
			hasFeedback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, feedback := validator.ValidatePasswordStrength(tt.password)
			assert.GreaterOrEqual(t, score, tt.minScore)
			assert.LessOrEqual(t, score, tt.maxScore)
			if tt.hasFeedback {
				assert.NotEmpty(t, feedback)
			}
		})
	}
}

func TestGetStrengthLabel(t *testing.T) {
	tests := []struct {
		name  string
		score int
		want  string
	}{
		{
			name:  "Very strong",
			score: 95,
			want:  "Rất mạnh",
		},
		{
			name:  "Strong",
			score: 80,
			want:  "Mạnh",
		},
		{
			name:  "Medium",
			score: 60,
			want:  "Trung bình",
		},
		{
			name:  "Weak",
			score: 30,
			want:  "Yếu",
		},
		{
			name:  "Very weak",
			score: 10,
			want:  "Rất yếu",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetStrengthLabel(tt.score)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPasswordValidator_AddCustomRule(t *testing.T) {
	validator := NewPasswordValidator()
	initialRules := len(validator.rules)

	customRule := ValidationRule{
		Check:   func(p string) bool { return len(p) >= 10 },
		Message: "Custom rule message",
	}

	validator.AddCustomRule(customRule)
	assert.Equal(t, initialRules+1, len(validator.rules))
}

func TestPasswordValidator_RemoveRule(t *testing.T) {
	validator := NewPasswordValidator()
	initialRules := len(validator.rules)

	// Remove first rule
	err := validator.RemoveRule(0)
	assert.NoError(t, err)
	assert.Equal(t, initialRules-1, len(validator.rules))

	// Try to remove invalid index
	err = validator.RemoveRule(100)
	assert.Error(t, err)

	// Try to remove negative index
	err = validator.RemoveRule(-1)
	assert.Error(t, err)
}
