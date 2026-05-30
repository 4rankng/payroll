package middleware

import (
	"strconv"
	"strings"

	authservice "api-server/internal/app/services/auth"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type AuthorizationMiddleware struct {
	authorizationService      *authservice.AuthorizationService
	projectPermissionService  *project.ProjectPermissionService
	employeePermissionService *employee.EmployeePermissionService
}

func NewAuthorizationMiddleware(authorizationService *authservice.AuthorizationService, projectPermissionService *project.ProjectPermissionService, employeePermissionService *employee.EmployeePermissionService) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{
		authorizationService:      authorizationService,
		projectPermissionService:  projectPermissionService,
		employeePermissionService: employeePermissionService,
	}
}

func (m *AuthorizationMiddleware) Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			c.Abort()
			return
		}

		userRole := c.GetString(constants.CtxUserRole)
		if userRole == "" {
			response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
			c.Abort()
			return
		}

		// If authorization service is not available, fall back to admin-only access
		if m.authorizationService == nil {
			if userRole != string(domain.RoleAdmin) {
				response.Forbidden(c, "Admin access required")
				c.Abort()
				return
			}
			c.Next()
			return
		}

		resource := c.Request.URL.Path
		action := c.Request.Method

		// Special handling for timesheet modifications - check if approved
		if strings.Contains(resource, "/timesheets/") && (action == "PUT" || action == "DELETE") && userRole == "partner" {
			timesheetID := extractTimesheetID(resource)
			if !m.authorizationService.CanAccessTimesheet(c.Request.Context(), userRole, resource, action, timesheetID) {
				response.Forbidden(c, "Cannot modify approved timesheet")
				c.Abort()
				return
			}
		} else {
			// Standard Casbin authorization check
			if !m.authorizationService.CanAccess(userRole, resource, action) {
				response.Forbidden(c, "Insufficient permissions")
				c.Abort()
				return
			}
		}

		// Check project-specific access for partners
		if userRole == string(domain.RolePartner) && m.isProjectSpecificRoute(resource) && m.projectPermissionService != nil {
			projectID := m.extractProjectID(resource)
			if projectID != nil {
				var hasAccess bool
				var err error

				// For write operations, check modify permission (excludes employee-based access)
				// For read operations, check access permission (includes employee-based access)
				if action == "PUT" || action == "PATCH" || action == "DELETE" {
					hasAccess, err = m.projectPermissionService.CanUserModifyProject(
						c.Request.Context(),
						*projectID,
						userID.(uint),
					)
				} else {
					hasAccess, err = m.projectPermissionService.CanUserAccessProject(
						c.Request.Context(),
						*projectID,
						userID.(uint),
					)
				}

				if err != nil || !hasAccess {
					response.Forbidden(c, "No access to this project")
					c.Abort()
					return
				}
			}
		}

		// Check employee-specific access for partners
		if userRole == string(domain.RolePartner) && m.isEmployeeSpecificRoute(resource) && m.employeePermissionService != nil {
			employeeID := m.extractEmployeeID(resource)
			if employeeID != nil {
				hasAccess, err := m.employeePermissionService.CanUserAccessEmployee(
					c.Request.Context(),
					*employeeID,
					userID.(uint),
				)
				if err != nil || !hasAccess {
					response.Forbidden(c, "No access to this employee")
					c.Abort()
					return
				}
			}
		}

		// Log successful authorization for debugging
		m.logAuthorizationSuccess(userID, userRole, resource, action)

		c.Next()
	}
}

func (m *AuthorizationMiddleware) logAuthorizationSuccess(userID interface{}, userRole, resource, action string) {
	// Only log in debug mode to avoid spam
	// Could be enhanced with proper structured logging
}

func extractTimesheetID(path string) *uint {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "timesheets" && i+1 < len(parts) {
			if id, err := strconv.ParseUint(parts[i+1], 10, 32); err == nil {
				timesheetID := uint(id)
				return &timesheetID
			}
		}
	}
	return nil
}

// extractProjectID extracts project ID from URL path
func (m *AuthorizationMiddleware) extractProjectID(path string) *uint {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "projects" && i+1 < len(parts) {
			if id, err := strconv.ParseUint(parts[i+1], 10, 32); err == nil {
				projectID := uint(id)
				return &projectID
			}
		}
	}
	return nil
}

// isProjectSpecificRoute checks if the route is project-specific
func (m *AuthorizationMiddleware) isProjectSpecificRoute(path string) bool {
	return strings.Contains(path, "/projects/") &&
		!strings.Contains(path, "/projects/summary") &&
		!strings.Contains(path, "/projects/shared")
}

// extractEmployeeID extracts employee ID from URL path
func (m *AuthorizationMiddleware) extractEmployeeID(path string) *uint {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if part == "employees" && i+1 < len(parts) {
			if id, err := strconv.ParseUint(parts[i+1], 10, 32); err == nil {
				employeeID := uint(id)
				return &employeeID
			}
		}
	}
	return nil
}

// isEmployeeSpecificRoute checks if the route is employee-specific
// Excludes routes that have their own creator-only checks in the handler/service layer
func (m *AuthorizationMiddleware) isEmployeeSpecificRoute(path string) bool {
	// Must be an employee route
	if !strings.Contains(path, "/employees/") {
		return false
	}

	// Exclude aggregate/collection routes (no employee ID in path).
	// Use HasSuffix or exact segment matching to avoid accidentally excluding
	// per-employee sub-routes like /employees/{id}/summary.
	aggregateExclusions := []string{
		"/employees/summary",
		"/employees/unassigned",
		"/employees/export",
		"/employees/init-users",
		"/employees/missing-bank-details",
	}
	for _, suffix := range aggregateExclusions {
		if strings.HasSuffix(path, suffix) {
			return false
		}
	}

	// Exclude CCCD lookup (value-based, not ID-based)
	if strings.Contains(path, "/employees/cccd/") {
		return false
	}

	// Exclude routes with their own creator-only/access control logic
	if strings.Contains(path, "/change-password") {
		return false
	}
	if strings.Contains(path, "/users") {
		return false
	}

	return true
}
