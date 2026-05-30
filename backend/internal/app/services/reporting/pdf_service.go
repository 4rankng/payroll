package reporting

import (
	"api-server/internal/app/services/payroll/pdf"
)

// NewPDFService creates a new PDF service
func NewPDFService(fontPath string) *pdf.Service {
	return pdf.NewService(fontPath)
}
