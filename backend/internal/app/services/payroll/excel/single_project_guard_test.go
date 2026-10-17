package excel

import (
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func guardData(entries map[EmployeeProjectKey]int64, employeeNames map[uint]string, projectNames map[uint]string) *BulkTransferData {
	data := &BulkTransferData{
		EmployeeProjectAmounts: entries,
		EmployeeData:           make(map[uint]Employee, len(employeeNames)),
		ProjectData:            make(map[uint]Project, len(projectNames)),
	}
	for id, name := range employeeNames {
		data.EmployeeData[id] = Employee{ID: id, Fullname: name}
	}
	for id, name := range projectNames {
		data.ProjectData[id] = Project{ID: id, Name: name}
	}
	return data
}

func TestValidateOneProjectPerCycle_AllowsOneProjectPerEmployee(t *testing.T) {
	data := guardData(map[EmployeeProjectKey]int64{
		{EmployeeID: 7, ProjectID: 12}:  5_000_000,
		{EmployeeID: 9, ProjectID: 12}:  2_000_000,
		{EmployeeID: 11, ProjectID: 13}: 800_000,
	}, map[uint]string{7: "Nguyễn Văn A", 9: "Trần B", 11: "Lê C"}, map[uint]string{12: "Site A", 13: "Site B"})

	require.NoError(t, ValidateOneProjectPerCycle(data))
}

func TestValidateOneProjectPerCycle_AllowsNilData(t *testing.T) {
	require.NoError(t, ValidateOneProjectPerCycle(nil))
}

func TestValidateOneProjectPerCycle_RefusesOneEmployeeInTwoProjects(t *testing.T) {
	data := guardData(map[EmployeeProjectKey]int64{
		{EmployeeID: 7, ProjectID: 12}: 5_000_000,
		{EmployeeID: 7, ProjectID: 13}: 250_000,
		{EmployeeID: 9, ProjectID: 12}: 2_000_000,
	}, map[uint]string{7: "Nguyễn Văn A", 9: "Trần B"}, map[uint]string{12: "PQC Hải Phòng", 13: "PQC Nam Định"})

	err := ValidateOneProjectPerCycle(data)

	require.Error(t, err)
	assert.True(t, domain.IsValidationError(err))
	var domainErr *domain.DomainError
	require.ErrorAs(t, err, &domainErr)
	assert.Equal(t, domain.ErrCodeMultipleProjectsInCycle, domainErr.Code)
	assert.Contains(t, err.Error(), "1 nhân viên")
	assert.Contains(t, err.Error(), "Nguyễn Văn A")
	assert.Contains(t, err.Error(), "PQC Hải Phòng")
	assert.Contains(t, err.Error(), "PQC Nam Định")
	assert.NotContains(t, err.Error(), "Trần B", "only the offending employee is named")
}

func TestValidateOneProjectPerCycle_ListsOffendersAndSummarisesTheRest(t *testing.T) {
	entries := make(map[EmployeeProjectKey]int64)
	employees := make(map[uint]string)
	for employeeID := uint(1); employeeID <= 7; employeeID++ {
		employees[employeeID] = "Employee " + string(rune('A'+employeeID-1))
		entries[EmployeeProjectKey{EmployeeID: employeeID, ProjectID: 12}] = 100
		entries[EmployeeProjectKey{EmployeeID: employeeID, ProjectID: 13}] = 100
	}
	data := guardData(entries, employees, map[uint]string{12: "Site A", 13: "Site B"})

	err := ValidateOneProjectPerCycle(data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "7 nhân viên")
	assert.Contains(t, err.Error(), "và 2 nhân viên khác")
	assert.NotContains(t, err.Error(), "Employee G", "the listing is capped at five names")
}

func TestValidateOneProjectPerCycle_FallsBackToIdentifiersWithoutNames(t *testing.T) {
	data := guardData(map[EmployeeProjectKey]int64{
		{EmployeeID: 7, ProjectID: 12}: 100,
		{EmployeeID: 7, ProjectID: 13}: 100,
	}, nil, nil)

	err := ValidateOneProjectPerCycle(data)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nhân viên #7")
	assert.Contains(t, err.Error(), "Dự án #12")
	assert.Contains(t, err.Error(), "Dự án #13")
}
