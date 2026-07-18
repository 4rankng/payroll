package events

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/domain"
	auditctx "api-server/internal/pkg/context"
)

func TestAuditEventHandler_PreservesMutationMetadata(t *testing.T) {
	ctx := auditctx.WithFullName(auditctx.WithUserID(context.Background(), 42), "Quản trị viên")

	tests := []struct {
		name     string
		event    domain.DomainEvent
		expected map[string]interface{}
	}{
		{
			name:  "advance payment fee schedule create",
			event: domain.NewAdvancePaymentFeeScheduleCreatedEvent(ctx, "schedule-1", "2026-08-01", "2%"),
			expected: map[string]interface{}{
				"schedule_id":    "schedule-1",
				"effective_date": "2026-08-01",
				"summary":        "2%",
			},
		},
		{
			name:  "disbursement fee schedule update",
			event: domain.NewDisbursementFeeScheduleUpdatedEvent(ctx, "schedule-2", "2026-09-01", "3.000 VND"),
			expected: map[string]interface{}{
				"schedule_id":    "schedule-2",
				"effective_date": "2026-09-01",
				"summary":        "3.000 VND",
			},
		},
		{
			name:  "external timesheet payment",
			event: domain.NewTimesheetBulkExternallyPaidEvent(ctx, 42, 3, "BANK-REF-123"),
			expected: map[string]interface{}{
				"count":     float64(3),
				"reference": "BANK-REF-123",
			},
		},
		{
			name:  "data import",
			event: domain.NewDataImportedEvent(ctx, "employee", 12, "employees.xlsx"),
			expected: map[string]interface{}{
				"data_type":    "employee",
				"record_count": float64(12),
				"file_name":    "employees.xlsx",
			},
		},
		{
			name:  "timesheet edit request",
			event: domain.NewTimesheetEditRequestCreatedEvent(ctx, 99, 42, "Điều chỉnh giờ công"),
			expected: map[string]interface{}{
				"timesheet_id": float64(99),
				"requested_by": float64(42),
				"reason":       "Điều chỉnh giờ công",
			},
		},
		{
			name:  "detached timesheet marking keeps actor",
			event: domain.NewTimesheetMarkingEvent(context.Background(), 42, []uint{1, 2}, "paid", "settlement", nil),
			expected: map[string]interface{}{
				"marked_as": "paid",
				"source":    "settlement",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enqueuer := &mockAuditEnqueuer{}
			handler := NewAuditEventHandler(enqueuer)

			if err := handler.Handle(ctx, tt.event); err != nil {
				t.Fatalf("Handle: %v", err)
			}
			if len(enqueuer.payloads) != 1 {
				t.Fatalf("expected one audit payload, got %d", len(enqueuer.payloads))
			}

			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(enqueuer.payloads[0].MetadataJSON), &metadata); err != nil {
				t.Fatalf("metadata unmarshal: %v", err)
			}
			for key, expected := range tt.expected {
				if actual, ok := metadata[key]; !ok || actual != expected {
					t.Errorf("metadata[%q] = %#v, want %#v", key, actual, expected)
				}
			}
		})
	}
}
