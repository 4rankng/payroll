package excel

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/pkg/constants"
)

// Mock implementations for testing
type mockSettingsConfigService struct {
	bulkPct float64
	calls   int
}

type fixedSettingsConfigService float64

func (s fixedSettingsConfigService) GetPaymentPercentageForSchedule(context.Context, string) float64 {
	return float64(s)
}

func (m *mockSettingsConfigService) GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64 {
	m.calls++
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
	withBackendWorkingDirectory(t)
	service := NewService(&mockSettingsConfigService{bulkPct: 1})

	response, err := service.GenerateBulkTransferExcel(
		context.Background(),
		newBulkTransferTestData(nil),
		&dto.ExportBulkTransferRequest{},
		"2026-07-01",
		"2026-07-07",
		"weekly",
	)

	require.NoError(t, err)
	assert.Equal(t, xlsxContentType, response.ContentType)
	assert.Equal(t, ".xlsx", response.FileExtension)
	rows := readWorkbookRows(t, response.Data)
	assert.Empty(t, rows)
}

func TestPartitionTransferRows_BoundaryAndStrictEquality(t *testing.T) {
	tests := []struct {
		name       string
		amounts    []int64
		wantTotals []int64
	}{
		{
			name:       "maximum allowed total stays in one workbook",
			amounts:    []int64{499_999_999},
			wantTotals: []int64{499_999_999},
		},
		{
			name:       "exact cap spills into another workbook",
			amounts:    []int64{300_000_000, 200_000_000},
			wantTotals: []int64{300_000_000, 200_000_000},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := make([]transferRow, 0, len(tt.amounts))
			for i, amount := range tt.amounts {
				rows = append(rows, transferRow{
					key:           EmployeeProjectKey{EmployeeID: uint(i + 1), ProjectID: 1},
					paymentAmount: amount,
				})
			}

			partitions, err := partitionTransferRows(rows)

			require.NoError(t, err)
			assert.Equal(t, tt.wantTotals, partitionTotals(partitions))
		})
	}
}

func TestService_BuildTransferRows_DeterministicAndPercentageCalculatedOnce(t *testing.T) {
	settings := &mockSettingsConfigService{bulkPct: 0.7}
	service := NewService(settings)
	entries := []bulkTransferTestEntry{
		{employeeID: 3, projectID: 2, amount: 300, transactionCode: "TX-3-2"},
		{employeeID: 1, projectID: 4, amount: 100, transactionCode: "TX-1-4"},
		{employeeID: 1, projectID: 2, amount: 200, transactionCode: "TX-1-2"},
	}
	dataA := newBulkTransferTestData(entries)
	dataB := newBulkTransferTestData([]bulkTransferTestEntry{entries[2], entries[0], entries[1]})

	rowsA, err := service.buildTransferRows(context.Background(), dataA, "weekly")
	require.NoError(t, err)
	rowsB, err := service.buildTransferRows(context.Background(), dataB, "weekly")
	require.NoError(t, err)

	assert.Equal(t, rowsA, rowsB)
	assert.Equal(t, []EmployeeProjectKey{
		{EmployeeID: 1, ProjectID: 2},
		{EmployeeID: 1, ProjectID: 4},
		{EmployeeID: 3, ProjectID: 2},
	}, transferRowKeys(rowsA))
	assert.Equal(t, []int64{140, 70, 210}, transferRowAmounts(rowsA))
	assert.Equal(t, 2, settings.calls, "payment percentage should be fetched once per export build")
}

func TestService_BuildTransferRows_RejectsInvalidAmounts(t *testing.T) {
	tests := []struct {
		name       string
		amount     int64
		percentage float64
	}{
		{name: "zero source", amount: 0, percentage: 1},
		{name: "negative source", amount: -1, percentage: 1},
		{name: "zero percentage", amount: 100, percentage: 0},
		{name: "negative percentage", amount: 100, percentage: -1},
		{name: "NaN percentage", amount: 100, percentage: math.NaN()},
		{name: "positive infinity percentage", amount: 100, percentage: math.Inf(1)},
		{name: "exact workbook cap", amount: 500_000_000, percentage: 1},
		{name: "non-representable result", amount: math.MaxInt64, percentage: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(fixedSettingsConfigService(tt.percentage))
			data := newBulkTransferTestData([]bulkTransferTestEntry{
				{employeeID: 1, projectID: 1, amount: tt.amount, transactionCode: "TX-1"},
			})

			rows, err := service.buildTransferRows(context.Background(), data, "weekly")

			assert.Error(t, err)
			assert.Nil(t, rows)
		})
	}
}

func TestPartitionTransferRows_UsesSequentialSortedPacking(t *testing.T) {
	rows := []transferRow{
		{key: EmployeeProjectKey{EmployeeID: 1, ProjectID: 1}, paymentAmount: 300_000_000},
		{key: EmployeeProjectKey{EmployeeID: 2, ProjectID: 1}, paymentAmount: 300_000_000},
		{key: EmployeeProjectKey{EmployeeID: 3, ProjectID: 1}, paymentAmount: 199_000_000},
	}

	partitions, err := partitionTransferRows(rows)

	require.NoError(t, err)
	assert.Equal(t, []int64{300_000_000, 499_000_000}, partitionTotals(partitions))
	assert.Equal(t, []EmployeeProjectKey{
		{EmployeeID: 1, ProjectID: 1},
	}, transferRowKeys(partitions[0]))
	assert.Equal(t, []EmployeeProjectKey{
		{EmployeeID: 2, ProjectID: 1},
		{EmployeeID: 3, ProjectID: 1},
	}, transferRowKeys(partitions[1]))
}

func TestSafeBulkTransferCycleLabel(t *testing.T) {
	tests := map[string]string{
		"weekly":             "weekly",
		" MONTHLY ":          "monthly",
		"flexible":           "payment",
		"../../private/data": "payment",
		"":                   "payment",
	}
	for input, expected := range tests {
		assert.Equal(t, expected, safeBulkTransferCycleLabel(input))
	}
}

func TestService_GenerateBulkTransferExcel_SingleWorkbookAtBoundary(t *testing.T) {
	withBackendWorkingDirectory(t)
	service := NewService(&mockSettingsConfigService{bulkPct: 1})
	data := newBulkTransferTestData([]bulkTransferTestEntry{
		{employeeID: 2, projectID: 1, amount: 200_000_000, transactionCode: "TX-2"},
		{employeeID: 1, projectID: 1, amount: 299_999_999, transactionCode: "TX-1"},
	})

	response, err := service.GenerateBulkTransferExcel(context.Background(), data, &dto.ExportBulkTransferRequest{}, "", "", "monthly")

	require.NoError(t, err)
	assert.Equal(t, xlsxContentType, response.ContentType)
	assert.Equal(t, ".xlsx", response.FileExtension)
	rows := readWorkbookRows(t, response.Data)
	require.Len(t, rows, 2)
	assert.Equal(t, []string{"TX-1", "TX-2"}, []string{rows[0].transactionCode, rows[1].transactionCode})
	assert.Equal(t, int64(499_999_999), sumWorkbookRows(rows))
}

func TestService_GenerateBulkTransferExcelWithPaymentPercentage_UsesCapturedValue(t *testing.T) {
	withBackendWorkingDirectory(t)
	settings := &mockSettingsConfigService{bulkPct: 0.8}
	service := NewService(settings)
	data := newBulkTransferTestData([]bulkTransferTestEntry{
		{employeeID: 1, projectID: 1, amount: 1_000, transactionCode: "TX-1"},
	})

	response, err := service.GenerateBulkTransferExcelWithPaymentPercentage(data, &dto.ExportBulkTransferRequest{}, "", "", "weekly", 0.7)

	require.NoError(t, err)
	rows := readWorkbookRows(t, response.Data)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(700), rows[0].amount)
	assert.Zero(t, settings.calls, "captured export percentage must not be read again")
}

func TestService_GenerateBulkTransferExcel_ExpandsWorksheetDimension(t *testing.T) {
	withBackendWorkingDirectory(t)
	service := NewService(&mockSettingsConfigService{bulkPct: 1})
	entries := make([]bulkTransferTestEntry, 0, 10)
	for i := 1; i <= 10; i++ {
		entries = append(entries, bulkTransferTestEntry{
			employeeID:      uint(i),
			projectID:       1,
			amount:          1_000,
			transactionCode: fmt.Sprintf("TX-%d", i),
		})
	}

	response, err := service.GenerateBulkTransferExcel(context.Background(), newBulkTransferTestData(entries), &dto.ExportBulkTransferRequest{}, "", "", "weekly")
	require.NoError(t, err)

	workbook, err := excelize.OpenReader(bytes.NewReader(response.Data))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workbook.Close()) })
	dimension, err := workbook.GetSheetDimension(constants.MBank_SheetName)
	require.NoError(t, err)
	assert.Equal(t, "A1:F12", dimension)
}

func TestService_GenerateBulkTransferExcel_MultipleReadableWorkbooksPreserveRows(t *testing.T) {
	withBackendWorkingDirectory(t)
	settings := &mockSettingsConfigService{bulkPct: 1}
	service := NewService(settings)
	data := newBulkTransferTestData([]bulkTransferTestEntry{
		{employeeID: 3, projectID: 1, amount: 200_000_000, transactionCode: "TX-3"},
		{employeeID: 1, projectID: 1, amount: 300_000_000, transactionCode: "TX-1"},
		{employeeID: 2, projectID: 1, amount: 200_000_000, transactionCode: "TX-2"},
	})

	response, err := service.GenerateBulkTransferExcel(context.Background(), data, &dto.ExportBulkTransferRequest{}, "", "", "weekly")

	require.NoError(t, err)
	assert.Equal(t, zipContentType, response.ContentType)
	assert.Equal(t, ".zip", response.FileExtension)
	assert.Equal(t, 1, settings.calls)

	archive, err := zip.NewReader(bytes.NewReader(response.Data), int64(len(response.Data)))
	require.NoError(t, err)
	require.Len(t, archive.File, 2)
	assert.Equal(t, "MBank_weekly_part_01.xlsx", archive.File[0].Name)
	assert.Equal(t, "MBank_weekly_part_02.xlsx", archive.File[1].Name)

	seenCodes := make(map[string]int)
	var combinedTotal int64
	for _, entry := range archive.File {
		reader, openErr := entry.Open()
		require.NoError(t, openErr)
		workbook, readErr := excelize.OpenReader(reader)
		require.NoError(t, readErr)
		rows := readOpenWorkbookRows(t, workbook)
		require.NoError(t, workbook.Close())
		require.NoError(t, reader.Close())

		workbookTotal := sumWorkbookRows(rows)
		assert.Less(t, workbookTotal, bulkTransferWorkbookLimit)
		combinedTotal += workbookTotal
		for _, row := range rows {
			seenCodes[row.transactionCode]++
		}
	}

	assert.Equal(t, int64(700_000_000), combinedTotal)
	assert.Equal(t, map[string]int{"TX-1": 1, "TX-2": 1, "TX-3": 1}, seenCodes)
}

type bulkTransferTestEntry struct {
	employeeID      uint
	projectID       uint
	amount          int64
	transactionCode string
}

type workbookRow struct {
	amount          int64
	transactionCode string
}

func newBulkTransferTestData(entries []bulkTransferTestEntry) *BulkTransferData {
	data := &BulkTransferData{
		EmployeeProjectAmounts:    make(map[EmployeeProjectKey]int64, len(entries)),
		EmployeeProjectTimesheets: make(map[EmployeeProjectKey][]uint, len(entries)),
		EmployeeData:              make(map[uint]Employee, len(entries)),
		ProjectData:               make(map[uint]Project, len(entries)),
		TransactionCodes:          make(map[EmployeeProjectKey]string, len(entries)),
	}
	for _, entry := range entries {
		key := EmployeeProjectKey{EmployeeID: entry.employeeID, ProjectID: entry.projectID}
		data.EmployeeProjectAmounts[key] = entry.amount
		data.EmployeeProjectTimesheets[key] = []uint{entry.employeeID}
		data.EmployeeData[entry.employeeID] = Employee{
			ID:                entry.employeeID,
			Fullname:          "Employee " + strconv.FormatUint(uint64(entry.employeeID), 10),
			BankAccountNumber: "ACC" + strconv.FormatUint(uint64(entry.employeeID), 10),
			BankAccountName:   "ACCOUNT NAME",
			Bank:              &Bank{BranchName: "MBANK"},
		}
		data.ProjectData[entry.projectID] = Project{ID: entry.projectID, Name: "Project"}
		data.TransactionCodes[key] = entry.transactionCode
	}
	return data
}

func withBackendWorkingDirectory(t *testing.T) {
	t.Helper()
	originalDirectory, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir("../../../../../"))
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(originalDirectory))
	})
}

func partitionTotals(partitions [][]transferRow) []int64 {
	totals := make([]int64, 0, len(partitions))
	for _, partition := range partitions {
		var total int64
		for _, row := range partition {
			total += row.paymentAmount
		}
		totals = append(totals, total)
	}
	return totals
}

func transferRowKeys(rows []transferRow) []EmployeeProjectKey {
	keys := make([]EmployeeProjectKey, 0, len(rows))
	for _, row := range rows {
		keys = append(keys, row.key)
	}
	return keys
}

func transferRowAmounts(rows []transferRow) []int64 {
	amounts := make([]int64, 0, len(rows))
	for _, row := range rows {
		amounts = append(amounts, row.paymentAmount)
	}
	return amounts
}

func readWorkbookRows(t *testing.T, data []byte) []workbookRow {
	t.Helper()
	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, workbook.Close()) })
	return readOpenWorkbookRows(t, workbook)
}

func readOpenWorkbookRows(t *testing.T, workbook *excelize.File) []workbookRow {
	t.Helper()
	rows := make([]workbookRow, 0)
	for rowIndex := 3; ; rowIndex++ {
		sequence, err := workbook.GetCellValue(constants.MBank_SheetName, fmt.Sprintf("A%d", rowIndex))
		require.NoError(t, err)
		if sequence == "" {
			break
		}
		amountText, err := workbook.GetCellValue(constants.MBank_SheetName, fmt.Sprintf("E%d", rowIndex), excelize.Options{RawCellValue: true})
		require.NoError(t, err)
		amount, err := strconv.ParseInt(amountText, 10, 64)
		require.NoError(t, err)
		transactionCode, err := workbook.GetCellValue(constants.MBank_SheetName, fmt.Sprintf("F%d", rowIndex))
		require.NoError(t, err)
		rows = append(rows, workbookRow{amount: amount, transactionCode: transactionCode})
	}
	return rows
}

func sumWorkbookRows(rows []workbookRow) int64 {
	var total int64
	for _, row := range rows {
		total += row.amount
	}
	return total
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
