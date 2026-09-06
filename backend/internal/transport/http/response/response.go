package response

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

type SuccessResponse struct {
	Status     string      `json:"status"`
	Data       interface{} `json:"data"`
	Message    string      `json:"message"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

type ErrorResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"http_status"`
	Code       string `json:"code,omitempty"`
	Details    any    `json:"details,omitempty"`
}

type Pagination struct {
	Page         int `json:"page"`
	PageSize     int `json:"pageSize"`
	TotalPages   int `json:"totalPages"`
	TotalRecords int `json:"totalRecords"`
}

func Success(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Status:  "success",
		Data:    data,
		Message: message,
	})
}

func SuccessCreated(c *gin.Context, data interface{}, message string) {
	c.JSON(http.StatusCreated, SuccessResponse{
		Status:  "success",
		Data:    data,
		Message: message,
	})
}

func SuccessWithPagination(c *gin.Context, data interface{}, message string, pagination Pagination) {
	c.JSON(http.StatusOK, SuccessResponse{
		Status:     "success",
		Data:       data,
		Message:    message,
		Pagination: &pagination,
	})
}

func SuccessEmpty(c *gin.Context, message string) {
	c.JSON(http.StatusOK, SuccessResponse{
		Status:  "success",
		Data:    []interface{}{},
		Message: message,
	})
}

func SuccessEmptyWithPagination(c *gin.Context, message string, pagination Pagination) {
	c.JSON(http.StatusOK, SuccessResponse{
		Status:     "success",
		Data:       []interface{}{},
		Message:    message,
		Pagination: &pagination,
	})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func BadRequest(c *gin.Context, message string) {
	BadRequestWithCode(c, message, "")
}

func BadRequestWithCode(c *gin.Context, message, code string) {
	// Clean the message to remove any technical array indexing
	translator := GetErrorTranslator()
	cleanedMessage := translator.CleanMessage(message)

	response := ErrorResponse{
		Status:     "error",
		Message:    cleanedMessage,
		HTTPStatus: http.StatusBadRequest,
	}

	c.JSON(http.StatusBadRequest, response)
}

func Unauthorized(c *gin.Context, message string) {
	UnauthorizedWithCode(c, message, "")
}

func UnauthorizedWithCode(c *gin.Context, message, code string) {
	response := ErrorResponse{
		Status:     "error",
		Message:    message,
		HTTPStatus: http.StatusUnauthorized,
	}

	c.JSON(http.StatusUnauthorized, response)
}

func Forbidden(c *gin.Context, message string) {
	ForbiddenWithCode(c, message, "")
}

func ForbiddenWithCode(c *gin.Context, message, code string) {
	response := ErrorResponse{
		Status:     "error",
		Message:    message,
		HTTPStatus: http.StatusForbidden,
	}

	c.JSON(http.StatusForbidden, response)
}

func NotFound(c *gin.Context, message string) {
	NotFoundWithCode(c, message, "")
}

func NotFoundWithCode(c *gin.Context, message, code string) {
	response := ErrorResponse{
		Status:     "error",
		Message:    message,
		HTTPStatus: http.StatusNotFound,
	}

	c.JSON(http.StatusNotFound, response)
}

func Conflict(c *gin.Context, message string) {
	ConflictWithCode(c, message, "")
}

func ConflictWithCode(c *gin.Context, message, code string) {
	response := ErrorResponse{
		Status:     "error",
		Message:    message,
		HTTPStatus: http.StatusConflict,
	}

	c.JSON(http.StatusConflict, response)
}

func InternalServerError(c *gin.Context, message string) {
	InternalServerErrorWithCode(c, message, "")
}

func InternalServerErrorWithCode(c *gin.Context, message, code string) {
	// A cancelled request context means the client is gone (navigation) or a
	// deadline fired — not a server fault. Surface 499/504 instead of a 500 so
	// client aborts don't pollute error metrics and alerting.
	if ctxErr := c.Request.Context().Err(); ctxErr != nil {
		if errors.Is(ctxErr, context.Canceled) {
			c.Status(499) // nginx convention: client closed request
			return
		}
		c.Status(http.StatusGatewayTimeout)
		return
	}
	response := ErrorResponse{
		Status:     "error",
		Message:    message,
		HTTPStatus: http.StatusInternalServerError,
	}

	c.JSON(http.StatusInternalServerError, response)
}

func TooManyRequests(c *gin.Context, message string, retryAfter int) {
	c.Header("Retry-After", strconv.Itoa(retryAfter))
	c.JSON(http.StatusTooManyRequests, gin.H{
		"status":      "error",
		"message":     message,
		"http_status": http.StatusTooManyRequests,
		"retry_after": retryAfter,
	})
}

// NewErrorResponse creates a new error response
func NewErrorResponse(status, message string) ErrorResponse {
	return ErrorResponse{
		Status:  status,
		Message: message,
	}
}

// HandleDomainError converts domain errors to appropriate HTTP responses
func HandleDomainError(c *gin.Context, err error) {
	// Cancelled/deadline errors are not server faults. Without this mapping a
	// user navigating away mid-query (or a slow dashboard query losing the race
	// against a client timeout) is recorded as a 500 in api_metrics.
	if errors.Is(err, context.Canceled) {
		c.Status(499) // nginx convention: client closed request
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		c.JSON(http.StatusGatewayTimeout, ErrorResponse{
			Status:     "error",
			Message:    "Xử lý quá thời gian, vui lòng thử lại",
			HTTPStatus: http.StatusGatewayTimeout,
			Code:       "REQUEST_TIMEOUT",
		})
		return
	}

	translator := GetErrorTranslator()

	if domainErr, ok := err.(*domain.DomainError); ok {
		// Use translator to get user-friendly message
		translatedMessage := translator.TranslateError(err)

		response := ErrorResponse{
			Status:  "error",
			Message: translatedMessage,
			Code:    domainErr.Code,
			Details: domainErr.Context,
		}

		switch domainErr.Type {
		case "NOT_FOUND", "not_found":
			response.HTTPStatus = http.StatusNotFound
		case "VALIDATION_ERROR", "validation_error":
			response.HTTPStatus = http.StatusBadRequest
		case "UNAUTHORIZED", "unauthorized":
			response.HTTPStatus = http.StatusUnauthorized
		case "FORBIDDEN", "forbidden":
			response.HTTPStatus = http.StatusForbidden
		case "CONFLICT", "conflict":
			response.HTTPStatus = http.StatusConflict
		case "EMPTY_RESULT", "empty_result":
			// Export produced zero rows: well-formed request but unfulfillable
			// given the current resource state (no eligible timesheets, or all
			// candidates were filtered out for missing bank info / SWIFT).
			// 422 — not 400 (body is valid) and not 500 (no server fault).
			response.HTTPStatus = http.StatusUnprocessableEntity
		case "INTERNAL_ERROR", "internal_error":
			response.HTTPStatus = http.StatusInternalServerError
		default:
			InternalServerError(c, constants.MsgInternalServerErrorVN)
			return
		}
		c.JSON(response.HTTPStatus, response)
	} else {
		// For non-domain errors, use translator to clean them up
		translatedMessage := translator.TranslateError(err)
		InternalServerError(c, translatedMessage)
	}
}
