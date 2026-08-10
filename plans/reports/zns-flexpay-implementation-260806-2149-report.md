# ZNS FlexPay Implementation Report

**Date:** 2026-08-06  
**Status:** ✅ Complete  
**Plan:** `260806-2149-zns-flexpay-notification`

---

## Summary

Implemented Zalo ZNS notifications for FlexPay salary notifications. When an Excel file (e.g., LGD) is uploaded for flexible projects, all employees with mobile numbers receive ZNS notifications with their accumulated income amount.

---

## Implementation

### Phase 1: FlexPayZNS Service ✅

**File Created:** `backend/internal/app/services/zaloconnect/flexpay_zns_service.go`

**Key Features:**
- Template ID: `619686` (SalaryNotification-v1)
- Parameters: `customer_name`, `max_amount`, `expiry_date`
- Phone normalization via existing `zalo.NormalizePhone()`
- Feature flag support via `IsEnabled()` check
- Fire-and-forget batch sending
- Graceful error handling (logs but doesn't fail import)

**Template Mapping:**
| Zalo Parameter | Source | Format |
|----------------|--------|--------|
| `customer_name` | Employee full name | Truncated to 30 chars |
| `max_amount` | Accumulated income | Raw number (VND) |
| `expiry_date` | End of forMonth | DD/MM/YYYY |

### Phase 2: Worker Integration ✅

**Files Modified:**

1. **`backend/internal/app/dto/advance_payment.go`**
   - Added `EmployeeZNSData` struct for notification data
   - Extended `ImportFlexPayFileResult` with `EmployeeZNSData []`

2. **`backend/internal/app/services/advance_payment/admin_flexpay_import.go`**
   - Collects employee data during Excel processing
   - Skips employees without mobile or zero amounts
   - Calculates expiry date as last day of `forMonth`

3. **`backend/internal/app/workers/import_job_worker.go`**
   - Added `FlexPayZNSService` field
   - Sends ZNS batch after successful import
   - Converts DTO data to service format

4. **`backend/internal/app/bootstrap/services/init.go`**
   - Created `flexPayZNSService` with `zaloProvider` + `IsEnabled` check
   - Added `FlexPayZNS` to `Services` struct

5. **`backend/internal/app/bootstrap/container.go`**
   - Passed `services.FlexPayZNS` to `ImportJobWorker`

### Phase 3: Tests ✅

**File Created:** `backend/internal/app/services/zaloconnect/flexpay_zns_service_test.go`

**Test Coverage:**
- Service construction
- Successful send
- No mobile skip
- Invalid phone skip
- Feature disabled skip
- Business error handling (-118)
- Batch sending
- Template data mapping
- String truncation
- Amount formatting

---

## Data Flow

```
Excel Upload → Handler → Asynq Queue → ImportJobWorker:
  1. Parse Excel file
  2. Create/Update employees + projects + assignments
  3. Collect employee data (name, mobile, amount, expiry)
  4. Mark job completed
  5. Send ZNS batch (fire-and-forget):
     - Normalize phone numbers
     - Build template data
     - Call zalo.Provider.Send()
     - Log results (success/failure)
```

---

## Configuration

**ZNS Template:** `619686` (SalaryNotification-v1)  
**Status:** Pending approval (Đang duyệt - 2-3 days)

**Cost:** ~300₫/message  
- Example: LGD batch with 162 employees = ~48,600₫

**Feature Toggle:** `zalo.enabled` settings table row  
- When `false`: ZNS silently skipped (no errors)

---

## Error Handling

| Scenario | Behavior |
|----------|----------|
| No mobile number | Skipped, logged |
| Invalid phone format | Skipped, logged |
| Template not approved (-131) | Logged per employee, doesn't fail import |
| No Zalo account (-118) | Logged per employee, doesn't fail import |
| Insufficient ZBS balance (-115/-137) | Logged per employee, doesn't fail import |
| Transport error | Logged, returned as error |

---

## LGD Excel File Structure

| Column | Data | ZNS Usage |
|--------|------|------------|
| 6 | Số điện thoại (mobile) | ZNS recipient |
| 8 | Họ và tên (full name) | `customer_name` |
| 11 | Mức thu nhập tích lũy (amount) | `max_amount` |

**Example Row:**
```
STT: 1 | Mã NV: 034204006956 | Bộ phận: LGD | Chức vụ: Nhân viên sản xuất
Chi nhánh: Hải Phòng | SĐT: 0366178061 | Tên: Nguyễn Việt Duy
STK: 0366178061 | Ngân hàng: MB BANK | Số tiền: 1680000
```

**Resulting ZNS:**
```
Recipient: 84366178061
Template: 619686
Data: {
  "customer_name": "Nguyễn Việt Duy",
  "max_amount": "1680000",
  "expiry_date": "31/08/2026"
}
```

---

## Dependencies

**Zalo Credentials:** Must be configured via admin UI or env vars
- App ID
- Secret Key  
- Access Token
- Refresh Token

**Template Approval:** Required before ZNS will succeed
- Error -131 will occur until approved

---

## Next Steps

1. **Wait for template approval** (2-3 business days)
2. **Test with LGD file** in staging environment
3. **Monitor logs** for ZNS send results
4. **Verify cost** per batch against ZBS balance

---

## Files Changed

| File | Change |
|------|--------|
| `backend/internal/app/services/zaloconnect/flexpay_zns_service.go` | Created |
| `backend/internal/app/services/zaloconnect/flexpay_zns_service_test.go` | Created |
| `backend/internal/app/dto/advance_payment.go` | Added `EmployeeZNSData` |
| `backend/internal/app/services/advance_payment/admin_flexpay_import.go` | Collect ZNS data |
| `backend/internal/app/workers/import_job_worker.go` | Send ZNS batch |
| `backend/internal/app/bootstrap/services/init.go` | Wire service |
| `backend/internal/app/bootstrap/container.go` | Pass to worker |

---

## Unresolved Questions

- None - implementation complete pending template approval

---

**Report generated:** 2026-08-06  
**Implementation time:** ~1 hour  
**Tests:** 12 test cases covering all major paths
