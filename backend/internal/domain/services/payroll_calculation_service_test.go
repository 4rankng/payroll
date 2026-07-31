package services

import (
	"context"
	"testing"
	"time"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
)

type payrateRepositoryStub struct {
	domain.PayrateRepository
	activePayrate *domain.Payrate
}

func (s payrateRepositoryStub) GetActiveByProjectAndDate(
	_ context.Context,
	_ uint,
	_ time.Time,
) (*domain.Payrate, error) {
	return s.activePayrate, nil
}

func TestCalculateTimesheetAmountUsesProjectPayUnit(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		isFlexible bool
		wantAmount int64
	}{
		{
			name:       "flexible project pays configured total once for the shift",
			isFlexible: true,
			wantAmount: 252000,
		},
		{
			name:       "standard project keeps hourly calculation",
			isFlexible: false,
			wantAmount: 2520000,
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			date := time.Date(2026, 7, 31, 0, 0, 0, 0, time.Local)
			timesheet := &domain.Timesheet{
				ProjectID:   58,
				Date:        date,
				HoursWorked: 10,
				PayType:     "Công nhân.ngày thường.08:00-18:00",
			}
			payrate := &domain.Payrate{
				ID:        91,
				ProjectID: 58,
				Project: domain.Project{
					ID:         58,
					IsFlexible: testCase.isFlexible,
				},
				Payrate: domain.PayrateConfiguration(
					`{"Công nhân":{"ngày thường":{"08:00-18:00":252000}}}`,
				),
			}
			service := NewPayrollCalculationService(
				nil,
				payrateRepositoryStub{activePayrate: payrate},
				nil,
			)

			amount, err := service.CalculateTimesheetAmount(context.Background(), timesheet)

			require.NoError(t, err)
			require.Equal(t, testCase.wantAmount, amount)
			require.Equal(t, int64(252000), timesheet.PayRate)
			require.Equal(t, uint(91), timesheet.PayrateID)
		})
	}
}
