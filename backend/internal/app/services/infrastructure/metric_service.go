package infrastructure

import (
	"context"
	"sort"
	"time"

	"api-server/internal/domain"
)

type MetricService struct {
	repo      domain.APIMetricRepository
	auditRepo domain.AuditLogRepository
}

func NewMetricService(repo domain.APIMetricRepository, auditRepo domain.AuditLogRepository) *MetricService {
	return &MetricService{repo: repo, auditRepo: auditRepo}
}

// GetAPISummary returns aggregated API call statistics for all records
func (s *MetricService) GetAPISummary(ctx context.Context, since time.Time, sortBy string) ([]domain.APIMetricSummary, error) {
	return s.repo.GetAllSummary(ctx, since, sortBy)
}

func (s *MetricService) GetErrorBreakdownByUser(ctx context.Context, since time.Time, minStatusCode int, limit int) ([]domain.UserErrorBreakdown, error) {
	return s.repo.GetErrorBreakdownByUser(ctx, since, minStatusCode, limit)
}

func (s *MetricService) GetLatencyTrend(ctx context.Context, since time.Time, groupBy string) ([]domain.LatencyTrendPoint, error) {
	return s.repo.GetLatencyTrend(ctx, since, groupBy)
}

func (s *MetricService) GetSlowestEndpoints(ctx context.Context, since time.Time, limit int) ([]domain.SlowestEndpoint, error) {
	return s.repo.GetSlowestEndpoints(ctx, since, limit)
}

func (s *MetricService) GetRecentErrors(ctx context.Context, since time.Time, limit int) ([]domain.RecentError, error) {
	return s.repo.GetRecentErrors(ctx, since, limit)
}

func (s *MetricService) GetErrorCount(ctx context.Context, since time.Time) (int64, error) {
	return s.repo.GetErrorCount(ctx, since)
}

func (s *MetricService) GetTopEndpointIDs(ctx context.Context, since time.Time, limit int) ([]domain.EndpointInfo, error) {
	return s.repo.GetTopEndpointIDs(ctx, since, limit)
}

func (s *MetricService) GetEndpointLatencyTrend(ctx context.Context, endpointID uint, since time.Time, groupBy string) ([]domain.LatencyTrendPoint, error) {
	return s.repo.GetEndpointLatencyTrend(ctx, endpointID, since, groupBy)
}

// FailedLoginAttempt represents a single failed login attempt
type FailedLoginAttempt struct {
	ID                  uint            `json:"id"`
	UserID              uint            `json:"user_id"`
	UserName            string          `json:"user_name"`
	AttemptedIdentifier string          `json:"attempted_identifier"`
	IPAddress           string          `json:"ip_address"`
	Browser             string          `json:"browser"`
	Platform            string          `json:"platform"`
	Location            *FailedLoginGeo `json:"location,omitempty"`
	Reason              string          `json:"reason"`
	CreatedAt           time.Time       `json:"created_at"`
}

// FailedLoginGeo holds the resolved geographic location for a failed login attempt.
type FailedLoginGeo struct {
	Country string `json:"country"`
	City    string `json:"city"`
	Region  string `json:"region"`
}

// LoginIdentifierSummary represents aggregated failed login counts per identifier
type LoginIdentifierSummary struct {
	Identifier string    `json:"identifier"`
	Count      int       `json:"count"`
	LastSeen   time.Time `json:"last_seen"`
	Reason     string    `json:"reason"`
}

// FailedLoginsResponse contains both individual attempts and aggregated summary
type FailedLoginsResponse struct {
	Attempts []FailedLoginAttempt     `json:"attempts"`
	Summary  []LoginIdentifierSummary `json:"summary"`
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strVal(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// extractLocation pulls the nested "location" map from audit log metadata.
func extractLocation(meta map[string]interface{}) *FailedLoginGeo {
	raw, ok := meta["location"]
	if !ok || raw == nil {
		return nil
	}
	loc, ok := raw.(map[string]interface{})
	if !ok {
		return nil
	}
	country := strVal(loc["country"])
	city := strVal(loc["city"])
	region := strVal(loc["region"])
	if country == "" && city == "" && region == "" {
		return nil
	}
	return &FailedLoginGeo{
		Country: country,
		City:    city,
		Region:  region,
	}
}

// GetFailedLogins returns failed login attempts and their aggregated summary
func (s *MetricService) GetFailedLogins(ctx context.Context, since time.Time, limit int) (*FailedLoginsResponse, error) {
	filters := domain.AuditFilters{
		Action:    []domain.AuditAction{domain.AuditActionLogin},
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     limit * 3,
	}

	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	identifierMap := make(map[string]*LoginIdentifierSummary)
	var attempts []FailedLoginAttempt

	for _, log := range logs {
		meta, _ := log.GetMetadata()
		if meta == nil {
			continue
		}
		success, ok := meta["success"].(bool)
		if !ok || success {
			continue
		}

		identifier := strVal(meta["attempted_identifier"])
		if identifier == "" {
			identifier = log.Message
		}

		attempt := FailedLoginAttempt{
			ID:                  log.ID,
			UserID:              log.UserID,
			UserName:            log.UserFullname,
			AttemptedIdentifier: identifier,
			IPAddress:           derefStr(log.IPAddress),
			Browser:             derefStr(log.Browser),
			Platform:            derefStr(log.Platform),
			Location:            extractLocation(meta),
			Reason:              strVal(meta["reason"]),
			CreatedAt:           log.CreatedAt,
		}
		attempts = append(attempts, attempt)

		if existing, found := identifierMap[identifier]; found {
			existing.Count++
			if log.CreatedAt.After(existing.LastSeen) {
				existing.LastSeen = log.CreatedAt
			}
		} else {
			identifierMap[identifier] = &LoginIdentifierSummary{
				Identifier: identifier,
				Count:      1,
				LastSeen:   log.CreatedAt,
				Reason:     strVal(meta["reason"]),
			}
		}

		if len(attempts) >= limit {
			break
		}
	}

	summary := make([]LoginIdentifierSummary, 0, len(identifierMap))
	for _, v := range identifierMap {
		summary = append(summary, *v)
	}
	sort.Slice(summary, func(i, j int) bool {
		return summary[i].Count > summary[j].Count
	})

	return &FailedLoginsResponse{
		Attempts: attempts,
		Summary:  summary,
	}, nil
}

// GetBrowserPlatformStats returns unique user counts grouped by browser and platform from audit logs.
// Aggregation runs at the database (GROUP BY + COUNT DISTINCT); see AuditLogRepository.
func (s *MetricService) GetBrowserPlatformStats(ctx context.Context, since time.Time) ([]domain.BrowserPlatformStat, error) {
	return s.auditRepo.GetBrowserPlatformStats(ctx, since)
}

// GetBrowserPlatformUsers returns users who used a specific browser+platform combination.
// Aggregation runs at the database; see AuditLogRepository.
func (s *MetricService) GetBrowserPlatformUsers(ctx context.Context, browser, platform string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return s.auditRepo.GetBrowserPlatformUsers(ctx, browser, platform, since)
}

// GetFailedLoginsByIdentifier returns failed login attempts for a specific identifier
func (s *MetricService) GetFailedLoginsByIdentifier(ctx context.Context, identifier string, since time.Time, limit int) ([]FailedLoginAttempt, error) {
	filters := domain.AuditFilters{
		Action:    []domain.AuditAction{domain.AuditActionLogin},
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     limit * 3,
	}

	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	var attempts []FailedLoginAttempt
	for _, log := range logs {
		meta, _ := log.GetMetadata()
		if meta == nil {
			continue
		}
		success, ok := meta["success"].(bool)
		if !ok || success {
			continue
		}

		logIdentifier := strVal(meta["attempted_identifier"])
		if logIdentifier != identifier {
			continue
		}

		attempts = append(attempts, FailedLoginAttempt{
			ID:                  log.ID,
			UserID:              log.UserID,
			UserName:            log.UserFullname,
			AttemptedIdentifier: logIdentifier,
			IPAddress:           derefStr(log.IPAddress),
			Browser:             derefStr(log.Browser),
			Platform:            derefStr(log.Platform),
			Location:            extractLocation(meta),
			Reason:              strVal(meta["reason"]),
			CreatedAt:           log.CreatedAt,
		})

		if len(attempts) >= limit {
			break
		}
	}

	return attempts, nil
}

// ─── OS / Browser grouped stats ──────────────────────────────────────────────

// GetOSGroupedStats returns unique user counts grouped by OS family (iOS, Android, etc.)
// with per-platform-version breakdowns. Aggregation runs at the database; see AuditLogRepository.
func (s *MetricService) GetOSGroupedStats(ctx context.Context, since time.Time) ([]domain.OSGroupStat, error) {
	return s.auditRepo.GetOSGroupedStats(ctx, since)
}

// GetBrowserGroupedStats returns unique user counts grouped by browser family (Safari, Chrome, etc.)
// with per-browser-version breakdowns. Aggregation runs at the database; see AuditLogRepository.
func (s *MetricService) GetBrowserGroupedStats(ctx context.Context, since time.Time) ([]domain.BrowserGroupStat, error) {
	return s.auditRepo.GetBrowserGroupedStats(ctx, since)
}

// GetOSGroupedUsers returns users who used any version within the given OS family.
// Aggregation runs at the database; see AuditLogRepository.
func (s *MetricService) GetOSGroupedUsers(ctx context.Context, osFamily string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return s.auditRepo.GetOSGroupedUsers(ctx, osFamily, since)
}

// GetBrowserGroupedUsers returns users who used any version within the given browser family.
// Aggregation runs at the database; see AuditLogRepository.
func (s *MetricService) GetBrowserGroupedUsers(ctx context.Context, browserFamily string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return s.auditRepo.GetBrowserGroupedUsers(ctx, browserFamily, since)
}
