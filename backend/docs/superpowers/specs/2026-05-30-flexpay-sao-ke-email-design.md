# Automated FlexPay Sao Ke Email (9th Monthly)

**Date:** 2026-05-30
**Status:** Approved
**Author:** Claude Code

---

## Problem

Flexible project advance payment reconciliation (sao ke) is currently sent manually by admins. It should be automated: requests stop on the 8th, sao ke email sends on the 9th.

## Solution

Two cron jobs in `scheduler_jobs.go` that reuse the exact same service calls as the manual `SendReconciliationEmail` handler (`email_handler.go:17-136`).

---

## Design

### Cron Jobs

| Job Name | Cron | Description |
|----------|------|-------------|
| `cancel_flexible_pending_requests` | `59 23 8 * *` | Cancel all pending/approved flexible advance payment requests for the current month |
| `send_flexible_sao_ke_email` | `0 9 9 * *` | Generate and send sao ke reconciliation email for the previous month |

### Flow

```
8th 23:59 → Cancel all pending/approved requests for current month
             Uses: FlexPayReconciliationService.CancelAllPendingRequests(ctx, currentMonth)
             Logs: cancelled count

9th 09:00 → Generate and send sao ke email for previous month
             1. Compute previousMonth = clock.Now() - 1 month
             2. FlexPayReconciliationService.GetCompletedRequestsByMonth(ctx, previousMonth)
             3. If no data → log "no completed requests" and return
             4. FlexPayReconciliationExporter.GenerateExcel(reportData, atDate)
             5. EmailService.SendAdvancePaymentReconciliationEmail() with fixed recipients
             6. Excel saved as asset for download history
```

### Recipients

Fixed list: `frankng.sg@gmail.com`, `anhbh@vfic.com.vn`

### Email Content

Identical to manual flow:
- **Subject:** `Sao kê thanh toán ứng lương - {YYYY-MM}`
- **Body:** Vietnamese template with bank transfer info (Chủ tài khoản: NGUYEN VIET DUNG, STK: 1357210887, Techcombank), total amount, due date (end of month)
- **Attachment:** `sao_ke_tt_{YYYY-MM}.xlsx`
- **Kind:** `domain.EmailKindAdvancePaymentReport`
- **Metadata:** `PayrollEmailMetadata` with totals, fee percentage, asset ID

### Month Computation

```go
now := clock.Now()
prevMonth := now.AddDate(0, -1, 0)
forMonth := advance_payment.GetCurrentMonthFromTime(prevMonth) // "YYYY-MM"
atDate := time.Date(prevMonth.Year(), prevMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
```

---

## Dependency Injection Changes

### `registerSchedulerJobs` signature

Add two parameters:
- `flexPayReconciliationService *services.FlexPayReconciliationService`
- `flexPayReconciliationExporter *flex_pay.FlexPayReconciliationExporter`

### `initScheduler` signature

Add same two parameters, pass through to `registerSchedulerJobs`.

### `container.go` → `initScheduler` call

Pass the two new arguments from existing service instances in the DI container.

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No completed requests for previous month | Log info, skip email (no error) |
| Excel generation failure | Log error, do not retry (manual send still available) |
| Email dispatch failure | Log error, do not retry (manual send still available) |
| Cancellation failure on 8th | Log error, non-blocking |

---

## Files Changed

| File | Change |
|------|--------|
| `internal/app/bootstrap/scheduler_jobs.go` | Add `cancel_flexible_pending_requests` and `send_flexible_sao_ke_email` cron jobs (~60-80 lines) |
| `internal/app/bootstrap/container.go` | Thread `FlexPayReconciliationService` and `FlexPayReconciliationExporter` through `initScheduler` (~4 lines) |

**No new files. No new services. No database changes.**

---

## Testing

- Verify cron jobs appear in `/admin/cron-health` page
- Verify job 1 cancels pending requests on 8th
- Verify job 2 generates and sends email on 9th
- Manually trigger via cron health page to test
- Check email history for the reconciliation email record
- Verify Excel asset is saved and downloadable
