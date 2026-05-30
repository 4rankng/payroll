package domain

import (
	"context"
	"fmt"

	"api-server/internal/pkg/clock"
	auditctx "api-server/internal/pkg/context"
)

// NewProjectEmployeesSyncedEvent creates a ProjectEmployeesSyncedEvent
func NewProjectEmployeesSyncedEvent(ctx context.Context, employeeID uint, oldFullName, newFullName string, updatedCount int, actorUserID uint, actorFullName string) ProjectEmployeesSyncedEvent {
	auditMessage := fmt.Sprintf("%s đã đồng bộ tên nhân viên thành %s (%d phân công được cập nhật)", actorFullName, newFullName, updatedCount)

	return ProjectEmployeesSyncedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectEmployeesSynced", employeeID, actorUserID, AuditActionUpdate, EntityTypeProjectEmployee, auditMessage),
		EmployeeID:   employeeID,
		OldFullName:  oldFullName,
		NewFullName:  newFullName,
		UpdatedCount: updatedCount,
		SyncedAt:     clock.Now(),
	}
}

// NewProjectCreatedEvent creates a ProjectCreatedEvent
func NewProjectCreatedEvent(ctx context.Context, project *Project, actorUserID uint, actorFullName string) ProjectCreatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeProject,
		actorFullName,
		project.Name,
	)

	return ProjectCreatedEvent{
		BaseEvent:  newBaseEventWithActor(ctx, "ProjectCreated", project.ID, actorUserID, AuditActionCreate, EntityTypeProject, auditMessage),
		Name:       project.Name,
		Code:       project.Code,
		ClientName: project.ClientName,
	}
}

// NewProjectUpdatedEvent creates a ProjectUpdatedEvent
func NewProjectUpdatedEvent(ctx context.Context, project *Project, actorUserID uint, actorFullName string, original *Project) ProjectUpdatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionUpdate,
		EntityTypeProject,
		actorFullName,
		project.Name,
	)

	return ProjectUpdatedEvent{
		BaseEvent:     newBaseEventWithActor(ctx, "ProjectUpdated", project.ID, actorUserID, AuditActionUpdate, EntityTypeProject, auditMessage),
		Name:          project.Name,
		Code:          project.Code,
		ClientName:    project.ClientName,
		ChangedFields: CompareProjects(original, project),
	}
}

// NewProjectDeletedEvent creates a ProjectDeletedEvent
func NewProjectDeletedEvent(ctx context.Context, projectID uint, name, code string, actorUserID uint, actorFullName string) ProjectDeletedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeProject,
		actorFullName,
		name,
	)

	return ProjectDeletedEvent{
		BaseEvent: newBaseEventWithActor(ctx, "ProjectDeleted", projectID, actorUserID, AuditActionDelete, EntityTypeProject, auditMessage),
		Name:      name,
		Code:      code,
	}
}

// NewProjectEmployeeUpdatedEvent creates a ProjectEmployeeUpdatedEvent
func NewProjectEmployeeUpdatedEvent(ctx context.Context, assignment *ProjectEmployee) ProjectEmployeeUpdatedEvent {
	projectName := ""
	employeeName := ""
	status := "active"
	if assignment.Project.ID != 0 {
		projectName = assignment.Project.Name
	}
	if assignment.Employee.ID != 0 {
		employeeName = assignment.Employee.Fullname
	}
	if assignment.LastDate != nil {
		status = "terminated"
	}
	actorName := auditctx.GetFullName(ctx)
	auditMessage := BuildEventAuditMessage(AuditActionUpdate, EntityTypeProjectEmployee, actorName, employeeName, projectName)
	return ProjectEmployeeUpdatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectEmployeeUpdated", assignment.ID, getUserIDFromContext(ctx), AuditActionUpdate, EntityTypeProjectEmployee, auditMessage),
		ProjectID:    assignment.ProjectID,
		ProjectName:  projectName,
		EmployeeID:   assignment.EmployeeID,
		EmployeeName: employeeName,
		Status:       status,
	}
}

// NewProjectEmployeeCreatedEvent creates a ProjectEmployeeCreatedEvent
func NewProjectEmployeeCreatedEvent(ctx context.Context, assignment *ProjectEmployee, projectName, employeeName string, actorUserID uint, actorFullName string) ProjectEmployeeCreatedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeProjectEmployee,
		actorFullName,
		employeeName,
		projectName,
		assignment.Position,
	)

	return ProjectEmployeeCreatedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectEmployeeCreated", assignment.ID, actorUserID, AuditActionCreate, EntityTypeProjectEmployee, auditMessage),
		ProjectID:    assignment.ProjectID,
		ProjectName:  projectName,
		EmployeeID:   assignment.EmployeeID,
		EmployeeName: employeeName,
		Role:         assignment.Position,
	}
}

// NewProjectEmployeeDeletedEvent creates a ProjectEmployeeDeletedEvent
func NewProjectEmployeeDeletedEvent(ctx context.Context, assignmentID, projectID uint, projectName, employeeName, reason string, actorUserID uint, actorFullName string) ProjectEmployeeDeletedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeProjectEmployee,
		actorFullName,
		employeeName,
		projectName,
	)

	return ProjectEmployeeDeletedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectEmployeeDeleted", assignmentID, actorUserID, AuditActionDelete, EntityTypeProjectEmployee, auditMessage),
		ProjectID:    projectID,
		ProjectName:  projectName,
		EmployeeID:   0, // Not available in deletion context
		EmployeeName: employeeName,
		Reason:       reason,
	}
}

// NewProjectAccessGrantedEvent creates a ProjectAccessGrantedEvent for sharing projects with users
func NewProjectAccessGrantedEvent(ctx context.Context, projectID uint, projectName string, targetUserID uint, targetUserName, targetUserRole string, grantedBy uint, actorFullName string) ProjectAccessGrantedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionCreate,
		EntityTypeProjectUser,
		actorFullName,
		projectName,
		targetUserName,
	)

	return ProjectAccessGrantedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectAccessGranted", projectID, grantedBy, AuditActionCreate, EntityTypeProjectUser, auditMessage),
		ProjectID:    projectID,
		ProjectName:  projectName,
		EmployeeID:   targetUserID, // Note: Field name is EmployeeID but contains target user ID for project sharing
		EmployeeName: targetUserName,
		Role:         targetUserRole,
		GrantedBy:    grantedBy,
	}
}

// NewProjectAccessRevokedEvent creates a ProjectAccessRevokedEvent for removing project access from users
func NewProjectAccessRevokedEvent(ctx context.Context, projectID uint, projectName string, targetUserID uint, targetUserName string, revokedBy uint, actorFullName string) ProjectAccessRevokedEvent {
	auditMessage := BuildEventAuditMessage(
		AuditActionDelete,
		EntityTypeProjectUser,
		actorFullName,
		projectName,
		targetUserName,
	)

	return ProjectAccessRevokedEvent{
		BaseEvent:    newBaseEventWithActor(ctx, "ProjectAccessRevoked", projectID, revokedBy, AuditActionDelete, EntityTypeProjectUser, auditMessage),
		ProjectID:    projectID,
		ProjectName:  projectName,
		EmployeeID:   targetUserID, // Note: Field name is EmployeeID but contains target user ID for project sharing
		EmployeeName: targetUserName,
		RevokedBy:    revokedBy,
	}
}
