package persistence

import (
	"context"
	"strings"
	"testing"

	"api-server/internal/domain"
	"api-server/internal/pkg/secret"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newSettingsSecretsRepo builds an in-memory settings repository whose value
// column is wide open for inspection, so the tests can assert on the bytes at
// rest rather than on what the repository hands back.
func newSettingsSecretsRepo(t *testing.T, cipher *secret.Cipher) (*SettingsRepository, *gorm.DB) {
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
	return NewSettingsRepositoryWithCipher(&Database{DB: db}, cipher), db
}

func testSettingsCipher(t *testing.T, seed byte) *secret.Cipher {
	t.Helper()
	key := make([]byte, secret.KeySize)
	for i := range key {
		key[i] = seed + byte(i)
	}
	cipher, err := secret.NewCipher(key)
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	return cipher
}

// storedValue reads the raw column, bypassing the repository's decryption.
func storedValue(t *testing.T, db *gorm.DB, key string) *string {
	t.Helper()
	var value *string
	if err := db.Raw("SELECT value FROM settings WHERE `key` = ?", key).Scan(&value).Error; err != nil {
		t.Fatalf("read stored value for %q: %v", key, err)
	}
	return value
}

func TestSettingsRepositorySealsProtectedValueAtRest(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 1))
	ctx := context.Background()
	plaintext := `{"app_id":"123","secret_key":"s3cr3t-key"}`
	row := &domain.Settings{Key: secret.ZaloCredentialsKey, Value: &plaintext, ValueType: domain.ValueTypeJSON}

	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if row.ID == 0 {
		t.Fatal("Create did not assign the primary key to the caller's struct")
	}
	if row.Value == nil || *row.Value != plaintext {
		t.Fatalf("caller's value = %v, want the plaintext it passed in", row.Value)
	}

	raw := storedValue(t, db, secret.ZaloCredentialsKey)
	if raw == nil || !secret.IsEncrypted(*raw) {
		t.Fatalf("stored value = %v, want a sealed value", raw)
	}
	if strings.Contains(*raw, "s3cr3t-key") {
		t.Fatal("the stored value still contains the plaintext secret")
	}

	loaded, err := repo.GetByKey(ctx, secret.ZaloCredentialsKey)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if loaded.Value == nil || *loaded.Value != plaintext {
		t.Fatalf("GetByKey value = %v, want %q", loaded.Value, plaintext)
	}

	byID, err := repo.GetByID(ctx, row.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if byID.Value == nil || *byID.Value != plaintext {
		t.Fatalf("GetByID value = %v, want %q", byID.Value, plaintext)
	}

	listed, err := repo.List(ctx, domain.SettingsFilters{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed) != 1 || listed[0].Value == nil || *listed[0].Value != plaintext {
		t.Fatalf("List returned %+v, want the decrypted value", listed)
	}
}

func TestSettingsRepositoryLeavesUnprotectedValuesAlone(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 2))
	ctx := context.Background()
	value := "true"
	row := &domain.Settings{Key: "zalo.enabled", Value: &value, ValueType: domain.ValueTypeBoolean}

	if err := repo.Create(ctx, row); err != nil {
		t.Fatalf("Create: %v", err)
	}
	raw := storedValue(t, db, "zalo.enabled")
	if raw == nil || *raw != value {
		t.Fatalf("stored value = %v, want %q", raw, value)
	}
}

func TestSettingsRepositoryReadsLegacyPlaintextProtectedValue(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 3))
	ctx := context.Background()
	legacy := `{"app_id":"legacy","refresh_token":"old"}`
	if err := db.Exec(
		"INSERT INTO settings (`key`, value, value_type) VALUES (?, ?, ?)",
		secret.ZaloCredentialsKey, legacy, string(domain.ValueTypeJSON),
	).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	loaded, err := repo.GetByKey(ctx, secret.ZaloCredentialsKey)
	if err != nil {
		t.Fatalf("GetByKey(legacy plaintext): %v", err)
	}
	if loaded.Value == nil || *loaded.Value != legacy {
		t.Fatalf("legacy value = %v, want the plaintext back", loaded.Value)
	}

	// Rewriting the row seals it, without the caller doing anything special.
	if err := repo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}
	raw := storedValue(t, db, secret.ZaloCredentialsKey)
	if raw == nil || !secret.IsEncrypted(*raw) {
		t.Fatalf("stored value after Update = %v, want a sealed value", raw)
	}
}

func TestSettingsRepositoryRefusesSealedValueWithoutKey(t *testing.T) {
	sealing, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 4))
	ctx := context.Background()
	plaintext := `{"app_id":"123"}`
	if err := sealing.Create(ctx, &domain.Settings{
		Key: secret.ZaloCredentialsKey, Value: &plaintext, ValueType: domain.ValueTypeJSON,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Same database, no key configured: the sealed row must not be handed back
	// as if it were the credential.
	keyless := NewSettingsRepositoryWithCipher(&Database{DB: db}, nil)
	if _, err := keyless.GetByKey(ctx, secret.ZaloCredentialsKey); err == nil {
		t.Fatal("GetByKey returned a sealed value without a configured key")
	}

	// The degraded mode still reads plaintext and writes plaintext, exactly as
	// the application behaved before encryption existed.
	other := "visible"
	if err := keyless.Create(ctx, &domain.Settings{Key: "transfer_bank_visible", Value: &other, ValueType: domain.ValueTypeString}); err != nil {
		t.Fatalf("Create (no key): %v", err)
	}
	if raw := storedValue(t, db, "transfer_bank_visible"); raw == nil || *raw != other {
		t.Fatalf("stored value without a key = %v, want %q", raw, other)
	}
}

func TestSettingsRepositoryCompareAndSwapSealedValue(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 5))
	ctx := context.Background()
	current := `{"refresh_token":"spent"}`
	if err := repo.Create(ctx, &domain.Settings{
		Key: secret.ZaloCredentialsKey, Value: &current, ValueType: domain.ValueTypeJSON,
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	next := `{"refresh_token":"successor"}`
	updated, err := repo.CompareAndSwapValue(ctx, secret.ZaloCredentialsKey, current, next, domain.ValueTypeJSON)
	if err != nil {
		t.Fatalf("CompareAndSwapValue: %v", err)
	}
	if !updated {
		t.Fatal("CompareAndSwapValue with the current plaintext did not match the sealed row")
	}
	if raw := storedValue(t, db, secret.ZaloCredentialsKey); raw == nil || !secret.IsEncrypted(*raw) {
		t.Fatalf("stored value = %v, want a sealed value", raw)
	}
	loaded, err := repo.GetByKey(ctx, secret.ZaloCredentialsKey)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if loaded.Value == nil || *loaded.Value != next {
		t.Fatalf("stored plaintext = %v, want %q", loaded.Value, next)
	}

	// A stale writer holding the superseded value must lose the race.
	stale := `{"refresh_token":"spent","last_error":"late"}`
	updated, err = repo.CompareAndSwapValue(ctx, secret.ZaloCredentialsKey, current, stale, domain.ValueTypeJSON)
	if err != nil {
		t.Fatalf("stale CompareAndSwapValue: %v", err)
	}
	if updated {
		t.Fatal("stale CompareAndSwapValue overwrote the rotated credential")
	}
	loaded, err = repo.GetByKey(ctx, secret.ZaloCredentialsKey)
	if err != nil {
		t.Fatalf("GetByKey after stale swap: %v", err)
	}
	if loaded.Value == nil || *loaded.Value != next {
		t.Fatalf("value after stale swap = %v, want %q", loaded.Value, next)
	}
}

func TestSettingsRepositoryCompareAndSwapSealedNullValue(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 6))
	ctx := context.Background()
	if err := db.Exec(
		"INSERT INTO settings (`key`, value, value_type) VALUES (?, NULL, ?)",
		secret.ZaloCredentialsKey, string(domain.ValueTypeJSON),
	).Error; err != nil {
		t.Fatalf("insert null row: %v", err)
	}

	if updated, err := repo.CompareAndSwapValue(ctx, secret.ZaloCredentialsKey, "something", "{}", domain.ValueTypeJSON); err != nil || updated {
		t.Fatalf("CompareAndSwapValue against NULL = %v, %v; want false, nil", updated, err)
	}
	updated, err := repo.CompareAndSwapValue(ctx, secret.ZaloCredentialsKey, "", `{"app_id":"123"}`, domain.ValueTypeJSON)
	if err != nil || !updated {
		t.Fatalf("CompareAndSwapValue with empty current = %v, %v; want true, nil", updated, err)
	}
	if raw := storedValue(t, db, secret.ZaloCredentialsKey); raw == nil || !secret.IsEncrypted(*raw) {
		t.Fatalf("stored value = %v, want a sealed value", raw)
	}
}

func TestSettingsRepositoryEncryptProtectedValuesBackfill(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, testSettingsCipher(t, 7))
	ctx := context.Background()
	legacy := `{"app_id":"legacy","secret_key":"plain"}`

	if keys, err := repo.EncryptProtectedValues(ctx); err != nil || len(keys) != 0 {
		t.Fatalf("backfill without rows = %v, %v; want no keys and no error", keys, err)
	}

	if err := db.Exec(
		"INSERT INTO settings (`key`, value, value_type) VALUES (?, ?, ?)",
		secret.ZaloCredentialsKey, legacy, string(domain.ValueTypeJSON),
	).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}
	visible := "true"
	if err := repo.Create(ctx, &domain.Settings{Key: "zalo.enabled", Value: &visible, ValueType: domain.ValueTypeBoolean}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	converted, err := repo.EncryptProtectedValues(ctx)
	if err != nil {
		t.Fatalf("EncryptProtectedValues: %v", err)
	}
	if len(converted) != 1 || converted[0] != secret.ZaloCredentialsKey {
		t.Fatalf("converted = %v, want [%s]", converted, secret.ZaloCredentialsKey)
	}
	sealed := storedValue(t, db, secret.ZaloCredentialsKey)
	if sealed == nil || !secret.IsEncrypted(*sealed) {
		t.Fatalf("stored value = %v, want a sealed value", sealed)
	}

	loaded, err := repo.GetByKey(ctx, secret.ZaloCredentialsKey)
	if err != nil {
		t.Fatalf("GetByKey: %v", err)
	}
	if loaded.Value == nil || *loaded.Value != legacy {
		t.Fatalf("backfilled value = %v, want the original plaintext", loaded.Value)
	}
	if raw := storedValue(t, db, "zalo.enabled"); raw == nil || *raw != visible {
		t.Fatalf("unprotected row changed to %v, want %q", raw, visible)
	}

	// Second boot: nothing left to convert, and the sealed bytes are untouched.
	converted, err = repo.EncryptProtectedValues(ctx)
	if err != nil {
		t.Fatalf("second EncryptProtectedValues: %v", err)
	}
	if len(converted) != 0 {
		t.Fatalf("second backfill converted %v, want nothing", converted)
	}
	if again := storedValue(t, db, secret.ZaloCredentialsKey); again == nil || *again != *sealed {
		t.Fatal("second backfill rewrote an already sealed value")
	}
}

func TestSettingsRepositoryBackfillIsNoOpWithoutKey(t *testing.T) {
	repo, db := newSettingsSecretsRepo(t, nil)
	ctx := context.Background()
	legacy := `{"app_id":"legacy"}`
	if err := db.Exec(
		"INSERT INTO settings (`key`, value, value_type) VALUES (?, ?, ?)",
		secret.ZaloCredentialsKey, legacy, string(domain.ValueTypeJSON),
	).Error; err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	converted, err := repo.EncryptProtectedValues(ctx)
	if err != nil {
		t.Fatalf("EncryptProtectedValues without key: %v", err)
	}
	if len(converted) != 0 {
		t.Fatalf("backfill without a key converted %v, want nothing", converted)
	}
	if raw := storedValue(t, db, secret.ZaloCredentialsKey); raw == nil || *raw != legacy {
		t.Fatalf("stored value = %v, want the untouched plaintext", raw)
	}
}
