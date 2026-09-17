package main

import (
	"os"
	"testing"

	excelparser "api-server/internal/app/services/excel"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestWeeklyPaymentFixtureHasCompleteImportPrerequisites(t *testing.T) {
	name, err := buildWeeklyPaymentFixture()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(name) })
	f, err := excelize.OpenFile(name)
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	format, err := excelparser.DetectFormat(f)
	require.NoError(t, err)
	require.Equal(t, excelparser.FormatWeeklyPayment, format.Format)
	parsed, err := excelparser.ParseWeeklyPaymentFile(f, format.WeeklyPaymentSheets, "2026-07")
	require.NoError(t, err)
	require.Len(t, parsed.Sheets, 1)
	require.Len(t, parsed.Sheets[0].Employees, 1)
	employee := parsed.Sheets[0].Employees[0]
	require.NotEmpty(t, employee.EmployeeCode)
	require.Len(t, employee.Entries, 2)
	require.Equal(t, "HC", employee.Entries[0].ShiftKey)
	require.Equal(t, "NN", employee.Entries[1].ShiftKey)
	banks, err := excelparser.ParseSTKSheet(f)
	require.NoError(t, err)
	require.Len(t, banks, 1)
	require.Equal(t, employee.EmployeeCode, banks[0].CCCD)
	require.NotEmpty(t, banks[0].BankAccount)
	_, err = excelparser.ParseWeeklyPaymentFile(f, format.WeeklyPaymentSheets, "2026-02")
	require.Error(t, err)
}
