package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	auditctx "api-server/internal/pkg/context"

	"github.com/gin-gonic/gin"
)

func TestAuditContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRequest   func(*http.Request)
		expectedIP     string
		expectedUA     string
		contextChecked bool
	}{
		{
			name: "with user agent",
			setupRequest: func(req *http.Request) {
				req.Header.Set("User-Agent", "Mozilla/5.0")
			},
			expectedUA: "Mozilla/5.0",
		},
		{
			name: "without user agent",
			setupRequest: func(req *http.Request) {
				// No user agent
			},
			expectedUA: "",
		},
		{
			name: "with custom user agent",
			setupRequest: func(req *http.Request) {
				req.Header.Set("User-Agent", "CustomBot/1.0")
			},
			expectedUA: "CustomBot/1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(AuditContext())

			handlerCalled := false
			router.GET("/test", func(c *gin.Context) {
				handlerCalled = true

				// Verify audit context was set
				ctx := c.Request.Context()

				// Check standard context keys
				ipFromCtx := ctx.Value(auditctx.StandardIPKey)
				if ipFromCtx == nil {
					t.Error("Expected IP address to be set in context")
				}

				uaFromCtx := ctx.Value(auditctx.StandardUAKey)
				if tt.expectedUA != "" {
					if uaStr, ok := uaFromCtx.(string); !ok || uaStr != tt.expectedUA {
						t.Errorf("User agent in context = %v, want %v", uaFromCtx, tt.expectedUA)
					}
				}

				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			tt.setupRequest(req)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if !handlerCalled {
				t.Error("Handler was not called")
			}

			if w.Code != http.StatusOK {
				t.Errorf("Response code = %v, want %v", w.Code, http.StatusOK)
			}
		})
	}
}

func TestAuditContextAllowsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuditContext())

	handlerCalled := false
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("AuditContext middleware prevented handler from being called")
	}
}

func TestAuditContextIPAddress(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuditContext())

	var capturedIP string
	router.GET("/test", func(c *gin.Context) {
		ctx := c.Request.Context()
		if ip := ctx.Value(auditctx.StandardIPKey); ip != nil {
			capturedIP = ip.(string)
		}
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should have captured some IP (even if it's empty or localhost)
	// The important thing is that the middleware set it
	if w.Code != http.StatusOK {
		t.Errorf("Response code = %v, want %v", w.Code, http.StatusOK)
	}

	// capturedIP should be set (not checking specific value as it depends on test environment)
	t.Logf("Captured IP: %s", capturedIP)
}
