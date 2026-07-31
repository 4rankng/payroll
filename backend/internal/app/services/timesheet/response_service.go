package timesheet

import (
	"context"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/timeutil"
	"api-server/internal/pkg/utils"
)

// TimesheetResponseService handles timesheet response transformations
type TimesheetResponseService struct {
	timesheetService *TimesheetService
}

// NewTimesheetResponseService creates a new timesheet response service
func NewTimesheetResponseService(timesheetService *TimesheetService) *TimesheetResponseService {
	return &TimesheetResponseService{
		timesheetService: timesheetService,
	}
}

// BuildTimesheetResponse builds a TimesheetResponse DTO from domain model
func (s *TimesheetResponseService) BuildTimesheetResponse(timesheet *domain.Timesheet) dto.TimesheetResponse {
	// Extract day type and hour type from paytype using domain service
	_, extractedDayType, extractedHourType := s.timesheetService.ParsePaytype(timesheet.PayType)

	return dto.TimesheetResponse{
		ID:               timesheet.ID,
		ProjectID:        timesheet.ProjectID,
		EmployeeID:       timesheet.EmployeeID,
		Date:             timesheet.Date.Format(timeutil.DateFormat),
		HoursWorked:      utils.RoundToTwoDecimals(timesheet.HoursWorked),
		PayType:          timesheet.PayType,
		HourType:         extractedHourType,
		DayType:          extractedDayType,
		PayrateID:        timesheet.PayrateID,
		PayRate:          utils.RoundToTwoDecimals(float64(timesheet.PayRate)),
		Amount:           utils.RoundToTwoDecimals(float64(timesheet.Amount)),
		Status:           string(timesheet.Status),
		PaymentStatus:    string(timesheet.PaymentStatus),
		PaymentReference: timesheet.PaymentReference,
		ForcePayroll:     timesheet.ForcePayroll,
		AllowedEdit:      timesheet.AllowedEdit,
		RequestEditID:    timesheet.RequestEditID,
		PaidAmount:       timesheet.PaidAmount,
		PaidAt:           timesheet.PaidAt,
		CreatedBy:        timesheet.CreatedBy,
		ApprovedBy:       timesheet.ApprovedBy,
		ApprovedAt:       timesheet.ApprovedAt,
		RejectionReason:  timesheet.RejectionReason,
		CreatedAt:        timesheet.CreatedAt,
		UpdatedAt:        timesheet.UpdatedAt,
	}
}

// BuildTimesheetWithDetailsResponse builds a TimesheetWithDetailsResponse DTO
func (s *TimesheetResponseService) BuildTimesheetWithDetailsResponse(timesheet *domain.Timesheet) dto.TimesheetWithDetailsResponse {
	response := dto.TimesheetWithDetailsResponse{
		TimesheetResponse: s.BuildTimesheetResponse(timesheet),
	}

	// Add project and employee details if available from preloaded relationships
	if timesheet.Project.ID != 0 {
		response.ProjectName = timesheet.Project.Name
		response.ProjectIsFlexible = timesheet.Project.IsFlexible
	}
	if timesheet.Employee.ID != 0 {
		response.EmployeeName = timesheet.Employee.Fullname
		response.EmployeeCode = timesheet.Employee.CCCD // Use CCCD as employee code for now
	}

	return response
}

// ParseTimesheetFilters parses query parameters into TimesheetFilters using business rules
func (s *TimesheetResponseService) ParseTimesheetFilters(ctx context.Context, params map[string]interface{}) domain.TimesheetFilters {
	filters := domain.TimesheetFilters{
		Limit:     20, // default pageSize
		Offset:    0,  // default
		SortBy:    "date",
		SortOrder: "desc",
	}

	// Apply business rules for pagination
	if pageSize, ok := params["pageSize"].(int); ok && pageSize > 0 {
		filters.Limit = pageSize
	}

	if page, ok := params["page"].(int); ok && page > 0 {
		filters.Offset = (page - 1) * filters.Limit
	}

	// Apply business rules for sorting
	if sortBy, ok := params["sortBy"].(string); ok && sortBy != "" {
		filters.SortBy = sortBy
	}

	if sortOrder, ok := params["sortOrder"].(string); ok && sortOrder != "" {
		filters.SortOrder = sortOrder
	}

	// Apply business rules for filtering
	if projectIDs, ok := params["projectIDs"].([]uint); ok && len(projectIDs) > 0 {
		filters.ProjectIDs = projectIDs
	}

	if employeeID, ok := params["employeeID"].(*uint); ok && employeeID != nil {
		filters.EmployeeID = employeeID
	}

	// Apply business rules for status filtering
	if statusArray, ok := params["status"].([]string); ok && len(statusArray) > 0 {
		var timesheetStatuses []domain.TimesheetStatus
		var paymentStatuses []domain.PaymentStatus
		pendingPaymentRequested := false

		for _, s := range statusArray {
			switch s {
			case "pending_approval":
				timesheetStatuses = append(timesheetStatuses, domain.TimesheetStatusPendingApproval)
			case "pending_payment":
				pendingPaymentRequested = true
			case "approved":
				timesheetStatuses = append(timesheetStatuses, domain.TimesheetStatusApproved)
			case "rejected":
				timesheetStatuses = append(timesheetStatuses, domain.TimesheetStatusRejected)
			case "paid":
				paymentStatuses = append(paymentStatuses, domain.PaymentStatusPaid)
			case "failed":
				paymentStatuses = append(paymentStatuses, domain.PaymentStatusFailed)
			case "cancelled":
				paymentStatuses = append(paymentStatuses, domain.PaymentStatusCancelled)
			}
		}

		// pending_payment is a complete business cohort, not one axis of the
		// generic status filter. Keep it deterministic if an external client
		// accidentally combines it with other status tokens.
		if pendingPaymentRequested {
			pendingPaymentFilters := domain.NewPendingPaymentTimesheetFilters()
			timesheetStatuses = pendingPaymentFilters.TimesheetStatus
			paymentStatuses = pendingPaymentFilters.PaymentStatus
		}

		if len(timesheetStatuses) > 0 {
			filters.TimesheetStatus = timesheetStatuses
		}
		if len(paymentStatuses) > 0 {
			filters.PaymentStatus = paymentStatuses
		}
	}

	// Handle standalone payment_status parameter (e.g. for approved-but-not-paid filter)
	if paymentStatusStr, ok := params["paymentStatus"].(string); ok && paymentStatusStr != "" {
		ps := domain.PaymentStatus(paymentStatusStr)
		// Only apply if not already set by status array parsing
		if len(filters.PaymentStatus) == 0 {
			filters.PaymentStatus = []domain.PaymentStatus{ps}
		}
	}

	// Apply business rules for pay type filtering
	if payType, ok := params["paytype"].([]string); ok && len(payType) > 0 {
		filters.PayType = payType
	}

	// Apply business rules for pending approval filtering
	if pendingApproval, ok := params["pendingApproval"].(bool); ok && pendingApproval {
		filters.PendingApproval = true
	}

	// Apply business rules for date filtering
	if fromDate, ok := params["fromDate"].(*time.Time); ok && fromDate != nil {
		filters.FromDate = fromDate
	}

	if toDate, ok := params["toDate"].(*time.Time); ok && toDate != nil {
		filters.ToDate = toDate
	}

	if allowedEdit, ok := params["allowedEdit"].(*bool); ok && allowedEdit != nil {
		filters.AllowedEdit = allowedEdit
	}

	if requestEditID, ok := params["requestEditID"].(*uint); ok && requestEditID != nil {
		filters.RequestEditID = requestEditID
	}

	if hasRequestEdit, ok := params["hasRequestEdit"].(*bool); ok && hasRequestEdit != nil {
		filters.HasRequestEdit = hasRequestEdit
	}

	if search, ok := params["search"].(string); ok && search != "" {
		filters.Search = search
	}

	return filters
}
