package specs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockWorkingDaysCounter is a mock implementation of WorkingDaysCounter
type MockWorkingDaysCounter struct {
	mock.Mock
}

func (m *MockWorkingDaysCounter) CountDistinctWorkingDays(ctx context.Context, employeeID, projectID uint, startDate, endDate time.Time) (int, error) {
	args := m.Called(ctx, employeeID, projectID, startDate, endDate)
	return args.Int(0), args.Error(1)
}

func TestNewWorkingDaysSpec(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 5

	spec := NewWorkingDaysSpec(mockCounter, threshold)

	assert.NotNil(t, spec)
	assert.Equal(t, mockCounter, spec.Counter)
	assert.Equal(t, threshold, spec.Threshold)
}

func TestWorkingDaysSpec_IsEligible_MeetsThreshold(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 5
	spec := NewWorkingDaysSpec(mockCounter, threshold)

	ctx := context.Background()
	employeeID := uint(1)
	projectID := uint(10)
	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	// Mock returns exactly the threshold
	mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(5, nil)

	eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

	assert.NoError(t, err)
	assert.True(t, eligible)
	mockCounter.AssertExpectations(t)
}

func TestWorkingDaysSpec_IsEligible_ExceedsThreshold(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 5
	spec := NewWorkingDaysSpec(mockCounter, threshold)

	ctx := context.Background()
	employeeID := uint(2)
	projectID := uint(20)
	fromDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 2, 28, 0, 0, 0, 0, time.UTC)

	// Mock returns more than threshold
	mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(10, nil)

	eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

	assert.NoError(t, err)
	assert.True(t, eligible)
	mockCounter.AssertExpectations(t)
}

func TestWorkingDaysSpec_IsEligible_BelowThreshold(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 5
	spec := NewWorkingDaysSpec(mockCounter, threshold)

	ctx := context.Background()
	employeeID := uint(3)
	projectID := uint(30)
	fromDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	// Mock returns less than threshold
	mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(3, nil)

	eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

	assert.NoError(t, err)
	assert.False(t, eligible)
	mockCounter.AssertExpectations(t)
}

func TestWorkingDaysSpec_IsEligible_ZeroDays(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 1
	spec := NewWorkingDaysSpec(mockCounter, threshold)

	ctx := context.Background()
	employeeID := uint(4)
	projectID := uint(40)
	fromDate := time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 4, 7, 0, 0, 0, 0, time.UTC)

	// Mock returns zero days
	mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(0, nil)

	eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

	assert.NoError(t, err)
	assert.False(t, eligible)
	mockCounter.AssertExpectations(t)
}

func TestWorkingDaysSpec_IsEligible_Error(t *testing.T) {
	mockCounter := new(MockWorkingDaysCounter)
	threshold := 5
	spec := NewWorkingDaysSpec(mockCounter, threshold)

	ctx := context.Background()
	employeeID := uint(5)
	projectID := uint(50)
	fromDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 5, 31, 0, 0, 0, 0, time.UTC)

	expectedErr := errors.New("database error")
	mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(0, expectedErr)

	eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.False(t, eligible)
	mockCounter.AssertExpectations(t)
}

func TestWorkingDaysSpec_IsEligible_DifferentThresholds(t *testing.T) {
	tests := []struct {
		name      string
		threshold int
		dayCount  int
		expected  bool
	}{
		{
			name:      "Threshold 1, meets requirement",
			threshold: 1,
			dayCount:  1,
			expected:  true,
		},
		{
			name:      "Threshold 10, exceeds requirement",
			threshold: 10,
			dayCount:  15,
			expected:  true,
		},
		{
			name:      "Threshold 20, below requirement",
			threshold: 20,
			dayCount:  18,
			expected:  false,
		},
		{
			name:      "Threshold 0, always eligible",
			threshold: 0,
			dayCount:  0,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCounter := new(MockWorkingDaysCounter)
			spec := NewWorkingDaysSpec(mockCounter, tt.threshold)

			ctx := context.Background()
			employeeID := uint(1)
			projectID := uint(1)
			fromDate := time.Now()
			toDate := time.Now()

			mockCounter.On("CountDistinctWorkingDays", ctx, employeeID, projectID, fromDate, toDate).Return(tt.dayCount, nil)

			eligible, err := spec.IsEligible(ctx, employeeID, projectID, fromDate, toDate)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, eligible)
			mockCounter.AssertExpectations(t)
		})
	}
}
