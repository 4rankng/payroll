package workers

import (
	"reflect"
	"testing"

	"api-server/internal/domain"
)

func TestPaymentFinalizationCandidatesExcludeRejectedAndPaidTimesheets(t *testing.T) {
	timesheets := []*domain.Timesheet{
		{ID: 1, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPending},
		{ID: 2, Status: domain.TimesheetStatusRejected, PaymentStatus: domain.PaymentStatusPending},
		{ID: 3, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusPaid},
		{ID: 4, Status: domain.TimesheetStatusApproved, PaymentStatus: domain.PaymentStatusFailed},
	}

	candidates := paymentFinalizationCandidates(timesheets)
	ids := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		ids = append(ids, candidate.ID)
	}
	if !reflect.DeepEqual(ids, []uint{1, 2, 4}) {
		t.Fatalf("candidate IDs = %v, want [1 2 4]", ids)
	}
}
