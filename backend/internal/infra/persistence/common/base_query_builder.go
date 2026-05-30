package common

import (
	"gorm.io/gorm"
)

// BaseQueryBuilder provides a template for query builders with common operations.
// It eliminates duplication of sorting, pagination, and preload logic across different query builders.
// Embed this struct in your domain-specific query builders to inherit common functionality.
type BaseQueryBuilder struct {
	DB               *gorm.DB // Exposed for use in derived builders
	filterBuilder    *FilterBuilder
	defaultSortBy    string
	defaultSortOrder string
	defaultPreloads  []string
}

// NewBaseQueryBuilder creates a new BaseQueryBuilder instance.
// Parameters:
//   - db: GORM database instance
//   - defaultSortBy: Default field to sort by (e.g., "created_at")
//   - defaultSortOrder: Default sort order ("ASC" or "DESC")
//   - defaultPreloads: Default relationships to preload
func NewBaseQueryBuilder(db *gorm.DB, defaultSortBy, defaultSortOrder string, defaultPreloads []string) *BaseQueryBuilder {
	return &BaseQueryBuilder{
		DB:               db,
		filterBuilder:    NewFilterBuilder(db),
		defaultSortBy:    defaultSortBy,
		defaultSortOrder: defaultSortOrder,
		defaultPreloads:  defaultPreloads,
	}
}

// GetFilterBuilder returns the FilterBuilder instance.
// Use this to apply common filters in your domain-specific query builders.
func (bqb *BaseQueryBuilder) GetFilterBuilder() *FilterBuilder {
	return bqb.filterBuilder
}

// BuildBaseQuery creates a base query for the given model with default preloads.
// This is typically the starting point for building more complex queries.
func (bqb *BaseQueryBuilder) BuildBaseQuery(model interface{}) *gorm.DB {
	query := bqb.DB.Model(model)
	return bqb.ApplyDefaultPreloads(query)
}

// ApplyDefaultPreloads applies the default preloads configured for this builder.
func (bqb *BaseQueryBuilder) ApplyDefaultPreloads(query *gorm.DB) *gorm.DB {
	for _, preload := range bqb.defaultPreloads {
		query = query.Preload(preload)
	}
	return query
}

// ApplyPreloads applies custom preloads to the query.
// Use this when you need different preloads than the defaults.
func (bqb *BaseQueryBuilder) ApplyPreloads(query *gorm.DB, preloads []string) *gorm.DB {
	for _, preload := range preloads {
		query = query.Preload(preload)
	}
	return query
}

// ApplyListOptions applies sorting and pagination to the query.
// This is the standard way to apply list options across all repositories.
// If sortBy is empty, the default sort field is used.
// If sortOrder is empty, "DESC" is used.
// If limit is 0 or negative, no limit is applied.
// If offset is 0 or negative, no offset is applied.
func (bqb *BaseQueryBuilder) ApplyListOptions(query *gorm.DB, sortBy, sortOrder string, limit, offset int) *gorm.DB {
	query = bqb.filterBuilder.ApplySorting(query, sortBy, sortOrder, bqb.defaultSortBy)
	query = bqb.filterBuilder.ApplyPagination(query, limit, offset)
	return query
}

// ApplySorting applies only sorting to the query.
// Use this when you need to sort but not paginate.
func (bqb *BaseQueryBuilder) ApplySorting(query *gorm.DB, sortBy, sortOrder string) *gorm.DB {
	return bqb.filterBuilder.ApplySorting(query, sortBy, sortOrder, bqb.defaultSortBy)
}

// ApplyPagination applies only pagination to the query.
// Use this when you need to paginate but not sort.
func (bqb *BaseQueryBuilder) ApplyPagination(query *gorm.DB, limit, offset int) *gorm.DB {
	return bqb.filterBuilder.ApplyPagination(query, limit, offset)
}
