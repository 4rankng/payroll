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

// ErrNotConfigured is returned when the admin has not yet pasted an
// access_token (and refresh_token) into the connection settings. The admin
// pastes both tokens manually from the Zalo OA Console; there is no OAuth
// authorization-code flow in this app.
var ErrNotConfigured = errors.New("zalo: chưa dán access token — mở Zalo OA Console, lấy token và dán vào Cấu hình kết nối")

// ErrRefreshFailed is returned when a token refresh attempt fails. The caller
// should surface this as a transient error; the stored tokens are unchanged.
var ErrRefreshFailed = errors.New("zalo: token refresh failed")

// errRefreshTokenRejected is returned (wrapped inside ErrRefreshFailed) when
// Zalo invalidates the refresh_token itself with -14014. Send uses it to
// surface the actionable "re-paste a fresh token pair" message instead of the
// misleading send-time -124. Unexported: only this package distinguishes it.
var errRefreshTokenRejected = errors.New("zalo: refresh token rejected")

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

	// -124: access token is invalid/expired — Zalo's authoritative signal that
	// the token is dead. Force a refresh regardless of the stored expires_at
	// (which may be a guessed +24h from manual paste), then retry once.
	if res.ErrorCode == ErrBadAccessToken {
		newToken, refreshErr := p.refreshLocked(ctx, true)
		if refreshErr != nil {
			p.log.Warn("zalo: -124 retry aborted (refresh failed)", "error", refreshErr)
			// A dead refresh_token (-14014) is the real, actionable cause.
			// Surface it instead of the misleading send-time -124 so the admin
			// knows to re-paste a fresh token pair.
			if errors.Is(refreshErr, errRefreshTokenRejected) {
				return SendResult{
					ErrorCode:  ErrInvalidRefreshToken,
					ErrorMsg:   ErrorMessage(ErrInvalidRefreshToken),
					HTTPStatus: res.HTTPStatus,
				}, nil
			}
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
	if creds.ExpiresAt != nil && time.Until(*creds.ExpiresAt) < p.cfg.RefreshBuffer {
		// About to expire — refresh proactively (short-circuits while still valid).
		tok, err := p.refreshLocked(ctx, false)
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
	defer func() { _ = resp.Body.Close() }()

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

// refreshLocked serializes refresh attempts. Two concurrent -124 retries would
// otherwise both call refresh, burning the single-use refresh_token. Under the
// lock we re-read creds in case another goroutine just refreshed.
//
// force ignores the stored expires_at and always exchanges the refresh_token.
// The reactive callers — a -124 from Zalo (Send) and an explicit admin
// "Kiểm tra kết nối" (RefreshNow) — MUST pass force=true: the stored expires_at
// can be a guessed value (manual paste records +24h, not the token's real
// lifetime), and honoring that guess over Zalo's authoritative "token invalid"
// left a dead token in place and looped on -124 forever. The proactive caller
// (getAccessToken) passes force=false so it short-circuits while the token is
// genuinely still within the refresh buffer.
func (p *Provider) refreshLocked(ctx context.Context, force bool) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	creds, err := p.creds.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("zalo: read credentials for refresh: %w", err)
	}
	// Another goroutine may have rotated while we waited for the lock.
	if !force && creds.ExpiresAt != nil && time.Until(*creds.ExpiresAt) >= p.cfg.RefreshBuffer {
		return creds.AccessToken, nil
	}
	return p.refresh(ctx, creds)
}

// refresh performs one OAuth refresh_token exchange and persists the new pair.
// Caller must hold p.mu (use refreshLocked from Send/getAccessToken/RefreshNow).
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
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok oauthTokenResponse
	if jsonErr := json.Unmarshal(respBody, &tok); jsonErr != nil {
		p.log.Error("zalo: refresh returned malformed JSON", "body", string(respBody))
		return "", fmt.Errorf("%w: malformed response", ErrRefreshFailed)
	}
	// Zalo must return a new access_token. The refresh_token may be absent —
	// Zalo does not always rotate it, and the existing refresh_token remains
	// valid in that case. Rejecting an access_token-only response (as the old
	// code did) caused the system to discard a perfectly good token and get
	// permanently stuck.
	if tok.AccessToken == "" {
		p.log.Error("zalo: refresh returned no access_token (admin needs to re-paste from OA Console)",
			"zalo_error", tok.Error, "zalo_message", tok.Message, "body", string(respBody))
		// -14014 = Zalo invalidated this refresh_token (single-use token already
		// consumed, or expired). Tag it distinctly (wrapped in ErrRefreshFailed so
		// existing errors.Is callers still match) so Send can surface the
		// actionable "re-paste a fresh pair" message instead of the misleading -124.
		if tok.Error == ErrInvalidRefreshToken {
			return "", fmt.Errorf("%w: %w (zalo -14014: refresh token không hợp lệ)", ErrRefreshFailed, errRefreshTokenRejected)
		}
		return "", fmt.Errorf("%w: no access_token in response (zalo error %d: %s)", ErrRefreshFailed, tok.Error, tok.Message)
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if tok.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(time.Hour)
	}

	// Persist BEFORE returning, so a crash between refresh and Send retry
	// doesn't leak a consumed refresh_token (R-Z1).
	next := creds
	next.AccessToken = tok.AccessToken
	// Zalo does not always rotate the refresh_token. When it's absent from the
	// response, the existing refresh_token is still valid — keep it.
	if tok.RefreshToken != "" {
		next.RefreshToken = tok.RefreshToken
	}
	next.ExpiresAt = &expiresAt
	if err := p.creds.Update(ctx, next); err != nil {
		return "", fmt.Errorf("zalo: persist refreshed tokens: %w", err)
	}
	p.log.Info("zalo: access token refreshed",
		"expires_at", expiresAt.Format(time.RFC3339),
		"refresh_rotated", tok.RefreshToken != "")
	return tok.AccessToken, nil
}

// oauthTokenResponse is the JSON body returned by the OAuth v4 token endpoint
// for both refresh_token and authorization_code grants.
//
// Zalo does NOT always return a new refresh_token — when it doesn't, the
// existing refresh_token is still valid. Callers must keep the old
// refresh_token in that case (see refresh).
type oauthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"` // may be empty — old token still valid
	ExpiresIn    int64  `json:"expires_in"`
	Error        int    `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
}

// RefreshNow forces a token refresh regardless of expiry. Used by the admin
// "Kiểm tra kết nối" button — it must actually contact Zalo to validate App ID
// + Secret + Refresh Token, so it forces the exchange ignoring the stored
// expires_at (which may be a guessed value that doesn't reflect real validity).
// Acquires the mutex so it cannot race an in-flight -124 retry. Returns
// ErrNotConfigured (via refresh) when no refresh_token is stored.
func (p *Provider) RefreshNow(ctx context.Context) error {
	_, err := p.refreshLocked(ctx, true)
	return err
}
