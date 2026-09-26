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

// DefaultZaloResetCodeTTL is how long a Zalo OTP reset session stays valid.
// Matches the ZALO_RESET_CODE_TTL default (10m).
const DefaultZaloResetCodeTTL = 10 * time.Minute

// DefaultZaloResetVerifiedTTL is how long a verified reset token stays usable.
const DefaultZaloResetVerifiedTTL = 5 * time.Minute

// ErrZaloResetVerifiedNotFound is returned when a verified reset token is
// absent, expired, or already consumed.
var ErrZaloResetVerifiedNotFound = errors.New("zalo reset: verified token not found or expired")

// ErrZaloResetSessionNotFound is returned when a session id is absent, expired,
// or already consumed.
var ErrZaloResetSessionNotFound = errors.New("zalo reset session not found or expired")

// ErrZaloResetInvalidCode is returned when the session exists but the submitted
// code does not match. The session is NOT consumed on a wrong code — the user
// may retry within the session's TTL (the rate limiter bounds total attempts).
var ErrZaloResetInvalidCode = errors.New("zalo reset: invalid code")

// ErrZaloResetStoreUnavailable is returned when Redis itself is unreachable.
// Distinct from not-found so the service maps it to a 500 "try again" instead
// of lying that the code is expired.
var ErrZaloResetStoreUnavailable = errors.New("zalo reset session store unavailable")

// zaloConsumeScript atomically checks the code hash and, only on match, deletes
// the session (single-use). On a wrong code the key survives so the caller can
// retry. Returns the userID on success, "-1" on missing key, "-2" on wrong code.
//
//	KEYS[1] = zreset:<sha256(sessionID)>
//	ARGV[1] = sha256(code) hex
var zaloConsumeScript = redis.NewScript(`
local v = redis.call("GET", KEYS[1])
if v == false then
	return "-1"
end
local uid, hash = string.match(v, "^(%d+):(.+)$")
if hash ~= ARGV[1] then
	return "-2"
end
redis.call("DEL", KEYS[1])
return uid
`)

// ZaloResetStore manages self-service Zalo-OTP password-reset sessions in Redis.
//
// A session is an opaque 256-bit base64url id (handed to the client) mapped to a
// Redis value of "<userID>:<sha256(code)-hex>". Only hashes are stored: the
// session id is hashed for the key name (so a Redis dump can't replay it), and
// the 6-digit code is hashed in the value (log hygiene). The plaintext code is
// never stored and is verified via the Lua script in a single round-trip.
//
// Lifecycle:
//   - Create: mint session id, store "<uid>:<codeHash>" with TTL → return id.
//   - CreateDummy: store "<uid>:<dummy>" with TTL → return id. Used on the
//     not-found / disabled path so the HTTP response shape is identical for
//     known and unknown mobiles (anti-enumeration). The dummy's userID is 0,
//     which Consume rejects as not-found (a real user ID is always ≥ 1).
//   - Consume: atomic compare-code-then-delete. Wrong code survives for retry.
type ZaloResetStore struct {
	client      *redis.Client
	ttl         time.Duration
	verifiedTTL time.Duration
}

// NewZaloResetStore constructs a store. ttl defaults to DefaultZaloResetCodeTTL.
// verifiedTTL (the lifetime of a verified reset token) defaults to
// DefaultZaloResetVerifiedTTL.
func NewZaloResetStore(client *redis.Client, ttl time.Duration) *ZaloResetStore {
	if ttl <= 0 {
		ttl = DefaultZaloResetCodeTTL
	}
	return &ZaloResetStore{
		client:      client,
		ttl:         ttl,
		verifiedTTL: DefaultZaloResetVerifiedTTL,
	}
}

// TTL returns the configured OTP-session lifetime, so callers can report
// expires_in without re-reading config.
func (s *ZaloResetStore) TTL() time.Duration { return s.ttl }

// Create generates a fresh session id, stores "<userID>:<codeHashHex>" under
// zreset:<sha256(sessionID)> with TTL, and returns the opaque session id.
func (s *ZaloResetStore) Create(ctx context.Context, userID uint, codeHashHex string) (string, error) {
	sid, err := newZaloOpaqueID()
	if err != nil {
		return "", fmt.Errorf("zalo reset: generate session id: %w", err)
	}
	val := strconv.FormatUint(uint64(userID), 10) + ":" + codeHashHex
	if err := s.client.Set(ctx, zaloSessionKey(sid), val, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("zalo reset: write session: %w", err)
	}
	return sid, nil
}

// CreateDummy stores a placeholder session (userID=0) so a not-found mobile
// gets back a structurally identical session id (anti-enumeration). The dummy
// can never be successfully consumed because no real user has ID 0; a confirm
// attempt against it resolves to ErrZaloResetSessionNotFound after the code
// compare (which fails against the placeholder hash) — but to keep the timing
// profile identical to a real session, we store a real random hash.
func (s *ZaloResetStore) CreateDummy(ctx context.Context) (string, error) {
	sid, err := newZaloOpaqueID()
	if err != nil {
		return "", fmt.Errorf("zalo reset: generate dummy session id: %w", err)
	}
	dummyHash := randomHashHex()
	val := "0:" + dummyHash
	if err := s.client.Set(ctx, zaloSessionKey(sid), val, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("zalo reset: write dummy session: %w", err)
	}
	return sid, nil
}

// Consume atomically validates the code against the session and, on match,
// deletes the session (single-use). On a wrong code the session survives for
// retry; on a missing/expired/consumed session it returns ErrZaloResetSessionNotFound.
//
// Note: the dummy session created by CreateDummy has userID=0; if a caller
// somehow supplied the matching dummy hash, Consume would return 0 — but the
// hash is cryptographically random and never disclosed, so this is not a
// practical concern. The service treats userID 0 as "not found".
func (s *ZaloResetStore) Consume(ctx context.Context, sessionID, codeHashHex string) (uint, error) {
	key := zaloSessionKey(sessionID)
	result, err := zaloConsumeScript.Run(ctx, s.client, []string{key}, codeHashHex).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrZaloResetSessionNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrZaloResetStoreUnavailable, err)
	}
	str, ok := result.(string)
	if !ok {
		return 0, ErrZaloResetSessionNotFound
	}
	switch str {
	case "-1":
		return 0, ErrZaloResetSessionNotFound
	case "-2":
		return 0, ErrZaloResetInvalidCode
	}
	uid, err := strconv.ParseUint(str, 10, 64)
	if err != nil || uid == 0 {
		// uid==0 means the dummy session — treat as not-found.
		return 0, ErrZaloResetSessionNotFound
	}
	return uint(uid), nil
}

// Delete removes a session without checking the code. Used by resend to
// replace a session with a fresh one (rotating the code).
func (s *ZaloResetStore) Delete(ctx context.Context, sessionID string) error {
	return s.client.Del(ctx, zaloSessionKey(sessionID)).Err()
}

func zaloSessionKey(sessionID string) string {
	sum := sha256.Sum256([]byte(sessionID))
	return "zreset:" + hex.EncodeToString(sum[:])
}

// CreateVerified mints a single-use token bound to userID after the OTP was
// verified. The token is stored under zreset-ok:<sha256(token)> with
// verifiedTTL and is consumed exactly once by ConsumeVerified.
func (s *ZaloResetStore) CreateVerified(ctx context.Context, userID uint) (string, error) {
	token, err := newZaloOpaqueID()
	if err != nil {
		return "", fmt.Errorf("zalo reset: generate verified token: %w", err)
	}
	val := strconv.FormatUint(uint64(userID), 10)
	if err := s.client.Set(ctx, zaloVerifiedKey(token), val, s.verifiedTTL).Err(); err != nil {
		return "", fmt.Errorf("zalo reset: write verified token: %w", err)
	}
	return token, nil
}

// ConsumeVerified atomically fetches and deletes a verified reset token
// (Redis GETDEL — single use). Returns the bound userID. A missing, expired, or
// already-consumed token returns ErrZaloResetVerifiedNotFound. A Redis outage
// returns ErrZaloResetStoreUnavailable.
func (s *ZaloResetStore) ConsumeVerified(ctx context.Context, token string) (uint, error) {
	val, err := s.client.GetDel(ctx, zaloVerifiedKey(token)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrZaloResetVerifiedNotFound
		}
		return 0, fmt.Errorf("%w: %v", ErrZaloResetStoreUnavailable, err)
	}
	uid, err := strconv.ParseUint(val, 10, 64)
	if err != nil || uid == 0 {
		return 0, ErrZaloResetVerifiedNotFound
	}
	return uint(uid), nil
}

func zaloVerifiedKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "zreset-ok:" + hex.EncodeToString(sum[:])
}

// newZaloOpaqueID returns a 256-bit base64url id (≈43 chars, no padding).
// Matches the entropy of OTPPendingStore.newOpaqueID / PasswordResetTokenStore.
func newZaloOpaqueID() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// randomHashHex returns 64 hex chars of a random SHA-256 — used for the dummy
// session's fake code hash so its Redis value is the same length/shape as a
// real session's.
func randomHashHex() string {
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
