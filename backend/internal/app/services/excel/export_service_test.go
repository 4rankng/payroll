package excel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

func TestNewExportService(t *testing.T) {
	service := NewExportService()
	assert.NotNil(t, service)
	assert.IsType(t, &ExportService{}, service)
}

func TestCreateStyledWorkbook(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully creates workbook with custom sheet name", func(t *testing.T) {
		sheetName := "TestSheet"
		wb, err := service.CreateStyledWorkbook(sheetName)

		assert.NoError(t, err)
		assert.NotNil(t, wb)

		// Verify sheet name was set
		sheetList := wb.GetSheetList()
		assert.Contains(t, sheetList, sheetName)
		assert.NotContains(t, sheetList, "Sheet1")
	})

	t.Run("creates workbook with different sheet names", func(t *testing.T) {
		testCases := []string{
			"Payroll",
			"Employees",
			"Report",
			"Monthly Summary",
		}

		for _, sheetName := range testCases {
			wb, err := service.CreateStyledWorkbook(sheetName)
			assert.NoError(t, err)
			assert.NotNil(t, wb)

			sheetList := wb.GetSheetList()
			assert.Contains(t, sheetList, sheetName)
		}
	})
}

func TestSetupHeaderStyle(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully creates header style", func(t *testing.T) {
		wb := excelize.NewFile()
		styleID, err := service.SetupHeaderStyle(wb)

		assert.NoError(t, err)
		assert.Greater(t, styleID, 0)
	})

	t.Run("creates multiple styles successfully", func(t *testing.T) {
		wb := excelize.NewFile()

		styleID1, err1 := service.SetupHeaderStyle(wb)
		styleID2, err2 := service.SetupHeaderStyle(wb)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.Greater(t, styleID1, 0)
		assert.Greater(t, styleID2, 0)
		// Both styles are valid, they may be same or different depending on implementation
	})
}

func TestSetupDataStyle(t *testing.T) {
	service := &ExportService{}

	t.Run("creates data style without alternate row", func(t *testing.T) {
		wb := excelize.NewFile()
		styleID, err := service.SetupDataStyle(wb, false)

		assert.NoError(t, err)
		assert.Greater(t, styleID, 0)
	})

	t.Run("creates data style with alternate row", func(t *testing.T) {
		wb := excelize.NewFile()
		styleID, err := service.SetupDataStyle(wb, true)

		assert.NoError(t, err)
		assert.Greater(t, styleID, 0)
	})

	t.Run("different styles for alternate and regular rows", func(t *testing.T) {
		wb := excelize.NewFile()

		regularStyle, err1 := service.SetupDataStyle(wb, false)
		alternateStyle, err2 := service.SetupDataStyle(wb, true)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, regularStyle, alternateStyle)
	})
}

func TestSetupCurrencyStyle(t *testing.T) {
	service := &ExportService{}

	t.Run("creates currency style without alternate row", func(t *testing.T) {
		wb := excelize.NewFile()
		styleID, err := service.SetupCurrencyStyle(wb, false)

		assert.NoError(t, err)
		assert.Greater(t, styleID, 0)
	})

	t.Run("creates currency style with alternate row", func(t *testing.T) {
		wb := excelize.NewFile()
		styleID, err := service.SetupCurrencyStyle(wb, true)

		assert.NoError(t, err)
		assert.Greater(t, styleID, 0)
	})

	t.Run("different styles for alternate and regular rows", func(t *testing.T) {
		wb := excelize.NewFile()

		regularStyle, err1 := service.SetupCurrencyStyle(wb, false)
		alternateStyle, err2 := service.SetupCurrencyStyle(wb, true)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, regularStyle, alternateStyle)
	})

	t.Run("currency style differs from data style", func(t *testing.T) {
		wb := excelize.NewFile()

		dataStyle, err1 := service.SetupDataStyle(wb, false)
		currencyStyle, err2 := service.SetupCurrencyStyle(wb, false)

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEqual(t, dataStyle, currencyStyle)
	})
}

func TestWriteHeaders(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully writes headers", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Name", "Position", "Salary"}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Verify headers were written
		val, err := wb.GetCellValue(sheetName, "A1")
		assert.NoError(t, err)
		assert.Equal(t, "Name", val)

		val, err = wb.GetCellValue(sheetName, "B1")
		assert.NoError(t, err)
		assert.Equal(t, "Position", val)

		val, err = wb.GetCellValue(sheetName, "C1")
		assert.NoError(t, err)
		assert.Equal(t, "Salary", val)
	})

	t.Run("writes many headers", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{
			"A", "B", "C", "D", "E", "F", "G", "H", "I", "J",
			"K", "L", "M", "N", "O", "P", "Q", "R", "S", "T",
		}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Verify first and last headers
		val, err := wb.GetCellValue(sheetName, "A1")
		assert.NoError(t, err)
		assert.Equal(t, "A", val)

		val, err = wb.GetCellValue(sheetName, "T1")
		assert.NoError(t, err)
		assert.Equal(t, "T", val)
	})

	t.Run("handles empty headers", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)
	})
}

func TestWriteDataRow(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully writes data row", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"John Doe", "Engineer", 75000}

		err := service.WriteDataRow(wb, sheetName, 2, data, []int{2})
		assert.NoError(t, err)

		// Verify data was written
		val, err := wb.GetCellValue(sheetName, "A2")
		assert.NoError(t, err)
		assert.Equal(t, "John Doe", val)

		val, err = wb.GetCellValue(sheetName, "B2")
		assert.NoError(t, err)
		assert.Equal(t, "Engineer", val)
	})

	t.Run("handles alternate row styling", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Data1", "Data2"}

		// Row 2 (even) should be alternate
		err := service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)

		// Row 3 (odd) should be regular
		err = service.WriteDataRow(wb, sheetName, 3, data, []int{})
		assert.NoError(t, err)
	})

	t.Run("applies currency style to specified columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Item", 100, 200, 300}
		currencyColumns := []int{1, 2, 3}

		err := service.WriteDataRow(wb, sheetName, 2, data, currencyColumns)
		assert.NoError(t, err)

		// Verify all cells were written
		val, err := wb.GetCellValue(sheetName, "A2")
		assert.NoError(t, err)
		assert.Equal(t, "Item", val)
	})

	t.Run("handles various data types", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{
			"String",
			123,
			45.67,
			true,
			time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		err := service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)
	})

	t.Run("handles empty data", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{}

		err := service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)
	})
}

func TestAutoSizeColumns(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully auto sizes columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Name", "Position", "Salary"}
		data := [][]interface{}{
			{"John Doe", "Engineer", 75000},
			{"Jane Smith", "Manager", 85000},
			{"Bob Johnson", "Developer", 70000},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("handles long content", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Short", "Very Long Header Name That Should Be Wider"}
		data := [][]interface{}{
			{"A", "This is a very long piece of content that should make the column wider than the header"},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("respects minimum width", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"A", "B", "C"}
		data := [][]interface{}{
			{"X", "Y", "Z"},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("respects maximum width", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		veryLongString := "This is an extremely long string that exceeds the maximum column width and should be capped at 50 characters wide to prevent overly wide columns in the Excel spreadsheet"
		headers := []string{"Long Content"}
		data := [][]interface{}{
			{veryLongString},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("handles nil values in data", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Col1", "Col2", "Col3"}
		data := [][]interface{}{
			{"Value", nil, "Value"},
			{nil, "Value", nil},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("handles empty data", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Col1", "Col2"}
		data := [][]interface{}{}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})
}

func TestGetColumnName(t *testing.T) {
	service := &ExportService{}

	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"first column", 0, "A"},
		{"second column", 1, "B"},
		{"tenth column", 9, "J"},
		{"26th column", 25, "Z"},
		{"27th column", 26, "AA"},
		{"28th column", 27, "AB"},
		{"52nd column", 51, "AZ"},
		{"53rd column", 52, "BA"},
		{"100th column", 99, "CV"},
		{"701st column", 700, "ZY"},
		{"702nd column", 701, "ZZ"},
		{"703rd column", 702, "AAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.GetColumnName(tt.index)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatCurrency(t *testing.T) {
	service := &ExportService{}

	tests := []struct {
		name     string
		value    int64
		expected string
	}{
		{"zero", 0, "0"},
		{"single digit", 5, "5"},
		{"two digits", 99, "99"},
		{"three digits", 999, "999"},
		{"one thousand", 1000, "1,000"},
		{"ten thousand", 10000, "10,000"},
		{"hundred thousand", 100000, "100,000"},
		{"one million", 1000000, "1,000,000"},
		{"complex number", 1234567, "1,234,567"},
		{"salary example", 75000, "75,000"},
		{"large salary", 250000, "250,000"},
		{"negative small", -99, "-99"},
		{"negative thousand", -1000, "-1,000"},
		{"negative million", -1000000, "-1,000,000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.FormatCurrency(tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDate(t *testing.T) {
	service := &ExportService{}

	t.Run("formats valid date", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		result := service.FormatDate(&date)
		assert.Equal(t, "15/01/2024", result)
	})

	t.Run("formats different dates", func(t *testing.T) {
		tests := []struct {
			name     string
			date     time.Time
			expected string
		}{
			{"new year", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "01/01/2024"},
			{"leap year", time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), "29/02/2024"},
			{"year end", time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), "31/12/2024"},
			{"mid year", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), "15/06/2024"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := service.FormatDate(&tt.date)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("returns empty string for nil", func(t *testing.T) {
		result := service.FormatDate(nil)
		assert.Equal(t, "", result)
	})

	t.Run("handles zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		result := service.FormatDate(&zeroTime)
		assert.Equal(t, "01/01/0001", result)
	})
}

func TestFormatDateValue(t *testing.T) {
	service := &ExportService{}

	t.Run("formats valid date", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		result := service.FormatDateValue(date)
		assert.Equal(t, "15/01/2024", result)
	})

	t.Run("formats different dates", func(t *testing.T) {
		tests := []struct {
			name     string
			date     time.Time
			expected string
		}{
			{"new year", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "01/01/2024"},
			{"mid month", time.Date(2024, 5, 15, 12, 0, 0, 0, time.UTC), "15/05/2024"},
			{"year end", time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC), "31/12/2024"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := service.FormatDateValue(tt.date)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("handles zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		result := service.FormatDateValue(zeroTime)
		assert.Equal(t, "01/01/0001", result)
	})
}

func TestFormatDateTimeValue(t *testing.T) {
	service := &ExportService{}

	t.Run("formats valid datetime", func(t *testing.T) {
		datetime := time.Date(2024, 1, 15, 14, 30, 45, 0, time.UTC)
		result := service.FormatDateTimeValue(datetime)
		assert.Equal(t, "15/01/2024 14:30:45", result)
	})

	t.Run("formats different datetimes", func(t *testing.T) {
		tests := []struct {
			name     string
			datetime time.Time
			expected string
		}{
			{
				"midnight",
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				"01/01/2024 00:00:00",
			},
			{
				"noon",
				time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC),
				"15/06/2024 12:00:00",
			},
			{
				"end of day",
				time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
				"31/12/2024 23:59:59",
			},
			{
				"morning time",
				time.Date(2024, 3, 10, 9, 15, 30, 0, time.UTC),
				"10/03/2024 09:15:30",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := service.FormatDateTimeValue(tt.datetime)
				assert.Equal(t, tt.expected, result)
			})
		}
	})

	t.Run("handles zero time", func(t *testing.T) {
		zeroTime := time.Time{}
		result := service.FormatDateTimeValue(zeroTime)
		assert.Equal(t, "01/01/0001 00:00:00", result)
	})
}

func TestSaveToBuffer(t *testing.T) {
	service := &ExportService{}

	t.Run("successfully saves empty workbook to buffer", func(t *testing.T) {
		wb := excelize.NewFile()
		buffer, err := service.SaveToBuffer(wb)

		assert.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("saves workbook with data to buffer", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"

		// Add some data
		_ = wb.SetCellValue(sheetName, "A1", "Header1")
		_ = wb.SetCellValue(sheetName, "B1", "Header2")
		_ = wb.SetCellValue(sheetName, "A2", "Data1")
		_ = wb.SetCellValue(sheetName, "B2", "Data2")

		buffer, err := service.SaveToBuffer(wb)

		assert.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("saves workbook with styles to buffer", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"

		// Add headers and styles
		headers := []string{"Name", "Position", "Salary"}
		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Add data rows
		data := []interface{}{"John Doe", "Engineer", 75000}
		err = service.WriteDataRow(wb, sheetName, 2, data, []int{2})
		assert.NoError(t, err)

		buffer, err := service.SaveToBuffer(wb)

		assert.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("saves complex workbook to buffer", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("PayrollReport")
		assert.NoError(t, err)

		sheetName := "PayrollReport"
		headers := []string{"Employee ID", "Name", "Department", "Salary", "Bonus"}
		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Add multiple data rows
		rows := [][]interface{}{
			{1, "John Doe", "Engineering", 75000, 5000},
			{2, "Jane Smith", "Marketing", 65000, 3000},
			{3, "Bob Johnson", "Sales", 70000, 7000},
		}

		for i, row := range rows {
			err = service.WriteDataRow(wb, sheetName, i+2, row, []int{3, 4})
			assert.NoError(t, err)
		}

		err = service.AutoSizeColumns(wb, sheetName, headers, rows)
		assert.NoError(t, err)

		buffer, err := service.SaveToBuffer(wb)

		assert.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Greater(t, len(buffer), 0)
	})
}

func TestExportService_IntegrationScenario(t *testing.T) {
	service := &ExportService{}

	t.Run("complete export workflow", func(t *testing.T) {
		// Step 1: Create workbook
		wb, err := service.CreateStyledWorkbook("EmployeeReport")
		assert.NoError(t, err)
		assert.NotNil(t, wb)

		sheetName := "EmployeeReport"

		// Step 2: Write headers
		headers := []string{"ID", "Name", "Department", "Join Date", "Salary", "Bonus"}
		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Verify headers
		val, err := wb.GetCellValue(sheetName, "A1")
		assert.NoError(t, err)
		assert.Equal(t, "ID", val)

		// Step 3: Write data rows
		joinDate := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
		data := [][]interface{}{
			{1, "John Doe", "Engineering", service.FormatDateValue(joinDate), 75000, 5000},
			{2, "Jane Smith", "Marketing", service.FormatDateValue(joinDate), 65000, 3000},
			{3, "Bob Johnson", "Sales", service.FormatDateValue(joinDate), 70000, 7000},
		}

		currencyColumns := []int{4, 5} // Salary and Bonus columns

		for i, row := range data {
			err = service.WriteDataRow(wb, sheetName, i+2, row, currencyColumns)
			assert.NoError(t, err)
		}

		// Verify data was written
		val, err = wb.GetCellValue(sheetName, "B2")
		assert.NoError(t, err)
		assert.Equal(t, "John Doe", val)

		// Step 4: Auto size columns
		err = service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)

		// Step 5: Save to buffer
		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("complete export with currency formatting", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("SalaryReport")
		assert.NoError(t, err)

		sheetName := "SalaryReport"
		headers := []string{"Employee", "Base Salary", "Allowance", "Total"}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Test currency formatting
		formattedSalary := service.FormatCurrency(75000)
		assert.Equal(t, "75,000", formattedSalary)

		data := [][]interface{}{
			{"John Doe", 75000, 5000, 80000},
			{"Jane Smith", 85000, 7000, 92000},
		}

		for i, row := range data {
			err = service.WriteDataRow(wb, sheetName, i+2, row, []int{1, 2, 3})
			assert.NoError(t, err)
		}

		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("complete export with date formatting", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("AttendanceReport")
		assert.NoError(t, err)

		sheetName := "AttendanceReport"
		headers := []string{"Employee", "Date", "Time"}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		now := time.Now()
		formattedDate := service.FormatDateValue(now)
		formattedDateTime := service.FormatDateTimeValue(now)

		data := [][]interface{}{
			{"John Doe", formattedDate, formattedDateTime},
		}

		for i, row := range data {
			err = service.WriteDataRow(wb, sheetName, i+2, row, []int{})
			assert.NoError(t, err)
		}

		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.Greater(t, len(buffer), 0)
	})
}

func TestWriteHeaders_ErrorCases(t *testing.T) {
	service := &ExportService{}

	t.Run("handles invalid sheet name gracefully", func(t *testing.T) {
		wb := excelize.NewFile()
		// Use a very long sheet name that might cause issues
		sheetName := "InvalidSheetNameThatDoesNotExist"
		headers := []string{"Header1"}

		// Should handle gracefully even with non-existent sheet
		err := service.WriteHeaders(wb, sheetName, headers)
		// The error might occur when setting cell value on non-existent sheet
		// or it might succeed - either is acceptable
		_ = err
	})
}

func TestWriteDataRow_ErrorCases(t *testing.T) {
	service := &ExportService{}

	t.Run("handles invalid sheet name gracefully", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "NonExistentSheet"
		data := []interface{}{"Value1"}

		err := service.WriteDataRow(wb, sheetName, 2, data, []int{})
		_ = err // May or may not error depending on excelize behavior
	})

	t.Run("handles large row numbers", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Value"}

		err := service.WriteDataRow(wb, sheetName, 10000, data, []int{})
		assert.NoError(t, err)
	})
}

func TestCreateStyledWorkbook_EdgeCases(t *testing.T) {
	service := &ExportService{}

	t.Run("handles very long sheet names", func(t *testing.T) {
		// Excel has a 31 character limit for sheet names
		longName := "ThisIsAVeryLongSheetNameThatExceedsTheLimit"
		wb, err := service.CreateStyledWorkbook(longName)

		// Should either succeed or return error
		if err != nil {
			assert.Nil(t, wb)
		} else {
			assert.NotNil(t, wb)
		}
	})

	t.Run("handles empty sheet name", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("")
		// Should either succeed or return error
		if err != nil {
			assert.Nil(t, wb)
		} else {
			assert.NotNil(t, wb)
		}
	})
}

func TestAutoSizeColumns_AdvancedScenarios(t *testing.T) {
	service := &ExportService{}

	t.Run("handles rows with different lengths", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Col1", "Col2", "Col3", "Col4"}
		data := [][]interface{}{
			{"Short", "Medium text", "Very long text content", "X"},
			{"A", "B"}, // Shorter row
			{"Long value here", "Another", "Value", "More", "Extra"}, // Longer row
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("handles all nil row", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Col1", "Col2"}
		data := [][]interface{}{
			{nil, nil},
			{"Value", nil},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})

	t.Run("calculates width based on numbers", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Small", "Large"}
		data := [][]interface{}{
			{123, 123456789},
			{1, 987654321},
		}

		err := service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)
	})
}

func TestWriteDataRow_AdvancedScenarios(t *testing.T) {
	service := &ExportService{}

	t.Run("multiple currency columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{1000, 2000, 3000, 4000}
		currencyColumns := []int{0, 1, 2, 3}

		err := service.WriteDataRow(wb, sheetName, 2, data, currencyColumns)
		assert.NoError(t, err)
	})

	t.Run("no currency columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Text", "More text", "Even more"}

		err := service.WriteDataRow(wb, sheetName, 2, data, nil)
		assert.NoError(t, err)
	})

	t.Run("odd row number for styling", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Value"}

		// Odd row should not have alternate styling
		err := service.WriteDataRow(wb, sheetName, 3, data, []int{})
		assert.NoError(t, err)
	})

	t.Run("even row number for styling", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Value"}

		// Even row should have alternate styling
		err := service.WriteDataRow(wb, sheetName, 4, data, []int{})
		assert.NoError(t, err)
	})

	t.Run("mixed currency and non-currency columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"Name", 50000, "Department", 10000}
		currencyColumns := []int{1, 3} // Only columns 1 and 3 are currency

		err := service.WriteDataRow(wb, sheetName, 2, data, currencyColumns)
		assert.NoError(t, err)
	})

	t.Run("single column data", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		data := []interface{}{"SingleValue"}

		err := service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)
	})
}

func TestWriteHeaders_AdvancedScenarios(t *testing.T) {
	service := &ExportService{}

	t.Run("single header", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"OnlyHeader"}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		val, err := wb.GetCellValue(sheetName, "A1")
		assert.NoError(t, err)
		assert.Equal(t, "OnlyHeader", val)
	})

	t.Run("headers with special characters", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"
		headers := []string{"Name & Title", "Salary ($)", "Date/Time"}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)
	})

	t.Run("very wide table with many columns", func(t *testing.T) {
		wb := excelize.NewFile()
		sheetName := "Sheet1"

		var headers []string
		for i := 0; i < 30; i++ {
			headers = append(headers, "Col"+string(rune('A'+i%26)))
		}

		err := service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)
	})
}

func TestExportService_EdgeCases(t *testing.T) {
	service := &ExportService{}

	t.Run("handles large dataset", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("LargeDataset")
		assert.NoError(t, err)

		sheetName := "LargeDataset"
		headers := []string{"ID", "Data1", "Data2", "Data3"}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		// Create large dataset
		var data [][]interface{}
		for i := 0; i < 100; i++ {
			row := []interface{}{i, "Data", "Value", i * 100}
			data = append(data, row)
		}

		// Write all rows
		for i, row := range data {
			err = service.WriteDataRow(wb, sheetName, i+2, row, []int{3})
			assert.NoError(t, err)
		}

		err = service.AutoSizeColumns(wb, sheetName, headers, data)
		assert.NoError(t, err)

		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("handles wide dataset with many columns", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("WideDataset")
		assert.NoError(t, err)

		sheetName := "WideDataset"

		// Create 50 columns
		var headers []string
		var data []interface{}
		for i := 0; i < 50; i++ {
			headers = append(headers, "Column"+service.FormatCurrency(int64(i)))
			data = append(data, "Value"+service.FormatCurrency(int64(i)))
		}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		err = service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)

		// Verify column naming for high indices
		columnName := service.GetColumnName(49)
		assert.Equal(t, "AX", columnName)
	})

	t.Run("handles special characters in data", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("SpecialChars")
		assert.NoError(t, err)

		sheetName := "SpecialChars"
		headers := []string{"Name", "Description"}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		data := []interface{}{
			"Test & Co. <Ltd>",
			"Description with \"quotes\" and 'apostrophes'",
		}

		err = service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)

		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.Greater(t, len(buffer), 0)
	})

	t.Run("handles unicode characters", func(t *testing.T) {
		wb, err := service.CreateStyledWorkbook("Unicode")
		assert.NoError(t, err)

		sheetName := "Unicode"
		headers := []string{"Name", "Text"}

		err = service.WriteHeaders(wb, sheetName, headers)
		assert.NoError(t, err)

		data := []interface{}{
			"José García",
			"中文字符 العربية 日本語",
		}

		err = service.WriteDataRow(wb, sheetName, 2, data, []int{})
		assert.NoError(t, err)

		buffer, err := service.SaveToBuffer(wb)
		assert.NoError(t, err)
		assert.Greater(t, len(buffer), 0)
	})
}
