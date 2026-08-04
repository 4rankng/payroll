package zaloconnect

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
)

// Settings keys (single source of truth for the two rows this service owns).
const (
	KeyEnabled      = "zalo.enabled"
	KeyCredentials  = "zalo.credentials"
	defaultTemplate = "617976" // OTP-ZNS-v1; admin may override via credentials.template_id
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
	Enabled    bool       `json:"enabled"`
	Configured bool       `json:"configured"` // has app_id + secret
	Connected  bool       `json:"connected"`  // has valid access + refresh tokens
	TemplateID string     `json:"template_id"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastError  string     `json:"last_error,omitempty"`
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

// Service owns the Zalo connection state. The admin pastes all four OA fields
// (app_id, secret, access_token, refresh_token) directly into the settings UI;
// there is no OAuth authorization-code flow. Service implements
// zalo.CredentialSource so the infra/zalo Provider can read/update tokens
// through it (token rotation happens via refresh_token on each Send).
type Service struct {
	repo settingsRepo
	clk  func() time.Time
	log  *slog.Logger

	// provider is set via SetProvider after construction to break the
	// construction cycle (Service needs Provider for RefreshNow/TestSend;
	// Provider needs Service as CredentialSource).
	provider *zalo.Provider
}

// NewService constructs the service. provider is nil initially — call
// SetProvider after wiring the Provider with this service as its CredentialSource.
func NewService(repo settingsRepo, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		repo: repo,
		clk:  time.Now,
		log:  log,
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
		Enabled:    enabled,
		Configured: c.AppID != "" && c.SecretKey != "",
		Connected:  c.AccessToken != "" && c.RefreshToken != "",
		TemplateID: tmpl,
		ExpiresAt:  c.ExpiresAt,
		LastError:  c.LastError,
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
//   - Each field is applied independently — empty fields mean "keep existing"
//     so the admin can edit one value without clobbering the others.
//   - When a non-empty AccessToken is provided AND differs from the current
//     one, ExpiresAt is reset to now + 24h so the Provider treats the freshly
//     pasted token as live (manual paste bypasses OAuth's expires_in).
//   - Rotating AppID no longer force-clears tokens: the admin is manually
//     managing tokens now, and a stale token will surface cleanly as a -124
//     on the next Send (which the Provider retries after refresh). The old
//     clear-on-rotate rule was OAuth-era defense that broke the manual paste
//     UX (any re-save with a placeholder App ID wiped the tokens).
//
// Empty SecretKey/RefreshToken/AccessToken never overwrite existing values —
// the UI sends empty for password fields the admin did not retype.
func (s *Service) SaveCredentials(ctx context.Context, in SaveCredentialsInput) error {
	return s.mutateCredentials(ctx, func(cur Credentials) Credentials {
		if in.AppID != "" {
			cur.AppID = in.AppID
		}
		if in.SecretKey != "" {
			cur.SecretKey = in.SecretKey
		}
		if in.TemplateID != "" {
			cur.TemplateID = in.TemplateID
		}
		if cur.TemplateID == "" {
			cur.TemplateID = defaultTemplate
		}

		// Manual token paste: overwrite only what was supplied. A new
		// AccessToken resets the expiry clock to +24h (matches the Zalo OA
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

// RefreshNow forces a token refresh via the Provider. Used by the admin
// "Kiểm tra kết nối" button — validates App ID + Secret + Refresh Token with
// Zalo's token endpoint WITHOUT sending a ZNS message (useful when the
// template is still pending approval). Returns the error if refresh fails.
func (s *Service) RefreshNow(ctx context.Context) error {
	if s.provider == nil {
		return errors.New("zalo: provider not wired")
	}
	return s.provider.RefreshNow(ctx)
}

// TestSendResult is the admin-facing outcome of a one-off test ZNS send. All
// fields come straight from zalo.SendResult; the struct is separate so the
// handler layer can attach it to a success response without leaking the Go
// error type.
type TestSendResult = zalo.SendResult

// TestSend fires one ZNS template message to a phone using the currently
// stored credentials, without touching the password-reset business flow (no
// OTP stored in Redis, no event published, no rate-limit beyond admin auth).
// This is the admin "Gửi thử" diagnostic — mirrors vfic_zns_preview_send in
// the PHP reference app. Defaults to the OTP template (617976) with sample
// values when the caller doesn't supply template_id/data.
func (s *Service) TestSend(ctx context.Context, phone, templateID string, data map[string]string) (TestSendResult, error) {
	if s.provider == nil {
		return zalo.SendResult{}, errors.New("zalo: provider not wired")
	}
	if templateID == "" {
		templateID = defaultTemplate
	}
	// Default sample data for the OTP template so a bare test request works
	// without the admin having to remember param keys. For other templates,
	// the caller must supply data — missing params surface as Zalo -1122.
	if data == nil && templateID == defaultTemplate {
		data = map[string]string{
			"otp_code":             "000000",
			"user_fullname":        "Test ZNS",
			"otp_valid_in_minutes": "5",
		}
	}
	trackingID := fmt.Sprintf("test_%d_%s", s.clk().Unix(), randomHex(4))
	return s.provider.Send(ctx, phone, templateID, trackingID, data)
}

// randomHex returns 2*n hex characters of cryptographic randomness. Tiny helper
// to avoid pulling in another import for the tracking ID suffix.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "00000000"
	}
	return hex.EncodeToString(b)
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

// isDomainNotFound reports whether err is a domain NotFoundError. Uses the
// domain's own IsNotFoundError helper (errors.Is against ErrNotFound).
func isDomainNotFound(err error) bool {
	return domain.IsNotFoundError(err)
}
