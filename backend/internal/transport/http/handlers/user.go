package handlers

import (
	"math"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *user.UserService
	jobManager  *user.PasswordResetJobManager
}

func NewUserHandler(userService *user.UserService, jobManager *user.PasswordResetJobManager) *UserHandler {
	return &UserHandler{
		userService: userService,
		jobManager:  jobManager,
	}
}

// @Summary Create a new user
// @Description Create a new user account
// @Tags users
// @Accept json
// @Produce json
// @Param user body dto.CreateUserRequest true "User creation data"
// @Success 201 {object} dto.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req dto.CreateUserRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	user, err := h.userService.CreateUser(c.Request.Context(), req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.SuccessCreated(c, user, constants.MsgUserCreatedSuccessfullyVN)
}

// @Summary Get user by ID
// @Description Get a user by their ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	user, err := h.userService.GetUser(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, dto.ToUserResponse(user), constants.MsgUserRetrievedSuccessfullyVN)
}

// @Summary Update user
// @Description Update user information
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body dto.UpdateUserRequest true "User update data"
// @Success 200 {object} dto.UserResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	var req dto.UpdateUserRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	user, err := h.userService.UpdateUser(c.Request.Context(), id, req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, user, constants.MsgUserUpdatedSuccessfullyVN)
}

// @Summary Delete user
// @Description Delete a user by ID
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	if err := h.userService.DeleteUser(c.Request.Context(), id); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.NoContent(c)
}

// @Summary List users
// @Description Get a paginated list of users with filtering and search
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Items per page" default(100)
// @Param sortBy query string false "Sort by field" default(created_at)
// @Param sortOrder query string false "Sort order (asc/desc)" default(desc)
// @Param role query string false "Filter by role (admin/partner/employee)"
// @Param search query string false "Search by username, fullname, email (case-insensitive, partial match, vietnamese normalization)"
// @Success 200 {object} dto.ListUsersResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users [get]
func (h *UserHandler) ListUsers(c *gin.Context) {
	var req dto.ListUsersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	users, total, err := h.userService.ListUsers(c.Request.Context(), req)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Calculate pagination info
	totalPages := int(math.Ceil(float64(total) / float64(req.PageSize)))

	pagination := response.Pagination{
		Page:         req.Page,
		PageSize:     req.PageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgUsersRetrievedSuccessfullyVN
	if len(users) == 0 {
		message = constants.MsgNoUsersFoundVN
	}

	response.SuccessWithPagination(c, users, message, pagination)
}

// @Summary Get user statistics summary
// @Description Get user statistics summary for admin dashboard
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.UserSummaryResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/summary [get]
func (h *UserHandler) GetUserSummary(c *gin.Context) {
	summary, err := h.userService.GetUserSummary(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, summary, constants.MsgUserSummaryFetchedSuccessfullyVN)
}

// @Summary Reset user password
// @Description Reset user password (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body dto.ResetUserPasswordRequest true "Reset password data"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/reset-password [post]
func (h *UserHandler) ResetUserPassword(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	var req dto.ResetUserPasswordRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	if err := h.userService.ResetUserPassword(c.Request.Context(), id, req.Password); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgUserPasswordResetSuccessfullyVN)
}

// @Summary Get user activity summary
// @Description Get user activity summary (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param days query int false "Number of days to look back" default(30)
// @Success 200 {object} dto.UserActivitiesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/{id}/activities [get]
func (h *UserHandler) GetUserActivities(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidUserIDVN)
	if !ok {
		return
	}

	var req dto.UserActivitiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, constants.MsgInvalidQueryParametersVN)
		return
	}

	days := req.Days
	if days == 0 {
		days = 30
	}

	activities, err := h.userService.GetUserActivities(c.Request.Context(), id, days)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, activities, constants.MsgUserActivitySummaryFetchedSuccessfullyVN)
}

// @Summary Reset first-time login passwords
// @Description Start password reset job for all users who have never logged in (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.StartPasswordResetJobResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/reset-first-time-login-password [post]
func (h *UserHandler) ResetFirstTimeLoginPasswords(c *gin.Context) {
	if err := h.jobManager.StartJob(c.Request.Context()); err != nil {
		if err == user.ErrJobAlreadyRunning {
			response.Conflict(c, err.Error())
			return
		}
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, nil, constants.MsgUserPasswordResetProcessingVN)
}

// @Summary Get password reset job status
// @Description Get the status of the password reset job (Admin only)
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} dto.PasswordResetJobStatusResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /users/reset-first-time-login-password [get]
func (h *UserHandler) GetPasswordResetJobStatus(c *gin.Context) {
	// Check if a job exists (in Redis or in-memory)
	if !h.jobManager.JobExists() {
		response.NotFound(c, constants.MsgUserPasswordResetJobNotFoundVN)
		return
	}

	status := h.jobManager.GetStatus()
	result := h.jobManager.GetResult()

	if status == dto.JobStatusCompleted && result != nil {
		responseData := dto.ResetFirstTimeLoginPasswordResponse{
			Total:     result.Total,
			Usernames: result.Usernames,
		}
		response.Success(c, responseData, constants.MsgUserPasswordResetSuccessfullyVN)
		return
	}

	response.Success(c, nil, constants.MsgUserPasswordResetProcessingVN)
}
