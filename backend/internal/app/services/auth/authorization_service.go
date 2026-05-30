package auth

import (
	"context"
	"log/slog"
	"path/filepath"

	"api-server/internal/app/services/timesheet"
	"api-server/internal/domain"

	"github.com/casbin/casbin/v2"
)

type AuthorizationService struct {
	enforcer         *casbin.Enforcer
	timesheetService *timesheet.TimesheetService
	logger           *slog.Logger
}

func NewAuthorizationService(
	modelPath, policyPath string,
	timesheetService *timesheet.TimesheetService,
	logger *slog.Logger,
) (*AuthorizationService, error) {
	enforcer, err := casbin.NewEnforcer(modelPath, policyPath)
	if err != nil {
		return nil, err
	}

	return &AuthorizationService{
		enforcer:         enforcer,
		timesheetService: timesheetService,
		logger:           logger,
	}, nil
}

func (s *AuthorizationService) CanAccess(userRole string, resource string, action string) bool {
	// Casbin's RegexMatch can panic on invalid patterns — recover gracefully.
	check := func(role string) (allowed bool, panicked bool) {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
				s.logger.Warn("Casbin enforcer panicked",
					"role", role,
					"resource", resource,
					"action", action,
					"panic", r,
				)
			}
		}()
		result, err := s.enforcer.Enforce(role, resource, action)
		if err != nil {
			s.logger.Error("Authorization check failed",
				"role", role, "resource", resource, "action", action, "error", err,
			)
			return false, false
		}
		return result, false
	}

	if result, _ := check("*"); result {
		return true
	}

	result, _ := check(userRole)
	if !result {
		s.logger.Info("Access denied by authorization policy",
			"role", userRole, "resource", resource, "action", action,
		)
	}
	return result
}

func (s *AuthorizationService) CanAccessTimesheet(ctx context.Context, userRole string, resource string, action string, timesheetID *uint) bool {
	if !s.CanAccess(userRole, resource, action) {
		return false
	}

	if (action == "PUT" || action == "DELETE") && timesheetID != nil && userRole == string(domain.RolePartner) {
		timesheet, err := s.timesheetService.GetTimesheet(ctx, *timesheetID)
		if err != nil {
			s.logger.Error("Failed to get timesheet for authorization check",
				"timesheet_id", *timesheetID,
				"error", err,
			)
			return false
		}

		if timesheet.Status == domain.TimesheetStatusApproved {
			s.logger.Info("Partner attempted to modify approved timesheet",
				"timesheet_id", *timesheetID,
				"status", timesheet.Status,
				"action", action,
			)
			return false
		}
	}

	return true
}

func (s *AuthorizationService) CanModifyTimesheets(ctx context.Context, userRole string, timesheetIDs []uint) bool {
	if userRole != string(domain.RolePartner) {
		return true
	}

	for _, timesheetID := range timesheetIDs {
		timesheet, err := s.timesheetService.GetTimesheet(ctx, timesheetID)
		if err != nil {
			s.logger.Error("Failed to get timesheet for bulk authorization check",
				"timesheet_id", timesheetID,
				"error", err,
			)
			return false
		}

		if timesheet.Status == domain.TimesheetStatusApproved {
			s.logger.Info("Partner attempted to modify approved timesheet in bulk operation",
				"timesheet_id", timesheetID,
				"status", timesheet.Status,
			)
			return false
		}
	}

	return true
}

// ReloadPolicy reloads the Casbin policy from disk at runtime.
func (s *AuthorizationService) ReloadPolicy() error {
	return s.enforcer.LoadPolicy()
}

func GetDefaultModelPath() string {
	return filepath.Join("configs", "casbin_model.conf")
}

func GetDefaultPolicyPath() string {
	return filepath.Join("configs", "casbin_policy.csv")
}
