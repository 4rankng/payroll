package middleware

import (
	"net/http"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ErrorMiddleware provides centralized error handling for all API routes
func ErrorMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(error); ok {
			handleError(c, err)
		} else {
			response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		}
		c.Abort()
	})
}

// ErrorHandler is a middleware that converts errors to standardized Vietnamese responses
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Handle errors that occurred during request processing
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			handleError(c, err)
		}
	}
}

// handleError converts domain errors to appropriate HTTP responses with Vietnamese messages
func handleError(c *gin.Context, err error) {
	switch {
	case domain.IsValidationError(err):
		response.BadRequest(c, err.Error())
	case domain.IsNotFoundError(err):
		response.NotFound(c, err.Error())
	case domain.IsConflictError(err):
		response.Conflict(c, err.Error())
	case domain.IsUnauthorizedError(err):
		response.Unauthorized(c, err.Error())
	case domain.IsForbiddenError(err):
		response.Forbidden(c, err.Error())
	default:
		// Log the original error for debugging
		c.Header("X-Error-ID", generateErrorID())
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
	}
}

// ValidationMiddleware provides request validation before reaching handlers
func ValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Validate content type for POST/PUT requests
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
			contentType := c.GetHeader("Content-Type")
			if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
				response.BadRequest(c, constants.MsgInvalidRequestFormatVN)
				c.Abort()
				return
			}
		}

		// Validate authorization header exists
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, constants.MsgAuthorizationHeaderRequiredVN)
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequestSizeMiddleware limits request body size
func RequestSizeMiddleware(maxSize int64) gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		if c.Request.ContentLength > maxSize {
			response.BadRequest(c, constants.MsgFileSizeExceedsLimitVN)
			c.Abort()
			return
		}
		c.Next()
	})
}

// generateErrorID creates a unique error ID for tracking
func generateErrorID() string {
	// Simple implementation - in production, use UUID or more sophisticated ID generation
	return "ERR-" + string(rune(int32(gin.Mode()[0]))) + "-" + "123456"
}
