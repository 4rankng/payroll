package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/dashboard/converter"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
)

// GetNewEmployees retrieves recently added employees
func (s *Service) GetNewEmployees(ctx context.Context, req *dto.NewEmployeesRequest) (*dto.NewEmployeesResponse, *response.Pagination, error) {
	s.logger.Info("Getting new employees", "page", req.Page, "pageSize", req.PageSize, "days", req.Days)

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 5
	}
	if req.Days <= 0 {
		req.Days = 30
	}

	// Try to get from cache (60-second TTL)
	type cachedResult struct {
		Response   dto.NewEmployeesResponse
		Pagination response.Pagination
	}

	cacheKey := fmt.Sprintf("dashboard:new-employees:page:%d:size:%d:days:%d", req.Page, req.PageSize, req.Days)
	var cached cachedResult
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		s.logger.Info("New employees retrieved from cache")
		return &cached.Response, &cached.Pagination, nil
	}

	// Calculate date range
	endDate := clock.Now()
	startDate := endDate.AddDate(0, 0, -req.Days)

	// Count total recent employees with efficient query
	totalCount, err := s.EmployeeRepo.CountRecentEmployees(ctx, startDate, endDate)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgFailedToCountNewEmployeesVN, err)
	}

	// Calculate pagination
	offset := (req.Page - 1) * req.PageSize
	limit := req.PageSize

	// Get paginated employees with proper LIMIT/OFFSET
	employees, err := s.EmployeeRepo.GetRecentEmployeesPaginated(ctx, startDate, endDate, limit, offset)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgFailedToGetRecentEmployeesVN, err)
	}

	// Collect employee IDs for batch project loading
	employeeIDs := make([]uint, len(employees))
	for i, emp := range employees {
		employeeIDs[i] = emp.ID
	}

	// Batch load current projects for all employees
	projectsByEmployee, err := s.ProjectEmployeeRepo.GetCurrentProjectsForEmployees(ctx, employeeIDs)
	if err != nil {
		s.logger.Warn("Failed to batch load projects", "error", err)
		projectsByEmployee = make(map[uint]*domain.Project)
	}

	// Build response items using shared converter
	employeeItems := converter.MapNewEmployeeItems(employees, projectsByEmployee)

	// Calculate total pages
	totalPages := (totalCount + int64(req.PageSize) - 1) / int64(req.PageSize)

	pagination := &response.Pagination{
		Page:         req.Page,
		PageSize:     req.PageSize,
		TotalPages:   int(totalPages),
		TotalRecords: int(totalCount),
	}

	response := dto.NewEmployeesResponse(employeeItems)

	// Cache the result for 60 seconds
	toCache := cachedResult{
		Response:   response,
		Pagination: *pagination,
	}
	if err := s.CacheService.Set(ctx, cacheKey, toCache, constants.EmployeeListCacheTTL); err != nil {
		s.logger.Warn("Failed to cache new employees", "error", err)
	}

	s.logger.Info("New employees retrieved successfully", "count", len(employeeItems), "total", totalCount)
	return &response, pagination, nil
}
