package excel

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// ExportService handles Excel export operations with professional styling
type ExportService struct{}

// NewExportService creates a new Excel export service
func NewExportService() *ExportService {
	return &ExportService{}
}

// ExcelColor constants for professional styling
const (
	HeaderBackgroundColor = "1F4788" // Dark blue
	HeaderFontColor       = "FFFFFF" // White
	AlternateRowColor     = "F2F2F2" // Light gray
	DefaultFontName       = "Calibri"
	HeaderFontSize        = 12
	DataFontSize          = 11
)

// CreateStyledWorkbook creates a new workbook with professional styling
func (s *ExportService) CreateStyledWorkbook(sheetName string) (*excelize.File, error) {
	f := excelize.NewFile()

	// Rename default Sheet1 to desired sheet name
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		return nil, fmt.Errorf("failed to set sheet name: %w", err)
	}

	return f, nil
}

// SetupHeaderStyle creates and returns header style ID
func (s *ExportService) SetupHeaderStyle(f *excelize.File) (int, error) {
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   HeaderFontSize,
			Family: DefaultFontName,
			Color:  HeaderFontColor,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{HeaderBackgroundColor},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Protection: &excelize.Protection{
			Locked: true,
		},
	})

	return headerStyle, err
}

// SetupDataStyle creates and returns data style ID
func (s *ExportService) SetupDataStyle(f *excelize.File, isAlternateRow bool) (int, error) {
	fillColor := ""
	if isAlternateRow {
		fillColor = AlternateRowColor
	}

	style := &excelize.Style{
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
			Vertical: "center",
		},
	}

	if fillColor != "" {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{fillColor},
			Pattern: 1,
		}
	}

	return f.NewStyle(style)
}

// SetupCurrencyStyle creates and returns currency style ID with Vietnamese format
func (s *ExportService) SetupCurrencyStyle(f *excelize.File, isAlternateRow bool) (int, error) {
	fillColor := ""
	if isAlternateRow {
		fillColor = AlternateRowColor
	}

	style := &excelize.Style{
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
		CustomNumFmt: stringPtr(`#,##0 ₫`), // Vietnamese format with comma as thousand separators: 3,712,000 ₫
	}

	if fillColor != "" {
		style.Fill = excelize.Fill{
			Type:    "pattern",
			Color:   []string{fillColor},
			Pattern: 1,
		}
	}

	return f.NewStyle(style)
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// WriteHeaders writes headers to the Excel file
func (s *ExportService) WriteHeaders(f *excelize.File, sheetName string, headers []string) error {
	headerStyle, err := s.SetupHeaderStyle(f)
	if err != nil {
		return fmt.Errorf("failed to create header style: %w", err)
	}

	for i, header := range headers {
		cell := fmt.Sprintf("%s1", s.GetColumnName(i))
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header value: %w", err)
		}
		if err := f.SetCellStyle(sheetName, cell, cell, headerStyle); err != nil {
			return fmt.Errorf("failed to set header style: %w", err)
		}
	}

	return nil
}

// WriteDataRow writes a data row to the Excel file
func (s *ExportService) WriteDataRow(f *excelize.File, sheetName string, rowNum int, data []interface{}, currencyColumns []int) error {
	isAlternateRow := rowNum%2 == 0

	dataStyle, err := s.SetupDataStyle(f, isAlternateRow)
	if err != nil {
		return fmt.Errorf("failed to create data style: %w", err)
	}

	currencyStyle, err := s.SetupCurrencyStyle(f, isAlternateRow)
	if err != nil {
		return fmt.Errorf("failed to create currency style: %w", err)
	}

	// Build O(1) lookup set once before the loop
	currencySet := make(map[int]bool, len(currencyColumns))
	for _, col := range currencyColumns {
		currencySet[col] = true
	}

	for i, value := range data {
		cell := fmt.Sprintf("%s%d", s.GetColumnName(i), rowNum)

		// Set cell value
		if err := f.SetCellValue(sheetName, cell, value); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}

		// Apply appropriate style
		if currencySet[i] {
			if err := f.SetCellStyle(sheetName, cell, cell, currencyStyle); err != nil {
				return fmt.Errorf("failed to set currency style: %w", err)
			}
		} else {
			if err := f.SetCellStyle(sheetName, cell, cell, dataStyle); err != nil {
				return fmt.Errorf("failed to set data style: %w", err)
			}
		}
	}

	return nil
}

// AutoSizeColumns automatically adjusts column widths based on content
func (s *ExportService) AutoSizeColumns(f *excelize.File, sheetName string, headers []string, data [][]interface{}) error {
	for i, header := range headers {
		columnName := s.GetColumnName(i)
		maxWidth := float64(len(header))

		// Check data rows for maximum width
		for _, row := range data {
			if i < len(row) && row[i] != nil {
				cellValue := fmt.Sprintf("%v", row[i])
				if float64(len(cellValue)) > maxWidth {
					maxWidth = float64(len(cellValue))
				}
			}
		}

		// Add padding and set minimum/maximum widths
		maxWidth = maxWidth*1.2 + 2 // Add 20% padding + 2 for margins
		if maxWidth < 10 {
			maxWidth = 10 // Minimum width
		}
		if maxWidth > 50 {
			maxWidth = 50 // Maximum width to prevent overly wide columns
		}

		if err := f.SetColWidth(sheetName, columnName, columnName, maxWidth); err != nil {
			return fmt.Errorf("failed to set column width: %w", err)
		}
	}

	return nil
}

// GetColumnName converts column index to Excel column name (A, B, C, ..., AA, AB, etc.)
func (s *ExportService) GetColumnName(index int) string {
	result := ""
	for index >= 0 {
		result = string(rune('A'+index%26)) + result
		index = index/26 - 1
	}
	return result
}

// FormatCurrency formats an integer value as currency with thousand separators
func (s *ExportService) FormatCurrency(value int64) string {
	str := strconv.FormatInt(value, 10)

	// Add thousand separators
	if len(str) <= 3 {
		return str
	}

	var result strings.Builder
	for i, digit := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String()
}

// FormatDate formats a time.Time to DD/MM/YYYY format
func (s *ExportService) FormatDate(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("02/01/2006")
}

// FormatDateValue formats a time.Time to DD/MM/YYYY format, handling nil values
func (s *ExportService) FormatDateValue(t time.Time) string {
	return t.Format("02/01/2006")
}

// FormatDateTimeValue formats a time.Time to DD/MM/YYYY HH:MM:SS format
func (s *ExportService) FormatDateTimeValue(t time.Time) string {
	return t.Format("02/01/2006 15:04:05")
}

// SaveToBuffer saves the Excel file to a byte buffer
func (s *ExportService) SaveToBuffer(f *excelize.File) ([]byte, error) {
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("failed to write Excel to buffer: %w", err)
	}

	return buffer.Bytes(), nil
}
