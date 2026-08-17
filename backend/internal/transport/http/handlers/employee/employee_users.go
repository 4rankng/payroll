package employee

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/employee"
)

// EmployeeUsersHandler handles employee sharing operations
type EmployeeUsersHandler struct {
	employeePermissionService *employee.EmployeePermissionService
}

// NewEmployeeUsersHandler creates a new employee users handler
func NewEmployeeUsersHandler(employeePermissionService *employee.EmployeePermissionService) *EmployeeUsersHandler {
	return &EmployeeUsersHandler{
		employeePermissionService: employeePermissionService,
	}
}

// ListEmployeeUsers lists all users with access to an employee
// @Summary List employee users
// @Description Get all users who have access to a specific employee
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Success 200 {array} dto.EmployeeUserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/employees/{id}/users [get]
func (h *EmployeeUsersHandler) ListEmployeeUsers(c *gin.Context) {
	employeeID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	users, err := h.employeePermissionService.GetEmployeeUsers(c.Request.Context(), employeeID)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	responses := make([]dto.EmployeeUserResponse, 0)
	for _, u := range users {
		userEmail := ""
		if u.User.Email != nil {
			userEmail = *u.User.Email
		}

		responses = append(responses, dto.EmployeeUserResponse{
			ID:           u.ID,
			UserID:       u.UserID,
			UserFullname: u.User.Fullname,
			UserEmail:    userEmail,
			GrantedBy:    u.GrantedBy,
			GrantedAt:    u.CreatedAt,
		})
	}

	response.Success(c, responses, constants.MsgEmployeeUsersRetrievedSuccessfullyVN)
}

// GrantEmployeeAccess grants access to an employee for a user
// @Summary Grant employee access
// @Description Grant access to an employee for a specific user (employee creator or admin can grant access)
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param request body dto.GrantEmployeeAccessRequest true "Grant access request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/employees/{id}/users [post]
func (h *EmployeeUsersHandler) GrantEmployeeAccess(c *gin.Context) {
	employeeID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	var req dto.GrantEmployeeAccessRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	err := h.employeePermissionService.GrantEmployeeAccess(
		c.Request.Context(),
		employeeID,
		req.UserID,
		userID,
		userRole,
	)

	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgAccessGrantedSuccessfullyVN)
}

// RequestEmployeeAccess lets a partner self-service claim management of an employee
// @Summary Request to manage an employee
// @Description Partner self-service claim: adds the employee to the partner's managed list (employee_users). Idempotent.
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/employees/{id}/request-access [post]
func (h *EmployeeUsersHandler) RequestEmployeeAccess(c *gin.Context) {
	employeeID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	alreadyManaged, err := h.employeePermissionService.RequestEmployeeAccess(
		c.Request.Context(),
		employeeID,
		userID,
		userRole,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if alreadyManaged {
		response.Success(c, nil, constants.MsgEmployeeAccessAlreadyRequestedVN)
		return
	}
	response.Success(c, nil, constants.MsgEmployeeAccessRequestedVN)
}

// RevokeEmployeeAccess revokes access to an employee from a user
// @Summary Revoke employee access
// @Description Revoke access to an employee from a specific user (employee creator or admin can revoke access)
// @Tags employees
// @Accept json
// @Produce json
// @Param id path int true "Employee ID"
// @Param userId path int true "User ID to revoke access from"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/employees/{id}/users/{userId} [delete]
func (h *EmployeeUsersHandler) RevokeEmployeeAccess(c *gin.Context) {
	employeeID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	userToRevoke, ok := helpers.ParseIDParam(c, "userId", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	err := h.employeePermissionService.RevokeEmployeeAccess(
		c.Request.Context(),
		employeeID,
		userToRevoke,
		userID,
		userRole,
	)

	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgAccessRevokedSuccessfullyVN)
}
