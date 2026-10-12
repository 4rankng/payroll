package dashboard

import (
	"context"
	"log/slog"
	"testing"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/mocks"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestGetEmployeeActivityStatsIncludesPerProjectCounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	userRepo := mocks.NewMockUserRepository(ctrl)
	userRepo.EXPECT().
		CountActiveEmployeesBySchedule(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(73, 0, 85, nil)
	userRepo.EXPECT().
		CountActiveEmployeesByProject(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]*domain.ProjectActiveCount{
			{ProjectID: 1, ProjectName: "Nha may VERICO", Count: 120},
			{ProjectID: 2, ProjectName: "San xuat xa phong", Count: 38},
		}, nil)

	service := NewService(
		userRepo,
		nil, // ProjectRepo
		nil, // EmployeeRepo
		nil, // TimesheetQueryRepo
		nil, // TimesheetAnalyticsRepo
		nil, // LedgerRepo
		nil, // NotificationRepo
		nil, // AuditLogRepo
		nil, // ProjectEmployeeRepo
		nil, // BulkTransferHistoryRepo
		nil, // AdvancePaymentRequestRepo
		nil, // AdvancePaymentRepo
		nil, // AttendanceRepo
		nil, // AttendanceFailedAttemptRepo
		nil, // SettingsConfigSvc
		nil, // CacheService
		nil, // SalaryCalculationSvc
		nil, // TimesheetRepo
		nil, // db
		slog.Default(),
	)

	stats, err := service.GetEmployeeActivityStats(context.Background(), "2026-10")
	assert.NoError(t, err)
	assert.Equal(t, 73, stats.ActiveWeekly)
	assert.Equal(t, 0, stats.ActiveMonthly)
	assert.Equal(t, 85, stats.ActiveFlexible)
	assert.Equal(t, 158, stats.ActiveTotal)
	assert.Equal(t, []*dto.ProjectActiveCountResponse{
		{ProjectID: 1, ProjectName: "Nha may VERICO", Count: 120},
		{ProjectID: 2, ProjectName: "San xuat xa phong", Count: 38},
	}, stats.ActiveByProject)
}
