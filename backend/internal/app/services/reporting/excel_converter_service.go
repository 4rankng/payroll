package reporting

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"

	"api-server/internal/pkg/excelkit"

	"github.com/xuri/excelize/v2"
)

type ExcelConverterService struct{}

func NewExcelConverterService() *ExcelConverterService {
	return &ExcelConverterService{}
}

// ExcelFile represents a processed Excel file
type ExcelFile struct {
	Data     *excelize.File
	FileName string
	IsXLS    bool
}

// ProcessExcelFile handles XLSX files only
func (s *ExcelConverterService) ProcessExcelFile(fileHeader *multipart.FileHeader) (*ExcelFile, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		_ = file.Close() // Ignore close errors as they are not critical for this operation
	}()

	// Read file content into buffer
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}

	// Open XLSX file directly
	excelFile, err := excelkit.OpenReader(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to open XLSX file: %w", err)
	}

	return &ExcelFile{
		Data:     excelFile,
		FileName: fileHeader.Filename,
		IsXLS:    false,
	}, nil
}

// GetSheetNames returns all sheet names from the Excel file
func (s *ExcelConverterService) GetSheetNames(file *ExcelFile) []string {
	return file.Data.GetSheetList()
}

// GetRowsFromSheet returns all rows from a specific sheet
func (s *ExcelConverterService) GetRowsFromSheet(file *ExcelFile, sheetName string) ([][]string, error) {
	rows, err := file.Data.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to get rows from sheet %s: %w", sheetName, err)
	}
	return rows, nil
}

// SaveAsXLSX saves the Excel file as XLSX format
func (s *ExcelConverterService) SaveAsXLSX(file *ExcelFile, filename string) error {
	return file.Data.SaveAs(filename)
}

// GetBuffer returns the Excel file as a buffer (useful for HTTP responses)
func (s *ExcelConverterService) GetBuffer(file *ExcelFile) (*bytes.Buffer, error) {
	buffer := new(bytes.Buffer)
	if err := file.Data.Write(buffer); err != nil {
		return nil, fmt.Errorf("failed to write Excel file to buffer: %w", err)
	}
	return buffer, nil
}

// GetNewExcelFile creates a new Excel file
func (s *ExcelConverterService) GetNewExcelFile() *ExcelFile {
	f := excelize.NewFile()
	return &ExcelFile{
		Data:     f,
		FileName: "new_file.xlsx",
		IsXLS:    false,
	}
}
