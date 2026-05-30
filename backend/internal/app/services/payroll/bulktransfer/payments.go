package bulktransfer

import (
	"context"
	"time"

	"api-server/internal/domain"
)

// buildPaymentUpdates centralizes construction of payment updates
func (s *Service) buildPaymentUpdates(ctx context.Context, timesheets []*domain.Timesheet, status domain.PaymentStatus, reference string, now time.Time) ([]domain.PaymentStatusUpdate, int) {
	var updates []domain.PaymentStatusUpdate
	// Use weekly payment percentage as default (timesheets have already been filtered by schedule during export)
	bulkTransferPaymentPercentage := s.settingsConfig.GetWeeklyPaymentPercentage(ctx)

	for _, ts := range timesheets {
		var paymentDate *time.Time
		var paidAmountPtr *int64
		if status == domain.PaymentStatusPaid {
			paymentDate = &now
			paidAmount := int64(float64(ts.Amount) * bulkTransferPaymentPercentage)
			paidAmountPtr = &paidAmount
		}
		ref := reference // capture per-iteration pointer safety
		updates = append(updates, domain.PaymentStatusUpdate{
			TimesheetID:      ts.ID,
			PaymentStatus:    status,
			PaymentReference: &ref,
			PaymentDate:      paymentDate,
			PaidAmount:       paidAmountPtr,
		})
	}
	return updates, len(updates)
}
