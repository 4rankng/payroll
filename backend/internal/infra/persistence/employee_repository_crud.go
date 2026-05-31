package persistence

import (
	"context"

	"api-server/internal/domain"

	"gorm.io/gorm"
)

func (r *EmployeeRepository) Create(ctx context.Context, employee *domain.Employee) error {
	return r.SafeCreate(ctx, employee)
}

func (r *EmployeeRepository) GetByID(ctx context.Context, id uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByIDQuery(id).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *EmployeeRepository) GetByIDs(ctx context.Context, ids []int64) ([]*domain.Employee, error) {
	if len(ids) == 0 {
		return []*domain.Employee{}, nil
	}

	var employees []*domain.Employee
	const batchSize = 1000 // Safe batch size to stay under MySQL's 32KB query limit

	// Fetch employees in batches to avoid exceeding MySQL max query length
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		var batchEmployees []*domain.Employee
		query := r.queryBuilder.BuildGetByIDsQuery(ctx, batch)

		err := query.Find(&batchEmployees).Error

		if err != nil {
			return nil, err
		}

		employees = append(employees, batchEmployees...)
	}

	return employees, nil
}

func (r *EmployeeRepository) GetByIDForUpdate(ctx context.Context, id uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByIDForUpdateQuery(id).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *EmployeeRepository) GetByUserID(ctx context.Context, userID uint) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByFieldQuery("user_id", userID).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *EmployeeRepository) GetByCCCD(ctx context.Context, cccd string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByFieldWithoutUserQuery("cccd", cccd).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *EmployeeRepository) GetByMobile(ctx context.Context, mobile string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByFieldWithoutUserQuery("mobile", mobile).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

// ExistsByCCCD checks if any non-deleted employee exists with the given CCCD.
// It avoids loading full entities and related associations for performance.
func (r *EmployeeRepository) ExistsByCCCD(ctx context.Context, cccd string) (bool, error) {
	var count int64
	err := r.queryBuilder.BuildExistsQuery("cccd", cccd).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *EmployeeRepository) GetByEmail(ctx context.Context, email string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByFieldQuery("email", email).First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

// ExistsByEmail checks if any non-deleted employee exists with the given email.
// It avoids loading full entities and related associations for performance.
func (r *EmployeeRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.queryBuilder.BuildExistsQuery("email", email).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *EmployeeRepository) GetByBankAccountNumber(ctx context.Context, bankAccountNumber string) (*domain.Employee, error) {
	var employee domain.Employee
	err := r.queryBuilder.BuildGetByFieldWithoutUserQuery("bank_account_number", bankAccountNumber).
		Where("TRIM(bank_account_number) = ?", bankAccountNumber).
		First(&employee).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("employee not found")
		}
		return nil, err
	}

	return &employee, nil
}

func (r *EmployeeRepository) Update(ctx context.Context, employee *domain.Employee) error {
	return r.SafeUpdate(ctx, employee)
}

func (r *EmployeeRepository) Delete(ctx context.Context, id uint) error {
	return r.SafeDelete(ctx, &domain.Employee{}, id)
}

func (r *EmployeeRepository) BulkCreate(ctx context.Context, employees []*domain.Employee) error {
	if len(employees) == 0 {
		return nil
	}

	return r.DB.WithContext(ctx).CreateInBatches(employees, 100).Error
}

// UpdateColumns performs a targeted update of specific columns for an employee.
func (r *EmployeeRepository) UpdateColumns(ctx context.Context, id uint, columns map[string]any) error {
	return r.DB.WithContext(ctx).Model(&domain.Employee{}).Where("id = ?", id).Updates(columns).Error
}
