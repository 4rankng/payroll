package transaction

import (
	"context"
	"errors"
	"testing"

	"api-server/internal/domain"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Test model for transaction testing
type TestModel struct {
	ID   uint   `gorm:"primaryKey"`
	Name string `gorm:"size:100"`
}

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(&TestModel{})
	assert.NoError(t, err)

	return db
}

func TestNewGormTransactionManager(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	assert.NotNil(t, tm)
	assert.Implements(t, (*domain.TransactionManager)(nil), tm)
}

func TestGormTransactionManager_WithTransaction_Success(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	var execCount int
	err := tm.WithTransaction(context.Background(), func(ctx context.Context) error {
		execCount++

		// Verify transaction context is set
		txCtx, ok := domain.GetTransactionFromContext(ctx)
		assert.True(t, ok)
		assert.NotNil(t, txCtx)
		assert.True(t, txCtx.IsTransactional)
		assert.NotNil(t, txCtx.TX)

		// Create test data
		model := &TestModel{Name: "test"}
		return txCtx.TX.Create(model).Error
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, execCount)

	// Verify data was committed
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestGormTransactionManager_WithTransaction_Rollback(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	testErr := errors.New("test error")
	err := tm.WithTransaction(context.Background(), func(ctx context.Context) error {
		txCtx, ok := domain.GetTransactionFromContext(ctx)
		assert.True(t, ok)

		// Create test data
		model := &TestModel{Name: "test"}
		if err := txCtx.TX.Create(model).Error; err != nil {
			return err
		}

		// Force rollback
		return testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)

	// Verify data was rolled back
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestGormTransactionManager_WithTransactionResult_Success(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	result, err := tm.WithTransactionResult(context.Background(), func(ctx context.Context) (interface{}, error) {
		txCtx, ok := domain.GetTransactionFromContext(ctx)
		assert.True(t, ok)

		// Create test data
		model := &TestModel{Name: "test-result"}
		if err := txCtx.TX.Create(model).Error; err != nil {
			return nil, err
		}

		return "success", nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "success", result)

	// Verify data was committed
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestGormTransactionManager_WithTransactionResult_Rollback(t *testing.T) {
	db := setupTestDB(t)
	tm := NewGormTransactionManager(db)

	testErr := errors.New("test error")
	result, err := tm.WithTransactionResult(context.Background(), func(ctx context.Context) (interface{}, error) {
		txCtx, ok := domain.GetTransactionFromContext(ctx)
		assert.True(t, ok)

		// Create test data
		model := &TestModel{Name: "test"}
		if err := txCtx.TX.Create(model).Error; err != nil {
			return nil, err
		}

		return nil, testErr
	})

	assert.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Nil(t, result)

	// Verify data was rolled back
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestNewGormUnitOfWork(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	assert.NotNil(t, uow)
	assert.Implements(t, (*domain.UnitOfWork)(nil), uow)
}

func TestGormUnitOfWork_RegisterNew(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	model := &TestModel{Name: "new"}
	err := uow.RegisterNew(context.Background(), model)

	assert.NoError(t, err)
}

func TestGormUnitOfWork_RegisterDirty(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	model := &TestModel{ID: 1, Name: "dirty"}
	err := uow.RegisterDirty(context.Background(), model)

	assert.NoError(t, err)
}

func TestGormUnitOfWork_RegisterDeleted(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	model := &TestModel{ID: 1, Name: "deleted"}
	err := uow.RegisterDeleted(context.Background(), model)

	assert.NoError(t, err)
}

func TestGormUnitOfWork_Commit_NewEntities(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	model1 := &TestModel{Name: "new1"}
	model2 := &TestModel{Name: "new2"}

	err := uow.RegisterNew(context.Background(), model1)
	assert.NoError(t, err)

	err = uow.RegisterNew(context.Background(), model2)
	assert.NoError(t, err)

	err = uow.Commit(context.Background())
	assert.NoError(t, err)

	// Verify entities were created
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(2), count)
}

func TestGormUnitOfWork_Commit_DirtyEntities(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	// Create initial entity
	model := &TestModel{Name: "original"}
	db.Create(model)

	// Update entity
	model.Name = "updated"
	err := uow.RegisterDirty(context.Background(), model)
	assert.NoError(t, err)

	err = uow.Commit(context.Background())
	assert.NoError(t, err)

	// Verify entity was updated
	var result TestModel
	db.First(&result, model.ID)
	assert.Equal(t, "updated", result.Name)
}

func TestGormUnitOfWork_Commit_DeletedEntities(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	// Create initial entity
	model := &TestModel{Name: "to-delete"}
	db.Create(model)

	// Delete entity
	err := uow.RegisterDeleted(context.Background(), model)
	assert.NoError(t, err)

	err = uow.Commit(context.Background())
	assert.NoError(t, err)

	// Verify entity was deleted
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestGormUnitOfWork_Commit_MixedOperations(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	// Create initial entities
	existing1 := &TestModel{Name: "existing1"}
	existing2 := &TestModel{Name: "existing2"}
	db.Create(existing1)
	db.Create(existing2)

	// Register mixed operations
	newModel := &TestModel{Name: "new"}
	err := uow.RegisterNew(context.Background(), newModel)
	assert.NoError(t, err)

	existing1.Name = "updated"
	err = uow.RegisterDirty(context.Background(), existing1)
	assert.NoError(t, err)

	err = uow.RegisterDeleted(context.Background(), existing2)
	assert.NoError(t, err)

	err = uow.Commit(context.Background())
	assert.NoError(t, err)

	// Verify results
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(2), count) // 1 existing updated + 1 new

	var result TestModel
	db.First(&result, existing1.ID)
	assert.Equal(t, "updated", result.Name)
}

func TestGormUnitOfWork_Rollback(t *testing.T) {
	db := setupTestDB(t)
	uow := NewGormUnitOfWork(db)

	// Register some operations
	model1 := &TestModel{Name: "new1"}
	model2 := &TestModel{Name: "new2"}

	err := uow.RegisterNew(context.Background(), model1)
	assert.NoError(t, err)

	err = uow.RegisterNew(context.Background(), model2)
	assert.NoError(t, err)

	// Rollback
	err = uow.Rollback(context.Background())
	assert.NoError(t, err)

	// Try to commit after rollback - should do nothing
	err = uow.Commit(context.Background())
	assert.NoError(t, err)

	// Verify no entities were created
	var count int64
	db.Model(&TestModel{}).Count(&count)
	assert.Equal(t, int64(0), count)
}
