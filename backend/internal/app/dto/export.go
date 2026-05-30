package dto

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ExportEmployeesRequest represents the request for exporting employees
type ExportEmployeesRequest struct {
	ProjectIDsStr string `form:"projectIds"` // Comma-separated project IDs, omit for all projects
	ProjectIDs    []uint `form:"-"`          // Parsed from ProjectIDsStr
}

// ParseProjectIDs parses comma-separated project IDs string into []uint
func (r *ExportEmployeesRequest) ParseProjectIDs() error {
	if r.ProjectIDsStr == "" {
		return nil
	}

	idStrs := strings.Split(r.ProjectIDsStr, ",")
	r.ProjectIDs = make([]uint, 0, len(idStrs))

	for _, idStr := range idStrs {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid project_id: %s", idStr)
		}
		r.ProjectIDs = append(r.ProjectIDs, uint(id))
	}

	return nil
}

// ExportTimesheetsRequest represents the request for exporting timesheets
type ExportTimesheetsRequest struct {
	FromDate      string   `form:"fromDate" binding:"required" time_format:"2006-01-02"`
	ToDate        string   `form:"toDate" binding:"required" time_format:"2006-01-02"`
	ProjectIDsStr string   `form:"project_ids"` // Comma-separated project IDs or repeated params
	EmployeeID    *uint    `form:"employee_id"` // Optional employee filter
	StatusStr     string   `form:"status"`      // Comma-separated status values
	ProjectIDs    []uint   `form:"-"`           // Parsed from ProjectIDsStr
	Statuses      []string `form:"-"`           // Parsed from StatusStr
}

// ParseProjectIDs parses comma-separated project IDs string into []uint
func (r *ExportTimesheetsRequest) ParseProjectIDs() error {
	if r.ProjectIDsStr == "" {
		return nil
	}

	idStrs := strings.Split(r.ProjectIDsStr, ",")
	r.ProjectIDs = make([]uint, 0, len(idStrs))

	for _, idStr := range idStrs {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid project_id: %s", idStr)
		}
		r.ProjectIDs = append(r.ProjectIDs, uint(id))
	}

	return nil
}

// ParseStatus parses comma-separated status values into []string
func (r *ExportTimesheetsRequest) ParseStatus() error {
	if r.StatusStr == "" {
		return nil
	}

	statusStrs := strings.Split(r.StatusStr, ",")
	r.Statuses = make([]string, 0, len(statusStrs))

	validStatuses := map[string]bool{
		"pending_approval": true,
		"approved":         true,
		"rejected":         true,
	}

	for _, statusStr := range statusStrs {
		statusStr = strings.TrimSpace(statusStr)
		if statusStr == "" {
			continue
		}
		if !validStatuses[statusStr] {
			return fmt.Errorf("invalid status: %s. Valid values are: pending_approval, approved, rejected", statusStr)
		}
		r.Statuses = append(r.Statuses, statusStr)
	}

	return nil
}

// ValidateDateRange validates that the date range is within acceptable limits
func (r *ExportTimesheetsRequest) ValidateDateRange() error {
	fromDate, err := time.Parse("2006-01-02", r.FromDate)
	if err != nil {
		return err
	}

	toDate, err := time.Parse("2006-01-02", r.ToDate)
	if err != nil {
		return err
	}

	// Check if from_date is before to_date
	if fromDate.After(toDate) {
		return fmt.Errorf("from_date must be before or equal to to_date")
	}

	// Check if date range is within 1 year
	oneYear := 365 * 24 * time.Hour
	if toDate.Sub(fromDate) > oneYear {
		return fmt.Errorf("date range cannot exceed 1 year")
	}

	return nil
}

// GetParsedDates returns parsed from and to dates
func (r *ExportTimesheetsRequest) GetParsedDates() (time.Time, time.Time, error) {
	fromDate, err := time.Parse("2006-01-02", r.FromDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	toDate, err := time.Parse("2006-01-02", r.ToDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return fromDate, toDate, nil
}

// PayrollReportRequest represents the request for exporting payroll report
type PayrollReportRequest struct {
	FromDate string `form:"fromDate" binding:"required" time_format:"2006-01-02"`
	ToDate   string `form:"toDate" binding:"required" time_format:"2006-01-02"`
}

// ValidateDateRange validates that the date range is within acceptable limits
func (r *PayrollReportRequest) ValidateDateRange() error {
	fromDate, err := time.Parse("2006-01-02", r.FromDate)
	if err != nil {
		return err
	}

	toDate, err := time.Parse("2006-01-02", r.ToDate)
	if err != nil {
		return err
	}

	// Check if from_date is before to_date
	if fromDate.After(toDate) {
		return fmt.Errorf("fromDate must be before or equal to toDate")
	}

	// Check if date range is within 1 year
	oneYear := 365 * 24 * time.Hour
	if toDate.Sub(fromDate) > oneYear {
		return fmt.Errorf("date range cannot exceed 1 year")
	}

	return nil
}

// GetParsedDates returns parsed from and to dates
func (r *PayrollReportRequest) GetParsedDates() (time.Time, time.Time, error) {
	fromDate, err := time.Parse("2006-01-02", r.FromDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	toDate, err := time.Parse("2006-01-02", r.ToDate)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return fromDate, toDate, nil
}

// PayrollReportByProjectRequest represents the request for exporting payroll report by project
type PayrollReportByProjectRequest struct {
	AtDate     string `form:"atDate" time_format:"2006-01-02"`
	ProjectIDs string `form:"projectIds"`
	FromDate   string `form:"fromDate" time_format:"2006-01-02"`
	ToDate     string `form:"toDate" time_format:"2006-01-02"`
}

// IsExportByProjectRange returns true when date range params are provided
func (r *PayrollReportByProjectRequest) IsExportByProjectRange() bool {
	return r.FromDate != "" && r.ToDate != ""
}

// GetParsedProjectIDs parses comma-separated project IDs. Returns nil for "all projects".
func (r *PayrollReportByProjectRequest) GetParsedProjectIDs() ([]uint, error) {
	if r.ProjectIDs == "" {
		return nil, nil
	}
	parts := strings.Split(r.ProjectIDs, ",")
	ids := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid project ID: %s", p)
		}
		ids = append(ids, uint(id))
	}
	return ids, nil
}

// GetParsedDateRange returns parsed fromDate and toDate in local timezone
func (r *PayrollReportByProjectRequest) GetParsedDateRange() (time.Time, time.Time, error) {
	from, err := time.ParseInLocation("2006-01-02", r.FromDate, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid fromDate: %w", err)
	}
	to, err := time.ParseInLocation("2006-01-02", r.ToDate, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid toDate: %w", err)
	}
	return from, to, nil
}

// ValidateDate validates that the date is applicable for report generation
func (r *PayrollReportByProjectRequest) ValidateDate() error {
	atDate, err := time.Parse("2006-01-02", r.AtDate)
	if err != nil {
		return err
	}

	day := atDate.Day()
	if day > 10 && day < 24 {
		return fmt.Errorf("this date is not applicable to generate report")
	}

	return nil
}

// GetParsedDate returns parsed atDate in the process's local timezone.
// The MySQL DSN uses loc=Local, so all time.Time values sent to MySQL
// are formatted in time.Local. Creating dates in the same location
// prevents timezone-offset shifts that exclude boundary dates.
func (r *PayrollReportByProjectRequest) GetParsedDate() (time.Time, error) {
	atDate, err := time.ParseInLocation("2006-01-02", r.AtDate, time.Local)
	if err != nil {
		return time.Time{}, err
	}

	return atDate, nil
}
