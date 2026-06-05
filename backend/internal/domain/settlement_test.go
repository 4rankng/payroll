package domain

import (
	"testing"

	"api-server/internal/pkg/clock"

	"github.com/stretchr/testify/assert"
)

func TestSettlement_Validate(t *testing.T) {
	yesterday := clock.Now().AddDate(0, 0, -1)

	tests := []struct {
		name       string
		settlement Settlement
		wantErr    bool
	}{
		{
			name: "valid settlement",
			settlement: Settlement{
				TransactionID:  1,
				Amount:         100000,
				SettlementDate: yesterday,
				CreatedBy:      1,
			},
			wantErr: false,
		},
		{
			name: "missing transaction_id",
			settlement: Settlement{
				Amount:         100000,
				SettlementDate: yesterday,
				CreatedBy:      1,
			},
			wantErr: true,
		},
		{
			name: "negative amount",
			settlement: Settlement{
				TransactionID:  1,
				Amount:         -100,
				SettlementDate: yesterday,
				CreatedBy:      1,
			},
			wantErr: true,
		},
		{
			name: "zero amount",
			settlement: Settlement{
				TransactionID:  1,
				Amount:         0,
				SettlementDate: yesterday,
				CreatedBy:      1,
			},
			wantErr: true,
		},
		{
			name: "zero settlement date",
			settlement: Settlement{
				TransactionID: 1,
				Amount:        100000,
				CreatedBy:     1,
			},
			wantErr: true,
		},
		{
			name: "future settlement date",
			settlement: Settlement{
				TransactionID:  1,
				Amount:         100000,
				SettlementDate: clock.Now().AddDate(0, 0, 1),
				CreatedBy:      1,
			},
			wantErr: true,
		},
		{
			name: "missing created_by",
			settlement: Settlement{
				TransactionID:  1,
				Amount:         100000,
				SettlementDate: yesterday,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.settlement.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSettlement_TableName(t *testing.T) {
	s := Settlement{}
	tableName := s.TableName()
	assert.Equal(t, "settlements", tableName)
}
