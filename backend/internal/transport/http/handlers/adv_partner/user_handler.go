package adv_partner

import (
	"context"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/user"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	employeeService        *employee.EmployeeService
	projectEmployeeService *project.ProjectEmployeeService
	userService            *user.UserService
}

func NewUserHandler(employeeService *employee.EmployeeService, projectEmployeeService *project.ProjectEmployeeService, userService *user.UserService) *UserHandler {
	return &UserHandler{
		employeeService:        employeeService,
		projectEmployeeService: projectEmployeeService,
		userService:            userService,
	}
}

// UpdateAdvPartnerUser updates employee info + user account (username, password) in one call.
func (h *UserHandler) UpdateAdvPartnerUser(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidEmployeeIDVN)
	if !ok {
		return
	}

	var req dto.UpdateAdvPartnerUserRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}
	userRole := c.GetString(constants.CtxUserRole)

	ctx := c.Request.Context()
	employeeID := uint(id)

	// 1. Update employee fields if any are provided
	if hasEmployeeFields(req) {
		emp, err := h.employeeService.GetEmployeeForUpdate(ctx, employeeID)
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}

		if req.Fullname != nil {
			emp.Fullname = *req.Fullname
		}
		if req.Email != nil {
			emp.Email = req.Email
		}
		if req.CCCD != nil {
			emp.CCCD = *req.CCCD
		}
		if req.Mobile != nil {
			emp.Mobile = *req.Mobile
		}
		if req.BankID != nil {
			emp.BankID = req.BankID
		}
		if req.BankAccountNumber != nil {
			emp.BankAccountNumber = *req.BankAccountNumber
		}
		if req.BankAccountName != nil {
			emp.BankAccountName = *req.BankAccountName
		}

		if err := h.employeeService.UpdateEmployee(ctx, emp, userID.(uint)); err != nil {
			logger := observability.GetLogger()
			logger.Error("Failed to update employee", "employee_id", employeeID, "error", err)
			response.HandleDomainError(c, err)
			return
		}
	}

	// 2. Update username if provided
	if req.Username != nil && *req.Username != "" {
		emp, err := h.employeeService.GetEmployee(ctx, employeeID)
		if err != nil {
			response.HandleDomainError(c, err)
			return
		}
		if emp.UserID == nil {
			response.BadRequest(c, constants.MsgEmployeeNoUserAccountVN)
			return
		}

		updateReq := dto.UpdateUserRequest{Username: req.Username}
		if _, err := h.userService.UpdateUser(ctx, *emp.UserID, updateReq); err != nil {
			response.HandleDomainError(c, err)
			return
		}
	}

	// 3. Update password if provided
	if req.Password != nil && *req.Password != "" {
		if err := h.employeeService.ChangeEmployeePassword(ctx, employeeID, *req.Password, userID.(uint), domain.UserRole(userRole)); err != nil {
			response.HandleDomainError(c, err)
			return
		}
	}

	// 4. Reload and return full employee response
	resp, err := h.buildFullEmployeeResponse(ctx, employeeID)
	if err != nil {
		logger := observability.GetLogger()
		logger.Error("Failed to reload updated employee", "employee_id", employeeID, "error", err)
		response.HandleDomainError(c, err)
		return
	}

	response.Success(c, resp, constants.MsgEmployeeUpdatedSuccessfullyVN)
}

func hasEmployeeFields(req dto.UpdateAdvPartnerUserRequest) bool {
	return req.Fullname != nil || req.Email != nil || req.CCCD != nil ||
		req.Mobile != nil || req.BankID != nil || req.BankAccountNumber != nil ||
		req.BankAccountName != nil
}

// buildFullEmployeeResponse mirrors the existing employee handler's response building:
// fetches User separately (not preloaded by GetByID), Bank, and current projects.
func (h *UserHandler) buildFullEmployeeResponse(ctx context.Context, employeeID uint) (*dto.EmployeeResponse, error) {
	emp, err := h.employeeService.GetEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	// Fetch user for username (not preloaded)
	var username *string
	if emp.UserID != nil {
		if u, err := h.employeeService.GetUserByID(ctx, *emp.UserID); err == nil {
			uname := u.Username
			username = &uname
		}
	}

	var bankInfo *dto.EmployeeBankInfo
	if emp.Bank != nil {
		bankInfo = &dto.EmployeeBankInfo{
			ID:         emp.Bank.ID,
			BranchName: emp.Bank.BranchName,
		}
	}

	var dateOfBirthStr *string
	if emp.DateOfBirth != nil {
		dobStr := emp.DateOfBirth.Format("2006-01-02")
		dateOfBirthStr = &dobStr
	}

	// Fetch current projects
	currentProjects := []dto.EmployeeProjectInfo{}
	if h.projectEmployeeService != nil {
		filters := domain.ProjectEmployeeFilters{
			EmployeeID: &emp.ID,
			ActiveOnly: true,
			SortBy:     "created_at",
			SortOrder:  "desc",
		}
		if assignments, err := h.projectEmployeeService.ListAssignments(ctx, filters); err == nil {
			for _, a := range assignments {
				if a.LastDate == nil {
					projectName, projectCode, clientName := "", "", ""
					if a.Project.ID > 0 {
						projectName = a.Project.Name
						projectCode = a.Project.Code
						clientName = a.Project.ClientName
					}
					currentProjects = append(currentProjects, dto.EmployeeProjectInfo{
						ProjectID:              a.ProjectID,
						ProjectEmployeeID:      a.ID,
						Name:                   projectName,
						Code:                   projectCode,
						ClientName:             clientName,
						Position:               a.Position,
						StartDate:              a.StartDate.Format("2006-01-02"),
						PaymentSchedule:        a.PaymentSchedule,
						PendingPaymentSchedule: a.PendingPaymentSchedule,
						IsFlexible:             a.Project.IsFlexible,
						CheckInEnabled:         a.CheckInEnabled,
					})
				}
			}
		}
	}

	resp := &dto.EmployeeResponse{
		ID:                emp.ID,
		Username:          username,
		Fullname:          emp.Fullname,
		Email:             emp.Email,
		CCCD:              emp.CCCD,
		Address:           emp.Address,
		Mobile:            emp.Mobile,
		Bank:              bankInfo,
		BankAccountNumber: emp.BankAccountNumber,
		BankAccountName:   emp.BankAccountName,
		DateOfBirth:       dateOfBirthStr,
		CreatedBy:         emp.CreatedBy,
		CreatedAt:         emp.CreatedAt,
		UpdatedAt:         emp.UpdatedAt,
		CurrentProjects:   currentProjects,
	}
	return resp, nil
}
