package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name            string
		timeout         time.Duration
		expectTimeout   bool
		handlerDuration time.Duration
	}{
		{
			name:            "zero timeout does not set timeout",
			timeout:         0,
			expectTimeout:   false,
			handlerDuration: 10 * time.Millisecond,
		},
		{
			name:            "negative timeout does not set timeout",
			timeout:         -1 * time.Second,
			expectTimeout:   false,
			handlerDuration: 10 * time.Millisecond,
		},
		{
			name:            "timeout set with positive duration",
			timeout:         100 * time.Millisecond,
			expectTimeout:   false,
			handlerDuration: 10 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestTimeout(tt.timeout))

			contextChecked := false
			router.GET("/test", func(c *gin.Context) {
				// Check if context has timeout
				_, hasDeadline := c.Request.Context().Deadline()

				if tt.timeout > 0 && !hasDeadline {
					t.Error("Expected context to have deadline but it doesn't")
				}

				if tt.timeout <= 0 && hasDeadline {
					t.Error("Did not expect context to have deadline but it does")
				}

				contextChecked = true

				// Simulate some work
				time.Sleep(tt.handlerDuration)

				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if !contextChecked {
				t.Error("Handler was not executed")
			}

			// For short durations, the request should complete successfully
			if w.Code != http.StatusOK {
				t.Errorf("Response code = %v, want %v", w.Code, http.StatusOK)
			}
		})
	}
}

func TestRequestTimeoutContextPropagation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	timeout := 5 * time.Second
	router.Use(RequestTimeout(timeout))

	var requestContext context.Context
	router.GET("/test", func(c *gin.Context) {
		requestContext = c.Request.Context()
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify the context has a deadline
	deadline, ok := requestContext.Deadline()
	if !ok {
		t.Error("Expected request context to have deadline")
	}

	// Verify the deadline is in the future
	if time.Until(deadline) < 0 {
		t.Error("Deadline is in the past")
	}
}

func TestRequestTimeoutAllowsNext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequestTimeout(1 * time.Second))

	handlerCalled := false
	router.GET("/test", func(c *gin.Context) {
		handlerCalled = true
		c.String(http.StatusOK, "Handler executed")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !handlerCalled {
		t.Error("RequestTimeout middleware prevented handler from being called")
	}

	if w.Code != http.StatusOK {
		t.Errorf("Response code = %v, want %v", w.Code, http.StatusOK)
	}
}
