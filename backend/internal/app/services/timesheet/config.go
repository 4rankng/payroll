package timesheet

import (
	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	infraports "api-server/internal/domain/ports/infrastructure"
	domainServices "api-server/internal/domain/services"
	"log/slog"
)

// Config consolidates all dependencies for TimesheetService
type Config struct {
	// Repositories
	TimesheetRepo       domain.TimesheetRepository
	EmployeeRepo        domain.EmployeeRepository
	ProjectRepo         domain.ProjectRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	PayrateRepo         domain.PayrateRepository

	// Domain Services
	ValidationService          *domainServices.TimesheetValidationService
	CalculationService         *domainServices.PayrollCalculationService
	AssignmentService          *domainServices.EmployeeAssignmentService
	PaytypeConstructionService *domainServices.PaytypeConstructionService
	TimesheetDomainService     *domainServices.TimesheetDomainService

	// Infrastructure
	TransactionManager      *infrastructure.TransactionManager
	TransactionOrchestrator *TransactionOrchestrator
	Events                  domain.EventBus
	Cache                   infraports.CachePort
	Logger                  *slog.Logger
}

// NewTimesheetServiceFromConfig creates a TimesheetService from config
func NewTimesheetServiceFromConfig(cfg *Config) *TimesheetService {
	// Initialize bulk operations with transaction support
	bulkOperations := NewBulkOperations(cfg.TimesheetRepo, cfg.TransactionManager)

	return &TimesheetService{
		timesheetRepo:              cfg.TimesheetRepo,
		employeeRepo:               cfg.EmployeeRepo,
		projectRepo:                cfg.ProjectRepo,
		projectEmployeeRepo:        cfg.ProjectEmployeeRepo,
		payrateRepo:                cfg.PayrateRepo,
		validationService:          cfg.ValidationService,
		calculationService:         cfg.CalculationService,
		assignmentService:          cfg.AssignmentService,
		paytypeConstructionService: cfg.PaytypeConstructionService,
		timesheetDomainService:     cfg.TimesheetDomainService,
		bulkOperations:             bulkOperations,
		transactionManager:         cfg.TransactionManager,
		events:                     cfg.Events,
		cache:                      cfg.Cache,
		logger:                     cfg.Logger,
	}
}
