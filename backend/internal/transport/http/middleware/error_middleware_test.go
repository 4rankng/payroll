package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateErrorID(t *testing.T) {
	errorID := generateErrorID()
	if errorID == "" {
		t.Error("generateErrorID() returned empty string")
	}

	if !strings.HasPrefix(errorID, "ERR-") {
		t.Errorf("generateErrorID() = %v, expected to start with 'ERR-'", errorID)
	}
}

func TestRequestSizeMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		maxSize        int64
		contentLength  int64
		expectAbort    bool
		expectedStatus int
	}{
		{
			name:           "request within size limit",
			maxSize:        1024,
			contentLength:  512,
			expectAbort:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "request at size limit",
			maxSize:        1024,
			contentLength:  1024,
			expectAbort:    false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "request exceeds size limit",
			maxSize:        1024,
			contentLength:  2048,
			expectAbort:    true,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero content length",
			maxSize:        1024,
			contentLength:  0,
			expectAbort:    false,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RequestSizeMiddleware(tt.maxSize))

			handlerCalled := false
			router.POST("/test", func(c *gin.Context) {
				handlerCalled = true
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodPost, "/test", nil)
			req.ContentLength = tt.contentLength
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.expectAbort && handlerCalled {
				t.Error("Handler was called but should have been aborted")
			}

			if !tt.expectAbort && !handlerCalled {
				t.Error("Handler was not called but should have been")
			}

			if w.Code != tt.expectedStatus {
				t.Errorf("Response code = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}

func TestErrorMiddlewareRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(ErrorMiddleware())

	router.GET("/panic", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// The middleware should catch the panic and return an error response
	if w.Code == http.StatusOK {
		t.Error("ErrorMiddleware did not handle panic, returned 200 OK")
	}
}

func TestErrorHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupError     func(*gin.Context)
		expectedStatus int
	}{
		{
			name: "no errors",
			setupError: func(c *gin.Context) {
				// No error
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(ErrorHandler())

			router.GET("/test", func(c *gin.Context) {
				tt.setupError(c)
				if len(c.Errors) == 0 {
					c.String(http.StatusOK, "OK")
				}
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Response code = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
