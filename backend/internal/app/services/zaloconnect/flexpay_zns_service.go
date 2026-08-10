package zaloconnect

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/infra/zalo"
)

// FlexPaySalaryTemplateID is the ZNS template for salary/payment notifications.
// Template: SalaryNotification-v1
// Status: Pending approval (Đang duyệt - 2-3 days)
const FlexPaySalaryTemplateID = "619686"

// FlexPayZNSData holds the data for ZNS notification.
type FlexPayZNSData struct {
	EmployeeName string    // customer_name (max 30 chars)
	Amount       int64     // max_amount (in VND, max 20 digits)
	ExpiryDate   time.Time // expiry_date (DD/MM/YYYY format)
	Mobile       string    // Phone number for ZNS
}

// FlexPayZNSService sends ZNS notifications for FlexPay salary notifications.
type FlexPayZNSService struct {
	provider zalo.Sender
	logger   *slog.Logger
	enabled  func(context.Context) (bool, error) // Feature flag check
}

// NewFlexPayZNSService creates a new FlexPay ZNS service.
func NewFlexPayZNSService(provider zalo.Sender, logger *slog.Logger, enabledCheck func(context.Context) (bool, error)) *FlexPayZNSService {
	if logger == nil {
		logger = slog.Default()
	}
	return &FlexPayZNSService{
		provider: provider,
		logger:   logger,
		enabled:  enabledCheck,
	}
}

// SendSalaryNotification sends ZNS to a single employee.
// Returns nil if the send succeeds or if ZNS is disabled.
// Returns error for transport/credential failures (business errors are logged only).
func (s *FlexPayZNSService) SendSalaryNotification(ctx context.Context, data FlexPayZNSData) error {
	// Check feature flag
	if s.enabled != nil {
		enabled, err := s.enabled(ctx)
		if err != nil {
			s.logger.Warn("zns: failed to check enabled flag, assuming disabled", "error", err)
			return nil
		}
		if !enabled {
			return nil // Silently skip if disabled
		}
	}

	if s.provider == nil {
		return fmt.Errorf("zns: provider not configured")
	}

	if data.Mobile == "" {
		s.logger.Warn("zns: skipping employee with no mobile number", "employee", data.EmployeeName)
		return nil
	}

	// Normalize phone number using existing utility
	phone := zalo.NormalizePhone(data.Mobile)
	if phone == "" {
		s.logger.Warn("zns: invalid phone number, skipping",
			"employee", data.EmployeeName,
			"mobile", data.Mobile)
		return nil
	}

	// Format expiry date as DD/MM/YYYY (Zalo template requirement)
	expiryDate := data.ExpiryDate.Format("02/01/2006")

	// Build template data
	templateData := map[string]string{
		"customer_name": truncateString(data.EmployeeName, 30),
		"max_amount":    formatAmount(data.Amount),
		"expiry_date":   expiryDate,
	}

	// Generate tracking ID for debugging
	trackingID := fmt.Sprintf("flexpay-%d-%s", time.Now().Unix(), phone)

	// Send ZNS via provider
	result, err := s.provider.Send(ctx, phone, FlexPaySalaryTemplateID, trackingID, templateData)
	if err != nil {
		// Transport/credential failures - log and return error
		s.logger.Error("zns: transport error sending notification",
			"phone", phone,
			"employee", data.EmployeeName,
			"error", err)
		return err
	}

	// Business errors from Zalo (logged but don't fail the batch)
	// ErrorCode 0 means success, non-zero means Zalo business error
	if result.ErrorCode != 0 {
		s.logger.Warn("zns: business error from Zalo",
			"phone", phone,
			"employee", data.EmployeeName,
			"error_code", result.ErrorCode,
			"error_message", result.ErrorMsg)
		return nil // Don't fail the batch for business errors
	}

	s.logger.Info("zns: sent successfully",
		"phone", phone,
		"employee", data.EmployeeName,
		"amount", data.Amount,
		"tracking_id", trackingID)

	return nil
}

// SendBatch sends notifications to multiple employees concurrently.
// Uses fire-and-forget pattern - errors are logged but not returned.
func (s *FlexPayZNSService) SendBatch(ctx context.Context, notifications []FlexPayZNSData) {
	if len(notifications) == 0 {
		return
	}

	s.logger.Info("zns: starting batch send", "count", len(notifications))

	for _, notification := range notifications {
		// Launch goroutine for each employee (fire-and-forget)
		go func(data FlexPayZNSData) {
			// Use background context since the worker may have moved on
			bgCtx := context.Background()
			if err := s.SendSalaryNotification(bgCtx, data); err != nil {
				// Error already logged in SendSalaryNotification
				return
			}
		}(notification)
	}
}

// formatAmount formats amount in VND without separators (Zalo expects raw number).
func formatAmount(amount int64) string {
	return fmt.Sprintf("%d", amount)
}

// truncateString truncates a string to max length, adding "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// For UTF-8, truncate by rune count not bytes
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}
