package zaloconnect

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
)

// fakeSettingsRepo is an in-memory settingsRepo.
type fakeSettingsRepo struct {
	mu   sync.Mutex
	rows map[string]*domain.Settings
}

func newFakeSettingsRepo() *fakeSettingsRepo {
	return &fakeSettingsRepo{rows: map[string]*domain.Settings{}}
}

func (r *fakeSettingsRepo) GetByKey(_ context.Context, key string) (*domain.Settings, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.rows[key]
	if !ok {
		return nil, domain.NewNotFoundError("setting not found")
	}
	cp := *s
	return &cp, nil
}

func (r *fakeSettingsRepo) Create(_ context.Context, s *domain.Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rows[s.Key]; exists {
		return domain.NewInternalError("setting already exists", nil)
	}
	cp := *s
	r.rows[s.Key] = &cp
	return nil
}

func (r *fakeSettingsRepo) Update(_ context.Context, s *domain.Settings) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.rows[s.Key]; !exists {
		return domain.NewNotFoundError("setting not found")
	}
	cp := *s
	r.rows[s.Key] = &cp
	return nil
}

// newTestService spins up a fake repo and returns the service + repo so the
// test can drive + inspect it. (No redis dependency — the OAuth state store
// was removed along with the OAuth flow.)
func newTestService(t *testing.T) (*Service, *fakeSettingsRepo) {
	t.Helper()
	repo := newFakeSettingsRepo()
	svc := NewService(repo, nil)
	return svc, repo
}

// --- GetStatus --------------------------------------------------------------

func TestGetStatus_NotConfigured(t *testing.T) {
	svc, _ := newTestService(t)
	st, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if st.Configured || st.Connected || st.Enabled {
		t.Errorf("fresh state should be unconfigured/disconnected/disabled: %+v", st)
	}
	if st.AppID != "" {
		t.Errorf("fresh AppID should be empty, got %q", st.AppID)
	}
	if st.TemplateID != "619684" {
		t.Errorf("default template_id = %q, want 619684", st.TemplateID)
	}
}

// --- SaveCredentials --------------------------------------------------------

func TestSaveCredentials_ThenStatusConfigured(t *testing.T) {
	svc, repo := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", SecretKey: "secret1",
	}); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	// Credentials row persisted.
	repo.mu.Lock()
	row, ok := repo.rows[KeyCredentials]
	repo.mu.Unlock()
	if !ok {
		t.Fatal("credentials row not created")
	}
	if !strings.Contains(*row.Value, `"app_id":"app1"`) {
		t.Errorf("credentials JSON = %q", *row.Value)
	}
	// Status reflects configured.
	st, _ := svc.GetStatus(context.Background())
	if !st.Configured {
		t.Error("expected Configured=true after save")
	}
	if st.AppID != "app1" {
		t.Errorf("Status.AppID = %q, want app1 (AppID is public, returned masked)", st.AppID)
	}
	if st.Connected {
		t.Error("expected Connected=false (no tokens pasted yet)")
	}
}

func TestSaveCredentials_PreservesTokensOnAppIDRotate(t *testing.T) {
	svc, _ := newTestService(t)
	// Seed with tokens for app1.
	_ = svc.mutateCredentials(context.Background(), func(c Credentials) Credentials {
		c.AppID = "app1"
		c.SecretKey = "secret1"
		c.AccessToken = "old-access"
		c.RefreshToken = "old-refresh"
		return c
	})
	// Re-saving with a DIFFERENT app_id preserves tokens now — the admin is
	// manually managing tokens, and a stale one surfaces cleanly as -124 on
	// the next Send (Provider retries after refresh).
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app2", SecretKey: "secret2",
	}); err != nil {
		t.Fatalf("err: %v", err)
	}
	got, _ := svc.Get(context.Background())
	if got.AccessToken != "old-access" || got.RefreshToken != "old-refresh" {
		t.Errorf("tokens should be preserved on app_id rotate, got access=%q refresh=%q", got.AccessToken, got.RefreshToken)
	}
}

func TestSaveCredentials_PreservesTokensWhenAppIDUnchanged(t *testing.T) {
	svc, _ := newTestService(t)
	// Seed connected state.
	_ = svc.mutateCredentials(context.Background(), func(c Credentials) Credentials {
		c.AppID = "app1"
		c.SecretKey = "secret1"
		c.AccessToken = "old-access"
		c.RefreshToken = "old-refresh"
		return c
	})
	// Re-saving credentials (same app_id, no token fields) must PRESERVE tokens.
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", SecretKey: "secret1",
	}); err != nil {
		t.Fatalf("err: %v", err)
	}
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != "old-access" || got.RefreshToken != "old-refresh" {
		t.Errorf("tokens should be preserved, got access=%q refresh=%q", got.AccessToken, got.RefreshToken)
	}
	st, _ := svc.GetStatus(context.Background())
	if !st.Connected {
		t.Error("expected Connected=true (tokens preserved)")
	}
}

func TestSaveCredentials_EmptyAppIDKeepsExisting(t *testing.T) {
	// Admin clicks Save with an empty App ID field (write-only pattern —
	// empty means "keep existing"). Must NOT clear the stored App ID.
	svc, _ := newTestService(t)
	_ = svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", SecretKey: "secret1",
	})
	// Re-save with empty App ID + a new access token.
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "", AccessToken: "new-access",
	}); err != nil {
		t.Fatalf("err: %v", err)
	}
	got, _ := svc.Get(context.Background())
	if got.AppID != "app1" {
		t.Errorf("empty App ID should keep existing, got %q", got.AppID)
	}
	if got.AccessToken != "new-access" {
		t.Errorf("access token should be updated, got %q", got.AccessToken)
	}
}

func TestSaveCredentials_ManualTokenPasteUsesAccessTokenBeforeRefresh(t *testing.T) {
	// Freeze the clock so we can assert the estimated access-token lifetime.
	svc, _ := newTestService(t)
	frozen := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	svc.clk = func() time.Time { return frozen }

	// Seed configured (no tokens).
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", SecretKey: "secret1",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Paste access_token + refresh_token manually.
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", AccessToken: "manual-access", RefreshToken: "manual-refresh",
	}); err != nil {
		t.Fatalf("paste: %v", err)
	}
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != "manual-access" || got.RefreshToken != "manual-refresh" {
		t.Errorf("manual paste failed: %+v", got)
	}
	// A manually pasted access token must be tried before consuming its refresh
	// token. If Zalo rejects it with -124, Provider.Send still force-refreshes.
	wantExpiry := frozen.Add(24 * time.Hour)
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("expiry = %v, want %v (use pasted access token first)", got.ExpiresAt, wantExpiry)
	}
	// Re-pasting the same access token is an explicit recovery action. It must
	// reset an expired estimate too; otherwise a production admin could not
	// recover after the former immediate-refresh behavior without changing the
	// token value.
	repastedAt := frozen.Add(time.Hour)
	svc.clk = func() time.Time { return repastedAt }
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", AccessToken: "manual-access",
	}); err != nil {
		t.Fatalf("re-paste: %v", err)
	}
	got, err = svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get after re-paste: %v", err)
	}
	wantExpiry = repastedAt.Add(24 * time.Hour)
	if got.ExpiresAt == nil || !got.ExpiresAt.Equal(wantExpiry) {
		t.Errorf("expiry after re-paste = %v, want %v", got.ExpiresAt, wantExpiry)
	}
	st, _ := svc.GetStatus(context.Background())
	if !st.Connected {
		t.Error("expected Connected=true after manual token paste")
	}
}

func TestSaveCredentials_EmptyTokensDontClobber(t *testing.T) {
	svc, _ := newTestService(t)
	// Seed connected state.
	_ = svc.mutateCredentials(context.Background(), func(c Credentials) Credentials {
		c.AppID = "app1"
		c.SecretKey = "secret1"
		c.AccessToken = "old-access"
		c.RefreshToken = "old-refresh"
		return c
	})
	// Re-save with empty token fields — tokens must survive (UI password fields
	// are empty when the admin doesn't retype them).
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app1", SecretKey: "", AccessToken: "", RefreshToken: "",
	}); err != nil {
		t.Fatalf("err: %v", err)
	}
	got, _ := svc.Get(context.Background())
	if got.AccessToken != "old-access" || got.RefreshToken != "old-refresh" {
		t.Errorf("empty fields clobbered tokens: %+v", got)
	}
}

// --- SetEnabled / IsEnabled -------------------------------------------------

func TestSetEnabled_HotToggle(t *testing.T) {
	svc, _ := newTestService(t)
	on, _ := svc.IsEnabled(context.Background())
	if on {
		t.Error("default should be disabled")
	}
	if err := svc.SetEnabled(context.Background(), true); err != nil {
		t.Fatalf("SetEnabled: %v", err)
	}
	on, _ = svc.IsEnabled(context.Background())
	if !on {
		t.Error("expected enabled after toggle on")
	}
	if err := svc.SetEnabled(context.Background(), false); err != nil {
		t.Fatalf("SetEnabled off: %v", err)
	}
	on, _ = svc.IsEnabled(context.Background())
	if on {
		t.Error("expected disabled after toggle off")
	}
}

// --- SeedFromEnvIfEmpty -----------------------------------------------------

func TestSeedFromEnvIfEmpty_FirstBoot(t *testing.T) {
	svc, repo := newTestService(t)
	err := svc.SeedFromEnvIfEmpty(context.Background(), EnvSeed{
		Enabled: true, AppID: "env-app", SecretKey: "env-secret", TemplateID: "",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	repo.mu.Lock()
	_, hasCreds := repo.rows[KeyCredentials]
	_, hasEnabled := repo.rows[KeyEnabled]
	repo.mu.Unlock()
	if !hasCreds || !hasEnabled {
		t.Error("seed should create both rows")
	}
	on, _ := svc.IsEnabled(context.Background())
	if !on {
		t.Error("seed enabled=true should set flag")
	}
}

func TestSeedFromEnvIfEmpty_NoOpWhenRowExists(t *testing.T) {
	svc, repo := newTestService(t)
	// Pre-create credentials row.
	_ = svc.SaveCredentials(context.Background(), SaveCredentialsInput{AppID: "db-app", SecretKey: "db-secret"})
	before := repo.rows[KeyCredentials].Value

	_ = svc.SeedFromEnvIfEmpty(context.Background(), EnvSeed{AppID: "env-app", SecretKey: "env-secret"})
	after := repo.rows[KeyCredentials].Value
	if before != after {
		t.Error("seed should NOT overwrite existing credentials (DB authoritative)")
	}
}

// --- TestSend --------------------------------------------------------------

func TestTestSend_ErrorsWhenProviderNotWired(t *testing.T) {
	svc, _ := newTestService(t)
	// Fresh service: provider is nil (SetProvider not called).
	_, err := svc.TestSend(context.Background(), "84987654321", "", nil)
	if err == nil {
		t.Fatal("expected error when provider not wired")
	}
}

// --- CredentialSource (Get/Update) -----------------------------------------

func TestGetUpdate_RoundTrip(t *testing.T) {
	svc, _ := newTestService(t)
	_ = svc.SaveCredentials(context.Background(), SaveCredentialsInput{AppID: "app1", SecretKey: "secret1"})

	// Simulate a token refresh via Update.
	future := time.Now().Add(time.Hour)
	err := svc.Update(context.Background(), zalo.Credentials{
		AppID: "app1", SecretKey: "secret1", TemplateID: "617976",
		AccessToken: "a1", RefreshToken: "r1", ExpiresAt: &future,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.AccessToken != "a1" || got.RefreshToken != "r1" {
		t.Errorf("round-trip failed: %+v", got)
	}
	st, _ := svc.GetStatus(context.Background())
	if !st.Connected {
		t.Error("expected Connected after Update with tokens")
	}
}
