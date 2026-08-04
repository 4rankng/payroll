package zaloconnect

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
)

// Settings keys (single source of truth for the two rows this service owns).
const (
	KeyEnabled      = "zalo.enabled"
	KeyCredentials  = "zalo.credentials"
	stateKeyPrefix  = "zalo:oauth:state:"
	defaultTemplate = "617976" // OTP-ZNS-v1; admin may override via credentials.template_id
	stateTTL        = 5 * time.Minute
)

// Credentials is the JSON payload of the zalo.credentials settings row. It is
// the on-disk form; the in-memory form the Provider consumes is zalo.Credentials
// (same shape, separated to keep the persistence layer substitutable).
type Credentials struct {
	AppID        string     `json:"app_id"`
	SecretKey    string     `json:"secret_key,omitempty"`
	TemplateID   string     `json:"template_id"`
	AccessToken  string     `json:"access_token,omitempty"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	LastError    string     `json:"last_error,omitempty"`
}

// Status is the masked, admin-facing view. No secret fields.
type Status struct {
	Enabled     bool       `json:"enabled"`
	Configured  bool       `json:"configured"` // has app_id + secret
	Connected   bool       `json:"connected"`  // has valid access + refresh tokens
	TemplateID  string     `json:"template_id"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	LastError   string     `json:"last_error,omitempty"`
	CallbackURL string     `json:"callback_url"`
}

// EnvSeed is the bootstrap-only seed values read from ZALO_* env vars. Used by
// SeedFromEnvIfEmpty on first boot only.
type EnvSeed struct {
	Enabled    bool
	AppID      string
	SecretKey  string
	TemplateID string
}

// settingsRepo is the narrow subset of domain.SettingsRepository this service
// needs. Defined as an interface so tests can inject a fake. Upsert replaces
// the raw *gorm.DB transaction path used in the first draft — it keeps the
// service DB-agnostic and unit-testable.
type settingsRepo interface {
	GetByKey(ctx context.Context, key string) (*domain.Settings, error)
	Create(ctx context.Context, s *domain.Settings) error
	Update(ctx context.Context, s *domain.Settings) error
}

// Service owns the Zalo connection state + admin OAuth orchestration. It
// implements zalo.CredentialSource so the infra/zalo Provider can read/update
// tokens through it.
type Service struct {
	repo        settingsRepo
	redis       *redis.Client
	callbackURL string
	clk         func() time.Time
	log         *slog.Logger

	// provider is set via SetProvider after construction to break the
	// construction cycle (Service needs Provider for OAuth exchange; Provider
	// needs Service as CredentialSource). It is only used by admin actions
	// (ExchangeCode/RefreshNow), never by the CredentialSource methods.
	provider *zalo.Provider
}

// NewService constructs the service. provider is nil initially — call
// SetProvider after wiring the Provider with this service as its CredentialSource.
func NewService(repo settingsRepo, rdb *redis.Client, callbackURL string, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo:        repo,
		redis:       rdb,
		callbackURL: callbackURL,
		clk:         time.Now,
		log:         log,
	}
}

// SetProvider completes the two-step bootstrap wire. Called once after the
// Provider is constructed with this service as its CredentialSource.
func (s *Service) SetProvider(p *zalo.Provider) { s.provider = p }

// --- zalo.CredentialSource implementation ---------------------------------

func (s *Service) Get(ctx context.Context) (zalo.Credentials, error) {
	c, err := s.loadCredentials(ctx)
	if err != nil {
		return zalo.Credentials{}, err
	}
	return toZaloCreds(c), nil
}

func (s *Service) Update(ctx context.Context, creds zalo.Credentials) error {
	return s.mutateCredentials(ctx, func(cur Credentials) Credentials {
		cur.AccessToken = creds.AccessToken
		cur.RefreshToken = creds.RefreshToken
		cur.ExpiresAt = creds.ExpiresAt
		if creds.AppID != "" {
			cur.AppID = creds.AppID
		}
		if creds.SecretKey != "" {
			cur.SecretKey = creds.SecretKey
		}
		if creds.TemplateID != "" {
			cur.TemplateID = creds.TemplateID
		}
		// A successful token op clears the last error.
		cur.LastError = ""
		return cur
	})
}

// --- admin-facing API ------------------------------------------------------

// GetStatus returns the masked connection status for the admin UI. Never
// includes secret_key, access_token, or refresh_token.
func (s *Service) GetStatus(ctx context.Context) (Status, error) {
	c, _ := s.loadCredentials(ctx) // empty creds on not-found is fine
	enabled, _ := s.IsEnabled(ctx)
	tmpl := c.TemplateID
	if tmpl == "" {
		tmpl = defaultTemplate
	}
	return Status{
		Enabled:     enabled,
		Configured:  c.AppID != "" && c.SecretKey != "",
		Connected:   c.AccessToken != "" && c.RefreshToken != "",
		TemplateID:  tmpl,
		ExpiresAt:   c.ExpiresAt,
		LastError:   c.LastError,
		CallbackURL: s.callbackURL,
	}, nil
}

// SaveCredentialsInput is the admin-supplied connection form payload. Empty
// optional fields mean "keep existing" so the admin can change just one value
// (e.g. re-paste an access_token) without clobbering the others.
type SaveCredentialsInput struct {
	AppID        string // required
	SecretKey    string // empty = keep existing
	TemplateID   string // empty = keep existing/default
	AccessToken  string // empty = keep existing (unless AppID changes)
	RefreshToken string // empty = keep existing (unless AppID changes)
}

// SaveCredentials updates the admin-configured fields. Token lifecycle rules:
//   - If AppID changes, all tokens are cleared (tokens are bound to a specific
//     OA app — a stale token from another app will only produce -124 errors).
//   - If AppID is unchanged, existing tokens are preserved so the admin can
//     edit one field (e.g. re-paste an expired access_token) without forcing a
//     full re-OAuth.
//   - When a non-empty AccessToken is provided AND differs from the current
//     one, ExpiresAt is reset to now + 24h so the Provider treats the freshly
//     pasted token as live (manual paste bypasses OAuth's expires_in).
//
// Empty SecretKey/RefreshToken never overwrite existing values — the UI sends
// empty for password fields the admin did not retype.
func (s *Service) SaveCredentials(ctx context.Context, in SaveCredentialsInput) error {
	return s.mutateCredentials(ctx, func(cur Credentials) Credentials {
		appIDChanged := in.AppID != "" && cur.AppID != "" && in.AppID != cur.AppID

		cur.AppID = in.AppID
		if in.SecretKey != "" {
			cur.SecretKey = in.SecretKey
		}
		if in.TemplateID != "" {
			cur.TemplateID = in.TemplateID
		}
		if cur.TemplateID == "" {
			cur.TemplateID = defaultTemplate
		}

		// Rotating AppID invalidates any existing tokens (app-bound).
		if appIDChanged {
			cur.AccessToken = ""
			cur.RefreshToken = ""
			cur.ExpiresAt = nil
			return cur
		}

		// Manual token paste (AppID unchanged): overwrite only what was supplied.
		// A new AccessToken resets the expiry clock to +24h (matches the Zalo OA
		// dashboard's typical access_token lifetime so the proactive-refresh
		// buffer doesn't immediately fire).
		if in.AccessToken != "" && in.AccessToken != cur.AccessToken {
			exp := s.clk().Add(24 * time.Hour)
			cur.ExpiresAt = &exp
		}
		if in.AccessToken != "" {
			cur.AccessToken = in.AccessToken
		}
		if in.RefreshToken != "" {
			cur.RefreshToken = in.RefreshToken
		}
		// A successful credential update clears any stale last_error.
		cur.LastError = ""
		return cur
	})
}

// SetEnabled flips the runtime feature toggle. Persists immediately; the next
// /auth/zalo-reset/request call reads it (hot toggle, no redeploy).
func (s *Service) SetEnabled(ctx context.Context, enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return s.upsertSetting(ctx, KeyEnabled, val, domain.ValueTypeBoolean)
}

// IsEnabled reads the runtime toggle. Returns false (not an error) if the row
// is missing — the feature is off by default.
func (s *Service) IsEnabled(ctx context.Context) (bool, error) {
	row, err := s.repo.GetByKey(ctx, KeyEnabled)
	if err != nil {
		if isDomainNotFound(err) {
			return false, nil
		}
		return false, err
	}
	if row.Value == nil {
		return false, nil
	}
	return *row.Value == "true", nil
}

// StartOAuth mints a single-use state param, stores it in Redis (5-min TTL),
// and returns the Zalo permission URL the admin's browser should navigate to.
func (s *Service) StartOAuth(ctx context.Context) (string, error) {
	c, err := s.loadCredentials(ctx)
	if err != nil {
		return "", err
	}
	if c.AppID == "" || c.SecretKey == "" {
		return "", errors.New("zalo: cannot start OAuth — save app_id and secret first")
	}
	state, err := randomToken(32)
	if err != nil {
		return "", fmt.Errorf("zalo: generate state: %w", err)
	}
	if err := s.redis.Set(ctx, stateKeyPrefix+state, "1", stateTTL).Err(); err != nil {
		return "", fmt.Errorf("zalo: store state: %w", err)
	}
	// Zalo OA permission URL (OAuth v4). redirect_uri + state travel as query.
	u := fmt.Sprintf("https://oauth.zaloapp.com/v4/permission?app_id=%s&redirect_uri=%s&state=%s",
		url.QueryEscape(c.AppID),
		url.QueryEscape(s.callbackURL),
		url.QueryEscape(state),
	)
	return u, nil
}

// HandleOAuthCallback validates the single-use state, exchanges the code for
// tokens via the Provider, and persists them. The state is consumed (Redis
// GETDEL) before the exchange so a replay is rejected.
func (s *Service) HandleOAuthCallback(ctx context.Context, code, state string) error {
	if code == "" || state == "" {
		return errors.New("zalo: missing code or state")
	}
	// Single-use state: GETDEL.
	n, err := s.redis.Del(ctx, stateKeyPrefix+state).Result()
	if err != nil {
		return fmt.Errorf("zalo: validate state: %w", err)
	}
	if n == 0 {
		return errors.New("zalo: invalid or expired OAuth state (possible replay)")
	}
	if s.provider == nil {
		return errors.New("zalo: provider not wired (SetProvider not called)")
	}
	creds, err := s.provider.ExchangeCode(ctx, code, "")
	if err != nil {
		// Record the error on the credentials row so the admin UI can surface it.
		_ = s.recordError(ctx, err.Error())
		return fmt.Errorf("zalo: exchange code: %w", err)
	}
	// ExchangeCode already persisted via the Provider's CredentialSource.Update.
	_ = creds
	s.log.Info("zalo: OAuth connect completed")
	return nil
}

// RefreshNow forces a token refresh via the Provider (manual "Làm mới token"
// button for debugging). Returns the error if the refresh fails.
func (s *Service) RefreshNow(ctx context.Context) error {
	if s.provider == nil {
		return errors.New("zalo: provider not wired")
	}
	// Trigger a refresh by calling Send with a no-op? Cleaner: expose a refresh
	// method on the Provider. For now, read creds and force-expire so the next
	// op refreshes. Simplest: call the internal path via a zero-cost send.
	// We instead directly invoke the Provider's refresh by reading creds and
	// writing an expired timestamp, but that is hacky. Prefer: Provider exposes
	// Refresh(ctx) — add it.
	return s.provider.RefreshNow(ctx)
}

// SeedFromEnvIfEmpty populates the settings rows from env on first boot ONLY.
// If the zalo.credentials row already exists, this is a no-op (DB authoritative).
func (s *Service) SeedFromEnvIfEmpty(ctx context.Context, seed EnvSeed) error {
	_, err := s.repo.GetByKey(ctx, KeyCredentials)
	if err == nil {
		// Row exists — DB is authoritative, do not overwrite.
		return nil
	}
	if !isDomainNotFound(err) {
		return fmt.Errorf("zalo: check seed: %w", err)
	}
	tmpl := seed.TemplateID
	if tmpl == "" {
		tmpl = defaultTemplate
	}
	creds := Credentials{
		AppID:      seed.AppID,
		SecretKey:  seed.SecretKey,
		TemplateID: tmpl,
	}
	if err := s.writeCredentials(ctx, creds); err != nil {
		return err
	}
	if seed.Enabled {
		if err := s.SetEnabled(ctx, true); err != nil {
			s.log.Warn("zalo: failed to seed enabled flag", "error", err)
		}
	}
	s.log.Info("zalo: seeded credentials from env (first boot)")
	return nil
}

// --- internals -------------------------------------------------------------

func (s *Service) loadCredentials(ctx context.Context) (Credentials, error) {
	row, err := s.repo.GetByKey(ctx, KeyCredentials)
	if err != nil {
		if isDomainNotFound(err) {
			return Credentials{}, nil // not configured yet
		}
		return Credentials{}, err
	}
	if row.Value == nil || *row.Value == "" {
		return Credentials{}, nil
	}
	var c Credentials
	if err := json.Unmarshal([]byte(*row.Value), &c); err != nil {
		return Credentials{}, fmt.Errorf("zalo: decode credentials: %w", err)
	}
	return c, nil
}

// mutateCredentials loads, mutates, and writes the credentials row. Concurrency
// is bounded by the Provider's internal mutex (token refreshes are serialized),
// so a simple read-modify-write through the repo is safe here. A true row lock
// would require a GORM transaction; we avoid it to keep the service DB-agnostic
// and unit-testable, trading off a narrow lost-update window on concurrent
// admin key-saves (an operational rarity) for testability.
func (s *Service) mutateCredentials(ctx context.Context, fn func(Credentials) Credentials) error {
	cur, _ := s.loadCredentials(ctx)
	next := fn(cur)
	return s.writeCredentials(ctx, next)
}

func (s *Service) writeCredentials(ctx context.Context, c Credentials) error {
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("zalo: encode credentials: %w", err)
	}
	val := string(raw)
	return s.upsertSetting(ctx, KeyCredentials, val, domain.ValueTypeJSON)
}

func (s *Service) upsertSetting(ctx context.Context, key, value string, vt domain.SettingsValueType) error {
	row, err := s.repo.GetByKey(ctx, key)
	if err != nil && !isDomainNotFound(err) {
		return fmt.Errorf("zalo: read setting %q: %w", key, err)
	}
	if err == nil {
		// Update existing row.
		row.Value = &value
		row.ValueType = vt
		return s.repo.Update(ctx, row)
	}
	// Create new row.
	row = &domain.Settings{Key: key, Value: &value, ValueType: vt}
	return s.repo.Create(ctx, row)
}

func (s *Service) recordError(ctx context.Context, msg string) error {
	return s.mutateCredentials(ctx, func(cur Credentials) Credentials {
		cur.LastError = msg
		return cur
	})
}

// --- helpers ---------------------------------------------------------------

func toZaloCreds(c Credentials) zalo.Credentials {
	return zalo.Credentials{
		AppID:        c.AppID,
		SecretKey:    c.SecretKey,
		TemplateID:   c.TemplateID,
		AccessToken:  c.AccessToken,
		RefreshToken: c.RefreshToken,
		ExpiresAt:    c.ExpiresAt,
	}
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// isDomainNotFound reports whether err is a domain NotFoundError. Uses the
// domain's own IsNotFoundError helper (errors.Is against ErrNotFound).
func isDomainNotFound(err error) bool {
	return domain.IsNotFoundError(err)
}
