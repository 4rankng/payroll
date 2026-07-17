// Package cache holds Redis-backed ephemeral stores for cross-request state
// that is not persisted to MySQL (e.g. pending OTP login sessions).
package cache

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultSessionTTL is how long a pending OTP session stays valid. Matches the
// OTP_CODE_TTL default (5m) configured in OTPConfig.
const DefaultSessionTTL = 5 * time.Minute

// ErrOTPSessionNotFound is returned when a session id is absent, expired, or
// has been superseded by a newer login for the same user.
var ErrOTPSessionNotFound = errors.New("otp session not found or expired")

var createSessionIfAbsentScript = redis.NewScript(`
local existing_id = redis.call("GET", KEYS[1])
if existing_id then
	local existing_key = ARGV[1] .. existing_id
	local remaining_ttl = redis.call("PTTL", existing_key)
	if remaining_ttl > 0 then
		redis.call("PEXPIRE", KEYS[1], remaining_ttl)
		return {existing_id, 0}
	end
	redis.call("DEL", KEYS[1])
end

redis.call("SET", KEYS[2], ARGV[2], "PX", ARGV[3])
redis.call("SET", KEYS[1], ARGV[4], "PX", ARGV[3])
return {ARGV[4], 1}
`)

var saveSessionScript = redis.NewScript(`
local remaining_ttl = redis.call("PTTL", KEYS[1])
if remaining_ttl <= 0 then
	return 0
end

local current_id = redis.call("GET", KEYS[2])
if not current_id or current_id ~= ARGV[1] then
	return 0
end

redis.call("SET", KEYS[1], ARGV[2], "PX", remaining_ttl)
redis.call("SET", KEYS[2], ARGV[1], "PX", remaining_ttl)
return remaining_ttl
`)

// OTPPendingSession is the per-login state stored under an opaque session id.
// It binds the second step (/auth/login/verify) to the first (/auth/login) and
// carries the information needed to validate the submitted code without ever
// holding the plaintext code itself — only its SHA-256 hash.
type OTPPendingSession struct {
	UserID       uint      `json:"user_id"`
	CodeHash     []byte    `json:"code_hash"`      // SHA-256 of the 6-digit code
	IP           string    `json:"ip"`             // RT-M5: bind to client IP
	UserAgent    string    `json:"user_agent"`     // RT-M5: bind to client UA
	Attempts     int       `json:"attempts"`       // verify attempts on this session
	LastResendAt time.Time `json:"last_resend_at"` // resend cooldown (Phase 3)
	CreatedAt    time.Time `json:"created_at"`
}

// OTPPendingStore manages pending OTP login sessions in Redis.
//
// RT-H1 (one live session per user): CreateSession overwrites any existing
// session for the same user via the secondary index key otp:user:<userID>.
// This prevents the brute-force amplifier where an attacker with the password
// creates N sessions and gets N × MaxAttempts attempts at the 6-digit code.
type OTPPendingStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewOTPPendingStore constructs a store using the shared Redis client.
func NewOTPPendingStore(client *redis.Client, ttl time.Duration) *OTPPendingStore {
	if ttl <= 0 {
		ttl = DefaultSessionTTL
	}
	return &OTPPendingStore{client: client, ttl: ttl}
}

// CreateSessionIfAbsent atomically creates a pending session unless the user
// already has a live one. Concurrent callers for the same user all receive the
// same session id, and only the caller that stored it receives created=true.
// A user index whose primary session has expired is replaced in the same Redis
// operation, so stale indexes do not block a new login attempt.
func (s *OTPPendingStore) CreateSessionIfAbsent(ctx context.Context, userID uint, codeHash []byte, ip, ua string) (sessionID string, created bool, err error) {
	sessionID, err = newOpaqueID()
	if err != nil {
		return "", false, fmt.Errorf("generate otp session id: %w", err)
	}

	session := OTPPendingSession{
		UserID:    userID,
		CodeHash:  codeHash,
		IP:        ip,
		UserAgent: ua,
		CreatedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return "", false, fmt.Errorf("marshal otp session: %w", err)
	}

	result, err := createSessionIfAbsentScript.Run(
		ctx,
		s.client,
		[]string{userSessionKey(userID), sessionKey(sessionID)},
		"otp:pending:",
		payload,
		s.ttl.Milliseconds(),
		sessionID,
	).Slice()
	if err != nil {
		return "", false, fmt.Errorf("create otp session: %w", err)
	}
	if len(result) != 2 {
		return "", false, fmt.Errorf("create otp session: unexpected redis result")
	}

	storedID, ok := result[0].(string)
	if !ok || storedID == "" {
		return "", false, fmt.Errorf("create otp session: invalid session id result")
	}
	createdValue, ok := result[1].(int64)
	if !ok {
		return "", false, fmt.Errorf("create otp session: invalid created result")
	}
	return storedID, createdValue == 1, nil
}

// CreateSession writes a new pending session for the user and returns the opaque
// session id to hand to the client. Any prior pending session for this user is
// invalidated (its id is overwritten in the secondary index; the old primary
// key is best-effort deleted). RT-H1.
func (s *OTPPendingStore) CreateSession(ctx context.Context, userID uint, codeHash []byte, ip, ua string) (string, error) {
	sessionID, err := newOpaqueID()
	if err != nil {
		return "", fmt.Errorf("generate otp session id: %w", err)
	}

	session := OTPPendingSession{
		UserID:    userID,
		CodeHash:  codeHash,
		IP:        ip,
		UserAgent: ua,
		CreatedAt: time.Now().UTC(),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return "", fmt.Errorf("marshal otp session: %w", err)
	}

	userKey := userSessionKey(userID)
	// Best-effort: invalidate the previous session id for this user before
	// overwriting the index. A stale id left in Redis simply expires at TTL.
	if prevID, err := s.client.Get(ctx, userKey).Result(); err == nil && prevID != "" {
		_ = s.client.Del(ctx, sessionKey(prevID)).Err()
	}

	// Write both keys with matching TTL in a pipeline.
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, sessionKey(sessionID), payload, s.ttl)
	pipe.Set(ctx, userKey, sessionID, s.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("write otp session: %w", err)
	}
	return sessionID, nil
}

// GetSession loads a pending session. Returns ErrOTPSessionNotFound if the id
// is absent/expired/superseded.
func (s *OTPPendingStore) GetSession(ctx context.Context, sessionID string) (*OTPPendingSession, error) {
	if sessionID == "" {
		return nil, ErrOTPSessionNotFound
	}
	payload, err := s.client.Get(ctx, sessionKey(sessionID)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrOTPSessionNotFound
		}
		return nil, fmt.Errorf("read otp session: %w", err)
	}
	var session OTPPendingSession
	if err := json.Unmarshal(payload, &session); err != nil {
		return nil, fmt.Errorf("unmarshal otp session: %w", err)
	}
	currentID, err := s.client.Get(ctx, userSessionKey(session.UserID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, ErrOTPSessionNotFound
		}
		return nil, fmt.Errorf("read otp user index: %w", err)
	}
	if currentID != sessionID {
		return nil, ErrOTPSessionNotFound
	}
	return &session, nil
}

// SaveSession writes back an updated session (e.g. after incrementing attempts
// or refreshing the resend timestamp), preserving the original TTL window via
// the per-user index. Used by Phase 3 resend (new code hash) and verify-fail
// (attempt counter).
func (s *OTPPendingStore) SaveSession(ctx context.Context, sessionID string, session *OTPPendingSession) error {
	payload, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal otp session: %w", err)
	}
	remainingTTL, err := saveSessionScript.Run(
		ctx,
		s.client,
		[]string{sessionKey(sessionID), userSessionKey(session.UserID)},
		sessionID,
		payload,
	).Int64()
	if err != nil {
		return fmt.Errorf("write otp session: %w", err)
	}
	if remainingTTL <= 0 {
		return ErrOTPSessionNotFound
	}
	return nil
}

// DeleteSession removes a pending session (e.g. after successful verify).
func (s *OTPPendingStore) DeleteSession(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.client.Del(ctx, sessionKey(sessionID)).Err()
}

// DeleteSessionsForUser removes ALL pending sessions for a user — both the
// current index and whichever id it points to. Called by RevokeUserTokens
// (RT-M4) so revoking a user also kills their in-flight OTP step.
func (s *OTPPendingStore) DeleteSessionsForUser(ctx context.Context, userID uint) error {
	userKey := userSessionKey(userID)
	sessionID, err := s.client.Get(ctx, userKey).Result()
	pipe := s.client.Pipeline()
	pipe.Del(ctx, userKey)
	if err == nil && sessionID != "" {
		pipe.Del(ctx, sessionKey(sessionID))
	}
	_, _ = pipe.Exec(ctx)
	return nil
}

func sessionKey(id string) string    { return "otp:pending:" + id }
func userSessionKey(uid uint) string { return fmt.Sprintf("otp:user:%d", uid) }

// newOpaqueID returns a 192-bit (32-byte) random id, base64url-encoded, suitable
// as a non-enumerable session identifier handed to untrusted clients.
func newOpaqueID() (string, error) {
	b := make([]byte, 24) // 24 bytes = 192 bits
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
