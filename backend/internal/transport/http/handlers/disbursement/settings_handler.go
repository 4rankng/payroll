package disbursement

import (
	"context"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/disbursement"
	"api-server/internal/domain/ports/infrastructure"
	"api-server/internal/domain/wallet"
	"api-server/internal/transport/http/response"
)

type SettingsHandler struct {
	registry *disbursement.Registry
	fees     *disbursement.FeeScheduleService
	wallet   wallet.WalletService
	flags    flagConfig
}

type flagConfig struct {
	OnepayForEmployee      bool
	NinepayForEmployee     bool
	OnepayForBulkTransfer  bool
	NinepayForBulkTransfer bool
}

func NewSettingsHandler(
	registry *disbursement.Registry,
	fees *disbursement.FeeScheduleService,
	walletSvc wallet.WalletService,
	onepayForEmployee, ninepayForEmployee bool,
	onepayForBulkTransfer, ninepayForBulkTransfer bool,
) *SettingsHandler {
	return &SettingsHandler{
		registry: registry,
		fees:     fees,
		wallet:   walletSvc,
		flags: flagConfig{
			OnepayForEmployee:      onepayForEmployee,
			NinepayForEmployee:     ninepayForEmployee,
			OnepayForBulkTransfer:  onepayForBulkTransfer,
			NinepayForBulkTransfer: ninepayForBulkTransfer,
		},
	}
}

func (h *SettingsHandler) Get(c *gin.Context) {
	ctx := c.Request.Context()
	out := gin.H{}

	names := h.registry.Names()
	registered := make([]gin.H, 0, len(names))
	labels := map[string]string{"9pay": "9Pay", "1pay": "OnePay"}
	for _, n := range names {
		l := n
		if lbl, ok := labels[n]; ok {
			l = lbl
		}
		registered = append(registered, gin.H{"name": n, "label": l})
	}
	out["registered_providers"] = registered

	active, err := h.registry.Active(ctx)
	if err != nil {
		out["active_provider"] = nil
		response.Success(c, out, "")
		return
	}

	name := active.Name()
	out["active_provider"] = gin.H{
		"name":              name,
		"for_employee":      h.forEmployee(name),
		"for_bulk_transfer": h.forBulkTransfer(name),
		"capabilities": gin.H{
			"balance_inquiry":       implements[infrastructure.BalanceReporter](active),
			"account_verifier":      implements[infrastructure.AccountVerifier](active),
			"report_export":         implements[infrastructure.ReportExporter](active),
			"status_poller":         implements[infrastructure.StatusPoller](active),
			"reconcile_via_inquiry": false,
		},
	}

	fee := h.buildFee(ctx, name)
	out["fee"] = fee

	if bal, err := h.wallet.GetBalance(ctx); err == nil {
		out["internal_balance"] = gin.H{
			"available": bal.Available,
			"currency":  bal.Currency,
		}
	}

	if reporter, ok := active.(infrastructure.BalanceReporter); ok {
		if res, err := reporter.GetBalance(ctx); err == nil {
			out["provider_balance"] = gin.H{
				"amount": res.Amount,
			}
		}
	}

	response.Success(c, out, "")
}

func (h *SettingsHandler) buildFee(ctx context.Context, providerName string) gin.H {
	active, err := h.fees.ActiveEntry(ctx)
	if err != nil {
		return gin.H{"current_vnd": 0, "effective_date": "", "upcoming": []gin.H{}}
	}

	upcoming := []gin.H{}
	all, _ := h.fees.List(ctx)
	for _, e := range all {
		if e.Provider == providerName && e.EffectiveDate > active.EffectiveDate {
			upcoming = append(upcoming, gin.H{
				"effective_date": e.EffectiveDate,
				"fee_vnd":        e.FeeVND,
				"notes":          e.Notes,
			})
		}
	}

	return gin.H{
		"current_vnd":    active.FeeVND,
		"effective_date": active.EffectiveDate,
		"upcoming":       upcoming,
	}
}

func (h *SettingsHandler) forEmployee(name string) bool {
	switch name {
	case "1pay":
		return h.flags.OnepayForEmployee
	case "9pay":
		return h.flags.NinepayForEmployee
	}
	return false
}

func (h *SettingsHandler) forBulkTransfer(name string) bool {
	switch name {
	case "1pay":
		return h.flags.OnepayForBulkTransfer
	case "9pay":
		return h.flags.NinepayForBulkTransfer
	}
	return false
}

func implements[T any](p infrastructure.DisbursementProvider) bool {
	_, ok := p.(T)
	return ok
}
