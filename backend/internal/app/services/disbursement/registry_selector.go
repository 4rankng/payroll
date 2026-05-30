package disbursement

import (
	"context"

	"github.com/gin-gonic/gin"
)

type providerHintKey struct{}

// ContextWithProviderHint returns a context carrying the given provider
// name as a hint for the registry's Active() selector.
func ContextWithProviderHint(ctx context.Context, name string) context.Context {
	return context.WithValue(ctx, providerHintKey{}, name)
}

// ProviderHintFromContext extracts the provider hint from ctx.
// Returns "" when no hint is set.
func ProviderHintFromContext(ctx context.Context) string {
	v, _ := ctx.Value(providerHintKey{}).(string)
	return v
}

// ProviderHintFromHeader pulls the provider preference from the
// "X-Disbursement-Provider" request header (set by the admin UI) and
// returns a new context with the hint attached.
func ProviderHintFromHeader(c *gin.Context) context.Context {
	name := c.GetHeader("X-Disbursement-Provider")
	if name == "" {
		return c.Request.Context()
	}
	return ContextWithProviderHint(c.Request.Context(), name)
}

// ContextSelector is a Selector that reads the provider hint from context.
type ContextSelector struct{}

func (ContextSelector) Choose(ctx context.Context) string {
	return ProviderHintFromContext(ctx)
}

// FlagSelector routes to a specific provider based on feature flags set
// at boot. It is consulted by Active(ctx) before the first-registered
// fallback, so a flagged provider takes priority when enabled.
//
// Context key "disbursement_use_case" determines the routing category:
//   - "employee" (or empty/missing) → employee advance-payment routing
//   - "bulk_transfer"              → bulk/payroll transfer routing
//
// Config-time validation guarantees at most one provider has each
// *_FOR_EMPLOYEE / *_FOR_BULK_TRANSFER flag set, so there is never a tie.
type FlagSelector struct {
	OnepayForEmployee      bool
	NinepayForEmployee     bool
	OnepayForBulkTransfer  bool
	NinepayForBulkTransfer bool
}

type useCaseKey struct{}

// ContextWithUseCase returns a context carrying the disbursement use-case
// identifier (e.g. "employee", "bulk_transfer"). The FlagSelector reads
// this to choose the right routing category.
func ContextWithUseCase(ctx context.Context, useCase string) context.Context {
	return context.WithValue(ctx, useCaseKey{}, useCase)
}

func useCaseFromContext(ctx context.Context) string {
	v, _ := ctx.Value(useCaseKey{}).(string)
	return v
}

func (fs FlagSelector) Choose(ctx context.Context) string {
	switch useCaseFromContext(ctx) {
	case "bulk_transfer":
		if fs.OnepayForBulkTransfer {
			return "1pay"
		}
		if fs.NinepayForBulkTransfer {
			return "9pay"
		}
	default:
		// employee / default
		if fs.OnepayForEmployee {
			return "1pay"
		}
		if fs.NinepayForEmployee {
			return "9pay"
		}
	}
	return ""
}

// ChainSelector tries each selector in order; first non-empty result wins.
type ChainSelector struct {
	Selectors []Selector
}

func (cs *ChainSelector) Choose(ctx context.Context) string {
	for _, s := range cs.Selectors {
		if name := s.Choose(ctx); name != "" {
			return name
		}
	}
	return ""
}
