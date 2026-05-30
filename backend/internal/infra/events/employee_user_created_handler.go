package events

import (
	"context"
	"fmt"

	"api-server/internal/app/services/employee"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
)

// EmployeeUserCreatedHandler handles EmployeeCreatedEvent and creates
// a corresponding user account for the employee asynchronously.
type EmployeeUserCreatedHandler struct {
	employeeRepo        domain.EmployeeRepository
	employeeUserService *employee.EmployeeUserService
}

func NewEmployeeUserCreatedHandler(
	employeeRepo domain.EmployeeRepository,
	employeeUserService *employee.EmployeeUserService,
) *EmployeeUserCreatedHandler {
	return &EmployeeUserCreatedHandler{
		employeeRepo:        employeeRepo,
		employeeUserService: employeeUserService,
	}
}

func (h *EmployeeUserCreatedHandler) Handle(ctx context.Context, event domain.DomainEvent) error {
	e, ok := event.(domain.EmployeeCreatedEvent)
	if !ok {
		if evt, okPtr := event.(*domain.EmployeeCreatedEvent); okPtr && evt != nil {
			e = *evt
		} else {
			return nil
		}
	}

	logger := observability.GetLogger()

	employeeID := e.AggregateID()
	emp, err := h.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		logger.Error("EmployeeUserCreatedHandler: failed to load employee", "employee_id", employeeID, "error", err)
		return fmt.Errorf("failed to load employee for EmployeeCreatedEvent: %w", err)
	}

	if emp.UserID != nil {
		return nil
	}

	baseUsername := utils.GenerateUsername(emp.Fullname)
	if baseUsername == "" {
		logger.Error("EmployeeUserCreatedHandler: failed to generate username for employee", "employee_id", emp.ID, "fullname", emp.Fullname)
		return fmt.Errorf("failed to generate username for employee")
	}

	username := h.employeeUserService.EnsureUniqueUsername(ctx, baseUsername)
	userID, err := h.employeeUserService.CreateUserForEmployee(ctx, emp, username)
	if err != nil {
		logger.Error("EmployeeUserCreatedHandler: failed to create user for employee", "employee_id", emp.ID, "error", err)
		return fmt.Errorf("failed to create user for employee: %w", err)
	}

	emp.UserID = &userID
	if err := h.employeeRepo.Update(ctx, emp); err != nil {
		logger.Error("EmployeeUserCreatedHandler: failed to link user to employee", "employee_id", emp.ID, "user_id", userID, "error", err)
		return fmt.Errorf("failed to link user to employee: %w", err)
	}

	logger.Info("EmployeeUserCreatedHandler: user account created for employee",
		"employee_id", emp.ID,
		"user_id", userID,
		"username", username,
	)

	return nil
}

func (h *EmployeeUserCreatedHandler) CanHandle(eventType string) bool {
	return eventType == "EmployeeCreated"
}
