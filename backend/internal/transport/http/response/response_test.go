package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
