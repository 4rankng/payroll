package employee

import (
	"api-server/internal/constants"
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/app/services/attendance"
	"api-server/internal/app/services/audit"
	"api-server/internal/app/services/user"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"api-server/internal/pkg/utils"
)

type EmployeeProfileService struct {
	EmployeeRepo        domain.EmployeeRepository
	TimesheetRepo       domain.TimesheetRepository
	ProjectEmployeeRepo domain.ProjectEmployeeRepository
	PayrateRepo         domain.PayrateRepository
	AttendanceRepo      domain.AttendanceRepository
	UserService         *user.UserService
	EventBus            domain.EventBus
}

type CheckInTargetStatus string

const (
	CheckInTargetReady        CheckInTargetStatus = "ready"
	CheckInTargetUnavailable  CheckInTargetStatus = "unavailable"
	CheckInTargetAmbiguous    CheckInTargetStatus = "ambiguous"
	CheckInTargetMissingGates CheckInTargetStatus = "missing_gates"
)

type EmployeeScheduleInfo struct {
	PaymentSchedule             string
	CheckInEnabled              bool
	CheckInTargetStatus         CheckInTargetStatus
	CheckInTarget               *CheckInTargetInfo
	CheckInGeofenceRadiusMeters *uint
	// Advisory shift window for the frontend check-in button gate. Zero-value
	// (time.Time{}) when no shift is configured. Values are absolute instants in
	// the application timezone so the frontend can compare them against its
	// wall-clock without timezone ambiguity.
	ShiftStart           time.Time
	ShiftEnd             time.Time
	CheckInWindowStart   time.Time
	CheckInWindowEnd     time.Time
	CheckOutWindowStart  time.Time
	CheckOutWindowEnd    time.Time
	// ShiftName is the admin-chosen display name for the active shift's
	// time-range, looked up from the project's ShiftNames. Empty when unset.
	ShiftName            string
	ScheduleWindows      []ScheduleWindowInfo
	ActiveScheduleWindow *ScheduleWindowInfo
}

// ScheduleWindowInfo is advisory timing information for one configured shift.
// Values are absolute instants in the application timezone (Asia/Ho_Chi_Minh).
type ScheduleWindowInfo struct {
	ShiftStart          time.Time
	ShiftEnd            time.Time
	CheckInWindowStart  time.Time
	CheckInWindowEnd    time.Time
	CheckOutWindowStart time.Time
	CheckOutWindowEnd   time.Time
	// ShiftName is the admin-chosen display name for this shift's time-range.
	// Empty when no name is configured.
	ShiftName string
}

type CheckInTargetInfo struct {
	ProjectID    uint
	ProjectName  string
	RadiusMeters uint
	Gates        []domain.GeofenceGate
}

func NewEmployeeProfileService(
	employeeRepo domain.EmployeeRepository,
	timesheetRepo domain.TimesheetRepository,
	projectEmployeeRepo domain.ProjectEmployeeRepository,
	payrateRepo domain.PayrateRepository,
	attendanceRepo domain.AttendanceRepository,
	userService *user.UserService,
	eventBus domain.EventBus,
) *EmployeeProfileService {
	return &EmployeeProfileService{
		EmployeeRepo:        employeeRepo,
		TimesheetRepo:       timesheetRepo,
		ProjectEmployeeRepo: projectEmployeeRepo,
		PayrateRepo:         payrateRepo,
		AttendanceRepo:      attendanceRepo,
		UserService:         userService,
		EventBus:            eventBus,
	}
}

// GetMyProfile gets the employee profile for the authenticated user
func (s *EmployeeProfileService) GetMyProfile(ctx context.Context, userID uint) (*domain.Employee, error) {
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return employee, nil
}

// GetEmployeePaymentSchedule gets the payment schedule for an employee from their active project assignment
func (s *EmployeeProfileService) GetEmployeePaymentSchedule(ctx context.Context, employeeID uint) string {
	return s.GetEmployeeScheduleInfo(ctx, employeeID).PaymentSchedule
}

// GetEmployeeCheckInEnabled returns true if any active assignment for the employee has check-in enabled
func (s *EmployeeProfileService) GetEmployeeCheckInEnabled(ctx context.Context, employeeID uint) bool {
	return s.GetEmployeeScheduleInfo(ctx, employeeID).CheckInEnabled
}

// GetEmployeeScheduleInfo returns the employee's active payment/check-in context.
// The check-in target intentionally mirrors attendance auto-detection: if the
// employee has multiple active flexible projects, the target is ambiguous before
// we inspect check-in flags or gates. That prevents the app from drawing a map
// for one project when check-in would require project selection.
func (s *EmployeeProfileService) GetEmployeeScheduleInfo(ctx context.Context, employeeID uint) EmployeeScheduleInfo {
	info := EmployeeScheduleInfo{
		PaymentSchedule:     string(domain.PaymentScheduleWeekly),
		CheckInTargetStatus: CheckInTargetUnavailable,
	}

	assignments, err := s.ProjectEmployeeRepo.GetByEmployee(ctx, employeeID)
	if err != nil {
		return info
	}

	// Single pass: collect both schedule and check-in from active assignments
	for _, assignment := range assignments {
		if assignment.LastDate == nil {
			if assignment.CheckInEnabled {
				info.CheckInEnabled = true
			}
			if assignment.PaymentSchedule == string(domain.PaymentScheduleFlexible) {
				info.PaymentSchedule = string(domain.PaymentScheduleFlexible)
			}
		}
	}

	// If no flexible schedule found, use first active assignment's schedule
	if info.PaymentSchedule == string(domain.PaymentScheduleWeekly) {
		for _, assignment := range assignments {
			if assignment.LastDate == nil && assignment.PaymentSchedule != "" {
				info.PaymentSchedule = assignment.PaymentSchedule
				break
			}
		}
	}

	s.populateCheckInTarget(ctx, employeeID, assignments, &info)
	return info
}

func (s *EmployeeProfileService) populateCheckInTarget(ctx context.Context, employeeID uint, assignments []*domain.ProjectEmployee, info *EmployeeScheduleInfo) {
	projects, err := s.ProjectEmployeeRepo.GetActiveProjectsForEmployee(ctx, employeeID)
	if err != nil {
		info.CheckInTargetStatus = CheckInTargetUnavailable
		return
	}

	var flexibleProjects []*domain.Project
	for _, project := range projects {
		if project != nil && project.IsFlexible {
			flexibleProjects = append(flexibleProjects, project)
		}
	}

	switch len(flexibleProjects) {
	case 0:
		info.CheckInTargetStatus = CheckInTargetUnavailable
		return
	case 1:
	default:
		info.CheckInTargetStatus = CheckInTargetAmbiguous
		return
	}

	project := flexibleProjects[0]
	var targetAssignment *domain.ProjectEmployee
	for _, assignment := range assignments {
		if assignment != nil && assignment.LastDate == nil && assignment.ProjectID == project.ID {
			targetAssignment = assignment
			break
		}
	}
	if targetAssignment == nil ||
		!targetAssignment.CheckInEnabled ||
		targetAssignment.PaymentSchedule != string(domain.PaymentScheduleFlexible) ||
		project.GeofenceRadiusMeters == 0 {
		info.CheckInTargetStatus = CheckInTargetUnavailable
		return
	}

	radius := project.GeofenceRadiusMeters
	info.CheckInGeofenceRadiusMeters = &radius
	if len(project.GeofenceGates) == 0 {
		info.CheckInTargetStatus = CheckInTargetMissingGates
		return
	}

	info.CheckInTargetStatus = CheckInTargetReady
	info.CheckInTarget = &CheckInTargetInfo{
		ProjectID:    project.ID,
		ProjectName:  project.Name,
		RadiusMeters: project.GeofenceRadiusMeters,
		Gates:        project.GeofenceGates,
	}

	// Resolve the advisory shift window for the frontend check-in button gate.
	// Reuses the same payrate-based shift resolution as AttendanceService.CheckIn
	// (resolveShifts → closestShift). Null/empty when no shift is configured —
	// the frontend treats empty as "no timing gate". The server's
	// validateCheckInWindow remains authoritative; this is informational only.
	s.populateShiftWindow(ctx, employeeID, project.ID, targetAssignment.Position, project.ShiftNames, info)
}

// populateShiftWindow resolves the employee's applicable shift today and the
// check-in and checkout windows, writing them to info. No-op (leaves fields
// empty) when the payrate is missing, malformed, or has no shift for the
// position. shiftNames carries admin-chosen display names keyed by time-range
// so each advisory window can be labelled for the employee card.
func (s *EmployeeProfileService) populateShiftWindow(ctx context.Context, employeeID, projectID uint, position string, shiftNames []domain.ShiftName, info *EmployeeScheduleInfo) {
	if s.PayrateRepo == nil || position == "" {
		return
	}
	payrates, err := s.PayrateRepo.GetByProject(ctx, projectID)
	if err != nil || len(payrates) == 0 {
		return
	}
	// Use the first (most recent) payrate — matches AttendanceService's pattern.
	payrate := payrates[0]
	flattened, err := payrate.Payrate.Flatten()
	if err != nil || len(flattened) == 0 {
		return
	}
	now := clock.Now()
	shiftStart, shiftEnd, checkInStart, checkInEnd, checkOutStart, checkOutEnd, ok := attendance.ResolveShiftWindows(flattened, position, now)
	if !ok {
		return
	}
	info.ShiftStart = shiftStart
	info.ShiftEnd = shiftEnd
	info.CheckInWindowStart = checkInStart
	info.CheckInWindowEnd = checkInEnd
	info.CheckOutWindowStart = checkOutStart
	info.CheckOutWindowEnd = checkOutEnd
	nameByRange := shiftNameMap(shiftNames)
	info.ScheduleWindows = formatScheduleWindows(attendance.ResolveAllShiftWindows(flattened, position, now, nameByRange))
	// Name the resolved active (closest) shift window so the employee card's
	// heading can show the admin label even when only the active window is read.
	info.ShiftName = nameByRange[shiftKey(shiftStart, shiftEnd)]

	if activeAttendance := s.findActiveAttendance(ctx, employeeID, projectID, now); activeAttendance != nil {
		activeShiftStart, activeShiftEnd, activeCheckInStart, activeCheckInEnd, activeCheckOutStart, activeCheckOutEnd, activeOK := attendance.ResolveShiftWindows(flattened, position, activeAttendance.CheckInTime)
		if activeOK {
			activeWindow := formatScheduleWindow(attendance.ShiftWindow{
				ShiftStart:          activeShiftStart,
				ShiftEnd:            activeShiftEnd,
				CheckInWindowStart:  activeCheckInStart,
				CheckInWindowEnd:    activeCheckInEnd,
				CheckOutWindowStart: activeCheckOutStart,
				CheckOutWindowEnd:   activeCheckOutEnd,
				Name:                nameByRange[shiftKey(activeShiftStart, activeShiftEnd)],
			})
			info.ActiveScheduleWindow = &activeWindow
		}
	}
}

// shiftNameMap converts a project's ShiftNames slice into a "HH:MM-HH:MM" ->
// name lookup. Returns nil when the slice is empty so ResolveAllShiftWindows
// skips map work entirely.
func shiftNameMap(names []domain.ShiftName) map[string]string {
	if len(names) == 0 {
		return nil
	}
	m := make(map[string]string, len(names))
	for _, n := range names {
		m[n.Range] = n.Name
	}
	return m
}

// shiftKey formats two absolute shift instants into the canonical
// "HH:MM-HH:MM" time-range key used to join payrate shifts and admin names.
func shiftKey(start, end time.Time) string {
	return start.Format("15:04") + "-" + end.Format("15:04")
}

func (s *EmployeeProfileService) findActiveAttendance(ctx context.Context, employeeID, projectID uint, now time.Time) *domain.Attendance {
	if s.AttendanceRepo == nil {
		return nil
	}
	fromDate := now.AddDate(0, 0, -1)
	attendances, err := s.AttendanceRepo.List(ctx, domain.AttendanceFilters{
		EmployeeID:           &employeeID,
		ProjectID:            &projectID,
		FromDate:             &fromDate,
		ToDate:               &now,
		UseCheckInTimeWindow: true,
		Limit:                10,
		SortBy:               "check_in_time",
		SortOrder:            "desc",
	})
	if err != nil {
		return nil
	}
	for _, attendance := range attendances {
		if attendance != nil && attendance.CheckOutTime == nil && attendance.SalaryRejectReason == nil {
			return attendance
		}
	}
	return nil
}

func formatScheduleWindows(windows []attendance.ShiftWindow) []ScheduleWindowInfo {
	if len(windows) == 0 {
		return nil
	}
	formatted := make([]ScheduleWindowInfo, 0, len(windows))
	for _, window := range windows {
		formatted = append(formatted, formatScheduleWindow(window))
	}
	return formatted
}

func formatScheduleWindow(window attendance.ShiftWindow) ScheduleWindowInfo {
	return ScheduleWindowInfo{
		ShiftStart:          window.ShiftStart,
		ShiftEnd:            window.ShiftEnd,
		CheckInWindowStart:  window.CheckInWindowStart,
		CheckInWindowEnd:    window.CheckInWindowEnd,
		CheckOutWindowStart: window.CheckOutWindowStart,
		CheckOutWindowEnd:   window.CheckOutWindowEnd,
		ShiftName:           window.Name,
	}
}

// UpdateMyProfile updates the employee's own profile (name, email, username)
func (s *EmployeeProfileService) UpdateMyProfile(ctx context.Context, userID uint, fullname *string, email *string, username *string) (*domain.Employee, error) {
	// Get employee and user
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Save a value copy before mutation for change detection
	originalEmployee := *employee

	user, err := s.UserService.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update employee fullname if provided
	if fullname != nil && *fullname != "" {
		sanitizedName := utils.ToVietnameseTitleCase(*fullname)
		employee.Fullname = sanitizedName
		user.Fullname = sanitizedName
	}

	// Update employee email if provided
	if email != nil {
		// Sanitize email before persistence (trim spaces, allow clearing)
		trimmedEmail := strings.TrimSpace(*email)
		if trimmedEmail == "" {
			employee.Email = nil
			user.Email = nil
		} else {
			employee.Email = &trimmedEmail
			user.Email = &trimmedEmail
		}
	}

	// Update username if provided and different
	if username != nil && *username != "" && user.Username != *username {
		// Check if username already exists
		exists, err := s.UserService.UserRepo.ExistsByUsername(ctx, *username)
		if err != nil {
			return nil, fmt.Errorf("failed to check username: %w", err)
		}
		if exists {
			return nil, domain.NewConflictError(constants.MsgUsernameAlreadyTakenVN)
		}

		user.Username = *username
	}

	// Update employee
	if err := s.EmployeeRepo.Update(ctx, employee); err != nil {
		return nil, fmt.Errorf("failed to update employee: %w", err)
	}

	// Update user
	if err := s.UserService.UserRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Publish domain event
	actorFullName := audit.GetActorFullName(ctx, s.UserService.UserRepo, userID)
	event := domain.NewEmployeeProfileUpdatedEvent(ctx, employee, userID, actorFullName, &originalEmployee)
	if err := s.EventBus.Publish(ctx, event); err != nil {
		// Log error but don't fail the operation
		logger := observability.GetLogger()
		logger.Error("Failed to publish EmployeeProfileUpdatedEvent", "error", err, "employee_id", employee.ID)
	}

	// Return updated employee
	return s.EmployeeRepo.GetByUserID(ctx, userID)
}

// UpdateMyPassword updates the employee's password
func (s *EmployeeProfileService) UpdateMyPassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	// Delegate to user service
	return s.UserService.ChangePassword(ctx, userID, currentPassword, newPassword, "", "")
}

// TimesheetEntry represents a timesheet entry for employee view
type TimesheetEntry struct {
	ID            uint                   `json:"id"`
	Date          time.Time              `json:"date"`
	Project       *TimesheetProjectInfo  `json:"project"`
	HoursWorked   float64                `json:"hours_worked"`
	Amount        int64                  `json:"amount"`
	Status        domain.TimesheetStatus `json:"timesheet_status"`
	PaymentStatus domain.PaymentStatus   `json:"payment_status"`
	PaidAmount    int64                  `json:"paid_amount"`
	PaymentDate   *time.Time             `json:"payment_date"`
	ApprovedAt    *time.Time             `json:"approved_at"`
	ApprovedBy    *TimesheetApproverInfo `json:"approved_by"`
	CreatedAt     time.Time              `json:"created_at"`
}

type TimesheetProjectInfo struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Code       string `json:"code"`
	ClientName string `json:"client_name"`
}

type TimesheetApproverInfo struct {
	ID       uint   `json:"id"`
	Fullname string `json:"fullname"`
}

// GetMyTimesheets gets timesheets for the authenticated employee
func (s *EmployeeProfileService) GetMyTimesheets(ctx context.Context, userID uint, filters domain.TimesheetFilters) ([]*TimesheetEntry, int64, error) {
	// Get employee
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	// Set employee filter
	filters.EmployeeID = &employee.ID

	// Get timesheets
	timesheets, err := s.TimesheetRepo.List(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get timesheets: %w", err)
	}

	// Get count
	count, err := s.TimesheetRepo.Count(ctx, filters)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count timesheets: %w", err)
	}

	// Convert to TimesheetEntry
	entries := make([]*TimesheetEntry, len(timesheets))
	for i, ts := range timesheets {
		entry := &TimesheetEntry{
			ID:            ts.ID,
			Date:          ts.Date,
			HoursWorked:   ts.HoursWorked,
			Amount:        ts.Amount,
			Status:        ts.Status,
			PaymentStatus: ts.PaymentStatus,
			PaidAmount:    ts.PaidAmount,
			PaymentDate:   ts.PaymentDate,
			ApprovedAt:    ts.ApprovedAt,
			CreatedAt:     ts.CreatedAt,
		}

		// Add project info
		if ts.Project != nil {
			entry.Project = &TimesheetProjectInfo{
				ID:         ts.Project.ID,
				Name:       ts.Project.Name,
				Code:       ts.Project.Code,
				ClientName: ts.Project.ClientName,
			}
		}

		// Add approver info
		if ts.ApprovedBy != nil {
			entry.ApprovedBy = &TimesheetApproverInfo{
				ID: *ts.ApprovedBy,
			}

			// Try to get approver fullname
			if ts.ApprovedUser != nil {
				entry.ApprovedBy.Fullname = ts.ApprovedUser.Fullname
			}
		}

		entries[i] = entry
	}

	return entries, count, nil
}

// EmployeeSummaryData represents summary statistics for an employee
type EmployeeSummaryData struct {
	WeeklySalary            int64                       `json:"weekly_salary"`
	WeeklyClockedHours      float64                     `json:"weekly_clocked_hours"`
	WeeklyPaidAmount        int64                       `json:"weekly_paid_amount"`
	TotalApprovedTimesheets int                         `json:"total_approved_timesheets"`
	TotalApprovedAmount     int64                       `json:"total_approved_amount"`
	TotalPaidTimesheets     int                         `json:"total_paid_timesheets"`
	TotalPaidAmount         int64                       `json:"total_paid_amount"`
	TotalPendingTimesheets  int                         `json:"total_pending_timesheets"`
	TotalPendingAmount      int64                       `json:"total_pending_amount"`
	CalculationPeriod       *CalculationPeriod          `json:"calculation_period"`
	LastPaymentDate         *time.Time                  `json:"last_payment_date"`
	CurrentProjects         []*CurrentProjectAssignment `json:"current_projects"`
}

type CalculationPeriod struct {
	Weeks    int    `json:"weeks"`
	FromDate string `json:"fromDate"`
	ToDate   string `json:"toDate"`
}

type CurrentProjectAssignment struct {
	ProjectID   uint      `json:"project_id"`
	ProjectName string    `json:"project_name"`
	ProjectCode string    `json:"project_code"`
	Position    string    `json:"position"`
	StartDate   time.Time `json:"start_date"`
}

// GetMySummary gets salary summary statistics for the authenticated employee
func (s *EmployeeProfileService) GetMySummary(ctx context.Context, userID uint, weeks int) (*EmployeeSummaryData, error) {
	// Get employee
	employee, err := s.EmployeeRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Calculate date range
	now := clock.Now()
	fromDate := now.AddDate(0, 0, -weeks*7)
	toDate := now

	// Get approved timesheets
	approvedFilters := domain.TimesheetFilters{
		EmployeeID:      &employee.ID,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		FromDate:        &fromDate,
		ToDate:          &toDate,
		Limit:           10000,
	}

	approvedTimesheets, err := s.TimesheetRepo.List(ctx, approvedFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get approved timesheets: %w", err)
	}

	// Get paid timesheets
	paidFilters := domain.TimesheetFilters{
		EmployeeID:      &employee.ID,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusApproved},
		PaymentStatus:   []domain.PaymentStatus{domain.PaymentStatusPaid},
		FromDate:        &fromDate,
		ToDate:          &toDate,
		Limit:           10000,
	}

	paidTimesheets, err := s.TimesheetRepo.List(ctx, paidFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get paid timesheets: %w", err)
	}

	// Get pending timesheets
	pendingFilters := domain.TimesheetFilters{
		EmployeeID:      &employee.ID,
		TimesheetStatus: []domain.TimesheetStatus{domain.TimesheetStatusPendingApproval, domain.TimesheetStatusRejected},
		FromDate:        &fromDate,
		ToDate:          &toDate,
		Limit:           10000,
	}

	pendingTimesheets, err := s.TimesheetRepo.List(ctx, pendingFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending timesheets: %w", err)
	}

	// Calculate totals
	var totalApprovedAmount int64
	var totalApprovedHours float64
	var totalPaidAmount int64
	var totalPendingAmount int64
	var lastPaymentDate *time.Time

	for _, ts := range approvedTimesheets {
		totalApprovedAmount += ts.Amount
		totalApprovedHours += ts.HoursWorked
	}

	for _, ts := range paidTimesheets {
		totalPaidAmount += ts.PaidAmount
		if ts.PaymentDate != nil && (lastPaymentDate == nil || ts.PaymentDate.After(*lastPaymentDate)) {
			lastPaymentDate = ts.PaymentDate
		}
	}

	for _, ts := range pendingTimesheets {
		totalPendingAmount += ts.Amount
	}

	// Calculate weekly averages
	weeklySalary := int64(0)
	weeklyHours := 0.0
	weeklyPaidAmount := int64(0)

	if weeks > 0 {
		weeklySalary = totalApprovedAmount / int64(weeks)
		weeklyHours = totalApprovedHours / float64(weeks)
		weeklyPaidAmount = totalPaidAmount / int64(weeks)
	}

	// Get current project assignments
	currentProjects, err := s.getCurrentProjects(ctx, employee.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current projects: %w", err)
	}

	summary := &EmployeeSummaryData{
		WeeklySalary:            weeklySalary,
		WeeklyClockedHours:      weeklyHours,
		WeeklyPaidAmount:        weeklyPaidAmount,
		TotalApprovedTimesheets: len(approvedTimesheets),
		TotalApprovedAmount:     totalApprovedAmount,
		TotalPaidTimesheets:     len(paidTimesheets),
		TotalPaidAmount:         totalPaidAmount,
		TotalPendingTimesheets:  len(pendingTimesheets),
		TotalPendingAmount:      totalPendingAmount,
		CalculationPeriod: &CalculationPeriod{
			Weeks:    weeks,
			FromDate: fromDate.Format("2006-01-02"),
			ToDate:   toDate.Format("2006-01-02"),
		},
		LastPaymentDate: lastPaymentDate,
		CurrentProjects: currentProjects,
	}

	return summary, nil
}

// getCurrentProjects gets current active project assignments for an employee
func (s *EmployeeProfileService) getCurrentProjects(ctx context.Context, employeeID uint) ([]*CurrentProjectAssignment, error) {
	// Get current project assignments
	assignments, err := s.ProjectEmployeeRepo.GetByEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}

	currentProjects := make([]*CurrentProjectAssignment, 0)
	now := clock.Now()

	for _, assignment := range assignments {
		// Only include active assignments (no end date or end date in future)
		if assignment.LastDate == nil && assignment.StartDate.Before(now) {
			currentProjects = append(currentProjects, &CurrentProjectAssignment{
				ProjectID:   assignment.ProjectID,
				ProjectName: assignment.Project.Name,
				ProjectCode: assignment.Project.Code,
				Position:    assignment.Position,
				StartDate:   assignment.StartDate,
			})
		}
	}

	return currentProjects, nil
}
