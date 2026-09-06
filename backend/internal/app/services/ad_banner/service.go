// Package ad_banner implements the employee ad-banner campaign service:
// admin CRUD with audit events, per-employee resolution for the portal's
// single ad slot, and the append-only CTA click ledger.
package ad_banner

import (
	"context"
	"log/slog"

	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/clock"
	auditctx "api-server/internal/pkg/context"
)

const (
	// liveListCacheKey caches the live campaign list (already in resolution
	// order). Campaign volume is tiny, so one entry serves every employee;
	// targeting is filtered per request in memory.
	liveListCacheKey = "ad_banners:live"
	// cacheInvalidationPattern matches every ad-banner cache entry.
	cacheInvalidationPattern = "ad_banners:*"
)

// Service orchestrates ad banner campaigns.
type Service struct {
	BannerRepo          domain.AdBannerRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	EmployeeRepo        domain.EmployeeRepository
	events              domain.EventBus
	cache               domain.CacheServiceUseCase
	logger              *slog.Logger
}

func NewService(
	bannerRepo domain.AdBannerRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	employeeRepo domain.EmployeeRepository,
	events domain.EventBus,
	cache domain.CacheServiceUseCase,
	logger *slog.Logger,
) *Service {
	return &Service{
		BannerRepo:          bannerRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		EmployeeRepo:        employeeRepo,
		events:              events,
		cache:               cache,
		logger:              logger,
	}
}

// ResolveForEmployeeByUser resolves the single winning campaign for the
// authenticated user, or nil when no campaign addresses them. Resolution:
// employee's active project assignments → live campaigns (priority DESC,
// created_at DESC) → first campaign targeting any of those projects.
//
// An employee with no active project assignment gets nil, not a broadcast:
// they are between assignments and targeting them is meaningless.
func (s *Service) ResolveForEmployeeByUser(ctx context.Context, userID uint) (*domain.AdBanner, error) {
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	if employee == nil {
		return nil, nil
	}

	projects, err := s.ProjectEmployeeRepo.GetActiveProjectsForEmployee(ctx, employee.ID)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, nil
	}

	projectIDs := make([]uint, 0, len(projects))
	for _, p := range projects {
		projectIDs = append(projectIDs, p.ID)
	}

	banners, err := s.listLive(ctx)
	if err != nil {
		return nil, err
	}

	// listLive is already in resolution order; the first targeting match wins.
	for _, banner := range banners {
		if banner.TargetsAnyProject(projectIDs) {
			return banner, nil
		}
	}
	return nil, nil
}

// listLive returns live campaigns through a short-lived microcache.
func (s *Service) listLive(ctx context.Context) ([]*domain.AdBanner, error) {
	var cached []*domain.AdBanner
	if err := s.cache.Get(ctx, liveListCacheKey, &cached); err == nil && len(cached) > 0 {
		return cached, nil
	}

	banners, err := s.BannerRepo.ListLiveAt(ctx, clock.Now())
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, liveListCacheKey, banners, constants.AdBannerCacheTTL)
	return banners, nil
}

// Create validates and persists a new campaign.
func (s *Service) Create(ctx context.Context, banner *domain.AdBanner) (*domain.AdBanner, error) {
	if err := banner.Validate(); err != nil {
		return nil, err
	}
	if err := s.BannerRepo.Create(ctx, banner); err != nil {
		s.logger.Error("Failed to create ad banner", "title", banner.Title, "error", err)
		return nil, domain.NewInternalError("Không thể tạo chiến dịch quảng cáo", err)
	}

	s.invalidateAfterCommit(ctx)
	if err := s.events.Publish(ctx, domain.NewAdBannerCreatedEvent(ctx, banner, auditctx.GetUserIDOrZero(ctx), actorName(ctx))); err != nil {
		s.logger.Warn("Failed to publish AdBannerCreatedEvent", "banner_id", banner.ID, "error", err)
	}
	return banner, nil
}

// Update loads the existing campaign, applies the new content/window, and
// saves. The resulting updated_at bump is the campaign version: the portal
// re-shows the sheet once per version (a republish, by design).
func (s *Service) Update(ctx context.Context, id uint, updated *domain.AdBanner) (*domain.AdBanner, error) {
	existing, err := s.BannerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	existing.Title = updated.Title
	existing.Body = updated.Body
	existing.Bullets = updated.Bullets
	existing.CTAs = updated.CTAs
	existing.Footer = updated.Footer
	existing.TargetProjectIDs = updated.TargetProjectIDs
	existing.Priority = updated.Priority
	existing.StartsAt = updated.StartsAt
	existing.EndsAt = updated.EndsAt
	existing.IsActive = updated.IsActive

	if err := existing.Validate(); err != nil {
		return nil, err
	}
	if err := s.BannerRepo.Update(ctx, existing); err != nil {
		s.logger.Error("Failed to update ad banner", "id", id, "error", err)
		return nil, domain.NewInternalError("Không thể cập nhật chiến dịch quảng cáo", err)
	}

	s.invalidateAfterCommit(ctx)
	if err := s.events.Publish(ctx, domain.NewAdBannerUpdatedEvent(ctx, existing, auditctx.GetUserIDOrZero(ctx), actorName(ctx))); err != nil {
		s.logger.Warn("Failed to publish AdBannerUpdatedEvent", "banner_id", existing.ID, "error", err)
	}
	return existing, nil
}

// Delete soft-deletes a campaign.
func (s *Service) Delete(ctx context.Context, id uint) error {
	existing, err := s.BannerRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := s.BannerRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete ad banner", "id", id, "error", err)
		return domain.NewInternalError("Không thể xóa chiến dịch quảng cáo", err)
	}

	s.invalidateAfterCommit(ctx)
	if err := s.events.Publish(ctx, domain.NewAdBannerDeletedEvent(ctx, existing, auditctx.GetUserIDOrZero(ctx), actorName(ctx))); err != nil {
		s.logger.Warn("Failed to publish AdBannerDeletedEvent", "banner_id", existing.ID, "error", err)
	}
	return nil
}

// ListWithStats returns every campaign plus its per-CTA click counts.
func (s *Service) ListWithStats(ctx context.Context) ([]*domain.AdBanner, map[uint]map[int]int64, error) {
	banners, err := s.BannerRepo.List(ctx, domain.AdBannerFilters{})
	if err != nil {
		return nil, nil, err
	}

	ids := make([]uint, 0, len(banners))
	for _, b := range banners {
		ids = append(ids, b.ID)
	}
	counts, err := s.BannerRepo.CTAClickCounts(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return banners, counts, nil
}

// RecordCTAClick appends a click for the authenticated user. Dead banners and
// unknown employees are silently ignored — a click-tracking outage must never
// block a worker from reaching the hotline.
func (s *Service) RecordCTAClick(ctx context.Context, bannerID, userID uint, ctaIndex int) error {
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil
		}
		return err
	}

	banner, err := s.BannerRepo.GetByID(ctx, bannerID)
	if err != nil {
		if domain.IsNotFoundError(err) {
			return nil
		}
		return err
	}
	if !banner.IsLiveAt(clock.Now()) {
		return nil
	}

	if err := s.BannerRepo.RecordCTAClick(ctx, bannerID, employee.ID, ctaIndex); err != nil {
		return err
	}
	s.logger.Info("Ad banner CTA clicked", "banner_id", bannerID, "employee_id", employee.ID, "cta_index", ctaIndex)
	return nil
}

// invalidateAfterCommit drops the resolve cache AFTER the mutation has
// succeeded (ADR-007): these are single-statement mutations, so the repo
// return already implies commit.
func (s *Service) invalidateAfterCommit(ctx context.Context) {
	if err := s.cache.DeletePattern(ctx, cacheInvalidationPattern); err != nil {
		s.logger.Warn("Failed to invalidate ad banner cache", "error", err)
	}
}

func actorName(ctx context.Context) string {
	name := auditctx.GetFullName(ctx)
	if name == "" {
		return "Unknown"
	}
	return name
}
