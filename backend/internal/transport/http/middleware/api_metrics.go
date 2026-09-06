package middleware

import (
	"context"
	"regexp"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"

	"github.com/gin-gonic/gin"
)

// normalizePathForMetrics converts specific ID-based paths to aggregated patterns
// Examples:
// - /api/v1/employees/140 → /api/v1/employees/*
// - /api/v1/assets/10/download → /api/v1/assets/*/download
// - /api/v1/employees/140/summary → /api/v1/employees/*/summary
// - /api/v1/employees/139/users → /api/v1/employees/*/users
// - /api/v1/projects/10/employees → /api/v1/projects/*/employees
// - /api/v1/employees/139/users/45 → /api/v1/employees/*/users/*
func normalizePathForMetrics(path string) string {
	// Common API prefix patterns to match
	apiV1Pattern := regexp.MustCompile(`^/api/v1/`)

	// If not an API v1 path, return as-is
	if !apiV1Pattern.MatchString(path) {
		return path
	}

	// List of resource segments that should be preserved (not aggregated)
	preservedSegments := map[string]bool{
		"summary":             true,
		"download":            true,
		"payroll":             true,
		"timesheet":           true,
		"timesheets":          true,
		"activities":          true,
		"schedule":            true,
		"entries":             true,
		"projects":            true,
		"users":               true,
		"employees":           true,
		"payrate":             true,
		"current":             true,
		"change-password":     true,
		"reset-password":      true,
		"add-to-payroll":      true,
		"request-edit":        true,
		"request-edit-cancel": true,
		"bulk-approve":        true,
		"settle":              true,
		"reverse":             true,
		"repay":               true,
		"repay-schedule":      true,
		"disburse":            true,
		"read":                true,
		"me":                  true,
		"login":               true,
		"logout":              true,
		"ready":               true,
		"healthz":             true,
		"metrics":             true,
	}

	// Split path into segments
	segments := regexp.MustCompile(`/`).Split(path, -1)

	// Process segments to identify numeric IDs and aggregate them
	result := make([]string, 0, len(segments))

	for i, segment := range segments {
		if segment == "" {
			continue // Skip empty segments from leading/trailing slashes
		}

		// Check if segment is a numeric ID (pure digits)
		if isNumeric(segment) {
			// Look ahead to see if the next segment should be preserved
			nextSegment := ""
			if i+1 < len(segments) {
				nextSegment = segments[i+1]
			}

			// If next segment is a preserved action, keep structure but replace ID with *
			if nextSegment != "" && preservedSegments[nextSegment] {
				result = append(result, "*")
			} else if i+1 < len(segments) && isNumeric(segments[i+1]) {
				// Double ID pattern (e.g., /users/:userId/projects/:projectId)
				result = append(result, "*")
			} else if i == 3 && len(result) == 2 {
				// This is likely a resource ID (position 3: /api/v1/resource/:id)
				result = append(result, "*")
			} else if i > 3 {
				// Any numeric segment beyond the resource level should be aggregated
				result = append(result, "*")
			} else {
				// Preserve the segment as-is
				result = append(result, segment)
			}
		} else {
			// Preserve non-numeric segments
			result = append(result, segment)
		}
	}

	// Reconstruct the path
	normalizedPath := "/" + strings.Join(result, "/")

	return normalizedPath
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	return regexp.MustCompile(`^\d+$`).MatchString(s)
}

// ClientSourceHeader marks the origin of API traffic. The integration suite
// tags its requests with it so they can be excluded from api_metrics.
const ClientSourceHeader = "X-Client-Source"

// ClientSourceIntegrationTest marks a request as coming from the integration
// test suite (backend/tests/integration).
const ClientSourceIntegrationTest = "integration-test"

// APIMetrics middleware collects API call statistics
func APIMetrics(repo domain.APIMetricRepository) gin.HandlerFunc {
	logger := observability.GetLogger()

	return func(c *gin.Context) {
		// Attach per-request DB metrics container to the request context
		ctxWithDBMetrics, _ := observability.WithDBMetrics(c.Request.Context())
		c.Request = c.Request.WithContext(ctxWithDBMetrics)

		// Record the time when the request started
		startTime := time.Now()

		// Process request (this executes the actual handler)
		c.Next()

		// Skip metrics collection for certain paths
		path := c.Request.URL.Path
		if shouldSkipMetrics(path) {
			return
		}

		// Skip integration-test traffic so the dashboards reflect real usage
		// instead of expected negative-case responses from the test suite.
		if c.Request.Header.Get(ClientSourceHeader) == ClientSourceIntegrationTest {
			return
		}

		// Normalize path for systematic endpoint aggregation
		normalizedPath := normalizePathForMetrics(path)

		// Get user ID from context if available
		var userID *uint
		if user, exists := c.Get("user_id"); exists {
			if u, ok := user.(uint); ok {
				userID = &u
			}
		}

		// Calculate request duration in milliseconds (AFTER handler completes)
		duration := time.Since(startTime).Milliseconds()

		// Get the response status code (AFTER handler completes)
		statusCode := c.Writer.Status()
		method := c.Request.Method

		// Record metric asynchronously to avoid blocking the response
		go func() {
			// Create a background context with timeout to prevent hanging goroutines
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Find or create endpoint record using normalized path
			endpointID, err := repo.FindOrCreateEndpoint(ctx, method, normalizedPath)
			if err != nil {
				logger.Error("Failed to find/create endpoint",
					"method", method,
					"path", normalizedPath,
					"original_path", path,
					"error", err)
				return
			}

			// Create metric record
			metric := &domain.APIMetric{
				EndpointID: endpointID,
				StatusCode: statusCode,
				UserID:     userID,
				DurationMs: duration,
				CalledAt:   startTime,
			}

			if err := repo.Create(ctx, metric); err != nil {
				logger.Error("Failed to record API metric",
					"method", method,
					"path", normalizedPath,
					"original_path", path,
					"status", statusCode,
					"error", err)
				// Don't return here, as we're in a goroutine and this error shouldn't affect the main request
			}
		}()
	}
}

// shouldSkipMetrics determines if metrics should be skipped for a given path
func shouldSkipMetrics(path string) bool {
	if strings.HasPrefix(path, "/api/v1/metrics/") {
		return true
	}

	skipPaths := []string{
		"/api/healthz",
		"/api/ready",
		"/metrics",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}

	return false
}
