package integration

// This file implements the read-only vendor timesheet surface
// (/api/v1/vendor/*), authenticated by an API key whose scope is
// domain.APIKeyScopeTimesheetRead.
//
// It lives in the integration package rather than a `vendor` subpackage
// because the repo's .gitignore excludes any directory named `vendor/`
// (meant for Go dependency vendoring), which would have silently dropped the
// handler from version control.
//
// Read-only is structural, not conventional: only GET routes are registered,
// every route passes middleware.RequireAPIKeyScope, and VendorTimesheetHandler
// holds references to list/read service methods exclusively. Adding a write
// path here would require deliberately changing those dependencies.

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/constants"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/transport/http/response"

	"github.com/gin-gonic/gin"
)

// Pagination bounds for the vendor surface. These are deliberately tighter
// than the internal UI endpoints: the vendor payload is full-fidelity
// (salary + PII), so an unbounded page would be a heavy dump.
const (
	defaultProjectsPageSize  = 20
	maxProjectsPageSize      = 100
	defaultEmployeesPageSize = 20
	maxEmployeesPageSize     = 100
	defaultTimesheetPageSize = 50
	maxTimesheetPageSize     = 200
)

// VendorTimesheetHandler serves the vendor read-only surface.
type VendorTimesheetHandler struct {
	timesheetService TimesheetReader
	projectRepo      domain.ProjectRepository
	employeeRepo     domain.EmployeeRepository
	projectEmployeeR ProjectEmployeeReader
}

// TimesheetReader is the timesheet surface this handler needs. It is an
// interface so the read-only guarantee is visible in the type: only list and
// count methods appear here, and adding a mutating capability requires
// changing this contract.
type TimesheetReader interface {
	ListTimesheets(ctx context.Context, filters domain.TimesheetFilters) ([]*domain.Timesheet, error)
	CountTimesheets(ctx context.Context, filters domain.TimesheetFilters) (int64, error)
}

// ProjectEmployeeReader exposes only the read needed to attach project codes
// to an employee row.
type ProjectEmployeeReader interface {
	GetActiveAssignmentsByEmployeeIDs(ctx context.Context, employeeIDs []uint) ([]*domain.ProjectEmployee, error)
}

// NewHandler constructs the vendor handler.
func NewVendorTimesheetHandler(
	timesheetService TimesheetReader,
	projectRepo domain.ProjectRepository,
	employeeRepo domain.EmployeeRepository,
	projectEmployeeR ProjectEmployeeReader,
) *VendorTimesheetHandler {
	return &VendorTimesheetHandler{
		timesheetService: timesheetService,
		projectRepo:      projectRepo,
		employeeRepo:     employeeRepo,
		projectEmployeeR: projectEmployeeR,
	}
}

// ListProjects returns the vendor's project vocabulary.
//
// GET /api/v1/vendor/projects?search=&page=&page_size=
func (h *VendorTimesheetHandler) ListProjects(c *gin.Context) {
	page, pageSize := parsePagination(c, defaultProjectsPageSize, maxProjectsPageSize)

	filters := domain.ProjectFilters{
		// The vendor surface has no calling user, so there is no
		// accessibility filter to apply; project visibility is governed by the
		// key's existence and its timesheet_read scope, not by a JWT identity.
		SkipAccessibilityFilter: true,
		Search:                  strings.TrimSpace(c.Query("search")),
		Limit:                   pageSize,
		Offset:                  (page - 1) * pageSize,
		SortBy:                  "name",
		SortOrder:               "asc",
	}

	projects, err := h.projectRepo.List(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	total, err := h.projectRepo.Count(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	items := make([]dto.VendorProjectItem, 0, len(projects))
	for _, p := range projects {
		items = append(items, dto.VendorProjectItem{
			Code:       p.Code,
			Name:       p.Name,
			ClientName: p.ClientName,
			Status:     string(p.ProjectStatus),
		})
	}

	response.SuccessWithPagination(c, items, constants.MsgVendorProjectsRetrievedVN, buildPagination(page, pageSize, total))
}

// ListEmployees returns the vendor's employee vocabulary, keyed by CCCD.
//
// GET /api/v1/vendor/employees?project_code=&search=&page=&page_size=
func (h *VendorTimesheetHandler) ListEmployees(c *gin.Context) {
	page, pageSize := parsePagination(c, defaultEmployeesPageSize, maxEmployeesPageSize)

	filters := domain.EmployeeFilters{
		SkipAccessibilityFilter: true,
		Search:                  strings.TrimSpace(c.Query("search")),
		Limit:                   pageSize,
		Offset:                  (page - 1) * pageSize,
		SortBy:                  "fullname",
		SortOrder:               "asc",
	}

	if code := strings.TrimSpace(c.Query("project_code")); code != "" {
		project, err := h.projectRepo.GetByCode(c.Request.Context(), code)
		if err != nil {
			h.respondCodeError(c, err, code)
			return
		}
		projectID := project.ID
		filters.ProjectID = &projectID
	}

	employees, err := h.employeeRepo.List(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	total, err := h.employeeRepo.Count(c.Request.Context(), filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	codesByEmployee, err := h.loadProjectCodes(c, employees)
	if err != nil {
		response.InternalServerError(c, constants.MsgInternalServerErrorVN)
		return
	}

	items := make([]dto.VendorEmployeeItem, 0, len(employees))
	for _, e := range employees {
		items = append(items, toVendorEmployeeItem(e, codesByEmployee[e.ID]))
	}

	response.SuccessWithPagination(c, items, constants.MsgVendorEmployeesRetrievedVN, buildPagination(page, pageSize, total))
}

// ListTimesheets returns timesheets filtered by project code and/or employee
// CCCD within a date range.
//
// At least one of project_code / cccd / from_date is required. With a
// full-fidelity payload, an unfiltered call would otherwise dump the entire
// company timesheet table — including salaries — in a single response.
//
// GET /api/v1/vendor/timesheets?project_code=&cccd=&from_date=&to_date=&status=&payment_status=&page=&page_size=
func (h *VendorTimesheetHandler) ListTimesheets(c *gin.Context) {
	ctx := c.Request.Context()

	projectCodes := splitMultiValue(c.QueryArray("project_code"))
	cccds := splitMultiValue(c.QueryArray("cccd"))
	fromDateRaw := strings.TrimSpace(c.Query("from_date"))
	toDateRaw := strings.TrimSpace(c.Query("to_date"))

	if len(projectCodes) == 0 && len(cccds) == 0 && fromDateRaw == "" {
		response.BadRequest(c, constants.MsgVendorFilterRequiredVN)
		return
	}

	filters := domain.TimesheetFilters{
		Limit:     0, // set after parsing page size
		SortBy:    "date",
		SortOrder: "desc",
	}

	// Resolve project codes to internal IDs. An unknown code is a 400, not a
	// 404: the request itself was well-formed, it just named a code that does
	// not exist. A 404 would suggest the endpoint is missing.
	for _, code := range projectCodes {
		project, err := h.projectRepo.GetByCode(ctx, code)
		if err != nil {
			h.respondCodeError(c, err, code)
			return
		}
		filters.ProjectIDs = append(filters.ProjectIDs, project.ID)
	}

	// Resolve CCCDs to internal IDs in one batch rather than N queries.
	if len(cccds) > 0 {
		employees, err := h.employeeRepo.ListByCCCDs(ctx, cccds)
		if err != nil {
			response.InternalServerError(c, constants.MsgInternalServerErrorVN)
			return
		}
		found := make(map[string]struct{}, len(employees))
		ids := make([]uint, 0, len(employees))
		for _, e := range employees {
			found[strings.TrimSpace(e.CCCD)] = struct{}{}
			ids = append(ids, e.ID)
		}
		// Name every unknown CCCD rather than failing on the first, so the
		// vendor can fix the whole batch in one round-trip.
		var missing []string
		for _, cccd := range cccds {
			if _, ok := found[cccd]; !ok {
				missing = append(missing, cccd)
			}
		}
		if len(missing) > 0 {
			response.BadRequest(c, fmt.Sprintf(constants.MsgVendorCCCDNotFoundVN, strings.Join(missing, ", ")))
			return
		}
		filters.EmployeeIDs = ids
	}

	// Parse dates in the Vietnam timezone. Using time.Parse here would drop
	// boundary days: a date parsed as UTC midnight compared against a UTC+7
	// column skips the first day of the range.
	if fromDateRaw != "" {
		d, err := time.ParseInLocation(timeutil.DateFormat, fromDateRaw, time.Local)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidFromDateFormatVN)
			return
		}
		from := timeutil.StartOfDay(d.UTC())
		filters.FromDate = &from
	}
	if toDateRaw != "" {
		d, err := time.ParseInLocation(timeutil.DateFormat, toDateRaw, time.Local)
		if err != nil {
			response.BadRequest(c, constants.MsgInvalidToDateFormatVN)
			return
		}
		// to_date is inclusive, so anchor it to the start of that day; the
		// repository's range comparison is date-based.
		to := timeutil.StartOfDay(d.UTC())
		filters.ToDate = &to
	}

	if statuses := splitMultiValue(c.QueryArray("status")); len(statuses) > 0 {
		for _, s := range statuses {
			filters.TimesheetStatus = append(filters.TimesheetStatus, domain.TimesheetStatus(s))
		}
	}
	if payments := splitMultiValue(c.QueryArray("payment_status")); len(payments) > 0 {
		for _, p := range payments {
			filters.PaymentStatus = append(filters.PaymentStatus, domain.PaymentStatus(p))
		}
	}

	page, pageSize := parsePagination(c, defaultTimesheetPageSize, maxTimesheetPageSize)
	filters.Limit = pageSize
	filters.Offset = (page - 1) * pageSize

	timesheets, err := h.timesheetService.ListTimesheets(ctx, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToListTimesheetsVN)
		return
	}

	total, err := h.timesheetService.CountTimesheets(ctx, filters)
	if err != nil {
		response.InternalServerError(c, constants.MsgFailedToCountTimesheetsVN)
		return
	}

	items := make([]dto.VendorTimesheetItem, 0, len(timesheets))
	for _, ts := range timesheets {
		items = append(items, toVendorTimesheetItem(ts))
	}

	response.SuccessWithPagination(c, items, constants.MsgVendorTimesheetsRetrievedVN, buildPagination(page, pageSize, total))
}

// respondCodeError maps an unknown project code to 400 and anything else to a
// 500, without leaking whether the backing store is reachable.
func (h *VendorTimesheetHandler) respondCodeError(c *gin.Context, err error, code string) {
	if domain.IsNotFoundError(err) {
		response.BadRequest(c, fmt.Sprintf(constants.MsgVendorProjectCodeNotFoundVN, code))
		return
	}
	response.InternalServerError(c, constants.MsgInternalServerErrorVN)
}

// loadProjectCodes resolves each employee's assigned project codes in one
// query, so the vendor can relate employees to projects without an extra call.
func (h *VendorTimesheetHandler) loadProjectCodes(c *gin.Context, employees []*domain.Employee) (map[uint][]string, error) {
	ids := make([]uint, 0, len(employees))
	for _, e := range employees {
		ids = append(ids, e.ID)
	}
	if len(ids) == 0 {
		return map[uint][]string{}, nil
	}

	assignments, err := h.projectEmployeeR.GetActiveAssignmentsByEmployeeIDs(c.Request.Context(), ids)
	if err != nil {
		return nil, err
	}

	// De-duplicate project IDs per employee before resolving names.
	unique := make(map[uint]map[uint]struct{}, len(ids))
	projectIDs := make([]uint, 0)
	for _, a := range assignments {
		if _, ok := unique[a.EmployeeID]; !ok {
			unique[a.EmployeeID] = map[uint]struct{}{}
		}
		if _, seen := unique[a.EmployeeID][a.ProjectID]; !seen {
			unique[a.EmployeeID][a.ProjectID] = struct{}{}
			projectIDs = append(projectIDs, a.ProjectID)
		}
	}

	if len(projectIDs) == 0 {
		return map[uint][]string{}, nil
	}

	projects, err := h.projectRepo.GetByIDs(c.Request.Context(), projectIDs)
	if err != nil {
		return nil, err
	}

	out := make(map[uint][]string, len(unique))
	for employeeID, set := range unique {
		codes := make([]string, 0, len(set))
		for projectID := range set {
			if p, ok := projects[projectID]; ok {
				codes = append(codes, p.Code)
			}
		}
		out[employeeID] = codes
	}
	return out, nil
}

func toVendorEmployeeItem(e *domain.Employee, projectCodes []string) dto.VendorEmployeeItem {
	if projectCodes == nil {
		projectCodes = []string{}
	}

	item := dto.VendorEmployeeItem{
		CCCD:              e.CCCD,
		Fullname:          e.Fullname,
		Mobile:            e.Mobile,
		Email:             e.Email,
		Address:           e.Address,
		BankID:            e.BankID,
		BankAccountName:   e.BankAccountName,
		BankAccountNumber: e.BankAccountNumber,
		BankAccountStatus: e.BankAccountStatus,
		ProjectCodes:      projectCodes,
	}
	if e.DateOfBirth != nil {
		formatted := e.DateOfBirth.Format(timeutil.DateFormat)
		item.DateOfBirth = &formatted
	}
	if e.BankAccountInvalidReason != nil {
		reason := *e.BankAccountInvalidReason
		item.BankAccountInvalidReason = &reason
	}
	return item
}

func toVendorTimesheetItem(ts *domain.Timesheet) dto.VendorTimesheetItem {
	item := dto.VendorTimesheetItem{
		Date:          ts.Date.Format(timeutil.DateFormat),
		HoursWorked:   ts.HoursWorked,
		PayType:       ts.PayType,
		PayRate:       ts.PayRate,
		Amount:        ts.Amount,
		PaidAmount:    ts.PaidAmount,
		Status:        string(ts.Status),
		PaymentStatus: string(ts.PaymentStatus),
	}

	if ts.Project != nil {
		item.ProjectCode = ts.Project.Code
	}
	if ts.Employee != nil {
		item.CCCD = ts.Employee.CCCD
		item.EmployeeName = ts.Employee.Fullname
	}
	if ts.PaymentReference != nil {
		ref := *ts.PaymentReference
		item.PaymentReference = &ref
	}
	if ts.PaidAt != nil {
		paidAt := ts.PaidAt.Format(time.RFC3339)
		item.PaidAt = &paidAt
	}
	if ts.RejectionReason != "" {
		reason := ts.RejectionReason
		item.RejectionReason = &reason
	}
	return item
}

// splitMultiValue accepts both a repeated query parameter
// (?project_code=A&project_code=B) and a comma-separated one (?project_code=A,B),
// because vendor HTTP clients differ in which they emit.
func splitMultiValue(values []string) []string {
	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, part := range strings.Split(raw, ",") {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				out = append(out, trimmed)
			}
		}
	}
	return out
}

// parsePagination reads page/page_size, clamping page_size to max.
func parsePagination(c *gin.Context, defaultSize, maxSize int) (page, pageSize int) {
	page = 1
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			page = parsed
		}
	}

	pageSize = defaultSize
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			pageSize = parsed
		}
	}
	if pageSize > maxSize {
		pageSize = maxSize
	}
	return page, pageSize
}

func buildPagination(page, pageSize int, total int64) response.Pagination {
	totalPages := 0
	if pageSize > 0 {
		totalPages = (int(total) + pageSize - 1) / pageSize
	}
	return response.Pagination{
		Page:         page,
		PageSize:     pageSize,
		TotalPages:   totalPages,
		TotalRecords: int(total),
	}
}
