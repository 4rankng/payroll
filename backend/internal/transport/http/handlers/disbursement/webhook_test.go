package disbursement

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	disbursementservice "api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	"api-server/internal/infra/persistence"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
)

type onepayAckTestProvider struct{}

func (onepayAckTestProvider) Name() string { return "1pay" }

func (onepayAckTestProvider) InitiateTransfer(context.Context, infrastructure.TransferRequest) (*infrastructure.TransferResult, error) {
	return nil, nil
}

func (onepayAckTestProvider) VerifyAndParseWebhook(context.Context, map[string]any) (*infrastructure.WebhookEvent, error) {
	return &infrastructure.WebhookEvent{
		RequestID:   "FT-ACK-001",
		ProviderRef: "OP-ACK-001",
		Status:      infrastructure.TransferStatusSuccess,
		Amount:      200_000,
	}, nil
}

// TestDecodeWebhookPayload_PreservesPlusInFormBody locks in the fix for
// the base64-corruption bug observed in production on 2026-05-07.
//
// 9pay sends the IPN as application/x-www-form-urlencoded with the
// `result` field carrying a raw standard-base64 string. Standard base64's
// alphabet includes '+', and 9pay does not percent-encode it as %2B.
// Go's net/url form decoder follows the urlencoded spec strictly:
// '+' decodes to space. That silently corrupts every base64 string
// containing '+' — roughly 1 in 4 IPNs in practice. The handler now
// parses the body itself with url.PathUnescape, which preserves '+'
// literally and still resolves %2B → '+'.
func TestDecodeWebhookPayload_PreservesPlusInFormBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Construct a result string that round-trips through base64 to a
	// value containing '+' — the exact failure mode of the rejected IPN.
	result := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("\xfb", 4)))
	if !strings.Contains(result, "+") {
		t.Fatalf("test fixture base64 must contain '+', got %q", result)
	}

	body := "result=" + result + "&checksum=ABC&version=v1"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	out, err := decodeWebhookPayload(c)
	if err != nil {
		t.Fatalf("decodeWebhookPayload: %v", err)
	}
	if got := out["result"]; got != result {
		t.Fatalf("result was corrupted by form decoder.\n  want %q\n  got  %v", result, got)
	}
	if got := out["checksum"]; got != "ABC" {
		t.Fatalf("checksum: want %q, got %v", "ABC", got)
	}
}

// TestDecodeWebhookPayload_ResolvesPercentEncodedPlus confirms the
// parser still handles spec-conformant senders that escape '+' as %2B.
func TestDecodeWebhookPayload_ResolvesPercentEncodedPlus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	want := "ab+cd/ef=="
	body := "result=" + url.PathEscape(want) + "&checksum=X"
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = req

	out, err := decodeWebhookPayload(c)
	if err != nil {
		t.Fatalf("decodeWebhookPayload: %v", err)
	}
	if got := out["result"]; got != want {
		t.Fatalf("result: want %q, got %v", want, got)
	}
}

// TestParseFormPreservingPlus_HandlesEdgeCases covers the parser's
// tolerance for malformed-but-realistic urlencoded bodies.
func TestParseFormPreservingPlus_HandlesEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		body string
		key  string
		want string
	}{
		{"raw plus survives", "result=a+b+c&checksum=x", "result", "a+b+c"},
		{"percent-encoded plus", "result=a%2Bb&checksum=x", "result", "a+b"},
		{"mixed slashes", "result=ab/cd%2F&checksum=x", "result", "ab/cd/"},
		{"empty value ok", "result=&checksum=x", "result", ""},
		{"trailing amp ok", "result=AAA&checksum=x&", "checksum", "x"},
		{"key without equals", "flag&checksum=x", "checksum", "x"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := parseFormPreservingPlus(c.body)
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if got := out[c.key]; got != c.want {
				t.Errorf("%s: got %v, want %q", c.key, got, c.want)
			}
		})
	}
}

func TestWriteWebhookAcknowledgement_OnePayExactContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	writeWebhookAcknowledgement(c, "1pay")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var got map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode acknowledgement: %v", err)
	}
	want := map[string]string{"error_code": "0", "message": "Success"}
	if got["error_code"] != want["error_code"] || got["message"] != want["message"] || len(got) != len(want) {
		t.Fatalf("acknowledgement = %v, want %v", got, want)
	}
}

func TestWebhookReceive_ValidDuplicateOnePayIPNReturnsSuccessAck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mini := miniredis.RunT(t)
	redisClient, err := persistence.NewRedisClient(persistence.RedisConfig{Addr: mini.Addr()})
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			t.Errorf("close Redis client: %v", err)
		}
	}()

	handler := NewWebhookHandler(
		disbursementservice.NewRegistry(onepayAckTestProvider{}),
		nil,
		nil,
		nil,
		redisClient,
		nil,
		nil,
	)

	for attempt := 1; attempt <= 2; attempt++ {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/webhooks/disbursement/1pay", strings.NewReader(`{}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler.ReceiveFrom("1pay")(c)

		if recorder.Code != http.StatusOK {
			t.Fatalf("attempt %d status = %d, want 200; body=%s", attempt, recorder.Code, recorder.Body.String())
		}
		var got map[string]string
		if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
			t.Fatalf("attempt %d decode acknowledgement: %v", attempt, err)
		}
		if got["error_code"] != "0" || got["message"] != "Success" || len(got) != 2 {
			t.Fatalf("attempt %d acknowledgement = %v", attempt, got)
		}
	}
}
