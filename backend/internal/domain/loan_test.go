package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoan_Validate(t *testing.T) {
	startDate := time.Now()
	endDate := startDate.AddDate(0, 12, 0)

	tests := []struct {
		name    string
		loan    Loan
		wantErr bool
	}{
		{
			name: "valid loan",
			loan: Loan{
				PrincipalAmount: 100000000,
				InterestRateBps: 1200,
				TermMonths:      12,
				StartDate:       startDate,
				EndDate:         endDate,
			},
			wantErr: false,
		},
		{
			name: "negative principal",
			loan: Loan{
				PrincipalAmount: -100,
				InterestRateBps: 1200,
				TermMonths:      12,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.loan.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoan_CalculateMonthlyInterest(t *testing.T) {
	loan := &Loan{
		OutstandingPrincipal: 100000000,
		InterestRateBps:      1200,
	}
	got := loan.CalculateMonthlyInterest()
	assert.Equal(t, int64(1000000), got)
}

func TestCalculateEndDate(t *testing.T) {
	startDate := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := CalculateEndDate(startDate, 12)
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, want, got)
}

func TestGenerateLoanCode(t *testing.T) {
	createdAt := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	got := GenerateLoanCode(createdAt, 1)
	assert.Equal(t, "LOAN-2026-001", got)
}
