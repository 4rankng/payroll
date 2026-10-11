package settings

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"api-server/internal/app/services/config"
	llmsvc "api-server/internal/app/services/llm"
	"api-server/internal/domain"
	llmport "api-server/internal/infra/llm"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/secret"
)

const fakeMimoKey = "mimo-sk-test-fake-key-never-real"

// newSettingsTestHarness builds the real settings service stack over an
// in-memory SQLite database, exactly as production wires it (repository seals
// protected values; the DTO masks them). Only the read paths need deps.
func newSettingsTestHarness(t *testing.T) (*Handler, *config.SettingsService, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE settings (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			value TEXT,
			value_type TEXT NOT NULL,
			deleted_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create settings table: %v", err)
	}

	key := make([]byte, secret.KeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}
	cipher, err := secret.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}

	repo := persistence.NewSettingsRepositoryWithCipher(&persistence.Database{DB: db}, cipher)
	settingsService := config.NewSettingsService(repo, nil, nil, nil, nil, nil, nil)
	handler := NewHandler(settingsService, nil)
	return handler, settingsService, db
}

// seedSetting writes one row through the repository (so protected keys are
// sealed at rest) and returns its ID.
func seedSetting(t *testing.T, settingsService *config.SettingsService, key, value string, valueType domain.SettingsValueType) uint {
	t.Helper()
	row := &domain.Settings{Key: key, Value: &value, ValueType: valueType}
	if err := settingsService.SettingsRepo.Create(context.Background(), row); err != nil {
		t.Fatalf("seed %s: %v", key, err)
	}
	return row.ID
}

func newSettingsRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/settings", handler.ListSettings)
	router.GET("/api/v1/settings/active", handler.GetActiveSettings)
	router.GET("/api/v1/settings/:id", handler.GetSetting)
	router.GET("/api/v1/settings/key/:key", handler.GetSettingByKey)
	router.POST("/api/v1/settings/mimo/test-connection", handler.TestMimoConnection)
	return router
}

type envelope struct {
	Data json.RawMessage `json:"data"`
}

func decodeSingle(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, body)
	}
	var single map[string]any
	if err := json.Unmarshal(env.Data, &single); err != nil {
		t.Fatalf("decode data: %v (%s)", err, env.Data)
	}
	return single
}

func decodeList(t *testing.T, body []byte) []map[string]any {
	t.Helper()
	var env envelope
	if err := json.Unmarshal(body, &env); err != nil {
		t.Fatalf("decode envelope: %v (%s)", err, body)
	}
	var list []map[string]any
	if err := json.Unmarshal(env.Data, &list); err != nil {
		t.Fatalf("decode data: %v (%s)", err, env.Data)
	}
	return list
}

// assertMasked asserts the mimo.api_key entry is masked: no value, masked and
// configured flags set — and the plaintext appears nowhere in the payload.
func assertMasked(t *testing.T, entry map[string]any, rawBody string) {
	t.Helper()
	if entry["key"] != "mimo.api_key" {
		t.Fatalf("entry = %v, want the mimo.api_key row", entry)
	}
	if entry["value"] != nil {
		t.Fatalf("value = %v, want nil for a masked secret", entry["value"])
	}
	if entry["masked"] != true {
		t.Fatalf("masked = %v, want true", entry["masked"])
	}
	if entry["configured"] != true {
		t.Fatalf("configured = %v, want true", entry["configured"])
	}
	if strings.Contains(rawBody, fakeMimoKey) {
		t.Fatal("the response body leaks the raw MiMo API key")
	}
}

// TestSettingsGetShapesMaskMimoAPIKey drives every settings GET shape — list,
// by id, by key, active — and asserts the sealed MiMo key is masked in all of
// them. This is the red-team exposure fix: those GETs are Authenticate-only and
// partner/employee hold GET grants on /settings/*.
func TestSettingsGetShapesMaskMimoAPIKey(t *testing.T) {
	handler, settingsService, _ := newSettingsTestHarness(t)
	keyID := seedSetting(t, settingsService, secret.MimoAPIKey, fakeMimoKey, domain.ValueTypeString)
	seedSetting(t, settingsService, config.SettingKeyMimoDefaultModel, "mimo-v2.6-flash", domain.ValueTypeString)
	router := newSettingsRouter(handler)

	// 1. List
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings?limit=50", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d (%s)", w.Code, w.Body)
	}
	var mimoEntry map[string]any
	for _, entry := range decodeList(t, w.Body.Bytes()) {
		if entry["key"] == "mimo.api_key" {
			mimoEntry = entry
		}
		if entry["key"] == "mimo.default_model" && entry["value"] != "mimo-v2.6-flash" {
			t.Fatalf("plain setting value = %v, want it untouched", entry["value"])
		}
	}
	if mimoEntry == nil {
		t.Fatal("list did not include mimo.api_key")
	}
	assertMasked(t, mimoEntry, w.Body.String())

	// 2. By ID
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/"+strconv.FormatUint(uint64(keyID), 10), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("by-id status = %d (%s)", w.Code, w.Body)
	}
	assertMasked(t, decodeSingle(t, w.Body.Bytes()), w.Body.String())

	// 3. By key
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/key/mimo.api_key", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("by-key status = %d (%s)", w.Code, w.Body)
	}
	assertMasked(t, decodeSingle(t, w.Body.Bytes()), w.Body.String())

	// 4. Active settings
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/active", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("active status = %d (%s)", w.Code, w.Body)
	}
	var active map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &active); err != nil {
		t.Fatalf("decode active: %v", err)
	}
	activeData := active["data"].(map[string]any)
	found := false
	for _, raw := range activeData["settings"].([]any) {
		entry := raw.(map[string]any)
		if entry["key"] == "mimo.api_key" {
			found = true
			assertMasked(t, entry, w.Body.String())
		}
	}
	if !found {
		t.Fatal("active settings did not include mimo.api_key")
	}
}

// fakeClient plays back a provider outcome for the handler test.
type fakeClient struct {
	result *llmport.TestConnectionResult
	err    error
}

func (f *fakeClient) Chat(context.Context, llmport.ChatRequest) (*llmport.ChatResponse, error) {
	panic("Chat must not be called by the test-connection handler")
}

func (f *fakeClient) TestConnection(context.Context) (*llmport.TestConnectionResult, error) {
	return f.result, f.err
}

func newTestConnectionHandler(t *testing.T, settingsService *config.SettingsService, client llmport.Client) *Handler {
	t.Helper()
	mimoService := llmsvc.NewService(
		config.NewSettingsConfigService(settingsService),
		nil,
		func(llmport.Config) (llmport.Client, error) { return client, nil },
	)
	return NewHandler(settingsService, mimoService)
}

func TestTestMimoConnectionSuccess(t *testing.T) {
	_, settingsService, _ := newSettingsTestHarness(t)
	seedSetting(t, settingsService, secret.MimoAPIKey, fakeMimoKey, domain.ValueTypeString)
	handler := newTestConnectionHandler(t, settingsService, &fakeClient{
		result: &llmport.TestConnectionResult{LatencyMS: 234, ModelCount: 2, DefaultModelPresent: true},
	})
	router := newSettingsRouter(handler)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/settings/mimo/test-connection", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body)
	}
	if strings.Contains(w.Body.String(), fakeMimoKey) {
		t.Fatal("the test-connection response leaks the raw MiMo API key")
	}
	body := decodeSingle(t, w.Body.Bytes())
	if body["success"] != true || body["code"] != "ok" {
		t.Fatalf("body = %v, want success/ok", body)
	}
	if body["latency_ms"] != float64(234) || body["model_count"] != float64(2) {
		t.Fatalf("body = %v, want latency 234 and 2 models", body)
	}
	msg, _ := body["message"].(string)
	if !strings.Contains(msg, "Kết nối MiMo thành công") || !strings.Contains(msg, "234 ms") {
		t.Fatalf("message = %q, want the Vietnamese success message with latency", msg)
	}
}

func TestTestMimoConnectionDesignedFailures(t *testing.T) {
	tests := []struct {
		name     string
		seedKey  bool
		client   *fakeClient
		wantCode string
		wantMsg  string
	}{
		{
			name:     "missing key",
			seedKey:  false,
			client:   &fakeClient{},
			wantCode: "missing_key",
			wantMsg:  "Chưa cấu hình Khóa bí mật MiMo",
		},
		{
			name:    "invalid key",
			seedKey: true,
			client: &fakeClient{err: &llmport.APIError{
				Kind: llmport.KindAuth, Message: llmport.MessageFor(llmport.KindAuth),
			}},
			wantCode: "invalid_key",
			wantMsg:  "Khóa API MiMo không hợp lệ",
		},
		{
			name:    "timeout",
			seedKey: true,
			client: &fakeClient{err: &llmport.APIError{
				Kind: llmport.KindTimeout, Message: llmport.MessageFor(llmport.KindTimeout),
			}},
			wantCode: "timeout",
			wantMsg:  "hết thời gian chờ",
		},
		{
			name:    "provider outage",
			seedKey: true,
			client: &fakeClient{err: &llmport.APIError{
				Kind: llmport.KindUnavailable, Message: llmport.MessageFor(llmport.KindUnavailable),
			}},
			wantCode: "unavailable",
			wantMsg:  "gián đoạn",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, settingsService, _ := newSettingsTestHarness(t)
			if tt.seedKey {
				seedSetting(t, settingsService, secret.MimoAPIKey, fakeMimoKey, domain.ValueTypeString)
			}
			router := newSettingsRouter(newTestConnectionHandler(t, settingsService, tt.client))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/settings/mimo/test-connection", nil))
			// Failure is a designed outcome of a connection test: HTTP 200 with
			// success=false + a stable outcome code for the UI error state.
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d (%s)", w.Code, w.Body)
			}
			body := decodeSingle(t, w.Body.Bytes())
			if body["success"] != false || body["code"] != tt.wantCode {
				t.Fatalf("body = %v, want success=false code=%s", body, tt.wantCode)
			}
			msg, _ := body["message"].(string)
			if !strings.Contains(msg, tt.wantMsg) {
				t.Fatalf("message = %q, want Vietnamese fragment %q", msg, tt.wantMsg)
			}
		})
	}
}

// TestMimoSettingsKeysPinSecretAllowlist pins the settings-key literal to the
// secret allowlist entry: a rename on either side must fail loudly here rather
// than silently unsealing the API key.
func TestMimoSettingsKeysPinSecretAllowlist(t *testing.T) {
	if config.SettingKeyMimoAPIKey != secret.MimoAPIKey {
		t.Fatalf("config.SettingKeyMimoAPIKey = %q, secret.MimoAPIKey = %q; must match",
			config.SettingKeyMimoAPIKey, secret.MimoAPIKey)
	}
	if !secret.IsProtectedKey(config.SettingKeyMimoAPIKey) {
		t.Fatal("mimo.api_key must be sealed at rest (secret.ProtectedKeys)")
	}
}
