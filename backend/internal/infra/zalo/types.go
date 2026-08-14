package zalo

import (
	"context"
	"time"
)

// Sender is the narrow port the zaloreset service depends on. The concrete
// implementation is *Provider; tests inject a fake.
type Sender interface {
	// Send dispatches one ZNS template message. The phone must already be in
	// domestic form (NormalizePhone is applied inside Send as a safety net).
	// Returns a SendResult whose ErrorCode is the Zalo business code (0 = OK);
	// the Go error is non-nil only for transport/credential-source failures.
	Send(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (SendResult, error)
}

// SendResult is the outcome of a single Send attempt. Business errors from Zalo
// populate ErrorCode/ErrorMsg with the Go error left nil; transport failures
// surface as the Go error.
type SendResult struct {
	MsgID      string `json:"msg_id,omitempty"`
	ErrorCode  int    `json:"error_code"`
	ErrorMsg   string `json:"error_msg"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

// Credentials is the full Zalo OA connection state. The first three fields are
// admin-configured; the last three are OAuth-rotated.
type Credentials struct {
	AppID        string
	SecretKey    string
	TemplateID   string
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}

// HasTokens reports whether OAuth has been completed (access + refresh present).
// Used to decide whether the connection is "configured but not connected".
func (c Credentials) HasTokens() bool {
	return c.AccessToken != "" && c.RefreshToken != ""
}

// Config holds the protocol-level knobs (endpoints + timeouts + refresh buffer).
// These are NOT the OA credentials — those come from CredentialSource.
type Config struct {
	SendURL       string        // https://business.openapi.zalo.me/message/template
	OAuthURL      string        // https://oauth.zaloapp.com/v4/oa/access_token
	RefreshBuffer time.Duration // refresh access_token this far before expiry (default 2h)
	HTTPTimeout   time.Duration // per-request timeout (default 15s)
}

// DefaultConfig returns the production Zalo endpoints with sane timeouts.
func DefaultConfig() Config {
	return Config{
		SendURL:       "https://business.openapi.zalo.me/message/template",
		OAuthURL:      "https://oauth.zaloapp.com/v4/oa/access_token",
		RefreshBuffer: 2 * time.Hour,
		HTTPTimeout:   15 * time.Second,
	}
}

// CredentialSource is the persistence seam. The zaloconnect service implements
// it against the generic settings table; tests inject an in-memory fake. The
// Provider calls Get on every Send and persists successful exchanges through
// an optimistic single-row update because Zalo refresh_tokens are single-use.
type CredentialSource interface {
	Get(ctx context.Context) (Credentials, error)
	// Update stores a fully validated configuration and token pair.
	Update(ctx context.Context, creds Credentials) error
	// UpdateTokens replaces only OAuth-rotated fields, preserving concurrent
	// admin metadata such as App ID, secret, and template.
	UpdateTokens(ctx context.Context, creds Credentials) error
}

// RefreshCoordinator serializes refresh-token exchanges across API processes.
// Acquire must wait until it owns the lease or the context is cancelled. The
// returned release function must delete only the lease owned by that caller.
// Implementations should renew a live lease while the caller owns it.
type RefreshCoordinator interface {
	Acquire(ctx context.Context) (release func(context.Context) error, err error)
}
