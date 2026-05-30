package validation

import (
	"strconv"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"github.com/gin-gonic/gin"
)

// RequestValidator provides standardized request validation utilities
type RequestValidator struct{}

// NewRequestValidator creates a new request validator
func NewRequestValidator() *RequestValidator {
	return &RequestValidator{}
}

// ValidateIDParam validates and parses ID from URL parameter
func (v *RequestValidator) ValidateIDParam(c *gin.Context, paramName string) (uint, error) {
	idStr := c.Param(paramName)
	if idStr == "" {
		return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		switch paramName {
		case "id":
			return 0, domain.NewValidationError(constants.MsgInvalidEmployeeIDVN)
		case "project_id":
			return 0, domain.NewValidationError(constants.MsgInvalidProjectIDVN)
		case "payrate_id":
			return 0, domain.NewValidationError(constants.MsgInvalidPayrateIDVN)
		default:
			return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
		}
	}

	if id == 0 {
		return 0, domain.NewValidationError(constants.MsgInvalidRequestBodyVN)
	}

	return uint(id), nil
}

// ValidateUserContext extracts and validates user ID from context
func (v *RequestValidator) ValidateUserContext(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, domain.NewForbiddenError(constants.MsgUserIDNotFoundInContextVN)
	}

	uid, ok := userID.(uint)
	if !ok || uid == 0 {
		return 0, domain.NewForbiddenError(constants.MsgInvalidUserIDVN)
	}

	return uid, nil
}

// ValidateDateParam validates and parses date parameter
func (v *RequestValidator) ValidateDateParam(c *gin.Context, paramName string) (*time.Time, error) {
	dateStr := c.Query(paramName)
	if dateStr == "" {
		return nil, nil // Optional parameter
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		switch paramName {
		case "from_date":
			return nil, domain.NewValidationError(constants.MsgInvalidFromDateFormatVN)
		case "to_date":
			return nil, domain.NewValidationError(constants.MsgInvalidToDateFormatVN)
		case "date":
			return nil, domain.NewValidationError(constants.MsgInvalidDateFormatVN)
		case "start_date":
			return nil, domain.NewValidationError(constants.MsgInvalidStartDateFormatVN)
		case "end_date":
			return nil, domain.NewValidationError(constants.MsgInvalidEndDateFormatVN)
		case "at_date":
			return nil, domain.NewValidationError(constants.MsgInvalidAtDateFormatVN)
		default:
			return nil, domain.NewValidationError(constants.MsgInvalidDateFormatVN)
		}
	}

	return &date, nil
}

// ValidateRequiredDateParam validates required date parameter
func (v *RequestValidator) ValidateRequiredDateParam(c *gin.Context, paramName string) (time.Time, error) {
	dateStr := c.Query(paramName)
	if dateStr == "" {
		return time.Time{}, domain.NewValidationError(constants.MsgDateParameterRequiredVN)
	}

	return time.Parse("2006-01-02", dateStr)
}

// ValidatePaginationParams validates pagination query parameters
func (v *RequestValidator) ValidatePaginationParams(c *gin.Context) (limit, offset int, err error) {
	// Default values
	limit = 50
	offset = 0

	// Parse limit
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, parseErr := strconv.Atoi(limitStr); parseErr == nil {
			if parsedLimit > 0 && parsedLimit <= 1000 { // Max limit of 1000
				limit = parsedLimit
			} else if parsedLimit > 1000 {
				return 0, 0, domain.NewValidationError(constants.MsgLimitCannotExceed1000VN)
			}
		} else {
			return 0, 0, domain.NewValidationError(constants.MsgInvalidQueryParametersVN)
		}
	}

	// Parse offset
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if parsedOffset, parseErr := strconv.Atoi(offsetStr); parseErr == nil {
			if parsedOffset >= 0 {
				offset = parsedOffset
			}
		} else {
			return 0, 0, domain.NewValidationError(constants.MsgInvalidQueryParametersVN)
		}
	}

	return limit, offset, nil
}

// ValidateSearchQuery validates search query parameters
func (v *RequestValidator) ValidateSearchQuery(c *gin.Context) (string, error) {
	query := c.Query("search")
	if query == "" {
		query = c.Query("q") // Alternative parameter name
	}

	if query == "" {
		return "", domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}

	if len(query) < 3 {
		return "", domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	return query, nil
}

// ValidateOptionalSearchQuery validates optional search query parameters
func (v *RequestValidator) ValidateOptionalSearchQuery(c *gin.Context) (string, error) {
	query := c.Query("search")
	if query == "" {
		query = c.Query("q") // Alternative parameter name
	}

	if query != "" && len(query) < 3 {
		return "", domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	return query, nil
}

// ValidateSortParams validates sorting parameters
func (v *RequestValidator) ValidateSortParams(c *gin.Context, allowedFields []string) (sortBy, sortOrder string, err error) {
	sortBy = c.Query("sort_by")
	sortOrder = c.Query("sort_order")

	// Validate sort_by
	if sortBy != "" {
		isValidField := false
		for _, field := range allowedFields {
			if sortBy == field {
				isValidField = true
				break
			}
		}
		if !isValidField {
			return "", "", domain.NewValidationError(constants.MsgInvalidSortFieldVN)
		}
	}

	// Validate sort_order
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return "", "", domain.NewValidationError(constants.MsgSortOrderMustBeAscOrDescVN)
	}

	return sortBy, sortOrder, nil
}
