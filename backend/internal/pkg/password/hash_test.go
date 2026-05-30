package password

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name       string
		password   string
		hashSecret string
		hashSalt   string
		wantErr    bool
	}{
		{
			name:       "Valid password with secret and salt",
			password:   "MyPassword123!",
			hashSecret: "secret123",
			hashSalt:   "salt456",
			wantErr:    false,
		},
		{
			name:       "Valid password without secret",
			password:   "MyPassword123!",
			hashSecret: "",
			hashSalt:   "salt456",
			wantErr:    false,
		},
		{
			name:       "Valid password without salt",
			password:   "MyPassword123!",
			hashSecret: "secret123",
			hashSalt:   "",
			wantErr:    false,
		},
		{
			name:       "Valid password without secret and salt",
			password:   "MyPassword123!",
			hashSecret: "",
			hashSalt:   "",
			wantErr:    false,
		},
		{
			name:       "Empty password",
			password:   "",
			hashSecret: "secret123",
			hashSalt:   "salt456",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password, tt.hashSecret, tt.hashSalt)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Empty(t, hash)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, hash)
				// Verify hash format
				assert.True(t, strings.HasPrefix(hash, "$argon2id$"))
				// Verify hash contains all components
				parts := strings.Split(hash, "$")
				assert.GreaterOrEqual(t, len(parts), 6)
			}
		})
	}
}

func TestHashPassword_Consistency(t *testing.T) {
	password := "MyPassword123!"
	hashSecret := "secret123"
	hashSalt := "salt456"

	// Generate two hashes with same inputs
	hash1, err1 := HashPassword(password, hashSecret, hashSalt)
	assert.NoError(t, err1)

	hash2, err2 := HashPassword(password, hashSecret, hashSalt)
	assert.NoError(t, err2)

	// Hashes should be different due to random salt
	assert.NotEqual(t, hash1, hash2)

	// But both should be valid argon2id hashes
	assert.True(t, strings.HasPrefix(hash1, "$argon2id$"))
	assert.True(t, strings.HasPrefix(hash2, "$argon2id$"))
}

func TestHashPassword_DifferentInputs(t *testing.T) {
	password := "MyPassword123!"

	// Test with different secrets
	hash1, err1 := HashPassword(password, "secret1", "salt1")
	assert.NoError(t, err1)

	hash2, err2 := HashPassword(password, "secret2", "salt1")
	assert.NoError(t, err2)

	// Hashes should be different (even if we ignore the random salt)
	assert.NotEqual(t, hash1, hash2)

	// Test with different salts
	hash3, err3 := HashPassword(password, "secret1", "salt1")
	assert.NoError(t, err3)

	hash4, err4 := HashPassword(password, "secret1", "salt2")
	assert.NoError(t, err4)

	assert.NotEqual(t, hash3, hash4)
}

func TestHashPassword_Format(t *testing.T) {
	password := "MyPassword123!"
	hashSecret := "secret123"
	hashSalt := "salt456"

	hash, err := HashPassword(password, hashSecret, hashSalt)
	assert.NoError(t, err)

	// Verify argon2id parameters are in the hash
	assert.Contains(t, hash, "m=65536") // Memory parameter
	assert.Contains(t, hash, "t=3")     // Iterations
	assert.Contains(t, hash, "p=2")     // Parallelism
	assert.Contains(t, hash, "v=19")    // Argon2 version
}
