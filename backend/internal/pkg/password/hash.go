package password

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// HashPassword replicates the argon2id settings from UserService.
// It uses a pepper from the provided secret and salt.
func HashPassword(pw, hashSecret, hashSalt string) (string, error) {
	// Mirror config in services.UserService: Memory 64*1024, Iterations 3, Parallelism 2, SaltLength 16, KeyLength 32
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	peppered := pw
	if hashSecret != "" {
		peppered += "::" + hashSecret
	}
	if hashSalt != "" {
		peppered += "::" + hashSalt
	}
	peppered = strings.TrimSpace(peppered)

	hash := argon2.IDKey([]byte(peppered), salt, 3, 64*1024, 2, 32)
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, 64*1024, 3, 2, b64Salt, b64Hash), nil
}
