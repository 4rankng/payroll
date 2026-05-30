package user

import (
	"log/slog"
	"os"
	"testing"

	"api-server/internal/pkg/password"

	"github.com/stretchr/testify/assert"
)

func TestApplyPeppering(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		hashSecret string
		hashSalt   string
		expected   string
	}{
		{
			name:       "No pepper - password only",
			password:   "mypassword",
			hashSecret: "",
			hashSalt:   "",
			expected:   "mypassword",
		},
		{
			name:       "With secret only",
			password:   "mypassword",
			hashSecret: "secret123",
			hashSalt:   "",
			expected:   "mypassword::secret123",
		},
		{
			name:       "With salt only",
			password:   "mypassword",
			hashSecret: "",
			hashSalt:   "salt456",
			expected:   "mypassword::salt456",
		},
		{
			name:       "With both secret and salt",
			password:   "mypassword",
			hashSecret: "secret123",
			hashSalt:   "salt456",
			expected:   "mypassword::secret123::salt456",
		},
		{
			name:       "Empty password with secret and salt",
			password:   "",
			hashSecret: "secret123",
			hashSalt:   "salt456",
			expected:   "::secret123::salt456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &UserService{
				hashSecret: tt.hashSecret,
				hashSalt:   tt.hashSalt,
			}

			result := service.applyPeppering(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := &UserService{
		passwordValidator: password.NewPasswordValidator(),
		logger:            logger,
	}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "Strong password",
			password: "StrongP@ssw0rd123",
			wantErr:  false,
		},
		{
			name:     "Weak password - too short",
			password: "short",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePassword(tt.password)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPasswordStrength(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := &UserService{
		passwordValidator: password.NewPasswordValidator(),
		logger:            logger,
	}

	tests := []struct {
		name          string
		password      string
		minScore      int
		checkFeedback bool
	}{
		{
			name:          "Strong password",
			password:      "VeryStr0ng!P@ssw0rd",
			minScore:      3,
			checkFeedback: false,
		},
		{
			name:          "Weak password",
			password:      "weak",
			minScore:      0,
			checkFeedback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, feedback := service.GetPasswordStrength(tt.password)
			assert.GreaterOrEqual(t, score, tt.minScore)
			if tt.checkFeedback {
				assert.NotEmpty(t, feedback)
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := &UserService{
		hashConfig: HashConfig{
			Memory:      64 * 1024,
			Iterations:  3,
			Parallelism: 2,
			SaltLength:  16,
			KeyLength:   32,
		},
		hashSecret: "test-secret",
		hashSalt:   "test-salt",
		logger:     logger,
	}

	password := "testPassword123"
	hash, err := service.hashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Hash should be different each time due to salt
	hash2, err := service.hashPassword(password)
	assert.NoError(t, err)
	assert.NotEqual(t, hash, hash2)
}

func TestVerifyPasswordHash(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := &UserService{
		hashConfig: HashConfig{
			Memory:      64 * 1024,
			Iterations:  3,
			Parallelism: 2,
			SaltLength:  16,
			KeyLength:   32,
		},
		hashSecret: "test-secret",
		hashSalt:   "test-salt",
		logger:     logger,
	}

	password := "testPassword123"
	hash, err := service.hashPassword(password)
	assert.NoError(t, err)

	// Should verify correct password
	result := service.VerifyPasswordHash(password, hash)
	assert.True(t, result)

	// Should fail with incorrect password
	result = service.VerifyPasswordHash("wrongPassword", hash)
	assert.False(t, result)
}
