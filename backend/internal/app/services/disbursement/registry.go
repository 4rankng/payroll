// Package disbursement wires the configured disbursement provider into
// callers via a small Registry. Use cases that need to send money out
// resolve the active provider through Registry.Active(ctx) rather than
// importing a concrete provider package.
//
// Provider selection happens at boot via env vars (e.g. ENABLE_NINEPAY).
// Bootstrap registers a provider only when its master switch is on AND
// its credentials are present, so the registry's contents at runtime
// already encode "what is allowed to send money." Active(ctx) returns
// ErrNoActiveProvider when nothing is registered.
package disbursement

import (
	"context"
	"errors"
	"fmt"

	"api-server/internal/domain/ports/infrastructure"
)

// ErrNoActiveProvider is returned by Active(ctx) when no provider is
// registered — typically because the master env flag is off or the
// configured provider's credentials were blank at boot.
var ErrNoActiveProvider = errors.New("disbursement: no active provider configured")

// ErrBalanceNotSupported is returned by GetProviderBalance when the active
// provider does not implement the BalanceReporter interface.
var ErrBalanceNotSupported = errors.New("disbursement: active provider does not support balance reporting")

// ErrDuplicatePaymentInProgress is returned by Initiate when a pending or
// authorised wallet_payment already exists for the same recipient. Prevents
// double disbursement when both auto-poller and manual admin flows target
// the same employee simultaneously.
var ErrDuplicatePaymentInProgress = errors.New("disbursement: a payment to this recipient is already in progress")

// ErrIPNAmountMismatch is returned by RecordIPN when an inbound IPN's stated
// amount does not match the wallet_payment's RequestedAmount (the amount we
// asked the provider to disburse at Initiate). A mismatch signals a forged or
// tampered callback and must never reach the FSM. It is terminal — asynq must
// not retry it (a forgery will not self-heal on the next poll).
var ErrIPNAmountMismatch = errors.New("disbursement: ipn amount does not match the recorded request")

// Registry holds all built disbursement providers and exposes the
// active one. With env-gated registration, "active" reduces to "the
// provider bootstrap chose to register" — there is exactly one in
// practice, since each provider has its own master switch.
//
// The registry is read-mostly after bootstrap; Register is not safe
// for concurrent use and should only be called during wiring.
type Registry struct {
	providers map[string]infrastructure.DisbursementProvider
	// order preserves Register call order so Active is deterministic
	// when multiple providers are registered in some future world.
	order    []string
	selector Selector
}

// NewRegistry builds a registry pre-populated with the given providers.
// Duplicate names are a programmer error and will panic — bootstrap
// should never register two providers under the same Name().
func NewRegistry(providers ...infrastructure.DisbursementProvider) *Registry {
	r := &Registry{
		providers: make(map[string]infrastructure.DisbursementProvider, len(providers)),
	}
	for _, p := range providers {
		r.Register(p)
	}
	return r
}

// Register adds a provider to the registry under its Name(). Bootstrap-only;
// not concurrency-safe.
func (r *Registry) Register(p infrastructure.DisbursementProvider) {
	if p == nil {
		return
	}
	name := p.Name()
	if name == "" {
		panic(fmt.Sprintf("disbursement: provider returned invalid Name() %q", name))
	}
	if _, exists := r.providers[name]; exists {
		panic(fmt.Sprintf("disbursement: provider %q registered twice", name))
	}
	r.providers[name] = p
	r.order = append(r.order, name)
}

// Selector returns the provider name to use for a given context.
// Returning "" means "no preference, use the default".
type Selector interface {
	Choose(ctx context.Context) string
}

// WithSelector attaches a selector to the registry. The selector is
// consulted by Active(ctx) before falling back to first-registered.
func (r *Registry) WithSelector(sel Selector) *Registry {
	r.selector = sel
	return r
}

// Active returns the registered provider, consulting the Selector first
// (if one is wired), then falling back to first-registered.
// Returns ErrNoActiveProvider when nothing is registered — the natural
// "disbursements are off" state when no master env flag is on.
func (r *Registry) Active(ctx context.Context) (infrastructure.DisbursementProvider, error) {
	if r.selector != nil {
		if name := r.selector.Choose(ctx); name != "" {
			if p, ok := r.providers[name]; ok {
				return p, nil
			}
		}
	}
	// Fallback: first registered (preserves legacy behaviour when only
	// one provider is wired).
	if len(r.order) == 0 {
		return nil, ErrNoActiveProvider
	}
	return r.providers[r.order[0]], nil
}

// ActiveTransferLimits resolves the active provider and returns its
// per-transfer amount bounds. Returns a zero-value TransferLimits (no
// restrictions) when no provider is registered or the provider doesn't
// implement TransferLimiter.
func (r *Registry) ActiveTransferLimits(ctx context.Context) infrastructure.TransferLimits {
	p, err := r.Active(ctx)
	if err != nil {
		return infrastructure.TransferLimits{}
	}
	if l, ok := p.(infrastructure.TransferLimiter); ok {
		return l.Limits()
	}
	return infrastructure.TransferLimits{}
}

// ByName returns a specific registered provider by Name(). Useful for
// per-provider webhook routes that need to dispatch to a known provider
// regardless of which one is currently active.
func (r *Registry) ByName(name string) (infrastructure.DisbursementProvider, bool) {
	p, ok := r.providers[name]
	return p, ok
}

// Names returns the registered provider names in registration order.
// Intended for diagnostics and admin tooling.
func (r *Registry) Names() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// ActiveProviderName returns the name of the active provider. Returns
// ErrNoActiveProvider when nothing is registered. Convenience wrapper
// for callers that only need the name, not the full provider instance.
func (r *Registry) ActiveProviderName(ctx context.Context) (string, error) {
	p, err := r.Active(ctx)
	if err != nil {
		return "", err
	}
	return p.Name(), nil
}

// GetProviderBalance resolves the active provider, asserts BalanceReporter,
// and returns the wallet balance from the payment provider (e.g. OnePay).
func (r *Registry) GetProviderBalance(ctx context.Context) (int64, error) {
	p, err := r.Active(ctx)
	if err != nil {
		return 0, err
	}
	reporter, ok := p.(infrastructure.BalanceReporter)
	if !ok {
		return 0, ErrBalanceNotSupported
	}
	result, err := reporter.GetBalance(ctx)
	if err != nil {
		return 0, err
	}
	return result.Amount, nil
}

// passthroughTranslator is the default ErrorTranslator returned when
// no provider is registered or the registered provider doesn't
// implement the capability. It returns "" for the empty code so
// callers can short-circuit, and the raw code otherwise.
type passthroughTranslator struct{}

func (passthroughTranslator) TranslateError(code string) string { return code }

// TranslateError resolves the user-facing Vietnamese message for a
// raw provider error code. Looks up the named provider in the
// registry and delegates if it implements infrastructure.ErrorTranslator;
// falls back to a passthrough (returns the code itself) when no
// provider is registered or the registered one lacks the capability.
//
// The pre-IPN service layer uses this to avoid importing concrete
// provider packages just to translate codes.
func (r *Registry) TranslateError(name, code string) string {
	if code == "" {
		return ""
	}
	if p, ok := r.providers[name]; ok {
		if t, ok := p.(infrastructure.ErrorTranslator); ok {
			return t.TranslateError(code)
		}
	}
	return passthroughTranslator{}.TranslateError(code)
}

// TranslateActiveError translates an error code via the active provider's
// translator. Returns the code itself when there is no active provider
// or the active provider does not implement ErrorTranslator.
func (r *Registry) TranslateActiveError(code string) string {
	if code == "" {
		return ""
	}
	p, err := r.Active(context.Background())
	if err != nil {
		return passthroughTranslator{}.TranslateError(code)
	}
	if t, ok := p.(infrastructure.ErrorTranslator); ok {
		return t.TranslateError(code)
	}
	return passthroughTranslator{}.TranslateError(code)
}

// ActiveErrorTranslator returns an ErrorTranslator that resolves codes
// against whichever provider is active at call time. Useful for
// service-layer wiring that must not import a concrete provider package.
// The returned translator is safe to use even when no provider is
// registered — it falls through to a passthrough.
func (r *Registry) ActiveErrorTranslator() infrastructure.ErrorTranslator {
	return registryErrorTranslator{registry: r}
}

type registryErrorTranslator struct{ registry *Registry }

func (t registryErrorTranslator) TranslateError(code string) string {
	return t.registry.TranslateActiveError(code)
}
