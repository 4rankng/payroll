package timesheet

import (
	"context"
	"time"

	"api-server/internal/domain"
)

const rejectUnpaidPostCommitTimeout = 5 * time.Second

// RejectUnpaidResult separates the committed destructive result from best-
// effort post-commit cache/audit delivery. Callers must not retry the write when
// PostCommitComplete is false because the rejection has already committed.
type RejectUnpaidResult struct {
	RejectedCount      int64
	PostCommitComplete bool
	Warnings           []string
}

// RejectUnpaidByProjectDateRange rejects all non-paid timesheets for one
// project in an inclusive date range. Selection and mutation remain inside one
// conditional database command, avoiding pagination limits and N+1 updates.
func (s *TimesheetService) RejectUnpaidByProjectDateRange(
	ctx context.Context,
	projectID uint,
	fromDate time.Time,
	toDate time.Time,
	rejectionReason string,
	rejectedBy uint,
) (*RejectUnpaidResult, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}
	rejector, ok := s.timesheetRepo.(domain.TimesheetUnpaidRejector)
	if !ok {
		return nil, domain.NewInternalError("Kho lưu trữ bảng chấm công không hỗ trợ từ chối theo phạm vi", nil)
	}

	result, err := s.transactionManager.WithTransactionResult(ctx, func(txCtx context.Context) (interface{}, error) {
		return rejector.RejectUnpaidByProjectDateRange(
			txCtx,
			projectID,
			fromDate,
			toDate,
			rejectionReason,
			rejectedBy,
		)
	})
	if err != nil {
		return nil, err
	}

	rejectedCount := result.(int64)
	event := domain.NewTimesheetBulkRejectedEvent(ctx, int(rejectedCount), &projectID, rejectedBy)
	return s.deliverRejectUnpaidPostCommit(event, rejectedCount, projectID), nil
}

func (s *TimesheetService) deliverRejectUnpaidPostCommit(
	event domain.TimesheetBulkRejectedEvent,
	rejectedCount int64,
	projectID uint,
) *RejectUnpaidResult {
	outcome := &RejectUnpaidResult{
		RejectedCount:      rejectedCount,
		PostCommitComplete: true,
		Warnings:           make([]string, 0),
	}

	// No database-backed event outbox is wired to this service. Use a bounded,
	// request-independent context so client disconnects cannot cancel audit/cache
	// delivery. Failures are explicit in the response because the write already
	// committed and retrying it would be unsafe/misleading.
	postCommitCtx, cancel := context.WithTimeout(context.Background(), rejectUnpaidPostCommitTimeout)
	defer cancel()
	cacheWarningAdded := false
	for _, pattern := range []string{"timesheets:list:*", "timesheets:summary:*", "dashboard:*"} {
		if s.cache == nil {
			outcome.PostCommitComplete = false
			if !cacheWarningAdded {
				outcome.Warnings = append(outcome.Warnings, "Không thể làm mới toàn bộ dữ liệu bộ nhớ đệm")
				cacheWarningAdded = true
			}
			s.logger.Error("Cache unavailable after rejecting unpaid timesheets", "pattern", pattern)
			continue
		}
		if err := s.cache.InvalidatePattern(postCommitCtx, pattern); err != nil {
			outcome.PostCommitComplete = false
			if !cacheWarningAdded {
				outcome.Warnings = append(outcome.Warnings, "Không thể làm mới toàn bộ dữ liệu bộ nhớ đệm")
				cacheWarningAdded = true
			}
			s.logger.Error("Failed to invalidate cache after rejecting unpaid timesheets", "pattern", pattern, "error", err)
		}
	}

	if s.events == nil {
		outcome.PostCommitComplete = false
		outcome.Warnings = append(outcome.Warnings, "Không thể ghi nhận đầy đủ sự kiện kiểm toán")
		s.logger.Error("Event bus unavailable after rejecting unpaid timesheets", "count", rejectedCount, "project_id", projectID)
	} else if err := s.events.Publish(postCommitCtx, event); err != nil {
		outcome.PostCommitComplete = false
		outcome.Warnings = append(outcome.Warnings, "Không thể ghi nhận đầy đủ sự kiện kiểm toán")
		s.logger.Error(
			"Failed to publish TimesheetBulkRejectedEvent",
			"count", rejectedCount,
			"project_id", projectID,
			"error", err,
		)
	}

	return outcome
}
