package services

import (
	"context"
	"io"
	"log/slog"
	"time"

	bootstrapRepos "api-server/internal/app/bootstrap/repositories"
	"api-server/internal/app/services"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/asset"
	"api-server/internal/app/services/attendance"
	"api-server/internal/app/services/auth"
	"api-server/internal/app/services/cleanup"
	"api-server/internal/app/services/config"
	"api-server/internal/app/services/dashboard"
	"api-server/internal/app/services/disbursement"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/flex_pay"
	infraServices "api-server/internal/app/services/infrastructure"
	"api-server/internal/app/services/ledger"
	"api-server/internal/app/services/loan"
	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/otp"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/payroll/bulktransfer"
	"api-server/internal/app/services/project"
	pushService "api-server/internal/app/services/push"
	"api-server/internal/app/services/reporting"
	"api-server/internal/app/services/settlement"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/app/services/user"
	"api-server/internal/app/workers"
	appConfig "api-server/internal/config"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/domain/wallet"
	"api-server/internal/infra/cache"
	"api-server/internal/infra/disbursement/ninepay"
	"api-server/internal/infra/disbursement/onepay"
	"api-server/internal/infra/email"
	"api-server/internal/infra/events"
	"api-server/internal/infra/persistence"
	"api-server/internal/infra/persistence/repositories"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"

	// Adapter imports
	cacheAdapter "api-server/internal/adapters/cache"
	notificationAdapter "api-server/internal/adapters/notification"
	serviceAdapters "api-server/internal/adapters/services"
	serviceports "api-server/internal/domain/ports/services"
	asynqinfra "api-server/internal/infra/asynq"
)

type Services struct {
	User                              *user.UserService
	PasswordResetJobManager           *user.PasswordResetJobManager
	Auth                              *auth.AuthService
	Authorization                     *auth.AuthorizationService
	Dashboard                         *dashboard.Service
	Project                           *project.ProjectService
	ProjectPermission                 *project.ProjectPermissionService
	Employee                          *employee.EmployeeService
	EmployeePermission                *employee.EmployeePermissionService
	EmployeeUser                      *employee.EmployeeUserService
	EmployeeProfile                   *employee.EmployeeProfileService
	ProjectEmployee                   *project.ProjectEmployeeService
	Bank                              *asset.BankService
	Payrate                           *payroll.PayrateService
	Timesheet                         *timesheet.TimesheetService
	TimesheetEditRequest              *timesheet.TimesheetEditRequestService
	PayrollReport                     *payroll.PayrollReportService
	Payroll                           *payroll.PayrollService
	PayrollReportExporter             *payroll.PayrollReportExporter
	PayrollReportByProjectService     *payroll.PayrollReportByProjectService
	PayrollReportByProjectExporter    *payroll.PayrollReportByProjectExporter
	Ledger                            *settlement.LedgerService
	Transaction                       *settlement.TransactionService
	TransactionPort                   serviceports.TransactionPort
	SettlementUpload                  *settlement.SettlementUploadService
	Notification                      *notification.NotificationService
	Settings                          *config.SettingsService
	SettingsConfig                    *config.SettingsConfigService
	ExcelConverter                    *reporting.ExcelConverterService
	Asset                             *asset.AssetService
	Cache                             domain.CacheServiceUseCase
	CacheService                      *infraServices.CacheService
	Lender                            *loan.LenderService
	Loan                              *loan.LoanService
	Email                             *notification.EmailService
	Metric                            *infraServices.MetricService
	Audit                             *infraServices.AuditService
	APIMetricCleanupService           *cleanup.APIMetricCleanupService
	Idempotency                       *infraServices.IdempotencyService
	Reconcile                         *ledger.ReconcileService
	OnePayFeeImport                   *settlement.OnePayFeeImportService
	AdvancePayment                    *advance_payment.Service
	AdvancePaymentFeeSchedule         *advance_payment.FeeScheduleService
	ImportProgress                    *advance_payment.ImportProgressService
	EmployeeImport                    *employee.ImportService
	EmployeeImportProgress            *infraServices.EmployeeImportProgressService
	FlexPayReconciliationService      *domainServices.FlexPayReconciliationService
	FlexPayReconciliationExporter     *flex_pay.FlexPayReconciliationExporter
	FlexPaySettlementService          *flex_pay.FlexPaySettlementService
	BulkTransferTransactionWorker     *workers.BulkTransferTransactionWorker
	BulkTransferPaymentWorkerConcrete *workers.BulkTransferPaymentWorker
	Push                              *pushService.PushService
	DisbursementRegistry              *disbursement.Registry
	DisbursementFeeSchedule           *disbursement.FeeScheduleService
	ProviderTransactions              *disbursement.WalletPaymentService
	WalletPaymentStats                *disbursement.StatsService
	Wallet                            wallet.WalletService
	WalletDemandForecast              *services.WalletDemandForecastService
	NinepayCloser                     io.Closer
	NinePayBulkTransfer               *bulktransfer.NinePayBulkTransferService
	NinePayBatchCompletion            *bulktransfer.NinePayBatchCompletionChecker
	AutoBulkTransfer                  bulktransfer.AutoBulkTransferService
	BulkTransferPayment               workers.TransferTimesheetUpdater
	BCCImport                         *services.BCCImportService
	Attendance                        *attendance.AttendanceService
}

func Initialize(repos *bootstrapRepos.Repositories, cfg *appConfig.Config, logger *slog.Logger, db *persistence.Database, redis *persistence.RedisClient, eventBus domain.EventBus, asynqClient *asynqinfra.Client, clk clock.Clock) *Services {
	// Step 1: Initialize concrete infrastructure services first
	cacheService := infraServices.NewCacheService(redis)

	// Step 2: Create infrastructure adapters (ports)
	cachePort := cacheAdapter.NewRedisAdapter(cacheService)

	// Step 3: Create base services needed for other services
	userService := user.NewUserService(repos.User, repos.AuditLog, eventBus, cfg.Security.HashSecret, cfg.Security.HashSalt)
	passwordResetJobManager := user.NewPasswordResetJobManager(userService, cacheService)
	pushSvc := pushService.NewPushService(repos.PushSubscription, cfg.Notification, logger)
	notificationService := notification.NewNotificationService(repos.Notification, repos.User, repos.Project, userService, pushSvc, logger)
	employeeNotificationService := notification.NewEmployeeNotificationService(notificationService, repos.Employee, logger)
	settingsService := config.NewSettingsService(repos.Settings, cacheService, eventBus)
	settingsConfigService := config.NewSettingsConfigService(settingsService)
	excelConverterService := reporting.NewExcelConverterService()
	pdfService := reporting.NewPDFService("fonts/Roboto-Regular.ttf")
	fileStorage := storage.NewLocalFileStorage(cfg.Asset.StoragePath, cfg.Asset.BaseURL)
	assetService := asset.NewAssetService(repos.Asset, fileStorage, cfg.Asset.MaxFileSize, eventBus)
	transactionManager := infraServices.NewTransactionManager(db.DB)

	// Step 4: Create domain services using infrastructure ports
	ledgerService := settlement.NewLedgerService(repos.Ledger, eventBus, cachePort)
	timesheetService := timesheet.NewTimesheetService(repos.Timesheet, repos.Employee, repos.Project, repos.ProjectEmployee, repos.Payrate, transactionManager, cachePort, eventBus)

	// Create email service for notification adapter (needed before adapter creation)
	emailNotificationPublisher := notification.NewEmailNotificationPublisher(repos.Notification, logger)
	payCycleNotificationPublisher := notification.NewPaymentCycleNotificationPublisher(repos.Notification, logger)
	payrollReportByProjectService := payroll.NewPayrollReportByProjectService(repos.Timesheet, repos.Project, repos.Employee)
	payrollReportByProjectExporter := payroll.NewPayrollReportByProjectExporter()
	payrollReportAdapter := notification.NewPayrollReportAdapter(payrollReportByProjectService, payrollReportByProjectExporter)

	// Select email provider based on environment
	var emailProvider domain.EmailDeliveryPort
	if cfg.App.Env != "production" {
		emailProvider = email.NewSandboxProvider(logger)
	} else {
		emailProvider = email.NewResendProvider(cfg.Notification.ResendAPIKey)
	}
	emailService := notification.NewEmailService(cfg.Notification, emailProvider, payrollReportAdapter, repos.Notification, repos.User, emailNotificationPublisher, assetService, logger)

	// Create notification adapter port
	notificationPort := notificationAdapter.NewEmailAdapter(notificationService, emailService)

	// Create timesheet edit request service with ports
	timesheetEditRequestService := timesheet.NewTimesheetEditRequestService(repos.TimesheetEditRequest, repos.Timesheet, repos.AuditLog, eventBus, transactionManager)

	// Create transaction service with ports
	transactionService := settlement.NewTransactionService(db.DB, repos.Transaction, repos.Settlement, repos.Ledger, repos.Loan, repos.User, repos.Timesheet, cachePort, eventBus)
	transactionPort := serviceAdapters.NewTransactionAdapter(transactionService)

	// Create salary calculation service
	salaryCalculationService := domainServices.NewSalaryCalculationService(repos.BulkTransferFile)

	// Create query repositories
	timesheetQueryRepo := repositories.NewTimesheetQueryRepository(db.DB)
	timesheetAnalyticsRepo := repositories.NewTimesheetAnalyticsRepository(db.DB)

	dashboardService := dashboard.NewService(repos.User, repos.Project, repos.Employee, repos.Timesheet, timesheetQueryRepo, timesheetAnalyticsRepo, repos.Ledger, repos.Notification, repos.AuditLog, repos.ProjectEmployee, repos.BulkTransferFile, repos.AdvancePaymentRequest, repos.AdvancePayment, repos.Attendance, repos.AttendanceFailedAttempt, settingsConfigService, cacheService, salaryCalculationService, logger)
	// Create remaining services
	settlementUploadService := settlement.NewSettlementUploadService(db.DB, repos.Timesheet, repos.Transaction, assetService, repos.Ledger, eventBus, repos.AdvancePaymentRequest, repos.AdvancePayment, repos.Notification)
	projectPermissionService := project.NewProjectPermissionService(repos.Project, repos.ProjectUser, repos.ProjectEmployee, repos.EmployeeUser, repos.User, eventBus)
	employeePermissionService := employee.NewEmployeePermissionService(repos.Employee, repos.EmployeeUser, repos.ProjectEmployee, repos.User, eventBus)
	payrollReportService := payroll.NewPayrollReportService(repos.Timesheet, repos.Employee, repos.ProjectEmployee, repos.Project, repos.BulkTransferFile)
	payrollReportExporter := payroll.NewPayrollReportExporter(settingsConfigService)
	authorizationService, err := auth.NewAuthorizationService(
		auth.GetDefaultModelPath(),
		auth.GetDefaultPolicyPath(),
		timesheetService,
		logger,
	)
	if err != nil {
		logger.Error("Failed to initialize authorization service", "error", err)
		authorizationService = nil
	}

	employeeUserService := employee.NewEmployeeUserService(repos.Employee, userService)
	employeeProfileService := employee.NewEmployeeProfileService(repos.Employee, repos.Timesheet, repos.ProjectEmployee, repos.Payrate, userService, eventBus)
	lenderService := loan.NewLenderService(repos.Lender, cacheService, eventBus)
	attendanceService := attendance.NewAttendanceService(repos.Attendance, repos.ProjectEmployee, repos.Project, repos.Payrate, repos.AdvancePayment, transactionManager, asynqClient, clk)

	// Geofence gates are now DB-backed (project.geofence_gates JSON column).
	// No in-memory registration needed.

	// Register global event handlers after all core services are constructed.
	auditEventHandler := events.NewAuditEventHandler(asynqClient)
	eventBus.SubscribeAll(auditEventHandler)

	// Register cache invalidation handler for async cache updates
	cacheInvalidationHandler := events.NewCacheInvalidationHandler(cacheService)
	eventBus.SubscribeAll(cacheInvalidationHandler)

	// Register employee user creation handler to decouple user creation from HTTP request path
	employeeUserCreatedHandler := events.NewEmployeeUserCreatedHandler(repos.Employee, employeeUserService)
	eventBus.SubscribeAll(employeeUserCreatedHandler)

	// Register settlement event handler for transaction and timesheet marking operations
	settlementEventHandler := events.NewSettlementEventHandler(
		db.DB,
		repos.Timesheet,
		repos.Transaction,
		repos.Settlement,
		repos.Ledger,
		transactionPort,
		eventBus,
		employeeNotificationService,
	)
	eventBus.SubscribeAll(settlementEventHandler)

	// Wire the settlement event handler as the synchronous settlement applier so the
	// upload flow settles each transaction inline (atomic + self-verifying) instead of
	// via a fire-and-forget event that can silently drop (txn 99 / timesheet 11579 bug).
	settlementUploadService.SetSettlementApplier(settlementEventHandler)

	logger.Info("Event bus initialized with handlers",
		"audit", true,
		"cache_invalidation", true,
		"employee_user_created", true,
		"settlement", true,
	)

	// Step 6: Create services that depend on other domain services using ports
	loanService := loan.NewLoanService(repos.Loan, repos.Lender, repos.Ledger, repos.Transaction, transactionPort, transactionManager, cachePort, eventBus, notificationPort, repos.LoanRepaymentSchedule)

	// Initialize FlexPay reconciliation services
	flexPayReconciliationService := domainServices.NewFlexPayReconciliationService(
		repos.AdvancePaymentRequest,
		repos.AdvancePayment,
		repos.Project,
		repos.Employee,
	)
	apiMetricCleanupService := cleanup.NewAPIMetricCleanupService(repos.APIMetric, logger)
	reconcileService := ledger.NewReconcileService(db.DB, repos.Ledger, repos.Notification)

	// Initialize import progress service
	importProgressService := advance_payment.NewImportProgressService(cacheService)

	// Fee schedule service drives all advance-payment fee resolution. Created
	// before the advance-payment service so we can inject its FeeResolver, and
	// bound back into settingsConfigService so legacy callers (bulk transfer
	// worker, payroll exporter, headline-rate UI) read the active schedule
	// instead of the retired flat-rate settings keys.
	feeScheduleService := advance_payment.NewFeeScheduleService(db.DB, cacheService, eventBus, logger)
	settingsConfigService.BindFeeScheduleResolver(feeScheduleService)

	// Pre-declare disbursement registry so the advance payment config closure
	// can reference it. The actual NewRegistry() + provider registration happens
	// later in this function — the closure only reads the variable at call-time.
	var disbursementRegistry *disbursement.Registry

	// Initialize advance payment service
	advancePaymentConfig := &advance_payment.Config{
		AdvancePaymentRepo:          repos.AdvancePayment,
		AdvancePaymentRequestRepo:   repos.AdvancePaymentRequest,
		TransactionCodeRepo:         repos.TransactionCode,
		TransactionRepo:             repos.Transaction,
		EmployeeRepo:                repos.Employee,
		ProjectEmployeeRepo:         repos.ProjectEmployee,
		ProjectRepo:                 repos.Project,
		BankRepo:                    repos.Bank,
		BulkTransferFileRepo:        repos.BulkTransferFile,
		AssetRepo:                   repos.Asset,
		CacheService:                cacheService,
		EmployeeUserService:         employeeUserService,
		LedgerRepo:                  repos.Ledger,
		EmployeeNotifier:            employeeNotificationService,
		FeeResolver:                 feeScheduleService,
		GetAdvancePaymentPercentage: settingsConfigService.GetAdvancePaymentPercentage,
		GetAdvancePaymentFeeMin:     settingsConfigService.GetAdvancePaymentFeeMin,
		GetAdvanceCashFeePercentage: settingsConfigService.GetAdvanceCashFeePercentage,
		GetPartnerCompany:           settingsConfigService.GetPartnerCompany,
		GetTransferLimits: func(ctx context.Context) advance_payment.ProviderTransferLimits {
			limits := disbursementRegistry.ActiveTransferLimits(ctx)
			return advance_payment.ProviderTransferLimits{
				MinAmount: limits.MinAmount,
				MaxAmount: limits.MaxAmount,
			}
		},
	}
	// Create audit service
	auditService := infraServices.NewAuditService(eventBus)
	onePayFeeImportService := settlement.NewOnePayFeeImportService(
		repos.TxWalletPayment,
		repos.Bank,
		repos.Transaction,
		transactionService,
		auditService,
		logger,
	)

	advancePaymentService := advance_payment.NewService(advancePaymentConfig, logger)

	// Initialize FlexPay reconciliation exporter
	flexPayReconciliationExporter := flex_pay.NewFlexPayReconciliationExporter()

	// Initialize FlexPay settlement service
	flexPaySettlementService := flex_pay.NewFlexPaySettlementService(
		db.DB,
		repos.AdvancePaymentRequest,
		repos.AdvancePayment,
		repos.Ledger,
		repos.Transaction,
		repos.SettlementUpload,
		settingsConfigService.GetPartnerCompany,
	)

	// Initialize employee import progress service
	employeeImportProgressService := infraServices.NewEmployeeImportProgressService(cacheService)

	idempotencyService := infraServices.NewIdempotencyService(redis.Client, logger)

	// Disbursement provider registry. Registration is gated at boot by
	// the master env flag (ENABLE_NINEPAY) AND the presence of provider
	// credentials. When the flag is off or creds are blank, the registry
	// stays empty and Active(ctx) returns ErrNoActiveProvider — use cases
	// then fail closed without ever touching the provider.
	disbursementRegistry = disbursement.NewRegistry()
	var ninepayCloser io.Closer
	switch {
	case !cfg.Disbursement.Ninepay.Enabled:
		logger.Info("disbursement: ENABLE_NINEPAY=false; 9pay provider not registered")
	case cfg.Disbursement.Ninepay.MerchantKey == "":
		logger.Info("disbursement: 9pay credentials not configured; provider not registered")
	default:
		// Resolve the merchant-portal device_id: env override > Settings
		// table > auto-generate-and-persist. Done at boot rather than per
		// request so the value is fixed for the lifetime of the process
		// and the Settings round-trip happens once.
		deviceID, devErr := resolveNinepayDeviceID(context.Background(), settingsService, cfg.Disbursement.Ninepay.WebDeviceID, logger)
		if devErr != nil {
			logger.Error("disbursement: resolve 9pay device_id failed; provider will not be registered", "error", devErr)
			break
		}
		ninepayClient, err := ninepay.NewClient(ninepay.Config{
			MerchantKey:       cfg.Disbursement.Ninepay.MerchantKey,
			SecretKey:         cfg.Disbursement.Ninepay.SecretKey,
			SecretKeyChecksum: cfg.Disbursement.Ninepay.SecretKeyChecksum,
			Endpoint:          cfg.Disbursement.Ninepay.Endpoint,
			WebEndpoint:       cfg.Disbursement.Ninepay.WebEndpoint,
			Portal: ninepay.PortalAuthConfig{
				AccountURL:  cfg.Disbursement.Ninepay.WebAccountURL,
				WebEndpoint: cfg.Disbursement.Ninepay.WebEndpoint,
				Username:    cfg.Disbursement.Ninepay.WebUsername,
				Password:    cfg.Disbursement.Ninepay.WebPassword,
				DeviceID:    deviceID,
				ClientID:    cfg.Disbursement.Ninepay.WebClientID,
				RedirectURI: cfg.Disbursement.Ninepay.WebRedirectURI,
			},
			WebMerchantID: cfg.Disbursement.Ninepay.WebMerchantID,
		}, logger)
		if err != nil {
			logger.Error("disbursement: 9pay client init failed; provider will not be registered",
				"error", err)
		} else {
			provider := ninepay.NewProvider(ninepayClient, logger)
			queued := ninepay.NewQueuedProvider(provider, cfg.Disbursement.Ninepay.TPS, cfg.Disbursement.Ninepay.QueueBuffer, logger)
			disbursementRegistry.Register(queued)
			ninepayCloser = queued
			logger.Info("disbursement: 9pay provider registered",
				"endpoint", cfg.Disbursement.Ninepay.Endpoint,
				"enabled_for_employee", cfg.Disbursement.Ninepay.EnabledForEmployee,
				"tps", cfg.Disbursement.Ninepay.TPS)
		}
	}

	// ---- OnePay ---------------------------------------------------------
	switch {
	case !cfg.Disbursement.Onepay.Enabled:
		logger.Info("disbursement: ENABLE_ONEPAY=false; onepay provider not registered")
	case cfg.Disbursement.Onepay.PartnerID == "" || cfg.Disbursement.Onepay.PartnerKey == "" || cfg.Disbursement.Onepay.AccountID == "":
		logger.Warn("disbursement: onepay disabled (creds missing)",
			"partner_id_set", cfg.Disbursement.Onepay.PartnerID != "",
			"partner_key_set", cfg.Disbursement.Onepay.PartnerKey != "",
			"account_id_set", cfg.Disbursement.Onepay.AccountID != "",
		)
	default:
		opClient, err := onepay.NewClient(onepay.Config{
			PartnerID:            cfg.Disbursement.Onepay.PartnerID,
			PartnerKey:           cfg.Disbursement.Onepay.PartnerKey,
			AccountID:            cfg.Disbursement.Onepay.AccountID,
			Endpoint:             cfg.Disbursement.Onepay.Endpoint,
			RequestExpirySeconds: cfg.Disbursement.Onepay.RequestExpirySeconds,
			HTTPTimeout:          cfg.Disbursement.Onepay.HTTPTimeout,
			TPS:                  cfg.Disbursement.Onepay.TPS,
			QueueBuffer:          cfg.Disbursement.Onepay.QueueBuffer,
		}, logger.With("provider", "1pay"))
		if err != nil {
			logger.Error("disbursement: onepay client init failed; provider will not be registered",
				"error", err)
		} else {
			opProvider := onepay.NewProvider(opClient, logger.With("provider", "1pay"))
			queued := onepay.NewQueuedProvider(opProvider, cfg.Disbursement.Onepay.TPS, cfg.Disbursement.Onepay.QueueBuffer, logger)
			disbursementRegistry.Register(queued)
			logger.Info("disbursement: onepay provider registered",
				"endpoint", cfg.Disbursement.Onepay.Endpoint,
				"for_employee", cfg.Disbursement.Onepay.EnabledForEmployee,
				"tps", cfg.Disbursement.Onepay.TPS)
		}
	}

	// Attach the selector chain: context hint (admin header) → feature
	// flags (employee routing) → first-registered fallback.
	sel := disbursement.ChainSelector{
		Selectors: []disbursement.Selector{
			disbursement.ContextSelector{},
			disbursement.FlagSelector{
				OnepayForEmployee:      cfg.Disbursement.Onepay.EnabledForEmployee,
				NinepayForEmployee:     cfg.Disbursement.Ninepay.EnabledForEmployee,
				OnepayForBulkTransfer:  cfg.Disbursement.Onepay.EnabledForBulkTransfer,
				NinepayForBulkTransfer: cfg.Disbursement.Ninepay.EnabledForBulkTransfer,
			},
		},
	}
	disbursementRegistry.WithSelector(&sel)

	// Disbursement fee schedule service: dated fee history mirroring the
	// advance-payment fee schedule pattern. The provider transaction service
	// stamps fee at INSERT time by resolving the active entry at clock.Now().
	disbursementFeeScheduleService := disbursement.NewFeeScheduleService(db.DB, eventBus, logger)
	disbursementFeeScheduleService.SetRegistry(disbursementRegistry)

	// provider_transactions service stack: persistence + state-machine
	// driver + read-only stats. Wired even when 9pay is dormant — the
	// table and stats endpoint exist regardless, and other providers
	// (future) will reuse the same service.
	//
	// Error translation is resolved at call time against the active
	// provider's ErrorTranslator capability, so the service stays
	// decoupled from any concrete provider package.
	disbursementErrorTranslator := disbursementRegistry.ActiveErrorTranslator()
	walletPaymentService := disbursement.NewWalletPaymentService(
		repos.TxWalletPayment,
		repos.AdvancePaymentRequest,
		repos.Employee,
		repos.Bank,
		notificationService,
		disbursementFeeScheduleService,
		disbursementErrorTranslator,
		disbursementRegistry,
		logger,
	)
	walletPaymentStats := disbursement.NewStatsService(repos.TxWalletPayment, disbursementErrorTranslator)

	// Initialize bulk transfer payment worker for per-transfer timesheet updates
	bulkTransferPaymentWorker := workers.NewBulkTransferPaymentWorker(
		db.DB,
		repos.Timesheet,
		repos.ProjectEmployee,
		repos.TransactionCode,
		settingsConfigService,
		idempotencyService,
		notificationService,
		employeeNotificationService,
		cachePort,
	)

	// Initialize bulk transfer transaction worker for async transaction creation
	bulkTransferTransactionWorker := workers.NewBulkTransferTransactionWorker(
		db.DB,
		repos.Transaction,
		repos.Timesheet,
		repos.Asset,
		ledgerService,
		transactionService,
		settingsConfigService,
		idempotencyService,
		eventBus,
	)

	// Create employee service config (11 params → 1)
	employeeConfig := &employee.Config{
		EmployeeRepo:          repos.Employee,
		TimesheetRepo:         repos.Timesheet,
		BankRepo:              repos.Bank,
		ProjectEmployeeRepo:   repos.ProjectEmployee,
		EmployeeUserRepo:      repos.EmployeeUser,
		UserRepo:              repos.User,
		EmployeeDomainService: domainServices.NewEmployeeDomainService(repos.Employee, repos.ProjectEmployee, repos.Timesheet),
		UserService:           userService,
		TransactionManager:    transactionManager,
		Events:                eventBus,
		Cache:                 cacheService,
		Logger:                logger,
	}

	walletService := services.NewWalletService(repos.WalletTopup, repos.WalletPayment, disbursementRegistry)
	walletDemandForecastService := services.NewWalletDemandForecastService(repos.AdvancePaymentRequest, walletService, clk, cfg.WalletForecast)

	// Create payroll service (which contains the bulk transfer module)
	payrollSvc := payroll.NewPayrollService(db.DB, repos.Timesheet, repos.Employee, repos.EmployeeUser, repos.Project, repos.ProjectEmployee, repos.User, ledgerService, transactionService, assetService, excelConverterService, settingsConfigService, repos.BulkTransferFile, repos.TransactionCode, pdfService, notificationService, eventBus, asynqClient)

	// Create auto bulk transfer service (enabled when any disbursement provider is registered)
	var autoBulkTransferSvc *bulktransfer.NinePayBulkTransferService
	var ninePayBatchCompletion *bulktransfer.NinePayBatchCompletionChecker
	if len(disbursementRegistry.Names()) > 0 {
		btSvc := payrollSvc.BulkTransferSvc()
		autoBulkTransferSvc = bulktransfer.NewNinePayBulkTransferService(
			btSvc.ExportService(),
			btSvc.FileRepo(),
			repos.TransactionCode,
			nil, // asynqClient set later in container.go
			eventBus,
			logger,
		)
		autoBulkTransferSvc.SetFeeProvider(disbursementFeeScheduleService)
		autoBulkTransferSvc.SetProviderResolver(disbursementRegistry)
		autoBulkTransferSvc.SetWalletPaymentRepo(repos.TxWalletPayment)
		ninePayBatchCompletion = bulktransfer.NewNinePayBatchCompletionChecker(
			repos.TransactionCode,
			repos.BulkTransferFile,
			repos.TxWalletPayment,
			idempotencyService,
			eventBus,
			logger,
		)
	}

	employeeService := employee.NewEmployeeService(employeeConfig)

	advancePaymentConfig.EmployeeService = employeeService
	// Initialize employee import service
	employeeImportService := employee.NewImportService(
		employeeService,
		repos.Employee,
		repos.Project,
		repos.ProjectEmployee,
		employeeUserService,
		employeeImportProgressService,
	)

	projectEmployeeSvc := project.NewProjectEmployeeService(repos.ProjectEmployee, repos.Employee, repos.EmployeeUser, repos.Project, repos.Timesheet, repos.AuditLog, transactionManager, repos.AdvancePayment, repos.Payrate, notificationPort, eventBus, payCycleNotificationPublisher, timesheetService, cacheService)

	// Email-OTP 2FA: build the pending-session store (Redis) + OTP service.
	//
	// OTP delivery ALWAYS uses the real Resend provider when OTP is enabled,
	// regardless of environment. Login depends on the code reaching the inbox,
	// and a sandbox/no-op provider would silently swallow the code and lock
	// every account out once MaxAttempts is hit. The Resend API key is
	// fail-fast validated at boot when OTP_ENABLE=true, so it is guaranteed
	// present here. General notifications keep the env-selected provider
	// (sandbox in dev) so development does not spam real mailboxes. When
	// OTP_ENABLE=false the service is still constructed (cheap) but
	// AuthService.requiresOTP() short-circuits and login behaves as before.
	otpEmailSender := emailProvider
	if cfg.OTP.Enabled {
		// Real Resend — login codes must reach inboxes.
		otpEmailSender = email.NewResendProvider(cfg.Notification.ResendAPIKey)
	}
	otpPendingStore := cache.NewOTPPendingStore(redis.Client, cfg.OTP.CodeTTL)
	otpService := otp.NewOTPService(otpPendingStore, repos.User, otpEmailSender, cfg.Notification.FromEmail, cfg.OTP, clk, logger)

	// Google OIDC nonce store (Redis) — single-use replay defense for id_tokens.
	nonceStore := cache.NewNonceStore(redis.Client, 10*time.Minute)

	// Self-hosted image CAPTCHA (base64Captcha + Redis). Reuses the shared
	// Redis client. When CAPTCHA_ENABLE=false the service is still constructed
	// (cheap) but RequiredForFailures() short-circuits.
	captchaService := auth.NewCaptchaService(redis.Client, auth.CaptchaConfig{
		Enabled:    cfg.Captcha.Enabled,
		Threshold:  cfg.Captcha.Threshold,
		CodeTTL:    cfg.Captcha.CodeTTL,
		CodeLength: cfg.Captcha.CodeLength,
	})

	servicesStruct := &Services{
		User:                              userService,
		PasswordResetJobManager:           passwordResetJobManager,
		Auth:                              auth.NewAuthService(userService, repos.Employee, repos.BlacklistedToken, eventBus, cfg.Auth.JWTSecret, cfg.Auth.AccessTTL, otpService, cfg.OTP, cfg.Google.ClientID, nonceStore, captchaService, logger),
		Authorization:                     authorizationService,
		Dashboard:                         dashboardService,
		Project:                           project.NewProjectService(repos.Project, repos.Employee, repos.ProjectEmployee, repos.Timesheet, eventBus, cacheService, logger),
		ProjectPermission:                 projectPermissionService,
		Employee:                          employeeService,
		EmployeePermission:                employeePermissionService,
		EmployeeUser:                      employeeUserService,
		EmployeeProfile:                   employeeProfileService,
		ProjectEmployee:                   projectEmployeeSvc,
		Bank:                              asset.NewBankService(repos.Bank, cacheService, eventBus),
		Payrate:                           payroll.NewPayrateService(repos.Payrate, eventBus, db.DB),
		Timesheet:                         timesheetService,
		TimesheetEditRequest:              timesheetEditRequestService,
		PayrollReport:                     payrollReportService,
		PayrollReportExporter:             payrollReportExporter,
		PayrollReportByProjectService:     payrollReportByProjectService,
		PayrollReportByProjectExporter:    payrollReportByProjectExporter,
		Transaction:                       transactionService,
		TransactionPort:                   transactionPort,
		SettlementUpload:                  settlementUploadService,
		Payroll:                           payrollSvc,
		Ledger:                            ledgerService,
		Notification:                      notificationService,
		Settings:                          settingsService,
		SettingsConfig:                    settingsConfigService,
		ExcelConverter:                    excelConverterService,
		Asset:                             assetService,
		Cache:                             cacheService,
		CacheService:                      cacheService,
		Lender:                            lenderService,
		Loan:                              loanService,
		Email:                             emailService,
		Metric:                            infraServices.NewMetricService(repos.APIMetric, repos.AuditLog),
		Audit:                             auditService,
		APIMetricCleanupService:           apiMetricCleanupService,
		Idempotency:                       idempotencyService,
		Reconcile:                         reconcileService,
		OnePayFeeImport:                   onePayFeeImportService,
		AdvancePayment:                    advancePaymentService,
		AdvancePaymentFeeSchedule:         feeScheduleService,
		ImportProgress:                    importProgressService,
		EmployeeImport:                    employeeImportService,
		EmployeeImportProgress:            employeeImportProgressService,
		FlexPayReconciliationService:      flexPayReconciliationService,
		FlexPayReconciliationExporter:     flexPayReconciliationExporter,
		FlexPaySettlementService:          flexPaySettlementService,
		BulkTransferTransactionWorker:     bulkTransferTransactionWorker,
		BulkTransferPaymentWorkerConcrete: bulkTransferPaymentWorker,
		Push:                              pushSvc,
		DisbursementRegistry:              disbursementRegistry,
		DisbursementFeeSchedule:           disbursementFeeScheduleService,
		ProviderTransactions:              walletPaymentService,
		WalletPaymentStats:                walletPaymentStats,
		NinepayCloser:                     ninepayCloser,
		Wallet:                            walletService,
		WalletDemandForecast:              walletDemandForecastService,
		NinePayBulkTransfer:               autoBulkTransferSvc,
		AutoBulkTransfer:                  autoBulkTransferSvc,
		NinePayBatchCompletion:            ninePayBatchCompletion,
		BulkTransferPayment:               bulkTransferPaymentWorker,
		BCCImport: services.NewBCCImportService(
			repos.Payrate,
			repos.Timesheet,
			timesheetService,
			repos.Asset,
			fileStorage,
			transactionManager,
			redis.Client,
			employeeService,
			employeeUserService,
			projectEmployeeSvc,
		),
		Attendance: attendanceService,
	}

	return servicesStruct
}
