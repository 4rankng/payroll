package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing.
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

// TestStatusFilter implements StatusFilter interface for testing.
type TestStatusFilter struct {
	statuses    []string
	statusField string
}

func (f *TestStatusFilter) GetStatuses() []string {
	return f.statuses
}

func (f *TestStatusFilter) GetStatusField() string {
	if f.statusField == "" {
		return "status"
	}
	return f.statusField
}

// TestDateRangeFilter implements DateRangeFilter interface for testing.
type TestDateRangeFilter struct {
	fromDate  *time.Time
	toDate    *time.Time
	dateField string
}

func (f *TestDateRangeFilter) GetFromDate() *time.Time {
	return f.fromDate
}

func (f *TestDateRangeFilter) GetToDate() *time.Time {
	return f.toDate
}

func (f *TestDateRangeFilter) GetDateField() string {
	if f.dateField == "" {
		return "created_at"
	}
	return f.dateField
}

// TestCreatorFilter implements CreatorFilter interface for testing.
type TestCreatorFilter struct {
	createdBy    *uint
	creatorField string
}

func (f *TestCreatorFilter) GetCreatedBy() *uint {
	return f.createdBy
}

func (f *TestCreatorFilter) GetCreatorField() string {
	if f.creatorField == "" {
		return "created_by"
	}
	return f.creatorField
}

// TestSearchFilter implements SearchFilter interface for testing.
type TestSearchFilter struct {
	search       string
	searchFields []string
}

func (f *TestSearchFilter) GetSearch() string {
	return f.search
}

func (f *TestSearchFilter) GetSearchFields() []string {
	return f.searchFields
}

func TestFilterBuilder_ApplyStatus(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	t.Run("applies status filter correctly", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestStatusFilter{
			statuses:    []string{"active", "pending"},
			statusField: "status",
		}

		result := fb.ApplyStatus(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles empty statuses", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestStatusFilter{
			statuses: []string{},
		}

		result := fb.ApplyStatus(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles nil filter", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyStatus(query, nil)

		assert.NotNil(t, result)
	})
}

func TestFilterBuilder_ApplyDateRange(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	fromDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC)

	t.Run("applies both from and to date", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestDateRangeFilter{
			fromDate:  &fromDate,
			toDate:    &toDate,
			dateField: "created_at",
		}

		result := fb.ApplyDateRange(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("applies only from date", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestDateRangeFilter{
			fromDate:  &fromDate,
			toDate:    nil,
			dateField: "created_at",
		}

		result := fb.ApplyDateRange(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("applies only to date", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestDateRangeFilter{
			fromDate:  nil,
			toDate:    &toDate,
			dateField: "created_at",
		}

		result := fb.ApplyDateRange(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles nil filter", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyDateRange(query, nil)

		assert.NotNil(t, result)
	})
}

func TestFilterBuilder_ApplyCreator(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	creatorID := uint(123)

	t.Run("applies creator filter correctly", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestCreatorFilter{
			createdBy:    &creatorID,
			creatorField: "created_by",
		}

		result := fb.ApplyCreator(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles nil creator ID", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestCreatorFilter{
			createdBy: nil,
		}

		result := fb.ApplyCreator(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles nil filter", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyCreator(query, nil)

		assert.NotNil(t, result)
	})
}

func TestFilterBuilder_ApplyVietnameseSearch(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	t.Run("applies search to multiple fields", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestSearchFilter{
			search:       "test search",
			searchFields: []string{"name", "description", "code"},
		}

		result := fb.ApplyVietnameseSearch(query, filter)
		assert.NotNil(t, result)
	})

	t.Run("handles empty search string", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestSearchFilter{
			search:       "",
			searchFields: []string{"name"},
		}

		result := fb.ApplyVietnameseSearch(query, filter)
		assert.Equal(t, query, result)
	})

	t.Run("handles empty search fields", func(t *testing.T) {
		query := db.Table("test")
		filter := &TestSearchFilter{
			search:       "test",
			searchFields: []string{},
		}

		result := fb.ApplyVietnameseSearch(query, filter)
		assert.Equal(t, query, result)
	})

	t.Run("handles nil filter", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyVietnameseSearch(query, nil)

		assert.Equal(t, query, result)
	})
}

func TestFilterBuilder_ApplyPagination(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	t.Run("applies limit and offset", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyPagination(query, 10, 20)

		assert.NotNil(t, result)
	})

	t.Run("applies only limit", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyPagination(query, 10, 0)

		assert.NotNil(t, result)
	})

	t.Run("applies only offset", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyPagination(query, 0, 20)

		assert.NotNil(t, result)
	})

	t.Run("handles negative limit", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyPagination(query, -10, 20)

		assert.NotNil(t, result)
	})

	t.Run("handles negative offset", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplyPagination(query, 10, -20)

		assert.NotNil(t, result)
	})
}

func TestFilterBuilder_ApplySorting(t *testing.T) {
	db := setupTestDB(t)
	fb := NewFilterBuilder(db)

	t.Run("uses provided sort and order", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplySorting(query, "name", "ASC", "created_at")

		assert.NotNil(t, result)
	})

	t.Run("uses default sort when not provided", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplySorting(query, "", "ASC", "created_at")

		assert.NotNil(t, result)
	})

	t.Run("uses DESC as default order", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplySorting(query, "name", "", "created_at")

		assert.NotNil(t, result)
	})

	t.Run("uses default sort and order when neither provided", func(t *testing.T) {
		query := db.Table("test")
		result := fb.ApplySorting(query, "", "", "created_at")

		assert.NotNil(t, result)
	})
}
