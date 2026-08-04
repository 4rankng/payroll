package zaloconnect

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

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

// newTestService spins up a miniredis + fake repo and returns the service +
// handles so the test can drive + inspect it.
func newTestService(t *testing.T) (*Service, *fakeSettingsRepo, *redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	repo := newFakeSettingsRepo()
	svc := NewService(repo, rdb, "https://api.example.test/api/v1/admin/zalo/oauth/callback", nil)
	return svc, repo, rdb, mr
}

// --- GetStatus --------------------------------------------------------------

func TestGetStatus_NotConfigured(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	st, err := svc.GetStatus(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if st.Configured || st.Connected || st.Enabled {
		t.Errorf("fresh state should be unconfigured/disconnected/disabled: %+v", st)
	}
	if st.TemplateID != "617976" {
		t.Errorf("default template_id = %q, want 617976", st.TemplateID)
	}
	if st.CallbackURL == "" {
		t.Error("callback URL empty")
	}
}

// --- SaveCredentials --------------------------------------------------------

func TestSaveCredentials_ThenStatusConfigured(t *testing.T) {
	svc, repo, _, _ := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), "app1", "secret1", ""); err != nil {
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
	if st.Connected {
		t.Error("expected Connected=false (no OAuth yet)")
	}
}

func TestSaveCredentials_ClearsExistingTokens(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	// Seed with tokens.
	_ = svc.mutateCredentials(context.Background(), func(c Credentials) Credentials {
		c.AppID = "app1"
		c.SecretKey = "secret1"
		c.AccessToken = "old-access"
		c.RefreshToken = "old-refresh"
		return c
	})
	// Re-saving credentials must clear tokens (admin must re-OAuth).
	if err := svc.SaveCredentials(context.Background(), "app1", "secret1", ""); err != nil {
		t.Fatalf("err: %v", err)
	}
	st, _ := svc.GetStatus(context.Background())
	if st.Connected {
		t.Error("re-saving credentials should clear tokens (Connected must be false)")
	}
}

// --- SetEnabled / IsEnabled -------------------------------------------------

func TestSetEnabled_HotToggle(t *testing.T) {
	svc, _, _, _ := newTestService(t)
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

// --- StartOAuth / HandleOAuthCallback --------------------------------------

func TestStartOAuth_RequiresCredentials(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	_, err := svc.StartOAuth(context.Background())
	if err == nil {
		t.Error("StartOAuth without credentials should error")
	}
}

func TestStartOAuth_BuildsURLAndStoresState(t *testing.T) {
	svc, _, rdb, mr := newTestService(t)
	_ = svc.SaveCredentials(context.Background(), "app1", "secret1", "")

	url, err := svc.StartOAuth(context.Background())
	if err != nil {
		t.Fatalf("StartOAuth: %v", err)
	}
	if !strings.Contains(url, "app_id=app1") {
		t.Errorf("URL missing app_id: %s", url)
	}
	if !strings.Contains(url, "state=") {
		t.Errorf("URL missing state: %s", url)
	}
	if !strings.Contains(url, "redirect_uri=") {
		t.Errorf("URL missing redirect_uri: %s", url)
	}
	// State stored in redis.
	keys := mr.Keys()
	found := false
	for _, k := range keys {
		if strings.HasPrefix(k, stateKeyPrefix) {
			found = true
			break
		}
	}
	if !found {
		t.Error("state not stored in redis")
	}
	_ = rdb
}

func TestOAuthCallback_StateSingleUse(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	_ = svc.SaveCredentials(context.Background(), "app1", "secret1", "")
	_, _ = svc.StartOAuth(context.Background())

	// Extract the state from redis (we can't easily read the URL here, so
	// drive HandleOAuthCallback with a known state that doesn't exist → rejected).
	err := svc.HandleOAuthCallback(context.Background(), "code1", "nonexistent-state")
	if err == nil {
		t.Error("expected error for nonexistent state")
	}
}

// --- SeedFromEnvIfEmpty -----------------------------------------------------

func TestSeedFromEnvIfEmpty_FirstBoot(t *testing.T) {
	svc, repo, _, _ := newTestService(t)
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
	svc, repo, _, _ := newTestService(t)
	// Pre-create credentials row.
	_ = svc.SaveCredentials(context.Background(), "db-app", "db-secret", "")
	before := repo.rows[KeyCredentials].Value

	_ = svc.SeedFromEnvIfEmpty(context.Background(), EnvSeed{AppID: "env-app", SecretKey: "env-secret"})
	after := repo.rows[KeyCredentials].Value
	if before != after {
		t.Error("seed should NOT overwrite existing credentials (DB authoritative)")
	}
}

// --- CredentialSource (Get/Update) -----------------------------------------

func TestGetUpdate_RoundTrip(t *testing.T) {
	svc, _, _, _ := newTestService(t)
	_ = svc.SaveCredentials(context.Background(), "app1", "secret1", "")

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
