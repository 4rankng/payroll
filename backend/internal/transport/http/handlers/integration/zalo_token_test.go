package integration

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/infra/zalo"

	"github.com/gin-gonic/gin"
)

// fakeCredentialReader is an in-memory CredentialReader for the token handler.
type fakeCredentialReader struct {
	creds zalo.Credentials
	err   error
}

func (f *fakeCredentialReader) Get(_ context.Context) (zalo.Credentials, error) {
	return f.creds, f.err
}

// TestZaloTokenOutcome pins the pull endpoint's status/message mapping.
// notConfigured derives from zalo.ErrNotConfigured with the "zalo: " log
// prefix stripped, so the endpoint stays in sync with the infra error text.
func TestZaloTokenOutcome(t *testing.T) {
	notConfigured := strings.TrimPrefix(zalo.ErrNotConfigured.Error(), "zalo: ")
	cases := []struct {
		name        string
		accessToken string
		err         error
		wantStatus  int
		wantMessage string
	}{
		{
			name:        "not configured → 503 with the actionable Vietnamese text",
			err:         zalo.ErrNotConfigured,
			wantStatus:  http.StatusServiceUnavailable,
			wantMessage: notConfigured,
		},
		{
			name:        "empty stored token → 503 with the same text",
			accessToken: "",
			wantStatus:  http.StatusServiceUnavailable,
			wantMessage: notConfigured,
		},
		{
			name:        "success → 200",
			accessToken: "live-token",
			wantStatus:  http.StatusOK,
			wantMessage: constants.MsgIntegrationZaloTokenVN,
		},
		{
			name:        "generic read failure → 500",
			err:         errors.New("db down"),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: constants.MsgZaloResetStoreDownVN,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStatus, gotMessage := zaloTokenOutcome(tc.accessToken, tc.err)
			if gotStatus != tc.wantStatus || gotMessage != tc.wantMessage {
				t.Errorf("zaloTokenOutcome(%q, %v) = (%d, %q), want (%d, %q)",
					tc.accessToken, tc.err, gotStatus, gotMessage, tc.wantStatus, tc.wantMessage)
			}
		})
	}
}

// TestZaloTokenHandler pins the HTTP shape through gin: envelope, token in the
// data field, and each failure mode's status.
func TestZaloTokenHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		reader      *fakeCredentialReader
		wantStatus  int
		wantStatus2 string // envelope "status" field
		wantToken   string
	}{
		{
			name:        "success",
			reader:      &fakeCredentialReader{creds: zalo.Credentials{AccessToken: "live-token"}},
			wantStatus:  http.StatusOK,
			wantStatus2: "success",
			wantToken:   "live-token",
		},
		{
			name:        "not configured",
			reader:      &fakeCredentialReader{err: zalo.ErrNotConfigured},
			wantStatus:  http.StatusServiceUnavailable,
			wantStatus2: "error",
		},
		{
			name:        "empty token",
			reader:      &fakeCredentialReader{},
			wantStatus:  http.StatusServiceUnavailable,
			wantStatus2: "error",
		},
		{
			name:        "generic error",
			reader:      &fakeCredentialReader{err: errors.New("db down")},
			wantStatus:  http.StatusInternalServerError,
			wantStatus2: "error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			_, router := gin.CreateTestContext(recorder)
			router.GET("/integration/zalo/token", NewZaloTokenHandler(tc.reader).GetToken)

			req := httptest.NewRequest(http.MethodGet, "/integration/zalo/token", nil)
			router.ServeHTTP(recorder, req)

			if recorder.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d (body: %s)", recorder.Code, tc.wantStatus, recorder.Body.String())
			}
			var envelope struct {
				Status  string                 `json:"status"`
				Message string                 `json:"message"`
				Data    *dto.ZaloTokenResponse `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v (body: %s)", err, recorder.Body.String())
			}
			if envelope.Status != tc.wantStatus2 {
				t.Errorf("envelope status = %q, want %q", envelope.Status, tc.wantStatus2)
			}
			if tc.wantToken != "" && (envelope.Data == nil || envelope.Data.AccessToken != tc.wantToken) {
				t.Errorf("data.access_token = %+v, want %q", envelope.Data, tc.wantToken)
			}
			if tc.wantToken == "" && tc.wantStatus == http.StatusServiceUnavailable {
				want := strings.TrimPrefix(zalo.ErrNotConfigured.Error(), "zalo: ")
				if envelope.Message != want {
					t.Errorf("message = %q, want the trimmed not-configured text", envelope.Message)
				}
			}
		})
	}
}
