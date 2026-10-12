package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeKeyRepo lets a test pre-seed a key row without touching a database.
type fakeKeyRepo struct {
	byHash map[string]*domain.APIKey
}

func newFakeKeyRepo() *fakeKeyRepo {
	return &fakeKeyRepo{byHash: map[string]*domain.APIKey{}}
}

func (r *fakeKeyRepo) Create(_ context.Context, key *domain.APIKey) error {
	r.byHash[key.KeyHash] = key
	return nil
}

func (r *fakeKeyRepo) GetByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	if k, ok := r.byHash[hash]; ok {
		return k, nil
	}
	return nil, domain.NewNotFoundError("api key not found")
}

func (r *fakeKeyRepo) List(_ context.Context) ([]*domain.APIKey, error) { return nil, nil }
func (r *fakeKeyRepo) Revoke(_ context.Context, id uint, _ time.Time) error {
	return nil
}
func (r *fakeKeyRepo) UpdateLastUsedAt(_ context.Context, _ uint, _ time.Time) error {
	return nil
}

// runScopeGate wires Authenticate + RequireAPIKeyScope around a trivial
// handler and returns the recorder, so a test can assert status codes.
func runScopeGate(t *testing.T, repo *fakeKeyRepo, plaintext string, required domain.APIKeyScope) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)

	svc := apikey.NewService(repo, nil)
	m := NewAPIKeyAuthMiddleware(svc, nil)

	r := gin.New()
	g := r.Group("/api/v1/vendor")
	g.Use(m.Authenticate())
	g.Use(m.RequireAPIKeyScope(required))
	g.GET("/timesheets", func(c *gin.Context) {
		c.String(http.StatusOK, "reached handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/vendor/timesheets", nil)
	if plaintext != "" {
		req.Header.Set("X-API-Key", plaintext)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

// seedKey inserts a key with the given scope and returns its plaintext.
func seedKey(t *testing.T, repo *fakeKeyRepo, scope domain.APIKeyScope) string {
	t.Helper()
	plaintext := "ttk_" + string(scope) + "_secret"
	repo.byHash[apikey.HashKey(plaintext)] = &domain.APIKey{
		ID: 1, Name: "k", Scope: scope, KeyHash: apikey.HashKey(plaintext),
	}
	return plaintext
}

func TestRequireAPIKeyScope_AllowsMatchingScope(t *testing.T) {
	repo := newFakeKeyRepo()
	key := seedKey(t, repo, domain.APIKeyScopeTimesheetRead)

	rec := runScopeGate(t, repo, key, domain.APIKeyScopeTimesheetRead)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "reached handler", rec.Body.String())
}

func TestRequireAPIKeyScope_RejectsUnknownScopeKey(t *testing.T) {
	repo := newFakeKeyRepo()
	// A key whose stored scope is not the guarded surface (e.g. a legacy
	// pre-removal row) must be refused — fail closed.
	key := seedKey(t, repo, domain.APIKeyScope("legacy-unknown"))

	rec := runScopeGate(t, repo, key, domain.APIKeyScopeTimesheetRead)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

// TestRequireAPIKeyScope_InvalidKeyIs401Not403 pins the ordering guarantee: an
// invalid credential is rejected by Authenticate as 401, and only a genuinely
// valid-but-under-scoped key reaches the guard as 403. That ordering keeps the
// two failure modes distinguishable to a caller that holds no valid key.
func TestRequireAPIKeyScope_InvalidKeyIs401Not403(t *testing.T) {
	repo := newFakeKeyRepo()

	rec := runScopeGate(t, repo, "ttk_garbage", domain.APIKeyScopeTimesheetRead)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAPIKeyScope_MissingHeaderIs401(t *testing.T) {
	repo := newFakeKeyRepo()

	rec := runScopeGate(t, repo, "", domain.APIKeyScopeTimesheetRead)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

// TestRequireAPIKeyScope_LegacyEmptyScopeFailsClosed proves a row with an
// empty stored scope matches no guarded surface at all.
func TestRequireAPIKeyScope_LegacyEmptyScopeFailsClosed(t *testing.T) {
	repo := newFakeKeyRepo()
	plaintext := "ttk_legacy_empty_scope"
	repo.byHash[apikey.HashKey(plaintext)] = &domain.APIKey{
		ID: 7, Name: "legacy", Scope: "", KeyHash: apikey.HashKey(plaintext),
	}

	rec := runScopeGate(t, repo, plaintext, domain.APIKeyScopeTimesheetRead)

	assert.Equal(t, http.StatusForbidden, rec.Code, "legacy empty scope must match nothing")
}

func TestRequireAPIKeyScope_MissingScopeInContextFailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := apikey.NewService(newFakeKeyRepo(), nil)
	m := NewAPIKeyAuthMiddleware(svc, nil)

	// Mount ONLY the guard, as a routing mistake that skipped Authenticate
	// would. The guarded route must refuse rather than become unprotected.
	r := gin.New()
	r.Use(m.RequireAPIKeyScope(domain.APIKeyScopeTimesheetRead))
	r.GET("/timesheets", func(c *gin.Context) { c.String(http.StatusOK, "reached handler") })

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/timesheets", nil))

	require.Equal(t, http.StatusForbidden, rec.Code)
	assert.NotContains(t, rec.Body.String(), "reached handler")
}

// TestAuthenticate_PublishesScopeInContext checks the middleware actually
// writes the raw scope the guard reads.
func TestAuthenticate_PublishesScopeInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := newFakeKeyRepo()
	plaintext := "ttk_vendor"
	repo.byHash[apikey.HashKey(plaintext)] = &domain.APIKey{
		ID: 3, Name: "vendor", Scope: domain.APIKeyScopeTimesheetRead, KeyHash: apikey.HashKey(plaintext),
	}

	svc := apikey.NewService(repo, nil)
	m := NewAPIKeyAuthMiddleware(svc, nil)

	r := gin.New()
	r.Use(m.Authenticate())
	var got any
	r.GET("/x", func(c *gin.Context) {
		got, _ = c.Get(constants.CtxAPIKeyScope)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-API-Key", plaintext)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	// Raw on publish: the guard compares the stored scope as-is so an empty or
	// unknown value fails closed.
	assert.Equal(t, string(domain.APIKeyScopeTimesheetRead), got)
}
