package excel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing
type mockSettingsConfigService struct {
	bulkPct float64
}

func (m *mockSettingsConfigService) GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64 {
	if m.bulkPct > 0 {
		return m.bulkPct
	}
	return 0.7
}

func TestNewService(t *testing.T) {
	// Setup
	settings := &mockSettingsConfigService{}

	// Execute
	service := NewService(settings)

	// Assert
	assert.NotNil(t, service)
	assert.NotNil(t, service.settingsConfig)
}

func TestService_ValidateAndFilterBulkTransferData_ValidData(t *testing.T) {
	// Setup
	service := NewService(&mockSettingsConfigService{})

	data := &BulkTransferData{
		EmployeeProjectAmounts: map[EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 1}: 1000,
			{EmployeeID: 2, ProjectID: 1}: 2000,
		},
		EmployeeProjectTimesheets: map[EmployeeProjectKey][]uint{
			{EmployeeID: 1, ProjectID: 1}: {1, 2},
			{EmployeeID: 2, ProjectID: 1}: {3, 4},
		},
		EmployeeData: map[uint]Employee{
			1: {
				ID:                1,
				Fullname:          "John Doe",
				BankAccountNumber: "123456789",
				BankAccountName:   "John Account",
				Bank: &Bank{
					ID:         1,
					BranchName: "Main Branch",
				},
			},
			2: {
				ID:                2,
				Fullname:          "Jane Smith",
				BankAccountNumber: "987654321",
				BankAccountName:   "Jane Account",
				Bank: &Bank{
					ID:         2,
					BranchName: "East Branch",
				},
			},
		},
		ProjectData: map[uint]Project{
			1: {ID: 1, Name: "Project Alpha"},
		},
	}

	// Execute
	result := service.ValidateAndFilterBulkTransferData(data)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalCount)
	assert.Equal(t, 2, result.ValidCount)
	assert.Equal(t, 0, result.SkippedCount)
	assert.Empty(t, result.SkippedEmployees)
	assert.Len(t, result.ValidData.EmployeeProjectAmounts, 2)
}

func TestService_ValidateAndFilterBulkTransferData_MissingAccountNumber(t *testing.T) {
	// Setup
	service := NewService(&mockSettingsConfigService{})

	data := &BulkTransferData{
		EmployeeProjectAmounts: map[EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 1}: 1000,
		},
		EmployeeProjectTimesheets: map[EmployeeProjectKey][]uint{
			{EmployeeID: 1, ProjectID: 1}: {1, 2},
		},
		EmployeeData: map[uint]Employee{
			1: {
				ID:                1,
				Fullname:          "John Doe",
				BankAccountNumber: "", // Missing
				BankAccountName:   "John Account",
			},
		},
		ProjectData: map[uint]Project{
			1: {ID: 1, Name: "Project Alpha"},
		},
	}

	// Execute
	result := service.ValidateAndFilterBulkTransferData(data)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, 0, result.ValidCount)
	assert.Equal(t, 1, result.SkippedCount)
	assert.Len(t, result.SkippedEmployees, 1)
	assert.Equal(t, "Missing bank account number", result.SkippedEmployees[0].Reason)
	assert.Equal(t, uint(1), result.SkippedEmployees[0].EmployeeID)
}

func TestService_ValidateAndFilterBulkTransferData_MissingAccountName(t *testing.T) {
	// Setup
	service := NewService(&mockSettingsConfigService{})

	data := &BulkTransferData{
		EmployeeProjectAmounts: map[EmployeeProjectKey]int64{
			{EmployeeID: 1, ProjectID: 1}: 1000,
		},
		EmployeeProjectTimesheets: map[EmployeeProjectKey][]uint{
			{EmployeeID: 1, ProjectID: 1}: {1, 2},
		},
		EmployeeData: map[uint]Employee{
			1: {
				ID:                1,
				Fullname:          "John Doe",
				BankAccountNumber: "123456789",
				BankAccountName:   "", // Missing
			},
		},
		ProjectData: map[uint]Project{
			1: {ID: 1, Name: "Project Alpha"},
		},
	}

	// Execute
	result := service.ValidateAndFilterBulkTransferData(data)

	// Assert
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.TotalCount)
	assert.Equal(t, 0, result.ValidCount)
	assert.Equal(t, 1, result.SkippedCount)
	assert.Len(t, result.SkippedEmployees, 1)
	assert.Equal(t, "Missing bank account name", result.SkippedEmployees[0].Reason)
}

func TestService_GenerateBulkTransferExcel_EmptyData(t *testing.T) {
	// Skip this test as it requires template files that may not exist in test environment
	t.Skip("Skipping test that requires template files")
}

func TestEmployeeProjectKey_Structure(t *testing.T) {
	// Test that the key structure is correct
	key := EmployeeProjectKey{
		EmployeeID: 1,
		ProjectID:  2,
	}

	assert.Equal(t, uint(1), key.EmployeeID)
	assert.Equal(t, uint(2), key.ProjectID)

	// Test that keys are comparable (can be used as map keys)
	keyMap := make(map[EmployeeProjectKey]string)
	keyMap[key] = "test"
	assert.Equal(t, "test", keyMap[key])
}

func TestEmployee_Structure(t *testing.T) {
	bank := &Bank{
		ID:         1,
		BranchName: "Main",
	}

	employee := Employee{
		ID:                1,
		Fullname:          "John Doe",
		BankAccountNumber: "123456",
		BankAccountName:   "Account Name",
		Bank:              bank,
	}

	assert.Equal(t, uint(1), employee.ID)
	assert.Equal(t, "John Doe", employee.Fullname)
	assert.Equal(t, "123456", employee.BankAccountNumber)
	assert.Equal(t, "Account Name", employee.BankAccountName)
	assert.NotNil(t, employee.Bank)
	assert.Equal(t, "Main", employee.Bank.BranchName)
}

func TestProject_Structure(t *testing.T) {
	project := Project{
		ID:   1,
		Name: "Test Project",
	}

	assert.Equal(t, uint(1), project.ID)
	assert.Equal(t, "Test Project", project.Name)
}

func TestBulkTransferValidationResult_Structure(t *testing.T) {
	validData := &BulkTransferData{
		EmployeeProjectAmounts:    make(map[EmployeeProjectKey]int64),
		EmployeeProjectTimesheets: make(map[EmployeeProjectKey][]uint),
		EmployeeData:              make(map[uint]Employee),
		ProjectData:               make(map[uint]Project),
	}

	skipped := []SkippedEmployee{
		{
			EmployeeID:   1,
			EmployeeName: "John",
			ProjectID:    2,
			ProjectName:  "Project A",
			Reason:       "Missing bank info",
		},
	}

	result := &BulkTransferValidationResult{
		ValidData:        validData,
		SkippedEmployees: skipped,
		TotalCount:       10,
		ValidCount:       9,
		SkippedCount:     1,
	}

	assert.NotNil(t, result.ValidData)
	assert.Len(t, result.SkippedEmployees, 1)
	assert.Equal(t, 10, result.TotalCount)
	assert.Equal(t, 9, result.ValidCount)
	assert.Equal(t, 1, result.SkippedCount)
}
