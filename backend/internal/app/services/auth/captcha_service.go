package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/altcha-org/altcha-lib-go/v2"
	"github.com/redis/go-redis/v9"
)

// CaptchaConfig controls when CAPTCHA is required and how ALTCHA challenges
// are issued.
type CaptchaConfig struct {
	Enabled       bool          // feature flag (CAPTCHA_ENABLE)
	Threshold     int           // require CAPTCHA after this many consecutive failures (default 3)
	ChallengeTTL  time.Duration // how long an ALTCHA challenge (and its solution) stays valid (default 5m)
	HMACSecret    string        // signs challenges and verifies solutions (CAPTCHA_HMAC_SECRET)
	PoWCost       int           // PBKDF2 iterations per derivation (CAPTCHA_POW_COST, default 5000)
	PoWDifficulty int           // expected number of KDF derivations the client performs (CAPTCHA_POW_DIFFICULTY, default 150)
}

const (
	defaultCaptchaThreshold  = 3
	defaultCaptchaCodeTTL    = 5 * time.Minute
	defaultCaptchaPoWCost    = 5000
	defaultCaptchaDifficulty = 150
	redisCaptchaPrefix       = "captcha:"
	// maxCaptchaPayloadSize bounds the base64 payload accepted at login. A
	// real ALTCHA payload is a few hundred bytes; anything larger is refused
	// without parsing (fail-closed, reported as an invalid captcha). Kept below
	// the login rate limiter's 4 KiB body cap so an oversized payload is
	// rejected with the captcha error rather than a body-parse error.
	maxCaptchaPayloadSize = 2048
	// powAlgorithm must match what the official widget bundles by default.
	powAlgorithm = "PBKDF2/SHA-256"
)

// CaptchaService issues ALTCHA proof-of-work challenges and verifies
// solutions. The user only ticks (or watches tick itself) a checkbox; no code
// is typed. Challenges are HMAC-signed, expire after ChallengeTTL, and each
// solution is single-use (consumed in Redis on first successful verification,
// so a replayed payload is rejected).
type CaptchaService struct {
	rdb        *redis.Client
	hmacSecret string
	cost       int
	difficulty int
	ttl        time.Duration
	enabled    bool
	threshold  int
}

// NewCaptchaService constructs the service. The Redis client is reused from the
// app's shared connection.
func NewCaptchaService(rdb *redis.Client, cfg CaptchaConfig) *CaptchaService {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = defaultCaptchaThreshold
	}
	cost := cfg.PoWCost
	if cost <= 0 {
		cost = defaultCaptchaPoWCost
	}
	difficulty := cfg.PoWDifficulty
	if difficulty <= 0 {
		difficulty = defaultCaptchaDifficulty
	}
	ttl := cfg.ChallengeTTL
	if ttl <= 0 {
		ttl = defaultCaptchaCodeTTL
	}
	return &CaptchaService{
		rdb:        rdb,
		hmacSecret: cfg.HMACSecret,
		cost:       cost,
		difficulty: difficulty,
		ttl:        ttl,
		enabled:    cfg.Enabled,
		threshold:  threshold,
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

// Generate creates a new ALTCHA challenge. The challenge is issued in
// deterministic-counter mode: the server hides a counter in [1, 2*difficulty)
// behind a key prefix, so the widget performs ~`difficulty` KDF derivations on
// average before it finds the matching key and the checkbox ticks itself.
// The default difficulty keeps solving within a few seconds on a low-end
// Android device while each derivation costs the client `cost` PBKDF2 rounds.
func (s *CaptchaService) Generate(ctx context.Context) (altcha.Challenge, error) {
	if s.hmacSecret == "" {
		return altcha.Challenge{}, fmt.Errorf("captcha hmac secret not configured")
	}
	// Wall-clock (not clock.Now): the expiry is a Unix second compared by
	// altcha-lib against time.Now internally; epoch time is timezone-free, so
	// the business clock's Asia/Ho_Chi_Minh location is irrelevant here.
	expiresAt := time.Now().Add(s.ttl)
	// counter in [1, 2*difficulty) -> expected `difficulty` derivations.
	counter := 1 + rand.IntN(2*s.difficulty)
	return altcha.CreateChallenge(altcha.CreateChallengeOptions{
		Algorithm:              powAlgorithm,
		DeriveKey:              altcha.DeriveKeyPBKDF2(),
		HMACSignatureSecret:    s.hmacSecret,
		HMACKeySignatureSecret: s.hmacSecret,
		Cost:                   s.cost,
		KeyLength:              32,
		Counter:                &counter,
		ExpiresAt:              &expiresAt,
	})
}

// Verify checks the client's base64 ALTCHA payload: bounded size, decodable,
// correctly signed, not expired, and not replayed. Every failure mode —
// absent, forged, malformed, oversized, expired, verification-failing, or
// replayed — returns false so login rejects with the same captcha error.
// A verified solution is consumed in Redis (SETNX), making it single-use.
func (s *CaptchaService) Verify(ctx context.Context, payloadB64 string) bool {
	if s.hmacSecret == "" || payloadB64 == "" || len(payloadB64) > maxCaptchaPayloadSize {
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(payloadB64)
	if err != nil {
		return false
	}
	var payload altcha.Payload
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return false
	}
	result, err := altcha.VerifySolution(altcha.VerifySolutionOptions{
		Challenge:              payload.Challenge,
		Solution:               payload.Solution,
		DeriveKey:              altcha.DeriveKeyPBKDF2(),
		HMACSignatureSecret:    s.hmacSecret,
		HMACKeySignatureSecret: s.hmacSecret,
	})
	if err != nil || !result.Verified {
		return false
	}
	// Single-use: the payload hash is the replay key; it only needs to live
	// until the challenge expires (after that the payload fails on expiry).
	key := redisCaptchaPrefix + "solved:" + sha256Hex(payloadB64)
	consumed, err := s.rdb.SetNX(ctx, key, 1, s.ttl).Result()
	if err != nil {
		// Redis unavailable: fail closed rather than allow unlimited replay.
		return false
	}
	return consumed
}

// --- helpers ---

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
