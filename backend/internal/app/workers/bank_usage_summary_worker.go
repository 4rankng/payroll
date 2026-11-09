package workers

import (
	"context"
	"log/slog"
)

// BankUsageSummaryRefresher is the narrow port the asynq worker needs from the
// dashboard service (see app/services/dashboard/bank_usage_summary.go).
type BankUsageSummaryRefresher interface {
	RefreshBankUsageSummary(ctx context.Context) error
}

// BankUsageSummaryWorker rebuilds the dashboard bank-usage summary snapshot on
// the asynq schedule (@every 2m). The refresh is a single aggregate scan plus
// an in-Go fold — cheap enough for the 1-vCPU prod box — and idempotent: the
// snapshot swap is transactional, so retries and overlapping runs are safe.
type BankUsageSummaryWorker struct {
	refresher BankUsageSummaryRefresher
	logger    *slog.Logger
}

func NewBankUsageSummaryWorker(refresher BankUsageSummaryRefresher, logger *slog.Logger) *BankUsageSummaryWorker {
	return &BankUsageSummaryWorker{refresher: refresher, logger: logger}
}

// Refresh recomputes the summary. Returning the error lets asynq retry with
// its backoff; a failed refresh leaves the previous snapshot untouched.
func (w *BankUsageSummaryWorker) Refresh(ctx context.Context) error {
	if w.refresher == nil {
		return nil
	}
	if err := w.refresher.RefreshBankUsageSummary(ctx); err != nil {
		w.logger.Error("bank usage summary refresh failed", "error", err)
		return err
	}
	return nil
}
