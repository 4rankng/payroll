package persistence

import (
	"context"
	"encoding/json"
	"testing"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openBulkTransferTransactionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := "file:" + uuid.NewString() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE bulk_transfer_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT NOT NULL,
			cycle TEXT,
			created_by INTEGER NOT NULL,
			from_date DATETIME,
			to_date DATETIME,
			for_month TEXT,
			transactions_count INTEGER NOT NULL DEFAULT 0,
			completed_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			transfer_amount INTEGER NOT NULL DEFAULT 0,
			data TEXT NOT NULL,
			asset_id INTEGER,
			created_at DATETIME,
			updated_at DATETIME,
			source TEXT NOT NULL DEFAULT 'manual',
			status TEXT NOT NULL DEFAULT 'exported',
			uploaded_at DATETIME
		);
		CREATE TABLE transaction_codes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT NOT NULL UNIQUE,
			data TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);
	`).Error; err != nil {
		t.Fatalf("create test tables: %v", err)
	}

	return db
}

func transactionContextForTest(tx *gorm.DB) context.Context {
	return domain.WithTransactionContext(context.Background(), &domain.TransactionContext{
		TX:              tx,
		IsTransactional: true,
	})
}

func TestBulkTransferFileRepositoryCreateUsesTransactionContext(t *testing.T) {
	db := openBulkTransferTransactionTestDB(t)
	repo := NewBulkTransferFileRepository(db, nil)

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	ctx := transactionContextForTest(tx)

	file := &domain.BulkTransferFile{
		Filename:          "result.xlsx",
		CreatedBy:         1,
		TransactionsCount: 1,
		CompletedCount:    1,
		TransferAmount:    100_000,
		Data:              `[{"transaction_code":"VFIC-test"}]`,
		Status:            "uploaded",
	}
	if err := repo.Create(ctx, file); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create bulk transfer metadata in transaction: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var count int64
	if err := db.Model(&domain.BulkTransferFile{}).
		Where("filename = ?", file.Filename).
		Count(&count).Error; err != nil {
		t.Fatalf("count bulk transfer metadata: %v", err)
	}
	if count != 0 {
		t.Fatalf("metadata count after rollback = %d, want 0", count)
	}
}

func TestTransactionCodeRepositoryUpdateUsesTransactionContext(t *testing.T) {
	db := openBulkTransferTransactionTestDB(t)
	repo := NewTransactionCodeRepository(&Database{DB: db})

	codeData, err := json.Marshal(domain.TransactionCodeData{
		WeeklyPay: &domain.CyclePayData{
			TimesheetIDs: []uint{101},
			EmployeeID:   202,
			Amount:       100_000,
		},
	})
	if err != nil {
		t.Fatalf("marshal transaction code data: %v", err)
	}
	code := &domain.TransactionCode{Code: "VFIC-test", Data: codeData}
	if err := db.Create(code).Error; err != nil {
		t.Fatalf("seed transaction code: %v", err)
	}

	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}
	ctx := transactionContextForTest(tx)

	if err := repo.UpdateFileIDByCodes(ctx, []string{code.Code}, 999); err != nil {
		_ = tx.Rollback()
		t.Fatalf("update transaction code in transaction: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var persisted domain.TransactionCode
	if err := db.Where("code = ?", code.Code).First(&persisted).Error; err != nil {
		t.Fatalf("reload transaction code: %v", err)
	}
	var persistedData domain.TransactionCodeData
	if err := json.Unmarshal(persisted.Data, &persistedData); err != nil {
		t.Fatalf("unmarshal persisted transaction code data: %v", err)
	}
	if persistedData.WeeklyPay == nil {
		t.Fatal("weekly pay data missing after rollback")
	}
	if persistedData.WeeklyPay.FileID != nil {
		t.Fatalf("file_id after rollback = %d, want nil", *persistedData.WeeklyPay.FileID)
	}
}
