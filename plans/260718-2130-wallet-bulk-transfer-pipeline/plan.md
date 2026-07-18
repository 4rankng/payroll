---
title: "Wallet Bulk Transfer Pipeline"
description: >-
  Admin uploads a "Yêu cầu chuyển tiền" Excel on /admin/wallet; the system
  parses it, initiates OnePay transfers per row (3 TPS throttled), records
  success/failure per row, books the aggregate OnePay fee (3,850 VND/txn from
  db schedule — charged regardless of outcome) as one Expense ledger entry per
  batch, and produces a downloadable "KQ Chuyen Tien" Excel mirroring the
  eMB_BulkPayment layout with status, FT number, and fee columns.
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
blockedBy: []
blocks: []
created: "2026-07-18T14:14:34.945Z"
createdBy: "ck:plan"
source: skill
---

# Wallet Bulk Transfer Pipeline

## Overview

Add a **file-driven bulk disbursement entry point** on the wallet page. Admin uploads the standard Vietcombank-style "Yêu cầu chuyển tiền" Excel (sheet `eMB_BulkPayment`); the backend parses the rows, initiates a OnePay `funds_transfers` call per row through the existing OnePay queue (3 TPS), tracks each row's outcome through the existing `wallet_payments` FSM, books the **aggregate** per-batch OnePay fee (3,850 VND/txn, charged on success **and** failure — OnePay charges whenever the endpoint is invoked) as a single settled `Expense` ledger transaction with party `OnePay`, and generates a downloadable "KQ Chuyen Tien" Excel mirroring the eMB result format (status, FT number/FT error, fee column).

This is the **write-side companion** to the read-only bank-transfer-history plan (`260718-1950-bank-transfer-history`, completed). It reuses the existing `bulktransfer` package internals (planner, result processor, ledger builder) and the OnePay queued provider. It does **not** reuse `NinePayBulkTransferService` directly because OnePay has no bulk API — each row is a separate throttled `PUT /funds_transfers` call.

### Key architectural decisions (locked)

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | **Auto-pipeline via OnePay API** (no manual portal roundtrip) | Admin uploads once; system initiates transfers via the existing OnePay queued provider and produces KQ automatically. End-to-end automated. |
| 2 | **Reject duplicate uploads** by VFIC transaction code | Each row's `Nội dung chuyển khoản` contains a unique VFIC code. If any row's code matches an existing non-failed `wallet_payments` row, the whole upload is rejected with 409 + conflict list. Prevents double-charges. |
| 3 | **Aggregate fee ledger per batch** (mirror `onepay_fee_import_service.go`) | One settled `Expense` transaction per upload batch = `sum(fee)` across all rows. Party `OnePay`. Debit Expense / Credit Cash. Mirrors the existing monthly aggregate pattern; avoids N ledger entries per batch. |
| 4 | **DB fee schedule is source of truth** (3,850 VND/txn) | Resolve via `DisbursementFeeScheduleEntry` active at upload time. The KQ `PHÍ` column from OnePay is ignored for booking — the DB schedule is the audited source. |

## Scope

**In:**
- New domain entity `BulkTransferBatch` (one per upload) linking N `wallet_payments` rows.
- Excel parser for "Yêu cầu chuyển tiền" (sheet `eMB_BulkPayment`, header at row 2, data row 3+, columns: STT, Account No., Beneficiary, Beneficiary Bank, Amount, Payment Detail).
- Excel generator for "KQ Chuyen Tien" mirroring the attached `KQ Chuyen Tien.xlsx` layout (header rows 1-4, data row 5+, 9 columns including PHÍ, status, FT number/error).
- Upload + process pipeline: parse → validate → create `wallet_payments` rows (status `pending`) → enqueue per-row OnePay transfers → aggregate results → book ledger → mark batch complete.
- Duplicate-rejection guard by VFIC code.
- Aggregate ledger booking: one Expense txn per batch = `sum(fee)` across all rows (success + failure).
- New HTTP routes under `/api/v1/wallet/bulk-transfer/*` (Admin only).
- Frontend: new "Tải lên chuyển tiền" dialog on `/admin/wallet` with upload, progress, per-row status polling, KQ download, batch history list.
- Asynq worker for per-row OnePay execution (extends existing worker pattern).

**Out:**
- Changes to existing 9Pay bulk transfer flow on `/payrolls`.
- Changes to `wallet_payments` FSM, status enum, or `DisbursementFeeSchedule` schema.
- Manual KQ override/re-upload (deferred — auto-generated KQ is authoritative).
- Monthly OnePay fee import changes (existing `onepay_fee_import_service.go` continues unchanged; per-batch booking is new and separate).
- Reverse/refund workflow for failed rows.
- Mobile-specific upload UI (mobile falls back to same dialog; upload UX optimized for desktop).
- Changes to transaction-code strategies (`transaction_code_strategy.go`) — VFIC codes are read as-is from the Excel.

## Acceptance Criteria

- [ ] Admin can upload a "Yêu cầu chuyển tiền" `.xlsx` (≤10MB, sheet `eMB_BulkPayment`) via `/admin/wallet`.
- [ ] Upload with any row whose VFIC code (from `Nội dung chuyển khoản`) matches an existing `wallet_payments` row in status `pending`/`verified`/`authorised`/`completed` is rejected with 409 and a list of conflicting codes.
- [ ] Each parsed row creates exactly one `wallet_payments` row (status `pending`) with `RequestID` derived from the VFIC code, `fee` stamped from the active `DisbursementFeeScheduleEntry` for provider `1pay`.
- [ ] One `InitiateTransfer` call is made per row via the OnePay queued provider (3 TPS throttle).
- [ ] Each row's terminal status (`completed` or `failed`) is reflected in `wallet_payments` within 60s of upload completion for typical 50-row batches.
- [ ] On batch completion, exactly one `transactions` row is created: `type=expense`, `party=OnePay`, `amount=sum(fee)` across all rows in the batch (success **and** failure), `status=settled`. Ledger entries: Debit Expense / Credit Cash.
- [ ] A "KQ Chuyen Tien" `.xlsx` is generated and downloadable, matching the attached reference layout (sheet `data`, header rows 1-4, 9 columns, FT number on success / error reason on failure).
- [ ] KQ `TRẠNG THÁI` column shows `Thành công` for completed rows and `Thất bại` for failed rows.
- [ ] KQ `PHÍ` column shows the per-row fee stamped from DB schedule (3,850 by default).
- [ ] Batch list (`GET /wallet/bulk-transfer/batches`) shows upload history with totals, counts, status.
- [ ] All money math uses `int64` VND (no floats). All time uses `clock.Now()` (Asia/Ho_Chi_Minh).
- [ ] Existing 9Pay bulk transfer on `/payrolls` and the read-only bank-transfer-history page are unchanged.
- [ ] `make api-test` passes; new integration tests cover happy path, duplicate rejection, partial-failure ledger booking.
- [ ] Frontend `pnpm lint && pnpm type-check` pass.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Domain & Parser](./phase-01-domain-parser.md) | Pending |
| 2 | [Backend Pipeline & Ledger](./phase-02-backend-pipeline-ledger.md) | Pending |
| 3 | [KQ Excel & Routes](./phase-03-kq-excel-routes.md) | Pending |
| 4 | [Frontend Upload UI](./phase-04-frontend-upload-ui.md) | Pending |
| 5 | [Verification](./phase-05-verification.md) | Pending |

## Dependencies

- **Phase 2 → Phase 1**: Pipeline service depends on `BulkTransferBatch` domain, repository, and parser.
- **Phase 3 → Phase 2**: KQ generator reads batch + wallet_payments state; routes wrap the pipeline service.
- **Phase 4 → Phase 3**: Frontend calls the final HTTP contract.
- **Phase 5 → all**: Integration tests exercise the full pipeline.
- **No external-plan dependencies.** The completed `260718-1950-bank-transfer-history` plan is read-only and unaffected.

## Risk Assessment

| Risk | Severity | Mitigation |
|------|----------|------------|
| OnePay charges 3,850 VND per row even on failure — a bad upload burns money fast | **High** | Duplicate-rejection guard + dry-run validation pass (account number format, amount ≥100,000 VND, bank code resolvable) before any OnePay call. Reject whole batch on any validation failure. |
| OnePay 3 TPS throttle = 50 rows takes ~17s; 500 rows takes ~3min | Medium | Asynq worker processes rows concurrently within TPS limit; frontend polls batch status every 5s; UI shows progress bar (`completed+failed / total`). |
| Per-row OnePay failure mid-batch leaves batch half-done | Medium | Each row's status is independently tracked in `wallet_payments` FSM. Batch is "completed" when all rows reach terminal state. Ledger booking waits for batch completion (asynq continuation task). |
| Race: admin uploads same file twice in quick succession | Medium | `BulkTransferBatch` table has unique index on `content_hash` (SHA-256 of normalized row tuples). Second upload hits duplicate-key → 409. |
| Fee schedule changes mid-batch | Low | Fee is stamped at row creation time (upload moment), not at OnePay call time. Stable per batch. |
| Excel format drift (column reordering, language change) | Medium | Parser uses header-name mapping (Vietnamese + English aliases), not fixed offsets. Rejects unknown headers with explicit error. |

## Non-Goals

- No new fee schedule types or wallet statuses.
- No reverse/refund/chargeback workflow.
- No manual KQ re-upload override.
- No provider switcher UI — OnePay is hardcoded as the active provider for this pipeline (matches production).
- No SMS/email notifications beyond existing `notifyEmployee`/`notifyInitiator` calls already in `WalletPaymentService`.
- No changes to the `BulkTransferFile` domain (that's the existing 9Pay timesheet-driven flow; this plan adds a parallel `BulkTransferBatch` for the wallet-upload flow).
