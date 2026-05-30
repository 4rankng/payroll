package observability

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	tenantThrottleCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenant_throttle_total",
			Help: "Total number of tenant throttle (semaphore) events",
		},
		[]string{"tenant"},
	)

	tenantQueueDropCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tenant_queue_drop_total",
			Help: "Total number of tasks dropped when tenant queue is full",
		},
		[]string{"tenant"},
	)

	// Cache metrics
	cacheHitsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_type"}, // dashboard, timesheet, bank, settings, etc.
	)

	cacheMissesCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_type"},
	)

	cacheOperationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cache_operation_duration_seconds",
			Help:    "Duration of cache operations in seconds",
			Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
		},
		[]string{"operation", "cache_type"}, // operation: get, set, delete
	)
)

func init() {
	// Register metrics with Prometheus default registry
	prometheus.MustRegister(tenantThrottleCounter)
	prometheus.MustRegister(tenantQueueDropCounter)
	prometheus.MustRegister(cacheHitsCounter)
	prometheus.MustRegister(cacheMissesCounter)
	prometheus.MustRegister(cacheOperationDuration)
}

// IncrementTenantThrottle increments the throttle counter for a tenant
func IncrementTenantThrottle(tenant string) {
	tenantThrottleCounter.WithLabelValues(tenant).Inc()
}

// IncrementTenantQueueDrop increments the queue drop counter for a tenant
func IncrementTenantQueueDrop(tenant string) {
	tenantQueueDropCounter.WithLabelValues(tenant).Inc()
}

// IncrementCacheHit increments the cache hit counter for a specific cache type
func IncrementCacheHit(cacheType string) {
	cacheHitsCounter.WithLabelValues(cacheType).Inc()
}

// IncrementCacheMiss increments the cache miss counter for a specific cache type
func IncrementCacheMiss(cacheType string) {
	cacheMissesCounter.WithLabelValues(cacheType).Inc()
}

// ObserveCacheOperation records the duration of a cache operation
func ObserveCacheOperation(operation, cacheType string, durationSeconds float64) {
	cacheOperationDuration.WithLabelValues(operation, cacheType).Observe(durationSeconds)
}

// MetricsHandler returns an http.Handler suitable for exposing metrics.
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}
