package persistence

import (
	"testing"

	"api-server/internal/domain"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositoryCreateUsesTransactionContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL,
			email TEXT,
			password TEXT NOT NULL,
			fullname TEXT NOT NULL,
			cccd TEXT,
			mobile TEXT,
			role TEXT NOT NULL,
			deleted_at DATETIME,
			last_login DATETIME,
			tokens_invalid_before DATETIME,
			otp_failed_attempts INTEGER NOT NULL DEFAULT 0,
			otp_locked_until DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create users table: %v", err)
	}

	repo := NewUserRepository(&Database{DB: db})
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatalf("begin transaction: %v", tx.Error)
	}

	user := &domain.User{
		Username: "transactional-user",
		Password: "hashed-password",
		Fullname: "Transactional User",
		Role:     domain.RoleEmployee,
	}
	if err := repo.Create(transactionContextForTest(tx), user); err != nil {
		_ = tx.Rollback()
		t.Fatalf("create user: %v", err)
	}
	if err := tx.Rollback().Error; err != nil {
		t.Fatalf("rollback transaction: %v", err)
	}

	var count int64
	if err := db.Model(&domain.User{}).Where("username = ?", user.Username).Count(&count).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 0 {
		t.Fatalf("users after rollback = %d, want 0", count)
	}
}
