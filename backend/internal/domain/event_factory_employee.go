package domain

import (
	"context"
	"fmt"
)

// NewEmployeeCreatedEvent creates an EmployeeCreatedEvent
func NewEmployeeCreatedEvent(ctx context.Context, employee *Employee, actorUserID uint, actorFullName string) EmployeeCreatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeEmployee,
		actorFullName,
		employee.Fullname,
	)

	return EmployeeCreatedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "EmployeeCreated", employee.ID, actorUserID, AuditActionCreate, EntityTypeEmployee, auditMessage),
		Fullname:  employee.Fullname,
		CCCD:      employee.CCCD,
	}
}

// NewEmployeeProjectAssignmentsRemovedEvent records the non-destructive delete
// outcome used when financial history requires the employee record to remain.
func NewEmployeeProjectAssignmentsRemovedEvent(ctx context.Context, employee *Employee, actorUserID uint, actorFullName string) EmployeeProjectAssignmentsRemovedEvent {
	auditMessage := fmt.Sprintf("%s đã gỡ nhân viên %s khỏi tất cả dự án để bảo toàn lịch sử tài chính", actorFullName, employee.Fullname)
	return EmployeeProjectAssignmentsRemovedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "EmployeeProjectAssignmentsRemoved", employee.ID, actorUserID, AuditActionUpdate, EntityTypeEmployee, auditMessage),
		Fullname:  employee.Fullname,
		CCCD:      employee.CCCD,
	}
}

// NewEmployeeUpdatedEvent creates an EmployeeUpdatedEvent
func NewEmployeeUpdatedEvent(ctx context.Context, employee *Employee, actorUserID uint, actorFullName string, original *Employee) EmployeeUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeEmployee,
		actorFullName,
		employee.Fullname,
	)

	return EmployeeUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "EmployeeUpdated", employee.ID, actorUserID, AuditActionUpdate, EntityTypeEmployee, auditMessage),
		Fullname:      employee.Fullname,
		CCCD:          employee.CCCD,
		ChangedFields: CompareEmployees(original, employee),
	}
}

// NewEmployeeDeletedEvent creates an EmployeeDeletedEvent
func NewEmployeeDeletedEvent(ctx context.Context, employeeID uint, fullname, cccd string, actorUserID uint, actorFullName string) EmployeeDeletedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeEmployee,
		actorFullName,
		fullname,
	)

	return EmployeeDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "EmployeeDeleted", employeeID, actorUserID, AuditActionDelete, EntityTypeEmployee, auditMessage),
		Fullname:  fullname,
		CCCD:      cccd,
	}
}

// NewEmployeeProfileUpdatedEvent creates an EmployeeProfileUpdatedEvent
func NewEmployeeProfileUpdatedEvent(ctx context.Context, employee *Employee, actorUserID uint, actorFullName string, original *Employee) EmployeeProfileUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeEmployee,
		actorFullName,
		employee.Fullname,
	)

	return EmployeeProfileUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "EmployeeProfileUpdated", employee.ID, actorUserID, AuditActionUpdate, EntityTypeEmployee, auditMessage),
		Fullname:      employee.Fullname,
		ChangedFields: CompareEmployees(original, employee),
	}
}

// NewEmployeeNameUpdatedEvent creates an EmployeeNameUpdatedEvent
func NewEmployeeNameUpdatedEvent(ctx context.Context, employeeID uint, oldFullName, newFullName, cccd string, actorUserID uint, actorFullName string) EmployeeNameUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeEmployee,
		actorFullName,
		newFullName,
	)

	return EmployeeNameUpdatedEvent{
		BaseEvent:   newBaseEventWithActor(ctx, "EmployeeNameUpdated", employeeID, actorUserID, AuditActionUpdate, EntityTypeEmployee, auditMessage),
		OldFullName: oldFullName,
		NewFullName: newFullName,
		CCCD:        cccd,
	}
}

// NewEmployeeAccessGrantedEvent creates an EmployeeAccessGrantedEvent
func NewEmployeeAccessGrantedEvent(ctx context.Context, employeeID uint, employeeName, projectName string, projectID uint, role string, grantedBy uint, actorFullName string) EmployeeAccessGrantedEvent {
	// Map role to Vietnamese for audit message
	vietnameseRole := MapRoleToVietnamese(role)

	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeEmployeeUser,
		actorFullName,
		employeeName,
		projectName,
		vietnameseRole,
	)

	return EmployeeAccessGrantedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "EmployeeAccessGranted", employeeID, grantedBy, AuditActionCreate, EntityTypeEmployeeUser, auditMessage),
		EmployeeID:   employeeID,
		EmployeeName: employeeName,
		ProjectID:    projectID,
		ProjectName:  projectName,
		Role:         role,
		GrantedBy:    grantedBy,
	}
}

// NewEmployeeAccessRevokedEvent creates an EmployeeAccessRevokedEvent
func NewEmployeeAccessRevokedEvent(ctx context.Context, employeeID uint, employeeName, projectName string, projectID uint, revokedBy uint, actorFullName string) EmployeeAccessRevokedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeEmployeeUser,
		actorFullName,
		employeeName,
		projectName,
	)

	return EmployeeAccessRevokedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "EmployeeAccessRevoked", employeeID, revokedBy, AuditActionDelete, EntityTypeEmployeeUser, auditMessage),
		EmployeeID:   employeeID,
		EmployeeName: employeeName,
		ProjectID:    projectID,
		ProjectName:  projectName,
		RevokedBy:    revokedBy,
	}
}
