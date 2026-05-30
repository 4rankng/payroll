package services

import (
	"api-server/internal/domain"
)

// TimesheetDomainService handles business logic for timesheet operations
type TimesheetDomainService struct {
	timesheetRepo              domain.TimesheetRepository
	projectEmployeeRepo        domain.ProjectEmployeeRepository
	payrateRepo                domain.PayrateRepository
	employeeRepo               domain.EmployeeRepository
	projectRepo                domain.ProjectRepository
	validationService          *TimesheetValidationService
	calculationService         *PayrollCalculationService
	paytypeConstructionService *PaytypeConstructionService
}

// NewTimesheetDomainService creates a new timesheet domain service
func NewTimesheetDomainService(
	timesheetRepo domain.TimesheetRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	payrateRepo domain.PayrateRepository,
	employeeRepo domain.EmployeeRepository,
	projectRepo domain.ProjectRepository,
	validationService *TimesheetValidationService,
	calculationService *PayrollCalculationService,
	paytypeConstructionService *PaytypeConstructionService,
) *TimesheetDomainService {
	return &TimesheetDomainService{
		timesheetRepo:              timesheetRepo,
		projectEmployeeRepo:        projectEmployeeRepo,
		payrateRepo:                payrateRepo,
		employeeRepo:               employeeRepo,
		projectRepo:                projectRepo,
		validationService:          validationService,
		calculationService:         calculationService,
		paytypeConstructionService: paytypeConstructionService,
	}
}
