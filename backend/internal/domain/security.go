package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"api-server/internal/pkg/clock"
	"gorm.io/gorm"
)

// BlacklistReason represents the reason for token blacklisting
type BlacklistReason string

const (
	BlacklistReasonLogout      BlacklistReason = "logout"
	BlacklistReasonSecurity    BlacklistReason = "security"
	BlacklistReasonExpired     BlacklistReason = "expired"
	BlacklistReasonAdminRevoke BlacklistReason = "admin_revoke"
)

// AuditAction represents the type of action performed
type AuditAction string

const (
	AuditActionCreate         AuditAction = "CREATE"
	AuditActionUpdate         AuditAction = "UPDATE"
	AuditActionDelete         AuditAction = "DELETE"
	AuditActionApprove        AuditAction = "APPROVE"
	AuditActionReject         AuditAction = "REJECT"
	AuditActionBulkApprove    AuditAction = "BULK_APPROVE"
	AuditActionBulkReject     AuditAction = "BULK_REJECT"
	AuditActionBulkReset      AuditAction = "BULK_RESET"
	AuditActionBulkCreate     AuditAction = "BULK_CREATE"
	AuditActionLogin          AuditAction = "LOGIN"
	AuditActionLogout         AuditAction = "LOGOUT"
	AuditActionChangePassword AuditAction = "CHANGE_PASSWORD"
	AuditActionView           AuditAction = "VIEW"
	AuditActionExport         AuditAction = "EXPORT"
	AuditActionImport         AuditAction = "IMPORT"
	AuditActionSettle         AuditAction = "SETTLE"
	AuditActionExternalPay    AuditAction = "EXTERNAL_PAY"
)

// EntityType represents the type of entity being audited
type EntityType string

const (
	EntityTypeUser                      EntityType = "user"
	EntityTypeProject                   EntityType = "project"
	EntityTypeEmployee                  EntityType = "employee"
	EntityTypeProjectEmployee           EntityType = "project_employee"
	EntityTypeProjectUser               EntityType = "project_user"
	EntityTypeEmployeeUser              EntityType = "employee_user"
	EntityTypeBank                      EntityType = "bank"
	EntityTypePayrate                   EntityType = "payrate"
	EntityTypeTimesheet                 EntityType = "timesheet"
	EntityTypePayroll                   EntityType = "payroll"
	EntityTypeLedgerEntry               EntityType = "ledger_entry"
	EntityTypeSettings                  EntityType = "settings"
	EntityTypeAsset                     EntityType = "asset"
	EntityTypeTransaction               EntityType = "transaction"
	EntityTypeLoan                      EntityType = "loan"
	EntityTypeLender                    EntityType = "lender"
	EntityTypeBulkTransferFile          EntityType = "bulk_transfer_file"
	EntityTypeAdvancePaymentFeeSchedule EntityType = "advance_payment_fee_schedule"
	EntityTypeDisbursementFeeSchedule   EntityType = "disbursement_fee_schedule"
)

// BlacklistedToken represents a blacklisted JWT token
type BlacklistedToken struct {
	ID            uint            `json:"id" gorm:"primarykey;type:bigint unsigned"`
	TokenJTI      string          `json:"token_jti" gorm:"type:varchar(255);not null;uniqueIndex;comment:'JWT ID'"`
	UserID        uint            `json:"user_id" gorm:"not null;type:bigint unsigned"`
	ExpiresAt     time.Time       `json:"expires_at" gorm:"not null"`
	BlacklistedAt time.Time       `json:"blacklisted_at"`
	Reason        BlacklistReason `json:"reason" gorm:"type:enum('logout','security','expired','admin_revoke');default:'logout'"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID         uint        `json:"id"          gorm:"primarykey;type:bigint unsigned"`
	UserID     uint        `json:"user_id"     gorm:"not null;type:bigint unsigned;index"`
	Action     AuditAction `json:"action"      gorm:"type:varchar(50);not null;index:idx_action_entity"`
	EntityType EntityType  `json:"entity_type" gorm:"type:varchar(50);not null;index:idx_action_entity"`
	EntityID   *uint       `json:"entity_id"   gorm:"type:bigint unsigned"`
	Message    string      `json:"message"     gorm:"type:text;not null"`
	IPAddress  *string     `json:"ip_address"  gorm:"type:varchar(45)"`
	Browser    *string     `json:"browser"     gorm:"type:varchar(100)"`
	Platform   *string     `json:"platform"    gorm:"type:varchar(100)"`
	Metadata   *string     `json:"metadata"    gorm:"type:json"`
	CreatedAt  time.Time   `json:"created_at"  gorm:"index"`

	// Joined fields (populated by repository queries, not stored in DB)
	// Using column tag with read-only (->) so GORM populates these from SELECT aliases
	UserFullname string `json:"user_fullname" gorm:"column:user_fullname;->"`
	UserUsername string `json:"user_username" gorm:"column:user_username;->"`
	UserRole     string `json:"user_role"     gorm:"column:user_role;->"`

	// Relationships
	User User `json:"user" gorm:"foreignKey:UserID;references:ID"`
}

// BlacklistedTokenRepository defines the interface for blacklisted token persistence operations
type BlacklistedTokenRepository interface {
	Create(ctx context.Context, token *BlacklistedToken) error
	GetByJTI(ctx context.Context, jti string) (*BlacklistedToken, error)
	IsBlacklisted(ctx context.Context, jti string) (bool, error)
	DeleteExpired(ctx context.Context) error
	List(ctx context.Context, filters BlacklistFilters) ([]*BlacklistedToken, error)
	Count(ctx context.Context, filters BlacklistFilters) (int64, error)
	GetByUser(ctx context.Context, userID uint) ([]*BlacklistedToken, error)
	BlacklistToken(ctx context.Context, jti string, userID uint, expiresAt time.Time, reason BlacklistReason) error
}

// AuditLogRepository defines the interface for audit log persistence operations
type AuditLogRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	GetByID(ctx context.Context, id uint) (*AuditLog, error)
	List(ctx context.Context, filters AuditFilters) ([]*AuditLog, error)
	Count(ctx context.Context, filters AuditFilters) (int64, error)
	DeleteOldLogs(ctx context.Context, olderThan time.Time) error

	// Analytics aggregations for the System Health dashboard. These run
	// GROUP BY + COUNT(DISTINCT) at the database so the application never
	// scans unbounded numbers of raw audit rows.
	GetBrowserPlatformStats(ctx context.Context, since time.Time) ([]BrowserPlatformStat, error)
	GetBrowserPlatformUsers(ctx context.Context, browser, platform string, since time.Time) ([]BrowserPlatformUser, error)
	GetOSGroupedStats(ctx context.Context, since time.Time) ([]OSGroupStat, error)
	GetBrowserGroupedStats(ctx context.Context, since time.Time) ([]BrowserGroupStat, error)
	GetOSGroupedUsers(ctx context.Context, osFamily string, since time.Time) ([]BrowserPlatformUser, error)
	GetBrowserGroupedUsers(ctx context.Context, browserFamily string, since time.Time) ([]BrowserPlatformUser, error)
}

// BlacklistFilters represents filtering options for blacklisted token queries
type BlacklistFilters struct {
	UserID    *uint
	Reason    []BlacklistReason
	FromDate  *time.Time
	ToDate    *time.Time
	Expired   *bool
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// AuditFilters represents filtering options for audit log queries
type AuditFilters struct {
	UserID     *uint
	Action     []AuditAction
	EntityType []EntityType
	EntityID   *uint
	FromDate   *time.Time
	ToDate     *time.Time
	IPAddress  *string
	Limit      int
	Offset     int
	SortBy     string
	SortOrder  string
}

// BrowserPlatformStat represents unique user/action counts for a browser+platform combination.
type BrowserPlatformStat struct {
	Browser      string `json:"browser"`
	Platform     string `json:"platform"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// BrowserPlatformUser represents a user who used a specific browser+platform (or family) combination.
type BrowserPlatformUser struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Fullname string `json:"fullname"`
	Role     string `json:"role"`
	Actions  int    `json:"actions"`
	LastSeen string `json:"last_seen"`
}

// VersionStat holds user/action counts for a single version within a family.
type VersionStat struct {
	Version      string `json:"version"`
	UniqueUsers  int    `json:"unique_users"`
	TotalActions int    `json:"total_actions"`
}

// OSGroupStat holds aggregated stats for one OS family with per-version breakdown.
type OSGroupStat struct {
	OSFamily     string        `json:"os_family"`
	UniqueUsers  int           `json:"unique_users"`
	TotalActions int           `json:"total_actions"`
	Versions     []VersionStat `json:"versions"`
}

// BrowserGroupStat holds aggregated stats for one browser family with per-version breakdown.
type BrowserGroupStat struct {
	BrowserFamily string        `json:"browser_family"`
	UniqueUsers   int           `json:"unique_users"`
	TotalActions  int           `json:"total_actions"`
	Versions      []VersionStat `json:"versions"`
}

// GORM hooks for AuditLog
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.Action == "" {
		return NewValidationError("audit log action is required")
	}
	if a.EntityType == "" {
		return NewValidationError("audit log entity_type is required")
	}
	return nil
}

// SetMetadata JSON-marshals v and stores it in the Metadata field.
// If v is nil, Metadata is set to nil.
func (a *AuditLog) SetMetadata(v map[string]interface{}) error {
	if v == nil {
		a.Metadata = nil
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s := string(b)
	a.Metadata = &s
	return nil
}

// GetMetadata JSON-unmarshals the Metadata field and returns the map.
// Returns nil, nil when Metadata is nil.
func (a *AuditLog) GetMetadata() (map[string]interface{}, error) {
	if a.Metadata == nil {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(*a.Metadata), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// GORM hooks for BlacklistedToken
func (bt *BlacklistedToken) BeforeCreate(tx *gorm.DB) error {
	if bt.BlacklistedAt.IsZero() {
		bt.BlacklistedAt = clock.Now()
	}
	return nil
}

// BlacklistedToken methods

// ValidateTokenJTI validates the token JTI
func (bt *BlacklistedToken) ValidateTokenJTI() error {
	if bt.TokenJTI == "" {
		return NewValidationError("token JTI is required")
	}
	return nil
}

// ValidateUserID validates the user ID
func (bt *BlacklistedToken) ValidateUserID() error {
	if bt.UserID == 0 {
		return NewValidationError("user ID is required")
	}
	return nil
}

// ValidateExpiresAt validates the expiration time
func (bt *BlacklistedToken) ValidateExpiresAt() error {
	if bt.ExpiresAt.IsZero() {
		return NewValidationError("expiration time is required")
	}
	return nil
}

// ValidateReason validates the blacklist reason
func (bt *BlacklistedToken) ValidateReason() error {
	validReasons := map[BlacklistReason]bool{
		BlacklistReasonLogout:      true,
		BlacklistReasonSecurity:    true,
		BlacklistReasonExpired:     true,
		BlacklistReasonAdminRevoke: true,
	}

	if !validReasons[bt.Reason] {
		return NewValidationError("invalid blacklist reason")
	}

	return nil
}

// IsValid validates the entire blacklisted token
func (bt *BlacklistedToken) IsValid() error {
	if err := bt.ValidateTokenJTI(); err != nil {
		return err
	}
	if err := bt.ValidateUserID(); err != nil {
		return err
	}
	if err := bt.ValidateExpiresAt(); err != nil {
		return err
	}
	if err := bt.ValidateReason(); err != nil {
		return err
	}
	return nil
}

// IsExpired returns true if the token has expired
func (bt *BlacklistedToken) IsExpired() bool {
	return clock.Now().After(bt.ExpiresAt)
}

// CanBeDeleted returns true if the token can be deleted (expired)
func (bt *BlacklistedToken) CanBeDeleted() bool {
	return bt.IsExpired()
}

// Helper functions

// Deprecated: Use BuildEventAuditMessage from audit_templates.go for new code.
// This function remains for backward compatibility with financial event factories.
// If entityName is provided, it will be included in the message (e.g., "đã tạo người dùng mới Tài Văn Lộc").
func BuildAuditMessage(action AuditAction, entityType EntityType, entityName string) string {
	if entityType == EntityTypeTimesheet {
		if customMessage, ok := buildTimesheetMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypeProjectEmployee {
		if customMessage, ok := buildProjectAssignmentMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypePayrate {
		if customMessage, ok := buildPayrateMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypeProjectUser {
		if customMessage, ok := buildProjectSharingMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypeEmployeeUser {
		if customMessage, ok := buildEmployeeSharingMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypeTransaction {
		if customMessage, ok := buildTransactionMessage(action, entityName); ok {
			return customMessage
		}
	}
	if entityType == EntityTypeLoan {
		if customMessage, ok := buildLoanMessage(action, entityName); ok {
			return customMessage
		}
	}

	var message string
	switch action {
	case AuditActionCreate:
		switch entityType {
		case EntityTypeUser:
			message = "đã tạo người dùng mới"
		case EntityTypeProject:
			message = "đã tạo dự án mới"
		case EntityTypeEmployee:
			message = "đã tạo nhân viên mới"
		case EntityTypeProjectEmployee:
			message = "đã phân công nhân viên vào dự án"
		case EntityTypeProjectUser:
			message = "đã chia sẻ dự án với người dùng"
		case EntityTypeEmployeeUser:
			message = "đã chia sẻ nhân viên với người dùng"
		case EntityTypeTimesheet:
			message = "đã tạo chấm công mới"
		case EntityTypePayroll:
			message = "đã tạo bảng lương mới"
		case EntityTypeLedgerEntry:
			message = "đã tạo bút toán mới"
		case EntityTypeBank:
			message = "đã thêm ngân hàng mới"
		case EntityTypePayrate:
			message = "đã tạo mức lương mới"
		case EntityTypeSettings:
			message = "đã tạo cài đặt mới"
		case EntityTypeAsset:
			message = "đã tạo tệp tin mới"
		default:
			message = "đã tạo dữ liệu mới"
		}
	case AuditActionUpdate:
		switch entityType {
		case EntityTypeUser:
			message = "đã cập nhật thông tin người dùng"
		case EntityTypeEmployee:
			message = "đã cập nhật thông tin nhân viên"
		case EntityTypeProject:
			message = "đã cập nhật thông tin dự án"
		case EntityTypeProjectEmployee:
			message = "đã cập nhật phân công nhân viên"
		case EntityTypeProjectUser:
			message = "đã cập nhật chia sẻ dự án"
		case EntityTypeEmployeeUser:
			message = "đã cập nhật chia sẻ nhân viên"
		case EntityTypeTimesheet:
			message = "đã cập nhật chấm công"
		case EntityTypePayroll:
			message = "đã cập nhật bảng lương"
		case EntityTypeLedgerEntry:
			message = "đã cập nhật bút toán"
		case EntityTypeBank:
			message = "đã cập nhật thông tin ngân hàng"
		case EntityTypePayrate:
			message = "đã cập nhật mức lương"
		case EntityTypeSettings:
			message = "đã cập nhật cài đặt"
		case EntityTypeAsset:
			message = "đã cập nhật tệp tin"
		default:
			message = "đã cập nhật dữ liệu"
		}
	case AuditActionDelete:
		switch entityType {
		case EntityTypeUser:
			message = "đã xóa người dùng"
		case EntityTypeEmployee:
			message = "đã xóa nhân viên"
		case EntityTypeProject:
			message = "đã xóa dự án"
		case EntityTypeProjectEmployee:
			message = "đã xóa phân công nhân viên khỏi dự án"
		case EntityTypeProjectUser:
			message = "đã xóa chia sẻ dự án"
		case EntityTypeEmployeeUser:
			message = "đã xóa chia sẻ nhân viên"
		case EntityTypeTimesheet:
			message = "đã xóa chấm công"
		case EntityTypePayroll:
			message = "đã xóa bảng lương"
		case EntityTypeLedgerEntry:
			message = "đã xóa bút toán"
		case EntityTypeBank:
			message = "đã xóa ngân hàng"
		case EntityTypePayrate:
			message = "đã xóa mức lương"
		case EntityTypeSettings:
			message = "đã xóa cài đặt"
		case EntityTypeAsset:
			message = "đã xóa tệp tin"
		default:
			message = "đã xóa dữ liệu"
		}
	case AuditActionApprove:
		switch entityType {
		case EntityTypeTimesheet:
			message = "đã duyệt chấm công"
		case EntityTypePayroll:
			message = "đã duyệt bảng lương"
		case EntityTypeLedgerEntry:
			message = "đã duyệt bút toán"
		default:
			message = "đã duyệt dữ liệu"
		}
	case AuditActionReject:
		switch entityType {
		case EntityTypeTimesheet:
			message = "đã từ chối chấm công"
		case EntityTypePayroll:
			message = "đã từ chối bảng lương"
		case EntityTypeLedgerEntry:
			message = "đã từ chối bút toán"
		default:
			message = "đã từ chối dữ liệu"
		}
	case AuditActionSettle:
		switch entityType {
		case EntityTypeTransaction:
			message = "đã thanh toán giao dịch"
		default:
			message = "đã thanh toán"
		}
	case AuditActionLogin:
		message = "đã đăng nhập vào hệ thống"
	case AuditActionLogout:
		message = "đã đăng xuất khỏi hệ thống"
	case AuditActionChangePassword:
		message = "đã đổi mật khẩu"
	case AuditActionExport:
		message = "đã xuất dữ liệu"
	case AuditActionImport:
		message = "đã nhập dữ liệu"
	default:
		message = "đã thực hiện thao tác hệ thống"
	}

	// Append entity name if provided, except for self-referential actions
	// (LOGIN, LOGOUT, CHANGE_PASSWORD don't need entity names as they are performed by the user on themselves)
	selfReferentialActions := map[AuditAction]bool{
		AuditActionLogin:          true,
		AuditActionLogout:         true,
		AuditActionChangePassword: true,
	}

	if entityName != "" && !selfReferentialActions[action] {
		message = fmt.Sprintf("%s %s", message, entityName)
	}

	return message
}

func buildProjectAssignmentMessage(action AuditAction, entityName string) (string, bool) {
	employeeName, projectName, ok := parseProjectAssignmentNames(entityName)
	if !ok {
		return "", false
	}

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã phân công nhân viên %s vào dự án %s", employeeName, projectName), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật phân công nhân viên %s vào dự án %s", employeeName, projectName), true
	case AuditActionDelete:
		return fmt.Sprintf("đã xóa phân công nhân viên %s khỏi dự án %s", employeeName, projectName), true
	default:
		return "", false
	}
}

func parseProjectAssignmentNames(entityName string) (string, string, bool) {
	if entityName == "" {
		return "", "", false
	}

	parts := strings.SplitN(entityName, "||", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	employeeName := strings.TrimSpace(parts[0])
	projectName := strings.TrimSpace(parts[1])
	if employeeName == "" || projectName == "" {
		return "", "", false
	}

	return employeeName, projectName, true
}

func parseTwoEntityNames(entityName string) (string, string, bool) {
	if entityName == "" {
		return "", "", false
	}

	parts := strings.SplitN(entityName, "||", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	first := strings.TrimSpace(parts[0])
	second := strings.TrimSpace(parts[1])
	if first == "" || second == "" {
		return "", "", false
	}

	return first, second, true
}

func buildPayrateMessage(action AuditAction, entityName string) (string, bool) {
	projectName := strings.TrimSpace(entityName)
	if projectName == "" {
		return "", false
	}

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã tạo mức lương mới cho dự án %s", projectName), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật mức lương cho dự án %s", projectName), true
	case AuditActionDelete:
		return fmt.Sprintf("đã xóa mức lương cho dự án %s", projectName), true
	default:
		return "", false
	}
}

func buildProjectSharingMessage(action AuditAction, entityName string) (string, bool) {
	projectName, userName, ok := parseTwoEntityNames(entityName)
	if !ok {
		return "", false
	}

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã chia sẻ dự án %s với người dùng %s", projectName, userName), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật chia sẻ dự án %s với người dùng %s", projectName, userName), true
	case AuditActionDelete:
		return fmt.Sprintf("đã hủy chia sẻ dự án %s với người dùng %s", projectName, userName), true
	default:
		return "", false
	}
}

func buildEmployeeSharingMessage(action AuditAction, entityName string) (string, bool) {
	employeeName, userName, ok := parseTwoEntityNames(entityName)
	if !ok {
		return "", false
	}

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã chia sẻ nhân viên %s với người dùng %s", employeeName, userName), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật chia sẻ nhân viên %s với người dùng %s", employeeName, userName), true
	case AuditActionDelete:
		return fmt.Sprintf("đã hủy chia sẻ nhân viên %s với người dùng %s", employeeName, userName), true
	default:
		return "", false
	}
}

func buildTimesheetMessage(action AuditAction, entityName string) (string, bool) {
	dateText, employeeName, projectName, ok := parseTimesheetEntityName(entityName)
	if !ok {
		return "", false
	}

	var parts []string
	switch action {
	case AuditActionCreate:
		parts = append(parts, "đã tạo chấm công")
	case AuditActionUpdate:
		parts = append(parts, "đã cập nhật chấm công")
	case AuditActionDelete:
		parts = append(parts, "đã xóa chấm công")
	case AuditActionApprove:
		parts = append(parts, "đã duyệt chấm công")
	case AuditActionReject:
		parts = append(parts, "đã từ chối chấm công")
	default:
		return "", false
	}

	if dateText != "" {
		parts = append(parts, fmt.Sprintf("ngày %s", dateText))
	}

	if employeeName != "" {
		parts = append(parts, fmt.Sprintf("của %s", employeeName))
	}

	if projectName != "" {
		parts = append(parts, fmt.Sprintf("tại dự án %s", projectName))
	}

	return strings.Join(parts, " "), true
}

func buildTransactionMessage(action AuditAction, entityName string) (string, bool) {
	amount, txType, ok := parseAmountAndType(entityName)
	if !ok {
		return "", false
	}

	amountText := formatVND(amount)
	typeLabel := getTransactionTypeLabel(txType)

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã tạo giao dịch %s số tiền %sđ", typeLabel, amountText), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật giao dịch %s số tiền %sđ", typeLabel, amountText), true
	case AuditActionDelete:
		return fmt.Sprintf("đã xóa giao dịch %s số tiền %sđ", typeLabel, amountText), true
	case AuditActionSettle:
		return fmt.Sprintf("đã thanh toán giao dịch số tiền %sđ", amountText), true
	default:
		return "", false
	}
}

func buildLoanMessage(action AuditAction, entityName string) (string, bool) {
	partyName, amount, ok := parseNameAndAmount(entityName)
	if !ok {
		return "", false
	}

	amountText := formatVND(amount)

	switch action {
	case AuditActionCreate:
		return fmt.Sprintf("đã tạo khoản vay số tiền %sđ với %s", amountText, partyName), true
	case AuditActionUpdate:
		return fmt.Sprintf("đã cập nhật khoản vay số tiền %sđ với %s", amountText, partyName), true
	case AuditActionDelete:
		return fmt.Sprintf("đã xóa khoản vay số tiền %sđ với %s", amountText, partyName), true
	case AuditActionApprove:
		return fmt.Sprintf("đã phê duyệt khoản vay số tiền %sđ với %s", amountText, partyName), true
	case AuditActionReject:
		return fmt.Sprintf("đã từ chối khoản vay của %s", partyName), true
	default:
		return "", false
	}
}

func parseAmountAndType(entityName string) (int64, string, bool) {
	if entityName == "" {
		return 0, "", false
	}

	parts := strings.SplitN(entityName, "||", 2)
	if len(parts) == 0 {
		return 0, "", false
	}

	amountStr := strings.TrimSpace(parts[0])
	if amountStr == "" {
		return 0, "", false
	}

	var amount int64
	if _, err := fmt.Sscanf(amountStr, "%d", &amount); err != nil {
		return 0, "", false
	}

	txType := ""
	if len(parts) == 2 {
		txType = strings.TrimSpace(parts[1])
	}

	return amount, txType, true
}

func parseNameAndAmount(entityName string) (string, int64, bool) {
	if entityName == "" {
		return "", 0, false
	}

	parts := strings.SplitN(entityName, "||", 2)
	if len(parts) == 0 {
		return "", 0, false
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", 0, false
	}

	if len(parts) == 1 || strings.TrimSpace(parts[1]) == "" {
		return name, 0, true
	}

	var amount int64
	if _, err := fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &amount); err != nil {
		return "", 0, false
	}

	return name, amount, true
}

func formatVND(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	n := len(s)
	if n <= 3 {
		return s
	}

	var segments []string
	for n > 3 {
		segments = append([]string{s[n-3 : n]}, segments...)
		n -= 3
	}
	segments = append([]string{s[0:n]}, segments...)

	return strings.Join(segments, ".")
}

func getTransactionTypeLabel(typeStr string) string {
	if strings.TrimSpace(typeStr) == "" {
		return "khác"
	}

	return TransactionType(typeStr).Label()
}

func parseTimesheetEntityName(entityName string) (string, string, string, bool) {
	if entityName == "" {
		return "", "", "", false
	}

	parts := strings.SplitN(entityName, "||", 3)
	for len(parts) < 3 {
		parts = append(parts, "")
	}

	var (
		dateText     string
		employeeName string
		projectName  string
	)

	rawDate := strings.TrimSpace(parts[0])
	if rawDate != "" {
		if t, err := time.Parse("2006-01-02", rawDate); err == nil {
			dateText = t.Format("02-01")
		}
	}

	employeeName = strings.TrimSpace(parts[1])
	projectName = strings.TrimSpace(parts[2])

	if dateText == "" && employeeName == "" && projectName == "" {
		return "", "", "", false
	}

	return dateText, employeeName, projectName, true
}

// Deprecated: Use BuildEventAuditMessage from audit_templates.go for new code.
// This function remains for backward compatibility with bulk timesheet events.
// Example: "tạo 18 chấm công", "duyệt 25 chấm công"
func BuildBulkAuditMessage(action AuditAction, entityType EntityType, count int) string {
	var verb string
	switch action {
	case AuditActionBulkCreate:
		verb = "tạo"
	case AuditActionBulkApprove:
		verb = "duyệt"
	case AuditActionBulkReject:
		verb = "từ chối"
	case AuditActionBulkReset:
		verb = "reset"
	default:
		verb = "xử lý"
	}

	var entityName string
	switch entityType {
	case EntityTypeTimesheet:
		entityName = "chấm công"
	case EntityTypePayroll:
		entityName = "bảng lương"
	case EntityTypeUser:
		entityName = "người dùng"
	case EntityTypeEmployee:
		entityName = "nhân viên"
	case EntityTypeProject:
		entityName = "dự án"
	case EntityTypeProjectEmployee:
		entityName = "phân công nhân viên"
	case EntityTypeProjectUser:
		entityName = "chia sẻ dự án"
	case EntityTypeEmployeeUser:
		entityName = "chia sẻ nhân viên"
	case EntityTypeLedgerEntry:
		entityName = "bút toán"
	case EntityTypeBank:
		entityName = "ngân hàng"
	case EntityTypePayrate:
		entityName = "mức lương"
	case EntityTypeSettings:
		entityName = "cài đặt"
	case EntityTypeAsset:
		entityName = "tệp tin"
	default:
		entityName = "bản ghi"
	}

	return fmt.Sprintf("%s %d %s", verb, count, entityName)
}

// CreateBlacklistedToken creates a new blacklisted token entry
func CreateBlacklistedToken(jti string, userID uint, expiresAt time.Time, reason BlacklistReason) *BlacklistedToken {
	return &BlacklistedToken{
		TokenJTI:  jti,
		UserID:    userID,
		ExpiresAt: expiresAt,
		Reason:    reason,
	}
}
