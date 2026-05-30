package domain

import (
	"context"
	"encoding/json"
	"time"
)

// APIMetric represents an API call metric in the domain
type APIMetric struct {
	ID         uint         `json:"id" gorm:"primarykey;type:bigint unsigned"`
	EndpointID uint         `json:"endpoint_id" gorm:"type:bigint unsigned;not null;index"`
	Endpoint   *APIEndpoint `json:"endpoint,omitempty" gorm:"foreignKey:EndpointID;references:ID"`
	StatusCode int          `json:"status_code" gorm:"not null"`
	UserID     *uint        `json:"user_id" gorm:"type:bigint unsigned;index"`
	DurationMs int64        `json:"duration_ms" gorm:"not null"`
	CalledAt   time.Time    `json:"called_at" gorm:"not null;index"`
}

// TableName specifies the table name for APIMetric
func (APIMetric) TableName() string {
	return "api_metrics"
}

// APIMetricSummary represents aggregated API call statistics
type APIMetricSummary struct {
	Endpoint string                  `json:"endpoint"`
	Count    int64                   `json:"count"`
	Latency  APIMetricLatencySummary `json:"latency"`
	Success  int64                   `json:"success"`
	Error    *APIMetricErrorSummary  `json:"error"`
}

type APIMetricLatencySummary struct {
	P95    int64 `json:"p95"`
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
}

type APIMetricErrorSummary struct {
	Total        int64
	StatusCounts map[string]int64
}

type HighErrorRateEndpoint struct {
	Method     string
	Path       string
	ErrorRate  float64
	TotalCount int64
	ErrorCount int64
}

type UserErrorBreakdown struct {
	UserID     uint      `json:"user_id"`
	Endpoint   string    `json:"endpoint"`
	StatusCode int       `json:"status_code"`
	Count      int64     `json:"count"`
	LastSeen   time.Time `json:"last_seen"`
}

type LatencyTrendPoint struct {
	Period string `json:"period"`
	AvgMs  int64  `json:"avg_ms"`
	P95Ms  int64  `json:"p95_ms"`
	Count  int64  `json:"count"`
}

type SlowestEndpoint struct {
	EndpointID uint   `json:"endpoint_id"`
	Endpoint   string `json:"endpoint"`
	P95Ms      int64  `json:"p95_ms"`
	AvgMs      int64  `json:"avg_ms"`
	Count      int64  `json:"count"`
}

type RecentError struct {
	Endpoint   string    `json:"endpoint"`
	StatusCode int       `json:"status_code"`
	UserID     *uint     `json:"user_id"`
	DurationMs int64     `json:"duration_ms"`
	CalledAt   time.Time `json:"called_at"`
}

func (e APIMetricErrorSummary) MarshalJSON() ([]byte, error) {
	payload := make(map[string]int64, len(e.StatusCounts)+1)
	payload["total"] = e.Total

	for status, count := range e.StatusCounts {
		payload[status] = count
	}

	return json.Marshal(payload)
}

// EndpointInfo holds basic endpoint identity
type EndpointInfo struct {
	EndpointID uint   `json:"endpoint_id"`
	Method     string `json:"method"`
	Path       string `json:"path"`
}

// APIMetricRepository defines the interface for API metric persistence operations
type APIMetricRepository interface {
	// Create records a new API metric
	Create(ctx context.Context, metric *APIMetric) error

	// GetSummary returns aggregated API call statistics for a date range
	GetSummary(ctx context.Context, fromDate, toDate time.Time, sortBy string) ([]APIMetricSummary, error)

	// GetAllSummary returns aggregated API call statistics for all records
	GetAllSummary(ctx context.Context, since time.Time, sortBy string) ([]APIMetricSummary, error)

	// FindOrCreateEndpoint finds an existing endpoint or creates a new one
	FindOrCreateEndpoint(ctx context.Context, method, path string) (uint, error)

	// DeleteOldRecords deletes records older than the specified number of days
	DeleteOldRecords(ctx context.Context, days int) (int64, error)

	// GetHighErrorRateEndpoints returns endpoints with error rate > threshold in the last duration
	GetHighErrorRateEndpoints(ctx context.Context, since time.Time, threshold float64) ([]HighErrorRateEndpoint, error)

	// GetErrorBreakdownByUser returns errors grouped by user_id + endpoint + status_code
	GetErrorBreakdownByUser(ctx context.Context, since time.Time, minStatusCode int, limit int) ([]UserErrorBreakdown, error)

	// GetLatencyTrend returns time-series latency data in hourly or daily buckets
	GetLatencyTrend(ctx context.Context, since time.Time, groupBy string) ([]LatencyTrendPoint, error)

	// GetSlowestEndpoints returns endpoints ranked by p95 latency
	GetSlowestEndpoints(ctx context.Context, since time.Time, limit int) ([]SlowestEndpoint, error)

	// GetRecentErrors returns the most recent N error records with endpoint details
	GetRecentErrors(ctx context.Context, since time.Time, limit int) ([]RecentError, error)

	// GetErrorCount returns the total number of error records (status >= 400) since the given time
	GetErrorCount(ctx context.Context, since time.Time) (int64, error)

	// GetEndpointLatencyTrend returns latency trend for a specific endpoint
	GetEndpointLatencyTrend(ctx context.Context, endpointID uint, since time.Time, groupBy string) ([]LatencyTrendPoint, error)

	// GetTopEndpointIDs returns the top N endpoint IDs by request count in the period
	GetTopEndpointIDs(ctx context.Context, since time.Time, limit int) ([]EndpointInfo, error)
}
