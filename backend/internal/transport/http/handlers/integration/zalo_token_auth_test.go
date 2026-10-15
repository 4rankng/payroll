package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"api-server/internal/app/services/apikey"
	"api-server/internal/domain"
	"api-server/internal/infra/zalo"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// endpointKeyRepo is an in-memory domain.APIKeyRepository for wiring the real
// API-key auth chain over the token endpoint.
type endpointKeyRepo struct {
	byHash map[string]*domain.APIKey
}

func (r *endpointKeyRepo) Create(_ context.Context, key *domain.APIKey) error {
	r.byHash[key.KeyHash] = key
	return nil
}

func (r *endpointKeyRepo) GetByHash(_ context.Context, keyHash string) (*domain.APIKey, error) {
	return r.byHash[keyHash], nil
}

func (r *endpointKeyRepo) List(context.Context) ([]*domain.APIKey, error)   { return nil, nil }
func (r *endpointKeyRepo) Revoke(context.Context, uint, time.Time) error    { return nil }
func (r *endpointKeyRepo) UpdateLastUsedAt(context.Context, uint, time.Time) error {
	return nil
}

func seedEndpointKey(repo *endpointKeyRepo, scope domain.APIKeyScope, revoked bool) string {
	plaintext := "ttk_" + string(scope) + "_secret"
	hash := apikey.HashKey(plaintext)
	row := &domain.APIKey{ID: 1, Name: "TingHire", Scope: scope, KeyHash: hash}
	if revoked {
		now := time.Now()
		row.RevokedAt = &now
	}
	repo.byHash[hash] = row
	return plaintext
}

// TestZaloTokenEndpointAuth pins the pull endpoint's credential contract on
// the live route chain: only an unrevoked key carrying the TingHire scope
// (tinghire — the scope the whole /integration surface uses) may pull
// the OA access token; a foreign-scope key is 403, and a missing or revoked
// key is 401. The 200 body carries the access token and never the refresh
// token.
func TestZaloTokenEndpointAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	creds := &fakeCredentialReader{creds: zalo.Credentials{AccessToken: "live-token"}}

	newRouter := func(repo *endpointKeyRepo) *gin.Engine {
		svc := apikey.NewService(repo, nil)
		m := middleware.NewAPIKeyAuthMiddleware(svc, nil)
		r := gin.New()
		g := r.Group("/api/v1/integration")
		g.Use(m.Authenticate())
		g.Use(m.RequireAPIKeyScope(domain.APIKeyScopeTingHire))
		g.GET("/zalo/token", NewZaloTokenHandler(creds).GetToken)
		return r
	}

	call := func(r *gin.Engine, plaintext string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/integration/zalo/token", nil)
		if plaintext != "" {
			req.Header.Set("X-API-Key", plaintext)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec
	}

	t.Run("vendor key pulls the access token without leaking the refresh token", func(t *testing.T) {
		repo := &endpointKeyRepo{byHash: map[string]*domain.APIKey{}}
		key := seedEndpointKey(repo, domain.APIKeyScopeTingHire, false)

		rec := call(newRouter(repo), key)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), `"access_token":"live-token"`)
		assert.NotContains(t, rec.Body.String(), "refresh")
	})

	t.Run("foreign-scope key is rejected", func(t *testing.T) {
		repo := &endpointKeyRepo{byHash: map[string]*domain.APIKey{}}
		key := seedEndpointKey(repo, domain.APIKeyScope("legacy_other"), false)

		rec := call(newRouter(repo), key)

		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("revoked key is rejected", func(t *testing.T) {
		repo := &endpointKeyRepo{byHash: map[string]*domain.APIKey{}}
		key := seedEndpointKey(repo, domain.APIKeyScopeTimesheetRead, true)

		rec := call(newRouter(repo), key)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("missing key is rejected", func(t *testing.T) {
		repo := &endpointKeyRepo{byHash: map[string]*domain.APIKey{}}

		rec := call(newRouter(repo), "")

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
