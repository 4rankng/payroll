package domain

import (
	"context"

	auditctx "api-server/internal/pkg/context"
)

// NewDisbursementFeeScheduleCreatedEvent emits the business-level audit event
// for creating a new disbursement fee schedule entry inside the JSON array.
// EntityID is 0 because the entry's stable identifier is a UUID string, which
// doesn't fit BIGINT — ScheduleID carries it.
func NewDisbursementFeeScheduleCreatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) DisbursementFeeScheduleCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeDisbursementFeeSchedule, actorName, effectiveDate, summary)
	return DisbursementFeeScheduleCreatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "DisbursementFeeScheduleCreated", 0, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeDisbursementFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

func NewDisbursementFeeScheduleUpdatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) DisbursementFeeScheduleUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeDisbursementFeeSchedule, actorName, effectiveDate, summary)
	return DisbursementFeeScheduleUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "DisbursementFeeScheduleUpdated", 0, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeDisbursementFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

func NewDisbursementFeeScheduleDeletedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) DisbursementFeeScheduleDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeDisbursementFeeSchedule, actorName, effectiveDate, summary)
	return DisbursementFeeScheduleDeletedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "DisbursementFeeScheduleDeleted", 0, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeDisbursementFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}
