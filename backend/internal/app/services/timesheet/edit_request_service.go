package timesheet

import (
	"api-server/internal/constants"
	"context"
	"fmt"

	"api-server/internal/app/services/infrastructure"
	"api-server/internal/domain"
	"api-server/internal/infra/observability"
	"gorm.io/gorm"
)

type TimesheetEditRequestService struct {
	editRequestRepo domain.TimesheetEditRequestRepository
	timesheetRepo   domain.TimesheetRepository
	auditLogRepo    domain.AuditLogRepository
	eventBus        domain.EventBus
	transactionMgr  *infrastructure.TransactionManager
}

func NewTimesheetEditRequestService(
	editRequestRepo domain.TimesheetEditRequestRepository,
	timesheetRepo domain.TimesheetRepository,
	auditLogRepo domain.AuditLogRepository,
	eventBus domain.EventBus,
	transactionManager *infrastructure.TransactionManager,
) *TimesheetEditRequestService {
	return &TimesheetEditRequestService{
		editRequestRepo: editRequestRepo,
		timesheetRepo:   timesheetRepo,
		auditLogRepo:    auditLogRepo,
		eventBus:        eventBus,
		transactionMgr:  transactionManager,
	}
}

// CreateEditRequest creates a new timesheet edit request
func (s *TimesheetEditRequestService) CreateEditRequest(ctx context.Context, timesheetID uint, requestedBy uint) (*domain.TimesheetEditRequest, error) {
	var result *domain.TimesheetEditRequest

	err := s.transactionMgr.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get timesheet and validate
		timesheet, err := s.timesheetRepo.GetByID(ctx, timesheetID)
		if err != nil {
			return err
		}

		// 2. Validate timesheet is approved
		if !timesheet.IsApproved() {
			return domain.NewValidationError(fmt.Sprintf("không thể yêu cầu chỉnh sửa: bảng chấm công chưa được phê duyệt (trạng thái hiện tại: %s)", timesheet.Status))
		}

		// 3. Validate payment status is pending (not paid/failed/cancelled)
		if timesheet.PaymentStatus != domain.PaymentStatusPending {
			return domain.NewValidationError(fmt.Sprintf("không thể yêu cầu chỉnh sửa: bảng chấm công đã được thanh toán hoặc hủy (trạng thái thanh toán: %s)", timesheet.PaymentStatus))
		}

		// 4. Check if there's already a pending request for this timesheet
		existingRequest, err := s.editRequestRepo.GetPendingForTimesheet(ctx, timesheetID)
		if err != nil {
			return err
		}
		if existingRequest != nil {
			return domain.NewValidationError(constants.MsgEditRequestAlreadyPendingVN)
		}

		// 5. Create the edit request
		editRequest := &domain.TimesheetEditRequest{
			TimesheetID: timesheetID,
			RequestedBy: requestedBy,
			Status:      domain.EditRequestStatusPending,
		}

		if err := editRequest.IsValid(); err != nil {
			return err
		}

		if err := s.editRequestRepo.Create(ctx, editRequest); err != nil {
			return err
		}

		// Link the pending request to the timesheet
		timesheet.RequestEditID = &editRequest.ID
		if err := s.timesheetRepo.Update(ctx, timesheet); err != nil {
			return err
		}

		// 6. Publish TimesheetEditRequestCreatedEvent for audit and notifications
		if s.eventBus != nil {
			event := domain.NewTimesheetEditRequestCreatedEvent(ctx, editRequest.TimesheetID, editRequest.RequestedBy, "")
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish TimesheetEditRequestCreatedEvent", "error", err)
			}
		}

		result = editRequest
		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// ApproveEditRequest approves an edit request and resets the timesheet
func (s *TimesheetEditRequestService) ApproveEditRequest(ctx context.Context, requestID uint, approvedBy uint) error {
	return s.transactionMgr.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get edit request
		editRequest, err := s.editRequestRepo.GetByID(ctx, requestID)
		if err != nil {
			return err
		}

		// 2. Validate it can be approved
		if !editRequest.CanBeApproved() {
			return domain.NewValidationError(fmt.Sprintf("không thể phê duyệt yêu cầu: trạng thái hiện tại là '%s'", editRequest.Status))
		}

		// 3. Get the timesheet
		timesheet, err := s.timesheetRepo.GetByID(ctx, editRequest.TimesheetID)
		if err != nil {
			return err
		}

		// 4. Reset timesheet to pending_approval and set allowed_edit flag
		timesheet.Status = domain.TimesheetStatusPendingApproval
		timesheet.AllowedEdit = true
		timesheet.ApprovedBy = nil
		timesheet.ApprovedAt = nil

		timesheet.RequestEditID = nil
		if err := s.timesheetRepo.Update(ctx, timesheet); err != nil {
			return err
		}

		// 5. Mark request as approved
		if err := editRequest.Approve(approvedBy); err != nil {
			return err
		}

		if err := s.editRequestRepo.Update(ctx, editRequest); err != nil {
			return err
		}

		// 6. Send notification to requester
		// Note: NotificationPort interface would need to be extended to support this use case
		// For now, we skip the notification

		// 7. Publish TimesheetEditRequestUpdatedEvent to reflect approval
		if s.eventBus != nil {
			event := domain.NewTimesheetEditRequestUpdatedEvent(ctx, editRequest.ID, editRequest.TimesheetID, editRequest.RequestedBy, string(editRequest.Status), "")
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish TimesheetEditRequestUpdatedEvent (approve)", "error", err)
			}
		}
		return nil
	})
}

// RejectEditRequest rejects an edit request
func (s *TimesheetEditRequestService) RejectEditRequest(ctx context.Context, requestID uint, rejectedBy uint) error {
	return s.transactionMgr.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get edit request
		editRequest, err := s.editRequestRepo.GetByID(ctx, requestID)
		if err != nil {
			return err
		}

		// 2. Validate it can be rejected
		if !editRequest.CanBeRejected() {
			return domain.NewValidationError(fmt.Sprintf("không thể từ chối yêu cầu: trạng thái hiện tại là '%s'", editRequest.Status))
		}

		// 3. Mark request as rejected
		if err := editRequest.Reject(rejectedBy); err != nil {
			return err
		}

		timesheet, err := s.timesheetRepo.GetByID(ctx, editRequest.TimesheetID)
		if err != nil {
			return err
		}
		timesheet.RequestEditID = nil
		if err := s.timesheetRepo.Update(ctx, timesheet); err != nil {
			return err
		}

		if err := s.editRequestRepo.Update(ctx, editRequest); err != nil {
			return err
		}

		// 4. Publish TimesheetEditRequestUpdatedEvent to reflect rejection
		if s.eventBus != nil {
			event := domain.NewTimesheetEditRequestUpdatedEvent(ctx, editRequest.ID, editRequest.TimesheetID, editRequest.RequestedBy, string(editRequest.Status), "")
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish TimesheetEditRequestUpdatedEvent (reject)", "error", err)
			}
		}

		return nil
	})
}

// ListEditRequests lists edit requests with filters
func (s *TimesheetEditRequestService) ListEditRequests(ctx context.Context, filters domain.TimesheetEditRequestFilters) ([]*domain.TimesheetEditRequest, error) {
	return s.editRequestRepo.List(ctx, filters)
}

// CountEditRequests counts edit requests with filters
func (s *TimesheetEditRequestService) CountEditRequests(ctx context.Context, filters domain.TimesheetEditRequestFilters) (int64, error) {
	return s.editRequestRepo.Count(ctx, filters)
}

// GetEditRequest gets a single edit request by ID
func (s *TimesheetEditRequestService) GetEditRequest(ctx context.Context, requestID uint) (*domain.TimesheetEditRequest, error) {
	return s.editRequestRepo.GetByID(ctx, requestID)
}

// GetPendingForTimesheet gets a pending edit request for a timesheet by a specific user
func (s *TimesheetEditRequestService) GetPendingForTimesheet(ctx context.Context, timesheetID uint, requestedBy uint) (*domain.TimesheetEditRequest, error) {
	filters := domain.TimesheetEditRequestFilters{
		TimesheetID: &timesheetID,
		RequestedBy: &requestedBy,
		Status:      []domain.TimesheetEditRequestStatus{domain.EditRequestStatusPending},
		Limit:       1,
	}

	requests, err := s.editRequestRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	if len(requests) == 0 {
		return nil, nil
	}

	return requests[0], nil
}

// CancelEditRequest allows a PARTNER to cancel their own pending edit request
func (s *TimesheetEditRequestService) CancelEditRequest(ctx context.Context, requestID uint, cancelledBy uint) error {
	return s.transactionMgr.ExecuteInTransaction(ctx, func(tx *gorm.DB) error {
		// 1. Get edit request
		editRequest, err := s.editRequestRepo.GetByID(ctx, requestID)
		if err != nil {
			return err
		}

		// 2. Validate the requester is cancelling their own request
		if editRequest.RequestedBy != cancelledBy {
			return domain.NewForbiddenError(constants.MsgCanOnlyCancelOwnEditRequestVN)
		}

		// 3. Validate it can be cancelled
		if !editRequest.CanBeCancelled() {
			return domain.NewValidationError(fmt.Sprintf("không thể hủy yêu cầu: trạng thái hiện tại là '%s', chỉ có thể hủy yêu cầu đang chờ phê duyệt", editRequest.Status))
		}

		// 4. Get the timesheet and clear the request_edit_id
		timesheet, err := s.timesheetRepo.GetByID(ctx, editRequest.TimesheetID)
		if err != nil {
			return err
		}

		timesheet.RequestEditID = nil
		if err := s.timesheetRepo.Update(ctx, timesheet); err != nil {
			return err
		}

		// 5. Soft delete the edit request
		if err := s.editRequestRepo.Delete(ctx, requestID); err != nil {
			return err
		}

		// 6. Publish TimesheetEditRequestUpdatedEvent to reflect cancellation
		if s.eventBus != nil {
			event := domain.NewTimesheetEditRequestUpdatedEvent(ctx, editRequest.ID, editRequest.TimesheetID, editRequest.RequestedBy, string(editRequest.Status), "")
			if err := s.eventBus.Publish(ctx, event); err != nil {
				// Log but don't fail the transaction if event publishing fails
				observability.GetLogger().Warn("failed to publish TimesheetEditRequestUpdatedEvent (cancel)", "error", err)
			}
		}

		return nil
	})
}
