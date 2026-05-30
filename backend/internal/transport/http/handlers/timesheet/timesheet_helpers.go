package timesheet

import (
	"context"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	constants "api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/config"
	"api-server/internal/app/services/employee"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/settlement"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
)

// Handler struct for all timesheet-related handlers
type Handler struct {
	timesheetService               *timesheet.TimesheetService
	payrollReportService           *payroll.PayrollReportService
	payrollReportExporter          *payroll.PayrollReportExporter
	payrollReportByProjectService  *payroll.PayrollReportByProjectService
	payrollReportByProjectExporter *payroll.PayrollReportByProjectExporter
	timesheetResponseService       *timesheet.TimesheetResponseService
	settingsConfigService          *config.SettingsConfigService
	projectService                 *project.ProjectService
	projectEmployeeService         *project.ProjectEmployeeService
	payrateService                 *payroll.PayrateService
	projectPermissionService       *project.ProjectPermissionService
	employeePermissionService      *employee.EmployeePermissionService
	settlementUploadService        *settlement.SettlementUploadService
	timesheetRepo                  domain.TimesheetRepository
	auditService                   interface{} // *infrastructure.AuditService
	clock                          clock.Clock
}

// NewHandler creates a new timesheet handler
func NewHandler(
	timesheetService *timesheet.TimesheetService,
	payrollReportService *payroll.PayrollReportService,
	settingsConfigService *config.SettingsConfigService,
	payrollReportExporter *payroll.PayrollReportExporter,
	payrollReportByProjectService *payroll.PayrollReportByProjectService,
	payrollReportByProjectExporter *payroll.PayrollReportByProjectExporter,
	projectService *project.ProjectService,
	projectEmployeeService *project.ProjectEmployeeService,
	payrateService *payroll.PayrateService,
	projectPermissionService *project.ProjectPermissionService,
	employeePermissionService *employee.EmployeePermissionService,
	settlementUploadService *settlement.SettlementUploadService,
	timesheetRepo domain.TimesheetRepository,
	auditService interface{},
	clk clock.Clock,
) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	timesheetResponseService := timesheet.NewTimesheetResponseService(timesheetService)
	return &Handler{
		timesheetService:               timesheetService,
		payrollReportService:           payrollReportService,
		payrollReportExporter:          payrollReportExporter,
		payrollReportByProjectService:  payrollReportByProjectService,
		payrollReportByProjectExporter: payrollReportByProjectExporter,
		timesheetResponseService:       timesheetResponseService,
		settingsConfigService:          settingsConfigService,
		projectService:                 projectService,
		projectEmployeeService:         projectEmployeeService,
		payrateService:                 payrateService,
		projectPermissionService:       projectPermissionService,
		employeePermissionService:      employeePermissionService,
		settlementUploadService:        settlementUploadService,
		timesheetRepo:                  timesheetRepo,
		auditService:                   auditService,
		clock:                          clk,
	}
}

// getUserContext extracts user ID and role from Gin context
func (h *Handler) getUserContext(c *gin.Context) (userID uint, userRole string, ok bool) {
	userIDInterface, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return 0, "", false
	}

	userRoleInterface, exists := c.Get(constants.CtxUserRole)
	if !exists {
		response.Forbidden(c, constants.MsgUserRoleNotFoundInContextVN)
		return 0, "", false
	}

	return userIDInterface.(uint), userRoleInterface.(string), true
}

// validateTimesheetID validates and parses timesheet ID from URL parameter
func (h *Handler) validateTimesheetID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, constants.MsgInvalidTimesheetIDVN)
		return 0, false
	}
	return uint(id), true
}

// buildTimesheetResponse builds a TimesheetResponse DTO from domain model
func (h *Handler) buildTimesheetResponse(timesheet *domain.Timesheet) dto.TimesheetResponse {
	return h.timesheetResponseService.BuildTimesheetResponse(timesheet)
}

// parseTimesheetFilters parses query parameters into TimesheetFilters
func (h *Handler) parseTimesheetFilters(c *gin.Context) domain.TimesheetFilters {
	params := h.extractQueryParams(c)
	return h.timesheetResponseService.ParseTimesheetFilters(c.Request.Context(), params)
}

// extractQueryParams extracts and converts query parameters to a map
func (h *Handler) extractQueryParams(c *gin.Context) map[string]interface{} {
	params := make(map[string]interface{})

	// Extract and convert pagination parameters
	if pageSize := c.Query("pageSize"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil && ps > 0 {
			params["pageSize"] = ps
		}
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			params["page"] = p
		}
	}

	// Extract sorting parameters
	if sortBy := c.Query("sortBy"); sortBy != "" {
		params["sortBy"] = sortBy
	}

	if sortOrder := c.Query("sortOrder"); sortOrder != "" {
		params["sortOrder"] = sortOrder
	}

	// Extract filter parameters
	if projectIDs := c.QueryArray("project_ids"); len(projectIDs) > 0 {
		var ids []uint
		for _, idStr := range projectIDs {
			if id, err := strconv.ParseUint(idStr, 10, 32); err == nil {
				ids = append(ids, uint(id))
			}
		}
		if len(ids) > 0 {
			params["projectIDs"] = ids
		}
	}

	if employeeID := c.Query("employee_id"); employeeID != "" {
		if id, err := strconv.ParseUint(employeeID, 10, 32); err == nil {
			uid := uint(id)
			params["employeeID"] = &uid
		}
	}

	// Extract status array
	if status := c.QueryArray("status"); len(status) > 0 {
		params["status"] = status
	}

	// Extract payment_status (for filtering approved-but-not-paid timesheets)
	if paymentStatus := c.Query("payment_status"); paymentStatus != "" {
		params["paymentStatus"] = paymentStatus
	}

	// Extract paytype array
	if payType := c.QueryArray("paytype"); len(payType) > 0 {
		params["paytype"] = payType
	}

	// Extract pending approval flag
	if pendingApproval := c.Query("pending_approval"); pendingApproval == "true" {
		params["pendingApproval"] = true
	}

	// Extract allowed edit flag (supports snake_case and camelCase)
	if allowedEdit := firstNonEmpty(c.Query("allowed_edit"), c.Query("allowedEdit")); allowedEdit != "" {
		if parsed, ok := parseBoolQueryParam(allowedEdit); ok {
			flag := parsed
			params["allowedEdit"] = &flag
		}
	}

	if requestEditID := firstNonEmpty(c.Query("request_edit_id"), c.Query("requestEditId")); requestEditID != "" {
		if id, err := strconv.ParseUint(requestEditID, 10, 32); err == nil {
			rid := uint(id)
			params["requestEditID"] = &rid
		}
	}

	if hasRequestEdit := firstNonEmpty(c.Query("has_request_edit"), c.Query("hasRequestEdit")); hasRequestEdit != "" {
		if parsed, ok := parseBoolQueryParam(hasRequestEdit); ok {
			flag := parsed
			params["hasRequestEdit"] = &flag
		}
	}

	// Extract date parameters (parse in local timezone to match DB storage)
	loc, _ := time.LoadLocation("Local")
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", fromDate, loc); err == nil {
			params["fromDate"] = &d
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if d, err := time.ParseInLocation("2006-01-02", toDate, loc); err == nil {
			params["toDate"] = &d
		}
	}

	if search := c.Query("search"); search != "" {
		params["search"] = search
	}

	return params
}

// buildTimesheetWithDetailsResponse builds a TimesheetWithDetailsResponse DTO
func (h *Handler) buildTimesheetWithDetailsResponse(timesheet *domain.Timesheet) dto.TimesheetWithDetailsResponse {
	return h.timesheetResponseService.BuildTimesheetWithDetailsResponse(timesheet)
}

// firstNonEmpty returns the first non-empty string from provided values
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// parseBoolQueryParam parses common truthy/falsey query parameter values
func parseBoolQueryParam(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "t", "yes", "y":
		return true, true
	case "0", "false", "f", "no", "n":
		return false, true
	default:
		return false, false
	}
}

// filterProjectsForPartner returns only project IDs the partner can access.
// If requestedIDs is nil/empty, returns all accessible project IDs.
func (h *Handler) filterProjectsForPartner(ctx context.Context, userID uint, requestedIDs []uint) ([]uint, error) {
	// Get all project IDs created by or shared with this partner
	filters := domain.ProjectFilters{
		AccessibleBy:  &userID,
		ProjectStatus: []domain.ProjectStatus{domain.ProjectStatusRunning},
	}
	projects, err := h.projectService.ListProjects(ctx, filters)
	if err != nil {
		return nil, err
	}
	accessibleIDs := make(map[uint]struct{}, len(projects))
	for _, p := range projects {
		accessibleIDs[p.ID] = struct{}{}
	}

	// If no specific projects requested, return all accessible
	if len(requestedIDs) == 0 {
		result := make([]uint, 0, len(accessibleIDs))
		for id := range accessibleIDs {
			result = append(result, id)
		}
		return result, nil
	}

	// Intersect requested with accessible
	var result []uint
	for _, id := range requestedIDs {
		if _, ok := accessibleIDs[id]; ok {
			result = append(result, id)
		}
	}
	return result, nil
}
