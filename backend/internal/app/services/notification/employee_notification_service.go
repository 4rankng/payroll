package notification

import (
	"context"
	"fmt"
	"log/slog"

	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

// EmployeeNotifier sends push notifications to employees about status changes.
type EmployeeNotifier interface {
	NotifyTimesheetPaid(ctx context.Context, timesheets []*domain.Timesheet)
	NotifyAdvancePaymentStatusChanged(ctx context.Context, employeeID uint, status domain.AdvancePaymentRequestStatus, amount uint64)
	// NotifyAdvancePaymentFailed tells the employee their advance transfer failed
	// permanently and why, so they can fix the cause (e.g. wrong bank account
	// holder name) and submit a new request.
	NotifyAdvancePaymentFailed(ctx context.Context, employeeID uint, amount uint64, detail string)
}

// EmployeeNotificationService sends push notifications to employees about status changes.
type EmployeeNotificationService struct {
	notificationService *NotificationService
	employeeRepo        domain.EmployeeRepository
	logger              *slog.Logger
}

// NewEmployeeNotificationService creates a new employee notification service.
func NewEmployeeNotificationService(
	notificationService *NotificationService,
	employeeRepo domain.EmployeeRepository,
	logger *slog.Logger,
) *EmployeeNotificationService {
	return &EmployeeNotificationService{
		notificationService: notificationService,
		employeeRepo:        employeeRepo,
		logger:              logger,
	}
}

// NotifyTimesheetPaid sends a notification to each employee whose timesheets were paid.
// It groups timesheets by employee and aggregates the total paid amount.
func (s *EmployeeNotificationService) NotifyTimesheetPaid(ctx context.Context, timesheets []*domain.Timesheet) {
	type empTotal struct {
		amount int64
		userID *uint
	}
	byEmployee := make(map[uint]*empTotal)

	for _, ts := range timesheets {
		e, ok := byEmployee[ts.EmployeeID]
		if !ok {
			e = &empTotal{}
			byEmployee[ts.EmployeeID] = e
		}
		e.amount += ts.Amount
	}

	for employeeID, e := range byEmployee {
		emp, err := s.employeeRepo.GetByID(ctx, employeeID)
		if err != nil {
			s.logger.Error("failed to get employee for timesheet paid notification", "employee_id", employeeID, "error", err)
			continue
		}
		e.userID = emp.UserID

		if e.userID == nil {
			continue
		}

		title := "Tiền công đã được thanh toán"
		message := fmt.Sprintf("Bạn đã nhận được %s tiền công.", utils.FormatVND(e.amount))

		if err := s.notificationService.CreateNotification(ctx, *e.userID, domain.NotificationTypeTimesheetPaid, title, message); err != nil {
			s.logger.Error("failed to send timesheet paid notification", "employee_id", employeeID, "error", err)
		}
	}
}

// NotifyAdvancePaymentStatusChanged notifies an employee about advance payment status change.
// Employees only care about: completed (paid), failed, cancelled.
func (s *EmployeeNotificationService) NotifyAdvancePaymentStatusChanged(ctx context.Context, employeeID uint, status domain.AdvancePaymentRequestStatus, amount uint64) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		s.logger.Error("failed to get employee for advance payment notification", "employee_id", employeeID, "error", err)
		return
	}

	if emp.UserID == nil {
		return
	}

	var title, message string
	amountStr := utils.FormatVND(int64(amount))

	switch status {
	case domain.AdvancePaymentStatusCompleted:
		title = "Tiền ứng lương đã được chuyển"
		message = fmt.Sprintf("Số tiền ứng %s đã được chuyển vào tài khoản của bạn.", amountStr)
	case domain.AdvancePaymentStatusFailed:
		title = "Chuyển tiền ứng lương thất bại"
		message = fmt.Sprintf("Việc chuyển số tiền ứng %s đã thất bại. Vui lòng liên hệ quản lý.", amountStr)
	case domain.AdvancePaymentStatusCancelled:
		title = "Yêu cầu ứng lương đã bị hủy"
		message = fmt.Sprintf("Yêu cầu ứng số tiền %s đã bị hủy.", amountStr)
	default:
		return
	}

	if err := s.notificationService.CreateNotification(ctx, *emp.UserID, domain.NotificationTypeAdvancePaymentStatusChanged, title, message); err != nil {
		s.logger.Error("failed to send advance payment status notification", "employee_id", employeeID, "status", status, "error", err)
	}
}

// NotifyAdvancePaymentFailed notifies an employee that their advance transfer
// failed permanently, including the provider's reason (detail) and what to do
// next. Used for data errors the employee can fix themselves, e.g. the bank
// account holder name not matching their registered account.
func (s *EmployeeNotificationService) NotifyAdvancePaymentFailed(ctx context.Context, employeeID uint, amount uint64, detail string) {
	emp, err := s.employeeRepo.GetByID(ctx, employeeID)
	if err != nil {
		s.logger.Error("failed to get employee for advance payment failure notification", "employee_id", employeeID, "error", err)
		return
	}

	if emp.UserID == nil {
		return
	}

	title := "Chuyển tiền ứng lương thất bại"
	message := fmt.Sprintf(
		"Số tiền ứng %s không thể chuyển: %s. Vui lòng kiểm tra và cập nhật lại thông tin ngân hàng (số tài khoản, chủ tài khoản) rồi tạo yêu cầu mới.",
		utils.FormatVND(int64(amount)), detail,
	)

	if err := s.notificationService.CreateNotification(ctx, *emp.UserID, domain.NotificationTypeAdvancePaymentStatusChanged, title, message); err != nil {
		s.logger.Error("failed to send advance payment failure notification", "employee_id", employeeID, "error", err)
	}
}
