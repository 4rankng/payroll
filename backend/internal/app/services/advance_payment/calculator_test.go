package advance_payment

import (
	"context"
	"testing"
	"time"
)

type stubResolver struct {
	fee uint64
}

func (s *stubResolver) ResolveFee(_ context.Context, _ uint64, _ time.Time) uint64 {
	return s.fee
}

func TestCalculator_CalculateFee(t *testing.T) {
	cases := []struct {
		name    string
		amount  uint64
		fee     uint64
		wantFee uint64
		wantNet uint64
	}{
		{"normal", 1_000_000, 20_000, 20_000, 980_000},
		{"floor applied", 100_000, 10_000, 10_000, 90_000},
		{"fee equals amount caps net at zero", 50_000, 50_000, 50_000, 0},
		{"fee exceeds amount caps net at zero", 50_000, 60_000, 50_000, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calc := NewCalculator(&stubResolver{fee: c.fee})
			gotFee, gotNet := calc.CalculateFee(context.Background(), c.amount, time.Now())
			if gotFee != c.wantFee || gotNet != c.wantNet {
				t.Errorf("fee=%d net=%d, want fee=%d net=%d", gotFee, gotNet, c.wantFee, c.wantNet)
			}
		})
	}
}
