package query_builders

import (
	"strings"
	"time"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

// TimesheetQueryBuilder handles complex query building for timesheet operations
type TimesheetQueryBuilder struct {
	*common.BaseQueryBuilder
}

// NewTimesheetQueryBuilder creates a new timesheet query builder
func NewTimesheetQueryBuilder(db *gorm.DB) *TimesheetQueryBuilder {
	return &TimesheetQueryBuilder{
		BaseQueryBuilder: common.NewBaseQueryBuilder(db, "date", "DESC", nil),
	}
}

// BuildListQuery builds a query for listing timesheets with filters
func (b *TimesheetQueryBuilder) BuildListQuery(filters domain.TimesheetFilters) *gorm.DB {
	query := b.DB.Model(&domain.Timesheet{})
	query = b.applyFilters(query, filters)
	query = b.ApplyListOptions(query, filters.SortBy, filters.SortOrder, filters.Limit, filters.Offset)
	return query
}

// BuildCountQuery builds a query for counting timesheets with filters
func (b *TimesheetQueryBuilder) BuildCountQuery(filters domain.TimesheetFilters) *gorm.DB {
	query := b.DB.Model(&domain.Timesheet{})
	return b.applyFilters(query, filters)
}

// BuildSummaryQuery builds a query for timesheet summary statistics
func (b *TimesheetQueryBuilder) BuildSummaryQuery(filters domain.TimesheetFilters) *gorm.DB {
	query := b.DB.Model(&domain.Timesheet{})
	return b.applyFilters(query, filters)
}

// BuildPaymentQuery builds a query for payment-related timesheet operations
func (b *TimesheetQueryBuilder) BuildPaymentQuery(filters domain.TimesheetFilters) *gorm.DB {
	query := b.DB.Model(&domain.Timesheet{})
	return b.applyFilters(query, filters)
}

// ApplyAccessControl applies partner-based access control to a query with optimized LEFT JOIN approach
func (b *TimesheetQueryBuilder) ApplyAccessControl(query *gorm.DB, filters domain.TimesheetFilters) *gorm.DB {
	if filters.EmployeeCreatedBy == nil && filters.EmployeeAssignedByPartner == nil {
		return query
	}

	// JOIN employees table for access control filtering
	query = query.Joins("INNER JOIN employees e ON timesheets.employee_id = e.id AND e.deleted_at IS NULL")
	return b.ApplyPartnerAccessControl(query, filters)
}

// ApplyPartnerAccessControl applies partner-based access control JOINs and WHERE conditions.
// The caller is responsible for ensuring the employees table is joined as alias 'e'.
func (b *TimesheetQueryBuilder) ApplyPartnerAccessControl(query *gorm.DB, filters domain.TimesheetFilters) *gorm.DB {
	if filters.EmployeeCreatedBy == nil && filters.EmployeeAssignedByPartner == nil {
		return query
	}

	if filters.EmployeeCreatedBy != nil && filters.EmployeeAssignedByPartner != nil {
		partnerID := *filters.EmployeeCreatedBy
		query = query.Joins(`
			LEFT JOIN (
				SELECT DISTINCT employee_id, created_by as partner_id, 'project' as access_type
				FROM project_employees
				WHERE created_by = ? AND deleted_at IS NULL
				UNION ALL
				SELECT DISTINCT employee_id, user_id as partner_id, 'user' as access_type
				FROM employee_users
				WHERE deleted_at IS NULL
			) access ON access.employee_id = e.id AND (access.partner_id = ? OR e.created_by = ?)
		`, partnerID, partnerID, partnerID).
			Where("e.created_by = ? OR access.employee_id IS NOT NULL OR EXISTS (SELECT 1 FROM project_users pu WHERE pu.project_id = timesheets.project_id AND pu.user_id = ? AND pu.deleted_at IS NULL)", partnerID, partnerID)
	} else if filters.EmployeeCreatedBy != nil {
		partnerID := *filters.EmployeeCreatedBy
		query = query.Joins(`
			LEFT JOIN employee_users eu
			ON eu.employee_id = e.id
			AND eu.user_id = ?
			AND eu.deleted_at IS NULL
		`, partnerID).
			Where("e.created_by = ? OR eu.employee_id IS NOT NULL OR EXISTS (SELECT 1 FROM project_users pu WHERE pu.project_id = timesheets.project_id AND pu.user_id = ? AND pu.deleted_at IS NULL)", partnerID, partnerID)
	} else if filters.EmployeeAssignedByPartner != nil {
		partnerID := *filters.EmployeeAssignedByPartner
		query = query.Joins(`
			LEFT JOIN (
				SELECT employee_id, created_by as partner_id
				FROM project_employees
				WHERE created_by = ? AND deleted_at IS NULL
				UNION ALL
				SELECT employee_id, user_id as partner_id
				FROM employee_users
				WHERE user_id = ? AND deleted_at IS NULL
			) access ON access.employee_id = e.id
		`, partnerID, partnerID).
			Where("access.employee_id IS NOT NULL OR EXISTS (SELECT 1 FROM project_users pu WHERE pu.project_id = timesheets.project_id AND pu.user_id = ? AND pu.deleted_at IS NULL)", partnerID)
	}

	return query
}

// applyFilters applies timesheet filters to a query
func (b *TimesheetQueryBuilder) applyFilters(query *gorm.DB, filters domain.TimesheetFilters) *gorm.DB {
	// Use common FilterBuilder for standard filters
	query = b.GetFilterBuilder().ApplyStatus(query, &filters)
	query = b.GetFilterBuilder().ApplyDateRange(query, &filters)

	// Handle project filtering - support multiple project IDs (timesheet-specific)
	if len(filters.ProjectIDs) > 0 {
		query = query.Where("timesheets.project_id IN ?", filters.ProjectIDs)
	}

	// Employee filter (timesheet-specific)
	if filters.EmployeeID != nil {
		query = query.Where("timesheets.employee_id = ?", *filters.EmployeeID)
	}

	// Multiple employee filter
	if len(filters.EmployeeIDs) > 0 {
		query = query.Where("timesheets.employee_id IN ?", filters.EmployeeIDs)
	}

	// Payment status filter (timesheet-specific)
	if len(filters.PaymentStatus) > 0 {
		query = query.Where("timesheets.payment_status IN ?", filters.PaymentStatus)
	}

	// Pay type filter (timesheet-specific)
	if len(filters.PayType) > 0 {
		query = query.Where("timesheets.paytype IN ?", filters.PayType)
	}

	// Payrate filter (timesheet-specific)
	if filters.PayrateID != nil {
		query = query.Where("timesheets.payrate_id = ?", *filters.PayrateID)
	}

	// Creator filter (only if no access control)
	if filters.CreatedBy != nil && filters.EmployeeCreatedBy == nil {
		query = b.GetFilterBuilder().ApplyCreator(query, &filters)
	}

	if filters.ApprovedBy != nil {
		query = query.Where("timesheets.approved_by = ?", *filters.ApprovedBy)
	}

	// Join employees table once if needed for access control or search
	needsEmployeeJoin := filters.EmployeeCreatedBy != nil || filters.EmployeeAssignedByPartner != nil || filters.Search != ""
	if needsEmployeeJoin {
		query = query.Joins("INNER JOIN employees e ON timesheets.employee_id = e.id AND e.deleted_at IS NULL")
	}

	// Apply access control filters (employees table already joined above)
	if filters.EmployeeCreatedBy != nil || filters.EmployeeAssignedByPartner != nil {
		query = b.ApplyPartnerAccessControl(query, filters)
	}

	// Search filter on employee fullname or code
	if filters.Search != "" {
		search := "%" + strings.ToLower(filters.Search) + "%"
		query = query.Where("LOWER(e.fullname) LIKE ? OR LOWER(e.code) LIKE ?", search, search)
	}

	// Boolean filters (timesheet-specific)
	if filters.ForcePayroll != nil {
		if *filters.ForcePayroll {
			query = query.Where("timesheets.force_payroll = 1")
		} else {
			query = query.Where("timesheets.force_payroll = 0")
		}
	}

	if filters.AllowedEdit != nil {
		if *filters.AllowedEdit {
			query = query.Where("timesheets.allowed_edit = 1")
		} else {
			query = query.Where("timesheets.allowed_edit = 0")
		}
	}

	if filters.RequestEditID != nil {
		query = query.Where("timesheets.request_edit_id = ?", *filters.RequestEditID)
	}

	if filters.HasRequestEdit != nil {
		if *filters.HasRequestEdit {
			query = query.Where("timesheets.allowed_edit = 1 OR timesheets.request_edit_id IS NOT NULL")
		} else {
			query = query.Where("timesheets.allowed_edit = 0 AND timesheets.request_edit_id IS NULL")
		}
	}

	// Exact date filter - use direct date comparison for better index usage (timesheet-specific)
	if filters.Date != nil {
		startOfDay := time.Date(filters.Date.Year(), filters.Date.Month(), filters.Date.Day(), 0, 0, 0, 0, filters.Date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)
		query = query.Where("timesheets.date >= ? AND timesheets.date < ?", startOfDay, endOfDay)
	}

	return query
}

// BuildProjectEmployeeDateQuery builds a query for project-employee-date specific lookups
func (b *TimesheetQueryBuilder) BuildProjectEmployeeDateQuery(
	projectID, employeeID uint,
	date time.Time,
	paytype string,
) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ? AND date = ? AND paytype = ?",
			projectID, employeeID, date, paytype)
}

// BuildProjectEmployeeDateHourTypeQuery builds a query for project-employee-date-hour type lookups
func (b *TimesheetQueryBuilder) BuildProjectEmployeeDateHourTypeQuery(
	projectID, employeeID uint,
	date time.Time,
	hourType string,
) *gorm.DB {
	// Match exact hour type at the end of paytype (format: "position.dayType.hourType")
	// Escape SQL wildcard characters to prevent unintended pattern matching
	escapedHourType := strings.ReplaceAll(strings.ReplaceAll(hourType, "%", "\\%"), "_", "\\_")

	// Use direct date comparison for better index usage
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	return b.DB.Model(&domain.Timesheet{}).
		Where("project_id = ? AND employee_id = ? AND date >= ? AND date < ? AND paytype LIKE ?",
			projectID, employeeID, startOfDay, endOfDay, "%."+escapedHourType)
}

// BuildProjectEmployeeRangeQuery builds a query for project-employee date range lookups
func (b *TimesheetQueryBuilder) BuildProjectEmployeeRangeQuery(
	projectID, employeeID uint,
	fromDate, toDate time.Time,
) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).
		Where("timesheets.project_id = ? AND timesheets.employee_id = ? AND timesheets.date BETWEEN ? AND ?",
			projectID, employeeID, fromDate, toDate)
}

// BuildProjectRangeQuery builds a query for project date range lookups
func (b *TimesheetQueryBuilder) BuildProjectRangeQuery(
	projectID uint,
	fromDate, toDate time.Time,
) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).
		Where("timesheets.project_id = ? AND timesheets.date BETWEEN ? AND ?", projectID, fromDate, toDate)
}

// BuildEmployeeRangeQuery builds a query for employee date range lookups
func (b *TimesheetQueryBuilder) BuildEmployeeRangeQuery(
	employeeID uint,
	fromDate, toDate time.Time,
) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).
		Where("timesheets.employee_id = ? AND timesheets.date BETWEEN ? AND ?", employeeID, fromDate, toDate)
}

// BuildBulkUpdateQuery builds a query for bulk updates
func (b *TimesheetQueryBuilder) BuildBulkUpdateQuery(ids []uint) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).Where("id IN ?", ids)
}

// BuildBulkApproveQuery builds a query for bulk approval operations
func (b *TimesheetQueryBuilder) BuildBulkApproveQuery(ids []uint) *gorm.DB {
	return b.DB.Model(&domain.Timesheet{}).Where("id IN ?", ids)
}
