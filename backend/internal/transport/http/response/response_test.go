package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

func TestHandleDomainErrorIncludesCodeAndDetails(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	err := domain.NewValidationErrorWithCode("ATTENDANCE_CHECK_IN_WINDOW", "Giờ vào làm không hợp lệ.").
		WithContext("guidance_type", "timing").
		WithContext("window_start", "07:00").
		WithContext("window_end", "09:00")
	HandleDomainError(context, err)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["message"] != "Giờ vào làm không hợp lệ." || body["code"] != "ATTENDANCE_CHECK_IN_WINDOW" {
		t.Fatalf("unexpected envelope: %#v", body)
	}
	details, ok := body["details"].(map[string]any)
	if !ok || details["guidance_type"] != "timing" || details["window_start"] != "07:00" || details["window_end"] != "09:00" {
		t.Fatalf("unexpected details: %#v", body["details"])
	}
}

// TestErrorResponsesHideServerInternals pins PAYR-37: error responses must
// never expose stack traces, filesystem paths or raw database errors — only
// friendly messages.
func TestErrorResponsesHideServerInternals(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		write       func(c *gin.Context)
		wantMessage string
	}{
		{
			name: "raw path error falls back to the friendly message",
			write: func(c *gin.Context) {
				InternalServerError(c, "open /app/data/evidence.csv: no such file or directory")
			},
			wantMessage: constants.MsgInternalServerErrorVN,
		},
		{
			name: "friendly prefix survives with the raw tail dropped",
			write: func(c *gin.Context) {
				InternalServerError(c, "Xử lý file thất bại: open /app/data/evidence.csv: no such file")
			},
			wantMessage: "Xử lý file thất bại",
		},
		{
			name: "stack frame file reference never leaks",
			write: func(c *gin.Context) {
				InternalServerError(c, "Không thể tải dữ liệu\n\t/app/main.go:12 +0x1a3")
			},
			wantMessage: "Không thể tải dữ liệu",
		},
		{
			name: "mysql server error code is cut",
			write: func(c *gin.Context) {
				InternalServerError(c, "Error 1054: Unknown column 'secret_path' in 'field list'")
			},
			wantMessage: constants.MsgInternalServerErrorVN,
		},
		{
			name: "windows path is cut on 400 responses too",
			write: func(c *gin.Context) {
				BadRequest(c, "Tệp không hợp lệ C:\\app\\data\\x.xlsx")
			},
			wantMessage: "Tệp không hợp lệ",
		},
		{
			name: "clean messages pass through unchanged",
			write: func(c *gin.Context) {
				InternalServerError(c, "Không thể xử lý file")
			},
			wantMessage: "Không thể xử lý file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)
			tt.write(ctx)

			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["message"] != tt.wantMessage {
				t.Fatalf("message = %q, want %q", body["message"], tt.wantMessage)
			}
		})
	}
}

func TestHandleDomainErrorHidesInternalsFromWrappedErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	err := fmt.Errorf("query failed: %w", errors.New("open /var/lib/mysql/ibdata1: permission denied"))
	HandleDomainError(ctx, err)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["message"] != constants.MsgInternalServerErrorVN {
		t.Fatalf("message = %q, want %q", body["message"], constants.MsgInternalServerErrorVN)
	}
}

func TestHandleDomainErrorHidesInternalsFromDomainErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	HandleDomainError(ctx, domain.NewInternalError("export failed: open /srv/payroll/private/key.pem: permission denied", errors.New("denied")))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	message, _ := body["message"].(string)
	if message == "" || strings.Contains(message, "/srv/") || strings.Contains(message, "key.pem") {
		t.Fatalf("message leaks internals: %q", message)
	}
}
