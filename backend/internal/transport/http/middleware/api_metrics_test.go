package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// countingMetricsRepo counts persistence calls. All other interface methods
// embed the nil interface: they are never reached in these tests.
type countingMetricsRepo struct {
	domain.APIMetricRepository
	endpointCalls atomic.Int64
	batchCalls    atomic.Int64
	batchMetrics  atomic.Int64
}

func (r *countingMetricsRepo) FindOrCreateEndpoint(ctx context.Context, method, path string) (uint, error) {
	r.endpointCalls.Add(1)
	return 1, nil
}

func (r *countingMetricsRepo) Create(ctx context.Context, metric *domain.APIMetric) error {
	r.batchCalls.Add(1)
	r.batchMetrics.Add(1)
	return nil
}

func (r *countingMetricsRepo) CreateBatch(ctx context.Context, metrics []*domain.APIMetric) error {
	r.batchCalls.Add(1)
	r.batchMetrics.Add(int64(len(metrics)))
	return nil
}

func newMetricsRouter(repo *countingMetricsRepo) *gin.Engine {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(APIMetrics(repo))
	router.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/metrics/api", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/users/:id", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func runMetricsRequest(t *testing.T, repo *countingMetricsRepo, path string, headerValue string) {
	t.Helper()
	doMetricsRequest(t, newMetricsRouter(repo), path, headerValue)
}

func doMetricsRequest(t *testing.T, router *gin.Engine, path string, headerValue string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if headerValue != "" {
		req.Header.Set(ClientSourceHeader, headerValue)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

// waitForRecord blocks until cond passes or the deadline expires.
// Metrics are flushed asynchronously by the 1s-ticker flusher.
func waitForRecord(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestAPIMetricsRecordsNormalTraffic(t *testing.T) {
	repo := &countingMetricsRepo{}
	runMetricsRequest(t, repo, "/api/v1/ping", "")
	waitForRecord(t, func() bool { return repo.batchMetrics.Load() == 1 })
	if repo.endpointCalls.Load() != 1 || repo.batchMetrics.Load() != 1 {
		t.Fatalf("expected 1 endpoint + 1 recorded metric, got %d/%d",
			repo.endpointCalls.Load(), repo.batchMetrics.Load())
	}
}

func TestAPIMetricsCachesEndpointPerDistinctPath(t *testing.T) {
	repo := &countingMetricsRepo{}
	router := newMetricsRouter(repo)
	// Same normalized path twice (different concrete IDs), then a second
	// distinct path — one middleware closure, so one endpoint cache.
	doMetricsRequest(t, router, "/api/v1/users/5", "")
	doMetricsRequest(t, router, "/api/v1/users/9", "")
	doMetricsRequest(t, router, "/api/v1/ping", "")
	waitForRecord(t, func() bool { return repo.batchMetrics.Load() == 3 })
	if repo.endpointCalls.Load() != 2 {
		t.Fatalf("expected FindOrCreateEndpoint once per distinct normalized path (2), got %d",
			repo.endpointCalls.Load())
	}
	if repo.batchMetrics.Load() != 3 {
		t.Fatalf("expected 3 recorded metrics, got %d", repo.batchMetrics.Load())
	}
}

func TestAPIMetricsSkipsIntegrationTestTraffic(t *testing.T) {
	repo := &countingMetricsRepo{}
	runMetricsRequest(t, repo, "/api/v1/ping", ClientSourceIntegrationTest)
	// Negative assertions cannot block forever: give the flusher goroutine a
	// window to (incorrectly) fire, then require silence.
	time.Sleep(150 * time.Millisecond)
	if repo.endpointCalls.Load() != 0 || repo.batchMetrics.Load() != 0 {
		t.Fatalf("integration-test traffic must not be recorded, got %d endpoint calls + %d metrics",
			repo.endpointCalls.Load(), repo.batchMetrics.Load())
	}
}

func TestAPIMetricsSkipsMetricsPaths(t *testing.T) {
	repo := &countingMetricsRepo{}
	runMetricsRequest(t, repo, "/api/v1/metrics/api", "")
	time.Sleep(150 * time.Millisecond)
	if repo.endpointCalls.Load() != 0 || repo.batchMetrics.Load() != 0 {
		t.Fatalf("metrics paths must not be recorded, got %d endpoint calls + %d metrics",
			repo.endpointCalls.Load(), repo.batchMetrics.Load())
	}
}
