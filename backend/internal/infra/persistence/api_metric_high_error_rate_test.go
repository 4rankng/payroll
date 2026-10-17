package persistence

import (
	"context"
	"net/http"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newHighErrorRateRepo wires APIMetricRepository over an in-memory SQLite
// database with the two tables the alert query reads (api_endpoints,
// api_metrics). SQLite-friendly types stand in for the MySQL `bigint
// unsigned` columns — an AutoMigrated non-INTEGER primary key is not a rowid
// alias in SQLite, so ids would come back NULL and the endpoint join would
// silently match nothing. The behavior under test — which rows enter the
// error-rate numerator/denominator — is independent of those types.
func newHighErrorRateRepo(t *testing.T) *APIMetricRepository {
	t.Helper()
	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.Exec(`CREATE TABLE api_endpoints (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		method VARCHAR(10) NOT NULL,
		path VARCHAR(255) NOT NULL,
		created_at DATETIME NOT NULL
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE api_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		endpoint_id INTEGER NOT NULL,
		status_code INTEGER NOT NULL,
		user_id INTEGER,
		duration_ms INTEGER NOT NULL,
		called_at DATETIME NOT NULL
	)`).Error)
	return &APIMetricRepository{BaseRepository: &BaseRepository{DB: db}}
}

// seedEndpointMetrics records count metrics of a single status for one
// endpoint (method + path), all inside the 24h alert window.
func seedEndpointMetrics(t *testing.T, repo *APIMetricRepository, method, path string, status, count int) {
	t.Helper()
	ctx := context.Background()
	endpointID, err := repo.FindOrCreateEndpoint(ctx, method, path)
	require.NoError(t, err)
	calledAt := time.Now().Add(-time.Hour)
	for i := 0; i < count; i++ {
		require.NoError(t, repo.Create(ctx, &domain.APIMetric{
			EndpointID: endpointID,
			StatusCode: status,
			DurationMs: 15,
			CalledAt:   calledAt,
		}))
	}
}

// TestGetHighErrorRateEndpointsStatelessMCPProbes pins the alert semantics:
// protocol-correct 405s on the stateless MCP transport must never surface as
// a high-error-rate endpoint, while every other failure — including real 401
// storms on the very same mount — must still alert.
func TestGetHighErrorRateEndpointsStatelessMCPProbes(t *testing.T) {
	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	t.Run("GET and HEAD 405 probes on the MCP mount do not alert", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		// The prod alert shape: ~2 probes/h for 24h, all 405, zero real traffic.
		seedEndpointMetrics(t, repo, "GET", "/api/v1/mcp", http.StatusMethodNotAllowed, 51)
		seedEndpointMetrics(t, repo, "HEAD", "/api/v1/mcp", http.StatusMethodNotAllowed, 51)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Empty(t, results, "pure 405 probe traffic must not raise the alert")
	})

	t.Run("DELETE 405 on the MCP mount does not alert", func(t *testing.T) {
		// Same SDK rule as GET: stateless mode has no session to DELETE.
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "DELETE", "/api/v1/mcp", http.StatusMethodNotAllowed, 51)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("401 storm on the MCP mount still alerts at full weight", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/mcp", http.StatusUnauthorized, 51)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "GET", results[0].Method)
		require.Equal(t, "/api/v1/mcp", results[0].Path)
		require.Equal(t, int64(51), results[0].TotalCount)
		require.Equal(t, int64(51), results[0].ErrorCount)
		require.InDelta(t, 1.0, results[0].ErrorRate, 1e-9)
	})

	t.Run("405 probes do not dilute a concurrent 401 storm", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/mcp", http.StatusMethodNotAllowed, 51)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/mcp", http.StatusUnauthorized, 51)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Len(t, results, 1, "the storm must be measured on real traffic only")
		require.Equal(t, int64(51), results[0].TotalCount, "probes are excluded from the denominator too")
		require.InDelta(t, 1.0, results[0].ErrorRate, 1e-9)
	})

	t.Run("405 on any other endpoint still alerts", func(t *testing.T) {
		// The exclusion is scoped to the MCP mount: a 405 anywhere else is a
		// routing misconfiguration worth waking someone up for.
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "DELETE", "/api/v1/employees", http.StatusMethodNotAllowed, 51)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, "/api/v1/employees", results[0].Path)
	})

	t.Run("ordinary failing endpoints are unaffected", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/users", http.StatusInternalServerError, 28)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/users", http.StatusOK, 36)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, int64(64), results[0].TotalCount)
		require.Equal(t, int64(28), results[0].ErrorCount)
		require.InDelta(t, 28.0/64.0, results[0].ErrorRate, 1e-9)
	})

	t.Run("login endpoint stays excluded", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "POST", "/api/v1/auth/login", http.StatusUnauthorized, 80)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("below the volume floor does not alert", func(t *testing.T) {
		repo := newHighErrorRateRepo(t)
		seedEndpointMetrics(t, repo, "GET", "/api/v1/mcp", http.StatusUnauthorized, 10)

		results, err := repo.GetHighErrorRateEndpoints(ctx, since, 0.10)
		require.NoError(t, err)
		require.Empty(t, results)
	})
}
