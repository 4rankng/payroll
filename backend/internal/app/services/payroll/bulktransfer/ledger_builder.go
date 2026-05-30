package bulktransfer

import (
	"context"
	"fmt"
	"math"
	"time"

	"api-server/internal/domain"
)

// LedgerPlan captures computed values for transaction/ledger entries
type LedgerPlan struct {
	Description      string
	ReceivableAmount int64
	CashOutAmount    int64
	RevenueOffset    int64
}

// BuildLedgerPlan computes amounts for receivable, cash out, and revenue offset
func BuildLedgerPlan(ctx context.Context, cfg SettingsConfigService, totalTransfer float64, partnerCompany, filename string) LedgerPlan {
	feePct := cfg.GetAdvanceCashFeePercentage(ctx)
	fee := totalTransfer * feePct
	receivable := int64(math.Round(totalTransfer + fee))
	cashOut := int64(math.Round(totalTransfer))
	desc := fmt.Sprintf("Trả lương cho %s, file: %s", partnerCompany, filename)
	return LedgerPlan{Description: desc, ReceivableAmount: receivable, CashOutAmount: cashOut, RevenueOffset: cashOut}
}

// Entries returns the manual ledger entries for cash out and revenue offset
func (p LedgerPlan) Entries(now time.Time, processedBy uint, partnerCompany string) []*domain.LedgerEntry {
	cashOutEntry := &domain.LedgerEntry{
		Date:      now,
		Account:   domain.AccountCash,
		Party:     "Nhân viên",
		Debit:     0,
		Credit:    p.CashOutAmount,
		CreatedBy: processedBy,
	}

	revenueOffsetEntry := &domain.LedgerEntry{
		Date:      now,
		Account:   domain.AccountRevenue,
		Party:     partnerCompany,
		Debit:     p.RevenueOffset,
		Credit:    0,
		CreatedBy: processedBy,
	}
	return []*domain.LedgerEntry{cashOutEntry, revenueOffsetEntry}
}
