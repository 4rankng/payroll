package domain

import (
	"context"

	auditctx "api-server/internal/pkg/context"
)

// NewWeeklyPaymentFeeScheduleCreatedEvent emits the business-level audit event
// for creating a new weekly-payment fee schedule entry inside the JSON array.
// EntityID is 0 because the entry identifier is a UUID string, which doesn't
// fit BIGINT — ScheduleID carries it.
func NewWeeklyPaymentFeeScheduleCreatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) WeeklyPaymentFeeScheduleCreatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeWeeklyPaymentFeeSchedule, actorName, effectiveDate, summary)
	return WeeklyPaymentFeeScheduleCreatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "WeeklyPaymentFeeScheduleCreated", 0, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeWeeklyPaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

// NewWeeklyPaymentFeeScheduleUpdatedEvent emits the update audit event for a
// future-dated weekly-payment fee schedule entry.
func NewWeeklyPaymentFeeScheduleUpdatedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) WeeklyPaymentFeeScheduleUpdatedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeWeeklyPaymentFeeSchedule, actorName, effectiveDate, summary)
	return WeeklyPaymentFeeScheduleUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "WeeklyPaymentFeeScheduleUpdated", 0, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeWeeklyPaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}

// NewWeeklyPaymentFeeScheduleDeletedEvent emits the delete audit event for a
// future-dated weekly-payment fee schedule entry.
func NewWeeklyPaymentFeeScheduleDeletedEvent(ctx context.Context, scheduleID, effectiveDate, summary string) WeeklyPaymentFeeScheduleDeletedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeWeeklyPaymentFeeSchedule, actorName, effectiveDate, summary)
	return WeeklyPaymentFeeScheduleDeletedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "WeeklyPaymentFeeScheduleDeleted", 0, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeWeeklyPaymentFeeSchedule, auditMessage),
		ScheduleID:    scheduleID,
		EffectiveDate: effectiveDate,
		Summary:       summary,
	}
}
