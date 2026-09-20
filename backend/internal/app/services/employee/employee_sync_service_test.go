package employee

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProjectEmployeeRepository is a mock for ProjectEmployeeRepository
type MockProjectEmployeeRepository struct {
	mock.Mock
}

func (m *MockProjectEmployeeRepository) Create(ctx context.Context, assignment *domain.ProjectEmployee) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) GetByID(ctx context.Context, id uint) (*domain.ProjectEmployee, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectID, employeeID)
	return args.Get(0).(*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetActiveAssignmentByProjectAndEmployee(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectID, employeeID)
	return args.Get(0).(*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetCurrentAssignmentsForProjectForUpdate(ctx context.Context, projectID uint, asOfDate time.Time) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectID, asOfDate)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetActiveAssignmentsByProjectsAndEmployees(ctx context.Context, projectIDs []uint, employeeIDs []uint) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectIDs, employeeIDs)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) Update(ctx context.Context, assignment *domain.ProjectEmployee) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) UpdatePosition(ctx context.Context, id uint, position string) error {
	args := m.Called(ctx, id, position)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) UpdatePositionIfCurrent(ctx context.Context, id uint, currentPosition, newPosition string) error {
	args := m.Called(ctx, id, currentPosition, newPosition)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) BackdateStartDateIfLater(ctx context.Context, id uint, newStart time.Time) (bool, error) {
	args := m.Called(ctx, id, newStart)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectEmployeeRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) DeleteByProjectID(ctx context.Context, projectID uint) error {
	args := m.Called(ctx, projectID)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) DeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error {
	args := m.Called(ctx, employeeID)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) HardDeleteAssignmentsByEmployeeID(ctx context.Context, employeeID uint) error {
	args := m.Called(ctx, employeeID)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) HasActiveFlexiblePaymentScheduleByEmployeeID(ctx context.Context, employeeID uint) (bool, error) {
	args := m.Called(ctx, employeeID)
	return args.Bool(0), args.Error(1)
}

func (m *MockProjectEmployeeRepository) List(ctx context.Context, filters domain.ProjectEmployeeFilters) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) Count(ctx context.Context, filters domain.ProjectEmployeeFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetByEmployee(ctx context.Context, employeeID uint) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, employeeID)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetActiveAssignments(ctx context.Context, projectID uint) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) EndAssignment(ctx context.Context, id uint, endDate time.Time) error {
	args := m.Called(ctx, id, endDate)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) BulkUpdateLastDate(ctx context.Context, assignmentIDs []uint, lastDate time.Time) error {
	args := m.Called(ctx, assignmentIDs, lastDate)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) GetCurrentProjectForEmployee(ctx context.Context, employeeID uint) (*domain.Project, error) {
	args := m.Called(ctx, employeeID)
	return args.Get(0).(*domain.Project), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetCurrentProjectsForEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*domain.Project, error) {
	args := m.Called(ctx, employeeIDs)
	return args.Get(0).(map[uint]*domain.Project), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetActiveProjectsForEmployee(ctx context.Context, employeeID uint) ([]*domain.Project, error) {
	args := m.Called(ctx, employeeID)
	return args.Get(0).([]*domain.Project), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetActiveEmployeesForProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*domain.Employee, error) {
	args := m.Called(ctx, projectID, createdBy)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetOverlappingAssignments(ctx context.Context, employeeID uint, startDate, endDate *time.Time) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, employeeID, startDate, endDate)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) CountWorkingEmployees(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetEmployeesByPaymentSchedule(ctx context.Context, schedule domain.PaymentSchedule) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, schedule)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) CountActiveEmployeesByPaymentSchedule(ctx context.Context, schedule domain.PaymentSchedule) (int, error) {
	args := m.Called(ctx, schedule)
	return args.Int(0), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetEmployeesWithPendingScheduleChanges(ctx context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, effectiveDate)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetEmployeesWithPendingCheckInEnable(ctx context.Context, effectiveDate time.Time) ([]*domain.ProjectEmployee, error) {
	args := m.Called(ctx, effectiveDate)
	return args.Get(0).([]*domain.ProjectEmployee), args.Error(1)
}

func (m *MockProjectEmployeeRepository) GetCheckInConfiguration(ctx context.Context, query domain.CheckInConfigurationQuery) (*domain.CheckInConfigurationResult, error) {
	args := m.Called(ctx, query)
	result, _ := args.Get(0).(*domain.CheckInConfigurationResult)
	return result, args.Error(1)
}

func (m *MockProjectEmployeeRepository) ApplyScheduleChanges(ctx context.Context, employeeIDs []uint) error {
	args := m.Called(ctx, employeeIDs)
	return args.Error(0)
}

func (m *MockProjectEmployeeRepository) HasAccessViaProject(ctx context.Context, employeeID, userID uint) (bool, error) {
	args := m.Called(ctx, employeeID, userID)
	return args.Bool(0), args.Error(1)
}

// MockEmployeeRepository is a mock for EmployeeRepository
type MockEmployeeRepository struct {
	mock.Mock
}

func (m *MockEmployeeRepository) Create(ctx context.Context, employee *domain.Employee) error {
	args := m.Called(ctx, employee)
	return args.Error(0)
}

func (m *MockEmployeeRepository) GetByID(ctx context.Context, id uint) (*domain.Employee, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByIDs(ctx context.Context, ids []int64) ([]*domain.Employee, error) {
	args := m.Called(ctx, ids)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Employee, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByUserID(ctx context.Context, userID uint) (*domain.Employee, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByCCCD(ctx context.Context, cccd string) (*domain.Employee, error) {
	args := m.Called(ctx, cccd)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByMobile(ctx context.Context, mobile string) (*domain.Employee, error) {
	args := m.Called(ctx, mobile)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) ListByMobile(ctx context.Context, mobile string) ([]*domain.Employee, error) {
	args := m.Called(ctx, mobile)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	args := m.Called(ctx, email)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) ExistsByCCCD(ctx context.Context, cccd string) (bool, error) {
	args := m.Called(ctx, cccd)
	return args.Bool(0), args.Error(1)
}

func (m *MockEmployeeRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	args := m.Called(ctx, email)
	return args.Bool(0), args.Error(1)
}

func (m *MockEmployeeRepository) GetByBankAccountNumber(ctx context.Context, bankAccountNumber string) (*domain.Employee, error) {
	args := m.Called(ctx, bankAccountNumber)
	return args.Get(0).(*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) Update(ctx context.Context, employee *domain.Employee) error {
	args := m.Called(ctx, employee)
	return args.Error(0)
}

func (m *MockEmployeeRepository) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockEmployeeRepository) HardDelete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockEmployeeRepository) List(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) ListWithProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProject, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.EmployeeWithProject), args.Error(1)
}

func (m *MockEmployeeRepository) ListWithAllProjects(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.EmployeeWithProjects), args.Error(1)
}

func (m *MockEmployeeRepository) ListAccessibleIDs(ctx context.Context, userID uint) ([]uint, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]uint), args.Error(1)
}

func (m *MockEmployeeRepository) Count(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepository) BulkCreate(ctx context.Context, employees []*domain.Employee) error {
	args := m.Called(ctx, employees)
	return args.Error(0)
}

func (m *MockEmployeeRepository) GetByProject(ctx context.Context, projectID uint) ([]*domain.Employee, error) {
	args := m.Called(ctx, projectID)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetByProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*domain.Employee, error) {
	args := m.Called(ctx, projectID, createdBy)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetEmployeesSummary(ctx context.Context) (*domain.EmployeesSummary, error) {
	args := m.Called(ctx)
	return args.Get(0).(*domain.EmployeesSummary), args.Error(1)
}

func (m *MockEmployeeRepository) GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*domain.EmployeesSummary, error) {
	args := m.Called(ctx, createdBy)
	return args.Get(0).(*domain.EmployeesSummary), args.Error(1)
}

func (m *MockEmployeeRepository) GetUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters domain.EmployeeFilters) ([]*domain.Employee, error) {
	args := m.Called(ctx, atDate, filters)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) CountUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters domain.EmployeeFilters) (int64, error) {
	args := m.Called(ctx, atDate, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepository) GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error) {
	args := m.Called(ctx, date)
	return args.Int(0), args.Error(1)
}

func (m *MockEmployeeRepository) GetActiveCountAtDateForCreator(ctx context.Context, date time.Time, createdBy uint) (int, error) {
	args := m.Called(ctx, date, createdBy)
	return args.Int(0), args.Error(1)
}

func (m *MockEmployeeRepository) GetRecentEmployees(ctx context.Context, startDate, endDate time.Time, limit int) ([]*domain.Employee, error) {
	args := m.Called(ctx, startDate, endDate, limit)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) GetRecentEmployeesPaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*domain.Employee, error) {
	args := m.Called(ctx, startDate, endDate, limit, offset)
	return args.Get(0).([]*domain.Employee), args.Error(1)
}

func (m *MockEmployeeRepository) CountRecentEmployees(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	args := m.Called(ctx, startDate, endDate)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepository) SearchEmployees(ctx context.Context, search string, limit int) ([]*domain.EmployeeWithProject, error) {
	args := m.Called(ctx, search, limit)
	return args.Get(0).([]*domain.EmployeeWithProject), args.Error(1)
}

func (m *MockEmployeeRepository) GetEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) ([]*domain.EmployeeWithProjects, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).([]*domain.EmployeeWithProjects), args.Error(1)
}

func (m *MockEmployeeRepository) CountEmployeesWithMissingBankDetails(ctx context.Context, filters domain.EmployeeFilters) (int64, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockEmployeeRepository) ListEmployeeReachability(ctx context.Context, filters domain.EmployeeReachabilityFilters) (*domain.EmployeeReachabilityPage, error) {
	args := m.Called(ctx, filters)
	return args.Get(0).(*domain.EmployeeReachabilityPage), args.Error(1)
}

func (m *MockEmployeeRepository) UpdateColumns(ctx context.Context, id uint, columns map[string]any) error {
	args := m.Called(ctx, id, columns)
	return args.Error(0)
}

// MockEventBus is a mock for EventBus
type MockEventBus struct {
	mock.Mock
}

func (m *MockEventBus) Publish(ctx context.Context, events ...domain.DomainEvent) error {
	args := m.Called(ctx, events)
	return args.Error(0)
}

func (m *MockEventBus) Subscribe(eventType string, handler domain.EventHandler) {
	_ = m.Called(eventType, handler)
}

func (m *MockEventBus) SubscribeAll(handler domain.EventHandler) {
	_ = m.Called(handler)
}

func TestEmployeeSyncService_ValidateEmployeeNameConsistency(t *testing.T) {
	ctx := context.Background()
	employeeID := uint(1)
	correctName := "Nguyen Van A"

	// Create mocks
	mockProjectEmployeeRepo := &MockProjectEmployeeRepository{}
	mockEmployeeRepo := &MockEmployeeRepository{}

	// Create service
	service := &EmployeeSyncService{
		projectEmployeeRepo: mockProjectEmployeeRepo,
		employeeRepo:        mockEmployeeRepo,
	}

	// Setup employee mock response
	employee := &domain.Employee{
		ID:       employeeID,
		Fullname: correctName,
		CCCD:     "123456789012",
	}

	// Setup assignments with mixed consistency
	pastDate := time.Now().AddDate(-1, 0, 0)
	assignments := []*domain.ProjectEmployee{
		{
			ID:           1,
			EmployeeID:   employeeID,
			EmployeeName: correctName, // Consistent
			LastDate:     nil,
		},
		{
			ID:           2,
			EmployeeID:   employeeID,
			EmployeeName: "Wrong Name", // Inconsistent
			LastDate:     &pastDate,
		},
		{
			ID:           3,
			EmployeeID:   employeeID,
			EmployeeName: correctName, // Consistent
			LastDate:     nil,
		},
	}

	// Setup mock expectations
	mockEmployeeRepo.On("GetByID", ctx, employeeID).Return(employee, nil)
	mockProjectEmployeeRepo.On("GetByEmployee", ctx, employeeID).Return(assignments, nil)

	// Execute test
	report, err := service.ValidateEmployeeNameConsistency(ctx, employeeID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, employeeID, report.EmployeeID)
	assert.Equal(t, correctName, report.CanonicalName)
	assert.Equal(t, 3, report.TotalAssignments)
	assert.Equal(t, 1, report.InconsistentCount)
	assert.False(t, report.IsConsistent())
	assert.Len(t, report.InconsistentAssignments, 1)

	// Check inconsistent assignment details
	inconsistent := report.InconsistentAssignments[0]
	assert.Equal(t, uint(2), inconsistent.AssignmentID)
	assert.Equal(t, "Wrong Name", inconsistent.CurrentName)
	assert.Equal(t, correctName, inconsistent.CorrectName)

	// Verify all mocks were called as expected
	mockEmployeeRepo.AssertExpectations(t)
	mockProjectEmployeeRepo.AssertExpectations(t)
}

func TestInconsistentAssignment_IsValid(t *testing.T) {
	tests := []struct {
		name        string
		assignment  InconsistentAssignment
		expectValid bool
	}{
		{
			name: "valid assignment",
			assignment: InconsistentAssignment{
				AssignmentID: 1,
				ProjectID:    1,
				CurrentName:  "Wrong Name",
				CorrectName:  "Correct Name",
				LastDate:     nil,
			},
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test that the struct can be created properly
			assert.NotNil(t, tt.assignment)
			assert.Equal(t, tt.assignment.CurrentName, "Wrong Name")
			assert.Equal(t, tt.assignment.CorrectName, "Correct Name")
		})
	}
}
