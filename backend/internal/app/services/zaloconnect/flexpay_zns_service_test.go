package zaloconnect

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"api-server/internal/infra/zalo"
)

// MockZaloProvider is a mock provider for testing
type MockZaloProvider struct {
	mu           sync.Mutex
	SendFunc     func(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (zalo.SendResult, error)
	RefreshCount int
	sendCount    int
}

func (m *MockZaloProvider) Send(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (zalo.SendResult, error) {
	m.mu.Lock()
	m.sendCount++
	m.mu.Unlock()
	if m.SendFunc != nil {
		return m.SendFunc(ctx, phone, templateID, trackingID, data)
	}
	return zalo.SendResult{ErrorCode: 0, ErrorMsg: ""}, nil
}

func (m *MockZaloProvider) SendCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sendCount
}

func (m *MockZaloProvider) RefreshNow(ctx context.Context) error {
	m.RefreshCount++
	return nil
}

// mockEnabledCheck returns enabled=true for testing
func mockEnabledCheck(ctx context.Context) (bool, error) {
	return true, nil
}

// TestNewFlexPayZNSService verifies service construction
func TestNewFlexPayZNSService(t *testing.T) {
	provider := &MockZaloProvider{}
	logger := slog.Default()

	service := NewFlexPayZNSService(provider, logger, mockEnabledCheck)

	if service == nil {
		t.Fatal("expected service to be non-nil")
	}
	if service.logger == nil {
		t.Error("logger not set correctly")
	}
}

// TestSendSalaryNotification_Success verifies successful ZNS send
func TestSendSalaryNotification_Success(t *testing.T) {
	provider := &MockZaloProvider{
		SendFunc: func(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (zalo.SendResult, error) {
			if templateID != FlexPaySalaryTemplateID {
				t.Errorf("template_id = %q, want %q", templateID, FlexPaySalaryTemplateID)
			}
			return zalo.SendResult{ErrorCode: 0, ErrorMsg: "", MsgID: "test-msg-id"}, nil
		},
	}

	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "0366178061",
	}

	err := service.SendSalaryNotification(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if provider.SendCount() != 1 {
		t.Errorf("expected 1 send, got %d", provider.SendCount())
	}
}

// TestSendSalaryNotification_NoMobile verifies skip when no mobile number
func TestSendSalaryNotification_NoMobile(t *testing.T) {
	provider := &MockZaloProvider{}
	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "",
	}

	err := service.SendSalaryNotification(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error for missing mobile, got %v", err)
	}

	if provider.SendCount() != 0 {
		t.Errorf("expected 0 sends for missing mobile, got %d", provider.SendCount())
	}
}

// TestSendSalaryNotification_InvalidPhone verifies skip for invalid phone
func TestSendSalaryNotification_InvalidPhone(t *testing.T) {
	provider := &MockZaloProvider{}
	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "123", // Invalid phone
	}

	err := service.SendSalaryNotification(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error for invalid phone, got %v", err)
	}

	if provider.SendCount() != 0 {
		t.Errorf("expected 0 sends for invalid phone, got %d", provider.SendCount())
	}
}

// TestSendSalaryNotification_Disabled verifies skip when feature disabled
func TestSendSalaryNotification_Disabled(t *testing.T) {
	provider := &MockZaloProvider{}
	disabledCheck := func(ctx context.Context) (bool, error) {
		return false, nil
	}
	service := NewFlexPayZNSService(provider, slog.Default(), disabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "0366178061",
	}

	err := service.SendSalaryNotification(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error when disabled, got %v", err)
	}

	if provider.SendCount() != 0 {
		t.Errorf("expected 0 sends when disabled, got %d", provider.SendCount())
	}
}

// TestSendSalaryNotification_BusinessError verifies business errors don't fail the send
func TestSendSalaryNotification_BusinessError(t *testing.T) {
	provider := &MockZaloProvider{
		SendFunc: func(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (zalo.SendResult, error) {
			// Simulate Zalo business error (e.g., no Zalo account)
			return zalo.SendResult{ErrorCode: -118, ErrorMsg: "Phone has no linked Zalo account"}, nil
		},
	}

	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "0366178061",
	}

	err := service.SendSalaryNotification(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error for business error, got %v", err)
	}

	if provider.SendCount() != 1 {
		t.Errorf("expected 1 send attempt, got %d", provider.SendCount())
	}
}

// TestSendBatch verifies batch sending
func TestSendBatch(t *testing.T) {
	provider := &MockZaloProvider{}
	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	notifications := []FlexPayZNSData{
		{
			EmployeeName: "Nguyễn Việt Duy",
			Amount:       1680000,
			ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			Mobile:       "0366178061",
		},
		{
			EmployeeName: "Bùi Thị Minh Phương",
			Amount:       1440000,
			ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
			Mobile:       "0764283662",
		},
	}

	service.SendBatch(context.Background(), notifications)

	// Should have sent both
	if provider.SendCount() != 2 {
		t.Errorf("expected 2 sends, got %d", provider.SendCount())
	}
}

// TestSendBatch_Empty verifies empty batch doesn't panic
func TestSendBatch_Empty(t *testing.T) {
	provider := &MockZaloProvider{}
	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	// Should not panic
	service.SendBatch(context.Background(), []FlexPayZNSData{})

	if provider.SendCount() != 0 {
		t.Errorf("expected 0 sends for empty batch, got %d", provider.SendCount())
	}
}

// TestTemplateDataMapping verifies template data is correctly formatted
func TestTemplateDataMapping(t *testing.T) {
	provider := &MockZaloProvider{
		SendFunc: func(ctx context.Context, phone, templateID, trackingID string, data map[string]string) (zalo.SendResult, error) {
			// Verify template data
			if data["customer_name"] != "Nguyễn Việt Duy" {
				t.Errorf("expected customer_name 'Nguyễn Việt Duy', got '%s'", data["customer_name"])
			}
			if data["max_amount"] != "1680000" {
				t.Errorf("expected max_amount '1680000', got '%s'", data["max_amount"])
			}
			if data["expiry_date"] != "31/08/2026" {
				t.Errorf("expected expiry_date '31/08/2026', got '%s'", data["expiry_date"])
			}
			return zalo.SendResult{ErrorCode: 0, ErrorMsg: "", MsgID: "test-msg-id"}, nil
		},
	}

	service := NewFlexPayZNSService(provider, slog.Default(), mockEnabledCheck)

	data := FlexPayZNSData{
		EmployeeName: "Nguyễn Việt Duy",
		Amount:       1680000,
		ExpiryDate:   time.Date(2026, 8, 31, 0, 0, 0, 0, time.UTC),
		Mobile:       "0366178061",
	}

	_ = service.SendSalaryNotification(context.Background(), data)
}

// TestTruncateString verifies string truncation
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exact length", 12, "exact length"},
		{"this is way too long", 10, "this is..."},
		{"Nguyễn Văn A", 10, "Nguyễn ..."},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

// TestFormatAmount verifies amount formatting
func TestFormatAmount(t *testing.T) {
	tests := []struct {
		amount   int64
		expected string
	}{
		{1680000, "1680000"},
		{15000000, "15000000"},
		{100000, "100000"},
		{0, "0"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.amount), func(t *testing.T) {
			result := formatAmount(tt.amount)
			if result != tt.expected {
				t.Errorf("formatAmount(%d) = %q, want %q", tt.amount, result, tt.expected)
			}
		})
	}
}
