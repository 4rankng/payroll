package workers

import (
	"math"
	"testing"

	"api-server/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestTransferPaidAmountsPreserveExportedTotal(t *testing.T) {
	for _, test := range []struct {
		name   string
		amount int64
		gross  []int64
		want   []int64
	}{
		{"configured percentage", 168000, []int64{240000}, []int64{168000}},
		{"old in-flight gross export", 240000, []int64{240000}, []int64{240000}},
		{"exact rounding remainder", 140002, []int64{100001, 100002}, []int64{70001, 70001}},
		{"stable tie", 1, []int64{1, 1}, []int64{1, 0}},
		{"large amounts do not overflow", math.MaxInt64, []int64{math.MaxInt64, math.MaxInt64}, []int64{math.MaxInt64/2 + 1, math.MaxInt64 / 2}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var rows []*domain.Timesheet
			for i, gross := range test.gross {
				rows = append(rows, &domain.Timesheet{ID: uint(i + 1), Amount: gross})
			}
			amounts, err := allocatePaidTransferAmount(rows, test.amount)
			require.NoError(t, err)
			var sum int64
			for i, want := range test.want {
				require.Equal(t, want, amounts[uint(i+1)])
				sum += amounts[uint(i+1)]
			}
			require.Equal(t, test.amount, sum)
			for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
				rows[i], rows[j] = rows[j], rows[i]
			}
			reordered, err := allocatePaidTransferAmount(rows, test.amount)
			require.NoError(t, err)
			require.Equal(t, amounts, reordered)
		})
	}
}

func TestTransferPaidAmountsRejectInvalidWages(t *testing.T) {
	for _, rows := range [][]*domain.Timesheet{nil, {nil}, {{ID: 1, Amount: 0}}, {{ID: 1, Amount: -1}}, {{ID: 1, Amount: 1}, {ID: 1, Amount: 2}}} {
		_, err := allocatePaidTransferAmount(rows, 100000)
		require.Error(t, err)
	}
}
