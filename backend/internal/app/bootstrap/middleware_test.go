package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetupCORSAllowsBCCImportIdempotencyKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	setupCORS(router, nil)
	router.POST("/api/v1/timesheets/partner-import", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/timesheets/partner-import", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", http.MethodPost)
	req.Header.Set(
		"Access-Control-Request-Headers",
		"authorization,content-type,idempotency-key",
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected preflight status %d, got %d", http.StatusNoContent, recorder.Code)
	}

	allowedHeaders := strings.ToLower(recorder.Header().Get("Access-Control-Allow-Headers"))
	if !strings.Contains(allowedHeaders, "idempotency-key") {
		t.Fatalf("expected Idempotency-Key in Access-Control-Allow-Headers, got %q", allowedHeaders)
	}
}
