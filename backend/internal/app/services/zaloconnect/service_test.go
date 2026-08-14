package zaloconnect

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
)

// fakeSettingsRepo is an in-memory settingsRepo.
type fakeSettingsRepo struct {
	mu            sync.Mutex
	rows          map[string]*domain.Settings
	beforeCASOnce sync.Once
	beforeCAS     func()
}

type sharedRefreshCoordinator struct {
	mu        sync.Mutex
	attempted chan struct{}
}

func (c *sharedRefreshCoordinator) Acquire(context.Context) (func(context.Context) error, error) {
	if c.attempted != nil {
		c.attempted <- struct{}{}
	}
	c.mu.Lock()
	return func(context.Context) error {
		c.mu.Unlock()
		return nil
	}, nil
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

func (r *fakeSettingsRepo) CompareAndSwapValue(_ context.Context, key, currentValue, nextValue string, valueType domain.SettingsValueType) (bool, error) {
	if r.beforeCAS != nil {
		r.beforeCASOnce.Do(r.beforeCAS)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	row, exists := r.rows[key]
	if !exists || row.Value == nil || *row.Value != currentValue {
		return false, nil
	}
	rowCopy := *row
	rowCopy.Value = &nextValue
	rowCopy.ValueType = valueType
	r.rows[key] = &rowCopy
	return true, nil
}

func TestSaveCredentials_ValidatesAndStoresRotatedPairImmediately(t *testing.T) {
	var oauthCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		oauthCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"successor-access","refresh_token":"successor-refresh","expires_in":3600}`)
	}))
	t.Cleanup(server.Close)

	svc, _ := newTestService(t)
	provider := zalo.NewProvider(svc, zalo.Config{OAuthURL: server.URL, SendURL: server.URL, HTTPTimeout: time.Second}, nil)
	svc.SetProvider(provider)

	err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret", TemplateID: "template",
		AccessToken: "pasted-access", RefreshToken: "pasted-refresh",
	})
	if err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	if oauthCalls != 1 {
		t.Fatalf("OAuth calls = %d, want 1", oauthCalls)
	}
	stored, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.AccessToken != "successor-access" || stored.RefreshToken != "successor-refresh" {
		t.Fatalf("stored pair = %#v, want validated successor", stored)
	}
}

func TestSaveCredentials_InvalidRefreshRejectedWithoutPersistingCandidate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error":-14014,"message":"Invalid refresh token","access_token":""}`)
	}))
	t.Cleanup(server.Close)

	svc, _ := newTestService(t)
	provider := zalo.NewProvider(svc, zalo.Config{OAuthURL: server.URL, SendURL: server.URL, HTTPTimeout: time.Second}, nil)
	svc.SetProvider(provider)

	err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret",
		AccessToken: "temporary-access", RefreshToken: "dead-refresh",
	})
	if err == nil || !strings.Contains(err.Error(), "Refresh Token không hợp lệ") {
		t.Fatalf("SaveCredentials error = %v, want immediate invalid-token error", err)
	}
	if !domain.IsValidationError(err) {
		t.Fatalf("error type = %T, want validation error", err)
	}
	stored, getErr := svc.Get(context.Background())
	if getErr != nil {
		t.Fatalf("Get: %v", getErr)
	}
	if stored.AccessToken != "" || stored.RefreshToken != "" {
		t.Fatalf("rejected candidate was persisted: %#v", stored)
	}
}

func TestSaveCredentials_OAuthOutageIsNotReportedAsInvalidToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `<html>temporary upstream failure</html>`)
	}))
	t.Cleanup(server.Close)

	svc, _ := newTestService(t)
	provider := zalo.NewProvider(svc, zalo.Config{OAuthURL: server.URL, SendURL: server.URL, HTTPTimeout: time.Second}, nil)
	svc.SetProvider(provider)
	err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret",
		AccessToken: "pasted-access", RefreshToken: "possibly-valid-refresh",
	})
	if err == nil {
		t.Fatal("SaveCredentials unexpectedly succeeded during OAuth outage")
	}
	if domain.IsValidationError(err) || strings.Contains(err.Error(), "Refresh Token không hợp lệ") {
		t.Fatalf("OAuth outage was misclassified as invalid token: %v", err)
	}
}

func TestSaveCredentials_AccessOnlyUpdateValidatesExistingRefreshImmediately(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"error":-14014,"message":"Invalid refresh token","access_token":""}`)
	}))
	t.Cleanup(server.Close)

	svc, _ := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret",
		AccessToken: "old-access", RefreshToken: "dead-refresh",
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	provider := zalo.NewProvider(svc, zalo.Config{OAuthURL: server.URL, SendURL: server.URL, HTTPTimeout: time.Second}, nil)
	svc.SetProvider(provider)

	err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{AccessToken: "replacement-access"})
	if err == nil || !domain.IsValidationError(err) {
		t.Fatalf("access-only SaveCredentials error = %v, want invalid existing refresh token", err)
	}
	stored, getErr := svc.Get(context.Background())
	if getErr != nil {
		t.Fatalf("Get: %v", getErr)
	}
	if stored.AccessToken != "old-access" || stored.RefreshToken != "dead-refresh" {
		t.Fatalf("failed validation changed stored pair: %#v", stored)
	}
}

func TestSaveCredentials_ConcurrentDifferentPairReturnsConflictWithoutSecondExchange(t *testing.T) {
	firstExchangeStarted := make(chan struct{})
	releaseFirstExchange := make(chan struct{})
	var callsMu sync.Mutex
	oauthCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callsMu.Lock()
		oauthCalls++
		call := oauthCalls
		callsMu.Unlock()
		if call == 1 {
			close(firstExchangeStarted)
			<-releaseFirstExchange
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"access_token":"successor-access","refresh_token":"successor-refresh","expires_in":3600}`)
	}))
	t.Cleanup(server.Close)

	repo := newFakeSettingsRepo()
	coordinator := &sharedRefreshCoordinator{attempted: make(chan struct{}, 2)}
	services := make([]*Service, 2)
	for i := range services {
		services[i] = NewService(repo, nil)
		provider := zalo.NewProvider(services[i], zalo.Config{OAuthURL: server.URL, SendURL: server.URL, HTTPTimeout: time.Second}, nil)
		provider.SetRefreshCoordinator(coordinator)
		services[i].SetProvider(provider)
	}
	firstInput := SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret",
		AccessToken: "first-access", RefreshToken: "first-refresh",
	}
	secondInput := SaveCredentialsInput{
		AppID: "payroll-app", SecretKey: "secret",
		AccessToken: "different-access", RefreshToken: "different-invalid-refresh",
	}

	errs := make(chan error, 2)
	go func() { errs <- services[0].SaveCredentials(context.Background(), firstInput) }()
	<-coordinator.attempted
	<-firstExchangeStarted
	go func() { errs <- services[1].SaveCredentials(context.Background(), secondInput) }()
	<-coordinator.attempted
	close(releaseFirstExchange)
	successes, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		err := <-errs
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrConflict):
			conflicts++
		default:
			t.Fatalf("SaveCredentials: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("results: successes=%d conflicts=%d, want 1 each", successes, conflicts)
	}
	callsMu.Lock()
	defer callsMu.Unlock()
	if oauthCalls != 1 {
		t.Fatalf("OAuth calls = %d, want one exchange and one conflict", oauthCalls)
	}
}

func TestMutateCredentials_RetriesWithoutRestoringRotatedToken(t *testing.T) {
	svc, repo := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "app", SecretKey: "secret", AccessToken: "old-access", RefreshToken: "old-refresh",
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	repo.beforeCAS = func() {
		repo.mu.Lock()
		defer repo.mu.Unlock()
		row := repo.rows[KeyCredentials]
		rotated := strings.ReplaceAll(*row.Value, "old-access", "new-access")
		rotated = strings.ReplaceAll(rotated, "old-refresh", "new-refresh")
		rowCopy := *row
		rowCopy.Value = &rotated
		repo.rows[KeyCredentials] = &rowCopy
	}

	if err := svc.recordError(context.Background(), "sample failure"); err != nil {
		t.Fatalf("recordError: %v", err)
	}
	stored, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.AccessToken != "new-access" || stored.RefreshToken != "new-refresh" {
		t.Fatalf("stale mutation restored consumed pair: %#v", stored)
	}
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

func TestUpdateTokensPreservesConnectionMetadata(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "current-app", SecretKey: "current-secret", TemplateID: "current-template",
		AccessToken: "old-access", RefreshToken: "old-refresh",
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := svc.UpdateTokens(context.Background(), zalo.Credentials{
		AppID: "stale-app", SecretKey: "stale-secret", TemplateID: "stale-template",
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: &future,
	}); err != nil {
		t.Fatalf("UpdateTokens: %v", err)
	}
	stored, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.AppID != "current-app" || stored.SecretKey != "current-secret" || stored.TemplateID != "current-template" {
		t.Fatalf("token-only update overwrote metadata: %#v", stored)
	}
	if stored.AccessToken != "new-access" || stored.RefreshToken != "new-refresh" {
		t.Fatalf("token-only update did not store successor pair: %#v", stored)
	}
}

func TestUpdateValidatedCredentialsReplacesFullConfiguration(t *testing.T) {
	svc, _ := newTestService(t)
	if err := svc.SaveCredentials(context.Background(), SaveCredentialsInput{
		AppID: "old-app", SecretKey: "old-secret", TemplateID: "old-template",
	}); err != nil {
		t.Fatalf("seed credentials: %v", err)
	}
	future := time.Now().Add(time.Hour)
	if err := svc.Update(context.Background(), zalo.Credentials{
		AppID: "new-app", SecretKey: "new-secret", TemplateID: "new-template",
		AccessToken: "new-access", RefreshToken: "new-refresh", ExpiresAt: &future,
	}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	stored, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if stored.AppID != "new-app" || stored.SecretKey != "new-secret" || stored.TemplateID != "new-template" {
		t.Fatalf("validated update did not replace full configuration: %#v", stored)
	}
}
