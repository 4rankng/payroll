package advance_payment

import (
	"context"
	"time"
)

// FeeResolver resolves a fee in VND for a given request amount and
// transaction date. Implemented by *FeeScheduleService; injected so the
// calculator stays unit-testable without a database.
type FeeResolver interface {
	ResolveFee(ctx context.Context, amount uint64, at time.Time) uint64
}

// Calculator computes the fee and net amount for an advance-payment request.
// All instances now consult a FeeResolver so fee logic is data-driven by the
// active schedule rather than a fixed percentage at startup.
type Calculator struct {
	resolver FeeResolver
}

// NewCalculator returns a calculator backed by the given resolver. Pass the
// FeeScheduleService in production; pass a stub in tests.
func NewCalculator(resolver FeeResolver) *Calculator {
	return &Calculator{resolver: resolver}
}

// CalculateFee returns (fee, netAmount) for a request created at time `at`.
// The transaction date governs which schedule applies.
func (c *Calculator) CalculateFee(ctx context.Context, requestAmount uint64, at time.Time) (fee, netAmount uint64) {
	fee = c.resolver.ResolveFee(ctx, requestAmount, at)
	if fee >= requestAmount {
		// Defensive: a fee that exceeds the request would underflow netAmount.
		// This should never happen with sane configs but we cap rather than wrap.
		return requestAmount, 0
	}
	netAmount = requestAmount - fee
	return
}

// CalculateMaxAdvance is unchanged from the legacy calculator: max advance =
// salary * percentage. Kept here so callers continue to import a single
// calculator type.
func (c *Calculator) CalculateMaxAdvance(salary uint64, percentage float64) uint64 {
	return uint64(float64(salary) * percentage)
}
