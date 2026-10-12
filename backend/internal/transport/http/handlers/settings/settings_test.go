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
	"api-server/internal/domain"
	"api-server/internal/infra/persistence"
	"api-server/internal/pkg/secret"
)

const fakeSealedSecret = "zalo-secret-test-fake-key-never-real"

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
	handler := NewHandler(settingsService)
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
		t.Fatalf("decode data: %v (%s)", env.Data, err)
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
		t.Fatalf("decode data: %v (%s)", env.Data, err)
	}
	return list
}

// assertMasked asserts the protected-key entry is masked: no value, masked and
// configured flags set — and the plaintext appears nowhere in the payload.
func assertMasked(t *testing.T, entry map[string]any, rawBody string) {
	t.Helper()
	if entry["key"] != secret.ZaloCredentialsKey {
		t.Fatalf("entry = %v, want the %s row", entry, secret.ZaloCredentialsKey)
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
	if strings.Contains(rawBody, fakeSealedSecret) {
		t.Fatal("the response body leaks the raw sealed secret")
	}
}

// TestSettingsGetShapesMaskProtectedKeys drives every settings GET shape —
// list, by id, by key, active — and asserts a sealed key is masked in all of
// them. This is the red-team exposure fix: those GETs are Authenticate-only and
// partner/employee hold GET grants on /settings/*.
func TestSettingsGetShapesMaskProtectedKeys(t *testing.T) {
	handler, settingsService, _ := newSettingsTestHarness(t)
	keyID := seedSetting(t, settingsService, secret.ZaloCredentialsKey, fakeSealedSecret, domain.ValueTypeString)
	seedSetting(t, settingsService, config.SettingKeyPartnerCompany, "CONG TY TEST", domain.ValueTypeString)
	router := newSettingsRouter(handler)

	// 1. List
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings?limit=50", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d (%s)", w.Code, w.Body)
	}
	var sealedEntry map[string]any
	for _, entry := range decodeList(t, w.Body.Bytes()) {
		if entry["key"] == secret.ZaloCredentialsKey {
			sealedEntry = entry
		}
		if entry["key"] == config.SettingKeyPartnerCompany && entry["value"] != "CONG TY TEST" {
			t.Fatalf("plain setting value = %v, want it untouched", entry["value"])
		}
	}
	if sealedEntry == nil {
		t.Fatalf("list did not include %s", secret.ZaloCredentialsKey)
	}
	assertMasked(t, sealedEntry, w.Body.String())

	// 2. By ID
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/"+strconv.FormatUint(uint64(keyID), 10), nil))
	if w.Code != http.StatusOK {
		t.Fatalf("by-id status = %d (%s)", w.Code, w.Body)
	}
	assertMasked(t, decodeSingle(t, w.Body.Bytes()), w.Body.String())

	// 3. By key
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/settings/key/"+secret.ZaloCredentialsKey, nil))
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
		if entry["key"] == secret.ZaloCredentialsKey {
			found = true
			assertMasked(t, entry, w.Body.String())
		}
	}
	if !found {
		t.Fatalf("active settings did not include %s", secret.ZaloCredentialsKey)
	}
}
