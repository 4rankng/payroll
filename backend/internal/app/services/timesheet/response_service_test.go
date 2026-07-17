package timesheet

import (
	"context"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestParseTimesheetFiltersPendingPaymentUsesPayrollEligibleCohort(t *testing.T) {
	service := NewTimesheetResponseService(nil)

	filters := service.ParseTimesheetFilters(context.Background(), map[string]interface{}{
		"status": []string{"pending_payment"},
	})

	assert.Equal(t, []domain.TimesheetStatus{domain.TimesheetStatusApproved}, filters.TimesheetStatus)
	assert.Equal(t, []domain.PaymentStatus{
		domain.PaymentStatusPending,
		domain.PaymentStatusFailed,
	}, filters.PaymentStatus)
}

func TestParseTimesheetFiltersPendingPaymentCannotBeBroadenedByOtherStatuses(t *testing.T) {
	service := NewTimesheetResponseService(nil)

	filters := service.ParseTimesheetFilters(context.Background(), map[string]interface{}{
		"status": []string{"pending_payment", "approved", "paid"},
	})

	assert.Equal(t, []domain.TimesheetStatus{domain.TimesheetStatusApproved}, filters.TimesheetStatus)
	assert.Equal(t, []domain.PaymentStatus{
		domain.PaymentStatusPending,
		domain.PaymentStatusFailed,
	}, filters.PaymentStatus)
}
