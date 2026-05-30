package persistence

import (
	"context"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// ProjectUserRepository implements the domain.ProjectUserRepository interface
type ProjectUserRepository struct {
	*BaseRepository
}

// NewProjectUserRepository creates a new project user repository
func NewProjectUserRepository(db *Database) domain.ProjectUserRepository {
	return &ProjectUserRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new project user record
func (r *ProjectUserRepository) Create(ctx context.Context, projectUser *domain.ProjectUser) error {
	return r.SafeCreate(ctx, projectUser)
}

// GetByProjectAndUser retrieves a project user by project ID and user ID
func (r *ProjectUserRepository) GetByProjectAndUser(ctx context.Context, projectID, userID uint) (*domain.ProjectUser, error) {
	var projectUser domain.ProjectUser
	err := r.DB.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&projectUser).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return &projectUser, nil
}

// Delete soft deletes a project user record
func (r *ProjectUserRepository) Delete(ctx context.Context, projectID, userID uint) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		result := db.Where("project_id = ? AND user_id = ?", projectID, userID).
			Delete(&domain.ProjectUser{})
		if result.Error != nil {
			return r.dbHelper.WrapDatabaseError(result.Error)
		}
		return nil
	})
}

// GetProjectUsers retrieves all users with access to a project
func (r *ProjectUserRepository) GetProjectUsers(ctx context.Context, projectID uint) ([]*domain.ProjectUser, error) {
	var projectUsers []*domain.ProjectUser
	err := r.DB.WithContext(ctx).
		Preload("User").
		Preload("GrantedByUser").
		Where("project_id = ?", projectID).
		Find(&projectUsers).Error

	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return projectUsers, nil
}

// GetUserProjects retrieves all project IDs that a user has access to
func (r *ProjectUserRepository) GetUserProjects(ctx context.Context, userID uint) ([]uint, error) {
	var projectIDs []uint
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectUser{}).
		Where("user_id = ?", userID).
		Pluck("project_id", &projectIDs).Error

	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return projectIDs, nil
}

// HasAccess checks if a user has access to a project
func (r *ProjectUserRepository) HasAccess(ctx context.Context, projectID, userID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.ProjectUser{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error

	if err != nil {
		return false, r.dbHelper.WrapDatabaseError(err)
	}

	return count > 0, nil
}
