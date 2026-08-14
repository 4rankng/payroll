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
	"strconv"
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

// ErrRefreshTokenRejected is returned (wrapped inside ErrRefreshFailed) when
// Zalo invalidates the refresh_token itself with -14014. Send uses it to
// surface the actionable "re-paste a fresh token pair" message instead of the
// misleading send-time -124. Application services also distinguish it from a
// temporary OAuth transport failure when validating newly pasted credentials.
var ErrRefreshTokenRejected = errors.New("zalo: refresh token rejected")

// ErrCredentialsChanged prevents a concurrent admin save from silently
// accepting a candidate that was never validated or persisted.
var ErrCredentialsChanged = errors.New("zalo: credentials changed while waiting for refresh authority")

// Provider is the stateless ZNS protocol client. It depends on a
// CredentialSource for tokens and uses an internal mutex to serialize
// refreshes so two concurrent -124 retries cannot double-spend the
// single-use refresh_token.
type Provider struct {
	creds       CredentialSource
	coordinator RefreshCoordinator
	cfg         Config
	http        *http.Client
	log         *slog.Logger
	mu          sync.Mutex
}

// SetRefreshCoordinator adds cross-process refresh ownership. Bootstrap wires
// the Redis implementation; tests and single-process tools may leave it nil
// and still retain the Provider's in-process mutex.
func (p *Provider) SetRefreshCoordinator(coordinator RefreshCoordinator) {
	p.coordinator = coordinator
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
		newToken, refreshErr := p.refreshLocked(ctx, true, token)
		if refreshErr != nil {
			p.log.Warn("zalo: -124 retry aborted (refresh failed)", "error", refreshErr)
			return res, refreshErr
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
		tok, err := p.refreshLocked(ctx, false, "")
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
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return SendResult{}, fmt.Errorf("zalo: send returned HTTP %d", resp.StatusCode)
	}
	var parsed templateMessageResponse
	if jsonErr := json.Unmarshal(respBody, &parsed); jsonErr != nil {
		return SendResult{}, fmt.Errorf("zalo: send returned malformed JSON (HTTP %d): %w", resp.StatusCode, jsonErr)
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
// lock we re-read creds in case another goroutine just refreshed. When
// rejectedAccessToken is non-empty and no longer matches storage, the waiter
// reuses the winner's token instead of spending the newly rotated refresh token.
//
// force ignores the stored expires_at and always exchanges the refresh_token.
// The reactive callers — a -124 from Zalo (Send) and an explicit admin
// "Kiểm tra kết nối" (RefreshNow) — MUST pass force=true: the stored expires_at
// can be a guessed value (manual paste records +24h, not the token's real
// lifetime), and honoring that guess over Zalo's authoritative "token invalid"
// left a dead token in place and looped on -124 forever. The proactive caller
// (getAccessToken) passes force=false so it short-circuits while the token is
// genuinely still within the refresh buffer.
func (p *Provider) refreshLocked(ctx context.Context, force bool, rejectedAccessToken string) (string, error) {
	return p.withRefreshAuthority(ctx, func() (string, error) {
		creds, err := p.creds.Get(ctx)
		if err != nil {
			return "", fmt.Errorf("zalo: read credentials for refresh: %w", err)
		}
		// This request failed with rejectedAccessToken before waiting for the lock.
		// A different request has since persisted another access token, so that
		// request already advanced the single-use refresh chain. Reuse its result.
		if rejectedAccessToken != "" && creds.AccessToken != "" && creds.AccessToken != rejectedAccessToken {
			return creds.AccessToken, nil
		}
		// Another goroutine or process may have rotated while we waited for the lock.
		if !force && creds.ExpiresAt != nil && time.Until(*creds.ExpiresAt) >= p.cfg.RefreshBuffer {
			return creds.AccessToken, nil
		}
		return p.refresh(ctx, creds)
	})
}

func (p *Provider) withRefreshAuthority(ctx context.Context, fn func() (string, error)) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.coordinator == nil {
		return fn()
	}

	release, err := p.coordinator.Acquire(ctx)
	if err != nil {
		return "", fmt.Errorf("zalo: coordinate token refresh: %w", err)
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if releaseErr := release(releaseCtx); releaseErr != nil {
			p.log.Error("zalo: failed to release refresh lease", "error", releaseErr)
		}
	}()
	return fn()
}

// refresh performs one OAuth refresh_token exchange and persists the new pair.
// Caller must hold p.mu (use refreshLocked from Send/getAccessToken/RefreshNow).
func (p *Provider) refresh(ctx context.Context, creds Credentials) (string, error) {
	next, err := p.exchangeRefreshToken(ctx, creds)
	if err != nil {
		return "", err
	}
	// Persist BEFORE returning, so a crash between refresh and Send retry
	// doesn't leak a consumed refresh_token (R-Z1).
	if err := p.creds.UpdateTokens(ctx, next); err != nil {
		return "", fmt.Errorf("zalo: persist refreshed tokens: %w", err)
	}
	p.log.Info("zalo: access token refreshed",
		"expires_at", next.ExpiresAt.Format(time.RFC3339),
		"refresh_rotated", next.RefreshToken != creds.RefreshToken)
	return next.AccessToken, nil
}

func (p *Provider) exchangeRefreshToken(ctx context.Context, creds Credentials) (Credentials, error) {
	if creds.RefreshToken == "" {
		return Credentials{}, ErrNotConfigured
	}
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"app_id":        {creds.AppID},
		"refresh_token": {creds.RefreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.OAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return Credentials{}, fmt.Errorf("zalo: build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", creds.SecretKey)

	resp, err := p.http.Do(req)
	if err != nil {
		return Credentials{}, fmt.Errorf("%w: %v", ErrRefreshFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var tok oauthTokenResponse
	if jsonErr := json.Unmarshal(respBody, &tok); jsonErr != nil {
		// Zalo signals an already-consumed single-use refresh_token with an
		// EMPTY HTTP-200 body (a well-formed but unknown token instead gets
		// -14014 JSON below). Treat empty as terminal rejection so the admin
		// sees "re-paste a fresh pair" instead of a transient-looking 500.
		if len(bytes.TrimSpace(respBody)) == 0 && resp.StatusCode == http.StatusOK {
			p.log.Error("zalo: refresh returned empty body (refresh token already consumed)",
				"http_status", resp.StatusCode)
			return Credentials{}, fmt.Errorf(
				"%w: %w (zalo: phản hồi rỗng — refresh token đã được dùng, dán cặp token mới)",
				ErrRefreshFailed, ErrRefreshTokenRejected)
		}
		p.log.Error("zalo: refresh returned malformed JSON",
			"http_status", resp.StatusCode,
			"body_len", len(respBody),
			"content_type", resp.Header.Get("Content-Type"))
		return Credentials{}, fmt.Errorf("%w: malformed response", ErrRefreshFailed)
	}
	if tok.Error == ErrInvalidRefreshToken {
		return Credentials{}, fmt.Errorf("%w: %w (zalo -14014: refresh token không hợp lệ)", ErrRefreshFailed, ErrRefreshTokenRejected)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		p.log.Error("zalo: refresh returned non-success HTTP status", "http_status", resp.StatusCode, "zalo_error", tok.Error)
		return Credentials{}, fmt.Errorf("%w: OAuth HTTP %d", ErrRefreshFailed, resp.StatusCode)
	}
	// Zalo must return a new access_token. The refresh_token may be absent —
	// Zalo does not always rotate it, and the existing refresh_token remains
	// valid in that case. Rejecting an access_token-only response (as the old
	// code did) caused the system to discard a perfectly good token and get
	// permanently stuck.
	if tok.AccessToken == "" {
		p.log.Error("zalo: refresh returned no access_token (admin needs to re-paste from OA Console)",
			"zalo_error", tok.Error, "zalo_message", tok.Message, "http_status", resp.StatusCode)
		// -14014 = Zalo invalidated this refresh_token (single-use token already
		// consumed, or expired). Tag it distinctly (wrapped in ErrRefreshFailed so
		// existing errors.Is callers still match) so Send can surface the
		// actionable "re-paste a fresh pair" message instead of the misleading -124.
		return Credentials{}, fmt.Errorf("%w: no access_token in response (zalo error %d: %s)", ErrRefreshFailed, tok.Error, tok.Message)
	}

	expiresAt := time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	if tok.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(time.Hour)
	}

	next := creds
	next.AccessToken = tok.AccessToken
	// Zalo does not always rotate the refresh_token. When it's absent from the
	// response, the existing refresh_token is still valid — keep it.
	if tok.RefreshToken != "" {
		next.RefreshToken = tok.RefreshToken
	}
	next.ExpiresAt = &expiresAt
	return next, nil
}

// ValidateAndStore exchanges an admin-supplied refresh token before accepting
// the credentials as configured. A rejected token therefore fails during save
// instead of remaining hidden until the access token expires about a day later.
func (p *Provider) ValidateAndStore(ctx context.Context, candidate Credentials, observed Credentials) error {
	_, err := p.withRefreshAuthority(ctx, func() (string, error) {
		current, getErr := p.creds.Get(ctx)
		if getErr != nil {
			return "", fmt.Errorf("zalo: read credentials before validation: %w", getErr)
		}
		// Another process changed the pair while this request waited. Reject this
		// candidate as a conflict: silently reusing the winner could falsely
		// report that a different, unvalidated pasted pair was accepted.
		if !sameCredentialGeneration(current, observed) {
			return "", ErrCredentialsChanged
		}
		next, exchangeErr := p.exchangeRefreshToken(ctx, candidate)
		if exchangeErr != nil {
			return "", exchangeErr
		}
		if updateErr := p.creds.Update(ctx, next); updateErr != nil {
			return "", fmt.Errorf("zalo: persist validated credentials: %w", updateErr)
		}
		p.log.Info("zalo: credentials validated and refresh token rotated",
			"expires_at", next.ExpiresAt.Format(time.RFC3339),
			"refresh_rotated", next.RefreshToken != candidate.RefreshToken)
		return next.AccessToken, nil
	})
	return err
}

// StoreConfiguration serializes non-refresh admin edits with token exchanges.
// It rejects stale saves instead of overwriting a configuration that changed
// while the request waited for cross-process refresh authority.
func (p *Provider) StoreConfiguration(ctx context.Context, candidate Credentials, observed Credentials) error {
	_, err := p.withRefreshAuthority(ctx, func() (string, error) {
		current, getErr := p.creds.Get(ctx)
		if getErr != nil {
			return "", fmt.Errorf("zalo: read credentials before save: %w", getErr)
		}
		if !sameCredentialGeneration(current, observed) {
			return "", ErrCredentialsChanged
		}
		if updateErr := p.creds.Update(ctx, candidate); updateErr != nil {
			return "", fmt.Errorf("zalo: persist credentials: %w", updateErr)
		}
		return candidate.AccessToken, nil
	})
	return err
}

func sameCredentialGeneration(current, observed Credentials) bool {
	return current.AppID == observed.AppID &&
		current.SecretKey == observed.SecretKey &&
		current.TemplateID == observed.TemplateID &&
		current.AccessToken == observed.AccessToken &&
		current.RefreshToken == observed.RefreshToken
}

// oauthTokenResponse is the JSON body returned by the OAuth v4 token endpoint
// for both refresh_token and authorization_code grants.
//
// Zalo does NOT always return a new refresh_token — when it doesn't, the
// existing refresh_token is still valid. Callers must keep the old
// refresh_token in that case (see refresh).
type oauthTokenResponse struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"` // may be empty — old token still valid
	ExpiresIn    optionalExpiry `json:"expires_in"`
	Error        flexibleInt64  `json:"error,omitempty"`
	Message      string         `json:"message,omitempty"`
}

// optionalExpiry accepts Zalo's numeric and quoted-numeric expires_in shapes.
// Expiry is advisory: an absent, invalid, non-positive, or duration-overflowing
// value must not discard an otherwise valid rotated token pair. Zero selects
// the conservative one-hour fallback in exchangeRefreshToken.
type optionalExpiry int64

func (e *optionalExpiry) UnmarshalJSON(data []byte) error {
	seconds, err := parseJSONInt64(data)
	const maxDurationSeconds = int64(1<<63-1) / int64(time.Second)
	if err != nil || seconds <= 0 || seconds > maxDurationSeconds {
		*e = 0
		return nil
	}
	*e = optionalExpiry(seconds)
	return nil
}

// flexibleInt64 decodes a JSON number or quoted numeric string for Zalo error
// codes, which have also been observed in both shapes.
type flexibleInt64 int64

func (f *flexibleInt64) UnmarshalJSON(data []byte) error {
	v, err := parseJSONInt64(data)
	if err != nil {
		return errors.New("zalo: non-numeric error code")
	}
	*f = flexibleInt64(v)
	return nil
}

func parseJSONInt64(data []byte) (int64, error) {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		return 0, nil
	}
	if strings.HasPrefix(raw, `"`) {
		var quoted string
		if err := json.Unmarshal(data, &quoted); err != nil {
			return 0, err
		}
		raw = strings.TrimSpace(quoted)
	}
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}

// RefreshNow forces a token refresh regardless of expiry. Used by the admin
// "Kiểm tra kết nối" button — it must actually contact Zalo to validate App ID
// + Secret + Refresh Token, so it forces the exchange ignoring the stored
// expires_at (which may be a guessed value that doesn't reflect real validity).
// Acquires the mutex so it cannot race an in-flight -124 retry. Returns
// ErrNotConfigured (via refresh) when no refresh_token is stored.
func (p *Provider) RefreshNow(ctx context.Context) error {
	creds, err := p.creds.Get(ctx)
	if err != nil {
		return fmt.Errorf("zalo: read credentials before forced refresh: %w", err)
	}
	_, err = p.refreshLocked(ctx, true, creds.AccessToken)
	return err
}
