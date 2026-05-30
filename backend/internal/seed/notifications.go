package seed

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	mathrand "math/rand/v2"
	"time"

	"api-server/internal/constants"
	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (s *Seeder) seedNotifications(ctx context.Context, db *gorm.DB) error {
	var users []domain.User
	if err := db.Find(&users).Error; err != nil {
		return fmt.Errorf("find users: %w", err)
	}

	notificationTypes := []struct {
		notifType string
		title     string
		message   string
	}{
		{"payroll_completed", "Lương đã được xử lý", "Lương tháng này của bạn đã được xử lý thành công."},
		{"timesheet_approved", "Bảng chấm công được phê duyệt", "Bảng chấm công của bạn tuần trước đã được phê duyệt."},
		{"project_assigned", "Phân công dự án mới", "Bạn đã được phân công vào dự án mới."},
		{"system_maintenance", "Bảo trì hệ thống", "Hệ thống sẽ được bảo trì vào cuối tuần này."},
		{"payment_reminder", "Nhắc nhở thanh toán", "Hạn thanh toán lương sắp đến, vui lòng chuẩn bị."},
	}

	for _, user := range users {
		numNotifications := 2 + mathrand.IntN(3)
		for range numNotifications {
			notif := notificationTypes[mathrand.IntN(len(notificationTypes))]
			createdAt := clock.Now().AddDate(0, 0, -mathrand.IntN(30))

			var readAt *time.Time
			if mathrand.IntN(10) >= 4 {
				readTime := createdAt.Add(time.Duration(mathrand.IntN(24)) * time.Hour)
				readAt = &readTime
			}

			recipientID := user.ID
			notification := struct {
				ID          uint       `gorm:"primarykey;type:bigint unsigned"`
				SenderID    uint       `gorm:"type:bigint unsigned;not null"`
				RecipientID *uint      `gorm:"type:bigint unsigned"`
				Type        string     `gorm:"type:varchar(50);not null"`
				Channel     string     `gorm:"type:varchar(20);not null;default:'in_app'"`
				Title       string     `gorm:"type:varchar(255);not null"`
				Message     string     `gorm:"type:text;not null"`
				ReadAt      *time.Time `gorm:"type:timestamp"`
				CreatedAt   time.Time  `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP"`
			}{
				SenderID:    constants.SystemUserID,
				RecipientID: &recipientID,
				Type:        notif.notifType,
				Channel:     "in_app",
				Title:       notif.title,
				Message:     notif.message,
				ReadAt:      readAt,
				CreatedAt:   createdAt,
			}

			if err := db.Model(&domain.Notification{}).Create(&notification).Error; err != nil {
				return fmt.Errorf("create notification for user %s: %w", user.Username, err)
			}
		}
	}

	return nil
}
