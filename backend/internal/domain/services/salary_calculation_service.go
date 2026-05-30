package services

import (
	"context"
	"time"

	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
)

// SalaryCalculationService handles salary-related calculations based on actual transfer data
type SalaryCalculationService struct {
	bulkTransferFileRepo domain.BulkTransferFileRepository
}

// NewSalaryCalculationService creates a new SalaryCalculationService instance
func NewSalaryCalculationService(bulkTransferFileRepo domain.BulkTransferFileRepository) *SalaryCalculationService {
	return &SalaryCalculationService{
		bulkTransferFileRepo: bulkTransferFileRepo,
	}
}

// CalculateAverageSalariesFromTransfers calculates average weekly and monthly salaries
// based on actual transfer amounts from bulk transfer files within the last 30 days
// Returns: (avgWeeklySalary, avgMonthlySalary, error)
func (s *SalaryCalculationService) CalculateAverageSalariesFromTransfers(ctx context.Context) (int64, int64, error) {
	// Calculate date range for last 30 days
	now := clock.Now()
	thirtyDaysAgo := now.AddDate(0, 0, -30)

	// Get recent high-value transfers (> 1M VND) from the last 30 days
	employeeTransfers, err := s.bulkTransferFileRepo.GetRecentHighValueTransfers(ctx, thirtyDaysAgo, now)
	if err != nil {
		return 0, 0, domain.NewInternalError("failed to get recent high-value transfers", err)
	}

	// If no transfers found, return 0
	if len(employeeTransfers) == 0 {
		return 0, 0, nil
	}

	var totalWeeklySum int64
	var employeeCount int

	// Calculate average weekly salary per employee
	for _, transfers := range employeeTransfers {
		if len(transfers) == 0 {
			continue
		}

		// Calculate average for this employee
		var employeeSum int64
		for _, amount := range transfers {
			employeeSum += amount
		}
		employeeAvg := employeeSum / int64(len(transfers))

		totalWeeklySum += employeeAvg
		employeeCount++
	}

	// Calculate overall average weekly salary
	var avgWeeklySalary int64
	if employeeCount > 0 {
		avgWeeklySalary = totalWeeklySum / int64(employeeCount)
	}

	// Calculate average monthly salary (weekly * 4)
	avgMonthlySalary := avgWeeklySalary * 4

	return avgWeeklySalary, avgMonthlySalary, nil
}

// Repository interface for dependency injection (for testing)
type SalaryCalculationRepository interface {
	GetRecentHighValueTransfers(ctx context.Context, fromDate, toDate time.Time) (map[uint][]int64, error)
}
