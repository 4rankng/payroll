package ninepay

import (
	"api-server/internal/pkg/clock"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

// portalTokenRefreshBuffer is how early before expiry we refresh the
// cached token. 9pay JWTs live for ~7200s; refreshing 5 minutes early
// gives plenty of slack for a slow round trip.
const portalTokenRefreshBuffer = 5 * time.Minute

// PortalAuthConfig is the input to NewPortalAuth. Required fields:
// AccountURL, WebEndpoint, Username, Password, ClientID, RedirectURI.
type PortalAuthConfig struct {
	// AccountURL is the base URL of the 9pay account host that runs the
	// Laravel session-login + OAuth authorize step. Sandbox:
	// https://sand-account-mcv.9pay.vn; production: https://account-mcv.9pay.vn.
	AccountURL string
	// WebEndpoint is the base URL of the 9pay merchant-web (back-office)
	// API host that exchanges the auth code for a JWT and serves the
	// reconciliation export endpoints. Sandbox: https://sand-be.9pay.vn;
	// production: https://be.9pay.vn.
	WebEndpoint string
	// Username and Password are the merchant-portal credentials.
	Username string
	Password string
	// DeviceID is a stable random 32-hex string per merchant; 9pay's
	// login form requires it, and the /api/get-token call echoes it.
	DeviceID string
	// ClientID is the OAuth client_id used in the authorize redirect.
	// Sandbox observed: 9687bc2f-69fe-4ef5-b9d2-1fc555d8696b. Not used
	// for client authentication — the get-token exchange trusts the
	// recently-authenticated Laravel session.
	ClientID string
	// RedirectURI is the registered OAuth callback URL. Sandbox:
	// https://sand-business.9pay.vn/callback.
	RedirectURI string
}

// PortalAuth obtains and caches JWT bearer tokens for the 9pay
// merchant-web API. Calls to GetBearerToken are safe under
// concurrency; the cached token is reused across requests until
// portalTokenRefreshBuffer before its expiry, at which point a single
// caller refreshes while the rest wait on the mutex.
//
// The merchant-web export endpoints (/api/transaction/disbursement/export
// and /api/transaction/download) authenticate with `Authorization: Bearer
// <jwt>`, distinct from the HMAC signature used by the disbursement
// payment API.
//
// Auth flow (Laravel session login + OAuth authorize + custom token
// exchange — observed by tracing the merchant-portal SPA):
//
//  0. GET  AccountURL/oauth/authorize?...   prime "intended URL" session
//  1. GET  AccountURL/login                 capture _token + cookies
//  2. POST AccountURL/login                 302 back to /oauth/authorize
//  3. GET  AccountURL/oauth/authorize?...   302 to RedirectURI?code=...&state=...
//  4. POST WebEndpoint/api/get-token        JSON {code, state, device} → JWT
//
// Step 4 deliberately does NOT use OAuth2's /oauth/token — the SPA's
// `/api/get-token` endpoint trusts the Laravel session that issued the
// code, so no client_secret is required. We mirror the SPA's wire
// shape exactly.
type PortalAuth struct {
	cfg    PortalAuthConfig
	logger *slog.Logger

	mu     sync.Mutex
	token  string
	expiry time.Time
}

// NewPortalAuth wires a PortalAuth. Returns nil if any required field
// is missing — callers should treat nil as "OAuth not configured" and
// fail closed when the merchant-web endpoints are invoked. The httpClient
// argument is accepted for symmetry with the rest of the package but the
// auth flow uses a private client with a cookie jar and no auto-redirect.
func NewPortalAuth(cfg PortalAuthConfig, _ *http.Client, logger *slog.Logger) *PortalAuth {
	if cfg.AccountURL == "" || cfg.WebEndpoint == "" {
		return nil
	}
	if cfg.Username == "" || cfg.Password == "" || cfg.ClientID == "" || cfg.RedirectURI == "" {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &PortalAuth{
		cfg:    cfg,
		logger: logger.With("component", "ninepay.portal_auth"),
	}
}

// invalidate clears the cached token so the next GetBearerToken call
// re-authenticates. Called by the web client when 9pay returns 401.
func (p *PortalAuth) invalidate() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.token = ""
	p.expiry = time.Time{}
}

// GetBearerToken returns a non-expired JWT, refreshing it if necessary.
// Subsequent calls return the cached token until portalTokenRefreshBuffer
// before expiry.
func (p *PortalAuth) GetBearerToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" && time.Until(p.expiry) > portalTokenRefreshBuffer {
		return p.token, nil
	}
	token, expiry, err := p.runAuthFlow(ctx)
	if err != nil {
		return "", err
	}
	p.token = token
	p.expiry = expiry
	p.logger.Info("portal: token refreshed",
		"expires_in_s", int(time.Until(expiry).Seconds()),
		"refresh_at", expiry.Add(-portalTokenRefreshBuffer).Format(time.RFC3339))
	return p.token, nil
}

// runAuthFlow executes the four-step Laravel + OAuth + get-token flow
// and returns the issued JWT and its absolute expiry time.
func (p *PortalAuth) runAuthFlow(ctx context.Context) (string, time.Time, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("portal: cookie jar: %w", err)
	}
	cli := &http.Client{
		Jar:     jar,
		Timeout: 30 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	accountURL := strings.TrimRight(p.cfg.AccountURL, "/")
	state := generatePortalState()

	if err := p.primeIntendedURL(ctx, cli, accountURL, state); err != nil {
		return "", time.Time{}, fmt.Errorf("portal: step 0 (prime intended url): %w", err)
	}
	csrf, err := p.fetchLoginCSRF(ctx, cli, accountURL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("portal: step 1 (fetch login csrf): %w", err)
	}
	authorizeURL, err := p.submitLogin(ctx, cli, accountURL, csrf)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("portal: step 2 (submit login): %w", err)
	}
	authCode, returnedState, err := p.fetchAuthCode(ctx, cli, authorizeURL)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("portal: step 3 (fetch auth code): %w", err)
	}
	return p.exchangeCodeForToken(ctx, cli, authCode, returnedState)
}

// primeIntendedURL hits /oauth/authorize unauthenticated to populate
// Laravel's session "intended url", so the subsequent /login POST
// redirects back to /oauth/authorize after successful auth. Without
// this priming step Laravel's redirect-after-login lands on the
// account-portal home page rather than /oauth/authorize.
func (p *PortalAuth) primeIntendedURL(ctx context.Context, cli *http.Client, accountURL, state string) error {
	q := url.Values{}
	q.Set("client_id", p.cfg.ClientID)
	q.Set("redirect_uri", p.cfg.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", "")
	q.Set("state", state)
	target := accountURL + "/oauth/authorize?" + q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := cli.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 500 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authorize prime failed: status=%d body=%s",
			resp.StatusCode, truncate(string(body), 256))
	}
	return nil
}

// generatePortalState returns a 16-char hex random string for the OAuth
// state parameter. Not security-critical (we own both ends of the
// flow), but 9pay rejects the request without it.
func generatePortalState() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// loginCSRFRegex matches Laravel's CSRF input. Loose enough to cope
// with arbitrary attribute order.
var loginCSRFRegex = regexp.MustCompile(`name="_token"[^>]*value="([^"]+)"|value="([^"]+)"[^>]*name="_token"`)

func (p *PortalAuth) fetchLoginCSRF(ctx context.Context, cli *http.Client, accountURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accountURL+"/login", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status=%d body=%s", resp.StatusCode, truncate(string(body), 256))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	m := loginCSRFRegex.FindSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("CSRF _token not found in login HTML")
	}
	for i := 1; i < len(m); i++ {
		if len(m[i]) > 0 {
			return string(m[i]), nil
		}
	}
	return "", fmt.Errorf("CSRF _token regex matched but captured no group")
}

func (p *PortalAuth) submitLogin(ctx context.Context, cli *http.Client, accountURL, csrf string) (string, error) {
	form := url.Values{}
	form.Set("_token", csrf)
	form.Set("username", p.cfg.Username)
	form.Set("password", p.cfg.Password)
	if p.cfg.DeviceID != "" {
		form.Set("device_id", p.cfg.DeviceID)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, accountURL+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := cli.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 3 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("expected 3xx redirect after login, got %d body=%s",
			resp.StatusCode, truncate(string(body), 256))
	}
	loc := resp.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("login redirect missing Location header")
	}
	if !strings.Contains(loc, "/oauth/authorize") {
		return "", fmt.Errorf("login redirected to unexpected URL: %s", loc)
	}
	if strings.HasPrefix(loc, "/") {
		loc = accountURL + loc
	}
	return loc, nil
}

// fetchAuthCode follows /oauth/authorize, capturing 302 Locations until
// it sees the redirect_uri (which carries ?code=<auth_code>&state=...).
// Returns both the code and the state so the caller can echo state to
// /api/get-token.
func (p *PortalAuth) fetchAuthCode(ctx context.Context, cli *http.Client, authorizeURL string) (string, string, error) {
	current := authorizeURL
	redirectPrefix := strings.TrimRight(p.cfg.RedirectURI, "/")
	for hop := 0; hop < 8; hop++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, current, nil)
		if err != nil {
			return "", "", err
		}
		req.Header.Set("Accept", "text/html,application/xhtml+xml")
		resp, err := cli.Do(req)
		if err != nil {
			return "", "", err
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode/100 == 3 {
			loc := resp.Header.Get("Location")
			if loc == "" {
				return "", "", fmt.Errorf("authorize: 3xx without Location at %s", current)
			}
			if strings.HasPrefix(loc, "/") {
				u, err := url.Parse(current)
				if err != nil {
					return "", "", err
				}
				loc = u.Scheme + "://" + u.Host + loc
			}
			if strings.HasPrefix(loc, redirectPrefix) {
				return extractCodeState(loc)
			}
			current = loc
			continue
		}
		return "", "", fmt.Errorf("authorize: unexpected status %d at %s body=%s",
			resp.StatusCode, current, truncate(string(body), 256))
	}
	return "", "", fmt.Errorf("authorize: too many redirects (>8); last url=%s", current)
}

func extractCodeState(callbackURL string) (string, string, error) {
	u, err := url.Parse(callbackURL)
	if err != nil {
		return "", "", fmt.Errorf("parse callback url: %w", err)
	}
	code := u.Query().Get("code")
	state := u.Query().Get("state")
	if code == "" {
		return "", "", fmt.Errorf("callback missing ?code")
	}
	return code, state, nil
}

// exchangeCodeForToken posts {code, state, device} to
// WebEndpoint/api/get-token — the SPA-internal endpoint that exchanges
// the freshly issued OAuth auth code for a JWT. No client_secret
// involved; 9pay trusts the recently authenticated Laravel session
// that produced the code.
//
// Response shape (observed): {access_token, refresh_token, info, ...}.
// Token TTL comes from decoding the JWT's `exp` claim, since 9pay does
// not return an `expires_in` field.
func (p *PortalAuth) exchangeCodeForToken(ctx context.Context, cli *http.Client, authCode, state string) (string, time.Time, error) {
	target := strings.TrimRight(p.cfg.WebEndpoint, "/") + "/api/get-token"
	payload, err := json.Marshal(map[string]string{
		"code":   authCode,
		"state":  state,
		"device": p.cfg.DeviceID,
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("marshal get-token body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, strings.NewReader(string(payload)))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("get-token failed: status=%d body=%s",
			resp.StatusCode, truncate(string(body), 512))
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", time.Time{}, fmt.Errorf("decode get-token response: %w (body: %s)",
			err, truncate(string(body), 256))
	}
	if out.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("get-token response missing access_token (body: %s)",
			truncate(string(body), 256))
	}
	expiry, err := jwtExpiry(out.AccessToken)
	if err != nil {
		// Fall back to the documented 7200s TTL if exp can't be decoded.
		p.logger.Warn("portal: could not decode JWT exp; using fallback TTL", "error", err)
		expiry = clock.Now().Add(2 * time.Hour)
	}
	return out.AccessToken, expiry, nil
}

// jwtExpiry decodes the second segment of a JWT (the payload) and
// returns the absolute time corresponding to its `exp` claim.
func jwtExpiry(token string) (time.Time, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, fmt.Errorf("malformed JWT (segments=%d)", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Some encoders pad — try standard URL encoding.
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return time.Time{}, fmt.Errorf("decode JWT payload: %w", err)
		}
	}
	// 9pay encodes exp as a float (e.g. 1778216220.046092), so decode it
	// as float64 — the standard RFC 7519 type is NumericDate which permits
	// fractional seconds.
	var claims struct {
		Exp float64 `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return time.Time{}, fmt.Errorf("unmarshal JWT claims: %w", err)
	}
	if claims.Exp == 0 {
		return time.Time{}, fmt.Errorf("JWT has no exp claim")
	}
	return time.Unix(int64(claims.Exp), 0), nil
}
