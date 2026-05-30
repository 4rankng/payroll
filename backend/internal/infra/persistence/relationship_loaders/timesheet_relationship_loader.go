package relationship_loaders

import (
	"context"
	"time"

	"api-server/internal/domain"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

// TimesheetRelationshipLoader handles efficient batch loading of timesheet relationships
type TimesheetRelationshipLoader struct {
	db *gorm.DB
}

// NewTimesheetRelationshipLoader creates a new timesheet relationship loader
func NewTimesheetRelationshipLoader(db *gorm.DB) *TimesheetRelationshipLoader {
	return &TimesheetRelationshipLoader{
		db: db,
	}
}

// RelationshipLoadOptions controls which relationships are loaded for timesheets.
type RelationshipLoadOptions struct {
	LoadProject  bool
	LoadEmployee bool
	LoadUsers    bool // CreatedUser + ApprovedUser
}

// DefaultRelationshipLoadOptions returns options that load everything.
func DefaultRelationshipLoadOptions() RelationshipLoadOptions {
	return RelationshipLoadOptions{LoadProject: true, LoadEmployee: true, LoadUsers: true}
}

// LoadTimesheetRelationships efficiently loads all relationships for a collection of timesheets.
// Uses parallel loading and consolidated user queries to minimize database round-trips.
func (l *TimesheetRelationshipLoader) LoadTimesheetRelationships(ctx context.Context, timesheets []*domain.Timesheet, includeRelations bool) error {
	if len(timesheets) == 0 || !includeRelations {
		return nil
	}
	return l.LoadTimesheetRelationshipsWithOptions(ctx, timesheets, DefaultRelationshipLoadOptions())
}

// LoadTimesheetRelationshipsWithOptions loads only the specified relationships.
func (l *TimesheetRelationshipLoader) LoadTimesheetRelationshipsWithOptions(ctx context.Context, timesheets []*domain.Timesheet, opts RelationshipLoadOptions) error {
	if len(timesheets) == 0 {
		return nil
	}

	projectIDSet := make(map[uint]struct{})
	employeeIDSet := make(map[uint]struct{})
	userIDSet := make(map[uint]struct{})

	for _, ts := range timesheets {
		if opts.LoadProject {
			projectIDSet[ts.ProjectID] = struct{}{}
		}
		if opts.LoadEmployee {
			employeeIDSet[ts.EmployeeID] = struct{}{}
		}
		if opts.LoadUsers {
			if ts.CreatedBy > 0 {
				userIDSet[ts.CreatedBy] = struct{}{}
			}
			if ts.ApprovedBy != nil && *ts.ApprovedBy > 0 {
				userIDSet[*ts.ApprovedBy] = struct{}{}
			}
		}
	}

	projectIDs := setToSlice(projectIDSet)
	employeeIDs := setToSlice(employeeIDSet)
	allUserIDs := setToSlice(userIDSet)

	var userMap map[uint]*domain.User
	var projectMap map[uint]*domain.Project
	var employeeMap map[uint]*domain.Employee

	g, gCtx := errgroup.WithContext(ctx)

	if opts.LoadUsers {
		g.Go(func() error {
			var err error
			userMap, err = l.LoadUsers(gCtx, allUserIDs)
			return err
		})
	}

	if opts.LoadProject {
		g.Go(func() error {
			var err error
			projectMap, err = l.LoadProjects(gCtx, projectIDs)
			return err
		})
	}

	if opts.LoadEmployee {
		g.Go(func() error {
			var err error
			employeeMap, err = l.LoadEmployees(gCtx, employeeIDs)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	l.assignRelationships(timesheets, projectMap, employeeMap, userMap, userMap)
	return nil
}

// LoadProjects batch loads projects by IDs
func (l *TimesheetRelationshipLoader) LoadProjects(ctx context.Context, projectIDs []uint) (map[uint]*domain.Project, error) {
	projectMap := make(map[uint]*domain.Project)
	if len(projectIDs) == 0 {
		return projectMap, nil
	}

	var projects []*domain.Project
	err := l.db.WithContext(ctx).
		Preload("Creator").
		Where("id IN ?", projectIDs).
		Find(&projects).Error
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		projectMap[p.ID] = p
	}

	return projectMap, nil
}

// LoadEmployees batch loads employees by IDs
func (l *TimesheetRelationshipLoader) LoadEmployees(ctx context.Context, employeeIDs []uint) (map[uint]*domain.Employee, error) {
	employeeMap := make(map[uint]*domain.Employee)
	if len(employeeIDs) == 0 {
		return employeeMap, nil
	}

	var employees []*domain.Employee
	err := l.db.WithContext(ctx).
		Preload("Creator").
		Where("id IN ?", employeeIDs).
		Find(&employees).Error
	if err != nil {
		return nil, err
	}

	for _, e := range employees {
		employeeMap[e.ID] = e
	}

	return employeeMap, nil
}

// LoadUsers batch loads users by IDs
func (l *TimesheetRelationshipLoader) LoadUsers(ctx context.Context, userIDs []uint) (map[uint]*domain.User, error) {
	userMap := make(map[uint]*domain.User)
	if len(userIDs) == 0 {
		return userMap, nil
	}

	var users []*domain.User
	err := l.db.WithContext(ctx).
		Where("id IN ?", userIDs).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		userMap[u.ID] = u
	}

	return userMap, nil
}

// LoadSingleTimesheetRelationships loads relationships for a single timesheet
func (l *TimesheetRelationshipLoader) LoadSingleTimesheetRelationships(ctx context.Context, timesheet *domain.Timesheet) error {
	if timesheet == nil {
		return nil
	}

	// Load project
	_ = l.db.WithContext(ctx).
		Preload("Creator").
		First(&timesheet.Project, timesheet.ProjectID).Error

	// Load employee
	_ = l.db.WithContext(ctx).
		Preload("Creator").
		First(&timesheet.Employee, timesheet.EmployeeID).Error

	// Load created user
	if timesheet.CreatedBy > 0 {
		l.db.WithContext(ctx).First(&timesheet.CreatedUser, timesheet.CreatedBy)
	}

	// Load approved user
	if timesheet.ApprovedBy != nil && *timesheet.ApprovedBy > 0 {
		l.db.WithContext(ctx).First(&timesheet.ApprovedUser, *timesheet.ApprovedBy)
	}

	return nil
}

// assignRelationships assigns loaded relationships to timesheets
func (l *TimesheetRelationshipLoader) assignRelationships(
	timesheets []*domain.Timesheet,
	projectMap map[uint]*domain.Project,
	employeeMap map[uint]*domain.Employee,
	createdUserMap map[uint]*domain.User,
	approvedUserMap map[uint]*domain.User,
) {
	for _, ts := range timesheets {
		// Assign project
		if project, exists := projectMap[ts.ProjectID]; exists {
			ts.Project = project
		}

		// Assign employee
		if employee, exists := employeeMap[ts.EmployeeID]; exists {
			ts.Employee = employee
		}

		// Assign created user
		if ts.CreatedBy > 0 {
			if user, exists := createdUserMap[ts.CreatedBy]; exists {
				ts.CreatedUser = user
			}
		}

		// Assign approved user
		if ts.ApprovedBy != nil && *ts.ApprovedBy > 0 {
			if user, exists := approvedUserMap[*ts.ApprovedBy]; exists {
				ts.ApprovedUser = user
			}
		}
	}
}

// LoadProjectEmployees loads project employee relationships
func (l *TimesheetRelationshipLoader) LoadProjectEmployees(ctx context.Context, projectID, employeeID uint) (*domain.ProjectEmployee, error) {
	var pe domain.ProjectEmployee
	err := l.db.WithContext(ctx).
		Where("employee_id = ? AND project_id = ? AND deleted_at IS NULL", employeeID, projectID).
		Order("created_at DESC").
		First(&pe).Error
	return &pe, err
}

// LoadLatestProjectEmployeeForDate loads the latest project employee assignment for a specific date
func (l *TimesheetRelationshipLoader) LoadLatestProjectEmployeeForDate(ctx context.Context, projectID, employeeID uint, date time.Time) (*domain.ProjectEmployee, error) {
	var pe domain.ProjectEmployee
	err := l.db.WithContext(ctx).
		Where("employee_id = ? AND project_id = ? AND deleted_at IS NULL AND (end_date IS NULL OR end_date >= ?)", employeeID, projectID, date).
		Order("created_at DESC").
		First(&pe).Error
	return &pe, err
}

// setToSlice converts a map[uint]struct{} to a []uint slice.
func setToSlice(set map[uint]struct{}) []uint {
	slice := make([]uint, 0, len(set))
	for id := range set {
		slice = append(slice, id)
	}
	return slice
}
