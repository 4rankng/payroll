# Phase 1: Create ZNS Service

**Status:** Completed  
**Dependencies:** None

## Context

Extract ZNS sending logic into a dedicated service that can be reused across the application. The existing `zaloconnect.Service` has `TestSend` but we need a production-ready method for employee notifications.

## Requirements

Create a new service method that:
- Accepts employee data (name, mobile, amount, expiry date)
- Normalizes phone numbers to `84xxxxxxxxx` format
- Sends ZNS using template ID 619686
- Handles errors gracefully (log, don't fail)
- Returns send result for each employee

## Files to Create/Modify

### New File: `backend/internal/app/services/zaloconnect/flexpay_zns_service.go`

```go
package zaloconnect

import (
	"context"
	"fmt"
	"time"

	"github.com/tingtingsoft/payroll/backend/internal/infra/zalo"
)

const (
	// FlexPaySalaryTemplateID is the ZNS template for salary/payment notifications
	FlexPaySalaryTemplateID = "619686" // SalaryNotification-v1
)

// FlexPayZNSData holds the data for ZNS notification
type FlexPayZNSData struct {
	EmployeeName  string    // customer_name
	Amount        int64     // max_amount (in VND)
	ExpiryDate    time.Time // expiry_date
	Mobile        string    // Phone number
}

// FlexPayZNSService sends ZNS notifications for FlexPay salary notifications
type FlexPayZNSService struct {
	provider *zalo.Provider
	logger   Logger
}

// NewFlexPayZNSService creates a new FlexPay ZNS service
func NewFlexPayZNSService(provider *zalo.Provider, logger Logger) *FlexPayZNSService {
	return &FlexPayZNSService{
		provider: provider,
		logger:   logger,
	}
}

// SendSalaryNotification sends ZNS to a single employee
func (s *FlexPayZNSService) SendSalaryNotification(ctx context.Context, data FlexPayZNSData) error {
	if data.Mobile == "" {
		return fmt.Errorf("mobile number is required")
	}

	// Normalize phone number
	phone := zalo.NormalizePhone(data.Mobile)

	// Format expiry date as DD/MM/YYYY
	expiryDate := data.ExpiryDate.Format("02/01/2006")

	// Build template data
	templateData := map[string]string{
		"customer_name": data.EmployeeName,
		"max_amount":    formatAmount(data.Amount),
		"expiry_date":   expiryDate,
	}

	// Generate tracking ID
	trackingID := fmt.Sprintf("flexpay-%d-%s", time.Now().Unix(), phone)

	// Send ZNS
	result, err := s.provider.Send(ctx, phone, FlexPaySalaryTemplateID, trackingID, templateData)
	if err != nil {
		s.logger.Error("failed to send ZNS", 
			"phone", phone, 
			"employee", data.EmployeeName,
			"error", err)
		return err
	}

	if !result.Success {
		s.logger.Warn("ZNS send failed",
			"phone", phone,
			"employee", data.EmployeeName,
			"error_code", result.ErrorCode,
			"error_message", result.ErrorMessage)
		return fmt.Errorf("ZNS send failed: %s", result.ErrorMessage)
	}

	s.logger.Info("ZNS sent successfully",
		"phone", phone,
		"employee", data.EmployeeName,
		"tracking_id", trackingID)

	return nil
}

// SendBatch sends notifications to multiple employees (fire-and-forget)
func (s *FlexPayZNSService) SendBatch(ctx context.Context, notifications []FlexPayZNSData) {
	for _, notification := range notifications {
		go func(data FlexPayZNSData) {
			if err := s.SendSalaryNotification(context.Background(), data); err != nil {
				// Error already logged in SendSalaryNotification
				return
			}
		}(notification)
	}
}

// formatAmount formats amount in VND (e.g., 1,000,000 for 1000000)
func formatAmount(amount int64) string {
	// Simple formatting without comma separators since template expects number
	return fmt.Sprintf("%d", amount)
}

// Logger interface for dependency injection
type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})
}
```

## Implementation Steps

1. Create `flexpay_zns_service.go` with above implementation
2. Add unit tests for phone normalization
3. Add unit tests for template data mapping
4. Verify import compatibility with existing code

## Tests

```go
// Test NormalizePhone
// Test formatAmount
// Test template data mapping
```

## Notes

- Uses existing `zalo.NormalizePhone()` from `backend/internal/infra/zalo/phone.go`
- Template ID constant for easy updating
- Fire-and-forget pattern for batch sends
- Logs all errors but doesn't fail the batch
