package repositories

import (
	"fmt"

	"api-server/internal/domain"
	domaintx "api-server/internal/domain/transactions"
	"api-server/internal/domain/wallet"
	"api-server/internal/infra/persistence"
)

type Repositories struct {
	User                    domain.UserRepository
	BlacklistedToken        domain.BlacklistedTokenRepository
	AuditLog                domain.AuditLogRepository
	Project                 domain.ProjectRepository
	ProjectUser             domain.ProjectUserRepository
	Employee                domain.EmployeeRepository
	EmployeeUser            domain.EmployeeUserRepository
	ProjectEmployee         domain.ProjectEmployeeRepository
	Bank                    domain.BankRepository
	Payrate                 domain.PayrateRepository
	Timesheet               domain.TimesheetRepository
	TimesheetEditRequest    domain.TimesheetEditRequestRepository
	Ledger                  domain.LedgerEntryRepository
	Transaction             domain.TransactionRepository
	Settlement              domain.SettlementRepository
	Notification            domain.NotificationRepository
	Settings                domain.SettingsRepository
	Asset                   domain.AssetRepository
	BulkTransferFile        domain.BulkTransferFileRepository
	Lender                  domain.LenderRepository
	Loan                    domain.LoanRepository
	LoanRepaymentSchedule   domain.LoanRepaymentScheduleRepository
	APIMetric               domain.APIMetricRepository
	AdvancePayment          domain.AdvancePaymentRepository
	AdvancePaymentRequest   domain.AdvancePaymentRequestRepository
	TransactionCode         domain.TransactionCodeRepository
	PushSubscription        domain.PushSubscriptionRepository
	CronJobStatus           *persistence.CronJobStatusRepository
	TxWalletPayment         domaintx.WalletPaymentRepository
	WalletTopup             wallet.WalletTopupRepository
	WalletPayment           wallet.WalletPaymentRepository
	WalletIPN               wallet.WalletIPNRepository
	Attendance              domain.AttendanceRepository
	AttendanceFailedAttempt domain.AttendanceFailedAttemptRepository
	SettlementUpload        domain.SettlementUploadRepository
}

func Initialize(db *persistence.Database, eventBus domain.EventBus) *Repositories {
	// Create TransactionCode repo first (needed by BulkTransferFile)
	transactionCodeRepo := persistence.NewTransactionCodeRepository(db)

	repos := &Repositories{
		User:                    persistence.NewUserRepository(db),
		BlacklistedToken:        persistence.NewBlacklistedTokenRepository(db),
		AuditLog:                persistence.NewAuditLogRepository(db),
		Project:                 persistence.NewProjectRepository(db),
		ProjectUser:             persistence.NewProjectUserRepository(db),
		Employee:                persistence.NewEmployeeRepository(db),
		EmployeeUser:            persistence.NewEmployeeUserRepository(db),
		ProjectEmployee:         persistence.NewProjectEmployeeRepository(db),
		Bank:                    persistence.NewBankRepository(db),
		Payrate:                 persistence.NewPayrateRepository(db),
		TimesheetEditRequest:    persistence.NewTimesheetEditRequestRepository(db),
		Ledger:                  persistence.NewLedgerEntryRepository(db),
		Transaction:             persistence.NewTransactionRepository(db.DB),
		Settlement:              persistence.NewSettlementRepository(db.DB),
		Notification:            persistence.NewNotificationRepository(db),
		Settings:                persistence.NewSettingsRepository(db),
		Asset:                   persistence.NewAssetRepository(db, eventBus),
		BulkTransferFile:        persistence.NewBulkTransferFileRepository(db.DB, transactionCodeRepo),
		Lender:                  persistence.NewLenderRepository(db),
		Loan:                    persistence.NewLoanRepository(db),
		LoanRepaymentSchedule:   persistence.NewLoanRepaymentScheduleRepository(db),
		APIMetric:               persistence.NewAPIMetricRepository(db),
		AdvancePayment:          persistence.NewAdvancePaymentRepository(db),
		AdvancePaymentRequest:   persistence.NewAdvancePaymentRequestRepository(db),
		TransactionCode:         transactionCodeRepo,
		PushSubscription:        persistence.NewPushSubscriptionRepository(db.DB),
		CronJobStatus:           persistence.NewCronJobStatusRepository(db),
		TxWalletPayment:         persistence.NewTxWalletPaymentRepository(db),
		Attendance:              persistence.NewAttendanceRepository(db.DB),
		AttendanceFailedAttempt: persistence.NewAttendanceFailedAttemptRepository(db.DB),
		SettlementUpload:        persistence.NewSettlementUploadRepository(db),
	}

	// Wallet repositories need *sql.DB for raw SQL queries
	sqlDB, err := db.DB.DB()
	if err != nil {
		panic(fmt.Sprintf("failed to get underlying sql.DB: %v", err))
	}
	repos.WalletTopup = persistence.NewWalletTopupRepository(sqlDB)
	repos.WalletPayment = persistence.NewWalletPaymentRepository(sqlDB)
	repos.WalletIPN = persistence.NewWalletIPNRepository(db)

	repos.Timesheet = persistence.NewTimesheetRepository(db, repos.ProjectEmployee)

	return repos
}
