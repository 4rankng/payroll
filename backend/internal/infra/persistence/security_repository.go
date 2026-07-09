package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"sort"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

var allowedAuditSortFields = map[string]bool{
	"created_at": true, "action": true, "entity_type": true,
	"user_id": true, "id": true, "ip_address": true,
}

var allowedBlacklistSortFields = map[string]bool{
	"blacklisted_at": true, "user_id": true, "expires_at": true, "id": true,
}

var allowedSortOrders = map[string]bool{"ASC": true, "DESC": true}

// BlacklistedTokenRepository
type BlacklistedTokenRepository struct {
	*BaseRepository
}

func NewBlacklistedTokenRepository(db *Database) domain.BlacklistedTokenRepository {
	return &BlacklistedTokenRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *BlacklistedTokenRepository) Create(ctx context.Context, token *domain.BlacklistedToken) error {
	return r.DB.WithContext(ctx).Create(token).Error
}

func (r *BlacklistedTokenRepository) GetByJTI(ctx context.Context, jti string) (*domain.BlacklistedToken, error) {
	var token domain.BlacklistedToken
	err := r.DB.WithContext(ctx).
		Select("blacklisted_tokens.*, users.fullname as user_fullname").
		Joins("LEFT JOIN users ON blacklisted_tokens.user_id = users.id AND users.deleted_at IS NULL").
		Where("blacklisted_tokens.token_jti = ?", jti).
		First(&token).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("blacklisted token not found")
		}
		return nil, err
	}

	return &token, nil
}

func (r *BlacklistedTokenRepository) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	var exists bool
	err := r.DB.WithContext(ctx).
		Model(&domain.BlacklistedToken{}).
		Select("COUNT(*) > 0").
		Where("token_jti = ? AND expires_at > ?", jti, clock.Now()).
		Scan(&exists).Error

	return exists, err
}

func (r *BlacklistedTokenRepository) DeleteExpired(ctx context.Context) error {
	return r.DB.WithContext(ctx).
		Where("expires_at <= ?", clock.Now()).
		Delete(&domain.BlacklistedToken{}).Error
}

func (r *BlacklistedTokenRepository) List(ctx context.Context, filters domain.BlacklistFilters) ([]*domain.BlacklistedToken, error) {
	var tokens []*domain.BlacklistedToken
	query := r.DB.WithContext(ctx).
		Select("blacklisted_tokens.*, users.fullname as user_fullname").
		Joins("LEFT JOIN users ON blacklisted_tokens.user_id = users.id AND users.deleted_at IS NULL")

	query = r.applyBlacklistFilters(query, filters)

	sortBy := "blacklisted_tokens.blacklisted_at"
	if filters.SortBy != "" && allowedBlacklistSortFields[filters.SortBy] {
		sortBy = "blacklisted_tokens." + filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		if upper := strings.ToUpper(filters.SortOrder); allowedSortOrders[upper] {
			sortOrder = upper
		}
	}

	query = query.Order(common.SanitizeSortColumn(sortBy, "blacklisted_tokens.blacklisted_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC"))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&tokens).Error
	return tokens, err
}

func (r *BlacklistedTokenRepository) Count(ctx context.Context, filters domain.BlacklistFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.BlacklistedToken{})
	query = r.applyBlacklistFilters(query, filters)
	err := query.Count(&count).Error
	return count, err
}

func (r *BlacklistedTokenRepository) GetByUser(ctx context.Context, userID uint) ([]*domain.BlacklistedToken, error) {
	var tokens []*domain.BlacklistedToken
	err := r.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("blacklisted_at DESC").
		Find(&tokens).Error

	return tokens, err
}

func (r *BlacklistedTokenRepository) BlacklistToken(ctx context.Context, jti string, userID uint, expiresAt time.Time, reason domain.BlacklistReason) error {
	token := &domain.BlacklistedToken{
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Reason:    reason,
	}

	return r.DB.WithContext(ctx).Create(token).Error
}

func (r *BlacklistedTokenRepository) applyBlacklistFilters(query *gorm.DB, filters domain.BlacklistFilters) *gorm.DB {
	if filters.UserID != nil {
		query = query.Where("user_id = ?", *filters.UserID)
	}

	if len(filters.Reason) > 0 {
		query = query.Where("reason IN ?", filters.Reason)
	}

	if filters.FromDate != nil {
		query = query.Where("blacklisted_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("blacklisted_at <= ?", *filters.ToDate)
	}

	if filters.Expired != nil {
		if *filters.Expired {
			query = query.Where("expires_at <= ?", clock.Now())
		} else {
			query = query.Where("expires_at > ?", clock.Now())
		}
	}

	return query
}

// AuditLogRepository
type AuditLogRepository struct {
	*BaseRepository
}

func NewAuditLogRepository(db *Database) domain.AuditLogRepository {
	return &AuditLogRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	return r.DB.WithContext(ctx).Create(log).Error
}

func (r *AuditLogRepository) GetByID(ctx context.Context, id uint) (*domain.AuditLog, error) {
	var log domain.AuditLog
	err := r.DB.WithContext(ctx).
		Select("audit_logs.*, users.fullname as user_fullname, users.username as user_username, users.role as user_role").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id AND users.deleted_at IS NULL").
		Where("audit_logs.id = ?", id).
		First(&log).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("audit log not found")
		}
		return nil, err
	}

	return &log, nil
}

func (r *AuditLogRepository) List(ctx context.Context, filters domain.AuditFilters) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	query := r.DB.WithContext(ctx).
		Select("audit_logs.*, users.fullname as user_fullname, users.username as user_username, users.role as user_role").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id AND users.deleted_at IS NULL")

	query = r.applyAuditFilters(query, filters)

	sortBy := "audit_logs.created_at"
	if filters.SortBy != "" && allowedAuditSortFields[filters.SortBy] {
		sortBy = "audit_logs." + filters.SortBy
	}

	sortOrder := "DESC"
	if filters.SortOrder != "" {
		if upper := strings.ToUpper(filters.SortOrder); allowedSortOrders[upper] {
			sortOrder = upper
		}
	}

	query = query.Order(common.SanitizeSortColumn(sortBy, "audit_logs.created_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC"))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&logs).Error
	return logs, err
}

func (r *AuditLogRepository) Count(ctx context.Context, filters domain.AuditFilters) (int64, error) {
	var count int64
	query := r.DB.WithContext(ctx).Model(&domain.AuditLog{})
	query = r.applyAuditFilters(query, filters)
	err := query.Count(&count).Error
	return count, err
}

func (r *AuditLogRepository) DeleteOldLogs(ctx context.Context, olderThan time.Time) error {
	return r.DB.WithContext(ctx).
		Where("created_at < ?", olderThan).
		Delete(&domain.AuditLog{}).Error
}

func (r *AuditLogRepository) applyAuditFilters(query *gorm.DB, filters domain.AuditFilters) *gorm.DB {
	if filters.UserID != nil {
		query = query.Where("audit_logs.user_id = ?", *filters.UserID)
	}
	if len(filters.Action) > 0 {
		query = query.Where("audit_logs.action IN ?", filters.Action)
	}
	if len(filters.EntityType) > 0 {
		query = query.Where("audit_logs.entity_type IN ?", filters.EntityType)
	}
	if filters.EntityID != nil {
		query = query.Where("audit_logs.entity_id = ?", *filters.EntityID)
	}
	if filters.IPAddress != nil {
		query = query.Where("audit_logs.ip_address = ?", *filters.IPAddress)
	}
	if filters.FromDate != nil {
		query = query.Where("audit_logs.created_at >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("audit_logs.created_at <= ?", *filters.ToDate)
	}
	return query
}

// SQL fragments for the System Health aggregations. These normalise the raw
// audit columns the same way the previous in-memory grouping did: NULL or empty
// becomes "Unknown", and a CASE maps the raw value to a canonical family. Kept as
// reusable constants so the derivation lives in exactly one place.
const (
	auditPlatformNorm = "COALESCE(NULLIF(TRIM(audit_logs.platform), ''), 'Unknown')"
	auditBrowserNorm  = "COALESCE(NULLIF(TRIM(audit_logs.browser), ''), 'Unknown')"

	auditOSFamilyExpr = "CASE " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'ios%' THEN 'iOS' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'android%' THEN 'Android' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'windows%' THEN 'Windows' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'macos%' THEN 'macOS' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'mac os%' THEN 'macOS' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.platform, ''))) LIKE 'linux%' THEN 'Linux' " +
		"ELSE 'Other' END"

	auditBrowserFamilyExpr = "CASE " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.browser, ''))) LIKE 'safari%' THEN 'Safari' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.browser, ''))) LIKE 'chrome%' THEN 'Chrome' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.browser, ''))) LIKE 'firefox%' THEN 'Firefox' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.browser, ''))) LIKE 'edge%' THEN 'Edge' " +
		"WHEN LOWER(TRIM(COALESCE(audit_logs.browser, ''))) LIKE 'opera%' THEN 'Opera' " +
		"ELSE 'Other' END"
)

// GetBrowserPlatformStats returns per browser+platform unique-user and action counts.
func (r *AuditLogRepository) GetBrowserPlatformStats(ctx context.Context, since time.Time) ([]domain.BrowserPlatformStat, error) {
	var stats []domain.BrowserPlatformStat
	err := r.DB.WithContext(ctx).
		Table("audit_logs").
		Select(auditBrowserNorm+" AS browser, "+
			auditPlatformNorm+" AS platform, "+
			"COUNT(DISTINCT audit_logs.user_id) AS unique_users, "+
			"COUNT(*) AS total_actions").
		Where("audit_logs.created_at >= ?", since).
		Group(auditBrowserNorm + ", " + auditPlatformNorm).
		Order("unique_users DESC, total_actions DESC").
		Scan(&stats).Error
	return stats, err
}

// browserPlatformUserRow is the scan target for the per-user aggregations.
type browserPlatformUserRow struct {
	UserID   uint      `gorm:"column:user_id"`
	Username string    `gorm:"column:username"`
	Fullname string    `gorm:"column:fullname"`
	Role     string    `gorm:"column:role"`
	Actions  int       `gorm:"column:actions"`
	LastSeen time.Time `gorm:"column:last_seen"`
}

// usersByCondition returns per-user action counts for audit rows matching the
// given extra WHERE condition (used to scope to a browser+platform pair or a
// derived OS/browser family). lastSeen is formatted to RFC3339 to preserve the
// existing response shape.
func (r *AuditLogRepository) usersByCondition(ctx context.Context, since time.Time, condition string, args ...any) ([]domain.BrowserPlatformUser, error) {
	var rows []browserPlatformUserRow
	q := r.DB.WithContext(ctx).
		Table("audit_logs").
		Select("audit_logs.user_id, users.username, users.fullname, users.role, "+
			"COUNT(*) AS actions, MAX(audit_logs.created_at) AS last_seen").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id AND users.deleted_at IS NULL").
		Where("audit_logs.created_at >= ?", since)
	if condition != "" {
		q = q.Where(condition, args...)
	}
	if err := q.
		Group("audit_logs.user_id, users.username, users.fullname, users.role").
		Order("actions DESC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]domain.BrowserPlatformUser, 0, len(rows))
	for _, row := range rows {
		result = append(result, domain.BrowserPlatformUser{
			UserID:   row.UserID,
			Username: row.Username,
			Fullname: row.Fullname,
			Role:     row.Role,
			Actions:  row.Actions,
			LastSeen: row.LastSeen.Format(time.RFC3339),
		})
	}
	return result, nil
}

// GetBrowserPlatformUsers returns users who used a specific browser+platform combination.
func (r *AuditLogRepository) GetBrowserPlatformUsers(ctx context.Context, browser, platform string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return r.usersByCondition(ctx, since,
		auditBrowserNorm+" = ? AND "+auditPlatformNorm+" = ?", browser, platform)
}

// GetOSGroupedUsers returns users who used any version within the given OS family.
func (r *AuditLogRepository) GetOSGroupedUsers(ctx context.Context, osFamily string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return r.usersByCondition(ctx, since, auditOSFamilyExpr+" = ?", osFamily)
}

// GetBrowserGroupedUsers returns users who used any version within the given browser family.
func (r *AuditLogRepository) GetBrowserGroupedUsers(ctx context.Context, browserFamily string, since time.Time) ([]domain.BrowserPlatformUser, error) {
	return r.usersByCondition(ctx, since, auditBrowserFamilyExpr+" = ?", browserFamily)
}

// familyStatRow is the family-level aggregation scan target.
type familyStatRow struct {
	Family       string `gorm:"column:family"`
	UniqueUsers  int    `gorm:"column:unique_users"`
	TotalActions int    `gorm:"column:total_actions"`
}

// versionStatRow is the (family, version) aggregation scan target.
type versionStatRow struct {
	Family       string `gorm:"column:family"`
	Version      string `gorm:"column:version"`
	UniqueUsers  int    `gorm:"column:unique_users"`
	TotalActions int    `gorm:"column:total_actions"`
}

// familyStats returns unique-user/action counts grouped by a derived family.
func (r *AuditLogRepository) familyStats(ctx context.Context, since time.Time, familyExpr string) ([]familyStatRow, error) {
	var rows []familyStatRow
	err := r.DB.WithContext(ctx).
		Table("audit_logs").
		Select(familyExpr+" AS family, "+
			"COUNT(DISTINCT audit_logs.user_id) AS unique_users, "+
			"COUNT(*) AS total_actions").
		Where("audit_logs.created_at >= ?", since).
		Group(familyExpr).
		Scan(&rows).Error
	return rows, err
}

// versionStats returns unique-user/action counts grouped by (derived family, raw version).
func (r *AuditLogRepository) versionStats(ctx context.Context, since time.Time, familyExpr, versionExpr string) ([]versionStatRow, error) {
	var rows []versionStatRow
	err := r.DB.WithContext(ctx).
		Table("audit_logs").
		Select(familyExpr+" AS family, "+versionExpr+" AS version, "+
			"COUNT(DISTINCT audit_logs.user_id) AS unique_users, "+
			"COUNT(*) AS total_actions").
		Where("audit_logs.created_at >= ?", since).
		Group(familyExpr + ", " + versionExpr).
		Scan(&rows).Error
	return rows, err
}

// GetOSGroupedStats returns per-OS-family stats with per-platform-version breakdowns.
// Two queries are used because family-level unique-user counts cannot be derived by
// summing per-version counts (a user may appear in several versions).
func (r *AuditLogRepository) GetOSGroupedStats(ctx context.Context, since time.Time) ([]domain.OSGroupStat, error) {
	families, err := r.familyStats(ctx, since, auditOSFamilyExpr)
	if err != nil {
		return nil, err
	}
	versions, err := r.versionStats(ctx, since, auditOSFamilyExpr, auditPlatformNorm)
	if err != nil {
		return nil, err
	}
	return buildOSGroupedStats(families, versions), nil
}

// GetBrowserGroupedStats returns per-browser-family stats with per-browser-version breakdowns.
func (r *AuditLogRepository) GetBrowserGroupedStats(ctx context.Context, since time.Time) ([]domain.BrowserGroupStat, error) {
	families, err := r.familyStats(ctx, since, auditBrowserFamilyExpr)
	if err != nil {
		return nil, err
	}
	versions, err := r.versionStats(ctx, since, auditBrowserFamilyExpr, auditBrowserNorm)
	if err != nil {
		return nil, err
	}
	return buildBrowserGroupedStats(families, versions), nil
}

// sortVersionStats orders version rows by unique users then total actions (desc).
func sortVersionStats(vs []domain.VersionStat) {
	sort.Slice(vs, func(i, j int) bool {
		if vs[i].UniqueUsers != vs[j].UniqueUsers {
			return vs[i].UniqueUsers > vs[j].UniqueUsers
		}
		return vs[i].TotalActions > vs[j].TotalActions
	})
}

// buildOSGroupedStats nests version rows under their family and sorts the result.
func buildOSGroupedStats(families []familyStatRow, versions []versionStatRow) []domain.OSGroupStat {
	versionsByFamily := make(map[string][]domain.VersionStat)
	for _, v := range versions {
		versionsByFamily[v.Family] = append(versionsByFamily[v.Family], domain.VersionStat{
			Version:      v.Version,
			UniqueUsers:  v.UniqueUsers,
			TotalActions: v.TotalActions,
		})
	}
	for f := range versionsByFamily {
		sortVersionStats(versionsByFamily[f])
	}

	result := make([]domain.OSGroupStat, 0, len(families))
	for _, fam := range families {
		vs := versionsByFamily[fam.Family]
		if vs == nil {
			vs = []domain.VersionStat{}
		}
		result = append(result, domain.OSGroupStat{
			OSFamily:     fam.Family,
			UniqueUsers:  fam.UniqueUsers,
			TotalActions: fam.TotalActions,
			Versions:     vs,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UniqueUsers != result[j].UniqueUsers {
			return result[i].UniqueUsers > result[j].UniqueUsers
		}
		return result[i].TotalActions > result[j].TotalActions
	})
	return result
}

// buildBrowserGroupedStats nests version rows under their family and sorts the result.
func buildBrowserGroupedStats(families []familyStatRow, versions []versionStatRow) []domain.BrowserGroupStat {
	versionsByFamily := make(map[string][]domain.VersionStat)
	for _, v := range versions {
		versionsByFamily[v.Family] = append(versionsByFamily[v.Family], domain.VersionStat{
			Version:      v.Version,
			UniqueUsers:  v.UniqueUsers,
			TotalActions: v.TotalActions,
		})
	}
	for f := range versionsByFamily {
		sortVersionStats(versionsByFamily[f])
	}

	result := make([]domain.BrowserGroupStat, 0, len(families))
	for _, fam := range families {
		vs := versionsByFamily[fam.Family]
		if vs == nil {
			vs = []domain.VersionStat{}
		}
		result = append(result, domain.BrowserGroupStat{
			BrowserFamily: fam.Family,
			UniqueUsers:   fam.UniqueUsers,
			TotalActions:  fam.TotalActions,
			Versions:      vs,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].UniqueUsers != result[j].UniqueUsers {
			return result[i].UniqueUsers > result[j].UniqueUsers
		}
		return result[i].TotalActions > result[j].TotalActions
	})
	return result
}
