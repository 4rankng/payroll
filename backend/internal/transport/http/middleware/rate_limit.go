package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiterGin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	limiterRedis "github.com/ulule/limiter/v3/drivers/store/redis"
	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// RateLimitConfig holds configuration for rate limiting using ulule/limiter
type RateLimitConfig struct {
	Rate     string // rate format like "100-M" (100/minute), "1000-H" (1000/hour)
	RedisURL string // Redis connection URL
}

// ipKeyGetter returns the real client IP via gin's trusted-proxy-aware c.ClientIP().
// Using c.ClientIP() instead of the default limiterGin key getter (which reads
// X-Real-IP / X-Forwarded-For directly) ensures the rate limit is scoped per
// actual client IP, not per proxy IP, and honours gin's SetTrustedProxies config.
func ipKeyGetter(c *gin.Context) string {
	return c.ClientIP()
}

// rateLimitReachedHandler writes the 429 response carrying the actual seconds
// until the window resets (not a hard-coded 60). ulule/limiter's gin middleware
// writes the X-RateLimit-* headers as RESPONSE headers (c.Header →
// c.Writer.Header()) just before invoking this handler. NOTE: c.GetHeader()
// reads REQUEST headers (c.Request.Header), so it would never see them — read
// the value back from the response writer instead. Reset is a Unix-seconds
// timestamp.
func rateLimitReachedHandler(c *gin.Context) {
	retryAfter := 60
	if resetStr := c.Writer.Header().Get("X-RateLimit-Reset"); resetStr != "" {
		if resetTs, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			now := clock.Now().Unix()
			if delta := resetTs - now; delta > 0 {
				retryAfter = int(delta)
			}
		}
	}
	response.TooManyRequests(c, constants.MsgRateLimitExceededVN, retryAfter)
}

// createRateLimiterWithKey builds a ulule/limiter middleware keyed by the given
// keyGetter. All rate limiters share this plumbing; only the key function
// differs (per-IP, per-account, etc.).
func createRateLimiterWithKey(config RateLimitConfig, keyGetter func(*gin.Context) string) gin.HandlerFunc {
	// Parse rate from config
	rate, err := limiter.NewRateFromFormatted(config.Rate)
	if err != nil {
		log.Printf("Error parsing rate %s: %v, using default", config.Rate, err)
		// Fallback to a default rate if parsing fails
		rate = limiter.Rate{
			Period: time.Minute,
			Limit:  1000,
		}
	}

	// Create Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: config.RedisURL,
	})

	// Create Redis store for limiter
	store, err := limiterRedis.NewStore(redisClient)
	if err != nil {
		log.Fatalf("Failed to create Redis store for rate limiter: %v", err)
	}

	// Create limiter instance
	instance := limiter.New(store, rate)

	// Attach middleware with the requested key getter and a shared JSON 429 response.
	return limiterGin.NewMiddleware(instance,
		limiterGin.WithKeyGetter(keyGetter),
		limiterGin.WithLimitReachedHandler(rateLimitReachedHandler),
	)
}

// CreateRateLimiter creates a new rate limiter using ulule/limiter with Redis store.
// Each counter is keyed per client IP (see ipKeyGetter above).
func CreateRateLimiter(config RateLimitConfig) gin.HandlerFunc {
	return createRateLimiterWithKey(config, ipKeyGetter)
}

// CreateLoginRateLimit creates a rate limiter specifically for login attempts.
// Limit: 10 attempts per minute per ACCOUNT — keyed by the username/CCCD/mobile
// the caller submitted (see loginAccountKeyGetter), NOT by source IP. Per-account
// keying means a distributed attacker rotating IPs cannot bypass the cap when
// hammering one account. When the body carries no identifier (malformed request,
// or a non-login endpoint mounted on this limiter), the key falls back to the
// client IP so anonymous traffic never collapses into a single global bucket.
func CreateLoginRateLimit(redisURL string) gin.HandlerFunc {
	return createRateLimiterWithKey(RateLimitConfig{
		Rate:     "10-M",
		RedisURL: redisURL,
	}, loginAccountKeyGetter)
}

// loginAccountKeyGetter extracts the account identifier from the login request
// body and returns a per-account limiter key, so the 10/min login cap is
// enforced per account, not per IP. For /auth/login the identifier is the
// submitted username/CCCD/mobile; for the OTP second step (/auth/login/verify,
// /auth/login/resend) it is the otp_session_id, which is bound 1:1 to one
// account for the duration of the OTP flow. The body is read with a small cap
// and restored, so the downstream handler's BindJSON re-reads it normally. When
// the body carries no identifier, the key falls back to the client IP — this
// keeps isolation per-source instead of collapsing every anonymous request into
// one shared (global) bucket.
func loginAccountKeyGetter(c *gin.Context) string {
	const maxBody = 4 << 10 // 4 KiB — login payloads are tiny; bound memory use
	if c.Request != nil && c.Request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody))
		// Always restore the body so the handler can re-read it.
		c.Request.Body = io.NopCloser(bytes.NewReader(raw))
		if err == nil {
			var payload struct {
				Username     string `json:"username"`
				OTPSessionID string `json:"otp_session_id"`
			}
			if json.Unmarshal(raw, &payload) == nil {
				if id := strings.TrimSpace(payload.Username); id != "" {
					if len(id) > 255 {
						id = id[:255]
					}
					return "account:" + strings.ToLower(id)
				}
				if id := strings.TrimSpace(payload.OTPSessionID); id != "" {
					if len(id) > 255 {
						id = id[:255]
					}
					return "otp-session:" + id
				}
			}
		}
	}
	// No identifier in the body — keep per-source isolation (never global).
	return "ip:" + c.ClientIP()
}

// CreateAPIRateLimit creates a rate limiter for general API requests.
// Limit: 10000 requests per minute per IP.
func CreateAPIRateLimit(redisURL string) gin.HandlerFunc {
	return CreateRateLimiter(RateLimitConfig{
		Rate:     "10000-M",
		RedisURL: redisURL,
	})
}

// CreateStrictRateLimit creates a stricter rate limiter for sensitive endpoints.
// Limit: 1000 requests per minute per IP.
func CreateStrictRateLimit(redisURL string) gin.HandlerFunc {
	return CreateRateLimiter(RateLimitConfig{
		Rate:     "1000-M",
		RedisURL: redisURL,
	})
}

// CreateUserRateLimit creates a rate limiter with a custom rate string.
func CreateUserRateLimit(rate string, redisURL string) gin.HandlerFunc {
	return CreateRateLimiter(RateLimitConfig{
		Rate:     rate,
		RedisURL: redisURL,
	})
}

// CreateEndpointRateLimit creates a rate limiter for specific endpoints.
func CreateEndpointRateLimit(rate string, redisURL string) gin.HandlerFunc {
	return CreateRateLimiter(RateLimitConfig{
		Rate:     rate,
		RedisURL: redisURL,
	})
}

// CreatePasswordResetRateLimit caps password-reset requests per EMAIL ADDRESS
// (read from the request body), not per IP — a rotating-IP attacker must not
// bypass the cap. Falls back to the client IP when the body has no email.
// perHour is the allowed requests per normalized email per hour.
func CreatePasswordResetRateLimit(redisURL string, perHour int) gin.HandlerFunc {
	if perHour <= 0 {
		perHour = 3
	}
	rate := fmt.Sprintf("%d-H", perHour)
	return createRateLimiterWithKey(RateLimitConfig{Rate: rate, RedisURL: redisURL}, passwordResetEmailKeyGetter)
}

// passwordResetEmailKeyGetter extracts the email from the request body and
// returns a per-email limiter key. Red Team H1: the email is Unicode case-folded
// + NFKC-normalized + zero-width-stripped so it matches the DB's
// utf8mb4_unicode_ci comparison (Alice@ / Álice@ / alice@\u200b collapse to one
// bucket). Red Team M1: the body is restored after reading or the downstream
// handler's BindJSON sees EOF and 400s every reset request.
func passwordResetEmailKeyGetter(c *gin.Context) string {
	const maxBody = 4 << 10
	if c.Request != nil && c.Request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody))
		c.Request.Body = io.NopCloser(bytes.NewReader(raw)) // CRITICAL: restore body
		if err == nil {
			var payload struct {
				Email string `json:"email"`
			}
			if json.Unmarshal(raw, &payload) == nil {
				if e := strings.TrimSpace(payload.Email); e != "" {
					normalized := normalizeEmailForRateLimit(e)
					if len(normalized) > 255 {
						normalized = normalized[:255]
					}
					return "pwreset-email:" + normalized
				}
			}
		}
	}
	return "ip:" + c.ClientIP()
}

// normalizeEmailForRateLimit applies NFKC normalization + Unicode case-folding +
// zero-width stripping so the limiter key matches MySQL's utf8mb4_unicode_ci
// comparison semantics. strings.ToLower alone is ASCII-only and would let
// "Alice@x" / "Álice@x" / "alice@\u200bx" bypass the per-email cap.
func normalizeEmailForRateLimit(email string) string {
	// NFKC first so compatibility chars fold (e.g. fullwidth ＠ → @).
	n := norm.NFKC.String(email)
	// Unicode case-fold (broader than ASCII ToLower — handles İ, ı, ß, etc.).
	n = cases.Fold().String(n)
	// Strip zero-width chars TrimSpace misses.
	n = strings.Map(func(r rune) rune {
		switch r {
		case '\u200b', '\u200c', '\u200d', '\ufeff': // ZWSP, ZWNJ, ZWJ, BOM
			return -1
		}
		return r
	}, n)
	return strings.TrimSpace(n)
}

// CreateZaloResetRateLimit caps /auth/zalo-reset/request per normalized MOBILE
// number (read from the body), falling back to client IP. Mirrors
// CreatePasswordResetRateLimit but keys on phone digits. perHour is the allowed
// requests per normalized mobile per hour (default 3).
func CreateZaloResetRateLimit(redisURL string, perHour int) gin.HandlerFunc {
	if perHour <= 0 {
		perHour = 3
	}
	rate := fmt.Sprintf("%d-H", perHour)
	return createRateLimiterWithKey(RateLimitConfig{Rate: rate, RedisURL: redisURL}, zaloResetMobileKeyGetter)
}

// zaloResetMobileKeyGetter extracts the mobile from the request body and
// returns a per-mobile limiter key. Phone digits are normalized (strip
// non-digits, drop leading 0/84) so "0987...", "+8498...", "8498...", and
// "0987 654 321" all collapse to one bucket. Body is restored after reading
// (same critical body-restore as passwordResetEmailKeyGetter).
func zaloResetMobileKeyGetter(c *gin.Context) string {
	const maxBody = 4 << 10
	if c.Request != nil && c.Request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody))
		c.Request.Body = io.NopCloser(bytes.NewReader(raw)) // CRITICAL: restore body
		if err == nil {
			var payload struct {
				Mobile string `json:"mobile"`
			}
			if json.Unmarshal(raw, &payload) == nil {
				if m := strings.TrimSpace(payload.Mobile); m != "" {
					return "zreset-mobile:" + normalizeMobileForRateLimit(m)
				}
			}
		}
	}
	return "ip:" + c.ClientIP()
}

// normalizeMobileForRateLimit strips non-digits and drops the leading 0 or 84
// country code so domestic and international forms of the same number share a
// bucket. A number with no digits falls back to the raw trimmed string.
func normalizeMobileForRateLimit(mobile string) string {
	var b strings.Builder
	for _, r := range mobile {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	digits := b.String()
	if digits == "" {
		return mobile
	}
	// Strip leading "84" country code first (so "84987654321" → "987654321"),
	// then any remaining leading "0" (so "0987654321" → "987654321"). The
	// order matters: a number already starting "840..." would lose the 84 then
	// the 0, collapsing to the right bucket.
	digits = strings.TrimPrefix(digits, "84")
	digits = strings.TrimLeft(digits, "0")
	return digits
}

// CreateZaloResetConfirmRateLimit caps /auth/zalo-reset/confirm and /resend per
// otp_session_id (read from the body), falling back to client IP. Bounds
// brute-force guesses against a harvested session id to 10/hour.
func CreateZaloResetConfirmRateLimit(redisURL string) gin.HandlerFunc {
	return createRateLimiterWithKey(RateLimitConfig{Rate: "10-H", RedisURL: redisURL}, zaloResetSessionKeyGetter)
}

// zaloResetSessionKeyGetter keys the confirm/resend limiter on the otp_session_id.
func zaloResetSessionKeyGetter(c *gin.Context) string {
	const maxBody = 4 << 10
	if c.Request != nil && c.Request.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBody))
		c.Request.Body = io.NopCloser(bytes.NewReader(raw)) // CRITICAL: restore body
		if err == nil {
			var payload struct {
				OTPSessionID string `json:"otp_session_id"`
			}
			if json.Unmarshal(raw, &payload) == nil {
				if sid := strings.TrimSpace(payload.OTPSessionID); sid != "" {
					return "zreset-sid:" + sid
				}
			}
		}
	}
	return "ip:" + c.ClientIP()
}
