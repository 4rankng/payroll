package excel

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"

	"github.com/xuri/excelize/v2"
)

// TimesheetTemplateData represents data needed to generate timesheet template
type TimesheetTemplateData struct {
	Project    *domain.Project
	Employees  []*domain.ProjectEmployee
	FromDate   time.Time
	ToDate     time.Time
	DayTypes   []string // e.g., ["Ngày thường", "Ngày nghỉ"]
	ShiftTypes []string // e.g., ["Ca ngày", "Ca đêm", "Tăng ca"]
}

// GenerateTimesheetTemplate generates XLSX template for timesheet entry
func (s *ExportService) GenerateTimesheetTemplate(data TimesheetTemplateData) (*excelize.File, error) {
	f, err := s.CreateStyledWorkbook("Template")
	if err != nil {
		return nil, err
	}

	sheetName := "Template"
	currentRow := 1

	// Set column widths
	// Column A (STT) - width for 4 digits
	if err := f.SetColWidth(sheetName, "A", "A", 6); err != nil {
		return nil, err
	}
	// Column B (Ngay thang) - width for YYYY-MM-DD (10 characters)
	if err := f.SetColWidth(sheetName, "B", "B", 12); err != nil {
		return nil, err
	}
	// Column C (Loai ngay) - width for dropdown
	if err := f.SetColWidth(sheetName, "C", "C", 15); err != nil {
		return nil, err
	}
	// Remaining columns for shift types
	for i := 0; i < len(data.ShiftTypes); i++ {
		col := s.GetColumnName(3 + i) // Start from column D (index 3)
		if err := f.SetColWidth(sheetName, col, col, 12); err != nil {
			return nil, err
		}
	}

	// Header Section - Project Info
	projectHeader := fmt.Sprintf("Dự án: [%d] %s (%s)",
		data.Project.ID,
		data.Project.Name,
		data.Project.Code)

	// Style for project header (locked)
	projectHeaderStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   14,
			Family: DefaultFontName,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
		},
		Protection: &excelize.Protection{
			Locked: true,
		},
	})
	if err != nil {
		return nil, err
	}

	if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", currentRow), projectHeader); err != nil {
		return nil, err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", currentRow), projectHeaderStyle); err != nil {
		return nil, err
	}
	currentRow += 2 // Skip a row for spacing

	// Generate date range for timesheet
	dates := generateDateRange(data.FromDate, data.ToDate)

	// Employee Section - iterate through each employee
	for _, employee := range data.Employees {
		// Employee header
		employeeHeader := fmt.Sprintf("Nhân viên: [%d] %s (%s)",
			employee.EmployeeID,
			employee.EmployeeName,
			employee.EmployeeCCCD)

		employeeHeaderStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold:   true,
				Size:   12,
				Family: DefaultFontName,
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", currentRow), employeeHeader); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", currentRow), employeeHeaderStyle); err != nil {
			return nil, err
		}
		currentRow++

		// Position row
		positionText := fmt.Sprintf("Vị trí: %s", employee.Position)
		positionStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   11,
				Family: DefaultFontName,
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		if err := f.SetCellValue(sheetName, fmt.Sprintf("B%d", currentRow), positionText); err != nil {
			return nil, err
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("B%d", currentRow), fmt.Sprintf("B%d", currentRow), positionStyle); err != nil {
			return nil, err
		}
		currentRow++

		// Table header
		headerStyle, err := s.SetupHeaderStyle(f)
		if err != nil {
			return nil, err
		}

		// Column headers: STT, Ngày tháng, Loại ngày, and shift types
		headers := []string{"STT", "Ngày tháng", "Loại ngày"}
		headers = append(headers, data.ShiftTypes...)

		for i, header := range headers {
			cell := fmt.Sprintf("%s%d", s.GetColumnName(i), currentRow)
			if err := f.SetCellValue(sheetName, cell, header); err != nil {
				return nil, err
			}
			if err := f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
				return nil, err
			}
		}
		currentRow++

		// Data rows - one row per date (locked for Date column)
		dataStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		dataStyleAlt, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{AlternateRowColor},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		// Style for center alignment (for STT - read-only)
		centerStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		centerStyleAlt, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{AlternateRowColor},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: true,
			},
		})
		if err != nil {
			return nil, err
		}

		// Unlocked styles for editable cells (Day Type dropdown and shift hours)
		dataStyleUnlocked, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return nil, err
		}

		dataStyleUnlockedAlt, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{AlternateRowColor},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return nil, err
		}

		// Unlocked center-aligned styles for shift hour columns
		centerStyleUnlocked, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return nil, err
		}

		centerStyleUnlockedAlt, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Size:   DataFontSize,
				Family: DefaultFontName,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{AlternateRowColor},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "CCCCCC", Style: 1},
				{Type: "top", Color: "CCCCCC", Style: 1},
				{Type: "bottom", Color: "CCCCCC", Style: 1},
				{Type: "right", Color: "CCCCCC", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
			Protection: &excelize.Protection{
				Locked: false,
			},
		})
		if err != nil {
			return nil, err
		}

		for i, date := range dates {
			isAlt := i%2 == 1

			// STT (column A)
			cellSTT := fmt.Sprintf("A%d", currentRow)
			if err := f.SetCellValue(sheetName, cellSTT, i+1); err != nil {
				return nil, err
			}
			if isAlt {
				if err := f.SetCellStyle(sheetName, cellSTT, cellSTT, centerStyleAlt); err != nil {
					return nil, err
				}
			} else {
				if err := f.SetCellStyle(sheetName, cellSTT, cellSTT, centerStyle); err != nil {
					return nil, err
				}
			}

			// Date (column B)
			cellDate := fmt.Sprintf("B%d", currentRow)
			if err := f.SetCellValue(sheetName, cellDate, date.Format("2006-01-02")); err != nil {
				return nil, err
			}
			if isAlt {
				if err := f.SetCellStyle(sheetName, cellDate, cellDate, dataStyleAlt); err != nil {
					return nil, err
				}
			} else {
				if err := f.SetCellStyle(sheetName, cellDate, cellDate, dataStyle); err != nil {
					return nil, err
				}
			}

			// Day Type dropdown (column C) - UNLOCKED for user input
			cellDayType := fmt.Sprintf("C%d", currentRow)
			if err := f.SetCellValue(sheetName, cellDayType, ""); err != nil {
				return nil, err
			}

			// Add data validation (dropdown) for day types
			dvRange := excelize.NewDataValidation(true)
			dvRange.Sqref = cellDayType
			if err := dvRange.SetDropList(data.DayTypes); err != nil {
				return nil, err
			}
			if err := f.AddDataValidation(sheetName, dvRange); err != nil {
				return nil, err
			}

			// Use unlocked style for editable cells
			if isAlt {
				if err := f.SetCellStyle(sheetName, cellDayType, cellDayType, dataStyleUnlockedAlt); err != nil {
					return nil, err
				}
			} else {
				if err := f.SetCellStyle(sheetName, cellDayType, cellDayType, dataStyleUnlocked); err != nil {
					return nil, err
				}
			}

			// Shift hour columns (D, E, F, ...) - UNLOCKED for user input
			for j := range data.ShiftTypes {
				cellShift := fmt.Sprintf("%s%d", s.GetColumnName(3+j), currentRow)
				if err := f.SetCellValue(sheetName, cellShift, ""); err != nil {
					return nil, err
				}
				// Use unlocked style for editable shift hour cells
				if isAlt {
					if err := f.SetCellStyle(sheetName, cellShift, cellShift, centerStyleUnlockedAlt); err != nil {
						return nil, err
					}
				} else {
					if err := f.SetCellStyle(sheetName, cellShift, cellShift, centerStyleUnlocked); err != nil {
						return nil, err
					}
				}
			}

			currentRow++
		}

		// Add spacing between employees
		currentRow += 2
	}

	// Add instruction sheet
	if err := s.addInstructionSheet(f); err != nil {
		return nil, err
	}

	// Protect the Template sheet to enforce cell locking
	// Empty password means users can unprotect if needed, but cells are locked by default
	if err := f.ProtectSheet(sheetName, &excelize.SheetProtectionOptions{
		SelectLockedCells:   true,
		SelectUnlockedCells: true,
		FormatCells:         false,
		FormatColumns:       false,
		FormatRows:          false,
		InsertColumns:       false,
		InsertRows:          false,
		InsertHyperlinks:    false,
		DeleteColumns:       false,
		DeleteRows:          false,
		Sort:                false,
		AutoFilter:          false,
		PivotTables:         false,
	}); err != nil {
		return nil, err
	}

	return f, nil
}

// generateDateRange generates a slice of dates from start to end (inclusive)
func generateDateRange(start, end time.Time) []time.Time {
	var dates []time.Time

	// Normalize to start of day
	current := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	endDate := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())

	for current.Before(endDate) || current.Equal(endDate) {
		dates = append(dates, current)
		current = current.AddDate(0, 0, 1)
	}

	return dates
}

// ExtractDayTypesFromPayrate extracts unique day types from payrate configuration
// Returns sorted list of day types (e.g., ["Ngày lễ", "Ngày nghỉ", "Ngày thường"])
func ExtractDayTypesFromPayrate(payrate *domain.Payrate) ([]string, error) {
	if payrate == nil {
		return []string{"Ngày thường"}, nil
	}

	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return nil, err
	}

	dayTypeMap := make(map[string]bool)

	// Parse flattened paths like: "position.dayType.shiftType"
	// Example: "phổ thông.Ngày thường.Ca ngày" -> extract "Ngày thường"
	for path := range flattened {
		parts := strings.Split(path, ".")
		if len(parts) >= 2 {
			dayType := parts[1] // Second level is day type
			dayTypeMap[dayType] = true
		}
	}

	// Convert to sorted slice
	dayTypes := make([]string, 0, len(dayTypeMap))
	for dayType := range dayTypeMap {
		dayTypes = append(dayTypes, dayType)
	}
	sort.Strings(dayTypes)

	if len(dayTypes) == 0 {
		return []string{"Ngày thường"}, nil
	}

	return dayTypes, nil
}

// ExtractShiftTypesFromPayrate extracts unique shift types from payrate configuration
// Returns sorted list of shift types (e.g., ["Ca ngày", "Ca đêm", "Tăng ca"])
func ExtractShiftTypesFromPayrate(payrate *domain.Payrate) ([]string, error) {
	if payrate == nil {
		return []string{"Ca ngày"}, nil
	}

	flattened, err := payrate.Payrate.Flatten()
	if err != nil {
		return nil, err
	}

	shiftTypeMap := make(map[string]bool)

	// Parse flattened paths like: "position.dayType.shiftType"
	// Example: "phổ thông.Ngày thường.Ca ngày" -> extract "Ca ngày"
	for path := range flattened {
		parts := strings.Split(path, ".")
		if len(parts) >= 3 {
			shiftType := parts[2] // Third level is shift type
			shiftTypeMap[shiftType] = true
		}
	}

	// Convert to sorted slice
	shiftTypes := make([]string, 0, len(shiftTypeMap))
	for shiftType := range shiftTypeMap {
		shiftTypes = append(shiftTypes, shiftType)
	}
	sort.Strings(shiftTypes)

	if len(shiftTypes) == 0 {
		return []string{"Ca ngày"}, nil
	}

	return shiftTypes, nil
}

// addInstructionSheet adds a sheet with instructions in Vietnamese
func (s *ExportService) addInstructionSheet(f *excelize.File) error {
	sheetName := "Hướng dẫn nhập công"

	// Create new sheet
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return err
	}

	// Set as active sheet (make it visible first)
	f.SetActiveSheet(index)

	// Set back to Template sheet as active
	templateIndex, err := f.GetSheetIndex("Template")
	if err != nil {
		return err
	}
	f.SetActiveSheet(templateIndex)

	// Set column widths
	if err := f.SetColWidth(sheetName, "A", "A", 80); err != nil {
		return err
	}

	// Title style
	titleStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   16,
			Family: DefaultFontName,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	// Heading style
	headingStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   12,
			Family: DefaultFontName,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "center",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	// Content style
	contentStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Size:   11,
			Family: DefaultFontName,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "left",
			Vertical:   "top",
			WrapText:   true,
		},
	})
	if err != nil {
		return err
	}

	currentRow := 1

	// Title
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "HƯỚNG DẪN NHẬP CÔNG"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), titleStyle); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheetName, currentRow, 30); err != nil {
		return err
	}
	currentRow += 2

	// Introduction
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "1. GIỚI THIỆU"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), headingStyle); err != nil {
		return err
	}
	currentRow++

	introText := "File Excel này được sử dụng để nhập công cho nhân viên trong dự án. Mỗi nhân viên sẽ có một bảng riêng để ghi nhận số giờ làm việc theo từng ngày trong khoảng thời gian đã chọn."
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), introText); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), contentStyle); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheetName, currentRow, 35); err != nil {
		return err
	}
	currentRow += 2

	// How to fill
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "2. CÁCH NHẬP CÔNG"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), headingStyle); err != nil {
		return err
	}
	currentRow++

	instructions := []string{
		"• Mỗi nhân viên có một bảng riêng gồm:",
		"  - Dòng 1: [Mã nhân viên] Tên nhân viên (CCCD)",
		"  - Dòng 2: Vị trí làm việc của nhân viên",
		"  - Dòng 3 trở đi: Bảng chấm công theo ngày",
		"• Cột 'STT': Số thứ tự tự động, không cần chỉnh sửa",
		"• Cột 'Ngày tháng': Ngày làm việc (định dạng YYYY-MM-DD), không cần chỉnh sửa",
		"• Cột 'Loại ngày': Chọn loại ngày từ danh sách dropdown (ví dụ: Ngày thường, Ngày nghỉ, Ngày lễ)",
		"• Các cột tiếp theo (Ca ngày, Ca đêm, Tăng ca...): Nhập số giờ làm việc cho từng ca",
	}

	for _, instruction := range instructions {
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), instruction); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), contentStyle); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheetName, currentRow, 20); err != nil {
			return err
		}
		currentRow++
	}
	currentRow++

	// Important notes
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "3. LƯU Ý QUAN TRỌNG"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), headingStyle); err != nil {
		return err
	}
	currentRow++

	notes := []string{
		"• Chỉ nhập số giờ vào các cột ca làm việc (Ca ngày, Ca đêm, Tăng ca...)",
		"• Số giờ có thể là số nguyên hoặc số thập phân (ví dụ: 8, 8.5, 4.25)",
		"• Bắt buộc phải chọn 'Loại ngày' trước khi nhập số giờ",
		"• Nếu nhân viên nghỉ trong ngày, để trống các cột số giờ hoặc nhập 0",
		"• Không chỉnh sửa cột STT và Ngày tháng",
		"• Không xóa hoặc thêm dòng vào bảng",
		"• Sau khi nhập xong, lưu file và gửi lại cho quản lý để import vào hệ thống",
	}

	for _, note := range notes {
		if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), note); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), contentStyle); err != nil {
			return err
		}
		if err := f.SetRowHeight(sheetName, currentRow, 20); err != nil {
			return err
		}
		currentRow++
	}
	currentRow++

	// Example
	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), "4. VÍ DỤ"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), headingStyle); err != nil {
		return err
	}
	currentRow++

	exampleText := "Ví dụ bảng công của nhân viên Nguyễn Văn A:\n\n" +
		"Nhân viên: [1] Nguyen Van A (0330123456)\n" +
		"Vị trí: Phổ thông\n\n" +
		"STT | Ngày tháng | Loại ngày   | Ca ngày | Ca đêm | Tăng ca\n" +
		"1   | 2025-01-01 | Ngày thường |    8    |   0    |    0\n" +
		"2   | 2025-01-02 | Ngày thường |    8    |   0    |    2\n" +
		"3   | 2025-01-03 | (trống)     |    0    |   0    |    0\n\n" +
		"Cách nhập:\n" +
		"- Dòng 1: Chọn 'Ngày thường' ở cột 'Loại ngày', nhập 8 vào 'Ca ngày'\n" +
		"- Dòng 2: Chọn 'Ngày thường', nhập 8 vào 'Ca ngày', nhập 2 vào 'Tăng ca'\n" +
		"- Dòng 3: Nhân viên nghỉ, để trống 'Loại ngày' hoặc nhập 0 vào các cột ca"

	if err := f.SetCellValue(sheetName, fmt.Sprintf("A%d", currentRow), exampleText); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetName, fmt.Sprintf("A%d", currentRow), fmt.Sprintf("A%d", currentRow), contentStyle); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheetName, currentRow, 200); err != nil {
		return err
	}

	return nil
}
