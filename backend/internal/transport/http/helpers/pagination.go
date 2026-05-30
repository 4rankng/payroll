package helpers

import (
	"strconv"

	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

const (
	// DefaultPageSize is the default number of items per page
	DefaultPageSize = 10
	// MaxPageSize is the maximum allowed page size
	MaxPageSize = 100
)

// PaginationParams holds pagination parameters parsed from request
type PaginationParams struct {
	Page     int
	PageSize int
	Limit    int
	Offset   int
}

// ParsePagination parses pagination parameters from query string.
// It validates and normalizes page and pageSize, then calculates limit and offset.
// If page or pageSize are not provided or invalid, it uses default values.
// pageSize is capped at MaxPageSize (100).
func ParsePagination(c *gin.Context, defaultPageSize int) PaginationParams {
	if defaultPageSize <= 0 {
		defaultPageSize = DefaultPageSize
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))

	// Validate page (minimum 1)
	if page < 1 {
		page = 1
	}

	// Validate pageSize (between 1 and MaxPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	// Calculate offset and limit
	offset := (page - 1) * pageSize
	limit := pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Limit:    limit,
		Offset:   offset,
	}
}

// CalculatePagination calculates pagination metadata from total count and page size.
// Returns a Pagination struct suitable for API responses.
func CalculatePagination(page, pageSize int, totalRecords int64) response.Pagination {
	totalPages := int(totalRecords) / pageSize
	if int(totalRecords)%pageSize > 0 {
		totalPages++
	}

	return response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(totalRecords),
	}
}

// ParsePaginationWithCustomMax parses pagination with a custom maximum page size.
// This is useful for endpoints that need different limits.
func ParsePaginationWithCustomMax(c *gin.Context, defaultPageSize, maxPageSize int) PaginationParams {
	if defaultPageSize <= 0 {
		defaultPageSize = DefaultPageSize
	}
	if maxPageSize <= 0 {
		maxPageSize = MaxPageSize
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", strconv.Itoa(defaultPageSize)))

	// Validate page (minimum 1)
	if page < 1 {
		page = 1
	}

	// Validate pageSize (between 1 and maxPageSize)
	if pageSize < 1 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	// Calculate offset and limit
	offset := (page - 1) * pageSize
	limit := pageSize

	return PaginationParams{
		Page:     page,
		PageSize: pageSize,
		Limit:    limit,
		Offset:   offset,
	}
}
