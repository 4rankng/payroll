---
title: "Wallet Bulk Transfer Pipeline"
description: >-
  Two-stage workflow: (1) Admin clicks "Chuyển OnePay" button on the timesheet
  page → system exports a OnePay-API-compatible "Yêu cầu chuyển tiền" Excel
  (eMB_BulkPayment format WITH a SWIFT code column resolved from
  employee.bank_id → banks.swift_code, AND a fresh VFIC transaction code per
  row); (2) Admin uploads that exported file to /admin/wallet → system parses
  each row, initiates a OnePay transfer via the full 5-step worker pattern,
  records success/failure per row, books the aggregate OnePay fee (3,850
  VND/txn from DB schedule) as one Expense ledger entry per batch, and
  produces a downloadable "KQ Chuyen Tien" Excel.
status: pending
priority: P1
branch: "main"
tags:
  - feature
  - backend
  - frontend
  - wallet
  - disbursement
  - onepay
  - ledger
  - timesheet
blockedBy: []
blocks: []
created: "2026-07-18T14:14:34.945Z"
createdBy: "ck:plan"
source: skill
revision: 7
revision_note: >-
  v7 adds the missing first half of the workflow: the EXPORT button on the
  timesheet page that generates the OnePay-compatible input file. v6 only
  handled the wallet-side upload/process pipeline and assumed the input file
  was produced elsewhere. v7 closes that gap. Reuses existing
  bulktransfer.ExportPlanner + payroll/excel.Service infrastructure; adds a
  new SWIFT column to the eMB_BulkPayment output.
---

# Wallet Bulk Transfer Pipeline

## Overview

End-to-end bulk wallet disbursement workflow spanning two admin pages:

### Stage 1 — Export (timesheet page)

Admin selects approved timesheets (by cycle, project, period) on `/admin/timesheet` and clicks the new **"Chuyển OnePay"** dropdown item. The system:
1. Plans the transfer set via the existing `bulktransfer.ExportPlanner` (reused from the 9Pay flow).
2. For each (employee × project × amount) entry: resolves bank → SWIFT via `BankRepository.FindByBankCode(employee.bank.bank_code).SwiftCode`, generates a fresh VFIC transaction code via the existing `TransactionCode` mechanism (links back to timesheet IDs).
3. Generates a "Yêu cầu chuyển tiền" `.xlsx` in `eMB_BulkPayment` format with 7 columns: STT, Account No, Beneficiary, Beneficiary Bank (Vietnamese display name), **Mã SWIFT** (NEW), Amount, Payment Detail (contains VFIC code).
4. Returns the file as a downloadable blob.

### Stage 2 — Upload + Process (wallet page)

Admin uploads the exported file to `/admin/wallet`. The system parses each row (SWIFT read directly — no bank resolution needed at this stage), enqueues one asynq task per row, and each task drives the **full 5-step transfer flow** mirroring `disbursement_execute_worker.go` exactly. On batch completion, books ONE aggregate Expense ledger transaction.

This is the **write-side companion** to the read-only bank-transfer-history plan (`260718-1950-bank-transfer-history`, completed). It reuses the existing `bulktransfer` package internals (planner, excel service) and the OnePay queued provider.

### Key architectural decisions (locked)

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | **Two-stage workflow: timesheet export → wallet upload** | Mirrors how admins actually work: select timesheets → generate request file → upload to wallet for processing. Reuses existing planner + excel infra for stage 1. |
| 2 | **Exporter lives on `/admin/timesheet` page** | Where approved timesheets already are. Adds "Chuyển OnePay" dropdown item alongside existing "Chuyển lô" and "Xuất bảng công" items. |
| 3 | **Exporter resolves SWIFT via `BankRepository.FindByBankCode(employee.bank.bank_code).SwiftCode`** | Employee records have `bank_id` → `Bank.BankCode` → `Bank.SwiftCode`. One hop. Employees missing bank info are skipped with a warning. |
| 4 | **Exporter generates fresh VFIC per row via existing `TransactionCode` mechanism** | Reuses `bulktransfer/export_service.go:142` pattern. Each VFIC links back to timesheet IDs for audit. |
| 5 | **Output format = eMB_BulkPayment + new "Mã SWIFT" column** | Same layout as the user's attached `Yeu cau chuyen tien.xlsx`, plus column G "Mã SWIFT". The wallet-page parser (Stage 2) reads SWIFT directly from this column — no bank-name resolution at upload time. |
| 6 | **Auto-pipeline via OnePay API in Stage 2** | Worker mirrors `disbursement_execute_worker.go` 5-step pattern exactly: `Initiate` → `RecordAccountCheck(Verified:true)` → `provider.InitiateTransfer` → `RecordSyncResponse(Accepted:...)`. |
| 7 | **`Initiate` is the SOLE `wallet_payments` insertion point** | Bulk service does NOT pre-insert. Worker calls `Initiate`; fee stamps correctly from schedule. |
| 8 | **Reject duplicates** by content hash + existing `idx_wp_request_id` UNIQUE | Existing UNIQUE catches every VFIC duplicate (both historical advance-payment rows and new bulk rows use VFIC as `request_id`). |
| 9 | **Aggregate fee ledger per batch** via dedicated asynq task | One Expense transaction per batch = `sum(fee)`. Idempotent via `ledger_txn_id IS NULL`. `completing`-batch recovery cron for stuck batches. |
| 10 | **DB fee schedule is source of truth** (3,850 VND/txn) | Production already at 3,850 (`app.log:46260`). `Initiate` stamps fee at INSERT. |
| 11 | **Worker mirrors `disbursement_execute_worker.go` EXACTLY** | `StatePending` guard, `Accepted` flag, `ErrFeeResolution`→SkipRetry, `FeeWaived:true` on preflight, only real `SyncResult` fields. |
| 12 | **Outbox via `bulk_transfer_batches.data` JSON + sweeper** | Persist parsed rows in batch for re-enqueue recovery + KQ display-name lookup. |
| 13 | **Use `wallet_payments.invoice_no` for FT number** | KQ column I reads `invoice_no` with `"Đang chờ FT"` fallback when invoice_no still starts with VFIC. |
| 14 | **`RequestID = VFIC code`** (≤20 chars, OnePay hard limit) | Verified against production logs (`request_id=VFIC3ba3ec31`). |

## Scope

**In (Stage 1 — Export):**
- New "Chuyển OnePay" dropdown item on `/admin/timesheet` page (next to existing "Chuyển lô").
- New endpoint `POST /api/v1/payrolls/export-onepay-bulk` accepting the same filter shape as the existing `export-bulk-transfer` (cycle, project, period, etc.).
- New exporter service that wraps `bulktransfer.ExportPlanner.Plan()` for row selection, then enriches each row with `SwiftCode` (via `BankRepository`) and generates the eMB_BulkPayment Excel with 7 columns.
- New VFIC generation per row (reusing the existing `TransactionCode` + `transactionCodeRepo.CreateBatch` pattern from `export_service.go:142`).
- Frontend: hook + dropdown item triggering blob download.

**In (Stage 2 — Upload + Process):**
- New domain entity `BulkTransferBatch` with `data` JSON column.
- Excel parser for "Yêu cầu chuyển tiền" — reads SWIFT directly from column G.
- Excel generator for "KQ Chuyen Tien".
- Upload pipeline + 5-step worker + outbox + recovery crons.
- Idempotent aggregate ledger booking.
- New HTTP routes under `/api/v1/wallet/bulk-transfer/*`.
- Frontend: upload dialog, progress polling, KQ download, batch history.

**Out:**
- Bank-name → SWIFT resolution at the wallet-upload (Stage 2) parser — SWIFT is in the file already (resolved at Stage 1 export).
- Changes to `WalletPaymentService.Initiate` signature.
- Changes to `wallet_payments` FSM or status enum.
- Fee-schedule migrations (production already at 3,850).
- Changes to the existing 9Pay bulk transfer flow on `/payrolls`.
- Manual KQ override/re-upload.
- Mobile-specific UI.
- Reverse/refund workflow.
- Pre-flight `CheckAccount` step in worker.
- `pending_ipns` buffer table.

## Acceptance Criteria

### Stage 1 — Export
- [ ] "Chuyển OnePay" dropdown item visible on `/admin/timesheet` next to "Chuyển lô".
- [ ] Clicking it (with approved timesheets selected by cycle/project/period) triggers `POST /api/v1/payrolls/export-onepay-bulk`.
- [ ] Endpoint reuses `ExportPlanner.Plan()` for row selection.
- [ ] Each row's SWIFT is resolved via `BankRepository.FindByBankCode(employee.bank.bank_code).SwiftCode`.
- [ ] Each row gets a fresh VFIC code via the existing `TransactionCode` mechanism (persisted via `transactionCodeRepo.CreateBatch`).
- [ ] Employees missing bank info are skipped; a warning field in the response lists them.
- [ ] Response is a downloadable `.xlsx` blob in `eMB_BulkPayment` format with 7 columns (STT, Account No, Beneficiary, Beneficiary Bank, **Mã SWIFT**, Amount, Payment Detail with VFIC).
- [ ] Filename format: `Yeu_cau_chuyen_tien_<cycle>_<YYYYMMDD_HHMMSS>.xlsx`.

### Stage 2 — Upload + Process
- [ ] Admin can upload the exported `.xlsx` (≤10MB) via `/admin/wallet`.
- [ ] Backend enforces 10MB upload limit + ZIP magic content-type sniff + multipart memory cap.
- [ ] Parser reads SWIFT code directly from column G — no `BankRepository.FindByBankCode` calls.
- [ ] Upload with same content hash as an existing batch is rejected with 409 + `existing_batch_id`.
- [ ] Upload where any row's VFIC code matches an existing non-terminal `wallet_payments` row's `request_id` is rejected with 409 + conflict list, before any row is created. Backed by existing `idx_wp_request_id` UNIQUE constraint.
- [ ] Worker calls `Initiate` (sole insertion point — fee stamps correctly), then guards `Status == StatePending`, then runs full 5-step flow with explicit `Accepted` flag and all canonical error branches.
- [ ] `RequestID = VFIC code` (≤20 chars).
- [ ] For typical 50-row batches, ≥90% of rows reach terminal status within 60s; remainder within 10min via status-inquiry poller.
- [ ] On all rows terminal, `wallet:book_batch_ledger` task books exactly one Expense transaction (when `sum(fee) > 0`): `type=expense`, `party=OnePay`, `amount=sum(fee)`, `status=settled`. `ledger_txn_id` set. Re-triggering is a no-op.
- [ ] When all rows fail at OnePay pre-flight (fee zeroed via `FeeWaived: true`), no Expense transaction is created.
- [ ] `wallet:bulk_completing_recovery` cron re-attempts booking for batches stuck in `completing` >10min.
- [ ] `wallet:bulk_stale_enqueue_sweeper` cron re-enqueues tasks for batches with `enqueue_state='pending'` after 2min.
- [ ] A "KQ Chuyen Tien" `.xlsx` is generated and downloadable. KQ column I reads `invoice_no` with `"Đang chờ FT"` fallback.
- [ ] All money math uses `int64` VND. All time uses `clock.Now()` (Asia/Ho_Chi_Minh).
- [ ] Existing 9Pay bulk transfer, monthly OnePay fee import, and read-only bank-transfer-history page continue to work unchanged.
- [ ] `make api-test` passes.
- [ ] Frontend `pnpm lint && pnpm type-check` pass.

## Phases

| Phase | Name | Stage | Status |
|-------|------|-------|--------|
| 1 | [Domain & Migrations](./phase-01-domain-parser.md) | 2 | Pending |
| 2 | [Timesheet Exporter (Chuyển OnePay)](./phase-02-timesheet-exporter.md) | 1 | Pending |
| 3 | [Backend Pipeline, Worker & Ledger](./phase-03-backend-pipeline-ledger.md) | 2 | Pending |
| 4 | [KQ Excel & Routes](./phase-04-kq-excel-routes.md) | 2 | Pending |
| 5 | [Frontend UI (Timesheet + Wallet)](./phase-05-frontend-ui.md) | 1+2 | Pending |
| 6 | [Verification](./phase-06-verification.md) | both | Pending |

## Dependencies

- Phase 2 (exporter) is independent — can ship first.
- Phase 3 → Phase 1 (worker needs domain + migrations).
- Phase 4 → Phase 3 (KQ generator reads wallet_payments).
- Phase 5 → Phase 2 + Phase 4 (UI for both stages).
- Phase 6 → all.

**Recommended ship order**: Phase 2 (exporter) first as a standalone win — admins can immediately use the exported file with the existing manual OnePay workflow. Then Phase 1+3+4+5 (wallet pipeline) for end-to-end automation.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| OnePay charges 3,850 VND per row even on failure | **High** | Validation before any OnePay call; `idx_wp_request_id` UNIQUE backstop; size + ZIP magic + `UnzipSizeLimit`. |
| Exporter produces file with wrong/missing SWIFT codes | **High** | Resolve via `BankRepository.FindByBankCode(employee.bank.bank_code).SwiftCode` at export time. Skip employees with missing bank info; surface them in a warning list. Admin can fix employee records and re-export. |
| Worker dies between batch INSERT commit and asynq enqueue | **High** | Outbox pattern (`enqueue_state` + `data` JSON) + stale-enqueue sweeper cron. |
| `Initiate` returns existing row on duplicate request_id → `fee=0` | **High** | Bulk service does NOT pre-insert. Worker is sole insertion point. |
| `RecordSyncResponse` marks row `failed` if `Accepted` not set | **High** | Worker sets `Accepted: result.Status == Pending \|\| Success` explicitly per canonical. |
| Missing `StatePending` guard → asynq retries hit illegal FSM transitions | **High** | Guard added per canonical worker:127-131. |
| `completing` status unrecoverable if ledger write fails | **High** | `wallet:bulk_completing_recovery` cron every 5min. |
| excelize default `UnzipSizeLimit=16GB` → zip bomb OOM | **High** | Parser passes `excelize.Options{UnzipSizeLimit:50<<20, UnzipXMLSizeLimit:10<<20}` + 5,000-row cap. |
| `c.FormFile` buffers to disk before size check → DoS | **High** | `http.MaxBytesReader` + `router.MaxMultipartMemory=10<<20`. |
| Filename unsanitized → log injection + XSS | **Medium** | Sanitize (strip control chars, cap 128, reject XSS chars). |
| Status-inquiry poller caps completion at ~20min/50 stranded rows | Medium | Documented; future per-batch status-poll task. |
| IPN before RecordSyncResponse → IPN discarded | Medium | Documented. Status-inquiry poller recovers. |
| OTP default off for money-moving endpoint | Medium | Startup warning if `OTP_ENABLE=false`. |
| Excel format drift (column reorder) | Medium | Header-name mapping. |
| Employee bank_code mismatch with bank table | Medium | Exporter logs warnings for unresolved employees; admin fixes data and re-exports. |

## Non-Goals

- No new fee schedule types or wallet statuses.
- No reverse/refund workflow.
- No manual KQ re-upload override.
- No provider switcher UI.
- ~~No SMS/email notifications beyond existing `notifyEmployee`/`notifyInitiator`~~ **REVERSED by Validation V6**: push notifications ARE required for all employee money receipts. See "Notification Path" above — Phase 3 extends `notifyEmployee` to handle bulk rows via `recipient_account_no → employee` lookup.
- No changes to `BulkTransferFile` domain (existing 9Pay flow).
- No pre-flight `CheckAccount` step in worker.
- No `pending_ipns` buffer table.

## Validation Log

### Session 1 — 2026-07-18

**Decisions confirmed:**

| # | Question | Decision |
|---|----------|----------|
| V1 | Exporter response shape | **Single POST returns blob + X- headers** (skipped count, total amount). Simplest UX — one click downloads the file; frontend reads headers to show toast. |
| V2 | Audit record at export time | **Yes, persist `BulkTransferFile`** (filename, counts, VFIC codes, asset_id). Provides audit trail from export → upload → ledger. Mirrors existing 9Pay flow. |
| V3 | Ship order | **Exporter first, then wallet pipeline.** Note: even after both ship, the workflow remains 2-step (export from timesheet page → manual upload to wallet page → system auto-processes via OnePay API). The manual upload checkpoint is intentional (audit/review before money moves). |
| V4 | Workflow shape | **2-step (export + upload).** Confirmed: NO single "Process OnePay" button that triggers transfers directly from the timesheet page. The download/upload checkpoint is an explicit gate. |
| V5 | Skipped-employees UX | **Toast + detail dialog.** Toast shows count; "Xem danh sách" opens a dialog listing skipped employees + reasons. |
| V6 | Employee notifications | **ENABLE push notifications for ALL employee money receipts** (weekly + flexible + bulk). NOT silent. **Requires new design** — see "Notification Path" below. |

### Notification Path (NEW — in-scope per V6)

The existing `notifyEmployee` in `WalletPaymentService` (line 525) is **hardwired to `advance_payment_requests`**: it looks up `EntityID` via `advancePaymentReqs.GetByID`, which returns an `AdvancePaymentRequest` containing `EmployeeID`. This works for FlexPay/advance-payment flows.

For weekly + flexible payroll (including bulk transfers), there is NO `advance_payment_request`. The link from `wallet_payments` → employee must go via a different path.

**Design options (to be decided during Phase 3 implementation):**

| Option | Description | Tradeoff |
|--------|-------------|----------|
| **A** | Extend `notifyEmployee` to handle non-advance rows: if `EntityID == nil`, look up employee via `recipient_account_no → employees.bank_account_number`. | Minimal change; reuses notification infrastructure. But couples notifications to bank-account-number lookup (what if account no. is shared or stale?). |
| **B** | Add a new `notifyEmployeeForBulkTransfer` method on `WalletPaymentService` that resolves employee via the bulk-batch data (account_no → employee). | Cleaner separation; bulk-specific. But duplicates notification logic. |
| **C** | Persist `entity_id` on bulk `wallet_payments` rows pointing to a new lightweight `bulk_payment_employees` table that links bulk row → employee. | Most normalized but adds a table + migration. |

**Recommended: Option A** — least invasive. The bulk worker (Phase 3) calls `ensureBulkBatchLink` after `Initiate`; that method can also stamp `entity_id` resolved via `recipient_account_no`. Then the existing `notifyEmployee` FSM hooks fire automatically on `OnEnterCompleted`.

**Scope addition for Phase 3:**
- `ensureBulkBatchLink` also resolves + stamps `entity_id` (lookup employee by `recipient_account_no`).
- Test: bulk row reaching `completed` triggers `notifyEmployee` (verified by mock notification service assertion).

**Scope addition for Phase 2 (exporter):**
- The exporter's persisted VFIC → timesheet → employee link (via `TransactionCode` data) makes the employee resolution at worker time straightforward. The `bulk_transfer_batches.data` JSON includes `account_no`; the worker uses it to look up `employee.id` and stamps `entity_id`.

This applies to **ALL employee money receipts** per V6 — weekly, flexible, and bulk. Existing weekly/flexible payroll flows that don't currently notify will also benefit if they route through `wallet_payments` (which they do).


## Red Team Review

### Prior rounds (v1-v6 history)

| Round | Findings | Status |
|-------|----------|--------|
| v1→v2 (3 reviewers) | 30 findings, 6 Critical | Applied to v2 |
| v2→v3 (Failure Mode + Assumption Destroyer) | 14 findings, 8 Critical | Applied to v3 |
| v3→v4 (Security Adversary) | 14 findings, 3 Critical | Applied to v4 |
| v4→v5 (misread user) | — | v5 discarded |
| v5→v6 (SWIFT column) | — | Eliminated bank-resolution complexity |
| **v6→v7 (exporter)** | — | Added Stage 1 exporter |

### v7 retains all v3-v6 critical fixes

All red-team fixes for the Stage 2 worker, ledger booking, outbox, security hardening are retained. Stage 1 (exporter) is a new addition that reuses existing, proven code (`ExportPlanner`, `excel.Service`, `TransactionCode`). Expected red-team surface for Stage 1 is small.

A fresh red-team pass against v7 should focus on:
- Exporter reuse correctness (does `ExportPlanner.Plan()` actually return the data shape we expect?)
- SWIFT resolution edge cases (employee with bank_code that doesn't exist in banks table)
- VFIC generation race (two concurrent exports of overlapping timesheets)
- Download endpoint authorization (admin-only)
