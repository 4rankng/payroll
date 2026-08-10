# Phase 3: Testing & Verification

**Status:** Pending  
**Dependencies:** Phase 1 & 2 complete

## Test Plan

### 1. Unit Tests

#### `flexpay_zns_service_test.go`

```go
func TestNormalizePhone(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"0366178061", "84366178061"},
        {"84366178061", "84366178061"},
        {"+84366178061", "84366178061"},
        {"", ""},
    }
    // ... test implementation
}

func TestFormatAmount(t *testing.T) {
    tests := []struct {
        amount   int64
        expected string
    }{
        {1680000, "1680000"},
        {15000000, "15000000"},
    }
    // ... test implementation
}

func TestTemplateDataMapping(t *testing.T) {
    // Verify template data is correctly formatted
}

func TestSendSalaryNotification_MockProvider(t *testing.T) {
    // Mock provider and verify Send is called with correct params
}
```

### 2. Integration Tests

#### Test with Sample Excel File

Use the LGD Excel file structure:

```go
func TestFlexPayImport_WithZNS(t *testing.T) {
    // 1. Create test Excel file with employee data
    // 2. Upload via import endpoint
    // 3. Wait for async job to complete
    // 4. Verify ZNS was sent to all employees
    // 5. Verify template parameters match Excel data
}
```

### 3. Manual Testing

#### Test Checklist

- [ ] Upload LGD Excel file via admin UI
- [ ] Verify employees are created/updated correctly
- [ ] Verify ZNS is sent to all employees with mobile numbers
- [ ] Verify ZNS content is correct (name, amount, date)
- [ ] Verify employees without mobile are skipped gracefully
- [ ] Verify import succeeds even if ZNS fails
- [ ] Check logs for ZNS send results

#### Test Scenarios

| Scenario | Expected Result |
|----------|-----------------|
| Normal employee with mobile | ZNS sent successfully |
| Employee without mobile | Skipped, logged |
| Invalid phone format | Logged as error, doesn't fail import |
| Template not approved (-131) | Logged as error per employee |
| No Zalo account (-118) | Logged as error per employee |
| Empty Excel | No ZNS sent |
| Excel with 162 rows (real LGD) | All 162 employees get ZNS |

### 4. Verification Commands

```bash
# Run backend tests
cd backend && go test ./internal/app/services/zaloconnect/... -v

# Run integration tests
cd backend && go test ./tests/integration/... -v -run TestFlexPay

# Check logs for ZNS sends
grep "ZNS sent" logs/app.log
grep "failed to send ZNS" logs/app.log
```

### 5. Monitoring After Deployment

- Check error logs for ZNS failures
- Monitor ZBS balance (error -115/-137)
- Track successful send rate
- Verify template approval status

## Success Criteria

- All unit tests pass
- Integration test with real Excel succeeds
- Manual test with LGD file sends ZNS to 162 employees
- No errors in import process
- Template parameters match Excel data exactly

## Rollback Plan

If issues found:
1. Disable ZNS via feature flag
2. Remove ZNS call from worker
3. Investigate logs for root cause
4. Fix and redeploy

## Known Risks

1. **Template not approved** - ZNS will fail with -131 until Zalo approves template (2-3 days)
2. **Employee coverage** - Only employees with mobile numbers will receive ZNS
3. **Cost** - 300₫/message × ~162 employees = ~48,600₫ per batch
