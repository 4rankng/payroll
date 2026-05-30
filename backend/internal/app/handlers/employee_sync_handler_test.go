package handlers

import (
	"context"
	"testing"

	"api-server/internal/app/services/employee"
	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockEmployeeSyncService is a mock for EmployeeSyncService
type MockEmployeeSyncService struct {
	mock.Mock
}

func (m *MockEmployeeSyncService) SyncEmployeeNameToActiveAssignments(ctx context.Context, employeeID uint, newFullName string) (*employee.SyncResult, error) {
	args := m.Called(ctx, employeeID, newFullName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*employee.SyncResult), args.Error(1)
}

func (m *MockEmployeeSyncService) SyncAllInconsistentData(ctx context.Context) (*employee.BulkSyncResult, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*employee.BulkSyncResult), args.Error(1)
}

func (m *MockEmployeeSyncService) ValidateEmployeeNameConsistency(ctx context.Context, employeeID uint) (*employee.ConsistencyReport, error) {
	args := m.Called(ctx, employeeID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*employee.ConsistencyReport), args.Error(1)
}

func TestEmployeeSyncHandler_Handle_EmployeeNameUpdatedEvent(t *testing.T) {
	ctx := context.Background()
	employeeID := uint(1)
	oldName := "Nguyen Van A"
	newName := "Nguyen Van B"
	cccd := "123456789012"

	// Create mock service
	mockSyncService := &MockEmployeeSyncService{}

	// Create handler
	handler := NewEmployeeSyncHandler(mockSyncService)

	// Create test event
	event := domain.EmployeeNameUpdatedEvent{
		BaseEvent: domain.BaseEvent{
			EventName:    "EmployeeNameUpdated",
			Timestamp:    domain.BaseEvent{}.OccurredAt(),
			EntityID:     employeeID,
			ActorUserID:  1,
			AuditMessage: "Updated employee name",
		},
		OldFullName: oldName,
		NewFullName: newName,
		CCCD:        cccd,
	}

	// Setup expected sync result
	expectedResult := &employee.SyncResult{
		EmployeeID:   employeeID,
		OldFullName:  oldName,
		NewFullName:  newName,
		UpdatedCount: 2,
		SyncedAt:     domain.BaseEvent{}.OccurredAt(),
	}

	// Setup mock expectation
	mockSyncService.On("SyncEmployeeNameToActiveAssignments", ctx, employeeID, newName).Return(expectedResult, nil)

	// Execute test
	err := handler.Handle(ctx, event)

	// Assertions
	assert.NoError(t, err)
	mockSyncService.AssertExpectations(t)
}

func TestEmployeeSyncHandler_Handle_UnsupportedEventType(t *testing.T) {
	ctx := context.Background()

	// Create mock service
	mockSyncService := &MockEmployeeSyncService{}

	// Create handler
	handler := NewEmployeeSyncHandler(mockSyncService)

	// Create an unsupported event
	unsupportedEvent := domain.EmployeeCreatedEvent{
		BaseEvent: domain.BaseEvent{
			EventName:    "EmployeeCreated",
			Timestamp:    domain.BaseEvent{}.OccurredAt(),
			EntityID:     1,
			ActorUserID:  1,
			AuditMessage: "Created employee",
		},
		Fullname: "Test Employee",
		CCCD:     "123456789012",
	}

	// Execute test
	err := handler.Handle(ctx, unsupportedEvent)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported event type")
	mockSyncService.AssertNotCalled(t, "SyncEmployeeNameToActiveAssignments")
}

func TestEmployeeSyncHandler_CanHandle(t *testing.T) {
	// Create mock service
	mockSyncService := &MockEmployeeSyncService{}

	// Create handler
	handler := NewEmployeeSyncHandler(mockSyncService)

	// Test supported event type
	assert.True(t, handler.CanHandle("EmployeeNameUpdated"))

	// Test unsupported event types
	assert.False(t, handler.CanHandle("EmployeeCreated"))
	assert.False(t, handler.CanHandle("EmployeeUpdated"))
	assert.False(t, handler.CanHandle("ProjectEmployeeUpdated"))
	assert.False(t, handler.CanHandle(""))
}

func TestEmployeeSyncHandler_Handle_SyncError(t *testing.T) {
	ctx := context.Background()
	employeeID := uint(1)
	newName := "Nguyen Van B"

	// Create mock service
	mockSyncService := &MockEmployeeSyncService{}

	// Create handler
	handler := NewEmployeeSyncHandler(mockSyncService)

	// Create test event
	event := domain.EmployeeNameUpdatedEvent{
		BaseEvent: domain.BaseEvent{
			EventName:    "EmployeeNameUpdated",
			Timestamp:    domain.BaseEvent{}.OccurredAt(),
			EntityID:     employeeID,
			ActorUserID:  1,
			AuditMessage: "Updated employee name",
		},
		OldFullName: "Nguyen Van A",
		NewFullName: newName,
		CCCD:        "123456789012",
	}

	// Setup mock expectation to return an error
	expectedError := assert.AnError
	mockSyncService.On("SyncEmployeeNameToActiveAssignments", ctx, employeeID, newName).Return(nil, expectedError)

	// Execute test
	err := handler.Handle(ctx, event)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync employee name")
	assert.Contains(t, err.Error(), expectedError.Error())
	mockSyncService.AssertExpectations(t)
}
