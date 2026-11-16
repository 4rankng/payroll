package user

import (
	"log/slog"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/stretchr/testify/assert"
)

// Legacy accounts imported with bcrypt hashes (e.g. nv478/nv548) must verify
// through VerifyPasswordHash's bcrypt fallback; unknown formats stay false.
func TestVerifyPasswordHashBcryptFallback(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	service := &UserService{
		hashSecret: "test-secret",
		hashSalt:   "test-salt",
		logger:     logger,
	}

	password := "Password123!"

	bcryptHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	assert.NoError(t, err)
	assert.Regexp(t, `^\$2[aby]\$`, string(bcryptHash))

	assert.True(t, service.VerifyPasswordHash(password, string(bcryptHash)), "correct password must verify against a legacy bcrypt hash")
	assert.False(t, service.VerifyPasswordHash("wrong-password", string(bcryptHash)), "wrong password must not verify")

	// Unknown / malformed formats must return false, never panic.
	assert.False(t, service.VerifyPasswordHash(password, "not-a-hash"))
	assert.False(t, service.VerifyPasswordHash(password, "$2c$10$truncated"))
}
