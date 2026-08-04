package zalo

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ErrNotConfigured is returned when no OAuth tokens are available (admin has
// saved app_id/secret but not yet completed the connect flow).
var ErrNotConfigured = errors.New("zalo: not configured (OAuth not completed)")

// ErrRefreshFailed is returned when a token refresh attempt fails. The caller
// should surface this as a transient error; the stored tokens are unchanged.
var ErrRefreshFailed = errors.New("zalo: token refresh failed")

// Provider is the stateless ZNS protocol client. It depends on a
// CredentialSource for tokens and uses an internal mutex to serialize
// refreshes so two concurrent -124 retries cannot double-spend the
// single-use refresh_token.
type Provider struct {
	creds CredentialSource
	cfg   Config
	http  *http.Client
	log   *slog.Logger
	mu    sync.Mutex
}

// NewProvider constructs a Provider. cfg zero-values are filled from
// DefaultConfig; the HTTP client timeout is cfg.HTTPTimeout (default 15s).
func NewProvider(creds CredentialSource, cfg Config, log *slog.Logger) *Provider {
	def := DefaultConfig()
	if cfg.SendURL == "" {
		cfg.SendURL = def.SendURL
	}
	if cfg.OAuthURL == "" {
		cfg.OAuthURL = def.OAuthURL
	}
	if cfg.RefreshBuffer <= 0 {
		cfg.RefreshBuffer = def.RefreshBuffer
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = def.HTTPTimeout
	}
	if log == nil {
		log = slog.Default()
	}
	return &Provider{
		creds: creds,
		cfg:   cfg,
		http:  &http.Client{Timeout: cfg.HTTPTimeout},
		log:   log,
	}
}

// templateMessageRequest is the JSON body posted to /message/template.
type templateMessageRequest struct {
	Phone        string            `json:"phone"`
	TemplateID   string            `json:"template_id"`
	TemplateData map[string]string `json:"template_data"`
	TrackingID   string            `json:"tracking_id"`
}

// templateMessageResponse is the JSON body returned by /message/template.
type templateMessageResponse struct {
	Error   int    `json:"error"`
	Message string `json:"message"`
	Data    struct {
		MsgID string `json:"msg_id"`
	} `json:"data"`
}

// Send dispatches one ZNS template message. It normalizes the phone, clamps
// params, refreshes the access token if it is about to expire, and retries
// exactly once on error -124 (bad access token).
//
// Business errors from Zalo are returned in SendResult.ErrorCode (Go error nil).
// Transport/credential failures are returned as the Go error.
func (p *Provider) Send(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (SendResult, error) {
	phone = NormalizePhone(phone)
	if phone == "" || templateID == "" {
		return SendResult{ErrorCode: -1, ErrorMsg: "Thiếu SĐT hoặc template_id"}, nil
	}
	data = ClampParams(templateID, data)

	creds, err := p.creds.Get(ctx)
	if err != nil {
		return SendResult{}, fmt.Errorf("zalo: read credentials: %w", err)
	}
	if creds.AppID == "" {
		return SendResult{}, ErrNotConfigured
	}

	token, err := p.getAccessToken(ctx, creds)
	if err != nil {
		return SendResult{}, err
	}

	res, err := p.doSend(ctx, phone, templateID, data, trackingID, token)
	if err != nil {
		return SendResult{}, err // transport error
	}

	// -124: access token is invalid/expired. Refresh once under the lock, retry once.
	if res.ErrorCode == ErrBadAccessToken {
		newToken, refreshErr := p.refreshTokenLocked(ctx)
		if refreshErr != nil {
			p.log.Warn("zalo: -124 retry aborted (refresh failed)", "error", refreshErr)
			return res, nil // surface the original -124 to the caller
		}
		res2, err := p.doSend(ctx, phone, templateID, data, trackingID, newToken)
		if err != nil {
			return SendResult{}, err
		}
		return res2, nil
	}
	return res, nil
}

// getAccessToken returns a live access token, refreshing first if the current
// one expires within RefreshBuffer.
func (p *Provider) getAccessToken(ctx context.Context, creds Credentials) (string, error) {
	if creds.AccessToken == "" {
		return "", ErrNotConfigured
	}
	if creds.ExpiresAt != nil && creds.ExpiresAt.Sub(time.Now()) < p.cfg.RefreshBuffer {
		// About to expire — refresh proactively.
		tok, err := p.refreshTokenLocked(ctx)
		if err != nil {
			// Refresh failed. If the token hasn't actually expired yet, fall
			// back to it (the server may still accept it); otherwise bail.
			if creds.ExpiresAt != nil && creds.ExpiresAt.After(time.Now()) {
				return creds.AccessToken, nil
			}
			return "", err
		}
		return tok, nil
	}
	return creds.AccessToken, nil
}

// doSend performs a single POST to /message/template. It does NOT retry.
func (p *Provider) doSend(ctx context.Context, phone, templateID string, data map[string]string, trackingID, accessToken string) (SendResult, error) {
	body := templateMessageRequest{
		Phone:        phone,
		TemplateID:   templateID,
		TemplateData: data,
		TrackingID:   trackingID,
	}
	raw, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.SendURL, bytes.NewReader(raw))
	if err != nil {
		return SendResult{}, fmt.Errorf("zalo: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", accessToken)

	resp, err := p.http.Do(req)
	if err != nil {
		return SendResult{}, fmt.Errorf("zalo: send: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed templateMessageResponse
	if jsonErr := json.Unmarshal(respBody, &parsed); jsonErr != nil {
		return SendResult{
			ErrorCode:  -1,
			ErrorMsg:   "Phản hồi không phải JSON",
			HTTPStatus: resp.StatusCode,
		}, nil
	}
	return SendResult{
		MsgID:      parsed.Data.MsgID,
		ErrorCode:  parsed.Error,
		ErrorMsg:   ErrorMessage(parsed.Error),
		HTTPStatus: resp.StatusCode,
	}, nil
}

// refreshTokenLocked serializes refresh attempts. Two concurrent -124 retries
// would otherwise both call refresh, burning the single-use refresh_token.
// Under the lock we re-read creds in case another goroutine just refreshed.
func (p *Provider) refreshTokenLocked(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	creds, err := p.creds.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("zalo: read credentials for refresh: %w", err)
	}
	// Another goroutine may have rotated while we waited for the lock.
	if creds.ExpiresAt != nil && creds.ExpiresAt.Sub(time.Now()) >= p.cfg.RefreshBuffer {
		return creds.AccessToken, nil
	}
	return p.refresh(ctx, creds)
}

// refresh performs one OAuth refresh_token exchange and persists the new pair.
// Caller must hold p.mu (use refreshTokenLocked from Send/getAccessToken).
func (p *Provider) refresh(ctx context.Context, creds Credentials) (string, error) {
	if creds.RefreshToken == "" {
		return "", ErrNotConfigured
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"app_id":        {creds.AppID},
		"refresh_token": {creds.RefreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.OAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("zalo: build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", creds.SecretKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrRefreshFailed, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok oauthTokenResponse
	if jsonErr := json.Unmarshal(respBody, &tok); jsonErr != nil {
		return "", fmt.Errorf("%w: malformed response", ErrRefreshFailed)
	}
	if tok.AccessToken == "" || tok.RefreshToken == "" {
		p.log.Error("zalo: refresh returned missing tokens (needs admin re-OAuth)",
			"has_access", tok.AccessToken != "", "has_refresh", tok.RefreshToken != "")
		return "", fmt.Errorf("%w: missing access_token or refresh_token", ErrRefreshFailed)
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if tok.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(time.Hour)
	}

	// Persist BEFORE returning, so a crash between refresh and Send retry
	// doesn't leak a consumed refresh_token (R-Z1).
	next := creds
	next.AccessToken = tok.AccessToken
	next.RefreshToken = tok.RefreshToken
	next.ExpiresAt = &expiresAt
	if err := p.creds.Update(ctx, next); err != nil {
		return "", fmt.Errorf("zalo: persist refreshed tokens: %w", err)
	}
	p.log.Info("zalo: access token refreshed", "expires_at", expiresAt.Format(time.RFC3339))
	return tok.AccessToken, nil
}

// oauthTokenResponse is the JSON body returned by the OAuth v4 token endpoint
// for both refresh_token and authorization_code grants.
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshNow forces a token refresh regardless of expiry. Called by the admin
// "Làm mới token" button (zaloconnect.Service.RefreshNow). Acquires the mutex
// so it cannot race an in-flight -124 retry refresh.
func (p *Provider) RefreshNow(ctx context.Context) error {
	creds, err := p.creds.Get(ctx)
	if err != nil {
		return fmt.Errorf("zalo: read credentials for forced refresh: %w", err)
	}
	if creds.RefreshToken == "" {
		return ErrNotConfigured
	}
	_, err = p.refreshTokenLocked(ctx)
	return err
}

// ExchangeCode trades a one-time authorization code (from the admin OAuth flow)
// for an access + refresh token pair. Called by the zaloconnect service's
// OAuth callback handler. Unlike refresh, this does NOT require the mutex — it
// is only ever invoked from the single admin callback path.
func (p *Provider) ExchangeCode(ctx context.Context, code, codeVerifier string) (Credentials, error) {
	creds, err := p.creds.Get(ctx)
	if err != nil {
		return Credentials{}, fmt.Errorf("zalo: read credentials for exchange: %w", err)
	}
	if creds.AppID == "" || creds.SecretKey == "" {
		return Credentials{}, ErrNotConfigured
	}

	form := url.Values{
		"grant_type": {"authorization_code"},
		"app_id":     {creds.AppID},
		"code":       {code},
	}
	if codeVerifier != "" {
		form.Set("code_verifier", codeVerifier)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.OAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Credentials{}, fmt.Errorf("zalo: build exchange request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", creds.SecretKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return Credentials{}, fmt.Errorf("zalo: exchange code: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok oauthTokenResponse
	if jsonErr := json.Unmarshal(respBody, &tok); jsonErr != nil {
		return Credentials{}, fmt.Errorf("zalo: exchange: malformed response: %w", jsonErr)
	}
	if tok.AccessToken == "" {
		return Credentials{}, fmt.Errorf("zalo: exchange returned no access_token (check app_id/secret/code)")
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if tok.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(time.Hour)
	}

	// Do NOT clobber an existing refresh_token with empty (PHP bug-fix). Zalo
	// may or may not return a new refresh_token on the authorization_code grant.
	next := creds
	next.AccessToken = tok.AccessToken
	if tok.RefreshToken != "" {
		next.RefreshToken = tok.RefreshToken
	}
	next.ExpiresAt = &expiresAt
	if err := p.creds.Update(ctx, next); err != nil {
		return Credentials{}, fmt.Errorf("zalo: persist exchanged tokens: %w", err)
	}
	p.log.Info("zalo: OAuth code exchanged", "expires_in", tok.ExpiresIn, "expires_at", expiresAt.Format(time.RFC3339))
	return next, nil
}
