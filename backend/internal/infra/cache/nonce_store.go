package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ErrNonceAlreadyUsed is returned when a nonce has been consumed (or was never
// set — a missing nonce claim is treated the same as a replay attempt, since
// the OIDC flow always mints one).
var ErrNonceAlreadyUsed = errors.New("nonce already used or invalid")

// NonceStore is a Redis-backed single-use nonce tracker for the Google OIDC
// id_token. Each login's nonce is generated client-side, sent to Google, and
// embedded in the returned id_token. The backend consumes it once here; a
// second presentation of the same nonce (a replay) is rejected.
//
// Nonces expire at the TTL (default 10 minutes — comfortably longer than the
// redirect round-trip). The key namespace is distinct from OTP pending sessions.
type NonceStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewNonceStore constructs a store. ttl should exceed the longest plausible
// Google-redirect round-trip (10m is generous).
func NewNonceStore(client *redis.Client, ttl time.Duration) *NonceStore {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &NonceStore{client: client, ttl: ttl}
}

// Consume atomically marks a nonce as used. It uses SETNX (SetNX) so the first
// caller wins; any later caller gets ErrNonceAlreadyUsed. An empty nonce is
// rejected outright — a valid OIDC flow always includes one.
func (s *NonceStore) Consume(ctx context.Context, nonce string) error {
	if nonce == "" {
		return ErrNonceAlreadyUsed
	}
	key := nonceKey(nonce)
	ok, err := s.client.SetNX(ctx, key, "1", s.ttl).Result()
	if err != nil {
		return fmt.Errorf("nonce store: %w", err)
	}
	if !ok {
		return ErrNonceAlreadyUsed
	}
	return nil
}

func nonceKey(nonce string) string { return "oauth:nonce:" + nonce }
