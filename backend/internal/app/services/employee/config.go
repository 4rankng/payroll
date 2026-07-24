package employee

import (
	"log/slog"

	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
)

// Config consolidates all dependencies for EmployeeService
// This reduces constructor complexity from 11 parameters to 1
type Config struct {
	// Repositories (6)
	EmployeeRepo        domain.EmployeeRepository
	TimesheetRepo       domain.TimesheetRepository
	BankRepo            domain.BankRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	EmployeeUserRepo    domain.EmployeeUserRepository
	UserRepo            domain.UserRepository

	// Domain & Application Services
	EmployeeDomainService *domainServices.EmployeeDomainService
	UserService           *user.UserService

	// Infrastructure
	TransactionManager domain.TransactionManager
	Events             domain.EventBus
	Cache              domain.CacheServiceUseCase
	Logger             *slog.Logger

	// BankAccountValidator performs live OnePay account verification on
	// create/update. Nil in environments without a configured provider
	// (e.g. some tests) — validation is then silently skipped.
	BankAccountValidator *BankAccountValidator
}
