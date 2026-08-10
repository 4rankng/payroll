# Phase 2: Worker Integration

**Status:** Pending  
**Dependencies:** Phase 1 complete

## Context

Integrate the FlexPayZNS service into the import job worker to send ZNS after successful FlexPay file import.

## Files to Modify

### 1. `backend/internal/app/workers/import_job_worker.go`

**Current `processJob` function (lines 54-91):**

```go
func (w *ImportJobWorker) processJob(ctx context.Context, assetID uint) error {
    // ... existing code ...
    
    if err := w.markJobCompleted(ctx, asset, result); err != nil {
        return err
    }
    
    // ← ADD ZNS HERE
    return nil
}
```

### 2. `backend/internal/app/services/advance_payment/admin_flexpay_import.go`

**Modify `ImportFlexPayFile` to return employee data for ZNS:**

The function already processes employee data at lines 180-297. We need to:
1. Collect successfully processed employees with their data
2. Return this data to the worker for ZNS sending

## Implementation Approach

### Option A: Return Employee Data from Import Service (Preferred)

Modify `ImportFlexPayFile` to return employee notification data:

```go
type EmployeeZNSData struct {
    EmployeeName string
    Mobile       string
    Amount       int64
    ExpiryDate   time.Time
}

func (s *AdminFlexPayImportService) ImportFlexPayFile(
    ctx context.Context,
    assetID uint,
    forMonth string,
) ([]EmployeeZNSData, error) {
    // ... existing import logic ...
    
    // Collect employees as they're processed
    for _, row := range rows {
        // ... existing employee creation ...
        
        employeeData := EmployeeZNSData{
            EmployeeName: fullName,
            Mobile:       mobile,
            Amount:       hanMuc,
            ExpiryDate:  monthEnd, // or appropriate date
        }
        employees = append(employees, employeeData)
    }
    
    return employees, nil
}
```

### Option B: Query After Import

Query the database after successful import to get employee data:
- More queries but less invasive
- Can handle data that was already in system

## Worker Integration Code

```go
// In import_job_worker.go

type ImportJobWorker struct {
    // ... existing fields ...
    znsService *zaloconnect.FlexPayZNSService
}

func (w *ImportJobWorker) processJob(ctx context.Context, assetID uint) error {
    // Get asset to determine import type
    asset, err := w.assetRepo.GetByID(ctx, assetID)
    if err != nil {
        return err
    }

    // Process import based on type
    var employeeData []zaloconnect.FlexPayZNSData
    var result *ImportResult

    switch asset.DataType {
    case "flexpay_import":
        employeeData, result, err = w.flexpayService.ImportFlexPayFile(ctx, assetID, forMonth)
        // ... handle result ...
        
    default:
        // ... other import types ...
    }

    if err != nil {
        return err
    }

    // Mark job completed
    if err := w.markJobCompleted(ctx, asset, result); err != nil {
        return err
    }

    // Send ZNS notifications (fire-and-forget)
    if w.znsService != nil && len(employeeData) > 0 {
        w.znsService.SendBatch(ctx, employeeData)
    }

    return nil
}
```

## Dependency Injection

### `backend/internal/infra/asynq/handlers.go`

Wire the ZNS service into the worker:

```go
// In handler initialization
znsService := zaloconnect.NewFlexPayZNSService(
    zaloProvider,
    logger,
)

importWorker := workers.NewImportJobWorker(
    // ... existing dependencies ...
    znsService, // ← Add this
)
```

## Data Flow

```
Excel Upload → Handler → Asynq Queue → Worker:
  1. Parse Excel
  2. Create/Update employees
  3. Create advance payment records
  4. Collect employee data for ZNS
  5. Mark job completed
  6. Send ZNS batch (async)
```

## Error Handling

- ZNS failures are logged but don't fail the import
- Employees without mobile numbers are skipped
- Invalid phone numbers are logged and skipped
- Template errors (e.g., -131 not approved) are logged per employee

## Testing

1. Unit test worker with mock ZNS service
2. Integration test with real Excel file
3. Verify ZNS sent to correct employees
4. Verify error handling

## Rollback

If ZNS causes issues, can disable by:
1. Setting `znsService = nil` in worker
2. Or adding feature flag check before sending
