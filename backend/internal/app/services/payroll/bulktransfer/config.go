package bulktransfer

import (
	"log/slog"

	"api-server/internal/app/services/payroll/excel"
	"api-server/internal/app/services/payroll/pdf"
	"api-server/internal/domain"

	"gorm.io/gorm"
)

// BulkTransferTaskPayload carries all fields needed to enqueue or process a
// bulk-transfer result task. Defined here (not in the asynq package) so that
// bulktransfer can reference it without importing asynq and creating a cycle.
type BulkTransferTaskPayload struct {
	EventID         string `json:"event_id"`
	AssetID         uint   `json:"asset_id"`
	Filename        string `json:"filename"`
	BulkFileID      uint   `json:"bulk_file_id"`
	TotalTransfers  int    `json:"total_transfers"`
	CompletedCount  int    `json:"completed_count"`
	FailedCount     int    `json:"failed_count"`
	TotalAmount     int64  `json:"total_amount"`
	ParsedDataJSON  string `json:"parsed_data_json"`
	UpdatedDataJSON string `json:"updated_data_json"`
	ProcessedBy     uint   `json:"processed_by"`
	ActorUserID     uint   `json:"actor_user_id"`
}

// BulkTransferEnqueuer abstracts asynq task enqueueing for bulk transfer workers,
// allowing ResultProcessor to schedule work without importing the asynq package.
type BulkTransferEnqueuer interface {
	EnqueueBulkTransferTransaction(payload BulkTransferTaskPayload) error
	EnqueueBulkTransferPayment(payload BulkTransferTaskPayload) error
}

// Config consolidates all dependencies for BulkTransferService
// This reduces constructor complexity from 18 parameters to 1
type Config struct {
	// Repositories (8)
	TimesheetRepo       TimesheetRepository
	EmployeeRepo        EmployeeRepository
	EmployeeUserRepo    EmployeeUserRepository
	ProjectRepo         ProjectRepository
	ProjectEmployeeRepo ProjectEmployeeRepository
	UserRepo            UserRepository
	FileRepo            BulkTransferFileRepository
	TransactionCodeRepo TransactionCodeRepository

	// External Services (8)
	LedgerService      LedgerService
	TransactionService TransactionService
	AssetService       AssetService
	AssetRepository    domain.AssetRepository
	ExcelConverter     ExcelConverter
	ExcelService       *excel.Service
	SettingsConfig     SettingsConfigService
	PDFService         *pdf.Service
	Notifier           Notifier

	// Infrastructure
	DB          *gorm.DB
	EventBus    domain.EventBus
	AsynqClient BulkTransferEnqueuer
	Logger      *slog.Logger
}
