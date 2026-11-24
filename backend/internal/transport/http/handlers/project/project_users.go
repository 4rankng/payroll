package project

import (
	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// ListProjectUsers lists all users with access to a project
// @Summary List project users
// @Description Get all users who have access to a specific project
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {array} dto.ProjectUserResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/projects/{id}/users [get]
func (h *Handler) ListProjectUsers(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	users, err := h.projectPermissionService.GetProjectUsers(c.Request.Context(), projectID)
	if err != nil {
		h.logger.Error("Failed to get project users", "projectID", projectID, "error", err)
		response.HandleDomainError(c, err)
		return
	}

	h.logger.Info("Retrieved project users", "projectID", projectID, "userCount", len(users))
	responses := make([]dto.ProjectUserResponse, 0)
	for _, u := range users {
		userEmail := ""
		if u.User.Email != nil {
			userEmail = *u.User.Email
		}

		responses = append(responses, dto.ProjectUserResponse{
			ID:           u.ID,
			UserID:       u.UserID,
			UserFullname: u.User.Fullname,
			UserEmail:    userEmail,
			GrantedBy:    u.GrantedBy,
			GrantedAt:    u.CreatedAt,
		})
	}

	response.Success(c, responses, constants.MsgProjectUsersRetrievedSuccessfullyVN)
}

// GrantProjectAccess grants access to a project for a user
// @Summary Grant project access
// @Description Grant access to a project for a specific user (project creator or admin can grant access)
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param request body dto.GrantProjectAccessRequest true "Grant access request"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/projects/{id}/users [post]
func (h *Handler) GrantProjectAccess(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	var req dto.GrantProjectAccessRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	err := h.projectPermissionService.GrantProjectAccess(
		c.Request.Context(),
		projectID,
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

// RevokeProjectAccess revokes access to a project from a user
// @Summary Revoke project access
// @Description Revoke access to a project from a specific user (project creator or admin can revoke access)
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param userId path int true "User ID to revoke access from"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/projects/{id}/users/{userId} [delete]
func (h *Handler) RevokeProjectAccess(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	userToRevoke, ok := helpers.ParseIDParam(c, "userId", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	err := h.projectPermissionService.RevokeProjectAccess(
		c.Request.Context(),
		projectID,
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

// RequestProjectAccess lets a partner self-service claim management of a project
// @Summary Request to manage a project
// @Description Partner self-service claim: adds the project to the partner's managed list (project_users). Granted immediately — no approval step. Idempotent.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/projects/{id}/request-access [post]
func (h *Handler) RequestProjectAccess(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	alreadyManaged, err := h.projectPermissionService.RequestProjectAccess(
		c.Request.Context(),
		projectID,
		userID,
		userRole,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if alreadyManaged {
		response.Success(c, nil, constants.MsgUserAlreadyHasAccessToProjectVN)
		return
	}
	response.Success(c, nil, constants.MsgProjectAccessRequestedVN)
}

// ReleaseProjectAccess lets a partner drop their own project management grant
// @Summary Release own project management
// @Description Partner self-service release: removes the caller's own project_users grant immediately — no approval step. Idempotent; the project creator cannot self-release.
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} response.SuccessResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/v1/projects/{id}/release-access [post]
func (h *Handler) ReleaseProjectAccess(c *gin.Context) {
	projectID, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	userID := c.GetUint(constants.CtxUserID)
	userRole := c.GetString(constants.CtxUserRole)

	alreadyReleased, err := h.projectPermissionService.RequestProjectRelease(
		c.Request.Context(),
		projectID,
		userID,
		userRole,
	)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	if alreadyReleased {
		response.Success(c, nil, constants.MsgAccessRevokedSuccessfullyVN)
		return
	}
	response.Success(c, nil, constants.MsgProjectAccessReleasedVN)
}
