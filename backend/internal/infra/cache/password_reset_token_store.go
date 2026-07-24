package cache

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultPasswordResetTokenTTL is how long a password-reset token stays valid.
// Matches the PASSWORD_RESET_TOKEN_TTL default (30m) configured in PasswordResetConfig.
const DefaultPasswordResetTokenTTL = 30 * time.Minute

// ErrPasswordResetTokenNotFound is returned when a token is absent, expired,
// or already consumed (GETDEL returned nothing).
var ErrPasswordResetTokenNotFound = errors.New("password reset token not found or expired")

// ErrPasswordResetStoreUnavailable is returned when Redis itself is unreachable
// (connection refused, timeout, network partition). Distinct from not-found so
// the service can map it to a 500 "try again" instead of lying to the user
// that their valid link is expired. (Red Team H5.)
var ErrPasswordResetStoreUnavailable = errors.New("password reset token store unavailable")

// consumeScript atomically reads-then-deletes the token via GETDEL (Redis ≥6.2;
// project runs Redis 7). Atomicity eliminates the race where two concurrent
// confirm requests both read a valid token before either deletes it.
var consumeScript = redis.NewScript(`
local v = redis.call("GETDEL", KEYS[1])
if v == false then
	return false
end
return v
`)

// PasswordResetTokenStore manages self-service password-reset tokens in Redis.
//
// Tokens are 256-bit (32-byte) random values, base64url-encoded (≈43 chars),
// handed to the caller to embed in a magic-link email. Only the SHA-256 hash
// of the token is stored in Redis, so a Redis dump does not leak usable tokens
// (same hygiene as OTPPendingStore storing code_hash). Keys are namespaced
// pwreset:<sha256-hex>, consistent with the otp:pending:* / otp:user:* convention.
type PasswordResetTokenStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewPasswordResetTokenStore constructs a store. ttl defaults to
// DefaultPasswordResetTokenTTL (30m) when zero or negative.
func NewPasswordResetTokenStore(client *redis.Client, ttl time.Duration) *PasswordResetTokenStore {
	if ttl <= 0 {
		ttl = DefaultPasswordResetTokenTTL
	}
	return &PasswordResetTokenStore{client: client, ttl: ttl}
}

// Create generates a fresh 32-byte token, stores its SHA-256 hash mapped to
// the userID under pwreset:<hashHex> with TTL, and returns the opaque token
// string to embed in the magic-link URL. The plaintext token is NEVER stored.
func (s *PasswordResetTokenStore) Create(ctx context.Context, userID uint) (string, error) {
	raw := make([]byte, 32) // 256 bits
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("password reset: generate token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw) // no padding; URL-safe
	hashHex := tokenHashHex(token)

	if err := s.client.Set(ctx, tokenKey(hashHex), strconv.FormatUint(uint64(userID), 10), s.ttl).Err(); err != nil {
		return "", fmt.Errorf("password reset: write token: %w", err)
	}
	return token, nil
}

// Consume atomically validates the token and deletes it (single-use). Returns
// the userID it was issued for.
//
// Error mapping (Red Team H5):
//   - ErrPasswordResetTokenNotFound: the token is absent/expired/already-used,
//     OR the stored value is malformed. The service maps this to 401.
//   - ErrPasswordResetStoreUnavailable: Redis is unreachable. The service maps
//     this to 500 so the user isn't lied to that their valid link is expired.
func (s *PasswordResetTokenStore) Consume(ctx context.Context, token string) (uint, error) {
	hashHex := tokenHashHex(token)
	result, err := consumeScript.Run(ctx, s.client, []string{tokenKey(hashHex)}).Result()
	if err != nil {
		// Redis connection / availability errors — NOT "key missing".
		if errors.Is(err, redis.Nil) {
			return 0, ErrPasswordResetTokenNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrPasswordResetStoreUnavailable, err)
	}
	// Lua returns bool `false` when GETDEL found nothing.
	if b, ok := result.(bool); ok && !b {
		return 0, ErrPasswordResetTokenNotFound
	}
	str, ok := result.(string)
	if !ok || str == "" {
		return 0, ErrPasswordResetTokenNotFound
	}
	uid, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, ErrPasswordResetTokenNotFound
	}
	return uint(uid), nil
}

// Delete removes a token (non-atomic). Provided for admin/teardown paths; the
// normal single-use path is Consume (atomic GETDEL).
func (s *PasswordResetTokenStore) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, tokenKey(tokenHashHex(token))).Err()
}

func tokenKey(hashHex string) string  { return "pwreset:" + hashHex }
func tokenHashHex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
