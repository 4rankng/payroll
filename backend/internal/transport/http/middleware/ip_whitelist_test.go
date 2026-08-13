package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newIPWhitelistRouter(allowed []string, opts ...Option) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(IPWhitelist(allowed, opts...))
	r.GET("/probe", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	return r
}

func do(t *testing.T, r *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestIPWhitelist_EmptyListIsNoop(t *testing.T) {
	r := newIPWhitelistRouter(nil)
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	w := do(t, r, req)
	if w.Code != http.StatusOK {
		t.Fatalf("empty list should pass through, got %d", w.Code)
	}
}

func TestIPWhitelist_AllowsListedClientIP(t *testing.T) {
	r := newIPWhitelistRouter([]string{"10.0.0.1"})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "10.0.0.1:54321"
	// Force c.ClientIP() to return our test address by setting trusted
	// proxies to the same value.
	_ = r.SetTrustedProxies([]string{"10.0.0.1"})
	w := do(t, r, req)
	if w.Code != http.StatusOK {
		t.Fatalf("listed IP should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}

func TestIPWhitelist_BlocksUnlistedClientIP(t *testing.T) {
	r := newIPWhitelistRouter([]string{"10.0.0.1"})
	_ = r.SetTrustedProxies([]string{"10.0.0.1"})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "10.0.0.99:54321"
	w := do(t, r, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("unlisted IP should be blocked, got %d", w.Code)
	}
}

func TestIPWhitelist_AllowsClientIPInCIDR(t *testing.T) {
	r := newIPWhitelistRouter([]string{"118.70.247.80/29"})
	_ = r.SetTrustedProxies([]string{"118.70.247.80/29"})

	for _, clientIP := range []string{"118.70.247.80", "118.70.247.83", "118.70.247.87"} {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.RemoteAddr = clientIP + ":54321"
		if w := do(t, r, req); w.Code != http.StatusOK {
			t.Fatalf("IP %s in allowed CIDR should pass, got %d", clientIP, w.Code)
		}
	}
}

func TestIPWhitelist_BlocksClientIPOutsideCIDR(t *testing.T) {
	r := newIPWhitelistRouter([]string{"118.70.247.80/29"})
	_ = r.SetTrustedProxies([]string{"118.70.247.0/24"})
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "118.70.247.88:54321"

	if w := do(t, r, req); w.Code != http.StatusForbidden {
		t.Fatalf("IP outside allowed CIDR should be blocked, got %d", w.Code)
	}
}

func TestIPWhitelist_AllowsExactIPAlongsideCIDR(t *testing.T) {
	r := newIPWhitelistRouter([]string{"203.0.113.5", "202.9.84.0/24"}, UseRemoteAddr())

	for _, clientIP := range []string{"203.0.113.5", "202.9.84.183"} {
		req := httptest.NewRequest(http.MethodGet, "/probe", nil)
		req.RemoteAddr = clientIP + ":54321"
		if w := do(t, r, req); w.Code != http.StatusOK {
			t.Fatalf("allowed IP %s should pass, got %d", clientIP, w.Code)
		}
	}
}

func TestIPWhitelist_InvalidEntryDoesNotDisableWhitelist(t *testing.T) {
	r := newIPWhitelistRouter([]string{"not-an-ip"}, UseRemoteAddr())
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "203.0.113.5:54321"

	if w := do(t, r, req); w.Code != http.StatusForbidden {
		t.Fatalf("invalid allowlist entry must fail closed, got %d", w.Code)
	}
}

func TestIPWhitelist_UseRemoteAddrBypassesTrustedProxies(t *testing.T) {
	// Simulate the "behind a proxy" scenario: c.ClientIP() would resolve
	// to the proxy (127.0.0.1), which is not in the allow list. With
	// UseRemoteAddr, the middleware must consult RemoteAddr instead.
	r := newIPWhitelistRouter([]string{"203.0.113.5"}, UseRemoteAddr())
	_ = r.SetTrustedProxies([]string{"127.0.0.1", "::1"})

	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "203.0.113.5:54321"
	if w := do(t, r, req); w.Code != http.StatusOK {
		t.Fatalf("listed RemoteAddr should pass, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req2.RemoteAddr = "198.51.100.7:54321"
	if w := do(t, r, req2); w.Code != http.StatusForbidden {
		t.Fatalf("unlisted RemoteAddr should be blocked, got %d", w.Code)
	}
}

func TestIPWhitelist_UseRemoteAddrStripsPort(t *testing.T) {
	// Allow list is plain IPs without ports; the middleware must strip
	// the port from "host:port" before comparing.
	r := newIPWhitelistRouter([]string{"203.0.113.5"}, UseRemoteAddr())
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	req.RemoteAddr = "203.0.113.5:443"
	if w := do(t, r, req); w.Code != http.StatusOK {
		t.Fatalf("host:port RemoteAddr should match, got %d", w.Code)
	}
}
