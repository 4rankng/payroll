package bootstrap

import (
	"log/slog"

	bootstrapInfra "api-server/internal/app/bootstrap/infrastructure"
	bootstrapRepos "api-server/internal/app/bootstrap/repositories"
	bootstrapServices "api-server/internal/app/bootstrap/services"
	"api-server/internal/app/services/auth"
	"api-server/internal/app/services/cleanup"
	dbExportSvc "api-server/internal/app/services/db_export"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/ledger"
	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/scheduler"
	"api-server/internal/app/workers"
	"api-server/internal/config"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/domain/wallet"
	asynqinfra "api-server/internal/infra/asynq"
	"api-server/internal/infra/events"
	"api-server/internal/infra/observability"
	"api-server/internal/infra/persistence"
	"api-server/internal/infra/storage"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/tenantqueue"
	"api-server/internal/transport/http/handlers"
	adminHandlers "api-server/internal/transport/http/handlers/admin"
	advPartnerHandlers "api-server/internal/transport/http/handlers/adv_partner"
	advancePaymentHandlers "api-server/internal/transport/http/handlers/advance_payment"
	attendanceHandlers "api-server/internal/transport/http/handlers/attendance"
	disbursementHandlers "api-server/internal/transport/http/handlers/disbursement"
	employeeHandlers "api-server/internal/transport/http/handlers/employee"
	pushHandlers "api-server/internal/transport/http/handlers/push"
	timesheetHandlers "api-server/internal/transport/http/handlers/timesheet"
	"api-server/internal/transport/http/middleware"

	"github.com/gin-gonic/gin"
)

type Container struct {
	Config             *config.Config
	Logger             *slog.Logger
	DB                 *persistence.Database
	Redis              *persistence.RedisClient
	Scheduler          *scheduler.Scheduler
	TenantQueueManager *tenantqueue.Manager
	AsynqClient        *asynqinfra.Client
	AsynqServer        *asynqinfra.Server

	Repos      *bootstrapRepos.Repositories
	Services   *bootstrapServices.Services
	Handlers   *Handlers
	Middleware *Middleware
}

type Handlers struct {
	User                 *handlers.UserHandler
	Auth                 *handlers.AuthHandler
	Dashboard            *handlers.DashboardHandler
	Project              *handlers.ProjectHandler
	Employee             *handlers.EmployeeHandler
	EmployeeUsers        *employeeHandlers.EmployeeUsersHandler
	EmployeeProfile      *handlers.EmployeeProfileHandler
	ProjectEmployee      *handlers.ProjectEmployeeHandler
	AdminClock           *adminHandlers.ClockHandler
	AdminAttendance      *adminHandlers.AttendanceHandler
	Bank                 *handlers.BankHandler
	Timesheet            *handlers.TimesheetHandler
	TimesheetEditRequest *handlers.TimesheetEditRequestHandler
	Payroll              *handlers.PayrollHandler
	Payrate              *handlers.PayrateHandler
	Ledger               *handlers.LedgerHandler
	Transaction          *handlers.TransactionHandler
	Health               *handlers.HealthHandler
	Notification         *handlers.NotificationHandler
	Settings             *handlers.SettingsHandler
	Asset                *handlers.AssetHandler
	Cache                *handlers.CacheHandler
	Lender               *handlers.LenderHandler
	Loan                 *handlers.LoanHandler
	Email                *handlers.EmailHandler
	Settlement           *handlers.SettlementHandler
	Metric               *handlers.MetricHandler
	Audit                *handlers.AuditHandler
	AdvancePayment       *advancePaymentHandlers.AdvancePaymentHandler
	AdvancePaymentFee    *advancePaymentHandlers.FeeScheduleHandler
	EmployeeImport       *employeeHandlers.ImportHandler
	DBExport             *handlers.DBExportHandler
	Push                 *pushHandlers.Handler
	Cron                 *handlers.CronHandler
	DisbursementWebhook  *disbursementHandlers.WebhookHandler
	DisbursementFee      *disbursementHandlers.FeeScheduleHandler
	DisbursementSettings *disbursementHandlers.SettingsHandler
	ManualDisbursement   *disbursementHandlers.ManualDisbursementHandler
	ReconciliationExport *disbursementHandlers.ReconciliationExportHandler
	ProviderTransactions *adminHandlers.WalletPaymentStatsHandler
	Wallet               *handlers.WalletHandler
	AdvPartnerUser       *advPartnerHandlers.UserHandler
	BCCImport            *timesheetHandlers.BCCImportHandler
	Attendance           *attendanceHandlers.Handler
}

type Middleware struct {
	Auth            *middleware.AuthMiddleware
	Authorization   *middleware.AuthorizationMiddleware
	LoginRateLimit  gin.HandlerFunc
	APIRateLimit    gin.HandlerFunc
	StrictRateLimit gin.HandlerFunc
	TenantSemaphore *middleware.TenantSemaphoreMiddleware
	APIMetrics      gin.HandlerFunc
}

func NewContainer(cfg *config.Config, version string) (*Container, error) {
	infra, err := bootstrapInfra.Initialize(cfg)
	if err != nil {
		return nil, err
	}

	eventBus := events.NewWorkerPoolEventBus(100, 10000)
	repos := bootstrapRepos.Initialize(infra.DB, eventBus)

	// Shared clock instance — must be created before services so attendance
	// service (and others) receive the correct FakeClock in non-prod.
	var clk clock.Clock
	env := cfg.App.Env
	if env == "production" {
		clk = clock.New()
	} else {
		clk = clock.NewAutoFake()
		clock.SetGlobal(clk)
	}

	// Initialize Asynq client before services so it can be injected into ResultProcessor
	asynqClient, err := asynqinfra.NewClient(cfg.Asynq)
	if err != nil {
		return nil, err
	}
	asynqServer := asynqinfra.NewServer(cfg.Asynq)

	services := bootstrapServices.Initialize(repos, cfg, infra.Logger, infra.DB, infra.Redis, eventBus, asynqClient, clk)

	h := initHandlers(services, repos, infra.DB, infra.Redis, infra.Logger, version, eventBus, cfg, asynqClient, clk)
	middlewares := initMiddleware(services.Auth, services.Authorization, services.ProjectPermission, services.EmployeePermission, repos.APIMetric, cfg)
	sched := initScheduler(services.Notification, services.Email, services.ProjectEmployee, services.APIMetricCleanupService, services.Reconcile, repos.APIMetric, repos.AdvancePaymentRequest, services.Wallet, services.FlexPayReconciliationService, cfg, infra.Logger)
	sched.SetRepo(repos.CronJobStatus)
	h.Cron = handlers.NewCronHandler(repos.CronJobStatus, sched)
	// init tenant queue manager for per-tenant background workers
	tqm := tenantqueue.NewManager(cfg.TenantQueueWorkers, cfg.TenantQueueBuffer)

	// Disbursement poller workers (conditional on provider flags)
	var disbursementPollerWorker *workers.DisbursementPollerWorker
	var disbursementExecuteWorker *workers.DisbursementExecuteWorker
	if cfg.Disbursement.EmployeeDisbursementEnabled() {
		disbursementPollerWorker = workers.NewDisbursementPollerWorker(
			repos.AdvancePaymentRequest,
			repos.AdvancePayment,
			repos.Employee,
			services.ProviderTransactions,
			services.DisbursementRegistry,
			asynqClient.AsynqClient(),
			services.Wallet,
			repos.User,
			services.Email,
			services.Notification,
			infra.Logger,
		)
		disbursementExecuteWorker = workers.NewDisbursementExecuteWorker(
			services.ProviderTransactions,
			services.DisbursementRegistry,
			repos.Bank,
			services.Wallet,
			infra.Logger,
		)
		infra.Logger.Info("disbursement poller: enabled")
	} else {
		infra.Logger.Info("disbursement poller: disabled (no provider has EnabledForEmployee)")
	}

	// Wire 9Pay bulk transfer async client
	if services.NinePayBulkTransfer != nil {
		services.NinePayBulkTransfer.SetAsynqClient(asynqClient.AsynqClient())
	}

	// Create 9Pay bulk execute worker (nil if not enabled)
	var ninePayBulkExecuteWorker *workers.NinePayBulkExecuteWorker
	if services.NinePayBulkTransfer != nil {
		ninePayBulkExecuteWorker = workers.NewNinePayBulkExecuteWorker(
			services.ProviderTransactions,
			services.DisbursementRegistry,
			repos.BulkTransferFile,
			infra.Logger,
		)
	}

	// Wallet settlement worker — only when employee disbursement is enabled
	var walletSettlementWorker *workers.WalletSettlementWorker
	var statusInquiryPollerWorker *workers.StatusInquiryPollerWorker
	if cfg.Disbursement.EmployeeDisbursementEnabled() {
		walletSettlementWorker = workers.NewWalletSettlementWorker(
			repos.WalletPayment,
			repos.AdvancePaymentRequest,
			repos.Transaction,
			repos.Ledger,
			infra.Logger,
		)
		statusInquiryPollerWorker = workers.NewStatusInquiryPollerWorker(
			repos.TxWalletPayment,
			services.ProviderTransactions,
			services.DisbursementRegistry,
			services.BulkTransferPayment,
			infra.Logger,
		)
	}

	// Register Asynq handlers and periodic tasks
	asynqHandlers := asynqinfra.NewHandlers(
		workers.NewEmployeeImportWorker(
			services.EmployeeImport,
			services.EmployeeImportProgress,
			cfg.Asset.StoragePath,
		),
		workers.NewImportJobWorker(
			repos.Asset,
			services.AdvancePayment,
			services.ImportProgress,
			cfg.Asset.StoragePath,
		),
		workers.NewIPNProcessWorker(services.ProviderTransactions, repos.WalletIPN, services.BulkTransferPayment, infra.Logger),
		disbursementPollerWorker,
		disbursementExecuteWorker,
		ninePayBulkExecuteWorker,
		services.BulkTransferTransactionWorker,
		services.BulkTransferPaymentWorkerConcrete,
		workers.NewAuditLogWriteWorker(repos.AuditLog),
		walletSettlementWorker,
		statusInquiryPollerWorker,
		workers.NewAutoRejectCheckoutWorker(services.Attendance),
		workers.NewAutoRejectSweepWorker(services.Attendance),
	)
	asynqinfra.RegisterHandlers(asynqServer, asynqHandlers)
	if err := asynqinfra.RegisterPeriodicTasks(asynqServer, asynqClient); err != nil {
		return nil, err
	}

	// Register disbursement poller periodic task (only when enabled)
	if disbursementPollerWorker != nil {
		if err := asynqinfra.RegisterDisbursementPoller(asynqServer); err != nil {
			return nil, err
		}
	}

	// Register wallet settlement periodic task (only when employee disbursement enabled)
	if walletSettlementWorker != nil {
		if err := asynqinfra.RegisterWalletSettlement(asynqServer); err != nil {
			return nil, err
		}
	}

	// Register status inquiry poller periodic task (only when employee disbursement enabled)
	if statusInquiryPollerWorker != nil {
		if err := asynqinfra.RegisterStatusInquiryPoller(asynqServer); err != nil {
			return nil, err
		}
	}

	// Register the auto-reject fallback sweep (always on) — finalizes attendance
	// records whose scheduled K+1h task was lost (Redis/process outage at check-in).
	if err := asynqinfra.RegisterAutoRejectSweep(asynqServer); err != nil {
		return nil, err
	}

	return &Container{
		Config:             cfg,
		Logger:             infra.Logger,
		DB:                 infra.DB,
		Redis:              infra.Redis,
		Scheduler:          sched,
		TenantQueueManager: tqm,
		AsynqClient:        asynqClient,
		AsynqServer:        asynqServer,
		Repos:              repos,
		Services:           services,
		Handlers:           h,
		Middleware:         middlewares,
	}, nil
}

func initHandlers(services *bootstrapServices.Services, repos *bootstrapRepos.Repositories, db *persistence.Database, redis *persistence.RedisClient, logger *slog.Logger, version string, eventBus domain.EventBus, cfg *config.Config, asynqClient *asynqinfra.Client, clk clock.Clock) *Handlers {
	healthCheckers := map[string]handlers.HealthChecker{
		"database": db,
		"redis":    redis,
	}

	// Create fileStorage for handlers
	fileStorage := storage.NewLocalFileStorage(cfg.Asset.StoragePath, cfg.Asset.BaseURL)

	// Create employee import handler with Asynq client
	employeeImportHandler := employeeHandlers.NewImportHandler(
		services.EmployeeImport,
		services.EmployeeImportProgress,
		fileStorage,
		asynqClient,
		services.Audit,
	)

	return &Handlers{
		User:                 handlers.NewUserHandler(services.User, services.PasswordResetJobManager),
		Auth:                 handlers.NewAuthHandler(services.Auth, services.User),
		Dashboard:            handlers.NewDashboardHandler(services.Dashboard),
		Project:              handlers.NewProjectHandlerWithServices(services.Project, services.Payrate, services.Timesheet, services.ProjectEmployee, services.ProjectPermission, clk),
		Employee:             handlers.NewEmployeeHandlerWithServices(services.Employee, services.Timesheet, services.ProjectEmployee, services.Audit, clk),
		EmployeeUsers:        employeeHandlers.NewEmployeeUsersHandler(services.EmployeePermission),
		EmployeeProfile:      handlers.NewEmployeeProfileHandler(services.EmployeeProfile, services.EmployeeUser),
		ProjectEmployee:      handlers.NewProjectEmployeeHandler(services.ProjectEmployee, services.Employee, services.Project, services.ProjectPermission, services.EmployeePermission, logger, clk),
		Bank:                 handlers.NewBankHandler(services.Bank),
		Timesheet:            handlers.NewTimesheetHandler(services.Timesheet, services.PayrollReport, services.SettingsConfig, services.PayrollReportExporter, services.PayrollReportByProjectService, services.PayrollReportByProjectExporter, services.Project, services.ProjectEmployee, services.Payrate, services.ProjectPermission, services.EmployeePermission, services.SettlementUpload, repos.Timesheet, services.Audit, clk),
		TimesheetEditRequest: handlers.NewTimesheetEditRequestHandler(services.TimesheetEditRequest),
		Payroll:              handlers.NewPayrollHandler(services.Payroll, services.AutoBulkTransfer, services.Cache, repos.BulkTransferFile, repos.Asset, fileStorage, eventBus),
		Payrate:              handlers.NewPayrateHandler(services.Payrate, services.Project, services.ProjectPermission, clk),
		Ledger:               handlers.NewLedgerHandler(services.Ledger, clk),
		Transaction:          handlers.NewTransactionHandler(services.Transaction, repos.BulkTransferFile, repos.Timesheet, services.Notification, eventBus, clk),
		Health:               handlers.NewHealthHandler(healthCheckers, version),
		Notification:         handlers.NewNotificationHandler(services.Notification),
		Settings:             handlers.NewSettingsHandler(services.Settings),
		Asset:                handlers.NewAssetHandler(services.Asset),
		Cache:                handlers.NewCacheHandler(services.Cache, logger),
		Lender:               handlers.NewLenderHandler(services.Lender),
		Loan:                 handlers.NewLoanHandler(services.Loan, clk),
		Email:                handlers.NewEmailHandler(services.Email),
		Settlement:           handlers.NewSettlementHandler(services.SettlementUpload, repos.Notification, clk),
		Metric:               handlers.NewMetricHandler(services.Metric, services.Cache, eventBus, clk),
		Audit:                handlers.NewAuditHandler(repos.AuditLog),
		AdvancePayment:       advancePaymentHandlers.NewAdvancePaymentHandler(services.AdvancePayment, repos.Asset, asynqClient, services.ImportProgress, fileStorage, services.FlexPayReconciliationService, services.FlexPayReconciliationExporter, services.FlexPaySettlementService, services.Email, services.Audit, repos.Notification, repos.WalletPayment, clk),
		AdvancePaymentFee:    advancePaymentHandlers.NewFeeScheduleHandler(services.AdvancePaymentFeeSchedule, clk),
		EmployeeImport:       employeeImportHandler,
		DBExport:             handlers.NewDBExportHandler(dbExportSvc.NewDBExportService(db, services.CacheService, cfg.Asset.StoragePath), services.Audit),
		Push:                 pushHandlers.NewHandler(services.Push),
		DisbursementWebhook: func() *disbursementHandlers.WebhookHandler {
			var onepayFileLogger *slog.Logger
			if cfg.Disbursement.Onepay.Enabled {
				onepayFileLogger, _ = observability.NewFileLogger("logs/payment-gateway.log")
			}
			return disbursementHandlers.NewWebhookHandler(services.DisbursementRegistry, services.ProviderTransactions, asynqClient, repos.WalletIPN, logger, onepayFileLogger)
		}(),
		DisbursementSettings: disbursementHandlers.NewSettingsHandler(
			services.DisbursementRegistry,
			services.DisbursementFeeSchedule,
			services.Wallet,
			cfg.Disbursement.Onepay.EnabledForEmployee,
			cfg.Disbursement.Ninepay.EnabledForEmployee,
			cfg.Disbursement.Onepay.EnabledForBulkTransfer,
			cfg.Disbursement.Ninepay.EnabledForBulkTransfer,
		),
		DisbursementFee: disbursementHandlers.NewFeeScheduleHandler(services.DisbursementFeeSchedule, clk),
		ManualDisbursement: disbursementHandlers.NewManualDisbursementHandler(services.DisbursementRegistry, services.ProviderTransactions, repos.Bank, repos.TransactionCode, services.Wallet, logger, func() *slog.Logger {
			if cfg.Disbursement.Onepay.Enabled {
				l, _ := observability.NewFileLogger("logs/payment-gateway.log")
				return l
			}
			return nil
		}()),
		ReconciliationExport: disbursementHandlers.NewReconciliationExportHandler(services.DisbursementRegistry, logger),
		ProviderTransactions: adminHandlers.NewWalletPaymentStatsHandler(services.WalletPaymentStats, logger),
		AdminClock:           adminHandlers.NewClockHandler(clk, cfg.App.Env),
		AdminAttendance:      adminHandlers.NewAttendanceHandler(services.Attendance, clk, logger),
		Wallet:               handlers.NewWalletHandler(services.Wallet, services.DisbursementRegistry, clk),
		AdvPartnerUser:       advPartnerHandlers.NewUserHandler(services.Employee, services.ProjectEmployee, services.User),
		BCCImport:            timesheetHandlers.NewBCCImportHandler(services.BCCImport, repos.Asset, fileStorage, services.ProjectPermission, services.Audit),
		Attendance:           attendanceHandlers.NewHandler(services.Attendance, repos.Employee, clk, logger),
	}
}

func initMiddleware(authService *auth.AuthService, authorizationService *auth.AuthorizationService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService, apiMetricRepo domain.APIMetricRepository, cfg *config.Config) *Middleware {
	tenantLimit := 10
	if cfg.TenantConcurrencyLimit > 0 {
		tenantLimit = cfg.TenantConcurrencyLimit
	}

	var authorizationMiddleware *middleware.AuthorizationMiddleware
	if authorizationService != nil {
		authorizationMiddleware = middleware.NewAuthorizationMiddleware(authorizationService, projectPermissionService, employeePermissionService)
	} else {
		authorizationMiddleware = middleware.NewAuthorizationMiddleware(nil, projectPermissionService, employeePermissionService)
	}

	return &Middleware{
		Auth:            middleware.NewAuthMiddleware(authService),
		Authorization:   authorizationMiddleware,
		LoginRateLimit:  middleware.CreateLoginRateLimit(cfg.Redis.Addr),
		APIRateLimit:    middleware.CreateAPIRateLimit(cfg.Redis.Addr),
		StrictRateLimit: middleware.CreateStrictRateLimit(cfg.Redis.Addr),
		TenantSemaphore: middleware.NewTenantSemaphoreMiddleware(tenantLimit),
		APIMetrics:      middleware.APIMetrics(apiMetricRepo),
	}
}

func initScheduler(notificationService *notification.NotificationService, emailService *notification.EmailService, projectEmployeeService *project.ProjectEmployeeService, apiMetricCleanupService *cleanup.APIMetricCleanupService, reconcileService *ledger.ReconcileService, apiMetricRepo domain.APIMetricRepository, advancePaymentReqRepo domain.AdvancePaymentRequestRepository, walletSvc wallet.WalletService, flexPayReconciliationSvc *domainServices.FlexPayReconciliationService, cfg *config.Config, logger *slog.Logger) *scheduler.Scheduler {
	s := scheduler.NewScheduler(
		logger,
		cfg.Scheduler.Timezone,
		cfg.Scheduler.Enabled,
	)

	registerSchedulerJobs(
		s,
		notificationService,
		emailService,
		projectEmployeeService,
		apiMetricCleanupService,
		reconcileService,
		apiMetricRepo,
		advancePaymentReqRepo,
		walletSvc,
		flexPayReconciliationSvc,
		logger,
	)

	return s
}
