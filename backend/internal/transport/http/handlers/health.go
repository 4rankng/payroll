package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthChecker interface {
	CheckHealth(ctx context.Context) error
}

type HealthHandler struct {
	checkers map[string]HealthChecker
	version  string
}

func NewHealthHandler(checkers map[string]HealthChecker, version string) *HealthHandler {
	return &HealthHandler{
		checkers: checkers,
		version:  version,
	}
}

type HealthResponse struct {
	Status  string                       `json:"status"`
	Checks  map[string]HealthCheckResult `json:"checks"`
	Version string                       `json:"version,omitempty"`
}

type HealthCheckResult struct {
	Status    string        `json:"status"`
	Duration  time.Duration `json:"duration,omitempty"`
	Error     string        `json:"error,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// @Summary Health Check
// @Description Get the health status of the API and its dependencies
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Success 503 {object} HealthResponse
// @Router /api/healthz [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	response := HealthResponse{
		Status:  "healthy",
		Checks:  make(map[string]HealthCheckResult),
		Version: h.version,
	}

	allHealthy := true

	for name, checker := range h.checkers {
		start := time.Now()
		result := HealthCheckResult{
			Timestamp: start,
		}

		if err := checker.CheckHealth(ctx); err != nil {
			result.Status = "unhealthy"
			result.Error = err.Error()
			allHealthy = false
		} else {
			result.Status = "healthy"
		}

		result.Duration = time.Since(start)
		response.Checks[name] = result
	}

	if !allHealthy {
		response.Status = "unhealthy"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// @Summary Readiness Check
// @Description Check if the API is ready to serve requests
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /api/ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	// For now, readiness is the same as health
	// In a more complex system, this might check different things
	h.HealthCheck(c)
}
