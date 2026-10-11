package persistence

import (
	"context"
	"errors"
	"strings"

	"api-server/internal/domain"
	"api-server/internal/infra/persistence/common"

	"gorm.io/gorm"
)

// ApprovalRequestRepository implements domain.ApprovalRequestRepository with
// GORM. Guarded transitions go through SaveTransition, a compare-and-swap on
// the status column so concurrent decisions can never both win.
type ApprovalRequestRepository struct {
	*BaseRepository
}

// NewApprovalRequestRepository creates the repository.
func NewApprovalRequestRepository(db *Database) domain.ApprovalRequestRepository {
	return &ApprovalRequestRepository{BaseRepository: NewBaseRepository(db)}
}

// isDuplicateEntry reports whether err is a unique-key violation (MySQL 1062).
func isDuplicateEntry(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "Duplicate entry") || strings.Contains(msg, "Error 1062")
}

// Create inserts a new request. A duplicate idempotency key surfaces as a
// CONFLICT domain error — "(type, cycle)" re-filings must be rejected, not
// silently duplicated.
func (r *ApprovalRequestRepository) Create(ctx context.Context, request *domain.ApprovalRequest) error {
	if err := r.SafeCreate(ctx, request); err != nil {
		if isDuplicateEntry(err) {
			return domain.NewConflictError(
				"Yêu cầu phê duyệt cho loại này và chu kỳ này đã tồn tại — không thể tạo trùng lặp").
				WithContext("idempotency_key", request.IdempotencyKey)
		}
		return err
	}
	return nil
}

// GetByID loads one request by id.
func (r *ApprovalRequestRepository) GetByID(ctx context.Context, id uint) (*domain.ApprovalRequest, error) {
	var request domain.ApprovalRequest
	if err := r.SafeFirst(ctx, &request, "id = ?", id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("không tìm thấy yêu cầu phê duyệt")
		}
		return nil, err
	}
	return &request, nil
}

// GetByIdempotencyKey loads one request by its unique (type, cycle) key.
func (r *ApprovalRequestRepository) GetByIdempotencyKey(ctx context.Context, key string) (*domain.ApprovalRequest, error) {
	var request domain.ApprovalRequest
	if err := r.SafeFirst(ctx, &request, "idempotency_key = ?", key); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.NewNotFoundError("không tìm thấy yêu cầu phê duyệt")
		}
		return nil, err
	}
	return &request, nil
}

// applyFilters narrows a query by the filter struct.
func applyApprovalFilters(query *gorm.DB, filters domain.ApprovalRequestFilters) *gorm.DB {
	if len(filters.Status) > 0 {
		query = query.Where("status IN ?", filters.Status)
	}
	if len(filters.Type) > 0 {
		query = query.Where("request_type IN ?", filters.Type)
	}
	if filters.RequesterID != nil {
		query = query.Where("requester_id = ?", *filters.RequesterID)
	}
	return query
}

// List returns requests matching the filters, newest first by default.
func (r *ApprovalRequestRepository) List(ctx context.Context, filters domain.ApprovalRequestFilters) ([]*domain.ApprovalRequest, error) {
	var requests []*domain.ApprovalRequest

	query := applyApprovalFilters(r.dbForContext(ctx).Model(&domain.ApprovalRequest{}), filters)

	sortBy := filters.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := filters.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}
	query = query.Order(common.SanitizeSortColumn(sortBy, "created_at") + " " + common.SanitizeSortOrder(sortOrder, "DESC"))

	if filters.Limit > 0 {
		query = query.Limit(filters.Limit)
	}
	if filters.Offset > 0 {
		query = query.Offset(filters.Offset)
	}

	if err := query.Find(&requests).Error; err != nil {
		return nil, err
	}
	return requests, nil
}

// Count returns how many requests match the filters.
func (r *ApprovalRequestRepository) Count(ctx context.Context, filters domain.ApprovalRequestFilters) (int64, error) {
	var total int64
	query := applyApprovalFilters(r.dbForContext(ctx).Model(&domain.ApprovalRequest{}), filters)
	if err := query.Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// SaveTransition persists a guarded transition with a compare-and-swap on the
// status column. The write only lands when the row still carries one of
// expectedStatuses; otherwise a CONFLICT domain error is returned and the
// caller must reload and decide again.
func (r *ApprovalRequestRepository) SaveTransition(ctx context.Context, request *domain.ApprovalRequest, expectedStatuses []domain.ApprovalRequestStatus) error {
	if len(expectedStatuses) == 0 {
		return domain.NewValidationError("trạng thái kỳ vọng là bắt buộc")
	}

	updates := map[string]interface{}{
		"status":        request.Status,
		"approver_id":   request.ApproverID,
		"decision_note": request.DecisionNote,
		"outcome":       request.Outcome,
		"retry_count":   request.RetryCount,
		"scheduled_at":  request.ScheduledAt,
		"executed_at":   request.ExecutedAt,
		"payload":       request.Payload,
	}

	result := r.dbForContext(ctx).
		Model(&domain.ApprovalRequest{}).
		Where("id = ? AND status IN ?", request.ID, expectedStatuses).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.NewConflictError(
			"Yêu cầu phê duyệt đã được xử lý bởi thao tác khác — vui lòng tải lại dữ liệu").
			WithContext("approval_request_id", request.ID)
	}
	return nil
}
