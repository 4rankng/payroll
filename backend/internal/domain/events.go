package domain

import (
	"context"
	"time"
)

// DomainEvent represents a domain event that occurred in the system
type DomainEvent interface {
	EventType() string
	OccurredAt() time.Time
	AggregateID() uint
	UserID() uint
	GetAuditMessage() string

	// New structured audit fields
	GetAction() AuditAction
	GetEntityType() EntityType
	GetIPAddress() string
	GetUserAgent() string
}

// BaseEvent provides common fields for all domain events
type BaseEvent struct {
	EventName    string
	Timestamp    time.Time
	EntityID     uint
	ActorUserID  uint
	AuditMessage string

	// New structured audit fields
	Action     AuditAction
	EntityType EntityType
	IPAddress  string
	UserAgent  string
}

func (e BaseEvent) EventType() string {
	return e.EventName
}

func (e BaseEvent) OccurredAt() time.Time {
	return e.Timestamp
}

func (e BaseEvent) AggregateID() uint {
	return e.EntityID
}

func (e BaseEvent) UserID() uint {
	return e.ActorUserID
}

func (e BaseEvent) GetAuditMessage() string {
	return e.AuditMessage
}

func (e BaseEvent) GetAction() AuditAction {
	return e.Action
}

func (e BaseEvent) GetEntityType() EntityType {
	return e.EntityType
}

func (e BaseEvent) GetIPAddress() string {
	return e.IPAddress
}

func (e BaseEvent) GetUserAgent() string {
	return e.UserAgent
}

// User Events

type UserCreatedEvent struct {
	BaseEvent
	Username string
	Fullname string
	Email    *string
	Role     UserRole
}

type UserUpdatedEvent struct {
	BaseEvent
	Username      string
	Fullname      string
	Email         *string
	Role          UserRole
	ChangedFields map[string]FieldChange
}

type UserDeletedEvent struct {
	BaseEvent
	Username string
	Fullname string
}

type PasswordChangedEvent struct {
	BaseEvent
	Method string // "self_service" or "admin_reset"
}

type UserLoginEvent struct {
	BaseEvent
	Username            string
	IPAddress           string
	UserAgent           string
	Success             bool
	Reason              string            // Only set for failed login
	AttemptedIdentifier string            // Value user typed in the username box
	Location            map[string]string // IP geolocation (country, city, region)
}

type UserLogoutEvent struct {
	BaseEvent
	Username string
}

// Employee Events

type EmployeeCreatedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeUpdatedEvent struct {
	BaseEvent
	Fullname      string
	CCCD          string
	ChangedFields map[string]FieldChange
}

type EmployeeDeletedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeProjectAssignmentsRemovedEvent struct {
	BaseEvent
	Fullname string
	CCCD     string
}

type EmployeeProfileUpdatedEvent struct {
	BaseEvent
	Fullname      string
	ChangedFields map[string]FieldChange
}

type EmployeeNameUpdatedEvent struct {
	BaseEvent
	OldFullName string
	NewFullName string
	CCCD        string
}

type ProjectEmployeesSyncedEvent struct {
	BaseEvent
	EmployeeID   uint
	OldFullName  string
	NewFullName  string
	UpdatedCount int
	SyncedAt     time.Time
}

// Project Events

type ProjectCreatedEvent struct {
	BaseEvent
	Name       string
	Code       string
	ClientName string
}

type ProjectUpdatedEvent struct {
	BaseEvent
	Name          string
	Code          string
	ClientName    string
	ChangedFields map[string]FieldChange
}

type ProjectDeletedEvent struct {
	BaseEvent
	Name string
	Code string
}

// ProjectEmployee Events

type ProjectEmployeeUpdatedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Status       string
}

// Timesheet Events

type TimesheetCreatedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	Date         time.Time
}

type TimesheetUpdatedEvent struct {
	BaseEvent
	EmployeeID    uint
	EmployeeName  string
	ProjectID     uint
	ProjectName   string
	ChangedFields map[string]FieldChange
}

type TimesheetDeletedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
}

type TimesheetApprovedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	ApproverID   uint
}

type TimesheetRejectedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	ApproverID   uint
}

type TimesheetBulkApprovedEvent struct {
	BaseEvent
	Count      int
	ProjectID  *uint
	ApproverID uint
}

type TimesheetBulkRejectedEvent struct {
	BaseEvent
	Count      int
	ProjectID  *uint
	ApproverID uint
}

type TimesheetBulkResetEvent struct {
	BaseEvent
	Count     int
	ProjectID *uint
	ResetBy   uint
}

type TimesheetBulkCreatedEvent struct {
	BaseEvent
	Count     int
	ProjectID *uint
	CreatedBy uint
}

type TimesheetBulkExternallyPaidEvent struct {
	BaseEvent
	Count     int
	Reference string
}

// Bank Events

type BankCreatedEvent struct {
	BaseEvent
	Name string
}

type BankUpdatedEvent struct {
	BaseEvent
	Name          string
	ChangedFields map[string]FieldChange
}

type BankDeletedEvent struct {
	BaseEvent
	Name string
}

// Payrate Events

type PayrateCreatedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
	Rate        int64
}

type PayrateUpdatedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
	Rate        int64
}

type PayrateDeletedEvent struct {
	BaseEvent
	ProjectID   uint
	ProjectName string
	EmployeeID  uint
}

// Transaction Events

type TransactionCreatedEvent struct {
	BaseEvent
	Amount       int64
	Type         string
	EmployeeID   uint
	EmployeeName string
	RelatedType  *string
	RelatedID    *uint
}

type TransactionUpdatedEvent struct {
	BaseEvent
	Amount        int64
	EmployeeID    uint
	EmployeeName  string
	ChangedFields map[string]FieldChange
}

type TransactionDeletedEvent struct {
	BaseEvent
	Amount       int64
	EmployeeName string
	EmployeeID   uint
}

type TransactionSettledEvent struct {
	BaseEvent
	Amount          int64
	SettlementID    uint
	TransactionDesc string
}

type SettlementCreatedEvent struct {
	BaseEvent
	TransactionID   uint
	Amount          int64
	SettlementUUID  *string
	TransactionCode *string
}

// Settlement Upload Events

type SettlementUploadProcessedEvent struct {
	BaseEvent
	AssetID          uint
	Filename         string
	TimesheetIDs     []uint
	SettlementAmount int64
	TransactionMap   map[uint][]uint // Transaction ID -> Timesheet IDs
	ProcessedBy      uint
}

type TimesheetMarkingEvent struct {
	BaseEvent
	TimesheetIDs  []uint
	MarkedAs      string // "paid" or similar
	Source        string // "manual_settlement", "upload_internal_sheet"
	SourceAssetID *uint  // Asset ID if from upload
}

type SettlementAppliedFromUploadEvent struct {
	BaseEvent
	TransactionID uint
	Amount        int64
	Source        string // "upload"
	FileID        uint
	Filename      string
	TimesheetIDs  []uint // timesheets to mark revenue_paid=1 atomically with settlement creation
}

type TimesheetsRevenuePaidFromInternalEvent struct {
	BaseEvent
	TimesheetIDs []uint
	FileID       uint
	Filename     string
}

// Loan Events

type LoanCreatedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	Amount       int64
}

type LoanUpdatedEvent struct {
	BaseEvent
	EmployeeID    uint
	EmployeeName  string
	Amount        int64
	ChangedFields map[string]FieldChange
}

type LoanDeletedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	Amount       int64
}

type LoanApprovedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	Amount       int64
}

type LoanRejectedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	Reason       string
}

type LoanDisbursedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	Amount       int64
}

// Asset Events

type AssetCreatedEvent struct {
	BaseEvent
	FileName string
}

type AssetDeletedEvent struct {
	BaseEvent
	FileName string
}

// Lender Events

type LenderCreatedEvent struct {
	BaseEvent
	Name string
}

type LenderUpdatedEvent struct {
	BaseEvent
	Name          string
	ChangedFields map[string]FieldChange
}

type LenderDeletedEvent struct {
	BaseEvent
	Name string
}

// Ledger Events

type LedgerEntryCreatedEvent struct {
	BaseEvent
	Account string
	Amount  int64
}

type LedgerEntryUpdatedEvent struct {
	BaseEvent
	Account string
	Amount  int64
}

type LedgerEntryDeletedEvent struct {
	BaseEvent
	Account string
}

// Settings Events

type SettingsCreatedEvent struct {
	BaseEvent
	Key string
}

type SettingsUpdatedEvent struct {
	BaseEvent
	Key string
}

type SettingsDeletedEvent struct {
	BaseEvent
	Key string
}

// Advance Payment Fee Schedule Events
//
// These describe per-entry CRUD on the JSON array stored under
// Settings.key = AdvancePaymentFeeScheduleSettingsKey. They sit alongside the
// SettingsUpdatedEvent that the Settings repo emits naturally — the schedule
// events carry the human-readable summary so admins see what *changed*, not
// just that the settings row was touched.
type AdvancePaymentFeeScheduleCreatedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string // YYYY-MM-DD
	Summary       string // e.g. "2% / 1.3% trên 3.500.000 VND"
}

type AdvancePaymentFeeScheduleUpdatedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string
	Summary       string
}

type AdvancePaymentFeeScheduleDeletedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string
	Summary       string
}

// Disbursement Fee Schedule Events
//
// Same shape as the advance-payment counterpart but for the per-transfer flat
// fee charged by the disbursement provider (9pay). The Summary field carries
// the human-readable fee amount in VND so admins see the rate change in audit
// without having to join other tables.
type DisbursementFeeScheduleCreatedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string // YYYY-MM-DD
	Summary       string // e.g. "200" (formatted VND amount)
}

type DisbursementFeeScheduleUpdatedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string
	Summary       string
}

type DisbursementFeeScheduleDeletedEvent struct {
	BaseEvent
	ScheduleID    string
	EffectiveDate string
	Summary       string
}

// Ad Banner Events — admin campaign lifecycle in the employee portal.

type AdBannerCreatedEvent struct {
	BaseEvent
	Title    string
	StartsAt time.Time
	EndsAt   time.Time
}

type AdBannerUpdatedEvent struct {
	BaseEvent
	Title    string
	StartsAt time.Time
	EndsAt   time.Time
}

type AdBannerDeletedEvent struct {
	BaseEvent
	Title string
}

// Import/Export Events

type DataImportedEvent struct {
	BaseEvent
	DataType    string
	RecordCount int
	FileName    string
}

type DataExportedEvent struct {
	BaseEvent
	DataType    string
	RecordCount int
	FileName    string
}

// Bulk Transfer Events

// BulkTransferFileExportedEvent is emitted when an admin exports a bulk-transfer
// bank file. Carries enough context for an audit log row to be useful without
// having to join other tables.
type BulkTransferFileExportedEvent struct {
	BaseEvent
	Filename             string
	Cycle                string // weekly / monthly / flexible
	FromDate             string // ISO date, empty if monthly
	ToDate               string // ISO date, empty if monthly
	ForMonth             string // YYYY-MM, empty if weekly
	TransactionsCount    int
	TotalAmount          int64
	CompanyWide          bool // false when project/employee filters scoped the export
	ForecastOutcomeItems []CashForecastOutcomeItem
}

// BulkTransferResultImportedEvent is emitted when an admin imports a bulk
// transfer result file (top-level IMPORT audit, distinct from the per-row
// downstream events).
type BulkTransferResultImportedEvent struct {
	BaseEvent
	AssetID        uint
	Filename       string
	TotalTransfers int
	CompletedCount int
	FailedCount    int
	TotalAmount    int64
	IsDuplicate    bool // true when the same file was uploaded before; downstream payment/txn workers do NOT re-run
}

// BulkTransferFileDownloadedEvent is emitted when an admin re-downloads a
// previously-generated bulk-transfer file. Sensitive enough to audit.
type BulkTransferFileDownloadedEvent struct {
	BaseEvent
	BulkTransferFileID uint
	AssetID            uint
	Filename           string
}

// PayrollHistoriesExportedEvent is emitted when an admin exports payroll
// histories to Excel. Crosses employee boundaries — audit it.
type PayrollHistoriesExportedEvent struct {
	BaseEvent
	FromDate string
	ToDate   string
	Filename string
}

type BulkTransferResultParsedEvent struct {
	BaseEvent
	BulkFileID      uint
	AssetID         uint
	Filename        string
	TotalTransfers  int
	CompletedCount  int
	FailedCount     int
	TotalAmount     int64
	ParsedDataJSON  string // JSON string of parsed results
	UpdatedDataJSON string // JSON string of updated bulk transfer data
	ProcessedBy     uint
	EventID         string // UUID of the outbox event for idempotency
}

type BulkTransferPaymentStatusUpdatedEvent struct {
	BaseEvent
	BulkFileID    uint
	TimesheetIDs  []uint
	PaymentStatus string
	TotalUpdated  int
}

type BulkTransferTransactionCreatedEvent struct {
	BaseEvent
	BulkFileID    uint
	TransactionID uint
	Amount        int64
	AssetID       uint
	Filename      string
}

// Timesheet Edit Request Events

type TimesheetEditRequestCreatedEvent struct {
	BaseEvent
	TimesheetID uint
	RequestedBy uint
	Reason      string
}

type TimesheetEditRequestUpdatedEvent struct {
	BaseEvent
	RequestID   uint
	TimesheetID uint
	RequestedBy uint
	NewStatus   string
	Reason      string
}

// ProjectEmployee Events

type ProjectEmployeeCreatedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Role         string
}

type ProjectEmployeeDeletedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Reason       string
}

// Permission Events

type EmployeeAccessGrantedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	Role         string
	GrantedBy    uint
}

type EmployeeAccessRevokedEvent struct {
	BaseEvent
	EmployeeID   uint
	EmployeeName string
	ProjectID    uint
	ProjectName  string
	RevokedBy    uint
}

type ProjectAccessGrantedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	Role         string
	GrantedBy    uint
}

type ProjectAccessRevokedEvent struct {
	BaseEvent
	ProjectID    uint
	ProjectName  string
	EmployeeID   uint
	EmployeeName string
	RevokedBy    uint
}

// EventBus defines the interface for publishing and subscribing to domain events
type EventBus interface {
	Publish(ctx context.Context, events ...DomainEvent) error
	Subscribe(eventType string, handler EventHandler)
	SubscribeAll(handler EventHandler)
}

// EventHandler handles domain events
type EventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
	CanHandle(eventType string) bool
}
