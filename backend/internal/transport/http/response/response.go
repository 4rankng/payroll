package response

import (
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
	translator := GetErrorTranslator()

	if domainErr, ok := err.(*domain.DomainError); ok {
		// Use translator to get user-friendly message
		translatedMessage := translator.TranslateError(err)

		switch domainErr.Type {
		case "NOT_FOUND", "not_found":
			NotFoundWithCode(c, translatedMessage, domainErr.Code)
		case "VALIDATION_ERROR", "validation_error":
			BadRequestWithCode(c, translatedMessage, domainErr.Code)
		case "UNAUTHORIZED", "unauthorized":
			UnauthorizedWithCode(c, translatedMessage, domainErr.Code)
		case "FORBIDDEN", "forbidden":
			ForbiddenWithCode(c, translatedMessage, domainErr.Code)
		case "CONFLICT", "conflict":
			ConflictWithCode(c, translatedMessage, domainErr.Code)
		case "INTERNAL_ERROR", "internal_error":
			InternalServerErrorWithCode(c, translatedMessage, domainErr.Code)
		default:
			InternalServerError(c, constants.MsgInternalServerErrorVN)
		}
	} else {
		// For non-domain errors, use translator to clean them up
		translatedMessage := translator.TranslateError(err)
		InternalServerError(c, translatedMessage)
	}
}
