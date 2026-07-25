package dashboard

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/infra/persistence/repositories"

	"gorm.io/gorm"
)

// PartnerScopeTTL is how long a resolved PartnerScope stays in Redis. It only
// needs to cover the lifetime of a single dashboard burst (4 concurrent
// analytics queries + the frontend refetch window), so 10s is enough to dedupe
// within a request while staying fresh against permission changes.
const PartnerScopeTTL = 10 * time.Second

// PartnerScopeResolver materialises the "what's visible to this partner" ID set
// once per request burst and caches it briefly in Redis. Without this, each of
// the 4 partner dashboard analytics queries re-evaluates partnerAccessCondition()
// — a 4-branch OR with 3 correlated EXISTS subqueries — on every row.
//
// The resolver runs 4 small index-backed lookups (one per access path), unions
// the employee IDs and project IDs in Go, and returns a repositories.PartnerScope
// that the analytics methods substitute for the correlated subqueries.
type PartnerScopeResolver struct {
	db     *gorm.DB
	cache  *infrastructure.CacheService
	logger *slog.Logger
}

// NewPartnerScopeResolver constructs a resolver bound to the given DB / cache.
func NewPartnerScopeResolver(db *gorm.DB, cache *infrastructure.CacheService, logger *slog.Logger) *PartnerScopeResolver {
	return &PartnerScopeResolver{db: db, cache: cache, logger: logger}
}

// Resolve returns the partner's visible employee + project ID sets, using a
// short-lived Redis cache to dedupe concurrent calls within the same burst.
func (r *PartnerScopeResolver) Resolve(ctx context.Context, partnerID uint) (*repositories.PartnerScope, error) {
	// Defensive: if the resolver was constructed without a DB (e.g. in a wiring
	// test), return an empty scope rather than nil-dereferencing.
	if r == nil || r.db == nil {
		return &repositories.PartnerScope{PartnerID: partnerID}, nil
	}

	cacheKey := fmt.Sprintf("partner_scope:%d", partnerID)

	if r.cache != nil {
		var cached repositories.PartnerScope
		if err := r.cache.Get(ctx, cacheKey, &cached); err == nil {
			cached.PartnerID = partnerID
			return &cached, nil
		}
	}

	scope, err := r.compute(ctx, partnerID)
	if err != nil {
		return nil, err
	}

	if r.cache != nil {
		if err := r.cache.Set(ctx, cacheKey, scope, PartnerScopeTTL); err != nil {
			r.logger.Warn("partner_scope_cache_set_failed", "partner_id", partnerID, "error", err)
		}
	}
	return scope, nil
}

// compute runs the 4 access-path lookups. They are small (each hits a dedicated
// index created by earlier migrations: idx_project_employees_project_created_by,
// idx_project_users_access_lookup, etc.) so we run them sequentially to keep the
// connection pool footprint low; goroutines would only add scheduling overhead.
func (r *PartnerScopeResolver) compute(ctx context.Context, partnerID uint) (*repositories.PartnerScope, error) {
	scope := &repositories.PartnerScope{PartnerID: partnerID}

	// Path (a): employees created by this partner.
	var ownedEmployeeIDs []uint
	if err := r.db.WithContext(ctx).
		Table("employees").
		Select("id").
		Where("created_by = ? AND deleted_at IS NULL", partnerID).
		Pluck("id", &ownedEmployeeIDs).Error; err != nil {
		return nil, fmt.Errorf("partner_scope owned employees: %w", err)
	}

	// Path (b): employees assigned by this partner to any project.
	var assignedEmployeeIDs []uint
	if err := r.db.WithContext(ctx).
		Table("project_employees").
		Select("DISTINCT employee_id").
		Where("created_by = ? AND deleted_at IS NULL", partnerID).
		Pluck("employee_id", &assignedEmployeeIDs).Error; err != nil {
		return nil, fmt.Errorf("partner_scope assigned employees: %w", err)
	}

	// Path (c): employees shared with this partner via employee_users.
	var sharedEmployeeIDs []uint
	if err := r.db.WithContext(ctx).
		Table("employee_users").
		Select("employee_id").
		Where("user_id = ? AND deleted_at IS NULL", partnerID).
		Pluck("employee_id", &sharedEmployeeIDs).Error; err != nil {
		return nil, fmt.Errorf("partner_scope shared employees: %w", err)
	}

	// Path (d): projects shared with this partner via project_users.
	var sharedProjectIDs []uint
	if err := r.db.WithContext(ctx).
		Table("project_users").
		Select("project_id").
		Where("user_id = ? AND deleted_at IS NULL", partnerID).
		Pluck("project_id", &sharedProjectIDs).Error; err != nil {
		return nil, fmt.Errorf("partner_scope shared projects: %w", err)
	}

	// Union all employee ID sources, dedupe, and sort.
	combinedEmployees := make([]uint, 0, len(ownedEmployeeIDs)+len(assignedEmployeeIDs)+len(sharedEmployeeIDs))
	combinedEmployees = append(combinedEmployees, ownedEmployeeIDs...)
	combinedEmployees = append(combinedEmployees, assignedEmployeeIDs...)
	combinedEmployees = append(combinedEmployees, sharedEmployeeIDs...)
	scope.EmployeeIDs = dedupeAndSortUint(combinedEmployees)
	scope.ProjectIDs = dedupeAndSortUint(sharedProjectIDs)

	return scope, nil
}

// dedupeAndSortUint returns the input slice with duplicates removed and values
// sorted ascending.
func dedupeAndSortUint(in []uint) []uint {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[uint]struct{}, len(in))
	out := make([]uint, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
