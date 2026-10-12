package query_builders

import (
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestBuildMissingMobileQueryMatchesOnlyEmptyMobile pins the nudge predicate:
// only employees with a blank mobile match, whitespace/NULL count as blank,
// and soft-deleted rows never surface.
func TestBuildMissingMobileQueryMatchesOnlyEmptyMobile(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE employees (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		fullname TEXT,
		mobile TEXT,
		created_at DATETIME,
		deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`INSERT INTO employees (id, fullname, mobile, created_at, deleted_at) VALUES
		(1, 'Blank', '', '2026-01-01 00:00:00', NULL),
		(2, 'Spaces', '   ', '2026-01-02 00:00:00', NULL),
		(3, 'Null', NULL, '2026-01-03 00:00:00', NULL),
		(4, 'HasMobile', '0912345678', '2026-01-04 00:00:00', NULL),
		(5, 'Deleted', '', '2026-01-05 00:00:00', '2026-01-06 00:00:00')`).Error)

	builder := NewEmployeeProjectQueryBuilder(db)

	var employees []*domain.Employee
	require.NoError(t, builder.BuildMissingMobileQuery(domain.EmployeeFilters{}).Find(&employees).Error)
	ids := make([]uint, 0, len(employees))
	for _, emp := range employees {
		ids = append(ids, emp.ID)
	}
	require.ElementsMatch(t, []uint{1, 2, 3}, ids)

	var count int64
	require.NoError(t, builder.BuildMissingMobileCountQuery(domain.EmployeeFilters{}).Count(&count).Error)
	require.EqualValues(t, 3, count)
}
