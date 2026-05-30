package dashboard

import (
	"context"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/dashboard/converter"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
)

// GetRecentActivities retrieves recent system activities
func (s *Service) GetRecentActivities(ctx context.Context, req *dto.RecentActivitiesRequest) (*dto.RecentActivitiesResponse, *response.Pagination, error) {
	s.logger.Info("Getting recent activities", "page", req.Page, "pageSize", req.PageSize, "sortBy", req.SortBy, "sortOrder", req.SortOrder)

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}
	if req.SortBy == "" {
		req.SortBy = "created_at"
	}
	if req.SortOrder == "" {
		req.SortOrder = "desc"
	}

	// Try to get from cache (30-second TTL for fresh data)
	type cachedResult struct {
		Response   dto.RecentActivitiesResponse
		Pagination response.Pagination
	}

	cacheKey := fmt.Sprintf("dashboard:recent-activities:page:%d:size:%d:sort:%s:%s", req.Page, req.PageSize, req.SortBy, req.SortOrder)
	var cached cachedResult
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		s.logger.Info("Recent activities retrieved from cache")
		return &cached.Response, &cached.Pagination, nil
	}

	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Create filters for audit logs
	filters := domain.AuditFilters{
		Limit:     req.PageSize,
		Offset:    offset,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	// Get audit logs with pagination
	auditLogs, err := s.AuditLogRepo.List(ctx, filters)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgFailedToGetRecentActivitiesVN, err)
	}

	// Get total count for pagination
	totalCount, err := s.AuditLogRepo.Count(ctx, filters)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgFailedToCountActivitiesVN, err)
	}

	repos := converter.Repositories{
		UserRepo:     s.UserRepo,
		EmployeeRepo: s.EmployeeRepo,
		ProjectRepo:  s.ProjectRepo,
	}

	// Use batch conversion to avoid N+1 query problem
	activitiesPtr, err := converter.ConvertAuditLogsToActivities(auditLogs, repos)
	if err != nil {
		return nil, nil, domain.NewInternalError(constants.MsgFailedToConvertAuditLogsToActivitiesVN, err)
	}

	// Convert []*dto.ActivityItem to []dto.ActivityItem
	activities := make([]dto.ActivityItem, 0, len(activitiesPtr))
	for _, activity := range activitiesPtr {
		if activity != nil {
			activities = append(activities, *activity)
		}
	}

	// Calculate total pages
	totalPages := int(totalCount+int64(req.PageSize)-1) / req.PageSize

	pagination := &response.Pagination{
		Page:         req.Page,
		PageSize:     req.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(totalCount),
	}

	resp := dto.RecentActivitiesResponse(activities)

	// Cache the result for 30 seconds
	toCache := cachedResult{
		Response:   resp,
		Pagination: *pagination,
	}
	if err := s.CacheService.Set(ctx, cacheKey, toCache, constants.CashFlowSummaryCacheTTL); err != nil {
		s.logger.Warn("Failed to cache recent activities", "error", err)
	}

	s.logger.Info("Recent activities retrieved successfully", "count", len(activities), "total", totalCount)
	return &resp, pagination, nil
}
