package zalo

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeCreds is an in-memory CredentialSource. It is safe for concurrent use.
type fakeCreds struct {
	mu     sync.Mutex
	cur    Credentials
	getErr error
	updErr error
	reads  int
	writes int
}

type mutexRefreshCoordinator struct {
	mu sync.Mutex
}

func (c *mutexRefreshCoordinator) Acquire(context.Context) (func(context.Context) error, error) {
	c.mu.Lock()
	return func(context.Context) error {
		c.mu.Unlock()
		return nil
	}, nil
}

func (f *fakeCreds) Get(_ context.Context) (Credentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.getErr != nil {
		return Credentials{}, f.getErr
	}
	return f.cur, nil
}

func (f *fakeCreds) Update(_ context.Context, c Credentials) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.writes++
	if f.updErr != nil {
		return f.updErr
	}
	f.cur = c
	return nil
}

func (f *fakeCreds) UpdateTokens(ctx context.Context, c Credentials) error {
	return f.Update(ctx, c)
}

func (f *fakeCreds) snapshot() Credentials {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cur
}

// --- phone.go ---------------------------------------------------------------

func TestNormalizePhone(t *testing.T) {
	cases := []struct{ in, want string }{
		{"0987654321", "84987654321"},
		{"+84987654321", "84987654321"},
		{"84987654321", "84987654321"},
		{"0987 654 321", "84987654321"},
		{"  0987-654-321  ", "84987654321"},
		{"0123", ""},          // too short after normalization
		{"", ""},              // empty
		{"abc", ""},           // no digits
		{"8412345", ""},       // has 84 prefix but rest not 9 digits
		{"+841234567890", ""}, // 84 + 10 digits (too long)
	}
	for _, c := range cases {
		got := NormalizePhone(c.in)
		if got != c.want {
			t.Errorf("NormalizePhone(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- params.go --------------------------------------------------------------

func TestClampParams(t *testing.T) {
	t.Run("known template clamps over-cap fields", func(t *testing.T) {
		longName := strings.Repeat("Nguyễn", 20) // > 30 runes
		out := ClampParams("617976", map[string]string{
			"otp_code":             "123456",
			"user_fullname":        longName,
			"otp_valid_in_minutes": "10",
		})
		if out["otp_code"] != "123456" {
			t.Errorf("otp_code mutated: %q", out["otp_code"])
		}
		if got := len([]rune(out["user_fullname"])); got != 30 {
			t.Errorf("user_fullname not clamped to 30 runes: got %d", got)
		}
		if out["otp_valid_in_minutes"] != "10" {
			t.Errorf("otp_valid_in_minutes mutated: %q", out["otp_valid_in_minutes"])
		}
	})
	t.Run("unknown template passes through", func(t *testing.T) {
		longName := strings.Repeat("x", 50)
		out := ClampParams("999999", map[string]string{"name": longName})
		if out["name"] != longName {
			t.Error("unknown template should not clamp")
		}
	})
	t.Run("salary notification clamps approved parameter caps", func(t *testing.T) {
		out := ClampParams("619686", map[string]string{
			"customer_name": strings.Repeat("x", 31),
			"max_amount":    strings.Repeat("1", 21),
			"expiry_date":   strings.Repeat("2", 21),
		})
		if len([]rune(out["customer_name"])) != 30 || len([]rune(out["max_amount"])) != 20 || len([]rune(out["expiry_date"])) != 20 {
			t.Errorf("salary notification parameters were not clamped: %+v", out)
		}
	})
	t.Run("nil input", func(t *testing.T) {
		if out := ClampParams("617976", nil); out != nil {
			t.Errorf("expected nil, got %v", out)
		}
	})
}

// --- errors.go --------------------------------------------------------------

func TestErrorMessage(t *testing.T) {
	// Spot-check representative codes from each category.
	cases := []struct {
		code int
		want string // substring that must appear
	}{
		{ErrOK, "Thành công"},
		{ErrUnknown, "không xác định"},
		{ErrInvalidPhone, "điện thoại"},
		{ErrInsufficientBal, "số dư"},
		{ErrNoZaloAccount, "chưa liên kết"},
		{ErrOANoPermission, "cấp quyền"},
		{ErrBadAccessToken, "token"},
		{ErrInvalidRefreshToken, "Refresh token"},
		{ErrTemplateTestOnly, "quản trị viên"},
		{ErrDailyQuotaPhone, "giới hạn"},
		{ErrBadTemplateID, "Template ID"},
		{ErrTemplateNotAppr, "phê duyệt"},
		{ErrMissingParam, "thiếu tham số"},
		{ErrParamOverCap, "giới hạn ký tự"},
		{-99999, "-99999"}, // unknown → generic
	}
	for _, c := range cases {
		got := ErrorMessage(c.code)
		if !strings.Contains(got, c.want) {
			t.Errorf("ErrorMessage(%d) = %q, want substring %q", c.code, got, c.want)
		}
	}
}

// --- provider.go: Send ------------------------------------------------------

// zaloMock stands up both the send and oauth endpoints and records calls.
type zaloMock struct {
	t            *testing.T
	sendResp     string // JSON body to return from /message/template
	oauthResp    string // JSON body to return from /oauth/v4/oa/access_token
	sendCalls    int
	oauthCalls   int
	mu           sync.Mutex
	sendHandler  http.HandlerFunc
	oauthHandler http.HandlerFunc
}

func newZaloMock(t *testing.T) *zaloMock {
	m := &zaloMock{t: t}
	m.sendResp = `{"error":0,"message":"Thành công","data":{"msg_id":"m1"}}`
	m.oauthResp = `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`
	m.rebuildHandlers()
	return m
}

func (m *zaloMock) rebuildHandlers() {
	m.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.sendCalls++
		m.mu.Unlock()
		// Verify request shape.
		if r.Header.Get("access_token") == "" {
			m.t.Error("send: missing access_token header")
		}
		var body templateMessageRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Phone == "" || body.TemplateID == "" {
			m.t.Error("send: missing phone or template_id in body")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, m.sendResp)
	}
	m.oauthHandler = func(w http.ResponseWriter, r *http.Request) {
		m.mu.Lock()
		m.oauthCalls++
		m.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, m.oauthResp)
	}
}

func (m *zaloMock) counts() (int, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCalls, m.oauthCalls
}

func (m *zaloMock) setSendError(code int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendResp = `{"error":` + itoa(code) + `,"message":"err"}`
}
func (m *zaloMock) setOAuthAccessTokenOnly() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Zalo does not always rotate the refresh_token — this response has only
	// a new access_token. The Provider must accept it and keep the old one.
	m.oauthResp = `{"access_token":"new-access","expires_in":3600}`
}
func (m *zaloMock) setOAuthMissingTokens() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oauthResp = `{"access_token":"","refresh_token":""}`
}

func (m *zaloMock) setOAuthInvalidRefreshToken() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Zalo's terminal -14014: the refresh_token is dead (single-use, already
	// consumed, or expired). No access_token comes back.
	m.oauthResp = `{"error":-14014,"error_name":"Invalid refresh token.","message":"","access_token":"","refresh_token":""}`
}

func (m *zaloMock) setOAuthEmptyBody() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Zalo answers an ALREADY-CONSUMED refresh_token with an empty HTTP-200
	// body (observed in production 2026-08-14; a merely unknown token gets
	// -14014 JSON instead). The Provider must classify this as terminal
	// rejection, not a transient malformed-response failure.
	m.oauthResp = ""
}

func (m *zaloMock) setOAuthHTMLError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// A non-JSON gateway page is transient infrastructure noise.
	m.oauthResp = `<html><body>504 Gateway Timeout</body></html>`
}

func (m *zaloMock) setOAuthResponse(response string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.oauthResp = response
}

func (m *zaloMock) setOAuthStringErrorCode() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Zalo sometimes reports the numeric error code as a quoted string.
	m.oauthResp = `{"error":"-14014","error_name":"Invalid refresh token.","access_token":"","refresh_token":""}`
}

func newProviderWithMock(t *testing.T) (*Provider, *zaloMock, *fakeCreds, *httptest.Server) {
	mock := newZaloMock(t)
	mux := http.NewServeMux()
	// Register dispatching closures so later swaps of mock.sendHandler/oauthHandler
	// are observed by the running server (important for the -124 retry test).
	mux.HandleFunc("/message/template", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		h := mock.sendHandler
		mock.mu.Unlock()
		h(w, r)
	})
	mux.HandleFunc("/oa/access_token", func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		h := mock.oauthHandler
		mock.mu.Unlock()
		h(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	creds := &fakeCreds{cur: Credentials{
		AppID:        "app1",
		SecretKey:    "secret1",
		TemplateID:   "617976",
		AccessToken:  "live-access",
		RefreshToken: "live-refresh",
		ExpiresAt:    ptrTime(time.Now().Add(24 * time.Hour)), // not expiring soon
	}}
	cfg := Config{
		SendURL:       srv.URL + "/message/template",
		OAuthURL:      srv.URL + "/oa/access_token",
		RefreshBuffer: 2 * time.Hour,
		HTTPTimeout:   5 * time.Second,
	}
	p := NewProvider(creds, cfg, nil)
	return p, mock, creds, srv
}

func ptrTime(t time.Time) *time.Time { return &t }

func TestProvider_Send_Happy(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	res, err := p.Send(context.Background(), "0987654321", "617976", "track-1", map[string]string{
		"otp_code":             "123456",
		"user_fullname":        "Nguyễn Văn A",
		"otp_valid_in_minutes": "10",
	})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Errorf("ErrorCode = %d, want 0 (%s)", res.ErrorCode, res.ErrorMsg)
	}
	if res.MsgID != "m1" {
		t.Errorf("MsgID = %q, want m1", res.MsgID)
	}
	sends, oauth := mock.counts()
	if sends != 1 {
		t.Errorf("send calls = %d, want 1", sends)
	}
	if oauth != 0 {
		t.Errorf("oauth calls = %d, want 0 (token not expiring)", oauth)
	}
}

func TestProvider_Send_RetriesOnBadAccessToken(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	// First send returns -124; the mock then flips to success for the retry.
	mock.mu.Lock()
	first := true
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		if first {
			first = false
			mock.mu.Unlock()
			_, _ = io.WriteString(w, `{"error":-124,"message":"bad token"}`)
			return
		}
		mock.mu.Unlock()
		_, _ = io.WriteString(w, `{"error":0,"data":{"msg_id":"m2"}}`)
	}
	mock.mu.Unlock()

	res, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Errorf("after retry ErrorCode = %d (%s)", res.ErrorCode, res.ErrorMsg)
	}
	if res.MsgID != "m2" {
		t.Errorf("MsgID = %q, want m2", res.MsgID)
	}
	sends, _ := mock.counts()
	if sends != 2 {
		t.Errorf("send calls = %d, want 2 (initial + retry)", sends)
	}
}

// TestProvider_Send_Minus124ForcesRefreshWithFutureExpiry is the core
// auto-refresh regression: when Zalo returns -124 (authoritative "access token
// invalid"), the Provider MUST exchange the refresh_token even if the stored
// expires_at is far in the future. Before the fix the refresh path short-
// circuited on the future expires_at, retried the send with the same dead
// token, and looped on -124 — so a manually-pasted token (expires_at = guessed
// +24h) could never self-heal. This is what keeps the chain "always alive".
func TestProvider_Send_Minus124ForcesRefreshWithFutureExpiry(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	// Credentials carry a FAR-FUTURE expiry (exactly what manual paste produces)
	// and a dead access token.
	creds.mu.Lock()
	creds.cur.AccessToken = "dead-access"
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(24 * time.Hour))
	creds.mu.Unlock()

	// First send returns -124 (dead token); the mock then serves success.
	mock.mu.Lock()
	first := true
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		if first {
			first = false
			mock.mu.Unlock()
			_, _ = io.WriteString(w, `{"error":-124,"message":"bad token"}`)
			return
		}
		mock.mu.Unlock()
		_, _ = io.WriteString(w, `{"error":0,"data":{"msg_id":"m2"}}`)
	}
	mock.mu.Unlock()

	res, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if res.ErrorCode != 0 {
		t.Fatalf("after forced refresh + retry, ErrorCode = %d (%s)", res.ErrorCode, res.ErrorMsg)
	}
	sends, oauth := mock.counts()
	if sends != 2 {
		t.Errorf("send calls = %d, want 2 (failed + retried)", sends)
	}
	if oauth != 1 {
		t.Errorf("oauth calls = %d, want 1 — -124 must force a refresh despite the future expires_at", oauth)
	}
	// The persisted token must be the refreshed one, proving the chain advanced.
	after := creds.snapshot()
	if after.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access (refresh must persist)", after.AccessToken)
	}
}

func TestProvider_Send_ConcurrentMinus124CoalescesRefresh(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)

	// Hold both requests until they have sent the same stale access token. The
	// first request then rotates the credential pair while the second waits on
	// Provider.mu. The waiter must reuse that freshly persisted access token,
	// not immediately spend the replacement refresh token again.
	staleRequests := make(chan struct{}, 2)
	releaseStale := make(chan struct{})
	mock.mu.Lock()
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		mock.mu.Unlock()
		if r.Header.Get("access_token") == "live-access" {
			staleRequests <- struct{}{}
			<-releaseStale
			_, _ = io.WriteString(w, `{"error":-124,"message":"bad token"}`)
			return
		}
		_, _ = io.WriteString(w, `{"error":0,"data":{"msg_id":"m2"}}`)
	}
	mock.mu.Unlock()

	type outcome struct {
		result SendResult
		err    error
	}
	outcomes := make(chan outcome, 2)
	for range 2 {
		go func() {
			result, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
			outcomes <- outcome{result: result, err: err}
		}()
	}
	<-staleRequests
	<-staleRequests
	close(releaseStale)

	for range 2 {
		got := <-outcomes
		if got.err != nil {
			t.Fatalf("Send err: %v", got.err)
		}
		if got.result.ErrorCode != 0 {
			t.Fatalf("ErrorCode = %d (%s), want success", got.result.ErrorCode, got.result.ErrorMsg)
		}
	}
	_, oauth := mock.counts()
	if oauth != 1 {
		t.Fatalf("oauth calls = %d, want 1 shared refresh for the rejected access token", oauth)
	}
}

func TestProvider_Send_TwoProvidersShareOneRefreshAuthority(t *testing.T) {
	p1, mock, creds, _ := newProviderWithMock(t)
	p2 := NewProvider(creds, p1.cfg, nil)
	p2.http = p1.http
	coordinator := &mutexRefreshCoordinator{}
	p1.SetRefreshCoordinator(coordinator)
	p2.SetRefreshCoordinator(coordinator)

	oldTokenArrived := make(chan struct{}, 2)
	releaseOldResponses := make(chan struct{})
	mock.mu.Lock()
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		mock.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.Header.Get("access_token") == "live-access" {
			oldTokenArrived <- struct{}{}
			<-releaseOldResponses
			_, _ = io.WriteString(w, `{"error":-124,"message":"expired"}`)
			return
		}
		_, _ = io.WriteString(w, `{"error":0,"message":"ok","data":{"msg_id":"m2"}}`)
	}
	mock.mu.Unlock()

	results := make(chan SendResult, 2)
	errs := make(chan error, 2)
	for _, provider := range []*Provider{p1, p2} {
		go func(p *Provider) {
			result, err := p.Send(context.Background(), "0987654321", "617976", "track", map[string]string{"otp": "123456"})
			results <- result
			errs <- err
		}(provider)
	}
	<-oldTokenArrived
	<-oldTokenArrived
	close(releaseOldResponses)

	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("Send error: %v", err)
		}
		if result := <-results; result.ErrorCode != 0 {
			t.Fatalf("Send result = %+v, want success", result)
		}
	}
	_, oauthCalls := mock.counts()
	if oauthCalls != 1 {
		t.Fatalf("OAuth calls = %d, want 1 across Provider instances", oauthCalls)
	}
	if got := creds.snapshot(); got.AccessToken != "new-access" || got.RefreshToken != "new-refresh" {
		t.Fatalf("stored credentials = %+v, want rotated pair", got)
	}
}

func TestProvider_StoreConfigurationRejectsStaleMetadataSave(t *testing.T) {
	observed := Credentials{
		AppID: "app", SecretKey: "secret", TemplateID: "template-a",
		AccessToken: "access", RefreshToken: "refresh",
	}
	creds := &fakeCreds{cur: observed}
	provider := NewProvider(creds, Config{}, nil)
	first := observed
	first.TemplateID = "template-b"
	if err := provider.StoreConfiguration(context.Background(), first, observed); err != nil {
		t.Fatalf("first StoreConfiguration: %v", err)
	}
	stale := observed
	stale.TemplateID = "template-c"
	if err := provider.StoreConfiguration(context.Background(), stale, observed); !errors.Is(err, ErrCredentialsChanged) {
		t.Fatalf("stale StoreConfiguration error = %v, want ErrCredentialsChanged", err)
	}
	if got := creds.snapshot().TemplateID; got != "template-b" {
		t.Fatalf("stored TemplateID = %q, want first writer's template-b", got)
	}
}

func TestProvider_Send_NoRetryOnBusinessError(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	mock.setSendError(ErrNoZaloAccount) // -118
	res, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	if res.ErrorCode != ErrNoZaloAccount {
		t.Errorf("ErrorCode = %d, want %d", res.ErrorCode, ErrNoZaloAccount)
	}
	sends, _ := mock.counts()
	if sends != 1 {
		t.Errorf("send calls = %d, want 1 (no retry on -118)", sends)
	}
}

func TestProvider_Send_ReturnsSanitizedErrorForUpstreamBadGateway(t *testing.T) {
	p, mock, _, _ := newProviderWithMock(t)
	mock.mu.Lock()
	mock.sendHandler = func(w http.ResponseWriter, r *http.Request) {
		mock.mu.Lock()
		mock.sendCalls++
		mock.mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, "<html><body>502 Bad Gateway</body></html>")
	}
	mock.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "619686", "t", map[string]string{
		"customer_name": "Nhân viên kiểm thử",
		"max_amount":    "1000000",
		"expiry_date":   "12/08/2026",
	})
	if err == nil {
		t.Fatal("expected an upstream 502 to be a retryable transport error")
	}
	if !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("error = %q, want HTTP status context", err)
	}
	if strings.Contains(err.Error(), "<html") {
		t.Fatalf("error leaked upstream HTML: %q", err)
	}
}

func TestProvider_Send_NotConfiguredWhenNoTokens(t *testing.T) {
	creds := &fakeCreds{cur: Credentials{AppID: "app1", SecretKey: "s", TemplateID: "617976"}} // no tokens
	p := NewProvider(creds, Config{SendURL: "http://x", OAuthURL: "http://y", HTTPTimeout: time.Second}, nil)
	_, err := p.Send(context.Background(), "0987654321", "617976", "t", nil)
	if err != ErrNotConfigured {
		t.Errorf("err = %v, want ErrNotConfigured", err)
	}
}

// --- provider.go: refresh ---------------------------------------------------

func TestProvider_Refresh_EmptyBodyClassifiedAsRejectedToken(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setOAuthEmptyBody()
	before := creds.snapshot()

	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // already expired
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if !errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("Send err = %v, want ErrRefreshTokenRejected (empty body = consumed refresh token)", err)
	}
	if after := creds.snapshot(); after.RefreshToken != before.RefreshToken {
		t.Errorf("refresh_token clobbered: was %q, now %q", before.RefreshToken, after.RefreshToken)
	}
}

func TestProvider_Refresh_HTMLBodyStaysTransientRefreshFailure(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setOAuthHTMLError()

	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // already expired
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if !errors.Is(err, ErrRefreshFailed) {
		t.Fatalf("Send err = %v, want ErrRefreshFailed", err)
	}
	if errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("HTML gateway page must not be classified as a rejected refresh token: %v", err)
	}
	if after := creds.snapshot(); after.RefreshToken != "live-refresh" {
		t.Fatalf("RefreshToken = %q, should be preserved", after.RefreshToken)
	}
}

func TestProvider_Refresh_EmptyNonSuccessBodyStaysTransient(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.mu.Lock()
	mock.oauthHandler = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}
	mock.mu.Unlock()

	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute))
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if !errors.Is(err, ErrRefreshFailed) {
		t.Fatalf("Send err = %v, want ErrRefreshFailed", err)
	}
	if errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("empty non-success response must remain transient: %v", err)
	}
}

func TestProvider_Refresh_NoAccessTokenRejected(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setOAuthMissingTokens()
	before := creds.snapshot() // has live-refresh

	// Force a refresh path by expiring the token.
	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // already expired
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err == nil {
		t.Fatal("expected error from failed refresh, got nil")
	}
	// The stored refresh_token must NOT have been clobbered with empty.
	after := creds.snapshot()
	if after.RefreshToken != before.RefreshToken {
		t.Errorf("refresh_token clobbered: was %q, now %q", before.RefreshToken, after.RefreshToken)
	}
}

func TestProvider_Send_Minus124WithInvalidRefreshTokenReturnsRetryableError(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setSendError(ErrBadAccessToken)
	mock.setOAuthInvalidRefreshToken()

	res, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if !errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("Send err = %v, want ErrRefreshTokenRejected", err)
	}
	if res.ErrorCode != ErrBadAccessToken {
		t.Fatalf("ErrorCode = %d, want original %d", res.ErrorCode, ErrBadAccessToken)
	}
	if after := creds.snapshot(); after.RefreshToken != "live-refresh" {
		t.Fatalf("RefreshToken = %q, should be preserved after rejection", after.RefreshToken)
	}
	sends, oauth := mock.counts()
	if sends != 1 || oauth != 1 {
		t.Fatalf("send/oauth calls = %d/%d, want 1/1", sends, oauth)
	}
}

func TestProvider_Refresh_AcceptsAdvisoryExpiresInShapes(t *testing.T) {
	tests := []struct {
		name        string
		expiresJSON string
		wantAfter   time.Duration
	}{
		{name: "number", expiresJSON: `7200`, wantAfter: 2 * time.Hour},
		{name: "quoted number", expiresJSON: `"7200"`, wantAfter: 2 * time.Hour},
		{name: "missing", wantAfter: time.Hour},
		{name: "null", expiresJSON: `null`, wantAfter: time.Hour},
		{name: "zero", expiresJSON: `0`, wantAfter: time.Hour},
		{name: "negative", expiresJSON: `-1`, wantAfter: time.Hour},
		{name: "invalid string", expiresJSON: `"not-a-number"`, wantAfter: time.Hour},
		{name: "decimal", expiresJSON: `7200.5`, wantAfter: time.Hour},
		{name: "duration overflow", expiresJSON: `9223372036854775807`, wantAfter: time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, mock, creds, _ := newProviderWithMock(t)
			response := `{"access_token":"new-access","refresh_token":"new-refresh"}`
			if tt.expiresJSON != "" {
				response = `{"access_token":"new-access","refresh_token":"new-refresh","expires_in":` + tt.expiresJSON + `}`
			}
			mock.setOAuthResponse(response)

			creds.mu.Lock()
			creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute))
			creds.mu.Unlock()

			startedAt := time.Now()
			_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
			if err != nil {
				t.Fatalf("Send should retain a valid token pair when expires_in is %s: %v", tt.name, err)
			}
			after := creds.snapshot()
			if after.AccessToken != "new-access" || after.RefreshToken != "new-refresh" {
				t.Fatalf("tokens = %q/%q, want new-access/new-refresh", after.AccessToken, after.RefreshToken)
			}
			if after.ExpiresAt == nil {
				t.Fatal("ExpiresAt is nil")
			}
			want := startedAt.Add(tt.wantAfter)
			if delta := after.ExpiresAt.Sub(want); delta < -time.Second || delta > time.Second {
				t.Fatalf("ExpiresAt = %v, want within 1s of %v", after.ExpiresAt, want)
			}
		})
	}
}

// TestProvider_Refresh_StringErrorCodeClassifiedAsRejected verifies a quoted
// numeric error code ("-14014") still maps to ErrRefreshTokenRejected instead
// of failing the whole JSON decode.
func TestProvider_Refresh_StringErrorCodeClassifiedAsRejected(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setOAuthStringErrorCode()

	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // expired
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if !errors.Is(err, ErrRefreshTokenRejected) {
		t.Fatalf("Send err = %v, want ErrRefreshTokenRejected (string -14014)", err)
	}
	if after := creds.snapshot(); after.RefreshToken != "live-refresh" {
		t.Fatalf("RefreshToken = %q, should be preserved after rejection", after.RefreshToken)
	}
}

// TestFlexibleInt64_UnmarshalJSON pins the error-code decode contract: numbers
// and quoted numbers are accepted, null decodes to 0, and garbage fails the
// whole decode so the malformed-response path stays intact.
func TestFlexibleInt64_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{`0`, 0, false},
		{`-14014`, -14014, false},
		{`"-14014"`, -14014, false},
		{`null`, 0, false},
		{`"abc"`, 0, true},
		{`"1.5"`, 0, true},
		{`"99999999999999999999"`, 0, true}, // overflows int64
	}
	for _, c := range cases {
		var got flexibleInt64
		err := json.Unmarshal([]byte(c.in), &got)
		if c.wantErr {
			if err == nil {
				t.Errorf("Unmarshal(%s) = %d, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Unmarshal(%s) error: %v", c.in, err)
			continue
		}
		if int64(got) != c.want {
			t.Errorf("Unmarshal(%s) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestOptionalExpiry_UnmarshalJSON pins the expires_in decode contract: any
// unusable value (absent, garbage, non-positive, overflowing) must decode to 0
// WITHOUT failing the body, so a bad expiry never discards a valid token pair.
func TestOptionalExpiry_UnmarshalJSON(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{`3600`, 3600},
		{`"3600"`, 3600},
		{`null`, 0},
		{`""`, 0},
		{`"abc"`, 0},
		{`0`, 0},
		{`-5`, 0},
		{`"99999999999999999999"`, 0}, // duration overflow
	}
	for _, c := range cases {
		var got optionalExpiry
		if err := json.Unmarshal([]byte(c.in), &got); err != nil {
			t.Errorf("Unmarshal(%s) error: %v, want nil (advisory field)", c.in, err)
			continue
		}
		if int64(got) != c.want {
			t.Errorf("Unmarshal(%s) = %d, want %d", c.in, got, c.want)
		}
	}
}

// TestProvider_Refresh_AccessTokenOnlyKeepsOldRefreshToken verifies that when
// Zalo returns a new access_token WITHOUT a refresh_token (which it does not
// always rotate), the Provider accepts it and preserves the existing
// refresh_token. This is the fix for the "token refresh failed: missing
// access_token or refresh_token" error that caused permanent breakage.
func TestProvider_Refresh_AccessTokenOnlyKeepsOldRefreshToken(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)
	mock.setOAuthAccessTokenOnly()
	before := creds.snapshot()
	oldRefresh := before.RefreshToken

	// Force a refresh path by expiring the token.
	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // expired
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send should succeed with access_token-only refresh, got: %v", err)
	}
	after := creds.snapshot()
	if after.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", after.AccessToken)
	}
	// The old refresh_token must be preserved (Zalo didn't rotate it).
	if after.RefreshToken != oldRefresh {
		t.Errorf("RefreshToken changed: was %q, now %q — should keep old when Zalo doesn't rotate",
			oldRefresh, after.RefreshToken)
	}
	if after.ExpiresAt == nil || after.ExpiresAt.Before(time.Now()) {
		t.Errorf("ExpiresAt not set/future: %v", after.ExpiresAt)
	}
}

func TestProvider_Refresh_PersistsNewPair(t *testing.T) {
	p, _, creds, _ := newProviderWithMock(t)
	creds.mu.Lock()
	creds.cur.ExpiresAt = ptrTime(time.Now().Add(-time.Minute)) // expired → triggers refresh
	creds.mu.Unlock()

	_, err := p.Send(context.Background(), "0987654321", "617976", "t", map[string]string{"otp_code": "1"})
	if err != nil {
		t.Fatalf("Send err: %v", err)
	}
	after := creds.snapshot()
	if after.AccessToken != "new-access" {
		t.Errorf("AccessToken = %q, want new-access", after.AccessToken)
	}
	if after.RefreshToken != "new-refresh" {
		t.Errorf("RefreshToken = %q, want new-refresh (single-use rotation)", after.RefreshToken)
	}
	if after.ExpiresAt == nil || after.ExpiresAt.Before(time.Now()) {
		t.Errorf("ExpiresAt not set/future: %v", after.ExpiresAt)
	}
}

func TestProvider_RefreshNow_ForcesRotationWithFutureExpiry(t *testing.T) {
	p, mock, creds, _ := newProviderWithMock(t)

	// The seeded access token is estimated valid for another 24 hours, but the
	// admin action is an explicit refresh-chain validation and must contact Zalo.
	if err := p.RefreshNow(context.Background()); err != nil {
		t.Fatalf("RefreshNow: %v", err)
	}
	_, oauth := mock.counts()
	if oauth != 1 {
		t.Fatalf("oauth calls = %d, want 1 forced refresh", oauth)
	}
	after := creds.snapshot()
	if after.AccessToken != "new-access" || after.RefreshToken != "new-refresh" {
		t.Fatalf("persisted tokens = %q/%q, want new-access/new-refresh", after.AccessToken, after.RefreshToken)
	}
}

// --- types.go ---------------------------------------------------------------

func TestCredentials_HasTokens(t *testing.T) {
	if (Credentials{}).HasTokens() {
		t.Error("empty creds should not have tokens")
	}
	if (Credentials{AccessToken: "a", RefreshToken: ""}).HasTokens() {
		t.Error("creds with only access token should not report HasTokens")
	}
	if !(Credentials{AccessToken: "a", RefreshToken: "r"}).HasTokens() {
		t.Error("creds with both tokens should report HasTokens")
	}
}

// --- params.go: clampRunes edge cases --------------------------------------

func TestClampRunes(t *testing.T) {
	if got := clampRunes("abc", 5); got != "abc" { // under cap, no alloc
		t.Errorf("under-cap = %q", got)
	}
	if got := clampRunes("abcdef", 3); len([]rune(got)) != 3 {
		t.Errorf("over-cap not truncated: %q", got)
	}
	if got := clampRunes("x", 0); got != "" { // zero cap → empty
		t.Errorf("zero-cap = %q, want empty", got)
	}
	// Vietnamese diacritics counted as single runes.
	if got := clampRunes("Nguyễn Văn A", 6); len([]rune(got)) != 6 {
		t.Errorf("diacritic rune count wrong: %q (%d runes)", got, len([]rune(got)))
	}
}

func TestNewProvider_FillsDefaults(t *testing.T) {
	p := NewProvider(&fakeCreds{}, Config{}, nil)
	if p.cfg.SendURL == "" || p.cfg.OAuthURL == "" || p.cfg.HTTPTimeout <= 0 {
		t.Errorf("defaults not filled: %+v", p.cfg)
	}
}
