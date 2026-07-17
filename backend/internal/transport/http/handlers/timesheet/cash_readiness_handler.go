package timesheet

import (
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// GetCashReadiness returns the advisory cash-prep forecast for the next timesheet
// Ky. Filter parsing and partner scoping mirror GetSummary, but the summary's
// outstanding-payment amount is not part of this forecast.
func (h *Handler) GetCashReadiness(c *gin.Context) {
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	userRole := c.GetString(constants.CtxUserRole)
	filters := domain.TimesheetFilters{}

	// Partner scope: only employees they created (same as GetSummary).
	if userRole == string(domain.RolePartner) {
		if uid, ok := userID.(uint); ok {
			filters.EmployeeCreatedBy = &uid
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	}

	if projectID := c.Query("project_id"); projectID != "" {
		if id, err := strconv.ParseUint(projectID, 10, 32); err == nil {
			filters.ProjectIDs = []uint{uint(id)}
		}
	}

	if employeeID := c.Query("employee_id"); employeeID != "" {
		if id, err := strconv.ParseUint(employeeID, 10, 32); err == nil {
			uid := uint(id)
			filters.EmployeeID = &uid
		}
	}

	// Parse date parameters in local timezone to match DB storage and the other
	// timesheet endpoints. time.Parse would interpret the date as UTC, shifting the
	// half-open range [fromDate, toDate) by the UTC offset and dropping boundary-day
	// entries (the 65.7M bug class).
	loc, _ := time.LoadLocation("Local")
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", fromDate, loc); err == nil {
			filters.FromDate = &d
		}
	}
	if toDate := c.Query("toDate"); toDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", toDate, loc); err == nil {
			filters.ToDate = &d
		}
	}

	cash, err := h.cashReadinessSvc.GetCashReadiness(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetCashReadinessVN)
		return
	}

	response.Success(c, toCashReadinessResponse(cash), constants.MsgCashReadinessRetrievedVN)
}

func toCashReadinessResponse(c *domain.CashReadiness) *dto.CashReadinessResponse {
	return &dto.CashReadinessResponse{
		ObservedApproved:  c.ObservedApproved,
		ProjectedP50:      c.ProjectedP50,
		ProjectedExpected: c.ProjectedExpected,
		ProjectedP95:      c.ProjectedP95,
		BandLower:         c.BandLower,
		BandUpper:         c.BandUpper,
		ExpectedTotal:     c.ExpectedTotal,
		WalletAvailable:   c.WalletAvailable,
		WalletAvailableOK: c.WalletAvailableOK,
		CashToPrepare:     c.CashToPrepare,
		Gap:               c.Gap,
		PrepareByDate:     c.PrepareByDate.Format(time.RFC3339),
		NextPayDate:       c.NextPayDate.Format(time.RFC3339),
		LeadDays:          c.LeadDays,
		Ky:                c.Ky,
		CycleDayToday:     c.CycleDayToday,
		Method:            c.Method,
		Confidence:        c.Confidence,
		BasisCycles:       c.BasisCycles,
		GrowthRate:        c.GrowthRate,
		GeneratedAt:       c.GeneratedAt,
	}
}
