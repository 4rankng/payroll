package excel

import (
	"context"
	"fmt"
	"os"
	"strings"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/excel"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/constants"

	"github.com/xuri/excelize/v2"
)

// Service handles Excel operations for payroll
type Service struct {
	settingsConfig SettingsConfigService
}

// SettingsConfigService interface for settings configuration operations
type SettingsConfigService interface {
	GetPaymentPercentageForSchedule(ctx context.Context, schedule string) float64
}

// BulkTransferData holds aggregated data for bulk transfer export
type BulkTransferData struct {
	EmployeeProjectAmounts    map[EmployeeProjectKey]int64
	EmployeeProjectTimesheets map[EmployeeProjectKey][]uint
	EmployeeData              map[uint]Employee
	ProjectData               map[uint]Project
	TransactionCodes          map[EmployeeProjectKey]string
}

// BulkTransferValidationResult holds results after validating and filtering data
type BulkTransferValidationResult struct {
	ValidData        *BulkTransferData
	SkippedEmployees []SkippedEmployee
	TotalCount       int
	ValidCount       int
	SkippedCount     int
}

// SkippedEmployee represents an employee that was skipped due to missing bank info
type SkippedEmployee struct {
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	Reason       string
}

// EmployeeProjectKey represents a unique employee-project combination
type EmployeeProjectKey struct {
	EmployeeID uint
	ProjectID  uint
}

// Employee represents employee data for export
type Employee struct {
	ID                uint
	Fullname          string
	BankAccountNumber string
	BankAccountName   string
	Bank              *Bank
}

// Bank represents bank information
type Bank struct {
	ID         uint
	BranchName string
	BankCode   string
}

// Project represents project data for export
type Project struct {
	ID   uint
	Name string
}

func NewService(settingsConfig SettingsConfigService) *Service {
	return &Service{
		settingsConfig: settingsConfig,
	}
}

// ValidateAndFilterBulkTransferData validates bank information and filters out invalid entries
func (s *Service) ValidateAndFilterBulkTransferData(data *BulkTransferData) *BulkTransferValidationResult {
	validEmployeeProjectAmounts := make(map[EmployeeProjectKey]int64)
	validEmployeeProjectTimesheets := make(map[EmployeeProjectKey][]uint)
	validEmployeeData := make(map[uint]Employee)
	validProjectData := make(map[uint]Project)
	validTransactionCodes := make(map[EmployeeProjectKey]string)
	var skippedEmployees []SkippedEmployee

	totalCount := len(data.EmployeeProjectAmounts)

	for key, amount := range data.EmployeeProjectAmounts {
		employee := data.EmployeeData[key.EmployeeID]
		project := data.ProjectData[key.ProjectID]

		// Validate bank account information
		if employee.BankAccountNumber == "" {
			skippedEmployees = append(skippedEmployees, SkippedEmployee{
				EmployeeID:   employee.ID,
				EmployeeName: employee.Fullname,
				ProjectID:    project.ID,
				ProjectName:  project.Name,
				Reason:       "Missing bank account number",
			})
			continue
		}

		if employee.BankAccountName == "" {
			skippedEmployees = append(skippedEmployees, SkippedEmployee{
				EmployeeID:   employee.ID,
				EmployeeName: employee.Fullname,
				ProjectID:    project.ID,
				ProjectName:  project.Name,
				Reason:       "Missing bank account name",
			})
			continue
		}

		// If validation passes, include in valid data
		validEmployeeProjectAmounts[key] = amount
		validEmployeeProjectTimesheets[key] = data.EmployeeProjectTimesheets[key]
		validEmployeeData[key.EmployeeID] = employee
		validProjectData[key.ProjectID] = project
		if data.TransactionCodes != nil {
			validTransactionCodes[key] = data.TransactionCodes[key]
		}
	}

	validData := &BulkTransferData{
		EmployeeProjectAmounts:    validEmployeeProjectAmounts,
		EmployeeProjectTimesheets: validEmployeeProjectTimesheets,
		EmployeeData:              validEmployeeData,
		ProjectData:               validProjectData,
		TransactionCodes:          validTransactionCodes,
	}

	return &BulkTransferValidationResult{
		ValidData:        validData,
		SkippedEmployees: skippedEmployees,
		TotalCount:       totalCount,
		ValidCount:       len(validEmployeeProjectAmounts),
		SkippedCount:     len(skippedEmployees),
	}
}

// GetBulkTransferTemplate returns the bulk transfer Excel template as XLSX bytes (MBank template)
func (s *Service) GetBulkTransferTemplate() ([]byte, error) {
	// Check if template file exists
	if _, err := os.Stat(constants.MBankTemplatePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("template file not found at %s", constants.MBankTemplatePath)
	}

	// Read the XLSX template file
	xlsxBytes, err := os.ReadFile(constants.MBankTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file: %w", err)
	}

	return xlsxBytes, nil
}

// GenerateBulkTransferExcel creates XLSX file with bulk transfer data using MBank template
func (s *Service) GenerateBulkTransferExcel(ctx context.Context, data *BulkTransferData, req *dto.ExportBulkTransferRequest, fromDate, toDate, cycle string) (*dto.ExportBulkTransferResponse, error) {
	// First validate and filter the data
	validationResult := s.ValidateAndFilterBulkTransferData(data)

	// Generate MBank Excel file (even if empty, to show template with headers only)
	excelBytes, err := s.generateMBankTransferExcel(ctx, validationResult.ValidData, fromDate, toDate, cycle)
	if err != nil {
		return nil, fmt.Errorf("failed to generate MBank Excel: %w", err)
	}

	// Convert skipped employees to DTO format
	var skippedEmployeeDTOs []dto.SkippedEmployeeInfo
	for _, skipped := range validationResult.SkippedEmployees {
		skippedEmployeeDTOs = append(skippedEmployeeDTOs, dto.SkippedEmployeeInfo{
			EmployeeID:   skipped.EmployeeID,
			EmployeeName: skipped.EmployeeName,
			ProjectID:    skipped.ProjectID,
			ProjectName:  skipped.ProjectName,
			Reason:       skipped.Reason,
		})
	}

	return &dto.ExportBulkTransferResponse{
		Data:             excelBytes,
		Files:            nil, // No longer using multiple files
		SkippedEmployees: skippedEmployeeDTOs,
		TotalEmployees:   validationResult.TotalCount,
		IncludedCount:    validationResult.ValidCount,
		SkippedCount:     validationResult.SkippedCount,
		FromDate:         fromDate,
		ToDate:           toDate,
		Cycle:            cycle,
	}, nil
}

// generateMBankTransferExcel creates MBank format Excel file
func (s *Service) generateMBankTransferExcel(ctx context.Context, data *BulkTransferData, fromDate, toDate, cycle string) ([]byte, error) {
	// Load MBank template
	f, err := excelize.OpenFile(constants.MBankTemplatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open MBank template: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			observability.GetLogger().Warn("failed to close Excel file", "error", closeErr)
		}
	}()

	// Clear existing data from row 3 onwards
	if err := s.clearExistingDataMBank(f); err != nil {
		return nil, fmt.Errorf("failed to clear existing data: %w", err)
	}

	// Fill in data starting from row 3 (rows 1-2 have headers)
	if err := s.fillMBankTransferData(ctx, f, data, fromDate, toDate, cycle); err != nil {
		return nil, fmt.Errorf("failed to fill transfer data: %w", err)
	}

	// Save to buffer (XLSX format) - MBank uses XLSX directly
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buffer.Bytes(), nil
}

// clearExistingDataMBank removes existing data from the MBank Excel template
func (s *Service) clearExistingDataMBank(f *excelize.File) error {
	clearRow := 3
	for {
		cellValue, err := f.GetCellValue(constants.MBank_SheetName, fmt.Sprintf("A%d", clearRow))
		if err != nil {
			return fmt.Errorf("failed to get cell value for clearing: %w", err)
		}

		// If column A is empty, stop clearing
		if cellValue == "" {
			break
		}

		// Clear entire row (columns A through H for MBank, including new tracking column H)
		columns := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
		for _, col := range columns {
			if err := f.SetCellValue(constants.MBank_SheetName, fmt.Sprintf("%s%d", col, clearRow), ""); err != nil {
				return fmt.Errorf("failed to clear cell %s%d: %w", col, clearRow, err)
			}
		}

		clearRow++
	}
	return nil
}

// fillMBankTransferData populates the MBank Excel template with transfer data
func (s *Service) fillMBankTransferData(ctx context.Context, f *excelize.File, data *BulkTransferData, fromDate, toDate, cycle string) error {
	row := 3
	stt := 1

	// Get payment percentage based on payment schedule (weekly or monthly)
	bulkTransferPaymentPercentage := s.settingsConfig.GetPaymentPercentageForSchedule(ctx, cycle)

	for key, totalAmount := range data.EmployeeProjectAmounts {
		employee := data.EmployeeData[key.EmployeeID]

		// Apply payment percentage from configuration
		paymentAmount := int64(float64(totalAmount) * bulkTransferPaymentPercentage)

		// Get transaction code from data
		transactionCode := data.TransactionCodes[key]

		// Get bank branch name with nil check
		var bankBranchName string
		if employee.Bank != nil {
			bankBranchName = employee.Bank.BranchName
		}

		// Set cell values with error checking (MBank format)
		// A3=STT, B3=Account Number, C3=Account Name, D3=Bank Name, E3=Amount, F3=Transaction Code
		cellUpdates := []struct {
			cell  string
			value any
		}{
			{fmt.Sprintf("A%d", row), stt},
			{fmt.Sprintf("B%d", row), employee.BankAccountNumber},
			{fmt.Sprintf("C%d", row), employee.BankAccountName},
			{fmt.Sprintf("D%d", row), bankBranchName},
			{fmt.Sprintf("E%d", row), paymentAmount},
			{fmt.Sprintf("F%d", row), transactionCode},
		}

		for _, update := range cellUpdates {
			if err := f.SetCellValue(constants.MBank_SheetName, update.cell, update.value); err != nil {
				return fmt.Errorf("failed to set cell %s: %w", update.cell, err)
			}
		}

		row++
		stt++
	}

	return nil
}

// ExportPayrollHistoriesToExcel creates Excel file with one sheet per project
func (s *Service) ExportPayrollHistoriesToExcel(ctx context.Context, histories []*domain.PaymentHistory, fromDate, toDate string) ([]byte, error) {
	f := excelize.NewFile()

	// Group by project
	projectGroups := make(map[string][]*domain.PaymentHistory)
	for _, h := range histories {
		projectGroups[h.ProjectName] = append(projectGroups[h.ProjectName], h)
	}

	// Create export service for professional styling
	exportService := excel.NewExportService()

	// Create sheet for each project
	for projectName, records := range projectGroups {
		sheetName := sanitizeSheetName(projectName)
		if _, err := f.NewSheet(sheetName); err != nil {
			return nil, fmt.Errorf("failed to create sheet %s: %w", sheetName, err)
		}

		// Define headers
		headers := []string{"STT", "Nhân viên", "CCCD", "Vị trí", "Số tiền", "Ngày thanh toán"}

		// Write headers with professional styling
		if err := exportService.WriteHeaders(f, sheetName, headers); err != nil {
			return nil, fmt.Errorf("failed to write headers: %w", err)
		}

		// Prepare data for auto-sizing
		var allData [][]interface{}

		// Write data rows with professional styling
		for i, record := range records {
			row := i + 2
			data := []interface{}{
				i + 1, // STT
				record.EmployeeName,
				record.EmployeeCCCD,
				record.Position,
				float64(record.TotalPaidAmount),    // Convert to float for currency formatting
				record.PaidAt.Format("2006-01-02"), // Keep YYYY-MM-DD format as requested
			}

			// Add to data collection for auto-sizing
			allData = append(allData, data)

			// Write row with currency formatting for column 4 (0-based index for "Số tiền")
			if err := exportService.WriteDataRow(f, sheetName, row, data, []int{4}); err != nil {
				return nil, fmt.Errorf("failed to write data row: %w", err)
			}
		}

		// Auto-size columns based on content
		if err := exportService.AutoSizeColumns(f, sheetName, headers, allData); err != nil {
			return nil, fmt.Errorf("failed to auto-size columns: %w", err)
		}
	}

	// Now remove the default Sheet1 after all project sheets are created
	if err := f.DeleteSheet("Sheet1"); err != nil {
		// If Sheet1 doesn't exist, that's fine - check common error messages
		errMsg := err.Error()
		if !strings.Contains(errMsg, "not exist") && !strings.Contains(errMsg, "does not exist") {
			return nil, fmt.Errorf("failed to delete default sheet: %w", err)
		}
	}

	// Write to buffer
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buf.Bytes(), nil
}

// sanitizeSheetName removes invalid characters and limits length
func sanitizeSheetName(name string) string {
	invalid := []string{"\\", "/", "?", "*", "[", "]"}
	for _, ch := range invalid {
		name = strings.ReplaceAll(name, ch, "")
	}
	if len(name) > 31 {
		name = name[:31]
	}
	return name
}
