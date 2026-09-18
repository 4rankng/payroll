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
	createCalls   atomic.Int64
}

func (r *countingMetricsRepo) FindOrCreateEndpoint(ctx context.Context, method, path string) (uint, error) {
	r.endpointCalls.Add(1)
	return 1, nil
}

func (r *countingMetricsRepo) Create(ctx context.Context, metric *domain.APIMetric) error {
	r.createCalls.Add(1)
	return nil
}

func runMetricsRequest(t *testing.T, repo *countingMetricsRepo, path string, headerValue string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(APIMetrics(repo))
	router.GET("/api/v1/ping", func(c *gin.Context) { c.Status(http.StatusOK) })
	router.GET("/api/v1/metrics/api", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, path, nil)
	if headerValue != "" {
		req.Header.Set(ClientSourceHeader, headerValue)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

// waitForRecord blocks until cond passes or the deadline expires.
// Metrics are recorded asynchronously after the handler completes.
func waitForRecord(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
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
	waitForRecord(t, func() bool { return repo.createCalls.Load() == 1 })
	if repo.endpointCalls.Load() != 1 || repo.createCalls.Load() != 1 {
		t.Fatalf("expected 1 endpoint + 1 create call, got %d/%d", repo.endpointCalls.Load(), repo.createCalls.Load())
	}
}

func TestAPIMetricsSkipsIntegrationTestTraffic(t *testing.T) {
	repo := &countingMetricsRepo{}
	runMetricsRequest(t, repo, "/api/v1/ping", ClientSourceIntegrationTest)
	// Negative assertions cannot block forever: give the recorder goroutine a
	// window to (incorrectly) fire, then require silence.
	time.Sleep(150 * time.Millisecond)
	if repo.endpointCalls.Load() != 0 || repo.createCalls.Load() != 0 {
		t.Fatalf("integration-test traffic must not be recorded, got %d endpoint + %d create calls",
			repo.endpointCalls.Load(), repo.createCalls.Load())
	}
}

func TestAPIMetricsSkipsMetricsPaths(t *testing.T) {
	repo := &countingMetricsRepo{}
	runMetricsRequest(t, repo, "/api/v1/metrics/api", "")
	time.Sleep(150 * time.Millisecond)
	if repo.endpointCalls.Load() != 0 || repo.createCalls.Load() != 0 {
		t.Fatalf("metrics paths must not be recorded, got %d endpoint + %d create calls",
			repo.endpointCalls.Load(), repo.createCalls.Load())
	}
}
