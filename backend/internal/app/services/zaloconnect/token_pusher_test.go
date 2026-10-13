package zaloconnect

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"api-server/internal/infra/zalo"
)

// captureHandler records slog records so tests can assert WARNs without
// depending on the default logger's output.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(context.Context, slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(string) slog.Handler      { return h }

func (h *captureHandler) hasWarn(t *testing.T) bool {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if r.Level >= slog.LevelWarn {
			return true
		}
	}
	return false
}

// fakeCredentialSource is an in-memory zalo.CredentialSource for PushCurrent.
type fakeCredentialSource struct {
	mu  sync.Mutex
	cur zalo.Credentials
	err error
}

func (f *fakeCredentialSource) Get(context.Context) (zalo.Credentials, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cur, f.err
}
func (f *fakeCredentialSource) Update(context.Context, zalo.Credentials) error { return nil }
func (f *fakeCredentialSource) UpdateTokens(context.Context, zalo.Credentials) error {
	return nil
}

// newReceivingServer returns a webhook stub that validates the API key and
// records the pushed access tokens.
func newReceivingServer(t *testing.T, wantKey string) (*httptest.Server, <-chan string, *int) {
	t.Helper()
	received := make(chan string, 8)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-API-Key") != wantKey {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var body tokenWebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- body.AccessToken
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, received, &calls
}

func TestTokenWebhookPusher_Success(t *testing.T) {
	srv, received, _ := newReceivingServer(t, "key-1")
	p := NewTokenWebhookPusher(srv.URL, "key-1", nil)

	p.Push(context.Background(), "tok-1")
	p.Push(context.Background(), "tok-2")

	for _, want := range []string{"tok-1", "tok-2"} {
		select {
		case got := <-received:
			if got != want {
				t.Fatalf("pushed token = %q, want %q", got, want)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("push of %q never arrived", want)
		}
	}
}

func TestTokenWebhookPusher_FailureIsLoggedNotPropagated(t *testing.T) {
	capture := &captureHandler{}
	log := slog.New(capture)

	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer rejecting.Close()

	p := NewTokenWebhookPusher(rejecting.URL, "key-1", log)
	p.Push(context.Background(), "tok-1") // must not panic, must log WARN

	unreachable := NewTokenWebhookPusher("http://127.0.0.1:1/", "key-1", log)
	unreachable.Push(context.Background(), "tok-2") // transport failure, still no panic

	if !capture.hasWarn(t) {
		t.Error("no WARN record logged for the failed pushes")
	}
}

func TestTokenWebhookPusher_SkipsWhenUnconfigured(t *testing.T) {
	cases := []struct {
		name string
		url  string
		key  string
	}{
		{"no url", "", "key-1"},
		{"no key", "http://127.0.0.1:1/", ""},
		{"neither", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewTokenWebhookPusher(tc.url, tc.key, nil)
			p.Push(context.Background(), "tok-1") // must be a silent debug skip
		})
	}
}

func TestTokenWebhookPusher_EmptyTokenIsSkipped(t *testing.T) {
	srv, received, _ := newReceivingServer(t, "key-1")
	p := NewTokenWebhookPusher(srv.URL, "key-1", nil)

	p.Push(context.Background(), "")

	select {
	case got := <-received:
		t.Fatalf("empty token was pushed: %q", got)
	case <-time.After(150 * time.Millisecond):
	}
}

func TestTokenWebhookPusher_PushCurrentSendsStoredToken(t *testing.T) {
	srv, received, _ := newReceivingServer(t, "key-1")
	p := NewTokenWebhookPusher(srv.URL, "key-1", nil)
	creds := &fakeCredentialSource{cur: zalo.Credentials{
		AppID:       "app1",
		AccessToken: "stored-access",
	}}

	p.PushCurrent(context.Background(), creds)

	select {
	case got := <-received:
		if got != "stored-access" {
			t.Fatalf("pushed token = %q, want stored-access", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("startup push never arrived")
	}
}

func TestTokenWebhookPusher_PushCurrentSkipsWithoutToken(t *testing.T) {
	srv, received, _ := newReceivingServer(t, "key-1")
	p := NewTokenWebhookPusher(srv.URL, "key-1", nil)

	p.PushCurrent(context.Background(), &fakeCredentialSource{}) // no access token
	p.PushCurrent(context.Background(), &fakeCredentialSource{err: errors.New("read failed")})
	// a read error is also fail-soft: nothing pushed, no panic

	select {
	case got := <-received:
		t.Fatalf("unexpected push: %q", got)
	case <-time.After(150 * time.Millisecond):
	}
}
