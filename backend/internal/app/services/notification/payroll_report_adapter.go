package notification

import (
	"context"
	"time"

	"api-server/internal/app/services/payroll"
	"api-server/internal/domain/ports/services"
	domainServices "api-server/internal/domain/services"
)

// PayrollReportAdapter implements the PayrollReportPort interface by delegating to payroll services
type PayrollReportAdapter struct {
	reportService   *payroll.PayrollReportByProjectService
	exporterService *payroll.PayrollReportByProjectExporter
}

// NewPayrollReportAdapter creates a new PayrollReportAdapter
func NewPayrollReportAdapter(
	reportService *payroll.PayrollReportByProjectService,
	exporterService *payroll.PayrollReportByProjectExporter,
) *PayrollReportAdapter {
	return &PayrollReportAdapter{
		reportService:   reportService,
		exporterService: exporterService,
	}
}

// GetProjectsForPayrollReport retrieves project payroll report data
func (a *PayrollReportAdapter) GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*domainServices.ProjectReportData, error) {
	return a.reportService.GetProjectsForPayrollReport(ctx, atDate)
}

// GenerateExcel generates Excel report and returns bytes with summary
func (a *PayrollReportAdapter) GenerateExcel(ctx context.Context, reportData []*domainServices.ProjectReportData, atDate time.Time) ([]byte, *services.PayrollReportSummary, error) {
	bytes, summary, err := a.exporterService.GenerateExcel(ctx, reportData, atDate)
	if err != nil {
		return nil, nil, err
	}

	// Convert payroll.PayrollReportSummary to services.PayrollReportSummary
	portSummary := &services.PayrollReportSummary{
		TotalAmount:    summary.TotalAmount,
		FeePercentage:  summary.FeePercentage,
		FeeAmount:      summary.FeeAmount,
		TotalWithFee:   summary.TotalWithFee,
		DueDate:        summary.DueDate,
		FormattedRange: summary.FormattedRange,
	}

	return bytes, portSummary, nil
}
