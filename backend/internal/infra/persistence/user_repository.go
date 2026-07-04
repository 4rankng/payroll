package persistence

import (
	"api-server/internal/pkg/clock"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence/common"
	"api-server/internal/pkg/timeutil"

	"gorm.io/gorm"
)

type UserRepository struct {
	*BaseRepository
}

func NewUserRepository(db *Database) domain.UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	if err := user.IsValid(); err != nil {
		return err
	}

	log := observability.GetLogger()
	log.Info("Creating user in database", "username", user.Username, "email", user.Email)

	if err := r.DB.WithContext(ctx).Create(user).Error; err != nil {
		log.Error("Database error when creating user", "error", err, "username", user.Username, "email", user.Email)

		// Check for MySQL duplicate entry error (Error 1062)
		errMsg := err.Error()
		if strings.Contains(errMsg, "Duplicate entry") {
			if strings.Contains(errMsg, "cccd") || strings.Contains(errMsg, "unique_user_cccd") {
				return domain.NewConflictError("Số CCCD này đã được sử dụng bởi tài khoản khác")
			}
			if strings.Contains(errMsg, "mobile") || strings.Contains(errMsg, "unique_user_mobile") {
				return domain.NewConflictError("Số điện thoại này đã được sử dụng bởi tài khoản khác")
			}
			if strings.Contains(errMsg, "username") || strings.Contains(errMsg, "users.username") {
				return domain.NewConflictError(fmt.Sprintf("username '%s' already exists", user.Username))
			}
			if strings.Contains(errMsg, "email") || strings.Contains(errMsg, "users.email") {
				var emailStr string
				if user.Email != nil {
					emailStr = *user.Email
				}
				return domain.NewConflictError(fmt.Sprintf("email '%s' already exists", emailStr))
			}
			return domain.NewConflictError("user already exists")
		}
		// Return the actual error message for better debugging
		return domain.NewInternalError(fmt.Sprintf("failed to create user: %v", err), err)
	}

	log.Info("User created successfully", "id", user.ID, "username", user.Username)
	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uint) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by ID", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByIDs(ctx context.Context, ids []uint) (map[uint]*domain.User, error) {
	if len(ids) == 0 {
		return make(map[uint]*domain.User), nil
	}

	var users []*domain.User
	if err := r.DB.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, domain.NewInternalError("failed to get users by IDs", err)
	}

	// Convert to map for O(1) lookup
	userMap := make(map[uint]*domain.User, len(users))
	for _, user := range users {
		userMap[user.ID] = user
	}

	return userMap, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by email", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by username", err)
	}

	return &user, nil
}

func (r *UserRepository) GetByUsernameIncludingDeleted(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Unscoped().Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by username", err)
	}

	return &user, nil
}

// GetUsernamesByPrefix returns all usernames (including soft-deleted) that start with the given prefix.
// Used to find existing username variants in a single query instead of N sequential lookups.
func (r *UserRepository) GetUsernamesByPrefix(ctx context.Context, prefix string) ([]string, error) {
	var usernames []string
	err := r.DB.WithContext(ctx).
		Unscoped().
		Model(&domain.User{}).
		Where("username LIKE ?", prefix+"%").
		Pluck("username", &usernames).Error
	if err != nil {
		return nil, domain.NewInternalError("failed to get usernames by prefix", err)
	}
	return usernames, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	if err := user.IsValid(); err != nil {
		return err
	}

	if err := r.DB.WithContext(ctx).Save(user).Error; err != nil {
		errMsg := err.Error()
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(errMsg, "Duplicate entry") {
			if strings.Contains(errMsg, "cccd") || strings.Contains(errMsg, "unique_user_cccd") {
				return domain.NewConflictError("Số CCCD này đã được sử dụng bởi tài khoản khác")
			}
			if strings.Contains(errMsg, "mobile") || strings.Contains(errMsg, "unique_user_mobile") {
				return domain.NewConflictError("Số điện thoại này đã được sử dụng bởi tài khoản khác")
			}
			if strings.Contains(errMsg, "email") {
				return domain.NewConflictError("Email này đã được sử dụng bởi tài khoản khác")
			}
			if strings.Contains(errMsg, "username") {
				return domain.NewConflictError("Tên đăng nhập này đã tồn tại")
			}
			return domain.NewConflictError("Thông tin đã tồn tại trong hệ thống")
		}
		return domain.NewInternalError("failed to update user", err)
	}

	return nil
}

func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uint, lastLogin time.Time) error {
	if err := r.DB.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("last_login", lastLogin).Error; err != nil {
		return domain.NewInternalError("failed to update last login", err)
	}
	return nil
}

// UpdateTokensInvalidBefore records the instant at which all previously-issued
// JWTs for this user become invalid. AuthService.ValidateToken rejects any token
// whose IssuedAt is older than this timestamp.
func (r *UserRepository) UpdateTokensInvalidBefore(ctx context.Context, userID uint, invalidBefore time.Time) error {
	if err := r.DB.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", userID).
		Update("tokens_invalid_before", invalidBefore).Error; err != nil {
		return domain.NewInternalError("failed to update tokens_invalid_before", err)
	}
	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	result := r.DB.WithContext(ctx).Delete(&domain.User{}, id)
	if result.Error != nil {
		return domain.NewInternalError("failed to delete user", result.Error)
	}

	if result.RowsAffected == 0 {
		return domain.NewNotFoundError("user not found")
	}

	return nil
}

func (r *UserRepository) Restore(ctx context.Context, id uint) error {
	return r.DB.WithContext(ctx).Unscoped().Model(&domain.User{}).Where("id = ?", id).Update("deleted_at", nil).Error
}

func (r *UserRepository) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	var users []*domain.User
	if err := r.DB.WithContext(ctx).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, domain.NewInternalError("failed to list users", err)
	}

	return users, nil
}

func (r *UserRepository) ListByRole(ctx context.Context, role domain.UserRole) ([]*domain.User, error) {
	var users []*domain.User
	if err := r.DB.WithContext(ctx).Where("role = ?", role).Find(&users).Error; err != nil {
		return nil, domain.NewInternalError("failed to list users by role", err)
	}

	return users, nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&domain.User{}).Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count users", err)
	}

	return count, nil
}

func (r *UserRepository) CountByRole(ctx context.Context, role domain.UserRole) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&domain.User{}).Where("role = ?", role).Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count users by role", err)
	}

	return count, nil
}

func (r *UserRepository) CountRecentLogins(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&domain.User{}).Where("last_login >= ?", since).Count(&count).Error; err != nil {
		return 0, domain.NewInternalError("failed to count recent logins", err)
	}

	return count, nil
}

func (r *UserRepository) CountActiveEmployeesBySchedule(ctx context.Context, since, until time.Time) (weekly, monthly, flexible int, err error) {
	type scheduleCount struct {
		Schedule string
		Count    int
	}

	var results []scheduleCount
	now := clock.Now()

	if err := r.DB.WithContext(ctx).
		Model(&domain.User{}).
		Select("pe.payment_schedule as schedule, COUNT(DISTINCT e.id) as count").
		Joins("JOIN employees e ON e.user_id = users.id AND e.deleted_at IS NULL").
		Joins("JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL AND pe.start_date <= ? AND (pe.last_date IS NULL OR pe.last_date >= ?)", now, now).
		Where("users.role = ?", "employee").
		Where("users.last_login >= ? AND users.last_login < ?", since, until).
		Group("pe.payment_schedule").
		Scan(&results).Error; err != nil {
		return 0, 0, 0, domain.NewInternalError("failed to count active employees by schedule", err)
	}

	for _, res := range results {
		switch res.Schedule {
		case "weekly":
			weekly = res.Count
		case "monthly":
			monthly = res.Count
		case "flexible":
			flexible = res.Count
		}
	}

	return weekly, monthly, flexible, nil
}

func (r *UserRepository) GetActiveEmployeesBySchedule(ctx context.Context, since, until time.Time, schedule string) ([]*domain.ActiveEmployeeUser, error) {
	type row struct {
		UserID     uint       `gorm:"column:user_id"`
		EmployeeID uint       `gorm:"column:employee_id"`
		Fullname   string     `gorm:"column:fullname"`
		Username   string     `gorm:"column:username"`
		LastLogin  *time.Time `gorm:"column:last_login"`
	}

	var rows []row
	now := clock.Now()

	if err := r.DB.WithContext(ctx).
		Model(&domain.User{}).
		Select("users.id as user_id, e.id as employee_id, users.fullname, users.username, users.last_login").
		Joins("JOIN employees e ON e.user_id = users.id AND e.deleted_at IS NULL").
		Joins("JOIN project_employees pe ON pe.employee_id = e.id AND pe.deleted_at IS NULL AND pe.start_date <= ? AND (pe.last_date IS NULL OR pe.last_date >= ?)", now, now).
		Where("users.role = ?", "employee").
		Where("users.last_login >= ? AND users.last_login < ?", since, until).
		Where("pe.payment_schedule = ?", schedule).
		Group("users.id, e.id, users.fullname, users.username, users.last_login, users.role, pe.payment_schedule").
		Order("users.last_login DESC").
		Scan(&rows).Error; err != nil {
		return nil, domain.NewInternalError("failed to get active employees by schedule", err)
	}

	result := make([]*domain.ActiveEmployeeUser, len(rows))
	for i, r := range rows {
		result[i] = &domain.ActiveEmployeeUser{
			UserID:     r.UserID,
			EmployeeID: r.EmployeeID,
			Fullname:   r.Fullname,
			Username:   r.Username,
			LastLogin:  r.LastLogin,
		}
	}
	return result, nil
}

func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	if err := r.DB.WithContext(ctx).Model(&domain.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, domain.NewInternalError("failed to check username existence", err)
	}

	return count > 0, nil
}

func (r *UserRepository) GetByCCCD(ctx context.Context, cccd string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Where("cccd = ?", cccd).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by CCCD", err)
	}
	return &user, nil
}

func (r *UserRepository) GetByMobile(ctx context.Context, mobile string) (*domain.User, error) {
	var user domain.User
	if err := r.DB.WithContext(ctx).Where("mobile = ?", mobile).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("user not found")
		}
		return nil, domain.NewInternalError("failed to get user by mobile", err)
	}
	return &user, nil
}

func (r *UserRepository) FindUsersWithNullLastLogin(ctx context.Context) ([]*domain.User, error) {
	var users []*domain.User
	if err := r.DB.WithContext(ctx).Where("last_login IS NULL").Find(&users).Error; err != nil {
		return nil, domain.NewInternalError("failed to find users with null last_login", err)
	}

	return users, nil
}

func (r *UserRepository) ListWithFilters(ctx context.Context, role *domain.UserRole, search string, ids []uint, lastLoginToday bool, sortBy string, sortOrder string, limit, offset int) ([]*domain.User, int64, error) {
	var users []*domain.User
	var total int64

	query := r.DB.WithContext(ctx).Model(&domain.User{})

	// Apply IDs filter if provided
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}

	// Apply role filter if provided
	if role != nil {
		query = query.Where("role = ?", *role)
	}

	// Apply search filter if provided (case-insensitive, partial match, vietnamese normalization)
	if search != "" {
		// Normalize search term for Vietnamese text
		normalizedSearch := "%" + search + "%"
		query = query.Where(
			"LOWER(username) LIKE LOWER(?) OR LOWER(fullname) LIKE LOWER(?) OR LOWER(email) LIKE LOWER(?)",
			normalizedSearch, normalizedSearch, normalizedSearch,
		)
	}

	// Apply last_login_today filter if enabled
	if lastLoginToday {
		query = query.Where("last_login >= ?", timeutil.TodayStart())
	}

	// Count total matching records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, domain.NewInternalError("failed to count users", err)
	}

	// Apply sorting (sanitize user-controlled sort fields against SQL injection)
	orderClause := common.SanitizeSortColumn(sortBy, "created_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC")

	// Apply pagination and fetch results
	if err := query.Order(orderClause).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, 0, domain.NewInternalError("failed to list users with filters", err)
	}

	return users, total, nil
}
