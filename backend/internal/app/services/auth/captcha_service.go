package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/mojocn/base64Captcha"
	"github.com/redis/go-redis/v9"
)

// CaptchaConfig controls when CAPTCHA is required and how long codes live.
type CaptchaConfig struct {
	Enabled    bool          // feature flag (CAPTCHA_ENABLED)
	Threshold  int           // require CAPTCHA after this many consecutive failures (default 3)
	CodeTTL    time.Duration // how long a captcha challenge stays valid (default 5m)
	CodeLength int           // number of digits in the code (default 5)
}

const (
	defaultCaptchaThreshold  = 3
	defaultCaptchaCodeTTL    = 5 * time.Minute
	defaultCaptchaCodeLength = 5
	redisCaptchaPrefix       = "captcha:"
)

// CaptchaService generates and verifies image CAPTCHAs using base64Captcha's
// digit driver, with the answer stored in Redis (self-hosted, no external
// service). Digit-only codes are easier for factory workers than letters (no
// ambiguity between uppercase/lowercase/l/I/O/0).
type CaptchaService struct {
	store     base64Captcha.Store
	driver    *base64Captcha.DriverDigit
	ttl       time.Duration
	enabled   bool
	threshold int
}

// NewCaptchaService constructs the service. The Redis client is reused from the
// app's shared connection.
func NewCaptchaService(rdb *redis.Client, cfg CaptchaConfig) *CaptchaService {
	threshold := cfg.Threshold
	if threshold <= 0 {
		threshold = defaultCaptchaThreshold
	}
	length := cfg.CodeLength
	if length <= 0 {
		length = defaultCaptchaCodeLength
	}
	ttl := cfg.CodeTTL
	if ttl <= 0 {
		ttl = defaultCaptchaCodeTTL
	}
	store := newRedisCaptchaStore(rdb, ttl)
	// DriverDigit: number-only captcha. Height/width tuned for mobile display.
	driver := base64Captcha.DriverDigit{
		Height: 80,
		Width:  200,
		Length: length,
		MaxSkew: 0.7,
		DotCount: 80,
	}
	return &CaptchaService{
		store:     store,
		driver:    &driver,
		ttl:       ttl,
		enabled:   cfg.Enabled,
		threshold: threshold,
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

// Generate creates a new CAPTCHA challenge. Returns:
//   - captchaID: opaque id to send back with the login attempt
//   - imageBase64: base64-encoded PNG (the full data:image/png;base64,... string)
func (s *CaptchaService) Generate(ctx context.Context) (captchaID, imageBase64 string, err error) {
	id, content, answer := s.driver.GenerateIdQuestionAnswer()
	item, err := s.driver.DrawCaptcha(content)
	if err != nil {
		return "", "", fmt.Errorf("captcha draw: %w", err)
	}
	if err := s.store.Set(id, answer); err != nil {
		return "", "", fmt.Errorf("captcha store: %w", err)
	}
	return id, item.EncodeB64string(), nil
}

// Verify checks the user's answer against the stored code. The code is
// single-use — it's deleted on verification regardless of correctness (so a
// replay is impossible even if the answer was right).
func (s *CaptchaService) Verify(ctx context.Context, captchaID, code string) bool {
	if captchaID == "" || code == "" {
		return false
	}
	return s.store.Verify(captchaID, code, true)
}

// --- Redis-backed store ---

type redisCaptchaStore struct {
	rdb *redis.Client
	ttl time.Duration
}

func newRedisCaptchaStore(rdb *redis.Client, ttl time.Duration) *redisCaptchaStore {
	return &redisCaptchaStore{rdb: rdb, ttl: ttl}
}

func (s *redisCaptchaStore) Set(id string, value string) error {
	return s.rdb.Set(context.Background(), redisCaptchaPrefix+id, value, s.ttl).Err()
}

func (s *redisCaptchaStore) Get(id string, clear bool) string {
	val, err := s.rdb.Get(context.Background(), redisCaptchaPrefix+id).Result()
	if err != nil {
		return ""
	}
	if clear {
		s.rdb.Del(context.Background(), redisCaptchaPrefix+id)
	}
	return val
}

func (s *redisCaptchaStore) Verify(id, answer string, clear bool) bool {
	stored := s.Get(id, clear)
	return stored != "" && stored == answer
}

var _ base64Captcha.Store = (*redisCaptchaStore)(nil)
