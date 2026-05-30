package reporting

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

// SearchService implements domain.SearchService
type SearchService struct {
	employeeRepo domain.EmployeeRepository
	projectRepo  domain.ProjectRepository
	ledgerRepo   domain.LedgerEntryRepository
}

// NewSearchService creates a new search service instance
func NewSearchService(
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	ledgerRepo domain.LedgerEntryRepository,
) *SearchService {
	return &SearchService{
		employeeRepo: employeeRepo,
		projectRepo:  projectRepo,
		ledgerRepo:   ledgerRepo,
	}
}

// Search performs a unified search across all searchable entities
func (s *SearchService) Search(ctx context.Context, query string, filters domain.SearchFilters) (*domain.SearchResults, error) {
	start := clock.Now()

	if query == "" {
		return &domain.SearchResults{
			Results:     []domain.SearchResult{},
			Total:       0,
			Query:       query,
			ProcessedIn: time.Since(start).Milliseconds(),
		}, nil
	}

	if len(strings.TrimSpace(query)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	// Set default limit if not provided - use 100 for max results
	limit := filters.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}

	var allResults []domain.SearchResult
	var totalCount int64

	// Search employees
	employees, err := s.SearchEmployees(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search employees: %w", err)
	}

	for _, emp := range employees {
		emailStr := ""
		if emp.Email != nil {
			emailStr = *emp.Email
		}

		result := domain.SearchResult{
			ID:        emp.ID,
			Title:     emp.Fullname,
			Subtitle:  fmt.Sprintf("%s | %s", emailStr, emp.CCCD),
			Type:      "employee",
			Relevance: s.calculateEmployeeRelevance(emp, query),
			Data:      emp,
		}
		allResults = append(allResults, result)
	}
	totalCount += int64(len(employees))

	// Search projects (to be implemented when ProjectRepository has search)
	// projects, err := s.SearchProjects(ctx, query, limit)
	// ... similar implementation

	// Search ledger entries (to be implemented when LedgerRepository has search)
	// entries, err := s.SearchLedgerEntries(ctx, query, limit)
	// ... similar implementation

	// Sort by relevance (descending)
	s.sortResultsByRelevance(allResults)

	// Apply pagination to results
	offset := filters.Offset
	if offset < 0 {
		offset = 0
	}

	paginatedResults := s.paginateResults(allResults, offset, limit)

	// Calculate pagination info
	page := (offset / limit) + 1
	totalPages := (int(totalCount) + limit - 1) / limit

	return &domain.SearchResults{
		Results:     paginatedResults,
		Total:       totalCount,
		Query:       query,
		Page:        page,
		PageSize:    limit,
		TotalPages:  totalPages,
		HasNextPage: page < totalPages,
		HasPrevPage: page > 1,
		ProcessedIn: time.Since(start).Milliseconds(),
	}, nil
}

// SearchEmployees searches for employees using Vietnamese text normalization
func (s *SearchService) SearchEmployees(ctx context.Context, query string, limit int) ([]*domain.EmployeeWithProject, error) {
	if query == "" {
		return nil, domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}

	if len(strings.TrimSpace(query)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	if limit <= 0 || limit > 100 {
		limit = 100
	}

	return s.employeeRepo.SearchEmployees(ctx, query, limit)
}

// SearchProjects searches for projects using Vietnamese text normalization
func (s *SearchService) SearchProjects(ctx context.Context, query string, limit int) ([]*domain.Project, error) {
	if query == "" {
		return nil, domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}

	if len(strings.TrimSpace(query)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	if limit <= 0 || limit > 100 {
		limit = 100
	}

	return s.projectRepo.SearchProjects(ctx, query, limit)
}

// SearchLedgerEntries searches for ledger entries using Vietnamese text normalization
func (s *SearchService) SearchLedgerEntries(ctx context.Context, query string, limit int) ([]*domain.LedgerEntry, error) {
	if query == "" {
		return nil, domain.NewValidationError(constants.MsgSearchQueryRequiredVN)
	}

	if len(strings.TrimSpace(query)) < 3 {
		return nil, domain.NewValidationError(constants.MsgSearchQueryMinLengthVN)
	}

	if limit <= 0 || limit > 100 {
		limit = 100
	}

	return s.ledgerRepo.SearchLedgerEntries(ctx, query, limit)
}

// GlobalSearch performs a comprehensive search across all entities
func (s *SearchService) GlobalSearch(ctx context.Context, query string, filters domain.SearchFilters) (*domain.SearchResults, error) {
	return s.Search(ctx, query, filters)
}

// Helper methods

// calculateEmployeeRelevance calculates relevance score for employee search results
func (s *SearchService) calculateEmployeeRelevance(emp *domain.EmployeeWithProject, query string) float64 {
	normalizedQuery := utils.NormalizeVietnamese(query)
	score := 0.0

	// Check exact matches (highest score)
	if strings.Contains(utils.NormalizeVietnamese(emp.Fullname), normalizedQuery) {
		score += 100.0
	}
	if emp.Email != nil && strings.Contains(utils.NormalizeVietnamese(*emp.Email), normalizedQuery) {
		score += 90.0
	}
	if strings.Contains(emp.CCCD, query) {
		score += 95.0
	}
	if strings.Contains(emp.Mobile, query) {
		score += 85.0
	}

	// Boost score if employee has active project
	if emp.ProjectID != nil {
		score += 10.0
	}

	// Normalize score to 0-100 range
	if score > 100 {
		score = 100
	}

	return score
}

// sortResultsByRelevance sorts search results by relevance score (descending)
func (s *SearchService) sortResultsByRelevance(results []domain.SearchResult) {
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[i].Relevance < results[j].Relevance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// paginateResults applies pagination to search results
func (s *SearchService) paginateResults(results []domain.SearchResult, offset, limit int) []domain.SearchResult {
	if offset >= len(results) {
		return []domain.SearchResult{}
	}

	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end]
}
