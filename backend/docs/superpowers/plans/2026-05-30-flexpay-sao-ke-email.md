# Automated FlexPay Sao Ke Email Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two cron jobs — one to cancel pending flexible advance payment requests on the 8th, another to generate and send the sao ke reconciliation email on the 9th.

**Architecture:** Reuse the exact same service calls as the manual `SendReconciliationEmail` handler. Two new cron job definitions in `scheduler_jobs.go`, two new parameters threaded through `initScheduler`.

**Tech Stack:** Go, existing scheduler framework, existing EmailService + FlexPayReconciliationService

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/app/bootstrap/scheduler_jobs.go` | Modify | Add 2 cron job definitions + update function signature |
| `internal/app/bootstrap/container.go` | Modify | Thread new deps through `initScheduler` |

---

### Task 1: Thread FlexPayReconciliationService and FlexPayReconciliationExporter through the scheduler DI chain

**Files:**
- Modify: `internal/app/bootstrap/container.go:153` (call site)
- Modify: `internal/app/bootstrap/container.go:399` (initScheduler signature + body)
- Modify: `internal/app/bootstrap/scheduler_jobs.go:1-33` (imports + function signature)

- [ ] **Step 1: Update `registerSchedulerJobs` signature in `scheduler_jobs.go`**

Add two new parameters to the function signature. Add required imports.

In `scheduler_jobs.go`, update imports to add:
```go
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
```

Update `registerSchedulerJobs` function signature (line 22-33):
```go
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
```

- [ ] **Step 2: Update `initScheduler` signature in `container.go`**

Update the `initScheduler` function at line 399. Add the two new parameters after `walletSvc`:

```go
func initScheduler(notificationService *notification.NotificationService, projectEmployeeService *project.ProjectEmployeeService, apiMetricCleanupService *cleanup.APIMetricCleanupService, reconcileService *ledger.ReconcileService, emailService *notification.EmailService, apiMetricRepo domain.APIMetricRepository, advancePaymentReqRepo domain.AdvancePaymentRequestRepository, walletSvc wallet.WalletService, flexPayReconciliationSvc *domainServices.FlexPayReconciliationService, flexPayReconciliationExporter *flex_pay.FlexPayReconciliationExporter, cfg *config.Config, logger *slog.Logger) *scheduler.Scheduler {
```

Add imports at top of `container.go`:
```go
domainServices "api-server/internal/domain/services"
"api-server/internal/app/services/flex_pay"
```

- [ ] **Step 3: Update `registerSchedulerJobs` call inside `initScheduler`**

Inside `initScheduler` (around line 406-417), pass the two new arguments:
```go
	registerSchedulerJobs(
		s,
		notificationService,
		projectEmployeeService,
		apiMetricCleanupService,
		reconcileService,
		emailService,
		apiMetricRepo,
		advancePaymentReqRepo,
		walletSvc,
		flexPayReconciliationSvc,
		flexPayReconciliationExporter,
		logger,
	)
```

- [ ] **Step 4: Update `initScheduler` call site in `NewContainer`**

At line 153 of `container.go`, update the call:
```go
sched := initScheduler(services.Notification, services.ProjectEmployee, services.APIMetricCleanupService, services.Reconcile, services.Email, repos.APIMetric, repos.AdvancePaymentRequest, services.Wallet, services.FlexPayReconciliationService, services.FlexPayReconciliationExporter, cfg, infra.Logger)
```

- [ ] **Step 5: Verify compilation**

Run: `go build ./...`
Expected: compiles with no errors (the new params are unused in scheduler_jobs.go but Go allows unused params)

- [ ] **Step 6: Commit**

```bash
git add internal/app/bootstrap/scheduler_jobs.go internal/app/bootstrap/container.go
git commit -m "refactor(scheduler): thread FlexPayReconciliation deps through scheduler DI"
```

---

### Task 2: Add cancel_flexible_pending_requests cron job (8th at 23:59)

**Files:**
- Modify: `internal/app/bootstrap/scheduler_jobs.go`

- [ ] **Step 1: Add the cron job after the existing `sync_wallet_balance` job (before the closing `}`)**

Insert after the `sync_wallet_balance` job (after line 306), before the closing `}` of `registerSchedulerJobs`:

```go
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
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add internal/app/bootstrap/scheduler_jobs.go
git commit -m "feat(scheduler): add cancel_flexible_pending_requests cron job (8th monthly)"
```

---

### Task 3: Add send_flexible_sao_ke_email cron job (9th at 9:00 AM)

**Files:**
- Modify: `internal/app/bootstrap/scheduler_jobs.go`

- [ ] **Step 1: Add the cron job after the cancel job**

Insert after the `cancel_flexible_pending_requests` job:

```go
	// 10. Send FlexPay sao ke email for previous month - 9th at 9:00 AM
	s.AddJob(scheduler.Job{
		Name:    "send_flexible_sao_ke_email",
		Cron:    "0 9 9 * *",
		Enabled: true,
		Handler: func() {
			ctx := context.Background()
			now := clock.Now()
			prevMonth := now.AddDate(0, -1, 0)
			forMonth := advance_payment.GetCurrentMonthFromTime(prevMonth)
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

			// Build email content - same template as manual handler
			parsedMonth, _ := time.Parse("2006-01", forMonth)
			endOfMonth := time.Date(parsedMonth.Year(), parsedMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC)
			dueDate := endOfMonth.Format("02/01/2006")
			totalCollect := fmt.Sprintf("%s đ", formatCurrencyVN(uint64(summary.TotalWithFee)))

			htmlBody := fmt.Sprintf(`<p>Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,</p>
<p>Chi tiết sao kê thanh toán ứng lương cho tháng %s được đính kèm trong email này để Quý Công ty tiện theo dõi và đối chiếu.</p>
<p>Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.</p>
<p><strong>Tổng tiền thanh toán: %s</strong></p>
<p><strong>Thông tin chuyển khoản:</strong><br>
Chủ tài khoản: NGUYEN VIET DUNG<br>
Số tài khoản: 1357210887<br>
Ngân hàng: TECHCOMBANK</p>
<p>Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.</p>
<p>Trân trọng,<br>Dịch vụ thanh toán TingTing</p>`, forMonth, dueDate, totalCollect)

			textBody := fmt.Sprintf(`Kính gửi: CÔNG TY CỔ PHẦN QUỐC TẾ THƯƠNG MẠI VÀ DỊCH VỤ VIỆT PHÁP,

Chi tiết sao kê thanh toán ứng lương cho tháng %s được đính kèm trong email này để Quý Công ty tiện theo dõi và đối chiếu.

Kính mong Quý Công ty kiểm tra và thanh toán số tiền dịch vụ trước ngày %s.

Tổng tiền thanh toán: %s

Thông tin chuyển khoản:
Chủ tài khoản: NGUYEN VIET DUNG
Số tài khoản: 1357210887
Ngân hàng: TECHCOMBANK

Trân trọng cảm ơn Quý Công Ty đã hợp tác và tin tưởng sử dụng dịch vụ của TingTing.

Trân trọng,
Dịch vụ thanh toán TingTing`, forMonth, dueDate, totalCollect)

			// Send email
			recipients := []string{"frankng.sg@gmail.com", "haianh211vn@gmail.com"}
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
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: compiles with no errors

- [ ] **Step 3: Commit**

```bash
git add internal/app/bootstrap/scheduler_jobs.go
git commit -m "feat(scheduler): add send_flexible_sao_ke_email cron job (9th monthly)"
```

---

### Task 4: Lint and final verification

**Files:**
- All modified files

- [ ] **Step 1: Run linter**

Run: `make lint`
Expected: no new errors

- [ ] **Step 2: Run build**

Run: `go build ./...`
Expected: clean build

- [ ] **Step 3: Verify cron jobs register correctly**

Start the server and check the `/admin/cron-health` page or call the cron status API. Both new jobs should appear:
- `cancel_flexible_pending_requests` — `59 23 8 * *` — enabled
- `send_flexible_sao_ke_email` — `0 9 9 * *` — enabled
