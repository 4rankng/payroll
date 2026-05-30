package handlers

import (
	"encoding/json"

	"api-server/internal/domain"
	"api-server/internal/infra/events"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/pkg/clock"
)

type MetricHandler struct {
	metricService *infrastructure.MetricService
	cacheService  domain.CacheServiceUseCase
	eventBus      interface{} // Can be WorkerPoolEventBus or RedisEventBus
	clock         clock.Clock
}

func NewMetricHandler(metricService *infrastructure.MetricService, cacheService domain.CacheServiceUseCase, eventBus interface{}, clk clock.Clock) *MetricHandler {
	if clk == nil {
		clk = clock.New()
	}
	return &MetricHandler{
		metricService: metricService,
		cacheService:  cacheService,
		eventBus:      eventBus,
		clock:         clk,
	}
}

type APILatency struct {
	P95    int64 `json:"p95"`
	Min    int64 `json:"min"`
	Max    int64 `json:"max"`
	Avg    int64 `json:"avg"`
	Median int64 `json:"median"`
}

type APIError struct {
	Total int64            `json:"total"`
	Codes map[string]int64 `json:"-"` // Internal representation
}

func (e APIError) MarshalJSON() ([]byte, error) {
	// Include total and all status codes in the JSON response
	result := make(map[string]int64, len(e.Codes)+1)
	result["total"] = e.Total
	for code, count := range e.Codes {
		result[code] = count
	}
	return json.Marshal(result)
}

type APISummaryItem struct {
	Endpoint string     `json:"endpoint"`
	Count    int64      `json:"count"`
	Latency  APILatency `json:"latency"`
	Success  int64      `json:"success"`
	Error    *APIError  `json:"error,omitempty"`
}

type APISummaryRequest struct {
	Days   int    `form:"days"`
	SortBy string `form:"sortBy"`
}

type APISummaryResponse struct {
	Status  string           `json:"status"`
	Message string           `json:"message"`
	Data    []APISummaryItem `json:"data"`
}

// GetAPISummary retrieves API call statistics for all records
// @Summary Get API call summary
// @Description Get aggregated API call counts for all endpoints from all available data
// @Tags metrics
// @Accept json
// @Produce json
// @Param sortBy query string false "Sort field (count, latency_p95, latency_median, error_total)"
// @Success 200 {object} APISummaryResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/metrics/api [get]
func (h *MetricHandler) GetAPISummary(c *gin.Context) {
	var req APISummaryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	// Validate and set default sortBy
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "count"
	}

	validSortFields := map[string]bool{
		"count":          true,
		"latency_p95":    true,
		"latency_median": true,
		"error_total":    true,
	}

	if !validSortFields[sortBy] {
		response.BadRequest(c, "sortBy must be one of: count, latency_p95, latency_median, error_total")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 1
	}
	since := h.clock.Now().AddDate(0, 0, -days)

	summary, err := h.metricService.GetAPISummary(c.Request.Context(), since, sortBy)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	transformedSummary := make([]APISummaryItem, len(summary))
	for i, item := range summary {
		var errorInfo *APIError
		if item.Error != nil {
			errorCodes := make(map[string]int64, len(item.Error.StatusCounts))
			for code, count := range item.Error.StatusCounts {
				errorCodes[code] = count
			}

			errorInfo = &APIError{
				Total: item.Error.Total,
				Codes: errorCodes,
			}
		}

		transformedSummary[i] = APISummaryItem{
			Endpoint: item.Endpoint,
			Count:    item.Count,
			Latency: APILatency{
				P95:    item.Latency.P95,
				Min:    item.Latency.Min,
				Max:    item.Latency.Max,
				Avg:    item.Latency.Avg,
				Median: item.Latency.Median,
			},
			Success: item.Success,
			Error:   errorInfo,
		}
	}

	response.Success(c, transformedSummary, "successfully fetch api summary")
}

type EventBusMetricsResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Data    *eventBusMetricsPayload `json:"data,omitempty"`
	Error   string                  `json:"error,omitempty"`
}

type eventBusMetricsPayload struct {
	EventsPublished   int64 `json:"EventsPublished"`
	EventsProcessed   int64 `json:"EventsProcessed"`
	EventsDropped     int64 `json:"EventsDropped"`
	HandlerErrors     int64 `json:"HandlerErrors"`
	HandlerPanics     int64 `json:"HandlerPanics"`
	AvgProcessingTime int64 `json:"AvgProcessingTime"`
	QueueDepth        int   `json:"QueueDepth"`
}

func makeEventBusMetricsPayload(snapshot events.EventBusMetricsSnapshot) *eventBusMetricsPayload {
	return &eventBusMetricsPayload{
		EventsPublished:   snapshot.EventsPublished,
		EventsProcessed:   snapshot.EventsProcessed,
		EventsDropped:     snapshot.EventsDropped,
		HandlerErrors:     snapshot.HandlerErrors,
		HandlerPanics:     snapshot.HandlerPanics,
		AvgProcessingTime: snapshot.AvgProcessingTime.Milliseconds(),
		QueueDepth:        snapshot.QueueDepth,
	}
}

// GetEventBusMetrics retrieves current event bus metrics
// @Summary Get event bus metrics
// @Description Get real-time metrics from the event bus including published, processed, dropped events, and queue depth
// @Tags metrics
// @Accept json
// @Produce json
// @Success 200 {object} EventBusMetricsResponse
// @Failure 500 {object} response.ErrorResponse
// @Failure 503 {object} EventBusMetricsResponse "Event bus not available or metrics not supported"
// @Security ApiKeyAuth
// @Router /api/v1/metrics/event-bus [get]
func (h *MetricHandler) GetEventBusMetrics(c *gin.Context) {
	// Try to get metrics from WorkerPoolEventBus
	if workerBus, ok := h.eventBus.(*events.WorkerPoolEventBus); ok {
		metrics := workerBus.GetMetrics()
		response.Success(c, makeEventBusMetricsPayload(metrics), "Event bus metrics retrieved successfully")
		return
	}

	// Try to get metrics from RedisEventBus
	if redisBus, ok := h.eventBus.(*events.RedisEventBus); ok {
		metrics := redisBus.GetMetrics()
		response.Success(c, makeEventBusMetricsPayload(metrics), "Event bus metrics retrieved successfully (Redis)")
		return
	}

	// Event bus doesn't support metrics
	response.InternalServerError(c, "Event bus metrics not available")
}

type CacheMetricsResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// GetCacheMetrics retrieves Redis cache statistics
// @Summary Get cache metrics
// @Description Get Redis cache statistics including hit rate, memory usage, and operations per second
// @Tags metrics
// @Accept json
// @Produce json
// @Success 200 {object} CacheMetricsResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /api/v1/metrics/cache [get]
func (h *MetricHandler) GetCacheMetrics(c *gin.Context) {
	stats, err := h.cacheService.GetRedisStats(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Parse and structure the most important stats
	data := make(map[string]interface{})

	// Core cache statistics
	if keyspaceHits, ok := stats["keyspace_hits"]; ok {
		data["keyspace_hits"] = keyspaceHits
	}
	if keyspaceMisses, ok := stats["keyspace_misses"]; ok {
		data["keyspace_misses"] = keyspaceMisses
	}

	// Memory statistics
	if usedMemory, ok := stats["used_memory"]; ok {
		data["used_memory"] = usedMemory
	}
	if usedMemoryHuman, ok := stats["used_memory_human"]; ok {
		data["used_memory_human"] = usedMemoryHuman
	}
	if usedMemoryPeak, ok := stats["used_memory_peak"]; ok {
		data["used_memory_peak"] = usedMemoryPeak
	}
	if usedMemoryPeakHuman, ok := stats["used_memory_peak_human"]; ok {
		data["used_memory_peak_human"] = usedMemoryPeakHuman
	}

	// Performance statistics
	if totalCommandsProcessed, ok := stats["total_commands_processed"]; ok {
		data["total_commands_processed"] = totalCommandsProcessed
	}
	if instantaneousOps, ok := stats["instantaneous_ops_per_sec"]; ok {
		data["instantaneous_ops_per_sec"] = instantaneousOps
	}

	// Include all raw stats for advanced users
	data["raw_stats"] = stats

	response.Success(c, data, "Cache metrics retrieved successfully")
}

type ErrorBreakdownByUserRequest struct {
	Days          int `form:"days"`
	MinStatusCode int `form:"minStatusCode"`
	Limit         int `form:"limit"`
}

func (h *MetricHandler) GetErrorBreakdownByUser(c *gin.Context) {
	var req ErrorBreakdownByUserRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 14
	}
	minStatusCode := req.MinStatusCode
	if minStatusCode <= 0 {
		minStatusCode = 401
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetErrorBreakdownByUser(c.Request.Context(), since, minStatusCode, limit)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Error breakdown by user retrieved successfully")
}

type LatencyTrendRequest struct {
	Days    int    `form:"days"`
	GroupBy string `form:"groupBy"`
}

func (h *MetricHandler) GetLatencyTrend(c *gin.Context) {
	var req LatencyTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	groupBy := req.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}
	if groupBy != "day" && groupBy != "hour" {
		response.BadRequest(c, "groupBy must be 'day' or 'hour'")
		return
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetLatencyTrend(c.Request.Context(), since, groupBy)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Latency trend retrieved successfully")
}

type SlowestEndpointsRequest struct {
	Days  int `form:"days"`
	Limit int `form:"limit"`
}

func (h *MetricHandler) GetSlowestEndpoints(c *gin.Context) {
	var req SlowestEndpointsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetSlowestEndpoints(c.Request.Context(), since, limit)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Slowest endpoints retrieved successfully")
}

type RecentErrorsRequest struct {
	Days  int `form:"days"`
	Limit int `form:"limit"`
}

func (h *MetricHandler) GetRecentErrors(c *gin.Context) {
	var req RecentErrorsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 1
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetRecentErrors(c.Request.Context(), since, limit)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Recent errors retrieved successfully")
}

type ErrorCountRequest struct {
	Days int `form:"days"`
}

func (h *MetricHandler) GetErrorCount(c *gin.Context) {
	var req ErrorCountRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 1
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	count, err := h.metricService.GetErrorCount(c.Request.Context(), since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, gin.H{"total": count}, "Error count retrieved successfully")
}

type EndpointLatencyTrendRequest struct {
	EndpointID uint   `form:"endpointId"`
	Days       int    `form:"days"`
	GroupBy    string `form:"groupBy"`
}

func (h *MetricHandler) GetEndpointLatencyTrend(c *gin.Context) {
	var req EndpointLatencyTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	if req.EndpointID == 0 {
		response.BadRequest(c, "endpointId is required")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	groupBy := req.GroupBy
	if groupBy == "" {
		groupBy = "day"
	}
	if groupBy != "day" && groupBy != "hour" {
		response.BadRequest(c, "groupBy must be 'day' or 'hour'")
		return
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetEndpointLatencyTrend(c.Request.Context(), req.EndpointID, since, groupBy)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Endpoint latency trend retrieved successfully")
}

type TopEndpointsRequest struct {
	Days  int `form:"days"`
	Limit int `form:"limit"`
}

func (h *MetricHandler) GetTopEndpoints(c *gin.Context) {
	var req TopEndpointsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetTopEndpointIDs(c.Request.Context(), since, limit)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Top endpoints retrieved successfully")
}

type BrowserPlatformStatsRequest struct {
	Days int `form:"days"`
}

func (h *MetricHandler) GetBrowserPlatformStats(c *gin.Context) {
	var req BrowserPlatformStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 30
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetBrowserPlatformStats(c.Request.Context(), since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Browser platform stats retrieved successfully")
}

type BrowserPlatformUsersRequest struct {
	Browser  string `form:"browser"`
	Platform string `form:"platform"`
	Days     int    `form:"days"`
}

func (h *MetricHandler) GetBrowserPlatformUsers(c *gin.Context) {
	var req BrowserPlatformUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	if req.Browser == "" || req.Platform == "" {
		response.BadRequest(c, "browser and platform are required")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 30
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetBrowserPlatformUsers(c.Request.Context(), req.Browser, req.Platform, since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Browser platform users retrieved successfully")
}

type OSStatsRequest struct {
	Days int `form:"days"`
}

func (h *MetricHandler) GetOSStats(c *gin.Context) {
	var req OSStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}
	days := req.Days
	if days <= 0 {
		days = 30
	}
	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetOSGroupedStats(c.Request.Context(), since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "OS grouped stats retrieved successfully")
}

type BrowserStatsRequest struct {
	Days int `form:"days"`
}

func (h *MetricHandler) GetBrowserStats(c *gin.Context) {
	var req BrowserStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}
	days := req.Days
	if days <= 0 {
		days = 30
	}
	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetBrowserGroupedStats(c.Request.Context(), since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Browser grouped stats retrieved successfully")
}

type OSUsersRequest struct {
	OSFamily string `form:"osFamily"`
	Days     int    `form:"days"`
}

func (h *MetricHandler) GetOSUsers(c *gin.Context) {
	var req OSUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}
	if req.OSFamily == "" {
		response.BadRequest(c, "osFamily is required")
		return
	}
	days := req.Days
	if days <= 0 {
		days = 30
	}
	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetOSGroupedUsers(c.Request.Context(), req.OSFamily, since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "OS users retrieved successfully")
}

type BrowserUsersRequest struct {
	BrowserFamily string `form:"browserFamily"`
	Days          int    `form:"days"`
}

func (h *MetricHandler) GetBrowserUsers(c *gin.Context) {
	var req BrowserUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}
	if req.BrowserFamily == "" {
		response.BadRequest(c, "browserFamily is required")
		return
	}
	days := req.Days
	if days <= 0 {
		days = 30
	}
	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetBrowserGroupedUsers(c.Request.Context(), req.BrowserFamily, since)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}
	response.Success(c, data, "Browser users retrieved successfully")
}

type FailedLoginsRequest struct {
	Days  int `form:"days"`
	Limit int `form:"limit"`
}

func (h *MetricHandler) GetFailedLogins(c *gin.Context) {
	var req FailedLoginsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 100
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetFailedLogins(c.Request.Context(), since, limit)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Failed logins retrieved successfully")
}

type FailedLoginsByIdentifierRequest struct {
	Days int `form:"days"`
}

func (h *MetricHandler) GetFailedLoginsByIdentifier(c *gin.Context) {
	identifier := c.Param("identifier")
	if identifier == "" {
		response.BadRequest(c, "Identifier is required")
		return
	}

	var req FailedLoginsByIdentifierRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}

	days := req.Days
	if days <= 0 {
		days = 7
	}

	since := h.clock.Now().AddDate(0, 0, -days)
	data, err := h.metricService.GetFailedLoginsByIdentifier(c.Request.Context(), identifier, since, 100)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, data, "Failed logins for identifier retrieved successfully")
}
