package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		useTLS         bool
		expectedHSTS   bool
		expectedHeader map[string]string
	}{
		{
			name:         "security headers without TLS",
			useTLS:       false,
			expectedHSTS: false,
			expectedHeader: map[string]string{
				"X-Content-Type-Options":  "nosniff",
				"X-Frame-Options":         "DENY",
				"X-XSS-Protection":        "1; mode=block",
				"Referrer-Policy":         "strict-origin-when-cross-origin",
				"Permissions-Policy":      "geolocation=(), microphone=(), camera=()",
				"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-eval' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://accounts.google.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://fonts.googleapis.com https://accounts.google.com https://oauth2.googleapis.com; frame-src https://accounts.google.com; frame-ancestors 'none'",
			},
		},
		{
			name:         "security headers with TLS",
			useTLS:       true,
			expectedHSTS: true,
			expectedHeader: map[string]string{
				"X-Content-Type-Options":    "nosniff",
				"X-Frame-Options":           "DENY",
				"X-XSS-Protection":          "1; mode=block",
				"Referrer-Policy":           "strict-origin-when-cross-origin",
				"Permissions-Policy":        "geolocation=(), microphone=(), camera=()",
				"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-eval' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://accounts.google.com; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https://cdnjs.cloudflare.com https://cdn.sheetjs.com https://fonts.googleapis.com https://accounts.google.com https://oauth2.googleapis.com; frame-src https://accounts.google.com; frame-ancestors 'none'",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test router
			router := gin.New()
			router.Use(SecurityHeaders())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			// Create a test request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.useTLS {
				req.TLS = &tls.ConnectionState{}
			}

			// Record the response
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Check all expected headers
			for header, expectedValue := range tt.expectedHeader {
				gotValue := w.Header().Get(header)
				if gotValue != expectedValue {
					t.Errorf("Header %s = %v, want %v", header, gotValue, expectedValue)
				}
			}

			// Check HSTS header presence
			hstsHeader := w.Header().Get("Strict-Transport-Security")
			if tt.expectedHSTS && hstsHeader == "" {
				t.Error("Expected HSTS header with TLS but got none")
			}
			if !tt.expectedHSTS && hstsHeader != "" {
				t.Errorf("Did not expect HSTS header without TLS but got: %s", hstsHeader)
			}

			// Verify the response
			if w.Code != http.StatusOK {
				t.Errorf("Response code = %v, want %v", w.Code, http.StatusOK)
			}
		})
	}
}

func TestSecurityHeadersMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a test router with the middleware
	router := gin.New()
	router.Use(SecurityHeaders())

	testHandlerCalled := false
	router.GET("/test", func(c *gin.Context) {
		testHandlerCalled = true
		c.String(http.StatusOK, "Handler executed")
	})

	// Create and execute request
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify middleware allows request to proceed
	if !testHandlerCalled {
		t.Error("Security headers middleware blocked the handler from executing")
	}

	// Verify at least one security header was set
	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Error("Security headers middleware did not set any headers")
	}
}
