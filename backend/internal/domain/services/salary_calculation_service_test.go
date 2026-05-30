package services

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockBulkTransferFileRepository is a mock implementation of domain.BulkTransferFileRepository
type MockBulkTransferFileRepository struct {
	mock.Mock
}

func (m *MockBulkTransferFileRepository) Create(ctx context.Context, file *domain.BulkTransferFile) error {
	args := m.Called(ctx, file)
	return args.Error(0)
}

func (m *MockBulkTransferFileRepository) GetByID(ctx context.Context, id uint) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetByIDWithAsset(ctx context.Context, id uint) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetByFilename(ctx context.Context, filename string) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx, filename)
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetByAssetID(ctx context.Context, assetID uint) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx, assetID)
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) FindByTimesheetIDs(ctx context.Context, timesheetIDs []uint) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx, timesheetIDs)
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) UpdateTransactionData(ctx context.Context, tx interface{}, id uint, updatedData string, transferResultID uint) error {
	args := m.Called(ctx, tx, id, updatedData, transferResultID)
	return args.Error(0)
}

func (m *MockBulkTransferFileRepository) UpdateWithLock(ctx context.Context, id uint, updates map[string]interface{}) error {
	args := m.Called(ctx, id, updates)
	return args.Error(0)
}

func (m *MockBulkTransferFileRepository) ListWithFilters(ctx context.Context, cycle string, fromDate, toDate *time.Time, limit, offset int) ([]*domain.BulkTransferFile, int64, error) {
	args := m.Called(ctx, cycle, fromDate, toDate, limit, offset)
	return args.Get(0).([]*domain.BulkTransferFile), args.Get(1).(int64), args.Error(2)
}

func (m *MockBulkTransferFileRepository) ListForUploadHistories(ctx context.Context, sortBy, sortOrder string, fromDate, toDate *time.Time, limit, offset int) ([]*domain.BulkTransferFile, int64, error) {
	args := m.Called(ctx, sortBy, sortOrder, fromDate, toDate, limit, offset)
	return args.Get(0).([]*domain.BulkTransferFile), args.Get(1).(int64), args.Error(2)
}

func (m *MockBulkTransferFileRepository) ListForPayrollReport(ctx context.Context, fromDate, toDate time.Time) ([]*domain.BulkTransferFile, error) {
	args := m.Called(ctx, fromDate, toDate)
	return args.Get(0).([]*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) DeleteOrphanFiles(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetRecentHighValueTransfers(ctx context.Context, fromDate, toDate time.Time) (map[uint][]int64, error) {
	args := m.Called(ctx, fromDate, toDate)
	return args.Get(0).(map[uint][]int64), args.Error(1)
}

func (m *MockBulkTransferFileRepository) CountEmployeesPaidInPeriod(ctx context.Context, fromDate, toDate time.Time) (int, error) {
	args := m.Called(ctx, fromDate, toDate)
	return args.Int(0), args.Error(1)
}

func (m *MockBulkTransferFileRepository) FindAllInDateRange(ctx context.Context, fromDate, toDate time.Time, files *[]*domain.BulkTransferFile) error {
	args := m.Called(ctx, fromDate, toDate, files)
	return args.Error(0)
}

func (m *MockBulkTransferFileRepository) FindByTimesheetIDGroups(ctx context.Context, timesheetIDs []uint) ([]*domain.BulkTransferFile, error) {
	args := m.Called(ctx, timesheetIDs)
	return args.Get(0).([]*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetSalaryDistribution(ctx context.Context, fromDate, toDate *time.Time) (map[string][]int64, error) {
	args := m.Called(ctx, fromDate, toDate)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]int64), args.Error(1)
}

func (m *MockBulkTransferFileRepository) GetMostRecentWithoutAsset(ctx context.Context) (*domain.BulkTransferFile, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.BulkTransferFile), args.Error(1)
}

func (m *MockBulkTransferFileRepository) FindDataByTransactionCodes(ctx context.Context, transactionCodes []string) ([]domain.BulkTransferFileDataEntry, string, error) {
	args := m.Called(ctx, transactionCodes)
	if args.Get(0) == nil {
		return nil, "", args.Error(1)
	}
	return args.Get(0).([]domain.BulkTransferFileDataEntry), args.String(1), args.Error(2)
}

func (m *MockBulkTransferFileRepository) ListResultUploads(ctx context.Context, fromDate, toDate *time.Time, limit, offset int, sortOrder string) ([]*domain.BulkTransferFile, int64, error) {
	args := m.Called(ctx, fromDate, toDate, limit, offset, sortOrder)
	return args.Get(0).([]*domain.BulkTransferFile), args.Get(1).(int64), args.Error(2)
}

func (m *MockBulkTransferFileRepository) UpdateCounts(ctx context.Context, id uint, completedCount, failedCount int) error {
	args := m.Called(ctx, id, completedCount, failedCount)
	return args.Error(0)
}

func (m *MockBulkTransferFileRepository) GetPendingUploadsByUser(ctx context.Context, userID uint) ([]*domain.BulkTransferFile, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.BulkTransferFile), args.Error(1)
}

func TestCalculateAverageSalariesFromTransfers(t *testing.T) {
	tests := []struct {
		name              string
		employeeTransfers map[uint][]int64
		expectedWeekly    int64
		expectedMonthly   int64
		expectError       bool
		repoError         bool
	}{
		{
			name: "Single employee with multiple transfers",
			employeeTransfers: map[uint][]int64{
				1: {1500000, 1800000, 1600000}, // Average: 1633333
			},
			expectedWeekly:  1633333,
			expectedMonthly: 6533332, // 1633333 * 4
			expectError:     false,
		},
		{
			name: "Multiple employees with different salaries",
			employeeTransfers: map[uint][]int64{
				1: {1500000, 1600000}, // Average: 1550000
				2: {2000000},          // Average: 2000000
				3: {1200000, 1300000}, // Average: 1250000
			},
			expectedWeekly:  1600000, // (1550000 + 2000000 + 1250000) / 3 = 1600000
			expectedMonthly: 6400000, // 1600000 * 4
			expectError:     false,
		},
		{
			name:              "No transfers found",
			employeeTransfers: map[uint][]int64{},
			expectedWeekly:    0,
			expectedMonthly:   0,
			expectError:       false,
		},
		{
			name: "Employee with empty transfer list",
			employeeTransfers: map[uint][]int64{
				1: {},        // Empty list should be skipped
				2: {1800000}, // Average: 1800000
			},
			expectedWeekly:  1800000,
			expectedMonthly: 7200000,
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock repository
			mockRepo := &MockBulkTransferFileRepository{}
			ctx := context.Background()

			// Setup mock expectations
			if tt.repoError {
				mockRepo.On("GetRecentHighValueTransfers", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
					Return(nil, assert.AnError)
			} else {
				mockRepo.On("GetRecentHighValueTransfers", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
					Return(tt.employeeTransfers, nil)
			}

			// Create service
			service := NewSalaryCalculationService(mockRepo)

			// Call the method
			weeklyAvg, monthlyAvg, err := service.CalculateAverageSalariesFromTransfers(ctx)

			// Assert results
			if tt.expectError {
				assert.Error(t, err)
				assert.Zero(t, weeklyAvg)
				assert.Zero(t, monthlyAvg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedWeekly, weeklyAvg)
				assert.Equal(t, tt.expectedMonthly, monthlyAvg)
			}

			// Verify all expectations were met
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestNewSalaryCalculationService(t *testing.T) {
	mockRepo := &MockBulkTransferFileRepository{}

	service := NewSalaryCalculationService(mockRepo)

	assert.NotNil(t, service)
	assert.Equal(t, mockRepo, service.bulkTransferFileRepo)
}

func TestCalculateAverageSalariesFromTransfers_RealCalculation(t *testing.T) {
	// Test with real calculation scenarios to ensure business logic is correct
	tests := []struct {
		name              string
		employeeTransfers map[uint][]int64
		expectedWeekly    int64
		expectedMonthly   int64
	}{
		{
			name: "Precise division test",
			employeeTransfers: map[uint][]int64{
				1: {1000000, 2000000, 3000000}, // Average: 2000000
				2: {4000000},                   // Average: 4000000
			},
			expectedWeekly:  3000000,  // (2000000 + 4000000) / 2
			expectedMonthly: 12000000, // 3000000 * 4
		},
		{
			name: "Large numbers test",
			employeeTransfers: map[uint][]int64{
				1: {15000000, 16000000}, // Average: 15500000
				2: {12000000},           // Average: 12000000
			},
			expectedWeekly:  13750000, // (15500000 + 12000000) / 2
			expectedMonthly: 55000000, // 13750000 * 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockBulkTransferFileRepository{}
			ctx := context.Background()

			mockRepo.On("GetRecentHighValueTransfers", ctx, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
				Return(tt.employeeTransfers, nil)

			service := NewSalaryCalculationService(mockRepo)
			weeklyAvg, monthlyAvg, err := service.CalculateAverageSalariesFromTransfers(ctx)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedWeekly, weeklyAvg, "Weekly average mismatch")
			assert.Equal(t, tt.expectedMonthly, monthlyAvg, "Monthly average mismatch")

			mockRepo.AssertExpectations(t)
		})
	}
}
