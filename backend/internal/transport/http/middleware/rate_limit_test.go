package middleware

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newContextWithBody(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	c.Request.RemoteAddr = "1.2.3.4:5678"
	return c
}

func TestLoginAccountKeyGetter_PerAccount(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "username extracted and lowercased",
			body: `{"username":"Alice","password":"secret"}`,
			want: "account:alice",
		},
		{
			name: "CCCD identifier treated as the account key",
			body: `{"username":"0123456789"}`,
			want: "account:0123456789",
		},
		{
			name: "whitespace trimmed",
			body: `{"username":"  bob  "}`,
			want: "account:bob",
		},
		{
			name: "different accounts yield different keys",
			body: `{"username":"carol"}`,
			want: "account:carol",
		},
		{
			name: "OTP verify body keyed by session (per-account proxy)",
			body: `{"otp_session_id":"sess-123","code":"000000"}`,
			want: "otp-session:sess-123",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newContextWithBody(tc.body)
			if got := loginAccountKeyGetter(c); got != tc.want {
				t.Fatalf("loginAccountKeyGetter = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestLoginAccountKeyGetter_BodyRestored proves the downstream handler can
// still read the full body after the key getter has peeked at it.
func TestLoginAccountKeyGetter_BodyRestored(t *testing.T) {
	body := `{"username":"Alice","password":"hunter2"}`
	c := newContextWithBody(body)
	_ = loginAccountKeyGetter(c)

	rest, err := io.ReadAll(c.Request.Body)
	if err != nil {
		t.Fatalf("re-read body: %v", err)
	}
	if string(rest) != body {
		t.Fatalf("body not restored: got %q, want %q", string(rest), body)
	}
}

// TestLoginAccountKeyGetter_FallbackToIP proves that a body without an
// identifier falls back to per-IP isolation rather than a shared global key.
func TestLoginAccountKeyGetter_FallbackToIP(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{name: "no username field", body: `{"password":"secret"}`},
		{name: "empty username", body: `{"username":"  "}`},
		{name: "malformed JSON", body: `not-json`},
		{name: "empty body", body: ``},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newContextWithBody(tc.body)
			got := loginAccountKeyGetter(c)
			if got != "ip:1.2.3.4" {
				t.Fatalf("loginAccountKeyGetter = %q, want %q", got, "ip:1.2.3.4")
			}
		})
	}
}
