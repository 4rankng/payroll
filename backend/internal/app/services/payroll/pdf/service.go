package pdf

import (
	"fmt"
	"strings"
	"time"

	"github.com/signintech/gopdf"
)

// Service handles PDF generation for payroll
type Service struct {
	fontPathRegular string
	fontPathBold    string
	fontPathMedium  string
}

// Approximate ascent ratio for typical TTF fonts (e.g., Roboto)
const ascentFactor = 0.8

// Small visual fine-tune to pull text slightly upward across fonts
const baselineFineTune = -5.0

// NewService creates a new PDF service
func NewService(fontPathRegular string) *Service {
	return &Service{
		fontPathRegular: fontPathRegular,
		fontPathBold:    "fonts/Roboto-Bold.ttf",
		fontPathMedium:  "fonts/Roboto-Medium.ttf",
	}
}

// BulkTransferHistoryData represents data for PDF generation
type BulkTransferHistoryData struct {
	Filename     string
	TotalTxn     int
	CompletedTxn int
	FailedTxn    int
	Items        []BulkTransferHistoryItem
}

// BulkTransferHistoryItem represents a single row in the PDF
type BulkTransferHistoryItem struct {
	Row                   int
	EmployeeName          string
	EmployeeBank          string
	EmployeeAccountNumber string
	EmployeeCCCD          string
	Amount                string
	PaymentStatus         string
	PaidAt                *string
}

// Column definition
type column struct {
	x     float64
	width float64
	name  string
}

// GenerateBulkTransferHistoryPDF generates a PDF for bulk transfer history
func (s *Service) GenerateBulkTransferHistoryPDF(data BulkTransferHistoryData, currentDate time.Time) ([]byte, error) {
	pdf := gopdf.GoPdf{}
	// Use landscape orientation (A4 rotated)
	pdf.Start(gopdf.Config{
		PageSize: gopdf.Rect{W: 842, H: 595}, // A4 landscape: 842 x 595 points
	})
	pdf.AddPage()

	// Add Vietnamese-supporting TTF fonts with different weights
	if err := pdf.AddTTFFont("Roboto-Regular", s.fontPathRegular); err != nil {
		return nil, fmt.Errorf("failed to add regular font: %w", err)
	}
	if err := pdf.AddTTFFont("Roboto-Bold", s.fontPathBold); err != nil {
		return nil, fmt.Errorf("failed to add bold font: %w", err)
	}
	if err := pdf.AddTTFFont("Roboto-Medium", s.fontPathMedium); err != nil {
		return nil, fmt.Errorf("failed to add medium font: %w", err)
	}

	// Set bold font for title
	if err := pdf.SetFont("Roboto-Bold", "", 18); err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	// Title - centered
	pdf.SetX(320)
	pdf.SetY(40)
	if err := pdf.Cell(nil, "KẾT QUẢ CHUYỂN TIỀN"); err != nil {
		return nil, fmt.Errorf("failed to write title: %w", err)
	}

	// Define table columns - total width 802pt (842 - 40pt margins)
	columns := []column{
		{x: 20, width: 40, name: "STT"},
		{x: 60, width: 140, name: "Họ tên"},
		{x: 200, width: 180, name: "Ngân hàng"},
		{x: 380, width: 100, name: "Số TK"},
		{x: 480, width: 100, name: "CCCD"},
		{x: 580, width: 70, name: "Số tiền (đ)"},
		{x: 650, width: 62, name: "Trạng thái"},
		{x: 712, width: 80, name: "Ngày trả"},
	}

	const tableStartY = 100
	const headerHeight = 20
	const cellPadding = 3
	const fontSize = 8
	const headerFontSize = 9

	// Draw table header
	if err := s.drawTableHeader(&pdf, columns, tableStartY, headerHeight, cellPadding, headerFontSize); err != nil {
		return nil, err
	}

	// Set regular font for table content
	if err := pdf.SetFont("Roboto-Regular", "", fontSize); err != nil {
		return nil, fmt.Errorf("failed to set font: %w", err)
	}

	// Draw table rows
	currentY := float64(tableStartY + headerHeight)
	rowNum := 1 // Start STT from 1

	for _, item := range data.Items {
		// Prepare row data
		statusText := "Thất bại"
		if item.PaymentStatus == "paid" {
			statusText = "Đã trả"
		}

		paidAtText := "-"
		if item.PaidAt != nil && len(*item.PaidAt) >= 10 {
			// Extract date part (YYYY-MM-DD) from any datetime format
			dateStr := (*item.PaidAt)[:10]
			if t, err := time.Parse("2006-01-02", dateStr); err == nil {
				paidAtText = t.Format("02/01/2006")
			}
		}

		rowData := []string{
			fmt.Sprintf("%d", rowNum), // Use sequential numbering starting from 1
			item.EmployeeName,
			item.EmployeeBank,
			item.EmployeeAccountNumber,
			item.EmployeeCCCD,
			item.Amount,
			statusText,
			paidAtText,
		}

		// Calculate row height based on wrapped text
		rowHeight := s.calculateRowHeight(&pdf, columns, rowData, cellPadding, fontSize)

		// Check if we need a new page
		if currentY+rowHeight > 560 {
			pdf.AddPage()
			currentY = 50.0

			// Redraw header on new page
			if err := s.drawTableHeader(&pdf, columns, currentY, headerHeight, cellPadding, headerFontSize); err != nil {
				return nil, err
			}
			currentY += headerHeight

			// Reset font for content
			if err := pdf.SetFont("Roboto-Regular", "", fontSize); err != nil {
				return nil, fmt.Errorf("failed to set font: %w", err)
			}
		}

		// Draw row
		if err := s.drawTableRow(&pdf, columns, rowData, currentY, rowHeight, cellPadding, fontSize); err != nil {
			return nil, err
		}

		currentY += rowHeight
		rowNum++ // Increment for next row
	}

	// Get PDF bytes
	return pdf.GetBytesPdf(), nil
}

// baselineShift returns how far we should push the baseline down from the
// top of a single text line so the visual glyph box is centered.
// We also compensate for extra leading between lines.
func baselineShift(fontSize, lineHeight float64) float64 {
	ascent := ascentFactor * fontSize
	leading := lineHeight - fontSize
	return ascent + (leading / 2.0) + baselineFineTune
}

// drawTableHeader draws the table header with borders and vertical centering
func (s *Service) drawTableHeader(pdf *gopdf.GoPdf, columns []column, y, height, padding, fontSize float64) error {
	if err := pdf.SetFont("Roboto-Medium", "", fontSize); err != nil {
		return fmt.Errorf("failed to set header font: %w", err)
	}

	lineHeight := fontSize * 1.4
	baseShift := baselineShift(fontSize, lineHeight)

	for _, col := range columns {
		pdf.SetLineWidth(0.5)
		pdf.SetStrokeColor(0, 0, 0)
		pdf.RectFromUpperLeftWithStyle(col.x, y, col.width, height, "D")

		textW, _ := pdf.MeasureTextWidth(col.name)
		textX := col.x + (col.width-textW)/2
		textY := y + (height-lineHeight)/2 + baseShift

		pdf.SetX(textX)
		pdf.SetY(textY)
		if err := pdf.Cell(nil, col.name); err != nil {
			return fmt.Errorf("failed to write header: %w", err)
		}
	}
	return nil
}

// drawTableRow draws a table row with borders and vertically centered wrapped text
func (s *Service) drawTableRow(pdf *gopdf.GoPdf, columns []column, rowData []string, y, height, padding, fontSize float64) error {
	for _, col := range columns {
		pdf.SetLineWidth(0.5)
		pdf.SetStrokeColor(0, 0, 0)
		pdf.RectFromUpperLeftWithStyle(col.x, y, col.width, height, "D")
	}

	lineHeight := fontSize * 1.6

	for i, col := range columns {
		if i >= len(rowData) {
			continue
		}

		text := rowData[i]
		cellW := col.width - 2*padding

		lines := s.wrapText(pdf, text, cellW, fontSize)
		totalTextHeight := float64(len(lines)) * lineHeight
		baseShift := baselineShift(fontSize, lineHeight)
		startY := y + (height-totalTextHeight)/2 + baseShift

		for j, ln := range lines {
			switch i {
			case 0: // STT center
				tw, _ := pdf.MeasureTextWidth(ln)
				pdf.SetX(col.x + (col.width-tw)/2)
			case 5: // Amount right
				tw, _ := pdf.MeasureTextWidth(ln)
				pdf.SetX(col.x + col.width - tw - padding)
			default: // left
				pdf.SetX(col.x + padding)
			}
			pdf.SetY(startY + float64(j)*lineHeight)
			if err := pdf.Cell(nil, ln); err != nil {
				return fmt.Errorf("failed to write cell text: %w", err)
			}
		}
	}
	return nil
}

// calculateRowHeight calculates the height needed for a row based on wrapped text
func (s *Service) calculateRowHeight(pdf *gopdf.GoPdf, columns []column, rowData []string, padding, fontSize float64) float64 {
	maxLines := 1

	for i, col := range columns {
		if i >= len(rowData) {
			continue
		}

		text := rowData[i]
		cellWidth := col.width - (2 * padding)

		lines := s.wrapText(pdf, text, cellWidth, fontSize)
		if len(lines) > maxLines {
			maxLines = len(lines)
		}
	}

	// Keep lineHeight consistent with drawTableRow
	lineHeight := fontSize * 1.6
	return (float64(maxLines) * lineHeight) + (2 * padding)
}

// wrapText wraps text to fit within specified width
func (s *Service) wrapText(pdf *gopdf.GoPdf, text string, maxWidth, fontSize float64) []string {
	if text == "" {
		return []string{""}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	var currentLine string

	for _, word := range words {
		testLine := currentLine
		if testLine != "" {
			testLine += " "
		}
		testLine += word

		// Measure actual text width for precise wrapping
		measuredWidth, err := pdf.MeasureTextWidth(testLine)
		if err != nil {
			measuredWidth = float64(len(testLine)) * fontSize * 0.5
		}

		if measuredWidth <= maxWidth {
			currentLine = testLine
		} else {
			// Current line is full, save it and start new line
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	// Add the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) == 0 {
		return []string{text}
	}

	return lines
}
