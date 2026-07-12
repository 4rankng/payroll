package project

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/transport/http/helpers"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"

	"api-server/internal/app/services/attendance"
	"api-server/internal/app/services/payroll"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/timesheet"
	"api-server/internal/pkg/clock"
	"api-server/internal/pkg/timeutil"
)

// Helper function to convert time.Time pointer to string pointer
func timeToStringPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format(timeutil.DateFormat)
	return &str
}

func validateSalaryPeriodDay(field string, value *int) error {
	if value == nil {
		return nil
	}
	if *value < 0 || *value > 28 {
		return fmt.Errorf("%s phải nằm trong khoảng 0-28", field)
	}
	return nil
}

// Helper function to convert domain geofence gates to DTO
func geofenceGatesToDTO(gates []domain.GeofenceGate) []dto.GeofenceGateRequest {
	result := make([]dto.GeofenceGateRequest, len(gates))
	for i, g := range gates {
		result[i] = dto.GeofenceGateRequest{Name: g.Name, Lat: g.Lat, Lng: g.Lng}
	}
	return result
}

// Helper function to convert domain shift names to DTO
func shiftNamesToDTO(names []domain.ShiftName) []dto.ShiftNameRequest {
	if len(names) == 0 {
		return []dto.ShiftNameRequest{}
	}
	result := make([]dto.ShiftNameRequest, len(names))
	for i, n := range names {
		result[i] = dto.ShiftNameRequest{Range: n.Range, Name: n.Name}
	}
	return result
}

// Helper function to convert DTO shift names to domain
func shiftNamesFromDTO(req []dto.ShiftNameRequest) []domain.ShiftName {
	if len(req) == 0 {
		return nil
	}
	result := make([]domain.ShiftName, len(req))
	for i, n := range req {
		result[i] = domain.ShiftName{Range: n.Range, Name: n.Name}
	}
	return result
}

// Helper function to convert ProjectWithEmployeeCount to ProjectResponse
func projectWithEmployeeCountToResponse(project *domain.ProjectWithEmployeeCount) dto.ProjectResponse {
	return dto.ProjectResponse{
		ID:                         project.ID,
		ClientName:                 project.ClientName,
		Name:                       project.Name,
		Code:                       project.Code,
		StartDate:                  timeToStringPtr(project.StartDate),
		EndDate:                    timeToStringPtr(project.EndDate),
		SalaryPeriodFrom:           project.SalaryPeriodFrom,
		SalaryPeriodTo:             project.SalaryPeriodTo,
		OffDays:                    project.OffDays,
		EmployeeCount:              project.EmployeeCount,
		WeeklySalaryEmployeeCount:  project.WeeklySalaryEmployeeCount,
		MonthlySalaryEmployeeCount: project.MonthlySalaryEmployeeCount,
		Status:                     string(project.ProjectStatus),
		IsFlexible:                 project.IsFlexible,
		GeofenceGates:              geofenceGatesToDTO(project.GeofenceGates),
		GeofenceRadiusMeters:       project.GeofenceRadiusMeters,
		ShiftNames:                 shiftNamesToDTO(project.ShiftNames),
		CreatedBy:                  project.CreatedBy,
		CreatedAt:                  project.CreatedAt,
		UpdatedAt:                  project.UpdatedAt,
	}
}

// Helper function to convert Project to ProjectResponse with zero employee count
func projectToResponse(project *domain.Project) dto.ProjectResponse {
	return dto.ProjectResponse{
		ID:                         project.ID,
		ClientName:                 project.ClientName,
		Name:                       project.Name,
		Code:                       project.Code,
		StartDate:                  timeToStringPtr(project.StartDate),
		EndDate:                    timeToStringPtr(project.EndDate),
		SalaryPeriodFrom:           project.SalaryPeriodFrom,
		SalaryPeriodTo:             project.SalaryPeriodTo,
		OffDays:                    project.OffDays,
		EmployeeCount:              0, // Default to 0 when not calculated
		WeeklySalaryEmployeeCount:  0, // Default to 0 when not calculated
		MonthlySalaryEmployeeCount: 0, // Default to 0 when not calculated
		Status:                     string(project.ProjectStatus),
		IsFlexible:                 project.IsFlexible,
		GeofenceGates:              geofenceGatesToDTO(project.GeofenceGates),
		GeofenceRadiusMeters:       project.GeofenceRadiusMeters,
		ShiftNames:                 shiftNamesToDTO(project.ShiftNames),
		CreatedBy:                  project.CreatedBy,
		CreatedAt:                  project.CreatedAt,
		UpdatedAt:                  project.UpdatedAt,
	}
}

type Handler struct {
	logger                   *slog.Logger
	projectService           *project.ProjectService
	payrateService           *payroll.PayrateService
	timesheetService         *timesheet.TimesheetService
	projectEmployeeService   *project.ProjectEmployeeService
	projectPermissionService *project.ProjectPermissionService
	clock                    clock.Clock
}

func NewHandlerWithServices(
	projectService *project.ProjectService,
	payrateService *payroll.PayrateService,
	timesheetService *timesheet.TimesheetService,
	projectEmployeeService *project.ProjectEmployeeService,
	projectPermissionService *project.ProjectPermissionService,
	clk clock.Clock,
) *Handler {
	if clk == nil {
		clk = clock.New()
	}
	return &Handler{
		logger:                   slog.Default(),
		projectService:           projectService,
		payrateService:           payrateService,
		timesheetService:         timesheetService,
		projectEmployeeService:   projectEmployeeService,
		projectPermissionService: projectPermissionService,
		clock:                    clk,
	}
}

// Helper function to build comprehensive project response with related data
func (h *Handler) buildDetailedProjectResponse(ctx context.Context, project *domain.Project) (*dto.ProjectDetailedResponse, error) {
	response := &dto.ProjectDetailedResponse{
		ID:                   project.ID,
		ClientName:           project.ClientName,
		Name:                 project.Name,
		Code:                 project.Code,
		StartDate:            timeToStringPtr(project.StartDate),
		EndDate:              timeToStringPtr(project.EndDate),
		SalaryPeriodFrom:     project.SalaryPeriodFrom,
		SalaryPeriodTo:       project.SalaryPeriodTo,
		OffDays:              project.OffDays,
		Status:               string(project.ProjectStatus),
		IsFlexible:           project.IsFlexible,
		GeofenceGates:        geofenceGatesToDTO(project.GeofenceGates),
		GeofenceRadiusMeters: project.GeofenceRadiusMeters,
		ShiftNames:           shiftNamesToDTO(project.ShiftNames),

		CreatedBy: project.CreatedBy,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}

	// Get employee assignments if service is available
	if h.projectEmployeeService != nil {
		activeFilters := domain.ProjectEmployeeFilters{
			ProjectID:  &project.ID,
			ActiveOnly: true,
		}

		// Use CountAssignments for accurate total (no Limit cap)
		activeEmployees, countErr := h.projectEmployeeService.CountAssignments(ctx, activeFilters)

		// Fetch assignments for position counts and recent employees
		listFilters := domain.ProjectEmployeeFilters{
			ProjectID:  &project.ID,
			ActiveOnly: true,
			Limit:      1000,
			Offset:     0,
			SortBy:     "created_at",
			SortOrder:  "desc",
		}

		if assignments, err := h.projectEmployeeService.ListAssignments(ctx, listFilters); err == nil && countErr == nil {
			positionCounts := make(map[string]int)
			var recentEmployees []dto.ProjectRecentEmployee

			// Deduplicate by employee — count position only from latest assignment
			seenEmployees := make(map[uint]bool)
			for i, assignment := range assignments {
				if seenEmployees[assignment.EmployeeID] {
					continue
				}
				seenEmployees[assignment.EmployeeID] = true
				positionCounts[assignment.Position]++

				if i < 5 {
					recentEmployees = append(recentEmployees, dto.ProjectRecentEmployee{
						EmployeeID: assignment.EmployeeID,
						FullName:   assignment.EmployeeName,
						Position:   assignment.Position,
						StartDate:  assignment.StartDate.Format("2006-01-02"),
					})
				}
			}

			var positions []dto.ProjectPositionCount
			for position, count := range positionCounts {
				positions = append(positions, dto.ProjectPositionCount{
					Position: position,
					Count:    count,
				})
			}

			response.EmployeeAssignments = &dto.ProjectEmployeeAssignments{
				TotalEmployees:  int(activeEmployees),
				Positions:       positions,
				RecentEmployees: recentEmployees,
			}
		}
	}

	// Get current payrate if service is available
	if h.payrateService != nil {
		if payrate, err := h.payrateService.GetActivePayrateByProjectAndDate(ctx, project.ID, h.clock.Now()); err == nil {
			var toDateStr *string
			if payrate.ToDate != nil {
				dateStr := payrate.ToDate.Format("2006-01-02")
				toDateStr = &dateStr
			}

			// Parse payrate configuration
			config, _ := payrate.Payrate.Parse()

			response.CurrentPayrate = &dto.ProjectCurrentPayrate{
				ID:            payrate.ID,
				FromDate:      payrate.FromDate.Format("2006-01-02"),
				ToDate:        toDateStr,
				Configuration: config,
				Status:        "active", // Payrate status field no longer exists, using active default
			}
		}
	}

	// Get timesheet summary if service is available
	if h.timesheetService != nil && h.projectEmployeeService != nil {
		// First get list of currently working employees
		activeFilters := domain.ProjectEmployeeFilters{
			ProjectID:  &project.ID,
			ActiveOnly: true, // Show only active assignments (LastDate is null)
			Limit:      1000,
		}

		activeEmployees, err := h.projectEmployeeService.ListAssignments(ctx, activeFilters)
		if err != nil {
			// If we can't get currently working employees, skip timesheet summary
		} else {
			// Create map of currently working employee IDs
			activeEmployeeIDs := make(map[uint]bool)
			for _, emp := range activeEmployees {
				// Since we filtered by ActiveOnly, all these employees should be active
				activeEmployeeIDs[emp.EmployeeID] = true
			}

			// Get timesheets for the project
			now := h.clock.Now()
			monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

			filters := domain.TimesheetFilters{
				ProjectIDs: []uint{project.ID},
				FromDate:   &monthStart,
				ToDate:     &now,
				Limit:      1000,
			}

			if timesheets, err := h.timesheetService.ListTimesheets(ctx, filters); err == nil {
				totalTimesheets := 0
				pendingApproval := 0
				approvedCount := 0
				totalHours := 0.0
				totalAmount := int64(0)
				currentMonthHours := 0.0
				var lastEntryDate *string

				// Only include timesheets from currently working employees
				for _, ts := range timesheets {
					if !activeEmployeeIDs[ts.EmployeeID] {
						continue // Skip timesheets from non-working employees
					}

					totalTimesheets++

					switch ts.Status {
					case domain.TimesheetStatusPendingApproval:
						pendingApproval++
					case domain.TimesheetStatusApproved:
						approvedCount++
					}

					totalHours += ts.HoursWorked
					totalAmount += ts.Amount
					currentMonthHours += ts.HoursWorked

					// Update last entry date
					dateStr := ts.Date.Format("2006-01-02")
					if lastEntryDate == nil || dateStr > *lastEntryDate {
						lastEntryDate = &dateStr
					}
				}

				response.TimesheetSummary = &dto.ProjectTimesheetSummary{
					TotalTimesheets:   totalTimesheets,
					PendingApproval:   pendingApproval,
					ApprovedCount:     approvedCount,
					TotalHoursWorked:  totalHours,
					TotalAmountVND:    totalAmount,
					CurrentMonthHours: currentMonthHours,
					LastEntryDate:     lastEntryDate,
				}
			}
		}
	}

	// Enhanced financial summary
	totalRevenue := project.TotalReceivedVND + project.PendingReceivableVND
	totalExpenses := project.TotalPayoutVND + project.PendingPayableVND
	netProfit := totalRevenue - totalExpenses
	profitMargin := 0.0
	if totalRevenue > 0 {
		profitMargin = (netProfit / totalRevenue) * 100
	}

	response.FinancialSummary = &dto.ProjectFinancialSummary{
		TotalPayoutVND:       project.TotalPayoutVND,
		PendingPayableVND:    project.PendingPayableVND,
		PendingReceivableVND: project.PendingReceivableVND,
		TotalReceivedVND:     project.TotalReceivedVND,
		TotalRevenueVND:      totalRevenue,
		TotalExpensesVND:     totalExpenses,
		NetProfitVND:         netProfit,
		ProfitMarginPercent:  profitMargin,
	}

	// Add recent activity (simplified - could be enhanced with audit logs)
	var recentActivity []dto.ProjectRecentActivity
	recentActivity = append(recentActivity, dto.ProjectRecentActivity{
		Type:        "project_updated",
		Description: "Project information was updated",
		Date:        project.UpdatedAt,
	})

	response.RecentActivity = recentActivity

	return response, nil
}

// CreateProject creates a new project
// @Summary Create a new project
// @Description Create a new project with the provided information
// @Tags projects
// @Accept json
// @Produce json
// @Param project body dto.CreateProjectRequest true "Project creation data"
// @Success 201 {object} dto.ProjectResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects [post]
func (h *Handler) CreateProject(c *gin.Context) {
	var req dto.CreateProjectRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Parse dates if provided
	var startDate, endDate *time.Time

	// Set start_date to today if not provided
	if req.StartDate != nil && *req.StartDate != "" {
		parsedDate, err := time.Parse(timeutil.DateFormat, *req.StartDate)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidStartDateFormatVN)
			return
		}
		startDate = &parsedDate
	} else {
		// Default to today if start_date not provided
		today := timeutil.StartOfDay(h.clock.NowUTC())
		startDate = &today
	}

	if req.EndDate != nil && *req.EndDate != "" {
		parsedDate, err := time.Parse(timeutil.DateFormat, *req.EndDate)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidEndDateFormatVN)
			return
		}
		endDate = &parsedDate
	}

	// Policy: partners may set the salary period on CreateProject (relaxed from
	// admin-only on 2026-06-10). UpdateProject still gates salary-period changes
	// to admins; if a partner PATCH sends a salary-period field, the whole
	// request is rejected with 403.
	if err := validateSalaryPeriodDay("salary_period_from", req.SalaryPeriodFrom); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := validateSalaryPeriodDay("salary_period_to", req.SalaryPeriodTo); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Set project status - default to draft if not provided
	projectStatus := domain.ProjectStatusDraft
	if req.Status != nil {
		projectStatus = domain.ProjectStatus(*req.Status)
	}

	// Create project model
	project := &domain.Project{
		ClientName:       req.ClientName,
		Name:             req.Name,
		Code:             req.Code,
		StartDate:        startDate,
		EndDate:          endDate,
		SalaryPeriodFrom: domain.NormalizeSalaryPeriodDay(req.SalaryPeriodFrom),
		SalaryPeriodTo:   domain.NormalizeSalaryPeriodDay(req.SalaryPeriodTo),
		ProjectStatus:    projectStatus,
		CreatedBy:        userID.(uint),
	}
	if req.OffDays != nil {
		project.OffDays = *req.OffDays
	} else {
		project.OffDays = 0 // default: no fixed off days
	}

	// Flexible project configuration
	if req.IsFlexible != nil {
		project.IsFlexible = *req.IsFlexible
	}
	if req.GeofenceGates != nil {
		gates := make([]domain.GeofenceGate, len(req.GeofenceGates))
		for i, g := range req.GeofenceGates {
			gates[i] = domain.GeofenceGate{Name: g.Name, Lat: g.Lat, Lng: g.Lng}
		}
		project.GeofenceGates = gates
	}
	if req.GeofenceRadiusMeters != nil {
		project.GeofenceRadiusMeters = *req.GeofenceRadiusMeters
	} else if project.IsFlexible {
		project.GeofenceRadiusMeters = 100 // sensible default
	}
	if project.IsFlexible {
		if err := project.ValidateGeofenceGates(); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	// Admin-named shifts (labels for payrate shifts). On create we accept the
	// shape only; the range↔payrate match is validated on update once the
	// payrate exists.
	if req.ShiftNames != nil {
		project.ShiftNames = shiftNamesFromDTO(req.ShiftNames)
		if err := project.ValidateShiftNames(nil); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	createdProject, err := h.projectService.CreateProject(c.Request.Context(), project, userID.(uint))
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	projectResponse := projectToResponse(createdProject)

	response.SuccessCreated(c, projectResponse, constants.MsgProjectCreatedSuccessfullyVN)
}

// GetProject retrieves a project by ID with comprehensive information
// @Summary Get a project by ID with comprehensive information
// @Description Get comprehensive project information including employee assignments, payrates, timesheets, and financial summary
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 200 {object} dto.ProjectDetailedResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id} [get]
func (h *Handler) GetProject(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	// Get basic project info first
	project, err := h.projectService.GetProject(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// If comprehensive services are available, build detailed response
	if h.timesheetService != nil || h.projectEmployeeService != nil {
		detailedResponse, err := h.buildDetailedProjectResponse(c.Request.Context(), project)
		if err != nil {
			response.InternalServerError(c, constants.MsgFailedToBuildComprehensiveResponseVN)
			return
		}
		response.Success(c, detailedResponse, constants.MsgProjectDetailsRetrievedSuccessfullyVN)
		return
	}

	// Fallback to basic response with employee count
	projectWithCount, err := h.projectService.GetProjectWithEmployeeCount(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	projectResponse := projectWithEmployeeCountToResponse(projectWithCount)
	response.Success(c, projectResponse, constants.MsgProjectRetrievedSuccessfullyVN)
}

// ListProjects lists all projects with pagination and filtering
// @Summary List projects
// @Description Get a list of projects with pagination and filtering options
// @Tags projects
// @Accept json
// @Produce json
// @Param status query []string false "Filter by status"
// @Param created_by query int false "Filter by creator user ID"
// @Param fromDate query string false "Filter by creation date from (YYYY-MM-DD)"
// @Param toDate query string false "Filter by creation date to (YYYY-MM-DD)"
// @Param search query string false "Search by project name, code, or client name"
// @Param page query int false "Page number" default(1)
// @Param pageSize query int false "Page size" default(100)
// @Param sort_by query string false "Sort by field" default(created_at)
// @Param sort_order query string false "Sort order (asc/desc)" default(desc)
// @Success 200 {object} dto.ListProjectsResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects [get]
func (h *Handler) ListProjects(c *gin.Context) {
	filters := domain.ProjectFilters{
		Limit:     100, // default
		Offset:    0,   // default
		SortBy:    "created_at",
		SortOrder: "desc",
	}

	// Handle page/pageSize pagination (new model)
	page := 1
	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	pageSize := 100
	if ps := c.Query("pageSize"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Convert to offset/limit for repository
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	if sortBy := c.Query("sort_by"); sortBy != "" {
		// Validate sortBy field to prevent SQL injection
		allowedSortFields := []string{
			"created_at", "updated_at", "name", "client_name", "code",
			"start_date", "end_date", "status",
		}
		validSortBy := false
		for _, field := range allowedSortFields {
			if sortBy == field {
				validSortBy = true
				break
			}
		}
		if validSortBy {
			filters.SortBy = sortBy
		}
	}

	if sortOrder := c.Query("sort_order"); sortOrder != "" {
		if sortOrder == "asc" || sortOrder == "desc" {
			filters.SortOrder = sortOrder
		}
	}

	// Handle search
	if search := c.Query("search"); search != "" {
		filters.Search = search
	}

	if status := c.QueryArray("status"); len(status) > 0 {
		filters.ProjectStatus = make([]domain.ProjectStatus, len(status))
		for i, s := range status {
			filters.ProjectStatus[i] = domain.ProjectStatus(s)
		}
	}

	if createdBy := c.Query("created_by"); createdBy != "" {
		if id, err := strconv.ParseUint(createdBy, 10, 32); err == nil {
			uid := uint(id)
			filters.CreatedBy = &uid
		}
	}

	// Handle date range filtering
	if fromDate := c.Query("fromDate"); fromDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", fromDate); err == nil {
			filters.FromDate = &parsedDate
		}
	}

	if toDate := c.Query("toDate"); toDate != "" {
		if parsedDate, err := time.Parse("2006-01-02", toDate); err == nil {
			// Set to end of day for inclusive filtering
			endOfDay := timeutil.EndOfDay(parsedDate)
			filters.ToDate = &endOfDay
		}
	}

	// Apply partner role filtering - show both created and shared projects
	userRole := c.GetString(constants.CtxUserRole)
	if userRole == string(domain.RolePartner) {
		userID, exists := c.Get(constants.CtxUserID)
		if !exists {
			response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
			return
		}
		if uid, ok := userID.(uint); ok {
			filters.AccessibleBy = &uid // Use AccessibleBy instead of CreatedBy
		} else {
			response.Forbidden(c, constants.MsgInvalidUserIDVN)
			return
		}
	}

	projects, err := h.projectService.ListProjectsWithEmployeeCount(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListProjectsVN)
		return
	}

	total, err := h.projectService.CountProjects(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountProjectsVN)
		return
	}

	var projectResponses []dto.ProjectResponse
	for _, project := range projects {
		projectResponses = append(projectResponses, projectWithEmployeeCountToResponse(project))
	}

	// Calculate pagination
	totalPages := (int(total) + pageSize - 1) / pageSize

	if len(projectResponses) == 0 {
		pagination := response.Pagination{
			Page:         page,
			PageSize:     pageSize,
			TotalPages:   totalPages,
			TotalRecords: int(total),
		}
		response.SuccessEmptyWithPagination(c, constants.MsgNoProjectsFoundVN, pagination)
		return
	}

	pagination := response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}

	message := constants.MsgProjectsRetrievedSuccessfullyVN
	if page > 1 {
		message = fmt.Sprintf(constants.MsgPageOfProjectsRetrievedVN, page)
	}

	response.SuccessWithPagination(c, projectResponses, message, pagination)
}

// UpdateProject updates an existing project
// @Summary Update a project
// @Description Update project information
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Param project body dto.UpdateProjectRequest true "Project update data"
// @Success 200 {object} dto.ProjectResponse
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id} [put]
func (h *Handler) UpdateProject(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	var req dto.UpdateProjectRequest
	if !helpers.BindJSON(c, &req) {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Get existing project
	project, err := h.projectService.GetProject(c.Request.Context(), id)
	if err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Access control handled by middleware (ProjectPermissionService.CanUserAccessProject)

	// Check if project status allows modifications
	if project.IsCompleted() || project.IsCancelled() {
		response.BadRequest(c, constants.MsgCannotModifyProjectWithStatusVN+": "+string(project.ProjectStatus))
		return
	}

	// Update fields
	if req.ClientName != nil {
		project.ClientName = *req.ClientName
	}
	if req.Name != nil {
		project.Name = *req.Name
	}
	if req.Code != nil {
		project.Code = *req.Code
	}
	if req.Description != nil {
		project.Description = *req.Description
	}
	if req.StartDate != nil {
		if *req.StartDate == "" {
			project.StartDate = nil
		} else {
			parsedDate, err := time.Parse(timeutil.DateFormat, *req.StartDate)
			if err != nil {
				response.BadRequest(c, constants.MsgInvalidStartDateFormatVN)
				return
			}
			project.StartDate = &parsedDate
		}
	}
	if req.EndDate != nil {
		if *req.EndDate == "" {
			project.EndDate = nil
		} else {
			parsedDate, err := time.Parse(timeutil.DateFormat, *req.EndDate)
			if err != nil {
				response.BadRequest(c, constants.MsgInvalidEndDateFormatVN)
				return
			}
			project.EndDate = &parsedDate
		}
	}
	if req.Status != nil {
		project.ProjectStatus = domain.ProjectStatus(*req.Status)
	}
	// Only admin can change salary period
	if req.SalaryPeriodFrom != nil || req.SalaryPeriodTo != nil {
		userRole := c.GetString(constants.CtxUserRole)
		if userRole != string(domain.RoleAdmin) {
			response.Forbidden(c, "Chỉ quản trị viên mới được thay đổi kỳ lương")
			return
		}
	}

	if req.SalaryPeriodFrom != nil {
		if err := validateSalaryPeriodDay("salary_period_from", req.SalaryPeriodFrom); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		project.SalaryPeriodFrom = domain.NormalizeSalaryPeriodDay(req.SalaryPeriodFrom)
	}
	if req.SalaryPeriodTo != nil {
		if err := validateSalaryPeriodDay("salary_period_to", req.SalaryPeriodTo); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		project.SalaryPeriodTo = domain.NormalizeSalaryPeriodDay(req.SalaryPeriodTo)
	}
	if req.OffDays != nil {
		project.OffDays = *req.OffDays
	}

	// Flexible project configuration
	if req.IsFlexible != nil {
		project.IsFlexible = *req.IsFlexible
	}

	// Geofence fields
	if req.GeofenceGates != nil {
		gates := make([]domain.GeofenceGate, len(req.GeofenceGates))
		for i, g := range req.GeofenceGates {
			gates[i] = domain.GeofenceGate{Name: g.Name, Lat: g.Lat, Lng: g.Lng}
		}
		project.GeofenceGates = gates
	}
	if req.GeofenceRadiusMeters != nil {
		project.GeofenceRadiusMeters = *req.GeofenceRadiusMeters
	}
	// Validate geofence when gates/radius are provided, or when IsFlexible is toggled on
	if req.GeofenceGates != nil || req.GeofenceRadiusMeters != nil || (req.IsFlexible != nil && *req.IsFlexible) {
		if project.IsFlexible {
			// Apply default radius if still zero (GORM sends Go zero value overriding DB default)
			if project.GeofenceRadiusMeters == 0 {
				project.GeofenceRadiusMeters = 100
			}
			if err := project.ValidateGeofenceGates(); err != nil {
				response.BadRequest(c, err.Error())
				return
			}
		}
	}

	// Admin-named shifts (labels for payrate shifts). When the request carries
	// shift_names, map them onto the project and validate against the payrate's
	// configured shift ranges so admins cannot name shifts the payrate lacks.
	if req.ShiftNames != nil {
		project.ShiftNames = shiftNamesFromDTO(req.ShiftNames)
		var payrateRanges []string
		if project.IsFlexible && len(project.ShiftNames) > 0 && h.payrateService != nil {
			if payrate, err := h.payrateService.GetActivePayrateByProjectAndDate(c.Request.Context(), project.ID, h.clock.Now()); err == nil && payrate != nil {
				if flattened, err := payrate.Payrate.Flatten(); err == nil {
					payrateRanges = attendance.ExtractShiftRanges(flattened)
				}
			}
		}
		if err := project.ValidateShiftNames(payrateRanges); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
	}

	if err := h.projectService.UpdateProject(c.Request.Context(), project, userID.(uint)); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	// Get updated project with employee count
	updatedProject, err := h.projectService.GetProjectWithEmployeeCount(c.Request.Context(), id)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToGetUpdatedProjectVN)
		return
	}

	projectResponse := projectWithEmployeeCountToResponse(updatedProject)

	response.Success(c, projectResponse, constants.MsgProjectUpdatedSuccessfullyVN)
}

// DeleteProject deletes a project
// @Summary Delete a project
// @Description Delete a project by ID
// @Tags projects
// @Accept json
// @Produce json
// @Param id path int true "Project ID"
// @Success 204 "No Content"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/{id} [delete]
func (h *Handler) DeleteProject(c *gin.Context) {
	id, ok := helpers.ParseIDParam(c, "id", constants.MsgInvalidProjectIDVN)
	if !ok {
		return
	}

	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	// Access control handled by middleware (ProjectPermissionService.CanUserAccessProject)

	if err := h.projectService.DeleteProject(c.Request.Context(), id, userID.(uint)); err != nil {
		response.HandleDomainError(c, err)
		return
	}

	response.NoContent(c)
}

// ActivateProjects manually triggers project activation for eligible draft projects
// @Summary Activate eligible projects
// @Description Manually trigger activation of draft projects whose start date has arrived or passed
// @Tags projects
// @Accept json
// @Produce json
// @Success 200 {object} dto.ProjectActivationResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Security ApiKeyAuth
// @Router /projects/activate [post]
func (h *Handler) ActivateProjects(c *gin.Context) {
	userID, exists := c.Get(constants.CtxUserID)
	if !exists {
		response.Forbidden(c, constants.MsgUserIDNotFoundInContextVN)
		return
	}

	err := h.projectService.AutoActivateProjects(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToActivateProjectsVN+": "+err.Error())
		return
	}

	activationResponse := dto.ProjectActivationResponse{
		Message:   constants.MsgProjectActivationCompletedVN,
		Timestamp: h.clock.Now(),
		UserID:    userID.(uint),
	}

	response.Success(c, activationResponse, constants.MsgProjectsActivatedSuccessfullyVN)
}
