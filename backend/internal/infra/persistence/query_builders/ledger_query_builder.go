package query_builders

import (
	"fmt"
	"sort"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// LedgerQueryBuilder handles query construction for ledger entries.
type LedgerQueryBuilder struct {
	db *gorm.DB
}

// NewLedgerQueryBuilder creates a new LedgerQueryBuilder.
func NewLedgerQueryBuilder(db *gorm.DB) *LedgerQueryBuilder {
	return &LedgerQueryBuilder{db: db}
}

// ApplyFilters applies LedgerFilters to a GORM query.
func (b *LedgerQueryBuilder) ApplyFilters(query *gorm.DB, filters domain.LedgerFilters) *gorm.DB {
	if len(filters.Account) > 0 {
		query = query.Where("account IN ?", filters.Account)
	}
	if filters.CreatedBy != nil {
		query = query.Where("created_by = ?", *filters.CreatedBy)
	}
	if filters.Party != nil {
		query = query.Where("party LIKE ?", "%"+*filters.Party+"%")
	}
	if filters.FromDate != nil {
		query = query.Where("date >= ?", *filters.FromDate)
	}
	if filters.ToDate != nil {
		query = query.Where("date <= ?", *filters.ToDate)
	}
	if filters.Search != "" {
		normalizedSearch := utils.NormalizeVietnameseForSearch(filters.Search)
		query = query.Where("search_normalized LIKE ?", normalizedSearch)
	}
	return query
}

// BuildListQuery builds a paginated, sorted list query for ledger entries.
func (b *LedgerQueryBuilder) BuildListQuery(filters domain.LedgerFilters) *gorm.DB {
	query := b.db.
		Preload("Creator").
		Preload("Asset")

	query = b.ApplyFilters(query, filters)

	sortBy := "date"
	if filters.SortBy != "" {
		sortBy = filters.SortBy
	}
	sortOrder := "DESC"
	if filters.SortOrder != "" {
		sortOrder = filters.SortOrder
	}
	query = query.Order(fmt.Sprintf("%s %s", common.SanitizeSortColumn(sortBy, "date"), common.SanitizeSortOrder(sortOrder, "DESC")))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}
	return query
}

// BuildCountQuery builds a count query for ledger entries.
func (b *LedgerQueryBuilder) BuildCountQuery(filters domain.LedgerFilters) *gorm.DB {
	query := b.db.Model(&domain.LedgerEntry{})
	return b.ApplyFilters(query, filters)
}

// SortEntriesByID sorts entries by ID for proper balance calculation sequence.
func (b *LedgerQueryBuilder) SortEntriesByID(entries []*domain.LedgerEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
}

// GetPeriodKey returns a string key for a date given a period granularity.
func (b *LedgerQueryBuilder) GetPeriodKey(date time.Time, period string) string {
	switch period {
	case "day":
		return date.Format("2006-01-02")
	case "week":
		year, week := date.ISOWeek()
		return fmt.Sprintf("%04d-%02d", year, week)
	case "month":
		return date.Format("2006-01")
	case "quarter":
		quarter := (int(date.Month()) + 2) / 3
		return fmt.Sprintf("%d-Q%d", date.Year(), quarter)
	case "year":
		return fmt.Sprintf("%d", date.Year())
	default:
		return date.Format("2006-01-02")
	}
}

// GeneratePeriodKeys generates all unique period keys within a date range.
func (b *LedgerQueryBuilder) GeneratePeriodKeys(startDate, endDate time.Time, period string) []string {
	keysMap := make(map[string]bool)
	current := startDate
	for !current.After(endDate) {
		keysMap[b.GetPeriodKey(current, period)] = true
		current = current.AddDate(0, 0, 1)
	}
	keys := make([]string, 0, len(keysMap))
	for k := range keysMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
