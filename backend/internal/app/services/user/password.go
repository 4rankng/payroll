package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// DefaultHashConfig returns the Argon2id parameters used by UserService.
// Exposed so out-of-process tooling (e.g. cmd/hashpw) can produce hashes
// that the running app will accept on login.
func DefaultHashConfig() HashConfig {
	return HashConfig{
		Memory:      64 * 1024,
		Iterations:  3,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
	}
}

// HashPassword produces an Argon2id encoded hash for the given password using
// the provided peppers and config. This is the single source of truth for
// hashing — UserService.hashPassword delegates here, and dev tooling reuses it.
func HashPassword(password, hashSecret, hashSalt string, cfg HashConfig) (string, error) {
	salt := make([]byte, cfg.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	peppered := ApplyPeppering(password, hashSecret, hashSalt)

	hash := argon2.IDKey(
		[]byte(peppered),
		salt,
		cfg.Iterations,
		cfg.Memory,
		cfg.Parallelism,
		cfg.KeyLength,
	)

	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		cfg.Memory,
		cfg.Iterations,
		cfg.Parallelism,
		b64Salt,
		b64Hash,
	), nil
}

// hashPassword generates a secure hash for the given password using Argon2ID
func (s *UserService) hashPassword(password string) (string, error) {
	s.logger.Info("Hashing password", "method", "argon2id")

	encodedHash, err := HashPassword(password, s.hashSecret, s.hashSalt, s.hashConfig)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		return "", err
	}

	s.logger.Info("Password hashed successfully")
	return encodedHash, nil
}

// VerifyPasswordHash verifies a password against an encoded hash using constant-time comparison
func (s *UserService) VerifyPasswordHash(password, encodedHash string) bool {
	s.logger.Info("Verifying password hash", "method", "argon2id")

	// Parse the encoded hash
	var version int
	var memory, iterations uint32
	var parallelism uint8
	var salt, hash string

	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" {
		s.logger.Info("Invalid hash format", "parts_count", len(parts))
		return false
	}

	// Parse version
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		s.logger.Info("Failed to parse version from hash", "error", err)
		return false
	}

	// Parse parameters
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		s.logger.Info("Failed to parse parameters from hash", "error", err)
		return false
	}

	salt = parts[4]
	hash = parts[5]

	// Decode salt and hash
	saltBytes, err := base64.RawStdEncoding.DecodeString(salt)
	if err != nil {
		s.logger.Info("Failed to decode salt", "error", err)
		return false
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(hash)
	if err != nil {
		s.logger.Info("Failed to decode hash", "error", err)
		return false
	}

	// Apply peppering (same as hashing)
	peppered := s.applyPeppering(password)

	// Hash the provided password with the same parameters
	providedHash := argon2.IDKey(
		[]byte(peppered),
		saltBytes,
		iterations,
		memory,
		parallelism,
		uint32(len(hashBytes)),
	)

	// Compare hashes using constant time comparison for security
	isValid := subtle.ConstantTimeCompare(hashBytes, providedHash) == 1

	s.logger.Info("Password verification completed",
		"is_valid", isValid,
		"memory", memory,
		"iterations", iterations,
		"parallelism", parallelism)

	return isValid
}

// ValidatePassword validates a password using Vietnamese requirements
func (s *UserService) ValidatePassword(password string) error {
	s.logger.Info("Validating password strength")

	err := s.passwordValidator.Validate(password)
	if err != nil {
		s.logger.Info("Password validation failed", "error", err.Error())
	} else {
		s.logger.Info("Password validation successful")
	}

	return err
}

// GetPasswordStrength returns password strength score and feedback
func (s *UserService) GetPasswordStrength(password string) (int, []string) {
	s.logger.Info("Calculating password strength")

	score, feedback := s.passwordValidator.ValidatePasswordStrength(password)

	s.logger.Info("Password strength calculated",
		"score", score,
		"feedback_count", len(feedback))

	return score, feedback
}

// ApplyPeppering applies the application-level pepper to the password.
// Exposed so dev tooling can reuse the exact peppering scheme.
func ApplyPeppering(password, hashSecret, hashSalt string) string {
	peppered := password

	if hashSecret != "" {
		peppered += "::" + hashSecret
	}

	if hashSalt != "" {
		peppered += "::" + hashSalt
	}

	return strings.TrimSpace(peppered)
}

// applyPeppering applies the application-level pepper to the password
// This adds an additional layer of security beyond salting
func (s *UserService) applyPeppering(password string) string {
	return ApplyPeppering(password, s.hashSecret, s.hashSalt)
}
