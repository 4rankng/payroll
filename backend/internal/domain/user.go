package domain

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// UserRole represents user role enum
type UserRole string

const (
	RoleAdmin      UserRole = "admin"
	RolePartner    UserRole = "partner"
	RoleEmployee   UserRole = "employee"
	RoleAdvPartner UserRole = "adv_partner"
	RoleAccountant UserRole = "accountant"
)

// User represents a user entity in the domain
type User struct {
	ID                  uint           `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Username            string         `json:"username" gorm:"type:varchar(255);uniqueIndex;not null"`
	Email               *string        `json:"email" gorm:"type:varchar(255);index"`
	Password            string         `json:"-" gorm:"longtext;not null"` // Never expose password in JSON
	Fullname            string         `json:"fullname" gorm:"type:varchar(255);not null"`
	CCCD                *string        `json:"cccd,omitempty" gorm:"type:varchar(20);uniqueIndex:unique_user_cccd_deleted_at;comment:'Citizen ID for admin/partner login'"`
	Mobile              *string        `json:"mobile,omitempty" gorm:"type:varchar(15);uniqueIndex:unique_user_mobile_deleted_at"`
	Role                UserRole       `json:"role" gorm:"type:enum('admin','partner','employee','adv_partner','accountant');not null;default:'employee'"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:unique_user_cccd_deleted_at;uniqueIndex:unique_user_mobile_deleted_at"`
	LastLogin           *time.Time     `json:"last_login" gorm:"type:datetime(3)"`
	TokensInvalidBefore *time.Time     `json:"-" gorm:"type:datetime(3);comment:'When non-null, any JWT issued before this instant is rejected. Used by RevokeUserTokens.'"`
	OTPFailedAttempts   int            `json:"-" gorm:"type:int;not null;default:0;comment:'consecutive failed OTP verifies; reset on success'"`
	OTPLockedUntil      *time.Time     `json:"-" gorm:"type:datetime(3);comment:'when set, OTP-gated login is refused until this instant'"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// GetAuditEntityType implements Auditable interface
func (u User) GetAuditEntityType() string {
	return "user"
}

// GetAuditEntityID implements Auditable interface
func (u User) GetAuditEntityID() uint {
	return u.ID
}

// IsActive returns true if the user is active (not soft deleted)
func (u User) IsActive() bool {
	return !u.DeletedAt.Valid
}

// ActiveEmployeeUser represents a user with their employee info for activity stats
type ActiveEmployeeUser struct {
	UserID     uint       `json:"user_id"`
	EmployeeID uint       `json:"employee_id"`
	Fullname   string     `json:"fullname"`
	Username   string     `json:"username"`
	LastLogin  *time.Time `json:"last_login"`
}

// UserRepository defines the interface for user persistence operations
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uint) (*User, error)
	GetByIDs(ctx context.Context, ids []uint) (map[uint]*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByUsernameIncludingDeleted(ctx context.Context, username string) (*User, error)
	GetUsernamesByPrefix(ctx context.Context, prefix string) ([]string, error)
	GetByCCCD(ctx context.Context, cccd string) (*User, error)
	GetByMobile(ctx context.Context, mobile string) (*User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	Update(ctx context.Context, user *User) error
	UpdateLastLogin(ctx context.Context, userID uint, lastLogin time.Time) error
	// UpdateTokensInvalidBefore sets the timestamp before which all of the user's
	// JWTs are considered invalid. Passing a nil timestamp clears the field.
	UpdateTokensInvalidBefore(ctx context.Context, userID uint, invalidBefore time.Time) error
	// UpdatePasswordAndInvalidateSessions sets the user's password hash AND
	// tokens_invalid_before in a single DB transaction. Used by self-service
	// password reset so the two writes commit atomically — a partial commit
	// (password changed but sessions not killed) would leave stolen JWTs valid
	// for up to 14 days (Red Team C1). The password is written via a column-
	// scoped UPDATE (not full-row Save) so concurrent profile edits aren't
	// clobbered (Red Team Failure-Mode-F8).
	UpdatePasswordAndInvalidateSessions(ctx context.Context, userID uint, hashedPassword string, invalidBefore time.Time) error
	// UpdateOTPLockout sets the user's consecutive failed-OTP-attempt count and
	// the optional lockout expiry. Pass lockedUntil=nil to clear an active lockout
	// (e.g. on successful verify, which also resets failedAttempts to 0).
	UpdateOTPLockout(ctx context.Context, userID uint, failedAttempts int, lockedUntil *time.Time) error
	Delete(ctx context.Context, id uint) error
	Restore(ctx context.Context, id uint) error
	List(ctx context.Context, limit, offset int) ([]*User, error)
	ListByRole(ctx context.Context, role UserRole) ([]*User, error)
	ListWithFilters(ctx context.Context, role *UserRole, search string, ids []uint, lastLoginToday bool, sortBy string, sortOrder string, limit, offset int) ([]*User, int64, error)
	Count(ctx context.Context) (int64, error)
	CountByRole(ctx context.Context, role UserRole) (int64, error)
	CountRecentLogins(ctx context.Context, since time.Time) (int64, error)
	CountActiveEmployeesBySchedule(ctx context.Context, since, until time.Time) (weekly, monthly, flexible int, err error)
	GetActiveEmployeesBySchedule(ctx context.Context, since, until time.Time, schedule string) ([]*ActiveEmployeeUser, error)
	FindUsersWithNullLastLogin(ctx context.Context, limit, offset int) ([]*User, error)
	CountUsersWithNullLastLogin(ctx context.Context) (int64, error)
}

// ValidateEmail validates the user's email format
func (u *User) ValidateEmail() error {
	// Email is optional, no validation needed since binding tags handle format validation
	return nil
}

// ValidateUsername validates the user's username
func (u *User) ValidateUsername() error {
	if u.Username == "" {
		return NewValidationError("username is required")
	}
	if len(u.Username) < 3 {
		return NewValidationError("username must be at least 3 characters long")
	}
	return nil
}

// ValidateFullname validates the user's fullname
func (u *User) ValidateFullname() error {
	if u.Fullname == "" {
		return NewValidationError("fullname is required")
	}
	return nil
}

// ValidateRole validates the user's role
func (u *User) ValidateRole() error {
	if u.Role != RoleAdmin && u.Role != RolePartner && u.Role != RoleEmployee && u.Role != RoleAdvPartner && u.Role != RoleAccountant {
		return NewValidationError("role must be either 'admin', 'partner', 'employee', 'adv_partner', or 'accountant'")
	}
	return nil
}

// IsValid validates the entire user entity
func (u *User) IsValid() error {
	if err := u.ValidateEmail(); err != nil {
		return err
	}
	if err := u.ValidateUsername(); err != nil {
		return err
	}
	if err := u.ValidateFullname(); err != nil {
		return err
	}
	if err := u.ValidateRole(); err != nil {
		return err
	}
	return nil
}

// IsAdmin returns true if user role is admin
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPartner returns true if user role is partner
func (u *User) IsPartner() bool {
	return u.Role == RolePartner
}

// IsEmployee returns true if user role is employee
func (u *User) IsEmployee() bool {
	return u.Role == RoleEmployee
}

// IsAdvPartner returns true if user role is adv_partner
func (u *User) IsAdvPartner() bool {
	return u.Role == RoleAdvPartner
}

// IsAccountant returns true if user role is accountant (kế toán). The
// accountant workspace covers timesheet approval and the bulk-transfer / sao
// ke pipeline only — enforced by Casbin, not by this predicate.
func (u *User) IsAccountant() bool {
	return u.Role == RoleAccountant
}
