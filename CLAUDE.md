## IMPORTANT
- Everytime you develop a new feature, you must run make api-test to ensure no regression
- Update the testing flow in backend/tests/integration with test scenarios and expected outcomes to guard against regression in the future development

## Architecture & Key Patterns

### Centralized Clock (`internal/pkg/clock/`)
- **All business time** uses `clock.Now()` (global) or injected `clock.Clock` interface
- `clock.Now()` returns time in Asia/Ho_Chi_Minh timezone
- `clock.NowUTC()` returns UTC time
- Handlers receive `clock.Clock` via DI (constructor injection)
- Infrastructure/services use global `clock.Now()` (no DI needed)
- **Non-prod environments**: FakeClock (auto-advance mode) with admin manipulation endpoint
  - `NewAutoFake()` starts unfrozen — behaves like RealClock until `Set()`/`Advance()` freezes it
  - `Reset()` unfreezes — restores auto-advance behavior for background workers
  - `IsFrozen()` reports whether clock is manually set
  - This prevents outbox/cron workers from getting stuck with stale time after clock manipulation tests
- **Production**: RealClock — manipulation endpoints not mounted (404)
- Admin endpoint: `GET/POST /api/v1/admin/clock/*` (auth + Casbin required)
- Integration test helpers: `SetServerTime`, `AdvanceServerTime`, `ResetServerTime`, `GetServerTime` in `tests/integration/clock_helper.go`
- **Exclusions**: Event bus duration measurements (`redis_event_bus.go`, `event_bus_with_workers.go`) and health check response-time measurement use `time.Now()` directly (not business time)

### Cache Invalidation
- `ProjectEmployeeService.UpdateAssignment` invalidates Redis cache key `assignment:{projectID}:{employeeID}` after commit
- Cache is used by `TimesheetValidationService.ValidateEmployeeAssignment` to avoid repeated DB lookups
- TTL: `constants.EmployeeAssignmentCacheTTL`
- **Lesson**: When updating assignments, always invalidate cache after transaction commit, not before

### 9Pay Integration
- IPN (Instant Payment Notification) is async — arrives ~3s after batch completion
- Tests must poll for `payment_status='paid'` rather than checking immediately
- Manual bulk transfer result upload marks timesheets as paid synchronously
- 9Pay mock sandbox sends IPNs to `http://host.docker.internal:8080/api/v1/webhooks/disbursement/9pay`

### Test Data Pollution
- `cleanupEmployeeTimesheets` skips paid timesheets (API blocks deletion of paid records)
- Previous test runs can leave paid timesheets that occupy all valid dates
- **Mitigation**: Direct DB cleanup via `DELETE FROM timesheets WHERE project_id=X AND employee_id=Y`
- Monthly employee assignment `start_date` cannot be moved backward if approved/paid timesheets exist after the new date (domain validation: `HasNonEditableTimesheetsAfterDate`)

### Integration Test Patterns
- **Always use `pageSize=200`** when querying timesheets in test polling — `pageSize=50` causes false failures when project has 60+ timesheets (timesheet ends up on page 2)
- `findAvailableDateWithMin(client, projectID, employeeID, start, minDate)` is the preferred date finder; respects assignment start_date to avoid `thời gian phân công nhân viên bắt đầu sau ngày chấm công` validation errors
- `TestData.WeeklyAssignment` / `TestData.MonthlyAssignment` store `EmployeeProjectInfo` (includes `StartDate`) from employee's `CurrentProjects` during discovery
- `findAvailableDateForward`: searches forward only, returns non-future dates
- When querying timesheets in tests, always include `employee_id` filter to avoid pagination issues
- Backend port: 8080, DB: `root:rootpassword@tcp(localhost:3306)/payroll_db`
- **Monthly timesheet tests skip gracefully** when all dates between assignment start_date and today are occupied by paid timesheets (data pollution from prior runs)
- Monthly employee assignment `start_date` can be very recent (e.g., yesterday), leaving no unoccupied dates between start and today
- **Date exhaustion**: After many test runs, all weekdays from assignment start to today can be occupied. Fix: hard-delete test employee's timesheets from DB before running tests
- **Pre-test DB cleanup command**: `docker exec payroll-mysql mysql -uroot -prootpassword payroll_db -e "DELETE FROM timesheets WHERE project_id=66 AND employee_id=691"`

## graphify

This project has a knowledge graph at graphify-out/ with god nodes, community structure, and cross-file relationships.

Rules:
- For codebase questions, first run `graphify query "<question>"` when graphify-out/graph.json exists. Use `graphify path "<A>" "<B>"` for relationships and `graphify explain "<concept>"` for focused concepts. These return a scoped subgraph, usually much smaller than GRAPH_REPORT.md or raw grep output.
- If graphify-out/wiki/index.md exists, use it for broad navigation instead of raw source browsing.
- Read graphify-out/GRAPH_REPORT.md only for broad architecture review or when query/path/explain do not surface enough context.
- After modifying code, run `graphify update .` to keep the graph current (AST-only, no API cost).
