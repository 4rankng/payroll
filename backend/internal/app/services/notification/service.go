package notification

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
)

var htmlTagRegex = regexp.MustCompile(`<[^>]+>`)

// stripHTML removes HTML tags and decodes common HTML entities for plain-text use
// (e.g. push notification bodies shown by the OS).
func stripHTML(s string) string {
	plain := htmlTagRegex.ReplaceAllString(s, "")
	// Decode a handful of common entities
	plain = strings.ReplaceAll(plain, "&amp;", "&")
	plain = strings.ReplaceAll(plain, "&lt;", "<")
	plain = strings.ReplaceAll(plain, "&gt;", ">")
	plain = strings.ReplaceAll(plain, "&quot;", "\"")
	plain = strings.ReplaceAll(plain, "&#39;", "'")
	plain = strings.ReplaceAll(plain, "&nbsp;", " ")
	return strings.TrimSpace(plain)
}

type PushSender interface {
	SendToUser(ctx context.Context, userID uint, title, body string) error
}

type NotificationService struct {
	notificationRepo domain.NotificationRepository
	userRepo         domain.UserRepository
	projectRepo      domain.ProjectRepository
	userService      *user.UserService
	pushSender       PushSender
	logger           *slog.Logger
}

func NewNotificationService(
	notificationRepo domain.NotificationRepository,
	userRepo domain.UserRepository,
	projectRepo domain.ProjectRepository,
	userService *user.UserService,
	pushSender PushSender,
	logger *slog.Logger,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		projectRepo:      projectRepo,
		userService:      userService,
		pushSender:       pushSender,
		logger:           logger,
	}
}

// CreateNotification creates a simple in-app notification with plain text content
func (s *NotificationService) CreateNotification(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string) error {
	return s.CreateNotificationWithContentType(ctx, userID, notificationType, title, message, domain.NotificationContentTypePlainText)
}

// CreateNotificationWithContentType creates a notification with specified content type
func (s *NotificationService) CreateNotificationWithContentType(ctx context.Context, userID uint, notificationType domain.NotificationType, title, message string, contentType domain.NotificationContentType) error {
	notification := &domain.Notification{
		SenderID:    constants.SystemUserID, // System-initiated push notifications
		RecipientID: &userID,
		Type:        notificationType,
		Channel:     domain.NotificationChannelPush,
		Title:       title,
		Message:     message,
		ContentType: contentType,
	}

	err := s.notificationRepo.Create(ctx, notification)
	if err != nil {
		return err
	}

	// Also send web push notification (fire-and-forget)
	s.firePushNotification(userID, title, message)

	return nil
}

// firePushNotification sends a web push notification in the background.
// The body is stripped of HTML tags so the OS notification shows plain text.
func (s *NotificationService) firePushNotification(userID uint, title, message string) {
	if s.pushSender == nil {
		return
	}
	plainBody := stripHTML(message)
	go func() {
		if err := s.pushSender.SendToUser(context.Background(), userID, title, plainBody); err != nil {
			s.logger.Error("failed to send push notification", "user_id", userID, "error", err)
		}
	}()
}

// GetUserNotifications retrieves notifications for a user with pagination
func (s *NotificationService) GetUserNotifications(ctx context.Context, userID uint, limit, offset int) ([]*domain.Notification, error) {
	return s.notificationRepo.GetByUserID(ctx, userID, limit, offset)
}

// GetUnreadNotifications retrieves unread notifications for a user
func (s *NotificationService) GetUnreadNotifications(ctx context.Context, userID uint) ([]*domain.Notification, error) {
	return s.notificationRepo.GetUnreadByUserID(ctx, userID)
}

// GetUnreadCount returns the count of unread notifications for a user
func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uint) (int64, error) {
	return s.notificationRepo.CountUnreadByUserID(ctx, userID)
}

// CountUserNotifications returns the total count of notifications for a user
func (s *NotificationService) CountUserNotifications(ctx context.Context, userID uint) (int64, error) {
	return s.notificationRepo.CountByUserID(ctx, userID)
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, notificationID uint) error {
	return s.notificationRepo.MarkAsRead(ctx, notificationID)
}

// MarkAsReadWithOwnership marks a notification as read after validating ownership
func (s *NotificationService) MarkAsReadWithOwnership(ctx context.Context, notificationID uint, userID uint) error {
	// First, get the notification to check ownership
	notifications, err := s.notificationRepo.GetByUserID(ctx, userID, 1000, 0)
	if err != nil {
		return err
	}

	// Check if the notification belongs to the user
	for _, notification := range notifications {
		if notification.ID == notificationID {
			return s.notificationRepo.MarkAsRead(ctx, notificationID)
		}
	}

	return domain.NewNotFoundError(constants.MsgNotificationNotFoundOrAccessDeniedVN)
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uint) error {
	return s.notificationRepo.MarkAllAsReadForUser(ctx, userID)
}

// MarkOldNotificationsAsRead marks notifications older than specified days as read for all users
func (s *NotificationService) MarkOldNotificationsAsRead(ctx context.Context, olderThanDays int) (int64, error) {
	return s.notificationRepo.MarkOldNotificationsAsRead(ctx, olderThanDays)
}

// NotifyUsersByRole sends notifications to all users with a specific role
func (s *NotificationService) NotifyUsersByRole(ctx context.Context, role domain.UserRole, notificationType domain.NotificationType, title, message string) error {
	// Get all active users with the specified role
	users, err := s.getUsersByRole(ctx, role)
	if err != nil {
		return err
	}

	// Create notifications for each user
	for _, userID := range users {
		if err := s.CreateNotification(ctx, userID, notificationType, title, message); err != nil {
			s.logger.Error("Failed to create notification", "error", err, "user_id", userID)
			continue
		}
	}

	return nil
}

// getUsersByRole is a helper method to get user IDs by role
func (s *NotificationService) getUsersByRole(ctx context.Context, role domain.UserRole) ([]uint, error) {
	// This is a simplified approach - in the real implementation you might want to use a more efficient query
	var userIDs []uint

	// Get count to initialize slice
	count, err := s.userRepo.CountByRole(ctx, role)
	if err != nil {
		return nil, err
	}

	if count == 0 {
		return userIDs, nil
	}

	// Get all users with pagination (assuming reasonable limits)
	users, err := s.userRepo.List(ctx, int(count), 0)
	if err != nil {
		return nil, err
	}

	// Filter by role and active status
	for _, user := range users {
		if user.Role == role {
			userIDs = append(userIDs, user.ID)
		}
	}

	return userIDs, nil
}

// NotifyUsersWithDefaultPassword checks users with default passwords and sends notifications
func (s *NotificationService) NotifyUsersWithDefaultPassword(ctx context.Context, adminPassword, partnerPassword string) ([]uint, error) {
	s.logger.Info("Starting default password notification check")

	var notifiedUserIDs []uint

	// Check admin users
	adminUsers, err := s.userRepo.ListByRole(ctx, domain.RoleAdmin)
	if err != nil {
		s.logger.Error("Failed to get admin users", "error", err)
		return nil, err
	}

	for _, user := range adminUsers {
		if s.userService.VerifyPasswordHash(adminPassword, user.Password) {
			title := "Cần đổi mật khẩu"
			message := "Bạn đang sử dụng mật khẩu mặc định. Vui lòng đổi mật khẩu ngay để bảo vệ tài khoản của bạn."

			if err := s.CreateNotification(ctx, user.ID, domain.NotificationTypePasswordChange, title, message); err != nil {
				s.logger.Error("Failed to create notification for admin user", "error", err, "user_id", user.ID)
				continue
			}
			notifiedUserIDs = append(notifiedUserIDs, user.ID)
			s.logger.Info("Notification sent to admin user", "user_id", user.ID)
		}
	}

	// Check partner users
	partnerUsers, err := s.userRepo.ListByRole(ctx, domain.RolePartner)
	if err != nil {
		s.logger.Error("Failed to get partner users", "error", err)
		return nil, err
	}

	for _, user := range partnerUsers {
		if s.userService.VerifyPasswordHash(partnerPassword, user.Password) {
			title := "Cần đổi mật khẩu"
			message := "Bạn đang sử dụng mật khẩu mặc định. Vui lòng đổi mật khẩu ngay để bảo vệ tài khoản của bạn."

			if err := s.CreateNotification(ctx, user.ID, domain.NotificationTypePasswordChange, title, message); err != nil {
				s.logger.Error("Failed to create notification for partner user", "error", err, "user_id", user.ID)
				continue
			}
			notifiedUserIDs = append(notifiedUserIDs, user.ID)
			s.logger.Info("Notification sent to partner user", "user_id", user.ID)
		}
	}

	s.logger.Info("Default password notification check completed", "notified_users", len(notifiedUserIDs))
	return notifiedUserIDs, nil
}

// NotifyPartnersByMonthlyProjects sends notifications to partners who created monthly payment projects
func (s *NotificationService) NotifyPartnersByMonthlyProjects(ctx context.Context, notificationType domain.NotificationType, title, message string) error {
	// Get all active projects
	filters := domain.ProjectFilters{
		ProjectStatus: []domain.ProjectStatus{domain.ProjectStatusRunning}, // Active projects
	}
	projects, err := s.projectRepo.List(ctx, filters)
	if err != nil {
		s.logger.Error("Failed to get active projects", "error", err)
		return err
	}

	if len(projects) == 0 {
		s.logger.Info("No active projects found, skipping partner notifications")
		return nil
	}

	// Collect unique partner user IDs from project creators
	partnerIDs := make(map[uint]bool)
	for _, project := range projects {
		// Only notify if creator is a partner
		creator, err := s.userRepo.GetByID(ctx, project.CreatedBy)
		if err != nil {
			s.logger.Error("Failed to get project creator", "error", err, "project_id", project.ID, "creator_id", project.CreatedBy)
			continue
		}

		if creator.Role == domain.RolePartner {
			partnerIDs[creator.ID] = true
		}
	}

	if len(partnerIDs) == 0 {
		s.logger.Info("No partner creators found for active projects, skipping notifications")
		return nil
	}

	// Send notifications to each unique partner
	successCount := 0
	for partnerID := range partnerIDs {
		if err := s.CreateNotification(ctx, partnerID, notificationType, title, message); err != nil {
			s.logger.Error("Failed to create notification for partner", "error", err, "partner_id", partnerID)
			continue
		}
		successCount++
	}

	s.logger.Info("Partner notifications sent", "total_active_projects", len(projects), "partners_notified", successCount)
	return nil
}

// SendCustomNotifications creates custom notifications for multiple users and roles
func (s *NotificationService) SendCustomNotifications(
	ctx context.Context,
	senderID uint,
	recipientIDs []uint,
	roles []domain.UserRole,
	title, message string,
	contentType domain.NotificationContentType,
) error {
	targets := make(map[uint]struct{})

	for _, recipientID := range recipientIDs {
		if recipientID == 0 {
			continue
		}
		targets[recipientID] = struct{}{}
	}

	for _, role := range roles {
		users, err := s.getUsersByRole(ctx, role)
		if err != nil {
			return err
		}
		for _, userID := range users {
			if userID == 0 {
				continue
			}
			targets[userID] = struct{}{}
		}
	}

	if len(targets) == 0 {
		return domain.NewValidationError(constants.MsgNoRecipientsOrRolesProvidedVN)
	}

	finalContentType := contentType
	if finalContentType == "" {
		finalContentType = domain.NotificationContentTypePlainText
	}

	for userID := range targets {
		if err := s.CreateCustomNotification(ctx, senderID, userID, title, message, finalContentType); err != nil {
			return err
		}
	}

	return nil
}

// CreateCustomNotification creates a custom notification from admin to a specific user with specified content type
func (s *NotificationService) CreateCustomNotification(ctx context.Context, senderID uint, recipientID uint, title, message string, contentType domain.NotificationContentType) error {
	// Validate that recipient exists
	recipient, err := s.userRepo.GetByID(ctx, recipientID)
	if err != nil {
		return domain.NewNotFoundError(constants.MsgRecipientUserNotFoundVN)
	}
	if !recipient.IsActive() {
		return domain.NewValidationError(constants.MsgRecipientUserNotActiveVN)
	}

	// Validate content type
	if contentType == "" {
		contentType = domain.NotificationContentTypePlainText
	}

	notification := &domain.Notification{
		SenderID:    senderID,
		RecipientID: &recipientID,
		Type:        domain.NotificationTypeCustom,
		Channel:     domain.NotificationChannelPush,
		Title:       title,
		Message:     message,
		ContentType: contentType,
	}

	if err := s.notificationRepo.Create(ctx, notification); err != nil {
		s.logger.Error("Failed to create custom notification", "error", err, "sender_id", senderID, "recipient_id", recipientID)
		return err
	}

	// Fire web push notification
	s.firePushNotification(recipientID, title, message)

	s.logger.Info("Custom notification created", "sender_id", senderID, "recipient_id", recipientID, "content_type", contentType)
	return nil
}
