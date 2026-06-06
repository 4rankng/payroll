package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNotificationTypes(t *testing.T) {
	// Test that all notification types are defined
	assert.Equal(t, NotificationType("timesheet_reminder"), NotificationTypeTimesheetReminder)
	assert.Equal(t, NotificationType("timesheet_approval"), NotificationTypeTimesheetApproval)
	assert.Equal(t, NotificationType("approval_request"), NotificationTypeApprovalRequest)
	assert.Equal(t, NotificationType("approval_result"), NotificationTypeApprovalResult)
	assert.Equal(t, NotificationType("payroll_complete"), NotificationTypePayrollComplete)
	assert.Equal(t, NotificationType("password_change"), NotificationTypePasswordChange)
	assert.Equal(t, NotificationType("loan_interest_due"), NotificationTypeLoanInterestDue)
	assert.Equal(t, NotificationType("payroll_report"), NotificationTypePayrollReport)
}

func TestNotificationPriority(t *testing.T) {
	assert.Equal(t, "⚠️ ", NotifyTypeImportant)
}

func TestNotificationChannels(t *testing.T) {
	assert.Equal(t, NotificationChannel("push"), NotificationChannelPush)
	assert.Equal(t, NotificationChannel("email"), NotificationChannelEmail)
}

func TestNotification_StructFields(t *testing.T) {
	recipientID := uint(10)
	emailRecipients := `["user1@example.com","user2@example.com"]`
	resendMessageID := "msg-123"
	readAt := time.Now()

	notification := Notification{
		ID:              1,
		SenderID:        5,
		RecipientID:     &recipientID,
		Type:            NotificationTypeTimesheetReminder,
		Channel:         NotificationChannelPush,
		Title:           "Reminder",
		Message:         "Please submit your timesheet",
		EmailRecipients: &emailRecipients,
		ResendMessageID: &resendMessageID,
		ReadAt:          &readAt,
	}

	assert.Equal(t, uint(1), notification.ID)
	assert.Equal(t, uint(5), notification.SenderID)
	assert.NotNil(t, notification.RecipientID)
	assert.Equal(t, uint(10), *notification.RecipientID)
	assert.Equal(t, NotificationTypeTimesheetReminder, notification.Type)
	assert.Equal(t, NotificationChannelPush, notification.Channel)
	assert.Equal(t, "Reminder", notification.Title)
	assert.Equal(t, "Please submit your timesheet", notification.Message)
	assert.NotNil(t, notification.EmailRecipients)
	assert.Contains(t, *notification.EmailRecipients, "user1@example.com")
	assert.NotNil(t, notification.ResendMessageID)
	assert.Equal(t, "msg-123", *notification.ResendMessageID)
	assert.NotNil(t, notification.ReadAt)
}

func TestNotification_NullableFields(t *testing.T) {
	notification := Notification{
		ID:              1,
		SenderID:        5,
		RecipientID:     nil, // Can be null for email notifications to external recipients
		Type:            NotificationTypePayrollReport,
		Channel:         NotificationChannelEmail,
		Title:           "Report",
		Message:         "Monthly payroll report",
		EmailRecipients: nil,
		ResendMessageID: nil,
		ReadAt:          nil,
	}

	assert.Nil(t, notification.RecipientID)
	assert.Nil(t, notification.EmailRecipients)
	assert.Nil(t, notification.ResendMessageID)
	assert.Nil(t, notification.ReadAt)
}

func TestNotification_EmailChannel(t *testing.T) {
	emailRecipients := `["admin@example.com"]`
	resendMessageID := "resend-abc-123"

	notification := Notification{
		ID:              1,
		SenderID:        1,
		RecipientID:     nil,
		Type:            NotificationTypePayrollReport,
		Channel:         NotificationChannelEmail,
		Title:           "Monthly Report",
		Message:         "Your monthly payroll report",
		EmailRecipients: &emailRecipients,
		ResendMessageID: &resendMessageID,
	}

	assert.Equal(t, NotificationChannelEmail, notification.Channel)
	assert.Nil(t, notification.RecipientID)
	assert.NotNil(t, notification.EmailRecipients)
	assert.NotNil(t, notification.ResendMessageID)
}

func TestNotification_PushChannel(t *testing.T) {
	recipientID := uint(10)

	notification := Notification{
		ID:          1,
		SenderID:    5,
		RecipientID: &recipientID,
		Type:        NotificationTypeTimesheetApproval,
		Channel:     NotificationChannelPush,
		Title:       "Approval Notification",
		Message:     "Your timesheet has been approved",
	}

	assert.Equal(t, NotificationChannelPush, notification.Channel)
	assert.NotNil(t, notification.RecipientID)
	assert.Equal(t, uint(10), *notification.RecipientID)
}

func TestNotification_ReadStatus(t *testing.T) {
	// Unread notification
	unreadNotification := Notification{
		ID:      1,
		ReadAt:  nil,
		Title:   "Unread",
		Message: "This notification is unread",
	}
	assert.Nil(t, unreadNotification.ReadAt)

	// Read notification
	readAt := time.Now()
	readNotification := Notification{
		ID:      2,
		ReadAt:  &readAt,
		Title:   "Read",
		Message: "This notification has been read",
	}
	assert.NotNil(t, readNotification.ReadAt)
}

func TestNotification_DifferentTypes(t *testing.T) {
	types := []NotificationType{
		NotificationTypeTimesheetReminder,
		NotificationTypeTimesheetApproval,
		NotificationTypeApprovalRequest,
		NotificationTypeApprovalResult,
		NotificationTypePayrollComplete,
		NotificationTypePasswordChange,
		NotificationTypeLoanInterestDue,
		NotificationTypePayrollReport,
	}

	for _, notifType := range types {
		notification := Notification{
			ID:       1,
			SenderID: 1,
			Type:     notifType,
			Channel:  NotificationChannelPush,
			Title:    "Test",
			Message:  "Test message",
		}
		assert.Equal(t, notifType, notification.Type)
	}
}
