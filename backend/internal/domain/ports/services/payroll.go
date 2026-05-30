package services

import (
	"context"
	"time"

	domainServices "api-server/internal/domain/services"
)

// PayrollPort defines the interface for cross-domain payroll service communication
type PayrollPort interface {
	CalculateSalary(ctx context.Context, employeeID uint, fromDate, toDate time.Time) (int64, error)
	GetPendingSalary(ctx context.Context, fromDate, toDate time.Time) (int64, error)
	GetPaidSalary(ctx context.Context, fromDate, toDate time.Time) (int64, error)
	ProcessPayroll(ctx context.Context, fromDate, toDate time.Time) error
}

// PayrollReportSummary captures key figures needed for presentation layers
type PayrollReportSummary struct {
	TotalAmount    int64
	FeePercentage  float64
	FeeAmount      int64
	TotalWithFee   int64
	DueDate        time.Time
	FormattedRange string
}

// PayrollReportPort defines the interface for payroll report operations
type PayrollReportPort interface {
	GetProjectsForPayrollReport(ctx context.Context, atDate time.Time) ([]*domainServices.ProjectReportData, error)
	GenerateExcel(reportData []*domainServices.ProjectReportData, atDate time.Time) ([]byte, *PayrollReportSummary, error)
}
