package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	altcha "github.com/altcha-org/altcha-lib-go/v2"
	"github.com/redis/go-redis/v9"
)

// CaptchaConfig controls when CAPTCHA is required and how long challenges live.
type CaptchaConfig struct {
	Enabled      bool          // feature flag (CAPTCHA_ENABLE)
	Threshold    int           // require CAPTCHA after this many consecutive failures (default 3)
	ChallengeTTL time.Duration // how long a captcha challenge stays valid (default 5m)
	HMACSecret   string        // HMAC-SHA256 secret that signs challenges (required when Enabled)
	Cost         int           // PoW difficulty: hash iterations per solve attempt (default 500)
}

const (
	defaultCaptchaThreshold    = 3
	defaultCaptchaChallengeTTL = 5 * time.Minute
	defaultCaptchaCost         = 500
	// redisCaptchaPrefix keys the single-use markers under captcha:used:<signature>.
	redisCaptchaPrefix = "captcha:"
	// maxCaptchaPayloadLen bounds the base64 solution payload accepted at
	// login. A real payload (signed parameters + solution) is a few hundred
	// bytes; anything larger is malformed input, not a solution.
	maxCaptchaPayloadLen = 4096
)

// CaptchaService generates and verifies ALTCHA proof-of-work CAPTCHAs
// (self-hosted, no external service). The browser solves the challenge
// automatically in the background — the user only sees a checkbox that ticks
// itself — which replaces the typed digit image that factory workers
// struggled with on mobile.
//
// Verification is cryptographic (HMAC-signed challenge parameters) plus a
// Redis single-use marker so a solved payload cannot be replayed.
type CaptchaService struct {
	rdb       *redis.Client
	ttl       time.Duration
	enabled   bool
	threshold int
	secret    string
	cost      int
}

// NewCaptchaService constructs the service. The Redis client is reused from the
// app's shared connection.
func NewCaptchaService(rdb *redis.Client, cfg CaptchaConfig) *CaptchaService {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = defaultCaptchaThreshold
	}
	ttl := cfg.ChallengeTTL
	if ttl <= 0 {
		ttl = defaultCaptchaChallengeTTL
	}
	cost := cfg.Cost
	if cost <= 0 {
		cost = defaultCaptchaCost
	}
	return &CaptchaService{
		rdb:       rdb,
		ttl:       ttl,
		enabled:   cfg.Enabled,
		threshold: threshold,
		secret:    cfg.HMACSecret,
		cost:      cost,
	}
}

// Enabled reports whether the CAPTCHA feature flag is on.
func (s *CaptchaService) Enabled() bool { return s.enabled }

// RequiredForFailures reports whether a CAPTCHA is needed given the account's
// current consecutive failure count.
func (s *CaptchaService) RequiredForFailures(failureCount int) bool {
	if !s.enabled {
		return false
	}
	return failureCount >= s.threshold
}

// Generate creates a new proof-of-work challenge for the login widget:
// random salt/nonce parameters, an expiry deadline, and an HMAC signature
// over the parameters so tampered challenges are rejected at verify time.
//
// time.Now (not clock.Now) is deliberate: this is short-lived infrastructure
// expiry, not business time — the altcha library compares expiry against its
// own wall clock internally.
func (s *CaptchaService) Generate(ctx context.Context) (altcha.Challenge, error) {
	expiresAt := time.Now().UTC().Add(s.ttl)
	challenge, err := altcha.CreateChallenge(altcha.CreateChallengeOptions{
		Algorithm:           string(altcha.SHA256),
		DeriveKey:           altcha.DeriveKeySHA(),
		HMACSignatureSecret: s.secret,
		Cost:                s.cost,
		ExpiresAt:           &expiresAt,
	})
	if err != nil {
		return altcha.Challenge{}, fmt.Errorf("captcha challenge: %w", err)
	}
	return challenge, nil
}

// Verify checks the widget's solution payload (base64 JSON of
// {challenge, solution}). Fails closed on every malformed, forged, expired,
// or unsolved payload, and on Redis errors (single-use cannot be enforced).
// A verified solution is consumed exactly once: a Redis SETNX marker keyed by
// the challenge signature rejects replays until the challenge TTL elapses.
func (s *CaptchaService) Verify(ctx context.Context, payloadB64 string) bool {
	if payloadB64 == "" || len(payloadB64) > maxCaptchaPayloadLen {
		return false
	}
	payload, err := decodePayload(payloadB64)
	if err != nil {
		return false
	}
	result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{
		Challenge:           payload.Challenge,
		Solution:            payload.Solution,
		DeriveKey:           altcha.DeriveKeySHA(),
		HMACSignatureSecret: s.secret,
	})
	if err != nil || !result.Verified || result.Expired {
		return false
	}
	used, err := s.rdb.SetNX(ctx, redisCaptchaPrefix+"used:"+payload.Challenge.Signature, 1, s.ttl).Result()
	if err != nil || !used {
		return false
	}
	return true
}

// decodePayload decodes the widget payload: base64 (standard or URL-safe,
// padded or raw) wrapping the JSON {challenge, solution} envelope.
func decodePayload(payloadB64 string) (payload altcha.Payload, err error) {
	var raw []byte
	if raw, err = base64.StdEncoding.DecodeString(payloadB64); err != nil {
		if raw, err = base64.RawURLEncoding.DecodeString(payloadB64); err != nil {
			return payload, fmt.Errorf("captcha payload base64: %w", err)
		}
	}
	if err = json.Unmarshal(raw, &payload); err != nil {
		return payload, fmt.Errorf("captcha payload json: %w", err)
	}
	return payload, nil
}
