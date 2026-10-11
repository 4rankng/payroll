package llmchat

import (
	"context"

	"api-server/internal/app/services/dashboard"
)

// partnerScopeAdapter adapts the materialised PartnerScope resolver (briefly
// Redis-cached ID sets) to the chat scope seam, so tool queries reuse the
// canonical partner visibility semantics instead of a private copy.
type partnerScopeAdapter struct {
	resolver *dashboard.PartnerScopeResolver
}

// NewScopeResolver builds the adapter. Returns nil for a nil resolver (the
// executor then leaves partner scopes empty — tools fail closed).
func NewScopeResolver(resolver *dashboard.PartnerScopeResolver) ScopeResolver {
	if resolver == nil {
		return nil
	}
	return partnerScopeAdapter{resolver: resolver}
}

func (a partnerScopeAdapter) ResolvePartnerScope(ctx context.Context, partnerID uint) ([]uint, []uint, error) {
	scope, err := a.resolver.Resolve(ctx, partnerID)
	if err != nil {
		return nil, nil, err
	}
	if scope == nil {
		return []uint{}, []uint{}, nil
	}
	return scope.EmployeeIDs, scope.ProjectIDs, nil
}
