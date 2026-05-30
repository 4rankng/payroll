package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransactionType_Label(t *testing.T) {
	tests := []struct {
		name string
		tt   TransactionType
		want string
	}{
		{
			name: "revenue type",
			tt:   TransactionTypeRevenue,
			want: "Doanh thu",
		},
		{
			name: "capital type",
			tt:   TransactionTypeCapital,
			want: "Vốn",
		},
		{
			name: "expense type",
			tt:   TransactionTypeExpense,
			want: "Chi phí",
		},
		{
			name: "loan disbursement type",
			tt:   TransactionTypeLoanDisbursement,
			want: "Giải ngân khoản vay",
		},
		{
			name: "loan repayment type",
			tt:   TransactionTypeLoanRepayment,
			want: "Trả nợ gốc",
		},
		{
			name: "unknown type defaults to expense",
			tt:   "unknown",
			want: "Chi phí",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tt.Label()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTransactionStatus_Label(t *testing.T) {
	tests := []struct {
		name   string
		status TransactionStatus
		want   string
	}{
		{
			name:   "settled status",
			status: TransactionStatusSettled,
			want:   "Đã thanh toán",
		},
		{
			name:   "pending status",
			status: TransactionStatusPending,
			want:   "Chờ thanh toán",
		},
		{
			name:   "unknown status defaults to pending",
			status: "unknown",
			want:   "Chờ thanh toán",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Label()
			assert.Equal(t, tt.want, got)
		})
	}
}
