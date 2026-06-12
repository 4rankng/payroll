# 14 — Infrastructure & Admin (Clock, Cron, Settings, Audit, Assets, Bank)

**Priority**: P2-P3 | **API Prefix**: `/api/v1/admin/*`, `/api/v1/audit`, `/api/v1/assets`, `/api/v1/banks`, `/api/v1/settings` | **Risk Level**: Low-Medium

## Business Rules

### Clock Manipulation (Admin Only)
- Admin-only endpoint (auth + Casbin required)
- **Not mounted in production** (returns 404)
- FakeClock starts unfrozen (behaves like RealClock)
- `Set()` / `Advance()` freezes the clock
- `Reset()` unfreezes — restores auto-advance behavior
- `IsFrozen()` reports whether clock is manually set
- Time plausibility check: date must be after 2020
- After reset, time must be within 5 seconds of real time
- AutoFakeClock prevents outbox/cron workers from getting stuck with stale time

### Cron Jobs
- 10+ cron jobs registered in the system
- Each job can be toggled on/off
- Job status tracked in `cron_job_status` table
- Jobs run according to schedule unless disabled

### Settings
- Key-value store with `value_type` (string, int, bool, json)
- System-wide configuration (fee rates, feature flags, etc.)
- Standard CRUD operations

### Audit Logs
- Append-only — entries cannot be modified or deleted
- All import/export operations emit audit events
- Date range filtering available
- Audit diff tracking for entity changes

### Assets
- File upload with `upload_type` classification
- Binary download available
- Referenced by Sao Ke (saoKeAssetId), BCC Import (original file), Audit
- Soft-deleted

### Bank CRUD
- Simple reference table for bank branches
- Referenced by Employee (bank details) and Disbursement
- `branch_name` is required field

## State Machines

```
Clock: [real time] → (set/advance) → [frozen/fake] → (reset) → [real time]
Cron: [enabled] ↔ (toggle) ↔ [disabled]
Asset: [uploaded] → [deleted] (soft)
Bank: [created] → [deleted] (soft)
```

## Test Scenarios

### F29 — Clock Manipulation

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F29-01 | Get current server time | Happy | GET `/admin/clock` | 200, current time in Asia/Ho_Chi_Minh |
| F29-02 | Set time to specific date | Happy | POST `/admin/clock/set` with date | 200, clock frozen at specified date |
| F29-03 | Advance time by duration | Happy | POST `/admin/clock/advance` with duration | 200, clock advanced by specified amount |
| F29-04 | Reset to real time | Happy | POST `/admin/clock/reset` | 200, clock unfrozen, auto-advance restored |
| F29-05 | Set time before 2020 | Negative | POST with date before 2020 | 400, plausibility check failed |
| F29-06 | Clock not available in production | Negative | Call endpoint in prod environment | 404 (endpoint not mounted) |
| F29-07 | Reset restores auto-advance | Edge | Set → advance → reset → wait → check time | Time continues advancing (not stuck) |
| F29-08 | Advance while frozen adds time | Edge | Set time → advance by 1h → check | Time = set_time + 1h |
| F29-09 | Clock affects advance payment cutoff | Integration | Set to day 15 → request advance | Blocked (locked period) |
| F29-10 | Clock affects cron job execution | Integration | Set to scheduled time → check cron runs | Cron fires at manipulated time |

### F28 — Cron Jobs

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F28-01 | List all cron jobs | Happy | GET `/admin/cron` | 200, list of all jobs with status |
| F28-02 | Toggle job on | Happy | POST `/admin/cron/:id/toggle` (disabled → enabled) | 200, job enabled |
| F28-03 | Toggle job off | Happy | POST `/admin/cron/:id/toggle` (enabled → disabled) | 200, job disabled |
| F28-04 | Toggle non-existent job | Negative | POST `/admin/cron/99999/toggle` | 404 or accepted (verify behavior) |
| F28-05 | Disabled job does not fire | Integration | Disable job → wait for scheduled time → verify | Job did not execute |
| F28-06 | Re-enabled job resumes | Integration | Disable → enable → wait for schedule | Job executes on next schedule |

### F27 — Settings

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F27-01 | List current settings | Happy | GET `/settings` | 200, all settings with values |
| F27-02 | Create setting | Happy | POST `/settings` with key, value, value_type | 201, setting created |
| F27-03 | Get setting by ID | Happy | GET `/settings/:id` | 200, setting details |
| F27-04 | Update setting value | Happy | PUT `/settings/:id` with new value | 200, value updated |
| F27-05 | Delete setting | Happy | DELETE `/settings/:id` | 200, setting deleted |
| F27-06 | Create with invalid value_type | Negative | POST with value_type="invalid" | 400, invalid type |
| F27-07 | Update non-existent setting | Negative | PUT `/settings/99999` | 404 or error |
| F27-08 | Setting persists across restarts | Integration | Create → restart service → GET | Setting still exists |

### F25 — Audit Logs

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F25-01 | List audit logs | Happy | GET `/audit` | 200, paginated audit entries |
| F25-02 | Filter by date range | Happy | GET `/audit?fromDate=X&toDate=Y` | 200, filtered entries |
| F25-03 | Non-existent audit log | Negative | GET `/audit/99999` | 404 or error |
| F25-04 | Import creates audit entry | Integration | Upload BCC → check audit logs | Import event logged with details |
| F25-05 | Financial changes create audit | Integration | Create transaction → check audit | Transaction event logged |

### F24 — Assets

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F24-01 | Upload file | Happy | POST `/assets` with file + upload_type | 201, asset created with ID |
| F24-02 | List assets | Happy | GET `/assets` | 200, paginated asset list |
| F24-03 | Download file | Happy | GET `/assets/:id/download` | 200, binary file content |
| F24-04 | Get asset metadata | Happy | GET `/assets/:id` | 200, metadata without binary |
| F24-05 | Non-existent asset | Negative | GET `/assets/99999` | 404 or error |
| F24-06 | Asset linked to sao ke | Integration | Upload → use as saoKeAssetId → send email | Asset attached to email |
| F24-07 | Asset linked to BCC import | Integration | Upload BCC → check import detail | Original file downloadable |

### F26 — Bank CRUD

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F26-01 | Create bank | Happy | POST `/banks` with branch_name | 201, bank created |
| F26-02 | List banks | Happy | GET `/banks` | 200, bank list |
| F26-03 | Get bank by ID | Happy | GET `/banks/:id` | 200, bank details |
| F26-04 | Update bank | Happy | PUT `/banks/:id` with new branch_name | 200, bank updated |
| F26-05 | Delete bank | Happy | DELETE `/banks/:id` | 200, soft-deleted |
| F26-06 | Create without branch_name | Negative | POST without branch_name | 400, branch_name required |
| F26-07 | Update non-existent bank | Negative | PUT `/banks/99999` | 404 or error |
| F26-08 | Delete non-existent bank | Negative | DELETE `/banks/99999` | 404 or error |
| F26-09 | Bank used in employee details | Integration | Create bank → assign to employee → GET employee | Bank details visible |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_clock_manipulation.go`
- Integration test: `backend/tests/integration/flow_cron.go`
- Integration test: `backend/tests/integration/flow_settings.go`
- Integration test: `backend/tests/integration/flow_audit.go`
- Integration test: `backend/tests/integration/flow_assets.go`
- Integration test: `backend/tests/integration/flow_bank_crud.go`
- Integration test: `backend/tests/integration/flow_infra.go` (placeholder)
- Clock helpers: `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`, `GetServerTime`
