package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	App          AppConfig
	DB           DBConfig
	Auth         AuthConfig
	Log          LogConfig
	RateLimit    RateLimitConfig
	Casbin       CasbinConfig
	Redis        RedisConfig
	Asynq        AsynqConfig
	Security     SecurityConfig
	CORS         CORSConfig
	Notification NotificationConfig
	WebSocket    WebSocketConfig
	Scheduler    SchedulerConfig
	Asset        AssetConfig
	Disbursement DisbursementConfig
	OTP          OTPConfig
	Google       GoogleConfig
	// Tenant concurrency limit for per-tenant middleware
	TenantConcurrencyLimit int
	// Request timeout applied to each incoming HTTP request (e.g. "10s")
	RequestTimeout time.Duration
	// Tenant queue worker settings for per-tenant background tasks
	TenantQueueWorkers int
	TenantQueueBuffer  int
}

type AppConfig struct {
	Env  string
	Port string
	// TrustedProxies is a list of CIDRs/IPs whose X-Forwarded-For headers gin
	// will trust when resolving the real client IP (used for rate limiting and
	// audit logging). Defaults to loopback only; set to your nginx/LB CIDR in
	// production via the TRUSTED_PROXIES env var (comma-separated).
	TrustedProxies []string
}

type DBConfig struct {
	Driver string
	DSN    string
	// Connection pool settings
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
	// If true, app expects to be behind a connection pooler like PgBouncer
	UseConnectionPooler bool
}

type AuthConfig struct {
	JWTSecret string
	AccessTTL time.Duration
}

// OTPConfig backs the email-OTP 2FA feature gated behind OTP_ENABLE (default off).
// When Enabled, admin/partner logins require an emailed one-time code; see
// plans/260704-1400-otp-2fa-admin-partner/. No stored secret is required
// (email-OTP generates codes per-login), so unlike a TOTP scheme there is no key.
type OTPConfig struct {
	Enabled        bool
	CodeTTL        time.Duration // pending-session TTL (default 5m)
	MaxAttempts    int           // per-account failed-verify cap (default 5)
	LockDuration   time.Duration // lockout window once cap hit (default 15m)
	ResendCooldown time.Duration // min gap between resend requests (default 30s)
}

// GoogleConfig holds Google OIDC settings. GoogleClientID is the OAuth client
// id_token audience; GoogleOAuthEnabled flips the "Sign in with Google" flow on
// (when off, the endpoint 404s / errors). The client ID is public by design —
// it is not a secret — but is fail-fast-validated at boot so a misconfigured
// deploy fails to start rather than failing at first Google login.
type GoogleConfig struct {
	OAuthEnabled bool
	ClientID     string
}

type LogConfig struct {
	Level string
	File  string
}

type RateLimitConfig struct {
	RPS int
}

type CasbinConfig struct {
	ModelPath  string
	PolicyPath string
}

type RedisConfig struct {
	Addr string
	DB   int
}

type AsynqConfig struct {
	RedisAddr       string
	RedisDB         int
	Concurrency     int
	RetryMax        int
	ShutdownTimeout time.Duration
}

type SecurityConfig struct {
	HashSecret string
	HashSalt   string
}

type CORSConfig struct {
	AllowOrigins     []string
	AllowCredentials bool
}

type NotificationConfig struct {
	EmailEnabled      bool
	FromEmail         string
	FromName          string
	DefaultRecipients []string
	DefaultCC         []string
	DefaultBCC        []string
	ResendAPIKey      string
	SendTimeout       time.Duration
	VAPIDPublicKey    string
	VAPIDPrivateKey   string
	VAPIDSubject      string // e.g. "mailto:admin@tingting.vip"
}

type WebSocketConfig struct {
	Enabled      bool
	Path         string
	Port         string
	AllowOrigins []string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingInterval time.Duration
}

type SchedulerConfig struct {
	Enabled                  bool
	Timezone                 string
	TimesheetReminderEnabled bool
	PayrollReminderEnabled   bool
	NotificationDays         []int // Days of month (9, 16, 23, -1 for last day)
	ReminderDays             []int // T-1 days (8, 15, 22, -2 for last-1 day)
}

type AssetConfig struct {
	StoragePath string
	BaseURL     string
	MaxFileSize int64
	CleanupDays int
}

type DisbursementConfig struct {
	Ninepay NinepayConfig
	Onepay  OnepayConfig
}

func (d DisbursementConfig) EmployeeDisbursementEnabled() bool {
	return (d.Ninepay.Enabled && d.Ninepay.EnabledForEmployee) ||
		(d.Onepay.Enabled && d.Onepay.EnabledForEmployee)
}

// NinepayConfig holds the credentials, endpoint, and feature flags for
// the 9pay disbursement provider. All values are env-only; nothing is
// read from the Settings DB at runtime so a deploy fully describes what
// 9pay can and cannot do.
//
// Two booleans gate the provider:
//
//   - Enabled (ENABLE_NINEPAY) — master switch. When false, the provider
//     is not registered, debug routes 404, and the IPN webhook 404. The
//     entire 9pay surface is dormant.
//   - EnabledForEmployee (ENABLE_NINEPAY_FOR_EMPLOYEE) — only meaningful
//     when Enabled=true. Routes employee/payroll disbursements through
//     9pay vs. the existing flow. When false, 9pay can be exercised via
//     the debug rig in isolation while real money continues through the
//     pre-existing path.
//
// Blank credentials (MerchantKey/SecretKey/SecretKeyChecksum) suppress
// registration even when Enabled=true — boot logs say so but does not
// crash. This lets staging and prod share an env file with credentials
// missing in the lower environment.
type NinepayConfig struct {
	Enabled                bool
	EnabledForEmployee     bool
	EnabledForBulkTransfer bool
	MerchantKey            string
	SecretKey              string
	SecretKeyChecksum      string
	Endpoint               string
	// WebEndpoint is the host serving the 9pay merchant-web (back-office)
	// API — distinct from the payment API in Endpoint. Production
	// be.9pay.vn vs payment.9pay.vn; sandbox sand-be.9pay.vn vs
	// sand-payment.9pay.vn. The reconciliation export endpoints
	// (/api/transaction/disbursement/export, /api/transaction/download)
	// live here. In dev we point both at the local mock.
	WebEndpoint string
	// Merchant-portal OAuth fields used for the merchant-web export
	// endpoints. They authenticate via Laravel session login + OAuth
	// authorization_code grant — distinct from the HMAC scheme used by
	// the disbursement payment API. Empty WebUsername/WebPassword means
	// "OAuth not configured"; the export endpoint will fail closed in
	// that case (and works only against the local mock).
	WebUsername     string
	WebPassword     string
	WebAccountURL   string
	WebDeviceID     string
	WebClientID     string
	WebClientSecret string
	WebRedirectURI  string
	// WebMerchantID is the numeric 9pay merchant ID used as the
	// merchant_id query param on the reconciliation export endpoint
	// (distinct from MerchantKey, which is the alphanumeric HMAC key).
	// Empty means "no filter" — return all transactions visible to the
	// authenticated user. Set to scope exports to a single merchant when
	// the portal account has access to multiple.
	WebMerchantID string
	TPS           int
	// QueueBuffer is the buffered-channel capacity for the transfer queue
	// wrapper. When full, callers receive an immediate error (back-pressure).
	QueueBuffer int
}

// OnepayConfig holds the credentials, endpoint, and feature flags for
// the OnePay disbursement provider. All values are env-only; nothing is
// read from the Settings DB at runtime.
//
// Two booleans gate the provider:
//
//   - Enabled (ENABLE_ONEPAY) — master switch. When false, the provider
//     is not registered and all OnePay routes return 404.
//   - EnabledForEmployee (ENABLE_ONEPAY_FOR_EMPLOYEE) — only meaningful
//     when Enabled=true. Routes employee/payroll disbursements through
//     OnePay vs. the existing flow.
type OnepayConfig struct {
	Enabled                bool
	EnabledForEmployee     bool
	EnabledForBulkTransfer bool
	PartnerID              string
	PartnerKey             string
	AccountID              string
	Endpoint               string
	RequestExpirySeconds   int
	HTTPTimeout            time.Duration
	TPS                    int
	QueueBuffer            int
	AllowedIPs             []string
	// AllowedIPsUseRemoteAddr, when true, makes the IP-whitelist middleware
	// read the client IP from the raw TCP socket (c.Request.RemoteAddr)
	// instead of c.ClientIP(). Required when running behind a reverse
	// proxy that is not in TRUSTED_PROXIES, or when you do not want
	// client-supplied X-Forwarded-For to influence the allow/deny decision.
	AllowedIPsUseRemoteAddr bool
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore errors in production)
	_ = godotenv.Load()

	// Determine environment-aware defaults
	env := getEnv("APP_ENV", "dev")
	storagePath := "./uploads"
	storageBaseURL := "http://localhost:8080/uploads"
	if env == "production" {
		storagePath = "/app/uploads"
		storageBaseURL = "https://tingting.vip/uploads"
	}

	cfg := &Config{
		App: AppConfig{
			Env:            getEnv("APP_ENV", "dev"),
			Port:           getEnv("APP_PORT", "8080"),
			TrustedProxies: parseStringSlice(getEnv("TRUSTED_PROXIES", "127.0.0.1,::1")),
		},
		DB: DBConfig{
			Driver:              getEnv("DB_DRIVER", "mysql"),
			DSN:                 getEnv("DB_DSN", "user:pass@tcp(localhost:3306)/appdb?parseTime=true&loc=Local"),
			MaxIdleConns:        parseInt(getEnv("DB_MAX_IDLE_CONNS", "10")),
			MaxOpenConns:        parseInt(getEnv("DB_MAX_OPEN_CONNS", "100")),
			ConnMaxLifetime:     parseDuration(getEnv("DB_CONN_MAX_LIFETIME", "1h")),
			UseConnectionPooler: parseBool(getEnv("DB_USE_POOLER", "false")),
		},
		Auth: AuthConfig{
			JWTSecret: getEnv("JWT_SECRET", ""),
			AccessTTL: 14 * 24 * time.Hour, // 14 days — hardcoded, not configurable via env
		},
		Log: LogConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			File:  getEnv("LOG_FILE", "logs/app.log"),
		},
		RateLimit: RateLimitConfig{
			RPS: parseInt(getEnv("RATE_LIMIT_RPS", "50")),
		},
		Casbin: CasbinConfig{
			ModelPath:  getEnv("CASBIN_MODEL_PATH", "./configs/casbin_model.conf"),
			PolicyPath: getEnv("CASBIN_POLICY_PATH", "./configs/casbin_policy.csv"),
		},
		Redis: RedisConfig{
			Addr: getEnv("REDIS_ADDR", "localhost:6379"),
			DB:   parseInt(getEnv("REDIS_DB", "0")),
		},
		Asynq: AsynqConfig{
			RedisAddr:       getEnv("ASYNQ_REDIS_ADDR", getEnv("REDIS_ADDR", "localhost:6379")),
			RedisDB:         parseInt(getEnv("ASYNQ_REDIS_DB", "1")),
			Concurrency:     parseInt(getEnv("ASYNQ_CONCURRENCY", "20")),
			RetryMax:        parseInt(getEnv("ASYNQ_RETRY_MAX", "5")),
			ShutdownTimeout: parseDuration(getEnv("ASYNQ_SHUTDOWN_TIMEOUT", "30s")),
		},
		Security: SecurityConfig{
			HashSecret: getEnv("HASH_SECRET", ""),
			HashSalt:   getEnv("HASH_SALT", ""),
		},
		CORS: CORSConfig{
			AllowOrigins:     parseStringSlice(getEnv("CORS_ALLOW_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000,https://tingting.vip,https://www.tingting.vip,https://pay.1stop.app")),
			AllowCredentials: parseBool(getEnv("CORS_ALLOW_CREDENTIALS", "true")),
		},
		Notification: NotificationConfig{
			EmailEnabled:      true,
			FromEmail:         "payroll@1stop.app",
			FromName:          "TingTing",
			DefaultRecipients: []string{},
			DefaultCC:         []string{},
			DefaultBCC:        []string{},
			ResendAPIKey:      getEnv("RESEND_API_KEY", ""),
			SendTimeout:       5 * time.Second,
			VAPIDPublicKey:    getEnv("VAPID_PUBLIC_KEY", ""),
			VAPIDPrivateKey:   getEnv("VAPID_PRIVATE_KEY", ""),
			VAPIDSubject:      getEnv("VAPID_SUBJECT", "mailto:admin@tingting.vip"),
		},
		WebSocket: WebSocketConfig{
			Enabled:      parseBool(getEnv("WEBSOCKET_ENABLED", "true")),
			Path:         getEnv("WEBSOCKET_PATH", "/ws"),
			Port:         getEnv("WEBSOCKET_PORT", "8081"),
			AllowOrigins: parseStringSlice(getEnv("WEBSOCKET_ALLOW_ORIGINS", "http://localhost:3000")),
			ReadTimeout:  parseDuration(getEnv("WEBSOCKET_READ_TIMEOUT", "60s")),
			WriteTimeout: parseDuration(getEnv("WEBSOCKET_WRITE_TIMEOUT", "10s")),
			PingInterval: parseDuration(getEnv("WEBSOCKET_PING_INTERVAL", "54s")),
		},
		Scheduler: SchedulerConfig{
			Enabled:                  parseBool(getEnv("SCHEDULER_ENABLED", "true")),
			Timezone:                 getEnv("SCHEDULER_TIMEZONE", "Asia/Ho_Chi_Minh"),
			TimesheetReminderEnabled: parseBool(getEnv("TIMESHEET_REMINDER_ENABLED", "true")),
			PayrollReminderEnabled:   parseBool(getEnv("PAYROLL_REMINDER_ENABLED", "true")),
			NotificationDays:         parseIntSlice(getEnv("NOTIFICATION_DAYS", "9,16,23,-1")),
			ReminderDays:             parseIntSlice(getEnv("REMINDER_DAYS", "8,15,22,-2")),
		},
		Asset: AssetConfig{
			StoragePath: getEnv("ASSET_STORAGE_PATH", storagePath),
			BaseURL:     getEnv("ASSET_BASE_URL", storageBaseURL),
			MaxFileSize: parseInt64(getEnv("ASSET_MAX_FILE_SIZE", "10485760")), // 10MB default
			CleanupDays: parseInt(getEnv("ASSET_CLEANUP_DAYS", "30")),
		},
		Disbursement: DisbursementConfig{
			Ninepay: newNinepayConfig(env),
			Onepay:  newOnepayConfig(env),
		},
		OTP: OTPConfig{
			Enabled:        parseBool(getEnv("OTP_ENABLE", "false")),
			CodeTTL:        parseDuration(getEnv("OTP_CODE_TTL", "5m")),
			MaxAttempts:    parseInt(getEnv("OTP_MAX_ATTEMPTS", "5")),
			LockDuration:   parseDuration(getEnv("OTP_LOCK_DURATION", "15m")),
			ResendCooldown: parseDuration(getEnv("OTP_RESEND_COOLDOWN", "30s")),
		},
		Google: GoogleConfig{
			OAuthEnabled: parseBool(getEnv("GOOGLE_OAUTH_ENABLED", "true")),
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		},
		TenantConcurrencyLimit: parseInt(getEnv("TENANT_CONCURRENCY_LIMIT", "10")),
		RequestTimeout:         parseDuration(getEnv("REQUEST_TIMEOUT", "10s")),
		TenantQueueWorkers:     parseInt(getEnv("TENANT_QUEUE_WORKERS", "5")),
		TenantQueueBuffer:      parseInt(getEnv("TENANT_QUEUE_BUFFER", "20")),
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// newNinepayConfig builds the NinepayConfig from environment variables.
// When APP_ENV is "development", it auto-enables the provider with mock
// sandbox defaults so local dev works without manual credential setup.
// Explicit env vars always override the dev defaults.
func newNinepayConfig(env string) NinepayConfig {
	isDev := env == "development" || env == "dev"

	ninepayEnabled := getEnv("ENABLE_NINEPAY", "")
	if ninepayEnabled == "" {
		ninepayEnabled = func() string {
			if isDev {
				return "true"
			}
			return "false"
		}()
	}

	defaultEndpoint := "https://payment.9pay.vn"
	// Production merchant-portal hosts (verified via DNS + scraping the
	// SPA at https://business.9pay.vn): the OAuth/SSO host is
	// gc-sso.9pay.vn, the API base for /api/get-token + /api/transaction
	// is gc-be.9pay.vn, and the callback lands on business.9pay.vn. The
	// older "be.9pay.vn" / "account-mcv.9pay.vn" guesses (extrapolated
	// from sandbox naming) DO NOT EXIST in DNS.
	defaultWebEndpoint := "https://gc-be.9pay.vn"
	defaultWebAccountURL := "https://gc-sso.9pay.vn"
	defaultWebRedirectURI := "https://business.9pay.vn/callback"
	// Production OAuth client_id from business.9pay.vn's app.js bundle.
	defaultWebClientID := "9699afca-f36f-4081-8e3c-ecf01aa11f2f"
	if isDev {
		defaultEndpoint = "http://localhost:9001"
		defaultWebEndpoint = "http://localhost:9001"
		defaultWebAccountURL = "https://sand-account-mcv.9pay.vn"
		defaultWebRedirectURI = "https://sand-business.9pay.vn/callback"
		// Sandbox client_id observed on the sand-business.9pay.vn SPA.
		defaultWebClientID = "9687bc2f-69fe-4ef5-b9d2-1fc555d8696b"
	}

	return NinepayConfig{
		Enabled:                parseBool(ninepayEnabled),
		EnabledForEmployee:     parseBool(getEnv("ENABLE_NINEPAY_FOR_EMPLOYEE", "false")),
		EnabledForBulkTransfer: parseBool(getEnv("ENABLE_NINEPAY_FOR_BULK_TRANSFER", "false")),
		MerchantKey:            getEnv("NINEPAY_MERCHANT_KEY", ""),
		SecretKey:              getEnv("NINEPAY_SECRET_KEY", ""),
		SecretKeyChecksum:      getEnv("NINEPAY_SECRET_KEY_CHECKSUM", ""),
		Endpoint:               getEnv("NINEPAY_ENDPOINT", defaultEndpoint),
		WebEndpoint:            getEnv("NINEPAY_BASE_URL", defaultWebEndpoint),
		WebUsername:            getEnv("NINEPAY_WEB_USERNAME", ""),
		WebPassword:            getEnv("NINEPAY_WEB_PASSWORD", ""),
		WebAccountURL:          getEnv("NINEPAY_WEB_ACCOUNT_URL", defaultWebAccountURL),
		WebDeviceID:            getEnv("NINEPAY_WEB_DEVICE_ID", ""),
		WebClientID:            getEnv("NINEPAY_WEB_CLIENT_ID", defaultWebClientID),
		WebClientSecret:        getEnv("NINEPAY_WEB_CLIENT_SECRET", ""),
		WebRedirectURI:         getEnv("NINEPAY_WEB_REDIRECT_URI", defaultWebRedirectURI),
		WebMerchantID:          getEnv("NINEPAY_WEB_MERCHANT_ID", ""),
		TPS:                    parseInt(getEnv("NINEPAY_TPS", "3")),
		QueueBuffer:            parseInt(getEnv("NINEPAY_QUEUE_BUFFER", "100")),
	}
}

// newOnepayConfig builds the OnepayConfig from environment variables.
// When APP_ENV is "development", it auto-enables the provider with
// sandbox defaults so local dev works without manual credential setup.
// Explicit env vars always override the dev defaults.
func newOnepayConfig(env string) OnepayConfig {
	isDev := env == "development" || env == "dev"

	onepayEnabled := getEnv("ENABLE_ONEPAY", "")
	if onepayEnabled == "" {
		if isDev {
			onepayEnabled = "true"
		} else {
			onepayEnabled = "false"
		}
	}

	defaultEndpoint := "https://onepay.vn"
	defaultEnabledForEmployee := "false"
	if isDev {
		defaultEndpoint = "http://localhost:9001"
		defaultEnabledForEmployee = "true"
	}

	return OnepayConfig{
		Enabled:                 parseBool(onepayEnabled),
		EnabledForEmployee:      parseBool(getEnv("ENABLE_ONEPAY_FOR_EMPLOYEE", defaultEnabledForEmployee)),
		EnabledForBulkTransfer:  parseBool(getEnv("ENABLE_ONEPAY_FOR_BULK_TRANSFER", "false")),
		PartnerID:               getEnv("ONEPAY_PARTNER_ID", ""),
		PartnerKey:              getEnv("ONEPAY_PARTNER_KEY", ""),
		AccountID:               getEnv("ONEPAY_ACCOUNT_ID", ""),
		Endpoint:                getEnv("ONEPAY_BASE_URL", defaultEndpoint),
		RequestExpirySeconds:    parseInt(getEnv("ONEPAY_REQUEST_EXPIRY_SECONDS", "3600")),
		HTTPTimeout:             parseDuration(getEnv("ONEPAY_HTTP_TIMEOUT", "30s")),
		TPS:                     parseInt(getEnv("ONEPAY_TPS", "3")),
		QueueBuffer:             parseInt(getEnv("ONEPAY_QUEUE_BUFFER", "100")),
		AllowedIPs:              parseStringSlice(getEnv("ONEPAY_ALLOWED_IPS", "202.9.84.102,202.9.84.103,116.97.110.81,116.97.110.82,116.97.110.83,116.97.110.84,116.97.110.85,116.97.110.86")),
		AllowedIPsUseRemoteAddr: parseBool(getEnv("ONEPAY_ALLOWED_IPS_USE_REMOTE_ADDR", "false")),
	}
}

func (c *Config) validate() error {
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET must be set")
	}

	if c.Security.HashSecret == "" {
		return fmt.Errorf("HASH_SECRET must be set")
	}

	if c.Security.HashSalt == "" {
		return fmt.Errorf("HASH_SALT must be set")
	}

	// Email-OTP 2FA fail-fast: when the feature is enabled, a working email
	// config is mandatory (codes are delivered via Resend). Without this, an
	// enabled-but-misconfigured deploy would silently lock out every admin/partner.
	if c.OTP.Enabled {
		if c.Notification.FromEmail == "" {
			return fmt.Errorf("EMAIL_FROM must be set when OTP_ENABLE=true (needed to deliver codes)")
		}
		if c.Notification.ResendAPIKey == "" {
			return fmt.Errorf("RESEND_API_KEY must be set when OTP_ENABLE=true")
		}
	}

	// Google OIDC fail-fast: when the flow is enabled, the client ID must be set
	// (it is the audience the id_token is bound to). The ID is public, not a
	// secret — but a missing one means Google login silently fails at runtime.
	if c.Google.OAuthEnabled && c.Google.ClientID == "" {
		return fmt.Errorf("GOOGLE_CLIENT_ID must be set when GOOGLE_OAUTH_ENABLED=true")
	}

	// Disbursement mutual-exclusion: at most one provider may handle each
	// routing category at a time. Boot-time fail-fast prevents silent
	// misrouting of funds.
	if c.Disbursement.Onepay.EnabledForEmployee && c.Disbursement.Ninepay.EnabledForEmployee {
		return fmt.Errorf(
			"config: ENABLE_NINEPAY_FOR_EMPLOYEE and ENABLE_ONEPAY_FOR_EMPLOYEE are both true; " +
				"exactly one provider may handle employee disbursements at a time — " +
				"set one to false",
		)
	}
	if c.Disbursement.Onepay.EnabledForBulkTransfer && c.Disbursement.Ninepay.EnabledForBulkTransfer {
		return fmt.Errorf(
			"config: ENABLE_NINEPAY_FOR_BULK_TRANSFER and ENABLE_ONEPAY_FOR_BULK_TRANSFER are both true; " +
				"exactly one provider may handle bulk transfers — " +
				"set one to false",
		)
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func parseInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func parseDuration(s string) time.Duration {
	d, _ := time.ParseDuration(s)
	return d
}

func parseBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func parseStringSlice(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseIntSlice(s string) []int {
	if s == "" {
		return []int{}
	}
	parts := strings.Split(s, ",")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if num, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			result = append(result, num)
		}
	}
	return result
}
