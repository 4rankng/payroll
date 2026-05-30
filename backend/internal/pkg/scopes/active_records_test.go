package scopes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestActiveModel implements ActiveScoped for testing
type TestActiveModel struct {
	ID     uint   `gorm:"primaryKey"`
	Name   string `gorm:"size:100"`
	Status string `gorm:"size:20"`
}

func (m *TestActiveModel) ActiveColumn() (string, interface{}) {
	return "status", "active"
}

// TestInactiveModel implements ActiveScoped with exclusion filter
type TestInactiveModel struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:100"`
	IsActive string `gorm:"size:20"`
}

func (m *TestInactiveModel) ActiveColumn() (string, interface{}) {
	return "is_active", "!deleted"
}

// TestNonActiveModel does not implement ActiveScoped
type TestNonActiveModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	// Register the plugin
	err = db.Use(&ActiveRecordsPlugin{})
	assert.NoError(t, err)

	// Migrate test models
	err = db.AutoMigrate(&TestActiveModel{}, &TestInactiveModel{}, &TestNonActiveModel{})
	assert.NoError(t, err)

	return db
}

func TestActiveRecordsPlugin_Name(t *testing.T) {
	plugin := ActiveRecordsPlugin{}
	assert.Equal(t, "active_records_plugin", plugin.Name())
}

func TestActiveRecordsPlugin_Initialize(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	plugin := &ActiveRecordsPlugin{}
	err = plugin.Initialize(db)
	assert.NoError(t, err)
}

func TestActiveRecordsPlugin_EqualityFilter(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})
	db.Create(&TestActiveModel{Name: "Active2", Status: "active"})
	db.Create(&TestActiveModel{Name: "Deleted", Status: "deleted"})

	// Query without any explicit filter - should only return active records
	var results []TestActiveModel
	db.Find(&results)

	assert.Equal(t, 2, len(results))
	for _, result := range results {
		assert.Equal(t, "active", result.Status)
	}
}

func TestActiveRecordsPlugin_ExclusionFilter(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestInactiveModel{Name: "Record1", IsActive: "active"})
	db.Create(&TestInactiveModel{Name: "Record2", IsActive: "inactive"})
	db.Create(&TestInactiveModel{Name: "Deleted", IsActive: "deleted"})

	// Query without any explicit filter - should exclude deleted records
	var results []TestInactiveModel
	db.Find(&results)

	assert.Equal(t, 2, len(results))
	for _, result := range results {
		assert.NotEqual(t, "deleted", result.IsActive)
	}
}

func TestActiveRecordsPlugin_NonActiveScopedModel(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestNonActiveModel{Name: "Record1"})
	db.Create(&TestNonActiveModel{Name: "Record2"})

	// Query should return all records (no filter applied)
	var results []TestNonActiveModel
	db.Find(&results)

	assert.Equal(t, 2, len(results))
}

func TestActiveRecordsPlugin_SkipActiveFilter(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})
	db.Create(&TestActiveModel{Name: "Deleted", Status: "deleted"})

	// Query with skip_active_filter - should return all records
	var results []TestActiveModel
	SkipActiveFilter(db).Find(&results)

	assert.Equal(t, 3, len(results))
}

func TestActiveRecordsPlugin_ExplicitFilter(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})
	db.Create(&TestActiveModel{Name: "Deleted", Status: "deleted"})

	// Query with name filter - active filter still applies
	var results []TestActiveModel
	db.Where("name = ?", "Active1").Find(&results)

	assert.Equal(t, 1, len(results))
	if len(results) > 0 {
		assert.Equal(t, "active", results[0].Status)
		assert.Equal(t, "Active1", results[0].Name)
	}
}

func TestActiveRecordsPlugin_Unscoped(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})
	db.Create(&TestActiveModel{Name: "Deleted", Status: "deleted"})

	// Query with Unscoped - should return all records
	var results []TestActiveModel
	db.Unscoped().Find(&results)

	assert.Equal(t, 3, len(results))
}

func TestActiveRecordsPlugin_First(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})

	// Query with First - should only return active record
	var result TestActiveModel
	err := db.First(&result).Error

	assert.NoError(t, err)
	assert.Equal(t, "active", result.Status)
}

func TestActiveRecordsPlugin_Count(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Active2", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})

	// Count should only count active records
	var count int64
	db.Model(&TestActiveModel{}).Count(&count)

	assert.Equal(t, int64(2), count)
}

func TestActiveRecordsPlugin_Where(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Active2", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})

	// Query with additional where clause
	var results []TestActiveModel
	db.Where("name LIKE ?", "Active%").Find(&results)

	assert.Equal(t, 2, len(results))
	for _, result := range results {
		assert.Equal(t, "active", result.Status)
		assert.Contains(t, result.Name, "Active")
	}
}

func TestSkipActiveFilter(t *testing.T) {
	db := setupTestDB(t)

	// Insert test data
	db.Create(&TestActiveModel{Name: "Active1", Status: "active"})
	db.Create(&TestActiveModel{Name: "Inactive1", Status: "inactive"})

	// Test SkipActiveFilter function
	query := SkipActiveFilter(db)
	assert.NotNil(t, query)

	// Should return all records
	var results []TestActiveModel
	query.Find(&results)

	assert.Equal(t, 2, len(results))
}
