package helpers

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParsePagination(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name             string
		queryParams      map[string]string
		defaultPageSize  int
		expectedPage     int
		expectedPageSize int
		expectedLimit    int
		expectedOffset   int
	}{
		{
			name:             "default values",
			queryParams:      map[string]string{},
			defaultPageSize:  10,
			expectedPage:     1,
			expectedPageSize: 10,
			expectedLimit:    10,
			expectedOffset:   0,
		},
		{
			name:             "page 2",
			queryParams:      map[string]string{"page": "2", "pageSize": "10"},
			defaultPageSize:  10,
			expectedPage:     2,
			expectedPageSize: 10,
			expectedLimit:    10,
			expectedOffset:   10,
		},
		{
			name:             "page 3 with custom pageSize",
			queryParams:      map[string]string{"page": "3", "pageSize": "20"},
			defaultPageSize:  10,
			expectedPage:     3,
			expectedPageSize: 20,
			expectedLimit:    20,
			expectedOffset:   40,
		},
		{
			name:             "pageSize exceeds max",
			queryParams:      map[string]string{"page": "1", "pageSize": "200"},
			defaultPageSize:  10,
			expectedPage:     1,
			expectedPageSize: 100, // capped at MaxPageSize
			expectedLimit:    100,
			expectedOffset:   0,
		},
		{
			name:             "invalid page (0)",
			queryParams:      map[string]string{"page": "0", "pageSize": "10"},
			defaultPageSize:  10,
			expectedPage:     1, // defaults to 1
			expectedPageSize: 10,
			expectedLimit:    10,
			expectedOffset:   0,
		},
		{
			name:             "invalid page (negative)",
			queryParams:      map[string]string{"page": "-5", "pageSize": "10"},
			defaultPageSize:  10,
			expectedPage:     1, // defaults to 1
			expectedPageSize: 10,
			expectedLimit:    10,
			expectedOffset:   0,
		},
		{
			name:             "invalid pageSize (0)",
			queryParams:      map[string]string{"page": "1", "pageSize": "0"},
			defaultPageSize:  10,
			expectedPage:     1,
			expectedPageSize: 10, // uses default
			expectedLimit:    10,
			expectedOffset:   0,
		},
		{
			name:             "invalid pageSize (negative)",
			queryParams:      map[string]string{"page": "1", "pageSize": "-10"},
			defaultPageSize:  10,
			expectedPage:     1,
			expectedPageSize: 10, // uses default
			expectedLimit:    10,
			expectedOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test context
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/?"+buildQueryString(tt.queryParams), nil)

			// Parse pagination
			result := ParsePagination(c, tt.defaultPageSize)

			// Verify results
			if result.Page != tt.expectedPage {
				t.Errorf("Page: expected %d, got %d", tt.expectedPage, result.Page)
			}
			if result.PageSize != tt.expectedPageSize {
				t.Errorf("PageSize: expected %d, got %d", tt.expectedPageSize, result.PageSize)
			}
			if result.Limit != tt.expectedLimit {
				t.Errorf("Limit: expected %d, got %d", tt.expectedLimit, result.Limit)
			}
			if result.Offset != tt.expectedOffset {
				t.Errorf("Offset: expected %d, got %d", tt.expectedOffset, result.Offset)
			}
		})
	}
}

func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name               string
		page               int
		pageSize           int
		totalRecords       int64
		expectedTotalPages int
	}{
		{
			name:               "exact division",
			page:               1,
			pageSize:           10,
			totalRecords:       100,
			expectedTotalPages: 10,
		},
		{
			name:               "with remainder",
			page:               1,
			pageSize:           10,
			totalRecords:       105,
			expectedTotalPages: 11,
		},
		{
			name:               "less than one page",
			page:               1,
			pageSize:           10,
			totalRecords:       5,
			expectedTotalPages: 1,
		},
		{
			name:               "zero records",
			page:               1,
			pageSize:           10,
			totalRecords:       0,
			expectedTotalPages: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePagination(tt.page, tt.pageSize, tt.totalRecords)

			if result.Page != tt.page {
				t.Errorf("Page: expected %d, got %d", tt.page, result.Page)
			}
			if result.PageSize != tt.pageSize {
				t.Errorf("PageSize: expected %d, got %d", tt.pageSize, result.PageSize)
			}
			if result.TotalPages != tt.expectedTotalPages {
				t.Errorf("TotalPages: expected %d, got %d", tt.expectedTotalPages, result.TotalPages)
			}
			if result.TotalRecords != int(tt.totalRecords) {
				t.Errorf("TotalRecords: expected %d, got %d", tt.totalRecords, result.TotalRecords)
			}
		})
	}
}

// Helper function to build query string from map
func buildQueryString(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	query := ""
	first := true
	for key, value := range params {
		if !first {
			query += "&"
		}
		query += key + "=" + value
		first = false
	}
	return query
}
