package dashboard

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/domain/services"
	"api-server/internal/infra/persistence/repositories"

	"gorm.io/gorm"
)

type Service struct {
	UserRepo                    domain.UserRepository
	ProjectRepo                 domain.ProjectRepository
	EmployeeRepo                domain.EmployeeRepository
	TimesheetDashboard          domain.TimesheetDashboardReader
	TimesheetPayment            domain.TimesheetPaymentUpdater
	TimesheetQueryRepo          *repositories.TimesheetQueryRepository
	TimesheetAnalyticsRepo      *repositories.TimesheetAnalyticsRepository
	LedgerRepo                  domain.LedgerEntryRepository
	NotificationRepo            domain.NotificationRepository
	AuditLogRepo                domain.AuditLogRepository
	ProjectEmployeeRepo         domain.ProjectEmployeeRepository
	BulkTransferFileRepo        domain.BulkTransferFileRepository
	AdvancePaymentRequestRepo   domain.AdvancePaymentRequestRepository
	AdvancePaymentRepo          domain.AdvancePaymentRepository
	AttendanceRepo              domain.AttendanceRepository
	AttendanceFailedAttemptRepo domain.AttendanceFailedAttemptRepository
	SettingsConfigSvc           *config.SettingsConfigService
	CacheService                *infrastructure.CacheService
	SalaryCalculationSvc        *services.SalaryCalculationService
	PartnerScopeResolver        *PartnerScopeResolver
	logger                      *slog.Logger
}

func NewService(
	userRepo domain.UserRepository,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
	timesheetRepo domain.TimesheetRepository,
	timesheetQueryRepo *repositories.TimesheetQueryRepository,
	timesheetAnalyticsRepo *repositories.TimesheetAnalyticsRepository,
	ledgerRepo domain.LedgerEntryRepository,
	notificationRepo domain.NotificationRepository,
	auditLogRepo domain.AuditLogRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	bulkTransferFileRepo domain.BulkTransferFileRepository,
	advancePaymentRequestRepo domain.AdvancePaymentRequestRepository,
	advancePaymentRepo domain.AdvancePaymentRepository,
	attendanceRepo domain.AttendanceRepository,
	attendanceFailedAttemptRepo domain.AttendanceFailedAttemptRepository,
	settingsConfigSvc *config.SettingsConfigService,
	cacheSvc *infrastructure.CacheService,
	salaryCalculationSvc *services.SalaryCalculationService,
	db *gorm.DB,
	logger *slog.Logger,
) *Service {
	return &Service{
		UserRepo:                    userRepo,
		ProjectRepo:                 projectRepo,
		EmployeeRepo:                employeeRepo,
		TimesheetDashboard:          timesheetRepo,
		TimesheetPayment:            timesheetRepo,
		TimesheetQueryRepo:          timesheetQueryRepo,
		TimesheetAnalyticsRepo:      timesheetAnalyticsRepo,
		LedgerRepo:                  ledgerRepo,
		NotificationRepo:            notificationRepo,
		AuditLogRepo:                auditLogRepo,
		ProjectEmployeeRepo:         projectEmployeeRepo,
		BulkTransferFileRepo:        bulkTransferFileRepo,
		AdvancePaymentRequestRepo:   advancePaymentRequestRepo,
		AdvancePaymentRepo:          advancePaymentRepo,
		AttendanceRepo:              attendanceRepo,
		AttendanceFailedAttemptRepo: attendanceFailedAttemptRepo,
		SettingsConfigSvc:           settingsConfigSvc,
		CacheService:                cacheSvc,
		SalaryCalculationSvc:        salaryCalculationSvc,
		PartnerScopeResolver:        NewPartnerScopeResolver(db, cacheSvc, logger),
		logger:                      logger,
	}
}

// GetHistoricalData retrieves historical dashboard data in the expected format
func (s *Service) GetHistoricalData(ctx context.Context, req *dto.HistoricalRequest) (*dto.HistoricalDataResponse, error) {
	now := clock.Now()

	// Fetch all three datasets concurrently
	type result struct {
		weeklyPay []dto.WeeklyPayData
		revenue   []dto.RevenueData
		capital   []dto.CapitalData
		err       error
	}

	wpCh := make(chan result, 1)
	revCh := make(chan result, 1)
	capCh := make(chan result, 1)

	go func() {
		data, err := s.getWeeklyPayHistory(ctx, now)
		wpCh <- result{weeklyPay: data, err: err}
	}()
	go func() {
		data, err := s.getRevenueHistory(ctx, now)
		revCh <- result{revenue: data, err: err}
	}()
	go func() {
		data, err := s.getCapitalHistory(ctx, now)
		capCh <- result{capital: data, err: err}
	}()

	wp := <-wpCh
	if wp.err != nil {
		return nil, fmt.Errorf("failed to fetch weekly pay: %w", wp.err)
	}
	rev := <-revCh
	if rev.err != nil {
		return nil, fmt.Errorf("failed to fetch revenue: %w", rev.err)
	}
	cap := <-capCh
	if cap.err != nil {
		return nil, fmt.Errorf("failed to fetch capital: %w", cap.err)
	}

	response := &dto.HistoricalDataResponse{
		Date: now.Format("2006-01-02"),
		Historical: dto.HistoricalDataItems{
			WeeklyPay: convertWeeklyPayToHistorical(wp.weeklyPay),
			Revenue:   convertRevenueToHistorical(rev.revenue),
			Capital:   convertCapitalToHistorical(cap.capital),
		},
	}

	s.logger.Info("Successfully fetched historical data",
		"weekly_pay_count", len(response.Historical.WeeklyPay),
		"revenue_count", len(response.Historical.Revenue),
		"capital_count", len(response.Historical.Capital))

	return response, nil
}

// Helper conversion functions for historical data
func convertWeeklyPayToHistorical(data []dto.WeeklyPayData) []dto.WeeklyPayHistorical {
	result := make([]dto.WeeklyPayHistorical, len(data))
	for i, item := range data {
		result[i] = dto.WeeklyPayHistorical{
			Date:              item.Date,
			PaidAmount:        int64(item.TotalWeeklyPay),
			PaidEmployees:     item.PaidEmployees,
			AvgPayPerEmployee: item.AvgPayPerEmployee,
		}
	}
	return result
}

func convertRevenueToHistorical(data []dto.RevenueData) []dto.RevenueHistorical {
	result := make([]dto.RevenueHistorical, len(data))
	for i, item := range data {
		result[i] = dto.RevenueHistorical{
			Date:          item.Date,
			TotalRevenue:  int64(item.TotalRevenue),
			TotalExpenses: int64(item.TotalExpenses),
			Profit:        int64(item.Profit),
		}
	}
	return result
}

func convertCapitalToHistorical(data []dto.CapitalData) []dto.CapitalHistorical {
	result := make([]dto.CapitalHistorical, len(data))
	for i, item := range data {
		result[i] = dto.CapitalHistorical{
			Date:   item.Date,
			Amount: int64(item.TotalCapital),
		}
	}
	return result
}

// GetMonthlyFinancials returns per-month revenue, expenses, and profit for the last 12 months,
// excluding months with zero profit.
func (s *Service) GetMonthlyFinancials(ctx context.Context, req *dto.MonthlyFinancialsRequest) (*dto.MonthlyFinancialsResponse, error) {
	const months = 12

	now := clock.Now()
	startDate := time.Date(now.Year(), now.Month()-time.Month(months-1), 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, time.UTC) // exclusive upper bound

	rows, err := s.TimesheetAnalyticsRepo.GetMonthlyFinancials(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch monthly financials: %w", err)
	}

	// Index results by month key for O(1) lookup
	rowMap := make(map[string]*repositories.MonthlyFinancialRow, len(rows))
	for i := range rows {
		rowMap[rows[i].Month] = &rows[i]
	}

	result := make([]dto.MonthlyFinancialRow, 0, months)
	var totalPaidOut, totalBilled, totalFee int64

	for i := months - 1; i >= 0; i-- {
		t := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		key := t.Format("2006-01")
		var paidOut, billed, fee int64
		if r := rowMap[key]; r != nil {
			paidOut = int64(r.PaidOut)
			billed = int64(r.Billed)
			fee = int64(r.FeeEarned)
		}
		// Exclude months with zero profit
		if fee == 0 {
			continue
		}
		result = append(result, dto.MonthlyFinancialRow{
			Month:     key,
			PaidOut:   paidOut,
			Billed:    billed,
			FeeEarned: fee,
		})
		totalPaidOut += paidOut
		totalBilled += billed
		totalFee += fee
	}

	return &dto.MonthlyFinancialsResponse{
		Period: "1y",
		Months: result,
		Total: dto.MonthlyFinancialRow{
			Month:     "total",
			PaidOut:   totalPaidOut,
			Billed:    totalBilled,
			FeeEarned: totalFee,
		},
	}, nil
}
