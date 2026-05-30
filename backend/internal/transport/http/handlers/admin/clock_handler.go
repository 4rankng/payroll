package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"api-server/internal/pkg/clock"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ClockHandler exposes admin-only endpoints for manipulating the server clock.
// Only available in non-production environments.
type ClockHandler struct {
	env string
}

// NewClockHandler creates a clock handler.
func NewClockHandler(clk clock.Clock, env string) *ClockHandler {
	return &ClockHandler{env: env}
}

// fake returns the current global clock as a FakeClock, or nil.
func (h *ClockHandler) fake() *clock.FakeClock {
	f, _ := clock.Global().(*clock.FakeClock)
	return f
}

// GetTime returns the server's current clock time.
// GET /api/v1/admin/clock
func (h *ClockHandler) GetTime(c *gin.Context) {
	if h.env == "production" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	clk := clock.Global()
	fc, isFake := clk.(*clock.FakeClock)
	isFrozen := isFake && fc.IsFrozen()
	now := clk.Now()
	response.Success(c, gin.H{
		"time":    now.Format(time.RFC3339),
		"unix":    now.Unix(),
		"is_fake": isFrozen,
	}, "ok")
}

// setTimeRequest is the request body for SetTime.
type setTimeRequest struct {
	Time string `json:"time" binding:"required"`
}

// SetTime sets the server clock to a specific moment.
// POST /api/v1/admin/clock/set
func (h *ClockHandler) SetTime(c *gin.Context) {
	if h.env == "production" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req setTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "missing or invalid 'time' field (RFC3339)")
		return
	}

	t, err := time.Parse(time.RFC3339, req.Time)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("invalid time format: %v", err))
		return
	}

	fc := h.fake()
	if fc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "clock manipulation not available: server using real clock",
		})
		return
	}

	previous := fc.Now()
	fc.Set(t)
	current := fc.Now()

	response.Success(c, gin.H{
		"previous": previous.Format(time.RFC3339),
		"current":  current.Format(time.RFC3339),
	}, "ok")
}

// advanceTimeRequest is the request body for AdvanceTime.
type advanceTimeRequest struct {
	Duration string `json:"duration" binding:"required"`
}

// AdvanceTime advances the server clock by a duration.
// POST /api/v1/admin/clock/advance
func (h *ClockHandler) AdvanceTime(c *gin.Context) {
	if h.env == "production" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	var req advanceTimeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "missing or invalid 'duration' field (e.g., '24h')")
		return
	}

	d, err := time.ParseDuration(req.Duration)
	if err != nil {
		response.BadRequest(c, fmt.Sprintf("invalid duration: %v", err))
		return
	}

	fc := h.fake()
	if fc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "clock manipulation not available: server using real clock",
		})
		return
	}

	previous := fc.Now()
	fc.Advance(d)
	current := fc.Now()

	response.Success(c, gin.H{
		"previous": previous.Format(time.RFC3339),
		"current":  current.Format(time.RFC3339),
		"advanced": d.String(),
	}, "ok")
}

// ResetTime resets the server clock to real system time.
// POST /api/v1/admin/clock/reset
// Unfreezes the FakeClock so it auto-advances from real time again.
func (h *ClockHandler) ResetTime(c *gin.Context) {
	if h.env == "production" {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	fc := h.fake()
	if fc == nil {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "clock manipulation not available: server using real clock",
		})
		return
	}

	previous := fc.Now()
	fc.Reset()
	current := clock.Now()

	response.Success(c, gin.H{
		"previous":        previous.Format(time.RFC3339),
		"current":         current.Format(time.RFC3339),
		"reset_to_system": true,
	}, "ok")
}

// MarshalJSON implements custom JSON marshaling for the handler response.
var _ = json.Marshal
