package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type EmployeeUser struct {
	ID         uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	EmployeeID uint           `json:"employee_id" gorm:"not null;type:bigint unsigned"`
	UserID     uint           `json:"user_id" gorm:"not null;type:bigint unsigned"`
	GrantedBy  uint           `json:"granted_by" gorm:"not null;type:bigint unsigned"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`

	// Relationships
	Employee      Employee `json:"employee" gorm:"foreignKey:EmployeeID;references:ID"`
	User          User     `json:"user" gorm:"foreignKey:UserID;references:ID"`
	GrantedByUser User     `json:"granted_by_user" gorm:"foreignKey:GrantedBy;references:ID"`
}

func (EmployeeUser) TableName() string {
	return "employee_users"
}

type EmployeeUserRepository interface {
	Create(ctx context.Context, employeeUser *EmployeeUser) error
	GetByEmployeeAndUser(ctx context.Context, employeeID, userID uint) (*EmployeeUser, error)
	Delete(ctx context.Context, employeeID, userID uint) error
	GetEmployeeUsers(ctx context.Context, employeeID uint) ([]*EmployeeUser, error)
	GetUserEmployees(ctx context.Context, userID uint) ([]uint, error)
	HasAccess(ctx context.Context, employeeID, userID uint) (bool, error)
}
