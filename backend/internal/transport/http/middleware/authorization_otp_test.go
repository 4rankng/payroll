package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

	authservice "api-server/internal/app/services/auth"
	"api-server/internal/config"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// resolveConfigPaths returns absolute paths to the casbin model + policy files,
// regardless of the package dir the test runs in (mirrors authorization_service_test.go).
func resolveConfigPaths(t *testing.T) (model, policy string) {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve caller for config path")
	}
	// middleware/ -> transport/http/middleware -> up to backend root.
	repoRoot := filepath.Join(filepath.Dir(here), "..", "..", "..", "..")
	return filepath.Join(repoRoot, "configs", "casbin_model.conf"),
		filepath.Join(repoRoot, "configs", "casbin_policy.csv")
}

// TestAuthorize_OTPEnforcement_Admin verifies the RT-C1 fix using the real
// Casbin policy: when OTP_ENABLE is on, an admin token (which the policy grants
// wildcard access, so it clears Casbin) must STILL be rejected if it lacks
// otp_verified=true. This is the regression guard for the money routes
// (/wallet, /admin/manual-disbursement) that the original "attach to protected
// group" design structurally missed.
func TestAuthorize_OTPEnforcement_Admin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(&discardHandler{})

	model, policy := resolveConfigPaths(t)
	authzSvc, err := authservice.NewAuthorizationService(model, policy, nil, logger)
	if err != nil {
		t.Fatalf("NewAuthorizationService: %v", err)
	}
	m := NewAuthorizationMiddleware(authzSvc, nil, nil, config.OTPConfig{Enabled: true})

	cases := []struct {
		name       string
		role       string
		path       string
		otpSet     bool
		otpValue   bool
		wantStatus int
	}{
		// RT-C1: admin reaching a money route without OTP -> 403.
		{"admin unverified -> 403", string(domain.RoleAdmin), "/api/v1/wallet/balance", false, false, http.StatusForbidden},
		{"admin otp_verified=false -> 403", string(domain.RoleAdmin), "/api/v1/wallet/balance", true, false, http.StatusForbidden},
		{"admin otp_verified=true -> 200", string(domain.RoleAdmin), "/api/v1/wallet/balance", true, true, http.StatusOK},
		// The accountant is money-adjacent (approves payroll, exports bank
		// transfer files), so it is OTP-gated like admin/partner. Probed on a
		// route the policy explicitly allows the accountant.
		{"accountant unverified -> 403", string(domain.RoleAccountant), "/api/v1/timesheets/summary", false, false, http.StatusForbidden},
		{"accountant otp_verified=true -> 200", string(domain.RoleAccountant), "/api/v1/timesheets/summary", true, true, http.StatusOK},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(constants.CtxUserID, uint(1))
				c.Set(constants.CtxUserRole, tc.role)
				if tc.otpSet {
					c.Set("otp_verified", tc.otpValue)
				}
				c.Next()
			})
			router.Use(m.Authorize())
			// /wallet is mounted on v1 directly in production (not behind the
			// `protected` group) — exactly the RT-C1 blind spot.
			router.GET("/api/v1/wallet/balance", func(c *gin.Context) { c.Status(http.StatusOK) })
			// A route the accountant policy explicitly allows, used by the
			// accountant cases below.
			router.GET("/api/v1/timesheets/summary", func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.wantStatus {
				t.Fatalf("got status %d, want %d (body=%s)", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestAuthorize_OTPDisabled_NoEnforcement verifies the kill-switch: with
// OTP_ENABLE=false, the OTP gate is a complete no-op (regression-safe —
// behavior identical to before the feature shipped).
func TestAuthorize_OTPDisabled_NoEnforcement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(&discardHandler{})

	model, policy := resolveConfigPaths(t)
	authzSvc, err := authservice.NewAuthorizationService(model, policy, nil, logger)
	if err != nil {
		t.Fatalf("NewAuthorizationService: %v", err)
	}
	m := NewAuthorizationMiddleware(authzSvc, nil, nil, config.OTPConfig{Enabled: false})

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(constants.CtxUserID, uint(1))
		c.Set(constants.CtxUserRole, string(domain.RoleAdmin))
		// Deliberately do NOT set otp_verified — flag is off, must not matter.
		c.Next()
	})
	router.Use(m.Authorize())
	router.GET("/api/v1/wallet/balance", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("flag off: admin without otp_verified should pass, got %d (body=%s)", w.Code, w.Body.String())
	}
}

// discardHandler is a no-op slog handler for quiet tests.
type discardHandler struct{}

func (h *discardHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (h *discardHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (h *discardHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *discardHandler) WithGroup(_ string) slog.Handler               { return h }
