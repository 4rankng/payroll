package bootstrap

import (
	"api-server/internal/pkg/clock"
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"api-server/internal/app/dto"
	"api-server/internal/app/services/advance_payment"
	"api-server/internal/app/services/cleanup"
	"api-server/internal/app/services/flex_pay"
	"api-server/internal/app/services/ledger"
	"api-server/internal/app/services/notification"
	"api-server/internal/app/services/project"
	"api-server/internal/app/services/scheduler"
	"api-server/internal/domain"
	domainServices "api-server/internal/domain/services"
	"api-server/internal/domain/wallet"
)

func registerSchedulerJobs(
	s *scheduler.Scheduler,
	notificationService *notification.NotificationService,
	projectEmployeeService *project.ProjectEmployeeService,
	apiMetricCleanupService *cleanup.APIMetricCleanupService,
	reconcileService *ledger.ReconcileService,
	emailService *notification.EmailService,
	apiMetricRepo domain.APIMetricRepository,
	advancePaymentReqRepo domain.AdvancePaymentRequestRepository,
	walletSyncService wallet.WalletService,
	flexPayReconciliationService *domainServices.FlexPayReconciliationService,
	flexPayReconciliationExporter *flex_pay.FlexPayReconciliationExporter,
	logger *slog.Logger,
) {
	// Helper for template rendering
	renderTemplate := func(template string) string {
		now := clock.Now()
		replacements := map[string]string{
			"{date}":  now.Format("02/01/2006"),
			"{day}":   fmt.Sprintf("%d", now.Day()),
			"{month}": fmt.Sprintf("%d", now.Month()),
			"{year}":  fmt.Sprintf("%d", now.Year()),
		}
		result := template
		for key, value := range replacements {
			result = strings.ReplaceAll(result, key, value)
		}
		return result
	}

	// 1. Apply pending payment schedule changes
	s.AddJob(scheduler.Job{
		Name:    "apply_payment_schedule_changes",
		Cron:    "0 1 * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting automatic payment schedule change application")
			if err := projectEmployeeService.ApplyPendingScheduleChanges(ctx); err != nil {
				logger.Error("Failed to apply pending payment schedule changes", "error", err)
			} else {
				logger.Info("Payment schedule changes applied successfully")
			}
		},
	})

	// 2. Cleanup old API metrics
	s.AddJob(scheduler.Job{
		Name:    "cleanup_old_api_metrics",
		Cron:    "0 0 15 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting old API metrics cleanup (30+ days old)")
			if count, err := apiMetricCleanupService.DeleteOldMetrics(ctx, 30); err != nil {
				logger.Error("Failed to cleanup old API metrics", "error", err)
			} else {
				logger.Info("Old API metrics cleanup completed successfully", "deleted_count", count)
			}
		},
	})

	// 3. Bank statement reminder - Day 2
	s.AddJob(scheduler.Job{
		Name:    "bank_statement_reminder_day_2",
		Cron:    "0 9 2 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			title := renderTemplate(domain.NotifyTypeImportant + " Gửi sao kê cho đối tác")
			message := renderTemplate("Nhắc nhở gửi sao kê ngân hàng cho công ty thanh toán vào ngày {day}/{month}/{year}")

			if err := notificationService.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeCustom, title, message); err != nil {
				logger.Error("Failed to send notification", "name", "bank_statement_reminder_day_2", "error", err)
			} else {
				logger.Info("Notification sent successfully", "name", "bank_statement_reminder_day_2")
			}
		},
	})

	// 4. Bank statement reminder - Day 25
	s.AddJob(scheduler.Job{
		Name:    "bank_statement_reminder_day_25",
		Cron:    "0 9 25 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			title := renderTemplate(domain.NotifyTypeImportant + " Gửi sao kê cho đối tác")
			message := renderTemplate("Nhắc nhở gửi sao kê ngân hàng cho công ty thanh toán vào ngày {day}/{month}/{year}")

			if err := notificationService.NotifyUsersByRole(ctx, domain.RoleAdmin, domain.NotificationTypeCustom, title, message); err != nil {
				logger.Error("Failed to send notification", "name", "bank_statement_reminder_day_25", "error", err)
			} else {
				logger.Info("Notification sent successfully", "name", "bank_statement_reminder_day_25")
			}
		},
	})

	// 5. Daily receivable reconciliation
	s.AddJob(scheduler.Job{
		Name:    "daily_receivable_reconciliation",
		Cron:    "0 1 * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting daily receivable reconciliation")

			// Use admin user ID (1) for system operations
			adminUserID := uint(1)

			result, err := reconcileService.ReconcileReceivableAccount(ctx, &ledger.ReconcileRequest{}, adminUserID)
			if err != nil {
				logger.Error("Failed to reconcile receivable account", "error", err)
			} else {
				logger.Info("Daily receivable reconciliation completed successfully",
					"success", result.Success,
					"message", result.Message,
					"was_fixed", result.WasFixed,
					"imbalance_amount", result.ImbalanceAmount,
					"new_balance", result.NewReceivableBalance,
					"write_off_created", result.WriteOffCreated,
					"write_off_amount", result.WriteOffAmount,
				)
			}
		},
	})

	// 6. Monitor high error rate endpoints
	s.AddJob(scheduler.Job{
		Name:    "monitor_high_error_rates",
		Cron:    "0 5 * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting high error rate monitoring")

			// Check for last 24 hours
			since := clock.Now().Add(-24 * time.Hour)
			threshold := 0.10 // 10%

			endpoints, err := apiMetricRepo.GetHighErrorRateEndpoints(ctx, since, threshold)
			if err != nil {
				logger.Error("Failed to get high error rate endpoints", "error", err)
				return
			}

			if len(endpoints) == 0 {
				logger.Info("No high error rate endpoints found")
				return
			}

			// Construct message
			var sb strings.Builder
			fmt.Fprintf(&sb, "%s High Error Rate Alert\n\n", domain.NotifyTypeImportant)
			sb.WriteString("The following endpoints have error rate > 10% in the last 24 hours:\n\n")

			for _, ep := range endpoints {
				fmt.Fprintf(&sb, "- %s %s: %.2f%% (%d/%d errors)\n",
					ep.Method, ep.Path, ep.ErrorRate*100, ep.ErrorCount, ep.TotalCount)
			}

			message := sb.String()
			title := fmt.Sprintf("%s High Error Rate Alert", domain.NotifyTypeImportant)

			// 1. Send in-app notification to admin (ID 1)
			if err := notificationService.CreateCustomNotification(ctx, 1, 1, title, message, domain.NotificationContentTypePlainText); err != nil {
				logger.Error("Failed to send high error rate notification to admin", "error", err)
			} else {
				logger.Info("High error rate notification sent to admin")
			}

			// 2. Send email to frank.nguyen.vd@gmail.com
			emailReq := &dto.SendEmailRequest{
				Recipients: []string{"frank.nguyen.vd@gmail.com"},
				Subject:    title,
				TextBody:   message,
			}

			if _, err := emailService.SendGenericEmail(ctx, emailReq); err != nil {
				logger.Error("Failed to send high error rate email", "error", err)
			} else {
				logger.Info("High error rate email sent to frank.nguyen.vd@gmail.com")
			}
		},
	})

	// 7. Advance payment reminder - daily at 8 PM ICT (9 PM SGT)
	s.AddJob(scheduler.Job{
		Name:    "advance_payment_reminder",
		Cron:    "0 20 * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting advance payment reminder check")

			count, err := advancePaymentReqRepo.CountPending(ctx)
			if err != nil {
				logger.Error("Failed to count pending advance payment requests", "error", err)
				return
			}

			if count == 0 {
				logger.Info("No pending advance payment requests, skipping reminder")
				return
			}

			now := clock.Now()
			currentMonth := advance_payment.GetCurrentMonthFromTime(now)

			grouped, err := advancePaymentReqRepo.GetPendingGroupedByEmployee(ctx, currentMonth)
			if err != nil {
				logger.Error("Failed to get pending advance payment requests grouped by employee", "error", err)
				return
			}

			if len(grouped) == 0 {
				logger.Info("No grouped pending requests for current month, skipping reminder")
				return
			}

			var totalAmount, totalFee, totalNetAmount uint64
			requests := make([]notification.AdvancePaymentReminderRequest, 0, len(grouped))
			for i, g := range grouped {
				totalAmount += g.TotalAmount
				totalFee += g.TotalFee
				totalNetAmount += g.NetAmount
				requests = append(requests, notification.AdvancePaymentReminderRequest{
					STT:           i + 1,
					EmployeeCCCD:  g.EmployeeCCCD,
					EmployeeName:  g.EmployeeName,
					ProjectCode:   g.ProjectCode,
					RequestAmount: formatCurrencyVN(int64(g.TotalAmount)) + " đ",
					Fee:           formatCurrencyVN(int64(g.TotalFee)) + " đ",
					NetAmount:     formatCurrencyVN(int64(g.NetAmount)) + " đ",
				})
			}

			data := &notification.AdvancePaymentReminderData{
				Date:               now.Format("02/01/2006"),
				TotalRequests:      fmt.Sprintf("%d", count),
				TotalEmployees:     fmt.Sprintf("%d", len(grouped)),
				TotalAmount:        formatCurrencyVN(int64(totalAmount)) + " đ",
				Requests:           requests,
				TotalRequestAmount: formatCurrencyVN(int64(totalAmount)) + " đ",
				TotalFee:           formatCurrencyVN(int64(totalFee)) + " đ",
				TotalNetAmount:     formatCurrencyVN(int64(totalNetAmount)) + " đ",
			}

			recipients := []string{"frankng.sg@gmail.com", "anhbh@vfic.com.vn"}
			if _, err := emailService.SendAdvancePaymentReminder(ctx, data, recipients); err != nil {
				logger.Error("Failed to send advance payment reminder email", "error", err)
			} else {
				logger.Info("Advance payment reminder email sent successfully",
					"pending_count", count,
					"employee_count", len(grouped),
				)
			}
		},
	})

	// 8. Sync wallet balance with provider every 30 minutes
	s.AddJob(scheduler.Job{
		Name:    "sync_wallet_balance",
		Cron:    "*/30 * * * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			logger.Info("Starting wallet balance sync with provider")

			result, err := walletSyncService.SyncBalance(ctx, 1) // system user ID
			if err != nil {
				logger.Error("Wallet balance sync failed", "error", err)
				return
			}

			if result.Adjusted {
				logger.Info("Wallet balance adjusted",
					"provider_balance", result.ProviderBalance,
					"local_balance", result.LocalBalance,
				)
			} else {
				logger.Info("Wallet balance in sync",
					"balance", result.LocalBalance,
				)
			}
		},
	})

	// 9. Cancel pending flexible advance payment requests - 8th at 23:59
	s.AddJob(scheduler.Job{
		Name:    "cancel_flexible_pending_requests",
		Cron:    "59 23 8 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			now := clock.Now()
			currentMonth := advance_payment.GetCurrentMonthFromTime(now)

			logger.Info("Starting cancellation of pending flexible advance payment requests",
				"month", currentMonth)

			cancelledCount, requestIDs, err := flexPayReconciliationService.CancelAllPendingRequests(ctx, currentMonth)
			if err != nil {
				logger.Error("Failed to cancel pending flexible advance payment requests",
					"month", currentMonth, "error", err)
				return
			}

			logger.Info("Cancelled pending flexible advance payment requests",
				"month", currentMonth,
				"cancelled_count", cancelledCount,
				"request_ids", requestIDs)
		},
	})

	// 10. Send FlexPay sao ke email for previous month - 9th at 9:00 AM
	s.AddJob(scheduler.Job{
		Name:    "send_flexible_sao_ke_email",
		Cron:    "0 9 9 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			now := clock.Now()
			prevMonth := now.AddDate(0, -1, 0)
			forMonth := prevMonth.Format("2006-01")
			atDate := time.Date(prevMonth.Year(), prevMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

			logger.Info("Starting FlexPay sao ke email generation",
				"previous_month", forMonth)

			// Get completed requests for the previous month
			reportData, err := flexPayReconciliationService.GetCompletedRequestsByMonth(ctx, forMonth)
			if err != nil {
				logger.Error("Failed to get completed requests for sao ke",
					"month", forMonth, "error", err)
				return
			}

			if len(reportData) == 0 {
				logger.Info("No completed flexible requests for previous month, skipping sao ke email",
					"month", forMonth)
				return
			}

			// Generate Excel file
			excelBytes, summary, err := flexPayReconciliationExporter.GenerateExcel(reportData, atDate)
			if err != nil {
				logger.Error("Failed to generate FlexPay sao ke Excel",
					"month", forMonth, "error", err)
				return
			}

			// Build email content - due date is end of the NEXT month (e.g. April advance → due end of May)
			endOfNextMonth := time.Date(prevMonth.Year(), prevMonth.Month()+2, 0, 0, 0, 0, 0, time.UTC)
			dueDate := endOfNextMonth.Format("02/01/2006")
			totalCollect := formatCurrencyVN(summary.TotalWithFee) + " đ"

			htmlBody, textBody := flex_pay.BuildSaoKeEmailBodies(forMonth, dueDate, totalCollect)

			// Send email
			recipients := []string{"frankng.sg@gmail.com", "anhbh@vfic.com.vn"}
			emailID, err := emailService.SendAdvancePaymentReconciliationEmail(ctx, &notification.ReconciliationEmailParams{
				ForMonth:   forMonth,
				Recipients: recipients,
				ExcelBytes: excelBytes,
				HTMLBody:   htmlBody,
				TextBody:   textBody,
				Subject:    fmt.Sprintf("Sao kê thanh toán ứng lương - %s", forMonth),
				Summary:    summary,
			})
			if err != nil {
				logger.Error("Failed to send FlexPay sao ke email",
					"month", forMonth, "error", err)
				return
			}

			logger.Info("FlexPay sao ke email sent successfully",
				"month", forMonth,
				"email_id", emailID,
				"project_count", len(reportData),
				"recipients", recipients)
		},
	})

}

func formatCurrencyVN(amount int64) string {
	s := fmt.Sprintf("%d", amount)
	negative := false
	if len(s) > 0 && s[0] == '-' {
		negative = true
		s = s[1:]
	}
	var result strings.Builder
	if negative {
		result.WriteByte('-')
	}
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteString(".")
		}
		result.WriteRune(r)
	}
	return result.String()
}
