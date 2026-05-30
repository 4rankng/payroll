package dashboard

import (
	"context"
	"fmt"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"
)

// GetSystemNotifications retrieves system notifications
func (s *Service) GetSystemNotifications(ctx context.Context, req *dto.SystemNotificationsRequest) (*dto.SystemNotificationsResponse, *response.Pagination, error) {
	s.logger.Info("Getting system notifications", "page", req.Page, "pageSize", req.PageSize, "include_read", req.IncludeRead)

	// Set defaults
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 20 {
		req.PageSize = 5
	}

	// Try to get from cache (15-second TTL for user-specific fresh data)
	type cachedResult struct {
		Response   dto.SystemNotificationsResponse
		Pagination response.Pagination
	}

	includeReadStr := "unread"
	if req.IncludeRead {
		includeReadStr = "all"
	}
	cacheKey := fmt.Sprintf("dashboard:notifications:page:%d:size:%d:mode:%s", req.Page, req.PageSize, includeReadStr)
	var cached cachedResult
	if err := s.CacheService.Get(ctx, cacheKey, &cached); err == nil {
		s.logger.Info("System notifications retrieved from cache")
		return &cached.Response, &cached.Pagination, nil
	}

	// Calculate offset
	offset := (req.Page - 1) * req.PageSize

	// Fetch data efficiently using separate queries instead of fetching 1000 records
	var notifications []*domain.Notification
	var totalCount int64
	var unreadCount int64
	var err error

	if req.IncludeRead {
		// Get all notifications with pagination
		notifications, err = s.NotificationRepo.GetByUserID(ctx, 0, req.PageSize, offset)
		if err != nil {
			return nil, nil, domain.NewInternalError(constants.MsgFailedToGetNotificationsVN, err)
		}

		// Get counts using efficient COUNT queries
		totalCount, err = s.NotificationRepo.CountByUserID(ctx, 0)
		if err != nil {
			return nil, nil, domain.NewInternalError(constants.MsgFailedToCountNotificationsVN, err)
		}

		unreadCount, err = s.NotificationRepo.CountUnreadByUserID(ctx, 0)
		if err != nil {
			return nil, nil, domain.NewInternalError(constants.MsgFailedToCountUnreadNotificationsVN, err)
		}
	} else {
		// Get only unread notifications
		notifications, err = s.NotificationRepo.GetUnreadByUserID(ctx, 0)
		if err != nil {
			return nil, nil, domain.NewInternalError(constants.MsgFailedToGetUnreadNotificationsVN, err)
		}

		// For unread-only mode, total count = unread count
		unreadCount = int64(len(notifications))
		totalCount = unreadCount

		// Apply pagination manually to unread notifications
		start := offset
		end := offset + req.PageSize
		if start > len(notifications) {
			notifications = []*domain.Notification{}
		} else {
			if end > len(notifications) {
				end = len(notifications)
			}
			notifications = notifications[start:end]
		}
	}

	notificationItems := make([]dto.NotificationItem, 0, len(notifications))
	for _, notif := range notifications {
		// Enhance notification message based on type and context
		enhancedMessage := getEnhancedNotificationMessage(notif)

		item := dto.NotificationItem{
			ID:        int(notif.ID),
			Title:     notif.Title,
			Message:   enhancedMessage,
			Type:      string(notif.Type),
			IsRead:    notif.ReadAt != nil,
			ActionURL: nil,
			CreatedAt: notif.CreatedAt,
			Icon:      getNotificationIcon(string(notif.Type)),
		}
		notificationItems = append(notificationItems, item)
	}

	// Calculate total pages
	totalPages := int((totalCount + int64(req.PageSize) - 1) / int64(req.PageSize))

	pagination := &response.Pagination{
		Page:         req.Page,
		PageSize:     req.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(totalCount),
	}

	response := &dto.SystemNotificationsResponse{
		Notifications: notificationItems,
		UnreadCount:   int(unreadCount),
	}

	// Cache the result for 15 seconds (user-specific data needs freshness)
	toCache := cachedResult{
		Response:   *response,
		Pagination: *pagination,
	}
	if err := s.CacheService.Set(ctx, cacheKey, toCache, constants.DashboardSummaryCacheTTL); err != nil {
		s.logger.Warn("Failed to cache system notifications", "error", err)
	}

	s.logger.Info("System notifications retrieved successfully", "total", len(notificationItems), "unread", unreadCount)
	return response, pagination, nil
}

// getEnhancedNotificationMessage creates more contextual notification messages
func getEnhancedNotificationMessage(notif *domain.Notification) string {
	if notif.Message == "" {
		return "Thông báo từ hệ thống"
	}

	// Check if message is too generic and enhance it
	switch string(notif.Type) {
	case "timesheet_approval":
		if notif.Message == "Timesheet needs approval" {
			return "Có chấm công cần được phê duyệt"
		}
	case "payroll_ready":
		if notif.Message == "Payroll is ready" {
			return "Bảng lương đã sẵn sàng để xử lý"
		}
	case "system_alert":
		if notif.Message == "System alert" {
			return "Cảnh báo từ hệ thống cần được xem xét"
		}
	case "employee_update":
		if notif.Message == "Employee updated" {
			return "Thông tin nhân viên đã được cập nhật"
		}
	case "project_update":
		if notif.Message == "Project updated" {
			return "Thông tin dự án đã được cập nhật"
		}
	}

	// Return original message if it's already meaningful
	return notif.Message
}

// getNotificationIcon returns appropriate icon based on notification type
func getNotificationIcon(notificationType string) string {
	switch notificationType {
	case "timesheet_approval":
		return "clock"
	case "payroll_ready":
		return "dollar-sign"
	case "system_error":
		return "alert-triangle"
	case "security_alert":
		return "shield-alert"
	case "employee_update":
		return "user"
	case "project_update":
		return "folder"
	case "system_alert":
		return "bell"
	default:
		return "bell"
	}
}
