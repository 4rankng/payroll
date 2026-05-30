package infrastructure

import (
	"context"
	"sort"
	"strings"
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

// BrowserPlatformStat represents unique user count for a browser+platform combination
type BrowserPlatformStat struct {
	Browser      string `json:"browser"`
	Platform     string `json:"platform"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// GetBrowserPlatformStats returns unique user counts grouped by browser and platform from audit logs
func (s *MetricService) GetBrowserPlatformStats(ctx context.Context, since time.Time) ([]BrowserPlatformStat, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     5000,
	}

	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type key struct{ browser, platform string }
	type entry struct {
		users   map[uint]struct{}
		actions int
	}
	grouped := make(map[key]*entry)

	for _, log := range logs {
		b := derefStr(log.Browser)
		p := derefStr(log.Platform)
		if b == "" {
			b = "Unknown"
		}
		if p == "" {
			p = "Unknown"
		}
		k := key{b, p}
		e, ok := grouped[k]
		if !ok {
			e = &entry{users: make(map[uint]struct{})}
			grouped[k] = e
		}
		e.users[log.UserID] = struct{}{}
		e.actions++
	}

	stats := make([]BrowserPlatformStat, 0, len(grouped))
	for k, e := range grouped {
		stats = append(stats, BrowserPlatformStat{
			Browser:      k.browser,
			Platform:     k.platform,
			UniqueUsers:  len(e.users),
			TotalActions: e.actions,
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].UniqueUsers != stats[j].UniqueUsers {
			return stats[i].UniqueUsers > stats[j].UniqueUsers
		}
		return stats[i].TotalActions > stats[j].TotalActions
	})

	return stats, nil
}

// BrowserPlatformUser represents a user who used a specific browser+platform combination
type BrowserPlatformUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	Actions  int    `json:"actions"`
	LastSeen string `json:"last_seen"`
}

// GetBrowserPlatformUsers returns users who used a specific browser+platform combination
func (s *MetricService) GetBrowserPlatformUsers(ctx context.Context, browser, platform string, since time.Time) ([]BrowserPlatformUser, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     10000,
	}

	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type userInfo struct {
		username string
		fullname string
		role     string
		actions  int
		lastSeen time.Time
	}
	users := make(map[uint]*userInfo)

	for _, log := range logs {
		b := derefStr(log.Browser)
		p := derefStr(log.Platform)
		if b == "" {
			b = "Unknown"
		}
		if p == "" {
			p = "Unknown"
		}
		if b != browser || p != platform {
			continue
		}
		u, ok := users[log.UserID]
		if !ok {
			u = &userInfo{
				username: log.UserUsername,
				fullname: log.UserFullname,
				role:     log.UserRole,
			}
			users[log.UserID] = u
		}
		u.actions++
		if log.CreatedAt.After(u.lastSeen) {
			u.lastSeen = log.CreatedAt
		}
	}

	result := make([]BrowserPlatformUser, 0, len(users))
	for uid, info := range users {
		result = append(result, BrowserPlatformUser{
			UserID:   uid,
			Username: info.username,
			Fullname: info.fullname,
			Role:     info.role,
			Actions:  info.actions,
			LastSeen: info.lastSeen.Format(time.RFC3339),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Actions > result[j].Actions
	})

	return result, nil
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

// extractOSFamily normalises a raw platform string ("iOS 16.7", "Android 12")
// into a canonical family name ("iOS", "Android", …).
func extractOSFamily(platform string) string {
	p := strings.ToLower(strings.TrimSpace(platform))
	switch {
	case strings.HasPrefix(p, "ios"):
		return "iOS"
	case strings.HasPrefix(p, "android"):
		return "Android"
	case strings.HasPrefix(p, "windows"):
		return "Windows"
	case strings.HasPrefix(p, "macos"), strings.HasPrefix(p, "mac os"):
		return "macOS"
	case strings.HasPrefix(p, "linux"):
		return "Linux"
	default:
		return "Other"
	}
}

// extractBrowserFamily normalises a raw browser string ("Safari 15E148", "Chrome 148")
// into a canonical family name ("Safari", "Chrome", …).
func extractBrowserFamily(browser string) string {
	b := strings.ToLower(strings.TrimSpace(browser))
	switch {
	case strings.HasPrefix(b, "safari"):
		return "Safari"
	case strings.HasPrefix(b, "chrome"):
		return "Chrome"
	case strings.HasPrefix(b, "firefox"):
		return "Firefox"
	case strings.HasPrefix(b, "edge"):
		return "Edge"
	case strings.HasPrefix(b, "opera"):
		return "Opera"
	default:
		return "Other"
	}
}

// VersionStat holds user/action counts for a single version within a family.
type VersionStat struct {
	Version      string `json:"version"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// OSGroupStat holds aggregated stats for one OS family with per-version breakdown.
type OSGroupStat struct {
	OSFamily     string        `json:"os_family"`
	UniqueUsers  int           `json:"unique_users"`
	TotalActions int           `json:"total_actions"`
	Versions     []VersionStat `json:"versions"`
}

// BrowserGroupStat holds aggregated stats for one browser family with per-version breakdown.
type BrowserGroupStat struct {
	BrowserFamily string        `json:"browser_family"`
	UniqueUsers   int           `json:"unique_users"`
	TotalActions  int           `json:"total_actions"`
	Versions      []VersionStat `json:"versions"`
}

// GetOSGroupedStats returns unique user counts grouped by OS family (iOS, Android, etc.)
// with per-platform-version breakdowns. Users are deduplicated within each family.
func (s *MetricService) GetOSGroupedStats(ctx context.Context, since time.Time) ([]OSGroupStat, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     5000,
	}
	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type versionKey struct{ family, version string }
	type versionEntry struct {
		users   map[uint]struct{}
		actions int
	}

	familyUsers := make(map[string]map[uint]struct{})   // family → unique users
	familyActions := make(map[string]int)                // family → total actions
	versionData := make(map[versionKey]*versionEntry)    // (family, version) → entry

	for _, log := range logs {
		p := derefStr(log.Platform)
		if p == "" {
			p = "Unknown"
		}
		family := extractOSFamily(p)
		vk := versionKey{family, p}

		if familyUsers[family] == nil {
			familyUsers[family] = make(map[uint]struct{})
		}
		familyUsers[family][log.UserID] = struct{}{}
		familyActions[family]++

		if versionData[vk] == nil {
			versionData[vk] = &versionEntry{users: make(map[uint]struct{})}
		}
		versionData[vk].users[log.UserID] = struct{}{}
		versionData[vk].actions++
	}

	// Build version lists per family
	versionsByFamily := make(map[string][]VersionStat)
	for vk, ve := range versionData {
		versionsByFamily[vk.family] = append(versionsByFamily[vk.family], VersionStat{
			Version:      vk.version,
			UniqueUsers:  len(ve.users),
			TotalActions: ve.actions,
		})
	}
	for family := range versionsByFamily {
		sort.Slice(versionsByFamily[family], func(i, j int) bool {
			if versionsByFamily[family][i].UniqueUsers != versionsByFamily[family][j].UniqueUsers {
				return versionsByFamily[family][i].UniqueUsers > versionsByFamily[family][j].UniqueUsers
			}
			return versionsByFamily[family][i].TotalActions > versionsByFamily[family][j].TotalActions
		})
	}

	result := make([]OSGroupStat, 0, len(familyUsers))
	for family, users := range familyUsers {
		result = append(result, OSGroupStat{
			OSFamily:     family,
			UniqueUsers:  len(users),
			TotalActions: familyActions[family],
			Versions:     versionsByFamily[family],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UniqueUsers != result[j].UniqueUsers {
			return result[i].UniqueUsers > result[j].UniqueUsers
		}
		return result[i].TotalActions > result[j].TotalActions
	})
	return result, nil
}

// GetBrowserGroupedStats returns unique user counts grouped by browser family (Safari, Chrome, etc.)
// with per-browser-version breakdowns. Users are deduplicated within each family.
func (s *MetricService) GetBrowserGroupedStats(ctx context.Context, since time.Time) ([]BrowserGroupStat, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     5000,
	}
	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type versionKey struct{ family, version string }
	type versionEntry struct {
		users   map[uint]struct{}
		actions int
	}

	familyUsers := make(map[string]map[uint]struct{})
	familyActions := make(map[string]int)
	versionData := make(map[versionKey]*versionEntry)

	for _, log := range logs {
		b := derefStr(log.Browser)
		if b == "" {
			b = "Unknown"
		}
		family := extractBrowserFamily(b)
		vk := versionKey{family, b}

		if familyUsers[family] == nil {
			familyUsers[family] = make(map[uint]struct{})
		}
		familyUsers[family][log.UserID] = struct{}{}
		familyActions[family]++

		if versionData[vk] == nil {
			versionData[vk] = &versionEntry{users: make(map[uint]struct{})}
		}
		versionData[vk].users[log.UserID] = struct{}{}
		versionData[vk].actions++
	}

	versionsByFamily := make(map[string][]VersionStat)
	for vk, ve := range versionData {
		versionsByFamily[vk.family] = append(versionsByFamily[vk.family], VersionStat{
			Version:      vk.version,
			UniqueUsers:  len(ve.users),
			TotalActions: ve.actions,
		})
	}
	for family := range versionsByFamily {
		sort.Slice(versionsByFamily[family], func(i, j int) bool {
			if versionsByFamily[family][i].UniqueUsers != versionsByFamily[family][j].UniqueUsers {
				return versionsByFamily[family][i].UniqueUsers > versionsByFamily[family][j].UniqueUsers
			}
			return versionsByFamily[family][i].TotalActions > versionsByFamily[family][j].TotalActions
		})
	}

	result := make([]BrowserGroupStat, 0, len(familyUsers))
	for family, users := range familyUsers {
		result = append(result, BrowserGroupStat{
			BrowserFamily: family,
			UniqueUsers:   len(users),
			TotalActions:  familyActions[family],
			Versions:      versionsByFamily[family],
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UniqueUsers != result[j].UniqueUsers {
			return result[i].UniqueUsers > result[j].UniqueUsers
		}
		return result[i].TotalActions > result[j].TotalActions
	})
	return result, nil
}

// GetOSGroupedUsers returns users who used any version within the given OS family.
func (s *MetricService) GetOSGroupedUsers(ctx context.Context, osFamily string, since time.Time) ([]BrowserPlatformUser, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     10000,
	}
	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type userInfo struct {
		username string
		fullname string
		role     string
		actions  int
		lastSeen time.Time
	}
	users := make(map[uint]*userInfo)

	for _, log := range logs {
		p := derefStr(log.Platform)
		if p == "" {
			p = "Unknown"
		}
		if extractOSFamily(p) != osFamily {
			continue
		}
		u, ok := users[log.UserID]
		if !ok {
			u = &userInfo{username: log.UserUsername, fullname: log.UserFullname, role: log.UserRole}
			users[log.UserID] = u
		}
		u.actions++
		if log.CreatedAt.After(u.lastSeen) {
			u.lastSeen = log.CreatedAt
		}
	}

	result := make([]BrowserPlatformUser, 0, len(users))
	for uid, info := range users {
		result = append(result, BrowserPlatformUser{
			UserID:   uid,
			Username: info.username,
			Fullname: info.fullname,
			Role:     info.role,
			Actions:  info.actions,
			LastSeen: info.lastSeen.Format(time.RFC3339),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Actions > result[j].Actions })
	return result, nil
}

// GetBrowserGroupedUsers returns users who used any version within the given browser family.
func (s *MetricService) GetBrowserGroupedUsers(ctx context.Context, browserFamily string, since time.Time) ([]BrowserPlatformUser, error) {
	filters := domain.AuditFilters{
		FromDate:  &since,
		SortBy:    "created_at",
		SortOrder: "DESC",
		Limit:     10000,
	}
	logs, err := s.auditRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	type userInfo struct {
		username string
		fullname string
		role     string
		actions  int
		lastSeen time.Time
	}
	users := make(map[uint]*userInfo)

	for _, log := range logs {
		b := derefStr(log.Browser)
		if b == "" {
			b = "Unknown"
		}
		if extractBrowserFamily(b) != browserFamily {
			continue
		}
		u, ok := users[log.UserID]
		if !ok {
			u = &userInfo{username: log.UserUsername, fullname: log.UserFullname, role: log.UserRole}
			users[log.UserID] = u
		}
		u.actions++
		if log.CreatedAt.After(u.lastSeen) {
			u.lastSeen = log.CreatedAt
		}
	}

	result := make([]BrowserPlatformUser, 0, len(users))
	for uid, info := range users {
		result = append(result, BrowserPlatformUser{
			UserID:   uid,
			Username: info.username,
			Fullname: info.fullname,
			Role:     info.role,
			Actions:  info.actions,
			LastSeen: info.lastSeen.Format(time.RFC3339),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Actions > result[j].Actions })
	return result, nil
}
