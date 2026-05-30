package handlers

import (
	"fmt"
	"strconv"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/notification"
	"api-server/internal/constants"
	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type NotificationHandler struct {
	notificationService *notification.NotificationService
}

func NewNotificationHandler(notificationService *notification.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// GetNotifications retrieves user notifications with pagination
// @Summary Get user notifications
// @Description Get a paginated list of notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} dto.PaginatedResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications [get]
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}
	userID := *userIDPtr

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 20
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Convert to offset/limit for service
	limit := pageSize
	offset := (page - 1) * pageSize

	notifications, err := h.notificationService.GetUserNotifications(c.Request.Context(), userID, limit, offset)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveNotificationsVN)
		return
	}

	// Convert to response format
	notificationResponses := make([]dto.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		notificationResponses[i] = dto.NotificationResponse{
			ID:          notification.ID,
			Type:        string(notification.Type),
			Channel:     string(notification.Channel),
			Title:       notification.Title,
			Message:     notification.Message,
			ContentType: string(notification.ContentType),
			ReadAt:      notification.ReadAt,
			CreatedAt:   notification.CreatedAt,
		}
	}

	// Get total count for pagination
	total, err := h.notificationService.CountUserNotifications(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountNotificationsVN)
		return
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	response.SuccessWithPagination(c, notificationResponses, "Notifications retrieved successfully", pagination)
}

// GetUnreadNotifications retrieves unread notifications for the user
// @Summary Get unread notifications
// @Description Get all unread notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} dto.UnreadNotificationsResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications/unread [get]
func (h *NotificationHandler) GetUnreadNotifications(c *gin.Context) {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}
	userID := *userIDPtr

	notifications, err := h.notificationService.GetUnreadNotifications(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToRetrieveNotificationsVN)
		return
	}

	// Convert to response format
	notificationResponses := make([]dto.NotificationResponse, len(notifications))
	for i, notification := range notifications {
		notificationResponses[i] = dto.NotificationResponse{
			ID:          notification.ID,
			Type:        string(notification.Type),
			Channel:     string(notification.Channel),
			Title:       notification.Title,
			Message:     notification.Message,
			ContentType: string(notification.ContentType),
			ReadAt:      notification.ReadAt,
			CreatedAt:   notification.CreatedAt,
		}
	}

	response.Success(c, gin.H{
		"notifications": notificationResponses,
		"count":         int64(len(notificationResponses)),
	}, constants.MsgUnreadNotificationsRetrievedVN)
}

// GetUnreadCount retrieves the count of unread notifications
// @Summary Get unread notification count
// @Description Get the count of unread notifications for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} dto.UnreadCountResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications/unread/count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}
	userID := *userIDPtr

	count, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetUnreadCountVN)
		return
	}

	response.Success(c, dto.UnreadCountResponse{
		Count: count,
	}, "Unread count retrieved successfully")
}

// MarkAsRead marks a notification as read
// @Summary Mark notification as read
// @Description Mark a specific notification as read
// @Tags notifications
// @Accept json
// @Produce json
// @Param id path int true "Notification ID"
// @Success 200 {object} dto.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}

	// Parse notification ID
	idParam := c.Param("id")
	notificationID, err := strconv.ParseUint(idParam, 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidNotificationIDVN)
		return
	}

	// Get current user ID for ownership validation
	ctxUserID := auditctx.GetUserID(c.Request.Context())
	if ctxUserID == nil {
		response.Forbidden(c, "User authentication required")
		return
	}
	userID := *ctxUserID

	err = h.notificationService.MarkAsReadWithOwnership(c.Request.Context(), uint(notificationID), userID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, "Notification not found or access denied")
			return
		}
		response.InternalServerError(c, constants.MsgFailedToMarkNotificationReadVN)
		return
	}

	response.Success(c, nil, "Notification marked as read successfully")
}

// MarkAllAsRead marks all notifications as read for the user
// @Summary Mark all notifications as read
// @Description Mark all notifications as read for the authenticated user
// @Tags notifications
// @Accept json
// @Produce json
// @Success 200 {object} dto.SuccessResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userIDPtr := auditctx.GetUserID(c.Request.Context())
	if userIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}
	userID := *userIDPtr

	err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToMarkAllNotificationsVN)
		return
	}

	response.Success(c, nil, "All notifications marked as read successfully")
}

// NotifyDefaultPasswordUsers sends notifications to users with default passwords
// @Summary Notify users with default passwords
// @Description Send notifications to users who are still using default passwords
// @Tags notifications
// @Accept json
// @Produce json
// @Param request body dto.DefaultPasswordRequest true "Default passwords for admin and partner roles"
// @Success 200 {object} dto.DefaultPasswordResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notifications/no-default-password [post]
func (h *NotificationHandler) NotifyDefaultPasswordUsers(c *gin.Context) {
	var req dto.DefaultPasswordRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	notifiedUserIDs, err := h.notificationService.NotifyUsersWithDefaultPassword(c.Request.Context(), req.Admin, req.Partner)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToNotifyDefaultPasswordUsersVN)
		return
	}

	count := len(notifiedUserIDs)
	message := fmt.Sprintf("Có %d người dùng hiện tại cần đổi mật khẩu", count)

	response.Success(c, notifiedUserIDs, message)
}

// CreateCustomNotification allows admin to broadcast a custom notification
// @Summary Send custom notification
// @Description Allows admin to send a custom notification to specific users or roles
// @Tags notifications
// @Accept json
// @Produce json
// @Param request body dto.CustomNotificationRequest true "Custom notification request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Failure 403 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Failure 500 {object} response.ErrorResponse
// @Security ApiKeyAuth
// @Router /notification [post]
func (h *NotificationHandler) CreateCustomNotification(c *gin.Context) {
	// Get sender (admin) ID from context
	senderIDPtr := auditctx.GetUserID(c.Request.Context())
	if senderIDPtr == nil {
		response.Forbidden(c, "User not authenticated")
		return
	}
	senderID := *senderIDPtr

	// Verify sender is admin
	userRole := c.GetString(constants.CtxUserRole)
	if userRole != string(domain.RoleAdmin) {
		response.Forbidden(c, constants.MsgOnlyAdminCanSendNotificationsVN)
		return
	}

	var req dto.CustomNotificationRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Parse and validate content type (default to plain_text if not provided)
	contentType := domain.NotificationContentType(req.ContentType)
	if contentType == "" {
		contentType = domain.NotificationContentTypePlainText
	}

	switch contentType {
	case domain.NotificationContentTypePlainText, domain.NotificationContentTypeHTML, domain.NotificationContentTypeMarkdown:
		// valid
	default:
		response.BadRequest(c, constants.MsgInvalidContentTypeVN)
		return
	}

	if len(req.RecipientIDs) == 0 && !req.ToAllAdmins && !req.ToAllPartners && !req.ToAllEmployees {
		response.BadRequest(c, constants.MsgAtLeastOneRecipientRequiredVN)
		return
	}

	targetRoles := make([]domain.UserRole, 0, 3)
	if req.ToAllAdmins {
		targetRoles = append(targetRoles, domain.RoleAdmin)
	}
	if req.ToAllPartners {
		targetRoles = append(targetRoles, domain.RolePartner)
	}
	if req.ToAllEmployees {
		targetRoles = append(targetRoles, domain.RoleEmployee)
	}

	err := h.notificationService.SendCustomNotifications(
		c.Request.Context(),
		senderID,
		req.RecipientIDs,
		targetRoles,
		req.Title,
		req.Message,
		contentType,
	)
	if err != nil {
		if domain.IsNotFoundError(err) {
			response.NotFound(c, err.Error())
			return
		}
		if domain.IsValidationError(err) {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, constants.MsgFailedToCreateNotificationVN)
		return
	}

	response.Success(c, nil, "Gửi thông báo thành công")
}
