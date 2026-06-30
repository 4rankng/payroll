package dashboard

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	// Test that NewService creates a service with all fields populated
	// We test with nil values since we're just testing the constructor
	service := NewService(
		nil, // UserRepo
		nil, // ProjectRepo
		nil, // EmployeeRepo
		nil, // TimesheetRepo
		nil, // TimesheetQueryRepo
		nil, // TimesheetAnalyticsRepo
		nil, // LedgerRepo
		nil, // NotificationRepo
		nil, // AuditLogRepo
		nil, // ProjectEmployeeRepo
		nil, // BulkTransferHistoryRepo
		nil, // AdvancePaymentRequestRepo
		nil, // AdvancePaymentRepo
		nil, // AttendanceRepo
		nil, // AttendanceFailedAttemptRepo
		nil, // SettingsConfigSvc
		nil, // CacheService
		nil, // SalaryCalculationSvc
		slog.Default(),
	)

	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
}
