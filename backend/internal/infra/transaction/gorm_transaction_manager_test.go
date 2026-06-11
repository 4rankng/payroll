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
