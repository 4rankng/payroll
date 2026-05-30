package domain

import (
	"context"

	auditctx "api-server/internal/pkg/context"
)

// NewAdvancePaymentFeeScheduleCreatedEvent emits the business-level audit
// event for creating a new schedule entry inside the JSON array. EntityID is
// set to 0 because the entry's stable identifier is the ULID/UUID string,
// which doesn't fit BIGINT — the ScheduleID field carries it.
func NewAdvancePaymentFeeScheduleCreatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) AdvancePaymentFeeScheduleCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeAdvancePaymentFeeSchedule, actorName, effectiveDate, summary)
	return AdvancePaymentFeeScheduleCreatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "AdvancePaymentFeeScheduleCreated", 0, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeAdvancePaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

func NewAdvancePaymentFeeScheduleUpdatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) AdvancePaymentFeeScheduleUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeAdvancePaymentFeeSchedule, actorName, effectiveDate, summary)
	return AdvancePaymentFeeScheduleUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "AdvancePaymentFeeScheduleUpdated", 0, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeAdvancePaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

func NewAdvancePaymentFeeScheduleDeletedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) AdvancePaymentFeeScheduleDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeAdvancePaymentFeeSchedule, actorName, effectiveDate, summary)
	return AdvancePaymentFeeScheduleDeletedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "AdvancePaymentFeeScheduleDeleted", 0, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeAdvancePaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}
