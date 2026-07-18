---
phase: 3
title: 'Backend: Add Endpoint & Wire Routes'
status: completed
priority: P1
effort: S
dependencies:
  - 1
  - 2
---

# Phase 3: Backend: Add Endpoint & Wire Routes

## Overview

Expose `SimulationService.Simulate` via a new admin-only endpoint `POST /api/v1/payrolls/simulate-settlement`, and wire an optional `IfMatchSnapshot` guard into the existing `ExportBulkTransfer` so a stale simulation cannot silently drive an export. Full DTO definitions live here.

## Requirements

- **Functional:** Admin-only endpoint returns `SimulationResult` JSON. Real export accepts optional `IfMatchSnapshot` and rejects with `409` when stale.
- **Non-functional:** No change to any existing endpoint's request/response shape. Permission enforced via existing `Authorize()` middleware already on the `/payrolls` group.

## Architecture

### Route + auth (red-team Finding 4 — explicit, not inherited)

`Authorize()` is Casbin RBAC (`backend/internal/transport/http/middleware/authorization.go:45`), NOT admin-default. The `/payrolls/*` group is admin-inherited **only by absence of partner/employee policy rows**, which is fragile. Phase 3 enforces admin TWO ways:

```go
// routes_disbursement.go — inside setupPayrollRoutes
payrolls.POST("/simulate-settlement", container.Handlers.Payroll.SimulateSettlement)
```

**AND** add an explicit Casbin policy row (red-team Finding 4):

```csv
# backend/configs/casbin_policy.csv — append
# Settlement simulation is admin-only (partner/employee get no row → denied)
# No explicit row needed for admin if admin has wildcard; verify by grep.
# If admin uses wildcards, the route is already allowed for admin. If admin uses
# explicit per-route rows, ADD:
# p, admin, /api/v1/payrolls/simulate-settlement, POST, allow
```

Verify the casbin model during implementation — the existing `export-bulk-transfer` route works for admin today, so mirror whatever mechanism makes THAT work for the new route.

**AND** add an in-handler admin check (belt-and-suspenders, mirrors `settle_from_notification.go:16`):

```go
// payroll_simulation_handler.go
func (h *PayrollHandler) SimulateSettlement(c *gin.Context) {
    // Defense-in-depth admin check (mirrors settlement/settle_from_notification.go:16)
    if !isAdminContext(c) {
        response.Forbidden(c, "Yêu cầu quyền admin")
        return
    }

    var req dto.SimulateSettlementRequest
    if err := c.ShouldBindJSON(&req); err != nil { response.BadRequest(c, ...); return }
    req.CreatedBy = userIDFromCtx(c) // for audit logging only — NOT a write

    result, err := h.payrollService.SimulateSettlement(c.Request.Context(), &req)
    if err != nil {
        if domain.IsValidationError(err) { response.BadRequest(c, err.Error()); return }
        h.logger.Error("simulate settlement", "error", err)
        response.InternalServerError(c, "Không thể mô phỏng đối soát")
        return
    }
    response.Success(c, result, "Mô phỏng đối soát thành công")
}
```

### DTO bank-account masking (red-team Finding 10)

The DTO assembler masks bank account numbers at the API boundary — the UI never sees raw numbers. Add a `maskBankAccount(s string) string` helper (keep last 4, prefix with `••••`):

```go
// In dto assembly:
BankAccountNumberMasked string `json:"bank_account_number"` // always masked, e.g. "••••1234"
// NO raw BankAccountNumber field is ever serialized.
```

Apply to every row type in `SimulationResult` (included, excluded, remaining, remainders items).

### Stale-snapshot guard on real export

```go
// export_service.go — inside Export(), right after planner.Plan() succeeds:
if req.IfMatchSnapshot != nil && !req.IfMatchSnapshot.IsZero() {
    if plan.SnapshotEpoch.After(*req.IfMatchSnapshot) {
        return nil, domain.ErrStaleSimulation  // mapped to 409 in handler
    }
}
```

DTO addition:
```go
type ExportBulkTransferRequest struct {
    // ... existing fields unchanged ...
    IfMatchSnapshot *time.Time `json:"if_match_snapshot,omitempty"` // optional; from prior simulation
}
```

Handler maps `domain.ErrStaleSimulation` → `response.Error(c, 409, "Dữ liệu đã thay đổi kể từ lần mô phỏng gần nhất — vui lòng chạy lại mô phỏng.")`.

**The guard is opt-in:** if `IfMatchSnapshot` is absent (the default), export behaves exactly as today. This keeps backward compatibility (acceptance criterion #7) and means the frontend can choose to pass it or not.

## API Contract (final)

### `POST /api/v1/payrolls/simulate-settlement`

**Request:**
```json
{
  "project_ids": [12],
  "employee_ids": [],
  "projected_cycle_count": 4,
  "for_month": null
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `project_ids` | `uint[]` | no | scope to projects; empty = all |
| `employee_ids` | `uint[]` | no | scope to employees; empty = all |
| `projected_cycle_count` | `int` | no | default 4, clamped to [1, 6] |
| `for_month` | `string` | no | `YYYY-MM`; if set, projects the 4 cycles of that month instead of starting from today |

**Response (200):**
```json
{
  "status": "success",
  "data": {
    "snapshot_epoch": "2026-07-17T03:24:11Z",
    "starting_cycle": { "index": 2, "month_ref": "2026-07-01", "from_date": "2026-07-08", "to_date": "2026-07-14", "pay_date": "2026-07-17" },
    "projected_cycle_count": 4,
    "verdict": "AN_TOAN_DE_XUAT",
    "summary": {
      "total_eligible_count": 42,
      "total_eligible_amount": 185000000,
      "total_included_count": 42,
      "total_included_amount": 185000000,
      "remaining_after_all_count": 0,
      "remaining_after_all_amount": 0,
      "all_settled": true
    },
    "reconciliation": {
      "exported_total": 185000000,
      "ledger_receivable": 185000000,
      "delta": 0,
      "reconciled": true
    },
    "cycles": [
      {
        "sequence": 1,
        "label": "Kỳ 2 (hiện tại)",
        "from_date": "2026-07-08",
        "to_date": "2026-07-14",
        "pay_date": "2026-07-17",
        "included_count": 18,
        "included_amount": 82000000,
        "excluded_count": 1,
        "remaining_after_count": 24,
        "remaining_after_amount": 103000000,
        "included": [
          { "employee_id": 7, "employee_name": "Nguyễn Văn A", "project_id": 12, "project_name": "Site A", "amount": 5000000, "timesheet_ids": [101, 102], "bank_account_number": "....1234" }
        ],
        "excluded": [
          { "employee_id": 9, "employee_name": "Trần B", "project_id": 12, "amount": 3000000, "timesheet_ids": [110], "reason": "Missing bank account number", "reason_code": "MISSING_BANK_ACCOUNT", "in_production": true }
        ],
        "remaining": [ /* same shape as included */ ],
        "findings": [
          { "severity": "warning", "code": "MISSING_BANK_CODE", "message": "...", "refs": [{"employee_id": 9}], "in_production": false }
        ]
      }
      // ... cycles 2, 3, 4
    ],
    "remainders": {
      "summary": {
        "total_count": 3,
        "total_amount": 12500000,
        "unpaid_wages_count": 1,    "unpaid_wages_amount": 5000000,
        "op_loss_count": 1,         "op_loss_amount": 4500000,
        "stuck_in_flight_count": 1, "stuck_in_flight_amount": 3000000
      },
      "items": [
        {
          "employee_id": 14, "employee_name": "Lê C", "project_id": 12, "project_name": "Site A",
          "amount": 5000000, "timesheet_ids": [88],
          "class": "UNPAID_WAGES",
          "reason": "Thuộc Kỳ 1 (ngày 1-7) nhưng không được bao phủ bởi các kỳ mô phỏng",
          "money_flow_evidence": null
        },
        {
          "employee_id": 14, "employee_name": "Lê C", "project_id": 12, "project_name": "Site A",
          "amount": 4500000, "timesheet_ids": [77],
          "class": "OP_LOSS",
          "reason": "Đã ứng lương qua ví (wallet_payment=completed) nhưng chưa tất toán công nợ",
          "money_flow_evidence": { "wallet_payment_id": 5521, "wallet_payment_status": "completed", "disbursed_at": "2026-07-05T08:00:00Z" }
        },
        {
          "employee_id": 21, "employee_name": "Phạm D", "project_id": 12, "project_name": "Site A",
          "amount": 3000000, "timesheet_ids": [66],
          "class": "STUCK_IN_FLIGHT",
          "reason": "Đang chờ kết quả ngân hàng (wallet_payment=authorised) — không xuất lại để tránh trùng",
          "money_flow_evidence": { "wallet_payment_id": 5530, "wallet_payment_status": "authorised", "initiated_at": "2026-07-09T10:00:00Z" }
        }
      ]
    },
    "warnings": [
      { "code": "PRODUCTION_DOES_NOT_VALIDATE", "message": "Sản xuất không kiểm tra mã ngân hàng, số tiền âm, hoặc trùng lặp giữa các kỳ. Xem chi tiết trong từng kỳ." },
      { "code": "PRIOR_CYCLE_STRAGGLERS", "message": "Có giao dịch thuộc kỳ trước vẫn chưa thanh toán — sẽ KHÔNG tự động bao phủ. Xem 'Còn lại'." }
    ]
  }
}
```

### `POST /api/v1/payrolls/export-bulk-transfer` (unchanged shape; new optional field)

```json
{
  "project_ids": [12],
  "fromDate": "2026-07-08",
  "toDate": "2026-07-14",
  "if_match_snapshot": "2026-07-17T03:24:11Z"  // ← NEW optional
}
```

Returns the existing `ExportBulkTransferResponse`, or **409** when `if_match_snapshot` is older than any relevant row's `updated_at`:
```json
{ "status": "error", "message": "Dữ liệu đã thay đổi kể từ lần mô phỏng gần nhất — vui lòng chạy lại mô phỏng.", "code": "STALE_SIMULATION" }
```

## Validation rules implemented (consolidated list with source)

| Rule | Where enforced | `in_production` flag |
|------|----------------|----------------------|
| Timesheet status = approved | `planner.Plan()` filter | ✅ prod |
| Payment status ∈ {pending, failed} | `planner.Plan()` filter | ✅ prod |
| Force-payroll admin override | `planner.Plan()` union | ✅ prod |
| Missing bank account number → exclude | `excel.ValidateAndFilterBulkTransferData` | ✅ prod |
| Missing bank account name → exclude | same | ✅ prod |
| Missing bank code → warning | sim validator | ❌ sim only |
| Zero/negative amount → blocking | sim validator | ❌ sim only |
| Duplicate timesheet across cycles → blocking | sim validator | ❌ sim only |
| Broken employee/project ref → blocking | sim validator | ❌ sim only (prod errors out) |
| Batch > 5000 rows → advisory warning | sim validator | ❌ sim only |
| Concurrent modification (snapshot drift across cycles) → warning | sim validator | ❌ sim only |
| Already-settled timesheet in eligible set → blocking | sim validator | ❌ sim only (prod status filter) |
| Currency/precision (always VND int64) → info | sim validator | ❌ sim only (trivially passes) |
| **Full-pool coverage gap detection (outstanding − covered)** | sim full-pool scan | ❌ sim only |
| **Remainder classification: UNPAID_WAGES / OP_LOSS / STUCK_IN_FLIGHT** | sim `remainder_classifier.go` | ❌ sim only |
| Reconciliation delta != 0 → affects verdict | sim reconciliation | ❌ sim only |
| Stale snapshot on real export → 409 | `export_service.Export` | ❌ new guard, opt-in |

## Related Code Files

- **Create:** `backend/internal/transport/http/handlers/payroll_simulation_handler.go`.
- **Modify:** `backend/internal/app/services/payroll/bulktransfer/service.go` — add `simulationService` field + `SimulateSettlement` method.
- **Modify:** `backend/internal/app/services/payroll/bulktransfer/export_service.go` — add `IfMatchSnapshot` check in `Export()`; add `domain.ErrStaleSimulation` mapping.
- **Modify:** `backend/internal/app/dto/payroll.go` — add `SimulateSettlementRequest`, `SimulationResult`, `CycleProjection`, `SimFinding`, `Reconciliation`, `VerdictSummary` DTOs; add `IfMatchSnapshot *time.Time` to `ExportBulkTransferRequest`.
- **Modify:** `backend/internal/app/bootstrap/routes_disbursement.go` — register `POST /payrolls/simulate-settlement`.
- **Modify:** `backend/internal/app/bootstrap/container*.go` — construct `SimulationService`, pass into `PayrollService`.
- **Modify:** `backend/internal/domain/errors.go` (or equivalent) — add `ErrStaleSimulation` sentinel + `IsValidationError` plumbing for 400 vs 409 distinction.
- **Modify:** `backend/internal/transport/http/handlers/payroll.go` — map `ErrStaleSimulation` → 409 in `ExportBulkTransfer`.

## Implementation Steps

1. **Define all DTOs** in `dto/payroll.go`. Mirror the JSON shapes above exactly — the frontend types in Phase 4 are generated from these.
2. **Add `domain.ErrStaleSimulation`** sentinel + ensure the response layer maps it to 409. Mirror how other domain errors (e.g. `domain.ErrValidationError`) are mapped in `response/`.
3. **Add `IfMatchSnapshot` check** in `ExportService.Export()` between `Plan()` and `Persist()`.
4. **Wire `IfMatchSnapshot`** through `ExportBulkTransferRequest` (it's already deserialized — just read the new optional field).
5. **Implement `PayrollHandler.SimulateSettlement`** in the new handler file.
6. **Implement `PayrollService.SimulateSettlement`** delegating to `simulationService`.
7. **Construct `SimulationService`** in the bootstrap container with its deps (`planner`, `ledgerRepo`, `clock`, `db`).
8. **Register the route** in `setupPayrollRoutes`.
9. **Manual smoke test** with curl: `curl -X POST -H "Authorization: Bearer $T" -d '{"projected_cycle_count":4}' http://localhost:8080/api/v1/payrolls/simulate-settlement | jq`.

## Success Criteria

- [ ] `POST /api/v1/payrolls/simulate-settlement` returns 200 with the `SimulationResult` shape above.
- [ ] Non-admin token → 403.
- [ ] `POST /api/v1/payrolls/export-bulk-transfer` without `if_match_snapshot` behaves exactly as before (backward compatible).
- [ ] `POST /api/v1/payrolls/export-bulk-transfer` with a stale `if_match_snapshot` returns 409.
- [ ] `POST /api/v1/payrolls/export-bulk-transfer` with a current `if_match_snapshot` succeeds normally.
- [ ] `make api-test` green (existing suite unaffected).

## Risk Assessment

**Risk: container wiring churn breaks other PayrollService consumers.**
Mitigation: add `simulationService` as a new constructor param at the **end**; update the single bootstrap call site. No existing caller changes.

**Risk: 409 mapping is inconsistent with the project's error convention.**
Mitigation: grep `response.` package for existing non-200 examples; mirror the closest one (likely `response.Error(c, http.StatusConflict, ...)` or a domain-error mapper).

**Risk: `IfMatchSnapshot` field name collides or confuses frontend.**
Mitigation: documented in DTO comment; Phase 4 makes it explicit in the service method. Optional field — absent = today's behavior.
