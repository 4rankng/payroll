package middleware

import (
	"log"
	"time"

	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	limiterGin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	limiterRedis "github.com/ulule/limiter/v3/drivers/store/redis"
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

// CreateRateLimiter creates a new rate limiter using ulule/limiter with Redis store.
// Each counter is keyed per client IP (see ipKeyGetter above).
func CreateRateLimiter(config RateLimitConfig) gin.HandlerFunc {
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

	// Attach middleware with explicit per-IP key getter and JSON rate-limit response
	return limiterGin.NewMiddleware(instance,
		limiterGin.WithKeyGetter(ipKeyGetter),
		limiterGin.WithLimitReachedHandler(func(c *gin.Context) {
			response.TooManyRequests(c, "Rate limit exceeded. Please try again later.", 60)
		}),
	)
}

// CreateLoginRateLimit creates a rate limiter specifically for login attempts.
// Limit: 10 attempts per minute per IP — tight enough to break credential
// stuffing / default-password sprays, loose enough that a normal user mistyping
// their password a few times is not blocked. Per-account lockout is a separate
// follow-up; this IP cap is the cheap baseline that defeats automated sprays.
func CreateLoginRateLimit(redisURL string) gin.HandlerFunc {
	return CreateRateLimiter(RateLimitConfig{
		Rate:     "10-M",
		RedisURL: redisURL,
	})
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
