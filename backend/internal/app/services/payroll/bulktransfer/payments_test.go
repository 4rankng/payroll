package bulktransfer

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestService_buildPaymentUpdates_Paid(t *testing.T) {
	// Setup
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "REF-ABC-123"

	timesheets := []*domain.Timesheet{
		{
			ID:     1,
			Amount: 1000000,
		},
		{
			ID:     2,
			Amount: 1500000,
		},
		{
			ID:     3,
			Amount: 2000000,
		},
	}

	// Execute
	updates, count := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPaid, reference, now)

	// Assert
	assert.Equal(t, 3, count)
	assert.Len(t, updates, 3)

	for i, update := range updates {
		assert.Equal(t, timesheets[i].ID, update.TimesheetID)
		assert.Equal(t, domain.PaymentStatusPaid, update.PaymentStatus)
		assert.NotNil(t, update.PaymentReference)
		assert.Equal(t, reference, *update.PaymentReference)
		assert.NotNil(t, update.PaymentDate)
		assert.True(t, update.PaymentDate.Equal(now))
		assert.NotNil(t, update.PaidAmount)

		expectedAmount := int64(float64(timesheets[i].Amount) * settings.bulkPct)
		assert.Equal(t, expectedAmount, *update.PaidAmount)
	}
}

func TestService_buildPaymentUpdates_Pending(t *testing.T) {
	// Setup
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "REF-PENDING-456"

	timesheets := []*domain.Timesheet{
		{
			ID:     10,
			Amount: 500000,
		},
		{
			ID:     20,
			Amount: 750000,
		},
	}

	// Execute
	updates, count := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPending, reference, now)

	// Assert
	assert.Equal(t, 2, count)
	assert.Len(t, updates, 2)

	for i, update := range updates {
		assert.Equal(t, timesheets[i].ID, update.TimesheetID)
		assert.Equal(t, domain.PaymentStatusPending, update.PaymentStatus)
		assert.NotNil(t, update.PaymentReference)
		assert.Equal(t, reference, *update.PaymentReference)
		assert.Nil(t, update.PaymentDate)
		assert.Nil(t, update.PaidAmount)
	}
}

func TestService_buildPaymentUpdates_EmptyTimesheets(t *testing.T) {
	// Setup
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "REF-EMPTY"

	timesheets := []*domain.Timesheet{}

	// Execute
	updates, count := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPaid, reference, now)

	// Assert
	assert.Equal(t, 0, count)
	assert.Len(t, updates, 0)
}

func TestService_buildPaymentUpdates_DifferentPercentage(t *testing.T) {
	// Setup - test with different percentage
	settings := &fakeSettingsConfig{bulkPct: 0.5}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "REF-50PCT"

	timesheets := []*domain.Timesheet{
		{
			ID:     1,
			Amount: 1000000,
		},
	}

	// Execute
	updates, count := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPaid, reference, now)

	// Assert
	assert.Equal(t, 1, count)
	assert.Len(t, updates, 1)

	assert.NotNil(t, updates[0].PaidAmount)
	expectedAmount := int64(1000000 * 0.5) // 50% of amount
	assert.Equal(t, expectedAmount, *updates[0].PaidAmount)
}

func TestService_buildPaymentUpdates_ZeroAmount(t *testing.T) {
	// Setup
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "REF-ZERO"

	timesheets := []*domain.Timesheet{
		{
			ID:     1,
			Amount: 0,
		},
	}

	// Execute
	updates, count := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPaid, reference, now)

	// Assert
	assert.Equal(t, 1, count)
	assert.Len(t, updates, 1)
	assert.NotNil(t, updates[0].PaidAmount)
	assert.Equal(t, int64(0), *updates[0].PaidAmount)
}

func TestService_buildPaymentUpdates_ReferenceIsUnique(t *testing.T) {
	// Setup
	settings := &fakeSettingsConfig{bulkPct: 0.7}
	service := &Service{settingsConfig: settings}

	ctx := context.Background()
	now := time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC)
	reference := "UNIQUE-REF-789"

	timesheets := []*domain.Timesheet{
		{ID: 1, Amount: 1000},
		{ID: 2, Amount: 2000},
	}

	// Execute
	updates, _ := service.buildPaymentUpdates(ctx, timesheets, domain.PaymentStatusPaid, reference, now)

	// Assert - all updates should have the same reference
	for _, update := range updates {
		assert.NotNil(t, update.PaymentReference)
		assert.Equal(t, reference, *update.PaymentReference)
	}

	// Verify each update has its own reference pointer (not sharing)
	assert.NotSame(t, updates[0].PaymentReference, updates[1].PaymentReference)
}
