package domain

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/phone"
	"api-server/internal/pkg/utils"

	"gorm.io/gorm"
)

// FieldChange records the before/after values of a single auditable field.
type FieldChange struct {
	Before any `json:"before"`
	After  any `json:"after"`
}

// CompareEmployees compares all auditable fields of two employees and returns
// a map of changed field names to their before/after values.
func CompareEmployees(original, updated *Employee) map[string]FieldChange {
	if original == nil || updated == nil {
		return nil
	}

	changes := make(map[string]FieldChange)

	if original.Fullname != updated.Fullname {
		changes["fullname"] = FieldChange{Before: original.Fullname, After: updated.Fullname}
	}
	if !equalStringPtr(original.Email, updated.Email) {
		changes["email"] = FieldChange{Before: strPtrToStr(original.Email), After: strPtrToStr(updated.Email)}
	}
	if original.CCCD != updated.CCCD {
		changes["cccd"] = FieldChange{Before: original.CCCD, After: updated.CCCD}
	}
	if original.Address != updated.Address {
		changes["address"] = FieldChange{Before: original.Address, After: updated.Address}
	}
	if original.Mobile != updated.Mobile {
		changes["mobile"] = FieldChange{Before: original.Mobile, After: updated.Mobile}
	}
	if !equalUintPtr(original.BankID, updated.BankID) {
		changes["bank_id"] = FieldChange{Before: uintPtrToVal(original.BankID), After: uintPtrToVal(updated.BankID)}
	}
	if original.BankAccountNumber != updated.BankAccountNumber {
		changes["bank_account_number"] = FieldChange{Before: original.BankAccountNumber, After: updated.BankAccountNumber}
	}
	if original.BankAccountName != updated.BankAccountName {
		changes["bank_account_name"] = FieldChange{Before: original.BankAccountName, After: updated.BankAccountName}
	}
	if !equalDatePtr(original.DateOfBirth, updated.DateOfBirth) {
		changes["date_of_birth"] = FieldChange{Before: timePtrToStr(original.DateOfBirth), After: timePtrToStr(updated.DateOfBirth)}
	}

	return changes
}

func equalStringPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalUintPtr(a, b *uint) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func equalTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}

func equalDatePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func strPtrToStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func uintPtrToVal(p *uint) any {
	if p == nil {
		return nil
	}
	return fmt.Sprintf("%d", *p)
}

func timePtrToStr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// Employee represents an employee in the domain
type Employee struct {
	ID                uint    `json:"id" gorm:"primarykey;type:bigint unsigned"`
	Fullname          string  `json:"fullname" gorm:"type:varchar(255);not null"`
	Email             *string `json:"email" gorm:"type:varchar(255);uniqueIndex:unique_email_deleted_at"`
	CCCD              string  `json:"cccd" gorm:"type:varchar(255);not null;uniqueIndex:unique_cccd_deleted_at;comment:'Citizen ID - Can cong cong dan (12 digits)'"`
	Address           string  `json:"address" gorm:"type:text"`
	Mobile            string  `json:"mobile" gorm:"type:varchar(15)"`
	BankID            *uint   `json:"bank_id" gorm:"type:bigint unsigned"`
	BankAccountNumber string  `json:"bank_account_number" gorm:"type:varchar(30)"`
	BankAccountName   string  `json:"bank_account_name" gorm:"type:varchar(255)"`

	// BankAccountStatus reflects the outcome of the last OnePay account
	// verification (see BankAccountStatus* constants). Defaults to "valid"
	// so pre-existing rows are treated as valid without a backfill.
	BankAccountStatus        string     `json:"bank_account_status" gorm:"type:varchar(20);not null;default:'valid';column:bank_account_status"`
	BankAccountInvalidReason *string    `json:"bank_account_invalid_reason,omitempty" gorm:"type:varchar(500);column:bank_account_invalid_reason"`
	BankAccountValidatedAt   *time.Time `json:"bank_account_validated_at,omitempty" gorm:"type:datetime;column:bank_account_validated_at"`

	DateOfBirth *time.Time     `json:"date_of_birth" gorm:"type:date"`
	UserID      *uint          `json:"user_id" gorm:"type:bigint unsigned;index"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index;uniqueIndex:unique_email_deleted_at;uniqueIndex:unique_cccd_deleted_at"`
	CreatedBy   uint           `json:"created_by" gorm:"not null;type:bigint unsigned"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`

	// Relationships
	Creator User  `json:"creator" gorm:"foreignKey:CreatedBy;references:ID"`
	Bank    *Bank `json:"bank,omitempty" gorm:"foreignKey:BankID;references:ID"`
	User    *User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// GetAuditEntityType implements Auditable interface
func (e Employee) GetAuditEntityType() string {
	return "employee"
}

// GetAuditEntityID implements Auditable interface
func (e Employee) GetAuditEntityID() uint {
	return e.ID
}

// EmployeeRepository defines the interface for employee persistence operations
type EmployeeRepository interface {
	Create(ctx context.Context, employee *Employee) error
	GetByID(ctx context.Context, id uint) (*Employee, error)
	GetByIDs(ctx context.Context, ids []int64) ([]*Employee, error)
	GetByIDForUpdate(ctx context.Context, id uint) (*Employee, error)
	GetByUserID(ctx context.Context, userID uint) (*Employee, error)
	GetByCCCD(ctx context.Context, cccd string) (*Employee, error)
	// GetByMobile resolves a single employee by mobile number. Duplicate numbers
	// are permitted by the schema, so it fails closed (conflict error) when more
	// than one employee shares the number instead of returning an arbitrary row.
	GetByMobile(ctx context.Context, mobile string) (*Employee, error)
	// ListByMobile returns every active employee sharing the mobile number,
	// ordered by id. Used by callers that must react to duplicates instead of
	// resolving one employee.
	ListByMobile(ctx context.Context, mobile string) ([]*Employee, error)
	GetByEmail(ctx context.Context, email string) (*Employee, error)
	ExistsByCCCD(ctx context.Context, cccd string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	GetByBankAccountNumber(ctx context.Context, bankAccountNumber string) (*Employee, error)
	Update(ctx context.Context, employee *Employee) error
	Delete(ctx context.Context, id uint) error
	HardDelete(ctx context.Context, id uint) error
	List(ctx context.Context, filters EmployeeFilters) ([]*Employee, error)
	ListWithProjects(ctx context.Context, filters EmployeeFilters) ([]*EmployeeWithProject, error)
	ListWithAllProjects(ctx context.Context, filters EmployeeFilters) ([]*EmployeeWithProjects, error)
	Count(ctx context.Context, filters EmployeeFilters) (int64, error)
	BulkCreate(ctx context.Context, employees []*Employee) error
	GetByProject(ctx context.Context, projectID uint) ([]*Employee, error)
	GetByProjectWithCreatorFilter(ctx context.Context, projectID uint, createdBy *uint) ([]*Employee, error)
	GetEmployeesSummary(ctx context.Context) (*EmployeesSummary, error)
	GetEmployeesSummaryForCreator(ctx context.Context, createdBy uint) (*EmployeesSummary, error)
	GetUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters EmployeeFilters) ([]*Employee, error)
	CountUnassignedEmployeesAtDate(ctx context.Context, atDate time.Time, filters EmployeeFilters) (int64, error)
	GetActiveCountAtDate(ctx context.Context, date time.Time) (int, error)
	GetActiveCountAtDateForCreator(ctx context.Context, date time.Time, createdBy uint) (int, error)
	GetRecentEmployees(ctx context.Context, startDate, endDate time.Time, limit int) ([]*Employee, error)
	GetRecentEmployeesPaginated(ctx context.Context, startDate, endDate time.Time, limit, offset int) ([]*Employee, error)
	CountRecentEmployees(ctx context.Context, startDate, endDate time.Time) (int64, error)
	SearchEmployees(ctx context.Context, search string, limit int) ([]*EmployeeWithProject, error)
	// ListAccessibleIDs returns IDs of all employees accessible to a user
	// (created by them, shared via employee_users, or assigned to their projects).
	// Used to annotate global-pool list rows with is_accessible.
	ListAccessibleIDs(ctx context.Context, userID uint) ([]uint, error)
	GetEmployeesWithMissingBankDetails(ctx context.Context, filters EmployeeFilters) ([]*EmployeeWithProjects, error)
	CountEmployeesWithMissingBankDetails(ctx context.Context, filters EmployeeFilters) (int64, error)
	// UpdateColumns performs a targeted update of specific columns for an employee.
	// Used by import flows to update bank info or user_id without a full Save.
	UpdateColumns(ctx context.Context, id uint, columns map[string]any) error
}

// EmployeeFilters represents filtering options for employee queries
type EmployeeFilters struct {
	CreatedBy               *uint
	ProjectID               *uint
	ProjectIDs              []uint
	AccessibleBy            *uint      // Filter employees accessible by user (owned or shared)
	SkipAccessibilityFilter bool       // Skip accessibility filter for separate query approach
	Search                  string     // Search in fullname, cccd, email
	Status                  string     // working or unassigned
	PaymentSchedule         string     // Filter by payment schedule: weekly, monthly, flexible
	FromDate                *time.Time // Filter by created_at >= fromDate
	ToDate                  *time.Time // Filter by created_at <= toDate
	Limit                   int
	Offset                  int
	SortBy                  string
	SortOrder               string
}

// GetStatuses implements common.StatusFilter interface
func (ef *EmployeeFilters) GetStatuses() []string {
	if ef.Status == "" {
		return nil
	}
	return []string{ef.Status}
}

// GetStatusField implements common.StatusFilter interface
func (ef *EmployeeFilters) GetStatusField() string {
	return "status"
}

// GetFromDate implements common.DateRangeFilter interface
func (ef *EmployeeFilters) GetFromDate() *time.Time {
	return ef.FromDate
}

// GetToDate implements common.DateRangeFilter interface
func (ef *EmployeeFilters) GetToDate() *time.Time {
	return ef.ToDate
}

// GetDateField implements common.DateRangeFilter interface
func (ef *EmployeeFilters) GetDateField() string {
	return "created_at"
}

// GetCreatedBy implements common.CreatorFilter interface
func (ef *EmployeeFilters) GetCreatedBy() *uint {
	return ef.CreatedBy
}

// GetCreatorField implements common.CreatorFilter interface
func (ef *EmployeeFilters) GetCreatorField() string {
	return "created_by"
}

// GetSearch implements common.SearchFilter interface
func (ef *EmployeeFilters) GetSearch() string {
	return ef.Search
}

// GetSearchFields implements common.SearchFilter interface
func (ef *EmployeeFilters) GetSearchFields() []string {
	return []string{"fullname", "cccd", "email"}
}

// ValidateFullname validates the employee's fullname
func (e *Employee) ValidateFullname() error {
	if e.Fullname == "" {
		return NewValidationError("fullname is required")
	}
	return nil
}

// ValidateCCCD validates the employee's citizen ID
func (e *Employee) ValidateCCCD() error {
	if e.CCCD == "" {
		return NewValidationError("CCCD is required")
	}

	// Check if all characters are alphanumeric
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]+$`, e.CCCD)
	if !matched {
		return NewValidationError("CCCD must contain only alphanumeric characters")
	}

	return nil
}

// ValidateEmail validates the employee's email format
func (e *Employee) ValidateEmail() error {
	if e.Email != nil && *e.Email != "" {
		// Basic email validation
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(*e.Email) {
			return NewValidationError("invalid email format")
		}
	}
	return nil
}

// ValidateMobile validates the employee's mobile number
func (e *Employee) ValidateMobile() error {
	if e.Mobile != "" {
		// Check if mobile contains only digits
		matched, _ := regexp.MatchString(`^\d+$`, e.Mobile)
		if !matched {
			return NewValidationError("mobile number must contain only digits")
		}
		// A 12-digit CCCD is never a mobile number: it means the CCCD was typed
		// into the phone field (mis-mapped import column or manual slip). Reject
		// it instead of persisting an unusable contact number.
		if phone.IsCCCDCard(e.Mobile) {
			return NewValidationError("mobile number must not be a CCCD")
		}
	}
	return nil
}

// ValidateBankingInfo validates banking information
func (e *Employee) ValidateBankingInfo() error {
	// If any banking field is provided, all banking fields are required
	if e.BankID != nil || e.BankAccountNumber != "" || e.BankAccountName != "" {
		if e.BankID == nil {
			return NewValidationError("bank selection is required when providing banking information")
		}
		if e.BankAccountNumber == "" {
			return NewValidationError("bank account number is required")
		}
		if e.BankAccountName == "" {
			return NewValidationError("bank account name is required")
		}
	}
	return nil
}

// ValidateDates validates date fields
func (e *Employee) ValidateDates() error {
	now := clock.Now()

	if e.DateOfBirth != nil {
		if e.DateOfBirth.After(now) {
			return NewValidationError("date of birth cannot be in the future")
		}
		// Must be at least 16 years old
		sixteenYearsAgo := now.AddDate(-16, 0, 0)
		if e.DateOfBirth.After(sixteenYearsAgo) {
			return NewValidationError("employee must be at least 16 years old")
		}
	}

	return nil
}

// IsValid validates the entire employee entity
func (e *Employee) IsValid() error {
	if err := e.ValidateFullname(); err != nil {
		return err
	}
	if err := e.ValidateCCCD(); err != nil {
		return err
	}
	if err := e.ValidateEmail(); err != nil {
		return err
	}
	if err := e.ValidateMobile(); err != nil {
		return err
	}
	if err := e.ValidateBankingInfo(); err != nil {
		return err
	}
	if err := e.ValidateDates(); err != nil {
		return err
	}
	return nil
}

// BankAccountStatus* enumerate the possible outcomes of a OnePay account
// verification stored on Employee.BankAccountStatus.
const (
	// BankAccountStatusValid means OnePay confirmed the account exists and
	// the holder name matches. This is also the default for rows that
	// predate this feature (assumption: existing accounts are valid).
	BankAccountStatusValid = "valid"
	// BankAccountStatusInvalid means OnePay confirmed the account does not
	// exist OR the holder name does not match the employee name. Such
	// employees appear in the missing-bank-details warning list with the
	// reason populated.
	BankAccountStatusInvalid = "invalid"
	// BankAccountStatusUnverified means the verification could not be
	// completed (OnePay unreachable, 5xx, timeout). Fail-open: these
	// employees are NOT shown in the warning list to avoid false alarms
	// during provider outages.
	BankAccountStatusUnverified = "unverified"
)

// HasBankingInfo returns true if employee has complete banking information
func (e *Employee) HasBankingInfo() bool {
	return e.BankID != nil && e.BankAccountNumber != "" && e.BankAccountName != ""
}

// NeedsBankAccountReview reports whether the employee should appear in the
// admin/partner bank-account warning list: either the banking info is
// missing entirely, OR OnePay has confirmed the account is invalid.
// Unverified accounts are deliberately excluded (fail-open).
func (e *Employee) NeedsBankAccountReview() bool {
	if !e.HasBankingInfo() {
		return true
	}
	return e.BankAccountStatus == BankAccountStatusInvalid
}

// GetAge calculates the employee's age in years
func (e *Employee) GetAge() int {
	if e.DateOfBirth == nil {
		return 0
	}

	now := clock.Now()
	age := now.Year() - e.DateOfBirth.Year()

	// Adjust age if birthday hasn't occurred this year
	if now.Month() < e.DateOfBirth.Month() ||
		(now.Month() == e.DateOfBirth.Month() && now.Day() < e.DateOfBirth.Day()) {
		age--
	}

	return age
}

// CanBeAssignedToProject checks if employee can be assigned to a project
func (e *Employee) CanBeAssignedToProject() bool {
	return !e.DeletedAt.Valid && e.HasBankingInfo()
}

// FormattedFullname returns the employee's full name in Vietnamese title case format.
// Each word's first letter is capitalized while preserving Vietnamese diacritics.
// Example: "đoàn thị hồng" -> "Đoàn Thị Hồng"
func (e *Employee) FormattedFullname() string {
	return utils.ToVietnameseTitleCase(e.Fullname)
}

// Additional types for repository operations

// EmployeeStatistics represents employee statistics
type EmployeeStatistics struct {
	TotalEmployees      int64 `json:"total_employees"`
	WithBankingInfo     int64 `json:"with_banking_info"`
	AssignedToProjects  int64 `json:"assigned_to_projects"`
	UnassignedEmployees int64 `json:"unassigned_employees"`
}

// EmployeesSummary represents aggregated employee metrics for the summary endpoint
type EmployeesSummary struct {
	TotalEmployees          int64 `json:"total_employees"`
	TotalWorkingEmployees   int64 `json:"total_working_employees"`
	EmployeesHiredThisMonth int64 `json:"employees_hired_this_month"`
	SalaryMonthToDate       int64 `json:"salary_month_to_date"`
	PaidMonthToDate         int64 `json:"paid_month_to_date"`
}

// EmployeeWithProject represents an employee with their current active project information
type EmployeeWithProject struct {
	Employee
	ProjectID         *uint      `json:"project_id,omitempty"`
	ProjectName       *string    `json:"project_name,omitempty"`
	ProjectCode       *string    `json:"project_code,omitempty"`
	ProjectClientName *string    `json:"project_client_name,omitempty"`
	Position          *string    `json:"position,omitempty"`
	StartDate         *time.Time `json:"start_date,omitempty"`
	LastDate          *time.Time `json:"last_date,omitempty"`
}

// EmployeeWithProjects represents an employee with all their current active projects
type EmployeeWithProjects struct {
	Employee
	CurrentProjects []CurrentProject `json:"current_projects"`
}

// CurrentProject represents a project assignment for an employee
type CurrentProject struct {
	ProjectID              uint       `json:"project_id"`
	ProjectEmployeeID      uint       `json:"project_employee_id"`
	Name                   string     `json:"name"`
	Code                   string     `json:"code"`
	ClientName             string     `json:"client_name"`
	Position               string     `json:"position"`
	StartDate              time.Time  `json:"start_date"`
	LastDate               *time.Time `json:"last_date"`
	PaymentSchedule        string     `json:"payment_schedule"`
	PendingPaymentSchedule *string    `json:"pending_payment_schedule,omitempty"`
	ScheduleEffectiveFrom  *time.Time `json:"schedule_effective_from,omitempty"`
	IsFlexible             bool       `json:"is_flexible"`
	CheckInEnabled         bool       `json:"check_in_enabled"`
	PendingCheckInEnabled  *bool      `json:"pending_check_in_enabled,omitempty"`
	CheckInEffectiveFrom   *time.Time `json:"check_in_effective_from,omitempty"`
}
