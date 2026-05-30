package advance_payment

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/xuri/excelize/v2"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
)

// ExportFlexPayEmployees generates an Excel file containing the list of employees under flexible payment
func (s *Service) ExportFlexPayEmployees(ctx context.Context, forMonth string) (*excelize.File, error) {
	// If forMonth is empty, get the latest available month from the database
	if forMonth == "" {
		latestMonth, err := s.config.AdvancePaymentRepo.GetLatestForMonth(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest for_month")
		}
		forMonth = latestMonth
	}

	filters := domain.EmployeeAdvanceStatsFilters{
		ForMonth:  &forMonth,
		SortBy:    "e.fullname",
		SortOrder: "ASC",
		Limit:     1000000, // No cap for export
		Offset:    0,
	}

	employees, _, _, err := s.GetEmployeeAdvanceStats(ctx, filters)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get employee advance stats for export")
	}

	if len(employees) == 0 {
		return nil, domain.NewValidationError(constants.MsgNoEmployeesThisMonthVN)
	}

	// Deduplicate employees by CCCD (keep first occurrence)
	seen := make(map[string]bool)
	uniqueEmployees := make([]*domain.EmployeeAdvanceStats, 0, len(employees))
	for _, emp := range employees {
		if !seen[emp.CCCD] {
			seen[emp.CCCD] = true
			uniqueEmployees = append(uniqueEmployees, emp)
		}
	}
	s.logger.Info("deduplicated employees for export", "before", len(employees), "after", len(uniqueEmployees))
	employees = uniqueEmployees

	f := excelize.NewFile()
	sheet := "DS"
	index, err := f.NewSheet(sheet)
	if err != nil {
		_ = f.Close()
		return nil, errors.Wrap(err, "failed to create sheet")
	}
	f.SetActiveSheet(index)
	_ = f.DeleteSheet("Sheet1")

	// Title and Subtitle
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "1F497D"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	subtitleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Size: 12, Color: "595959"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	_ = f.SetCellValue(sheet, "A1", "DANH SÁCH NHÂN VIÊN ỨNG LƯƠNG LINH HOẠT")
	_ = f.MergeCell(sheet, "A1", "J1")
	_ = f.SetCellStyle(sheet, "A1", "J1", titleStyle)
	_ = f.SetRowHeight(sheet, 1, 24)

	_ = f.SetCellValue(sheet, "A2", fmt.Sprintf("Tháng: %s", forMonth))
	_ = f.MergeCell(sheet, "A2", "J2")
	_ = f.SetCellStyle(sheet, "A2", "J2", subtitleStyle)
	_ = f.SetRowHeight(sheet, 2, 18)

	headers := []string{"STT", "Họ và tên", "Mã dự án", "Tên đăng nhập", "Email", "CCCD", "Số điện thoại", "Ngân hàng", "Số tài khoản", "Ngày tạo"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		_ = f.SetCellValue(sheet, cell, h)
	}

	// Styles for borders and alignment
	borderParams := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1},
		{Type: "right", Color: "000000", Style: 1},
		{Type: "bottom", Color: "000000", Style: 1},
	}

	// Style header row
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2F75B5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borderParams,
	})
	_ = f.SetCellStyle(sheet, "A3", "J3", headerStyle)
	_ = f.SetRowHeight(sheet, 3, 22)

	// Freeze header pane (below row 3)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	centerDataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    borderParams,
	})
	leftDataStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    borderParams,
	})

	for i, emp := range employees {
		row := i + 4
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i+1)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), emp.Fullname)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), emp.ProjectCode)
		_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), emp.Username)
		emailStr := ""
		if emp.Email != nil {
			emailStr = *emp.Email
		}
		_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), emailStr)
		_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), emp.CCCD)
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), emp.Mobile)
		bankNameStr := ""
		if emp.BankName != nil {
			bankNameStr = *emp.BankName
		}
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), bankNameStr)
		_ = f.SetCellValue(sheet, fmt.Sprintf("I%d", row), emp.BankAccountNumber)
		if emp.CreatedAt != nil {
			_ = f.SetCellValue(sheet, fmt.Sprintf("J%d", row), emp.CreatedAt.Format("02/01/2006"))
		}
		_ = f.SetRowHeight(sheet, row, 20)

		// Apply styles
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), centerDataStyle) // STT
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("E%d", row), leftDataStyle)   // Name, Project, Username, Email
		_ = f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("G%d", row), centerDataStyle) // CCCD, Mobile
		_ = f.SetCellStyle(sheet, fmt.Sprintf("H%d", row), fmt.Sprintf("I%d", row), leftDataStyle)   // Bank name, Account number
		_ = f.SetCellStyle(sheet, fmt.Sprintf("J%d", row), fmt.Sprintf("J%d", row), centerDataStyle) // Date
	}

	// Auto-fit column widths
	_ = f.SetColWidth(sheet, "A", "A", 6)
	_ = f.SetColWidth(sheet, "B", "B", 25)
	_ = f.SetColWidth(sheet, "C", "C", 18)
	_ = f.SetColWidth(sheet, "D", "D", 20)
	_ = f.SetColWidth(sheet, "E", "E", 25)
	_ = f.SetColWidth(sheet, "F", "F", 18)
	_ = f.SetColWidth(sheet, "G", "G", 15)
	_ = f.SetColWidth(sheet, "H", "H", 20)
	_ = f.SetColWidth(sheet, "I", "I", 20)
	_ = f.SetColWidth(sheet, "J", "J", 14)

	s.logger.Info("exported flex pay employee list", "for_month", forMonth, "total", len(employees))

	return f, nil
}

// ImportFlexibleEmployeeList imports an Excel file containing employees under flexible payment schedule
func (s *Service) ImportFlexibleEmployeeList(ctx context.Context, file *excelize.File, createdBy uint) (*dto.ImportFlexibleEmployeeListResult, error) {
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("no sheets found in Excel file")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, errors.Wrap(err, "failed to read Excel file")
	}

	result := &dto.ImportFlexibleEmployeeListResult{}
	var errors []dto.ImportRowError

	// Data starts at row 2 (index 1), column mapping:
	// B(1): CCCD, C(2): Project Name, D(3): Position, E(4): Mobile, F(5): Full Name
	// G(6): Bank Account Name, H(7): Bank Account Number, I(8): Bank Name

	for i := 1; i < len(rows); i++ {
		row := rows[i]

		// Get CCCD from column B (index 1)
		cccd := strings.TrimSpace(row[1])
		if cccd == "" {
			break // Stop at empty row
		}

		// Ensure row has enough columns
		if len(row) < 9 {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: "Không đủ cột dữ liệu",
			})
			continue
		}

		result.TotalRows++

		projectName := strings.TrimSpace(row[2])
		position := strings.TrimSpace(row[3])
		mobile := strings.TrimSpace(row[4])
		fullName := strings.TrimSpace(row[5])
		bankAccountName := strings.TrimSpace(row[6])
		bankAccountNumber := strings.TrimSpace(row[7])
		bankName := strings.TrimSpace(row[8])

		// Validate required fields
		if projectName == "" {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: "Tên dự án không được để trống",
			})
			continue
		}

		if fullName == "" {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: "Tên nhân viên không được để trống",
			})
			continue
		}

		// Get or create project (use project name as code)
		project, projectCreated, err := s.getOrCreateProject(ctx, projectName, createdBy)
		if err != nil {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: fmt.Sprintf("Lỗi tạo dự án: %v", err),
			})
			continue
		}
		if projectCreated {
			result.ProjectsCreated++
		} else {
			result.ProjectsSkipped++
		}

		// Get or create employee with mobile support
		employee, employeeCreated, employeeUpdated, err := s.getOrCreateEmployee(ctx, cccd, fullName, bankAccountNumber, bankAccountName, bankName, mobile, createdBy)
		if err != nil {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: fmt.Sprintf("Lỗi tạo nhân viên: %v", err),
			})
			continue
		}
		if employeeCreated {
			result.EmployeesCreated++
		} else if employeeUpdated {
			result.EmployeesUpdated++
		} else {
			result.EmployeesSkipped++
		}

		// Get or create assignment
		assignmentCreated, err := s.getOrCreateAssignment(ctx, project.ID, employee.ID, cccd, fullName, position, createdBy)
		if err != nil {
			errors = append(errors, dto.ImportRowError{
				Row:   i + 1,
				CCCD:  cccd,
				Error: fmt.Sprintf("Lỗi tạo phân công: %v", err),
			})
			continue
		}
		if assignmentCreated {
			result.AssignmentsCreated++
		} else {
			result.AssignmentsSkipped++
		}
	}

	result.Errors = errors

	s.logger.Info("imported flexible employee list",
		"total_rows", result.TotalRows,
		"employees_created", result.EmployeesCreated,
		"employees_updated", result.EmployeesUpdated,
		"projects_created", result.ProjectsCreated,
		"assignments_created", result.AssignmentsCreated,
		"errors", len(errors),
	)

	return result, nil
}
