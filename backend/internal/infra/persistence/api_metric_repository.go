package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

type APIMetricRepository struct {
	*BaseRepository
}

func NewAPIMetricRepository(db *Database) domain.APIMetricRepository {
	return &APIMetricRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// calculatePercentile computes the p-th percentile from a slice of values
func calculatePercentile(values []int64, p float64) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	index := int(float64(len(values)-1) * p)
	return values[index]
}

func (r *APIMetricRepository) Create(ctx context.Context, metric *domain.APIMetric) error {
	// The APIMetric uses an auto-incrementing primary key, so duplicate errors should not occur
	// under normal circumstances. But let's handle any potential issues gracefully.
	if err := r.DB.WithContext(ctx).Create(metric).Error; err != nil {
		return domain.NewInternalError("failed to create API metric", err)
	}
	return nil
}

// FindOrCreateEndpoint finds an existing endpoint or creates a new one
func (r *APIMetricRepository) FindOrCreateEndpoint(ctx context.Context, method, path string) (uint, error) {
	var endpoint domain.APIEndpoint

	// Try to find existing endpoint first
	err := r.DB.WithContext(ctx).
		Where("method = ? AND path = ?", method, path).
		First(&endpoint).Error

	if err == nil {
		// Endpoint exists, return its ID
		return endpoint.ID, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		// Some other error occurred
		return 0, domain.NewInternalError("failed to query API endpoint", err)
	}

	// Endpoint doesn't exist, try to create it
	endpoint = domain.APIEndpoint{
		Method: method,
		Path:   path,
	}

	// Create with error handling for potential race conditions
	if err := r.DB.WithContext(ctx).Create(&endpoint).Error; err != nil {
		// Check if the error is a duplicate key error (race condition)
		if isDuplicateKeyError(err) {
			// If it's a duplicate key error, another goroutine created the endpoint
			// Try to fetch the existing record
			var existingEndpoint domain.APIEndpoint
			err = r.DB.WithContext(ctx).
				Where("method = ? AND path = ?", method, path).
				First(&existingEndpoint).Error

			if err != nil {
				return 0, domain.NewInternalError("failed to query API endpoint after conflict", err)
			}
			return existingEndpoint.ID, nil
		}
		return 0, domain.NewInternalError("failed to create API endpoint", err)
	}

	return endpoint.ID, nil
}

func (r *APIMetricRepository) GetSummary(ctx context.Context, fromDate, toDate time.Time, sortBy string) ([]domain.APIMetricSummary, error) {
	var results []struct {
		EndpointID uint
		Method     string
		Path       string
		Count      int64
		MinLatency int64
		MaxLatency int64
		AvgLatency float64
		Success    int64
		Error      int64
	}

	// Map sortBy to SQL column (prevent SQL injection)
	sortByMap := map[string]string{
		"count":          "count DESC",
		"latency_p95":    "latency_p95 DESC",
		"latency_median": "median_latency DESC",
		"error_total":    "error DESC",
	}

	orderBy := sortByMap[sortBy]
	if orderBy == "" {
		orderBy = "count DESC" // Default fallback
	}

	// Query to calculate min, max, avg, success count, and error count
	// Percentiles (p95, median) will be calculated in Go to avoid GROUP_CONCAT limitations
	query := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select(`
			ae.id as endpoint_id,
			ae.method,
			ae.path,
			COUNT(*) as count,
			MIN(am.duration_ms) as min_latency,
			MAX(am.duration_ms) as max_latency,
			AVG(am.duration_ms) as avg_latency,
			SUM(CASE WHEN am.status_code >= 200 AND am.status_code < 300 THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN am.status_code >= 400 THEN 1 ELSE 0 END) as error
		`).
		Where("am.called_at >= ? AND am.called_at <= ?", fromDate, toDate).
		Group("ae.id, ae.method, ae.path").
		Order(orderBy)

	if err := query.Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics summary", err)
	}

	// Fetch all duration_ms values per endpoint for percentile calculation
	var durationRows []struct {
		EndpointID uint
		DurationMs int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("am.endpoint_id, am.duration_ms").
		Where("am.called_at >= ? AND am.called_at <= ?", fromDate, toDate).
		Find(&durationRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics durations", err)
	}

	// Group durations by endpoint for percentile calculation
	durationsByEndpoint := make(map[uint][]int64)
	for _, row := range durationRows {
		durationsByEndpoint[row.EndpointID] = append(durationsByEndpoint[row.EndpointID], row.DurationMs)
	}

	var errorRows []struct {
		EndpointID uint
		StatusCode int
		Count      int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics").
		Select("endpoint_id, status_code, COUNT(*) as count").
		Where("called_at >= ? AND called_at <= ? AND status_code >= 400", fromDate, toDate).
		Group("endpoint_id, status_code").
		Find(&errorRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics summary", err)
	}

	errorDetails := make(map[uint]map[string]int64)
	for _, row := range errorRows {
		if row.EndpointID == 0 {
			continue
		}

		details, ok := errorDetails[row.EndpointID]
		if !ok {
			details = make(map[string]int64)
			errorDetails[row.EndpointID] = details
		}

		details[strconv.Itoa(row.StatusCode)] = row.Count
	}

	summary := make([]domain.APIMetricSummary, 0, len(results))
	for _, result := range results {
		endpoint := fmt.Sprintf("%s %s", result.Method, result.Path)
		var statusCounts map[string]int64
		if details, ok := errorDetails[result.EndpointID]; ok && len(details) > 0 {
			statusCounts = make(map[string]int64, len(details))
			for code, count := range details {
				statusCounts[code] = count
			}
		}

		var errorSummary *domain.APIMetricErrorSummary
		if result.Error > 0 {
			if statusCounts == nil {
				statusCounts = make(map[string]int64)
			}
			errorSummary = &domain.APIMetricErrorSummary{
				Total:        result.Error,
				StatusCounts: statusCounts,
			}
		}

		// Calculate percentiles in Go from the duration values
		durations := durationsByEndpoint[result.EndpointID]
		p95 := calculatePercentile(durations, 0.95)
		median := calculatePercentile(durations, 0.50)

		summary = append(summary, domain.APIMetricSummary{
			Endpoint: endpoint,
			Count:    result.Count,
			Latency: domain.APIMetricLatencySummary{
				P95:    p95,
				Min:    result.MinLatency,
				Max:    result.MaxLatency,
				Avg:    int64(math.Round(result.AvgLatency)),
				Median: median,
			},
			Success: result.Success,
			Error:   errorSummary,
		})
	}

	return summary, nil
}

func (r *APIMetricRepository) GetAllSummary(ctx context.Context, since time.Time, sortBy string) ([]domain.APIMetricSummary, error) {
	var results []struct {
		EndpointID uint
		Method     string
		Path       string
		Count      int64
		MinLatency int64
		MaxLatency int64
		AvgLatency float64
		Success    int64
		Error      int64
	}

	// Map sortBy to SQL column (prevent SQL injection)
	sortByMap := map[string]string{
		"count":          "count DESC",
		"latency_p95":    "latency_p95 DESC",
		"latency_median": "median_latency DESC",
		"error_total":    "error DESC",
	}

	orderBy := sortByMap[sortBy]
	if orderBy == "" {
		orderBy = "count DESC" // Default fallback
	}

	// Query to calculate min, max, avg, success count, and error count
	// Percentiles (p95, median) will be calculated in Go to avoid GROUP_CONCAT limitations
	query := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select(`
			ae.id as endpoint_id,
			ae.method,
			ae.path,
			COUNT(*) as count,
			MIN(am.duration_ms) as min_latency,
			MAX(am.duration_ms) as max_latency,
			AVG(am.duration_ms) as avg_latency,
			SUM(CASE WHEN am.status_code >= 200 AND am.status_code < 300 THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN am.status_code >= 400 THEN 1 ELSE 0 END) as error
		`).
		Where("am.called_at >= ?", since).
		Group("ae.id, ae.method, ae.path").
		Order(orderBy)

	if err := query.Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics summary", err)
	}

	// Fetch all duration_ms values per endpoint for percentile calculation
	var durationRows []struct {
		EndpointID uint
		DurationMs int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("am.endpoint_id, am.duration_ms").
		Where("am.called_at >= ?", since).
		Find(&durationRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics durations", err)
	}

	// Group durations by endpoint for percentile calculation
	durationsByEndpoint := make(map[uint][]int64)
	for _, row := range durationRows {
		durationsByEndpoint[row.EndpointID] = append(durationsByEndpoint[row.EndpointID], row.DurationMs)
	}

	var errorRows []struct {
		EndpointID uint
		StatusCode int
		Count      int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics").
		Select("endpoint_id, status_code, COUNT(*) as count").
		Where("status_code >= 400").
		Where("called_at >= ?", since).
		Group("endpoint_id, status_code").
		Find(&errorRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get API metrics summary", err)
	}

	errorDetails := make(map[uint]map[string]int64)
	for _, row := range errorRows {
		if row.EndpointID == 0 {
			continue
		}

		details, ok := errorDetails[row.EndpointID]
		if !ok {
			details = make(map[string]int64)
			errorDetails[row.EndpointID] = details
		}

		details[strconv.Itoa(row.StatusCode)] = row.Count
	}

	summary := make([]domain.APIMetricSummary, 0, len(results))
	for _, result := range results {
		endpoint := fmt.Sprintf("%s %s", result.Method, result.Path)
		var statusCounts map[string]int64
		if details, ok := errorDetails[result.EndpointID]; ok && len(details) > 0 {
			statusCounts = make(map[string]int64, len(details))
			for code, count := range details {
				statusCounts[code] = count
			}
		}

		var errorSummary *domain.APIMetricErrorSummary
		if result.Error > 0 {
			if statusCounts == nil {
				statusCounts = make(map[string]int64)
			}
			errorSummary = &domain.APIMetricErrorSummary{
				Total:        result.Error,
				StatusCounts: statusCounts,
			}
		}

		// Calculate percentiles in Go from the duration values
		durations := durationsByEndpoint[result.EndpointID]
		p95 := calculatePercentile(durations, 0.95)
		median := calculatePercentile(durations, 0.50)

		summary = append(summary, domain.APIMetricSummary{
			Endpoint: endpoint,
			Count:    result.Count,
			Latency: domain.APIMetricLatencySummary{
				P95:    p95,
				Min:    result.MinLatency,
				Max:    result.MaxLatency,
				Avg:    int64(math.Round(result.AvgLatency)),
				Median: median,
			},
			Success: result.Success,
			Error:   errorSummary,
		})
	}

	return summary, nil
}

func (r *APIMetricRepository) DeleteOldRecords(ctx context.Context, days int) (int64, error) {
	cutoffDate := clock.Now().AddDate(0, 0, -days)

	result := r.DB.WithContext(ctx).
		Where("called_at < ?", cutoffDate).
		Delete(&domain.APIMetric{})

	if result.Error != nil {
		return 0, domain.NewInternalError("failed to delete old API metrics", result.Error)
	}

	return result.RowsAffected, nil
}

// isDuplicateKeyError checks if the error is caused by a duplicate key violation
func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "Duplicate entry") || // MySQL
		strings.Contains(errStr, "UNIQUE constraint failed") || // SQLite
		strings.Contains(errStr, "duplicate key value violates unique constraint") // PostgreSQL
}

func (r *APIMetricRepository) GetHighErrorRateEndpoints(ctx context.Context, since time.Time, threshold float64) ([]domain.HighErrorRateEndpoint, error) {
	var results []domain.HighErrorRateEndpoint

	// Query to calculate error rate for each endpoint
	// Exclude login endpoint (path contains 'login')
	// Only consider endpoints with total_count >= 50
	query := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select(`
			ae.method,
			ae.path,
			COUNT(*) as total_count,
			SUM(CASE WHEN am.status_code >= 400 THEN 1 ELSE 0 END) as error_count,
			CAST(SUM(CASE WHEN am.status_code >= 400 THEN 1 ELSE 0 END) AS FLOAT) / COUNT(*) as error_rate
		`).
		Where("am.called_at >= ?", since).
		Where("ae.path NOT LIKE ?", "%login%").
		Group("ae.id, ae.method, ae.path").
		Having("total_count >= 50 AND error_rate > ?", threshold).
		Order("error_rate DESC")

	if err := query.Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get high error rate endpoints", err)
	}

	return results, nil
}

func (r *APIMetricRepository) GetErrorBreakdownByUser(ctx context.Context, since time.Time, minStatusCode int, limit int) ([]domain.UserErrorBreakdown, error) {
	var results []struct {
		UserID     uint
		Method     string
		Path       string
		StatusCode int
		Count      int64
		LastSeen   time.Time
	}

	query := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select("am.user_id, ae.method, ae.path, am.status_code, COUNT(*) as count, MAX(am.called_at) as last_seen").
		Where("am.status_code >= ? AND am.called_at >= ? AND am.user_id IS NOT NULL", minStatusCode, since).
		Group("am.user_id, ae.id, ae.method, ae.path, am.status_code").
		Order("count DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get error breakdown by user", err)
	}

	breakdown := make([]domain.UserErrorBreakdown, 0, len(results))
	for _, row := range results {
		breakdown = append(breakdown, domain.UserErrorBreakdown{
			UserID:     row.UserID,
			Endpoint:   fmt.Sprintf("%s %s", row.Method, row.Path),
			StatusCode: row.StatusCode,
			Count:      row.Count,
			LastSeen:   row.LastSeen,
		})
	}

	return breakdown, nil
}

func (r *APIMetricRepository) GetLatencyTrend(ctx context.Context, since time.Time, groupBy string) ([]domain.LatencyTrendPoint, error) {
	dateFormat := "%Y-%m-%d"
	if groupBy == "hour" {
		dateFormat = "%Y-%m-%d %H:00"
	}

	var results []struct {
		Period string
		AvgMs  float64
		Count  int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("DATE_FORMAT(am.called_at, ?) as period, AVG(am.duration_ms) as avg_ms, COUNT(*) as count", dateFormat).
		Where("am.called_at >= ?", since).
		Group("period").
		Order("period ASC").
		Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get latency trend", err)
	}

	var durationRows []struct {
		Period     string
		DurationMs int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("DATE_FORMAT(am.called_at, ?) as period, am.duration_ms", dateFormat).
		Where("am.called_at >= ?", since).
		Find(&durationRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get latency trend durations", err)
	}

	durationsByPeriod := make(map[string][]int64)
	for _, row := range durationRows {
		durationsByPeriod[row.Period] = append(durationsByPeriod[row.Period], row.DurationMs)
	}

	trend := make([]domain.LatencyTrendPoint, 0, len(results))
	for _, result := range results {
		p95 := calculatePercentile(durationsByPeriod[result.Period], 0.95)
		trend = append(trend, domain.LatencyTrendPoint{
			Period: result.Period,
			AvgMs:  int64(math.Round(result.AvgMs)),
			P95Ms:  p95,
			Count:  result.Count,
		})
	}

	return trend, nil
}

func (r *APIMetricRepository) GetSlowestEndpoints(ctx context.Context, since time.Time, limit int) ([]domain.SlowestEndpoint, error) {
	var results []struct {
		EndpointID uint
		Method     string
		Path       string
		AvgMs      float64
		Count      int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select("ae.id as endpoint_id, ae.method, ae.path, AVG(am.duration_ms) as avg_ms, COUNT(*) as count").
		Where("am.called_at >= ?", since).
		Group("ae.id, ae.method, ae.path").
		Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get slowest endpoints", err)
	}

	var durationRows []struct {
		EndpointID uint
		DurationMs int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("am.endpoint_id, am.duration_ms").
		Where("am.called_at >= ?", since).
		Find(&durationRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get slowest endpoint durations", err)
	}

	durationsByEndpoint := make(map[uint][]int64)
	for _, row := range durationRows {
		durationsByEndpoint[row.EndpointID] = append(durationsByEndpoint[row.EndpointID], row.DurationMs)
	}

	endpoints := make([]domain.SlowestEndpoint, 0, len(results))
	for _, result := range results {
		p95 := calculatePercentile(durationsByEndpoint[result.EndpointID], 0.95)
		endpoints = append(endpoints, domain.SlowestEndpoint{
			EndpointID: result.EndpointID,
			Endpoint:   fmt.Sprintf("%s %s", result.Method, result.Path),
			P95Ms:      p95,
			AvgMs:      int64(math.Round(result.AvgMs)),
			Count:      result.Count,
		})
	}

	sort.Slice(endpoints, func(i, j int) bool { return endpoints[i].P95Ms > endpoints[j].P95Ms })

	if limit > 0 && len(endpoints) > limit {
		endpoints = endpoints[:limit]
	}

	return endpoints, nil
}

func (r *APIMetricRepository) GetRecentErrors(ctx context.Context, since time.Time, limit int) ([]domain.RecentError, error) {
	var results []struct {
		Method     string
		Path       string
		StatusCode int
		UserID     *uint
		DurationMs int64
		CalledAt   time.Time
	}

	query := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select("ae.method, ae.path, am.status_code, am.user_id, am.duration_ms, am.called_at").
		Where("am.status_code >= 400 AND am.called_at >= ?", since).
		Order("am.called_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get recent errors", err)
	}

	errors := make([]domain.RecentError, 0, len(results))
	for _, row := range results {
		errors = append(errors, domain.RecentError{
			Endpoint:   fmt.Sprintf("%s %s", row.Method, row.Path),
			StatusCode: row.StatusCode,
			UserID:     row.UserID,
			DurationMs: row.DurationMs,
			CalledAt:   row.CalledAt,
		})
	}

	return errors, nil
}

func (r *APIMetricRepository) GetErrorCount(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).
		Table("api_metrics").
		Where("status_code >= 400 AND called_at >= ?", since).
		Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count errors", err)
	}
	return count, nil
}

func (r *APIMetricRepository) GetTopEndpointIDs(ctx context.Context, since time.Time, limit int) ([]domain.EndpointInfo, error) {
	var results []domain.EndpointInfo

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Joins("INNER JOIN api_endpoints ae ON am.endpoint_id = ae.id").
		Select("ae.id as endpoint_id, ae.method, ae.path").
		Where("am.called_at >= ?", since).
		Group("ae.id, ae.method, ae.path").
		Order("COUNT(*) DESC").
		Limit(limit).
		Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get top endpoint IDs", err)
	}

	return results, nil
}

func (r *APIMetricRepository) GetEndpointLatencyTrend(ctx context.Context, endpointID uint, since time.Time, groupBy string) ([]domain.LatencyTrendPoint, error) {
	dateFormat := "%Y-%m-%d"
	if groupBy == "hour" {
		dateFormat = "%Y-%m-%d %H:00"
	}

	var results []struct {
		Period string
		AvgMs  float64
		Count  int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("DATE_FORMAT(am.called_at, ?) as period, AVG(am.duration_ms) as avg_ms, COUNT(*) as count", dateFormat).
		Where("am.endpoint_id = ? AND am.called_at >= ?", endpointID, since).
		Group("period").
		Order("period ASC").
		Find(&results).Error; err != nil {
		return nil, domain.NewInternalError("failed to get endpoint latency trend", err)
	}

	var durationRows []struct {
		Period     string
		DurationMs int64
	}

	if err := r.DB.WithContext(ctx).
		Table("api_metrics am").
		Select("DATE_FORMAT(am.called_at, ?) as period, am.duration_ms", dateFormat).
		Where("am.endpoint_id = ? AND am.called_at >= ?", endpointID, since).
		Find(&durationRows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get endpoint latency trend durations", err)
	}

	durationsByPeriod := make(map[string][]int64)
	for _, row := range durationRows {
		durationsByPeriod[row.Period] = append(durationsByPeriod[row.Period], row.DurationMs)
	}

	trend := make([]domain.LatencyTrendPoint, 0, len(results))
	for _, result := range results {
		p95 := calculatePercentile(durationsByPeriod[result.Period], 0.95)
		trend = append(trend, domain.LatencyTrendPoint{
			Period: result.Period,
			AvgMs:  int64(math.Round(result.AvgMs)),
			P95Ms:  p95,
			Count:  result.Count,
		})
	}

	return trend, nil
}
