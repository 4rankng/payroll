package employee

import (
	"api-server/internal/app/dto"
	"api-server/internal/app/services/employee"
	"api-server/internal/constants"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// CheckEmployeeDuplicates finds existing employees matching the given identifiers.
// Global by design — the point is to surface employees created by other partners
// before a duplicate is created. Only masked identifiers the requester typed are
// revealed.
// @Summary Check for duplicate employees
// @Description Find existing employees matching CCCD, mobile, or email before creating a new one. At least one identifier required.
// @Tags employees
// @Accept json
// @Produce json
// @Param cccd query string false "Citizen ID to check (exact match)"
// @Param mobile query string false "Mobile number to check (normalized, exact match)"
// @Param email query string false "Email to check (case-insensitive, exact match)"
// @Success 200 {object} dto.DuplicateCheckResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/duplicate-check [get]
func (h *Handler) CheckEmployeeDuplicates(c *gin.Context) {
	params := employee.DuplicateCheckParams{
		CCCD:   c.Query("cccd"),
		Mobile: c.Query("mobile"),
		Email:  c.Query("email"),
	}

	matches, err := h.employeeService.CheckDuplicates(c.Request.Context(), params)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	items := make([]dto.DuplicateCheckItemResponse, 0, len(matches))
	for _, m := range matches {
		items = append(items, dto.DuplicateCheckItemResponse{
			ID:                  m.ID,
			Fullname:            m.Fullname,
			CCCDMasked:          m.CCCDMasked,
			MobileMasked:        m.MobileMasked,
			EmailMasked:         m.EmailMasked,
			CurrentProjectNames: m.CurrentProjectNames,
			CreatedByName:       m.CreatedByName,
			CreatedAt:           m.CreatedAt,
			MatchedOn:           m.MatchedOn,
		})
	}

	response.Success(c, dto.DuplicateCheckResponse{
		HasDuplicates: len(items) > 0,
		Data:          items,
	}, constants.MsgEmployeeDuplicatesFoundVN)
}

// GetDuplicateIdentities reports phone numbers and CCCDs carried by more than
// one employee record.
//
// Such a number cannot log in: identity.Resolver refuses to choose between the
// colliding accounts, so the operator sees a successful password reset and the
// employee still cannot authenticate. This endpoint is the admin-facing way to
// see which records need disambiguating. It is strictly read-only — merging or
// reassigning duplicate records is a business decision, not an automatic one.
// @Summary List duplicated employee identities
// @Description List mobiles and CCCDs shared by more than one employee record; those accounts cannot log in via mobile
// @Tags employees
// @Accept json
// @Produce json
// @Success 200 {object} dto.DuplicateIdentitiesResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /employees/duplicate-identities [get]
func (h *Handler) GetDuplicateIdentities(c *gin.Context) {
	groups, err := h.employeeService.GetDuplicateIdentities(c.Request.Context())
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	items := make([]dto.DuplicateIdentityGroupResponse, 0, len(groups))
	for _, g := range groups {
		items = append(items, dto.DuplicateIdentityGroupResponse{
			Field:       g.Field,
			Value:       g.Value,
			EmployeeIDs: g.EmployeeIDs,
			UserIDs:     g.UserIDs,
			Count:       g.Count,
		})
	}

	response.Success(c, dto.DuplicateIdentitiesResponse{
		Total: len(items),
		Data:  items,
	}, constants.MsgEmployeeDuplicatesFoundVN)
}
