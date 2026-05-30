package domain

import (
	"context"

	auditctx "api-server/internal/pkg/context"
)

// NewTimesheetApprovedEvent creates a TimesheetApprovedEvent
func NewTimesheetApprovedEvent(ctx context.Context, timesheetID, employeeID, projectID, approverID uint, employeeName, projectName string) TimesheetApprovedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionApprove, EntityTypeTimesheet, actorName, employeeName, projectName)
	return TimesheetApprovedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TimesheetApproved", timesheetID, approverID, AuditActionApprove, EntityTypeTimesheet, auditMessage),
		EmployeeID:   employeeID,
		EmployeeName: employeeName,
		ProjectID:    projectID,
		ProjectName:  projectName,
		ApproverID:   approverID,
	}
}

// NewTimesheetRejectedEvent creates a TimesheetRejectedEvent
func NewTimesheetRejectedEvent(ctx context.Context, timesheetID, employeeID, projectID, approverID uint, employeeName, projectName string) TimesheetRejectedEvent {
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionReject, EntityTypeTimesheet, actorName, employeeName, projectName)
	return TimesheetRejectedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TimesheetRejected", timesheetID, approverID, AuditActionReject, EntityTypeTimesheet, auditMessage),
		EmployeeID:   employeeID,
		EmployeeName: employeeName,
		ProjectID:    projectID,
		ProjectName:  projectName,
		ApproverID:   approverID,
	}
}

// NewTimesheetBulkApprovedEvent creates a TimesheetBulkApprovedEvent
func NewTimesheetBulkApprovedEvent(ctx context.Context, count int, projectID *uint, approverID uint) TimesheetBulkApprovedEvent {
	auditMessage := BuildBulkAuditMessage(AuditActionBulkApprove, EntityTypeTimesheet, count)
	return TimesheetBulkApprovedEvent{
		BaseEvent:  newBaseEventWithActor(ctx, "TimesheetBulkApproved", 0, approverID, AuditActionBulkApprove, EntityTypeTimesheet, auditMessage),
		Count:      count,
		ProjectID:  projectID,
		ApproverID: approverID,
	}
}

// NewTimesheetBulkRejectedEvent creates a TimesheetBulkRejectedEvent
func NewTimesheetBulkRejectedEvent(ctx context.Context, count int, projectID *uint, rejectedBy uint) TimesheetBulkRejectedEvent {
	auditMessage := BuildBulkAuditMessage(AuditActionBulkReject, EntityTypeTimesheet, count)
	return TimesheetBulkRejectedEvent{
		BaseEvent:  newBaseEventWithActor(ctx, "TimesheetBulkRejected", 0, rejectedBy, AuditActionBulkReject, EntityTypeTimesheet, auditMessage),
		Count:      count,
		ProjectID:  projectID,
		ApproverID: rejectedBy,
	}
}

// NewTimesheetBulkResetEvent creates a TimesheetBulkResetEvent
func NewTimesheetBulkResetEvent(ctx context.Context, count int, projectID *uint, resetBy uint) TimesheetBulkResetEvent {
	auditMessage := BuildBulkAuditMessage(AuditActionBulkReset, EntityTypeTimesheet, count)
	return TimesheetBulkResetEvent{
		BaseEvent: newBaseEventWithActor(ctx, "TimesheetBulkReset", 0, resetBy, AuditActionBulkReset, EntityTypeTimesheet, auditMessage),
		Count:     count,
		ProjectID: projectID,
		ResetBy:   resetBy,
	}
}

// NewTimesheetBulkCreatedEvent creates a TimesheetBulkCreatedEvent
func NewTimesheetBulkCreatedEvent(ctx context.Context, count int, projectID *uint, createdBy uint) TimesheetBulkCreatedEvent {
	auditMessage := BuildBulkAuditMessage(AuditActionBulkCreate, EntityTypeTimesheet, count)
	return TimesheetBulkCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "TimesheetBulkCreated", 0, createdBy, AuditActionBulkCreate, EntityTypeTimesheet, auditMessage),
		Count:     count,
		ProjectID: projectID,
		CreatedBy: createdBy,
	}
}

// NewTimesheetCreatedEvent creates a TimesheetCreatedEvent
func NewTimesheetCreatedEvent(ctx context.Context, timesheet *Timesheet) TimesheetCreatedEvent {
	var employeeName, projectName string
	if timesheet.Employee != nil && timesheet.Employee.ID != 0 {
		employeeName = timesheet.Employee.Fullname
	}
	if timesheet.Project != nil && timesheet.Project.ID != 0 {
		projectName = timesheet.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionCreate, EntityTypeTimesheet, actorName, employeeName, projectName)
	return TimesheetCreatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TimesheetCreated", timesheet.ID, getUserIDFromContext(ctx), AuditActionCreate, EntityTypeTimesheet, auditMessage),
		EmployeeID:   timesheet.EmployeeID,
		EmployeeName: employeeName,
		ProjectID:    timesheet.ProjectID,
		ProjectName:  projectName,
		Date:         timesheet.Date,
	}
}

// NewTimesheetUpdatedEvent creates a TimesheetUpdatedEvent
func NewTimesheetUpdatedEvent(ctx context.Context, timesheet *Timesheet, original *Timesheet) TimesheetUpdatedEvent {
	var employeeName, projectName string
	if timesheet.Employee != nil && timesheet.Employee.ID != 0 {
		employeeName = timesheet.Employee.Fullname
	}
	if timesheet.Project != nil && timesheet.Project.ID != 0 {
		projectName = timesheet.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeTimesheet, actorName, employeeName, projectName)
	return TimesheetUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "TimesheetUpdated", timesheet.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeTimesheet, auditMessage),
		EmployeeID:    timesheet.EmployeeID,
		EmployeeName:  employeeName,
		ProjectID:     timesheet.ProjectID,
		ProjectName:   projectName,
		ChangedFields: CompareTimesheets(original, timesheet),
	}
}

// NewTimesheetDeletedEvent creates a TimesheetDeletedEvent
func NewTimesheetDeletedEvent(ctx context.Context, timesheet *Timesheet) TimesheetDeletedEvent {
	var employeeName, projectName string
	if timesheet.Employee != nil && timesheet.Employee.ID != 0 {
		employeeName = timesheet.Employee.Fullname
	}
	if timesheet.Project != nil && timesheet.Project.ID != 0 {
		projectName = timesheet.Project.Name
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionDelete, EntityTypeTimesheet, actorName, employeeName, projectName)
	return TimesheetDeletedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "TimesheetDeleted", timesheet.ID, getUserIDFromContext(ctx), AuditActionDelete, EntityTypeTimesheet, auditMessage),
		EmployeeID:   timesheet.EmployeeID,
		EmployeeName: employeeName,
		ProjectID:    timesheet.ProjectID,
		ProjectName:  projectName,
	}
}

// NewTimesheetEditRequestCreatedEvent creates a TimesheetEditRequestCreatedEvent
func NewTimesheetEditRequestCreatedEvent(ctx context.Context, timesheetID, requestedBy uint, reason string) TimesheetEditRequestCreatedEvent {
	return TimesheetEditRequestCreatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "TimesheetEditRequestCreated", timesheetID, requestedBy, AuditActionCreate, EntityTypeTimesheet, ""),
		TimesheetID: timesheetID,
		RequestedBy: requestedBy,
		Reason:      reason,
	}
}

// NewTimesheetEditRequestUpdatedEvent creates a TimesheetEditRequestUpdatedEvent
func NewTimesheetEditRequestUpdatedEvent(ctx context.Context, requestID, timesheetID, requestedBy uint, newStatus, reason string) TimesheetEditRequestUpdatedEvent {
	return TimesheetEditRequestUpdatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "TimesheetEditRequestUpdated", requestID, requestedBy, AuditActionUpdate, EntityTypeTimesheet, ""),
		RequestID:   requestID,
		TimesheetID: timesheetID,
		RequestedBy: requestedBy,
		NewStatus:   newStatus,
		Reason:      reason,
	}
}

// NewTimesheetMarkingEvent creates a TimesheetMarkingEvent
func NewTimesheetMarkingEvent(
	ctx context.Context,
	timesheetIDs []uint,
	markedAs string,
	source string,
	sourceAssetID *uint,
) TimesheetMarkingEvent {
	return TimesheetMarkingEvent{
		BaseEvent:     newBaseEvent(ctx, "TimesheetMarking", 0, AuditActionUpdate, EntityTypeTimesheet), // AggregateID is 0 for multiple timesheets
		TimesheetIDs:  timesheetIDs,
		MarkedAs:      markedAs,
		Source:        source,
		SourceAssetID: sourceAssetID,
	}
}

// NewTimesheetsRevenuePaidFromInternalEvent creates a TimesheetsRevenuePaidFromInternalEvent
func NewTimesheetsRevenuePaidFromInternalEvent(
	ctx context.Context,
	timesheetIDs []uint,
	fileID uint,
	filename string,
) TimesheetsRevenuePaidFromInternalEvent {
	return TimesheetsRevenuePaidFromInternalEvent{
		BaseEvent:    newBaseEvent(ctx, "TimesheetsRevenuePaidFromInternal", 0, AuditActionUpdate, EntityTypeTimesheet),
		TimesheetIDs: timesheetIDs,
		FileID:       fileID,
		Filename:     filename,
	}
}

// NewTimesheetBulkExternallyPaidEvent creates a TimesheetBulkExternallyPaidEvent
func NewTimesheetBulkExternallyPaidEvent(ctx context.Context, actorID uint, count int, reference string) TimesheetBulkExternallyPaidEvent {
	return TimesheetBulkExternallyPaidEvent{
		BaseEvent: newBaseEventWithActor(ctx, "TimesheetBulkExternallyPaid", 0, actorID, AuditActionExternalPay, EntityTypeTimesheet,
			BuildBulkAuditMessage(AuditActionExternalPay, EntityTypeTimesheet, count)),
		Count:     count,
		Reference: reference,
	}
}
