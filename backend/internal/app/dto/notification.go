package dto

import (
	"time"
)

// NotificationResponse represents the response containing notification data
type NotificationResponse struct {
	ID          uint       `json:"id"`
	Type        string     `json:"type"`
	Channel     string     `json:"channel"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	ContentType string     `json:"content_type"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// UnreadCountResponse represents the response containing unread notification count
type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

// UnreadNotificationsResponse represents the response containing unread notifications with count
type UnreadNotificationsResponse struct {
	Data  []NotificationResponse `json:"data"`
	Count int64                  `json:"count"`
}

// PaginatedResponse represents a paginated response structure
type PaginatedResponse struct {
	Data   interface{} `json:"data"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
	Total  int         `json:"total"`
}

// DefaultPasswordRequest represents the request for checking default passwords
type DefaultPasswordRequest struct {
	Admin   string `json:"admin" binding:"required"`
	Partner string `json:"partner" binding:"required"`
}

// DefaultPasswordResponse represents the response for default password notifications
type DefaultPasswordResponse struct {
	Data []uint `json:"data"`
}

// NotificationRequest represents the request for creating a notification
type NotificationRequest struct {
	RecipientID uint   `json:"recipient_id"`
	Title       string `json:"title" binding:"required"`
	Message     string `json:"message" binding:"required"`
	Type        string `json:"type"`
	ContentType string `json:"content_type"` // optional: "plain_text" (default), "html", "markdown"
	Channel     string `json:"channel"`
}

// EmailNotificationRequest represents the request for sending an email notification
type EmailNotificationRequest struct {
	RecipientEmail string `json:"recipient_email" binding:"required,email"`
	Subject        string `json:"subject" binding:"required"`
	Body           string `json:"body" binding:"required"`
	HTMLBody       string `json:"html_body"`
	Template       string `json:"template"`
}

// BroadcastNotificationRequest represents the request for broadcasting a notification
type BroadcastNotificationRequest struct {
	Title          string `json:"title" binding:"required"`
	Message        string `json:"message" binding:"required"`
	ContentType    string `json:"content_type"` // optional: "plain_text" (default), "html", "markdown"
	ToAllPartners  bool   `json:"to_all_partners"`
	ToAllAdmins    bool   `json:"to_all_admins"`
	ToAllEmployees bool   `json:"to_all_employees"`
	RecipientIDs   []uint `json:"recipient_ids"` // if specific recipients are specified
	Channel        string `json:"channel"`
}

// CustomNotificationRequest represents the request for creating a custom notification
type CustomNotificationRequest struct {
	RecipientIDs   []uint `json:"recipient_ids"` // nil or empty slice means use role broadcast flags instead
	Title          string `json:"title" binding:"required"`
	Message        string `json:"message" binding:"required"`
	ContentType    string `json:"content_type"` // optional: "plain_text" (default), "html", "markdown"
	ToAllPartners  bool   `json:"to_all_partners"`
	ToAllAdmins    bool   `json:"to_all_admins"`
	ToAllEmployees bool   `json:"to_all_employees"`
}
