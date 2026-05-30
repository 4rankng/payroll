package validation

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewRequestValidator(t *testing.T) {
	v := NewRequestValidator()
	if v == nil {
		t.Error("NewRequestValidator() returned nil")
	}
}

func TestValidatePaginationParams(t *testing.T) {
	v := NewRequestValidator()

	tests := []struct {
		name        string
		queryParams map[string]string
		wantLimit   int
		wantOffset  int
		wantErr     bool
	}{
		{
			name:        "default values",
			queryParams: map[string]string{},
			wantLimit:   50,
			wantOffset:  0,
			wantErr:     false,
		},
		{
			name:        "custom limit and offset",
			queryParams: map[string]string{"limit": "10", "offset": "20"},
			wantLimit:   10,
			wantOffset:  20,
			wantErr:     false,
		},
		{
			name:        "max limit",
			queryParams: map[string]string{"limit": "1000"},
			wantLimit:   1000,
			wantOffset:  0,
			wantErr:     false,
		},
		{
			name:        "limit exceeds max",
			queryParams: map[string]string{"limit": "1001"},
			wantLimit:   0,
			wantOffset:  0,
			wantErr:     true,
		},
		{
			name:        "invalid limit format",
			queryParams: map[string]string{"limit": "invalid"},
			wantLimit:   0,
			wantOffset:  0,
			wantErr:     true,
		},
		{
			name:        "invalid offset format",
			queryParams: map[string]string{"offset": "invalid"},
			wantLimit:   0,
			wantOffset:  0,
			wantErr:     true,
		},
		{
			name:        "negative limit ignored",
			queryParams: map[string]string{"limit": "-10"},
			wantLimit:   50,
			wantOffset:  0,
			wantErr:     false,
		},
		{
			name:        "negative offset ignored",
			queryParams: map[string]string{"offset": "-5"},
			wantLimit:   50,
			wantOffset:  0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			// Set up request with query params
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			for key, val := range tt.queryParams {
				q.Add(key, val)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			gotLimit, gotOffset, err := v.ValidatePaginationParams(c)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaginationParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotLimit != tt.wantLimit {
					t.Errorf("ValidatePaginationParams() limit = %v, want %v", gotLimit, tt.wantLimit)
				}
				if gotOffset != tt.wantOffset {
					t.Errorf("ValidatePaginationParams() offset = %v, want %v", gotOffset, tt.wantOffset)
				}
			}
		})
	}
}

func TestValidateOptionalSearchQuery(t *testing.T) {
	v := NewRequestValidator()

	tests := []struct {
		name        string
		queryParams map[string]string
		want        string
		wantErr     bool
	}{
		{
			name:        "valid search query",
			queryParams: map[string]string{"search": "test query"},
			want:        "test query",
			wantErr:     false,
		},
		{
			name:        "alternative param name q",
			queryParams: map[string]string{"q": "test query"},
			want:        "test query",
			wantErr:     false,
		},
		{
			name:        "empty query is valid for optional",
			queryParams: map[string]string{},
			want:        "",
			wantErr:     false,
		},
		{
			name:        "query too short",
			queryParams: map[string]string{"search": "ab"},
			want:        "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			for key, val := range tt.queryParams {
				q.Add(key, val)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			got, err := v.ValidateOptionalSearchQuery(c)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOptionalSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("ValidateOptionalSearchQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateSortParams(t *testing.T) {
	v := NewRequestValidator()

	tests := []struct {
		name          string
		queryParams   map[string]string
		allowedFields []string
		wantSortBy    string
		wantSortOrder string
		wantErr       bool
	}{
		{
			name:          "valid sort params",
			queryParams:   map[string]string{"sort_by": "name", "sort_order": "asc"},
			allowedFields: []string{"name", "created_at"},
			wantSortBy:    "name",
			wantSortOrder: "asc",
			wantErr:       false,
		},
		{
			name:          "desc order",
			queryParams:   map[string]string{"sort_by": "created_at", "sort_order": "desc"},
			allowedFields: []string{"name", "created_at"},
			wantSortBy:    "created_at",
			wantSortOrder: "desc",
			wantErr:       false,
		},
		{
			name:          "no sort params",
			queryParams:   map[string]string{},
			allowedFields: []string{"name"},
			wantSortBy:    "",
			wantSortOrder: "",
			wantErr:       false,
		},
		{
			name:          "invalid sort field",
			queryParams:   map[string]string{"sort_by": "invalid_field"},
			allowedFields: []string{"name", "created_at"},
			wantSortBy:    "",
			wantSortOrder: "",
			wantErr:       true,
		},
		{
			name:          "invalid sort order",
			queryParams:   map[string]string{"sort_by": "name", "sort_order": "invalid"},
			allowedFields: []string{"name"},
			wantSortBy:    "",
			wantSortOrder: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			q := req.URL.Query()
			for key, val := range tt.queryParams {
				q.Add(key, val)
			}
			req.URL.RawQuery = q.Encode()
			c.Request = req

			gotSortBy, gotSortOrder, err := v.ValidateSortParams(c, tt.allowedFields)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSortParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotSortBy != tt.wantSortBy {
					t.Errorf("ValidateSortParams() sortBy = %v, want %v", gotSortBy, tt.wantSortBy)
				}
				if gotSortOrder != tt.wantSortOrder {
					t.Errorf("ValidateSortParams() sortOrder = %v, want %v", gotSortOrder, tt.wantSortOrder)
				}
			}
		})
	}
}

func TestValidateUserContext(t *testing.T) {
	v := NewRequestValidator()

	tests := []struct {
		name       string
		setupCtx   func(*gin.Context)
		wantUserID uint
		wantErr    bool
	}{
		{
			name: "valid user context",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(123))
			},
			wantUserID: 123,
			wantErr:    false,
		},
		{
			name: "missing user_id",
			setupCtx: func(c *gin.Context) {
				// Don't set user_id
			},
			wantUserID: 0,
			wantErr:    true,
		},
		{
			name: "zero user_id",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", uint(0))
			},
			wantUserID: 0,
			wantErr:    true,
		},
		{
			name: "invalid type",
			setupCtx: func(c *gin.Context) {
				c.Set("user_id", "not a uint")
			},
			wantUserID: 0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupCtx(c)

			got, err := v.ValidateUserContext(c)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.wantUserID {
				t.Errorf("ValidateUserContext() = %v, want %v", got, tt.wantUserID)
			}
		})
	}
}
