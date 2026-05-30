package persistence

import (
	"context"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

// EmployeeUserRepository implements the domain.EmployeeUserRepository interface
type EmployeeUserRepository struct {
	*BaseRepository
}

// NewEmployeeUserRepository creates a new employee user repository
func NewEmployeeUserRepository(db *Database) domain.EmployeeUserRepository {
	return &EmployeeUserRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// Create creates a new employee user record
func (r *EmployeeUserRepository) Create(ctx context.Context, employeeUser *domain.EmployeeUser) error {
	return r.SafeCreate(ctx, employeeUser)
}

// GetByEmployeeAndUser retrieves an employee user by employee ID and user ID
func (r *EmployeeUserRepository) GetByEmployeeAndUser(ctx context.Context, employeeID, userID uint) (*domain.EmployeeUser, error) {
	var employeeUser domain.EmployeeUser
	err := r.DB.WithContext(ctx).
		Where("employee_id = ? AND user_id = ?", employeeID, userID).
		First(&employeeUser).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return &employeeUser, nil
}

// Delete soft deletes an employee user record
func (r *EmployeeUserRepository) Delete(ctx context.Context, employeeID, userID uint) error {
	return r.dbHelper.ExecuteWithRetry(ctx, func(db *gorm.DB) error {
		result := db.Where("employee_id = ? AND user_id = ?", employeeID, userID).
			Delete(&domain.EmployeeUser{})
		if result.Error != nil {
			return r.dbHelper.WrapDatabaseError(result.Error)
		}
		return nil
	})
}

// GetEmployeeUsers retrieves all users with access to an employee
func (r *EmployeeUserRepository) GetEmployeeUsers(ctx context.Context, employeeID uint) ([]*domain.EmployeeUser, error) {
	var employeeUsers []*domain.EmployeeUser
	err := r.DB.WithContext(ctx).
		Preload("User").
		Preload("GrantedByUser").
		Where("employee_id = ?", employeeID).
		Find(&employeeUsers).Error

	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return employeeUsers, nil
}

// GetUserEmployees retrieves all employee IDs that a user has access to
func (r *EmployeeUserRepository) GetUserEmployees(ctx context.Context, userID uint) ([]uint, error) {
	var employeeIDs []uint
	err := r.DB.WithContext(ctx).
		Model(&domain.EmployeeUser{}).
		Where("user_id = ?", userID).
		Pluck("employee_id", &employeeIDs).Error

	if err != nil {
		return nil, r.dbHelper.WrapDatabaseError(err)
	}

	return employeeIDs, nil
}

// HasAccess checks if a user has access to an employee
func (r *EmployeeUserRepository) HasAccess(ctx context.Context, employeeID, userID uint) (bool, error) {
	var count int64
	err := r.DB.WithContext(ctx).
		Model(&domain.EmployeeUser{}).
		Where("employee_id = ? AND user_id = ?", employeeID, userID).
		Count(&count).Error

	if err != nil {
		return false, r.dbHelper.WrapDatabaseError(err)
	}

	return count > 0, nil
}
