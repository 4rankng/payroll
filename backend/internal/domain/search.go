package domain

import "context"

// SearchFilters represents common search filters
type SearchFilters struct {
	Query     string
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// SearchResult represents a search result with relevance scoring
type SearchResult struct {
	ID        uint    `json:"id"`
	Title     string  `json:"title"`
	Subtitle  string  `json:"subtitle"`
	Type      string  `json:"type"`
	Relevance float64 `json:"relevance"`
	Data      any     `json:"data,omitempty"`
}

// SearchResults represents paginated search results
type SearchResults struct {
	Results     []SearchResult `json:"results"`
	Total       int64          `json:"total"`
	Query       string         `json:"query"`
	Page        int            `json:"page"`
	PageSize    int            `json:"page_size"`
	TotalPages  int            `json:"total_pages"`
	HasNextPage bool           `json:"has_next_page"`
	HasPrevPage bool           `json:"has_prev_page"`
	ProcessedIn int64          `json:"processed_in_ms"`
}

// Searchable represents any entity that can be searched
type Searchable interface {
	GetSearchTitle() string
	GetSearchSubtitle() string
	GetSearchType() string
	GetSearchData() any
}

// SearchService defines the interface for search operations
type SearchService interface {
	Search(ctx context.Context, query string, filters SearchFilters) (*SearchResults, error)
	SearchEmployees(ctx context.Context, query string, limit int) ([]*EmployeeWithProject, error)
	SearchProjects(ctx context.Context, query string, limit int) ([]*Project, error)
	SearchLedgerEntries(ctx context.Context, query string, limit int) ([]*LedgerEntry, error)
	GlobalSearch(ctx context.Context, query string, filters SearchFilters) (*SearchResults, error)
}

// TextNormalizer defines the interface for text normalization
type TextNormalizer interface {
	Normalize(text string) string
	NormalizeForSearch(searchTerm string) string
}
