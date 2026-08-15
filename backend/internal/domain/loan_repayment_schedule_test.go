package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAllocateRepaymentScheduleComponents_InterestBeforePrincipal(t *testing.T) {
	const (
		principal       = int64(500_000_000)
		monthlyInterest = int64(3_125_000)
	)

	schedules := make([]LoanRepaymentSchedule, 0, 12)
	for period := 1; period <= 11; period++ {
		schedules = append(schedules, LoanRepaymentSchedule{
			Period:  period,
			DueDate: time.Date(2026, time.Month(period+1), 15, 0, 0, 0, 0, time.UTC),
			Amount:  monthlyInterest,
		})
	}
	schedules = append(schedules, LoanRepaymentSchedule{
		Period:  12,
		DueDate: time.Date(2027, time.January, 15, 0, 0, 0, 0, time.UTC),
		Amount:  principal + monthlyInterest,
	})

	require.NoError(t, AllocateRepaymentScheduleComponents(principal, schedules))

	for _, schedule := range schedules[:11] {
		require.Zero(t, schedule.PrincipalAmount)
		require.Equal(t, monthlyInterest, schedule.InterestAmount)
	}
	require.Equal(t, principal, schedules[11].PrincipalAmount)
	require.Equal(t, monthlyInterest, schedules[11].InterestAmount)
}

func TestAllocateRepaymentScheduleComponents_RejectsTotalBelowPrincipal(t *testing.T) {
	schedules := []LoanRepaymentSchedule{{Amount: 99_999_999}}

	err := AllocateRepaymentScheduleComponents(100_000_000, schedules)

	require.Error(t, err)
	require.True(t, IsValidationError(err))
}
