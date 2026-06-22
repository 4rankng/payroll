package persistence

import (
	"context"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

type TimesheetEditRequestRepository struct {
	*BaseRepository
}

func NewTimesheetEditRequestRepository(db *Database) domain.TimesheetEditRequestRepository {
	return &TimesheetEditRequestRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

func (r *TimesheetEditRequestRepository) Create(ctx context.Context, request *domain.TimesheetEditRequest) error {
	return r.SafeCreate(ctx, request)
}

func (r *TimesheetEditRequestRepository) GetByID(ctx context.Context, id uint) (*domain.TimesheetEditRequest, error) {
	var request domain.TimesheetEditRequest

	err := r.DB.WithContext(ctx).
		Preload("Timesheet").
		Preload("Timesheet.Project").
		Preload("Timesheet.Employee").
		Preload("RequestedUser").
		Preload("ApprovedUser").
		Preload("RejectedUser").
		Where("id = ?", id).
		First(&request).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.NewNotFoundError("edit request not found")
		}
		return nil, err
	}

	return &request, nil
}

func (r *TimesheetEditRequestRepository) Update(ctx context.Context, request *domain.TimesheetEditRequest) error {
	return r.SafeUpdate(ctx, request)
}

func (r *TimesheetEditRequestRepository) Delete(ctx context.Context, id uint) error {
	return r.SafeDelete(ctx, &domain.TimesheetEditRequest{}, id)
}

func (r *TimesheetEditRequestRepository) List(ctx context.Context, filters domain.TimesheetEditRequestFilters) ([]*domain.TimesheetEditRequest, error) {
	var requests []*domain.TimesheetEditRequest

	query := r.DB.WithContext(ctx).
		Joins("JOIN timesheets ON timesheet_edit_requests.timesheet_id = timesheets.id").
		Preload("Timesheet").
		Preload("Timesheet.Project").
		Preload("Timesheet.Employee").
		Preload("RequestedUser").
		Preload("ApprovedUser").
		Preload("RejectedUser")

	// Apply filters
	if len(filters.Status) > 0 {
		query = query.Where("status IN ?", filters.Status)
	}

	if filters.RequestedBy != nil {
		query = query.Where("requested_by = ?", *filters.RequestedBy)
	}

	if filters.TimesheetID != nil {
		query = query.Where("timesheet_id = ?", *filters.TimesheetID)
	}

	if filters.FromDate != nil {
		query = query.Where("timesheet_edit_requests.created_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("timesheet_edit_requests.created_at <= ?", *filters.ToDate)
	}

	// Only return edit requests tied to timesheets still editable or pending reset link
	query = query.Where("timesheets.allowed_edit = 1 OR timesheets.request_edit_id IS NOT NULL")

	// Sorting
	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "timesheet_edit_requests.created_at"
	} else if !strings.Contains(sortBy, ".") {
		sortBy = "timesheet_edit_requests." + sortBy
	}
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	query = query.Order(common.SanitizeSortColumn(sortBy, "timesheet_edit_requests.created_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC"))

	// Pagination
	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	err := query.Find(&requests).Error
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (r *TimesheetEditRequestRepository) Count(ctx context.Context, filters domain.TimesheetEditRequestFilters) (int64, error) {
	var count int64

	query := r.DB.WithContext(ctx).
		Model(&domain.TimesheetEditRequest{}).
		Joins("JOIN timesheets ON timesheet_edit_requests.timesheet_id = timesheets.id")

	// Apply filters
	if len(filters.Status) > 0 {
		query = query.Where("status IN ?", filters.Status)
	}

	if filters.RequestedBy != nil {
		query = query.Where("requested_by = ?", *filters.RequestedBy)
	}

	if filters.TimesheetID != nil {
		query = query.Where("timesheet_id = ?", *filters.TimesheetID)
	}

	if filters.FromDate != nil {
		query = query.Where("timesheet_edit_requests.created_at >= ?", *filters.FromDate)
	}

	if filters.ToDate != nil {
		query = query.Where("timesheet_edit_requests.created_at <= ?", *filters.ToDate)
	}

	query = query.Where("timesheets.allowed_edit = 1 OR timesheets.request_edit_id IS NOT NULL")

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

func (r *TimesheetEditRequestRepository) GetPendingForTimesheet(ctx context.Context, timesheetID uint) (*domain.TimesheetEditRequest, error) {
	var request domain.TimesheetEditRequest

	err := r.DB.WithContext(ctx).
		Where("timesheet_id = ?", timesheetID).
		Where("status = ?", domain.EditRequestStatusPending).
		First(&request).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No pending request found - not an error
		}
		return nil, err
	}

	return &request, nil
}

func (r *TimesheetEditRequestRepository) Approve(ctx context.Context, id uint, approvedBy uint) error {
	return r.DB.WithContext(ctx).
		Model(&domain.TimesheetEditRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      domain.EditRequestStatusApproved,
			"approved_by": approvedBy,
			"rejected_by": nil,
		}).Error
}

func (r *TimesheetEditRequestRepository) Reject(ctx context.Context, id uint, rejectedBy uint) error {
	return r.DB.WithContext(ctx).
		Model(&domain.TimesheetEditRequest{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      domain.EditRequestStatusRejected,
			"rejected_by": rejectedBy,
			"approved_by": nil,
		}).Error
}
