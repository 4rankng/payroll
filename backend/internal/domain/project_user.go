package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ProjectUser struct {
	ID        uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	ProjectID uint           `json:"project_id" gorm:"not null;type:bigint unsigned"`
	UserID    uint           `json:"user_id" gorm:"not null;type:bigint unsigned"`
	GrantedBy uint           `json:"granted_by" gorm:"not null;type:bigint unsigned"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Relationships
	Project       Project `json:"project" gorm:"foreignKey:ProjectID;references:ID"`
	User          User    `json:"user" gorm:"foreignKey:UserID;references:ID"`
	GrantedByUser User    `json:"granted_by_user" gorm:"foreignKey:GrantedBy;references:ID"`
}

func (ProjectUser) TableName() string {
	return "project_users"
}

type ProjectUserRepository interface {
	Create(ctx context.Context, projectUser *ProjectUser) error
	GetByProjectAndUser(ctx context.Context, projectID, userID uint) (*ProjectUser, error)
	Delete(ctx context.Context, projectID, userID uint) error
	GetProjectUsers(ctx context.Context, projectID uint) ([]*ProjectUser, error)
	GetUserProjects(ctx context.Context, userID uint) ([]uint, error)
	HasAccess(ctx context.Context, projectID, userID uint) (bool, error)
}
