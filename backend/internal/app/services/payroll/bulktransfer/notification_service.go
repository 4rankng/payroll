package bulktransfer

import (
	"context"
	"fmt"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/domain"
	"api-server/internal/pkg/utils"
)

// NotificationService handles partner notifications for bulk transfer payments
type NotificationService struct {
	employeeRepo     EmployeeRepository
	employeeUserRepo EmployeeUserRepository
	userRepo         UserRepository
	notifier         Notifier
	rowParser        *RowParser
}

// NewNotificationService creates a new notification service instance
func NewNotificationService(
	employeeRepo EmployeeRepository,
	employeeUserRepo EmployeeUserRepository,
	userRepo UserRepository,
	notifier Notifier,
	rowParser *RowParser,
) *NotificationService {
	return &NotificationService{
		employeeRepo:     employeeRepo,
		employeeUserRepo: employeeUserRepo,
		userRepo:         userRepo,
		notifier:         notifier,
		rowParser:        rowParser,
	}
}

type employeePayment struct {
	name   string
	amount int64
}

// NotifyPartnersOfPayments groups paid employees by their owner (partner) and sends notifications
func (ns *NotificationService) NotifyPartnersOfPayments(ctx context.Context, details []dto.BulkTransferResultItemDetail, filename string, processedAt time.Time) {
	if ns.notifier == nil {
		return
	}

	// Avoid unused parameter warnings
	_ = filename
	_ = processedAt

	// Group payments by partner
	partnerPayments := ns.groupPaymentsByPartner(ctx, details)
	if len(partnerPayments) == 0 {
		return
	}

	// Send notifications to each partner
	for userID, empMap := range partnerPayments {
		ns.sendPartnerNotification(ctx, userID, empMap)
	}
}

// groupPaymentsByPartner builds map of partnerUserID -> employeeID -> aggregated payment data
func (ns *NotificationService) groupPaymentsByPartner(ctx context.Context, details []dto.BulkTransferResultItemDetail) map[uint]map[uint]*employeePayment {
	partnerEmployees := make(map[uint]map[uint]*employeePayment)

	for _, d := range details {
		if d.EmployeeID == nil {
			continue
		}
		if strings.ToLower(d.PaymentStatus) != "paid" || strings.ToLower(d.Status) != "success" {
			continue
		}

		// Fetch employee to get owner and canonical name
		emp, err := ns.employeeRepo.GetByID(ctx, *d.EmployeeID)
		if err != nil || emp == nil {
			continue
		}

		name := emp.FormattedFullname()

		// Parse amount from detail (best-effort)
		var amt int64
		if a, err := ns.rowParser.ParseAmountFromDetail(d.Amount); err == nil {
			amt = a
		}

		// Notify owner if they are a partner
		ownerID := emp.CreatedBy
		if ns.userRepo != nil {
			if user, err := ns.userRepo.GetByID(ctx, ownerID); err == nil && user != nil && user.Role == domain.RolePartner {
				ns.addPaymentToPartner(partnerEmployees, ownerID, emp.ID, name, amt)
			}
		}

		// Notify assigned users who are partners
		if ns.employeeUserRepo != nil {
			if eUsers, err := ns.employeeUserRepo.GetEmployeeUsers(ctx, emp.ID); err == nil {
				for _, eu := range eUsers {
					isPartner := ns.isUserPartner(ctx, eu)
					if isPartner {
						ns.addPaymentToPartner(partnerEmployees, eu.UserID, emp.ID, name, amt)
					}
				}
			}
		}
	}

	return partnerEmployees
}

// isUserPartner checks if a user has partner role
func (ns *NotificationService) isUserPartner(ctx context.Context, eu *domain.EmployeeUser) bool {
	// Prefer role from preloaded User
	if eu.User.Role == domain.RolePartner {
		return true
	}

	// Fallback to userRepo if missing
	if ns.userRepo != nil {
		if u, err := ns.userRepo.GetByID(ctx, eu.UserID); err == nil && u != nil {
			return u.Role == domain.RolePartner
		}
	}

	return false
}

// addPaymentToPartner adds or updates payment information for a partner
func (ns *NotificationService) addPaymentToPartner(partnerEmployees map[uint]map[uint]*employeePayment, userID, employeeID uint, name string, amount int64) {
	if _, ok := partnerEmployees[userID]; !ok {
		partnerEmployees[userID] = make(map[uint]*employeePayment)
	}

	ep := partnerEmployees[userID][employeeID]
	if ep == nil {
		ep = &employeePayment{name: name}
		partnerEmployees[userID][employeeID] = ep
	}
	ep.amount += amount
}

// sendPartnerNotification sends a notification to a partner about their employees' payments
func (ns *NotificationService) sendPartnerNotification(ctx context.Context, userID uint, empMap map[uint]*employeePayment) {
	title := "Thông báo chi trả lương"

	// Build per-employee lines: "- Name: amount"
	lines := make([]string, 0, len(empMap))
	for _, ep := range empMap {
		amountStr := utils.FormatVND(ep.amount)
		if ep.amount == 0 {
			amountStr = "--"
		}
		lines = append(lines, fmt.Sprintf("- %s: %s", ep.name, amountStr))
	}

	body := strings.Join(lines, "\n")

	// Best-effort notify; ignore individual errors
	_ = ns.notifier.CreateNotification(ctx, userID, domain.NotificationTypePayrollComplete, title, body)
}
