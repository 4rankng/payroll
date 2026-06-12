# 12 — Dashboard & Reporting

**Priority**: P2 | **API Prefix**: `/api/v1/dashboard` | **Risk Level**: Low

## Business Rules

- All dashboard endpoints are read-only (no state mutations)
- Data aggregated from all modules (employees, projects, timesheets, transactions, wallet, ledger)
- Partner-specific dashboard views available (filtered to partner's projects)
- Historical data available with date range filters
- Metrics endpoint provides system health and API performance data

## Dashboard Views

| View | Description | Data Sources |
|------|-------------|-------------|
| Summary | High-level KPIs | All modules |
| Financial Overview | Revenue, expenses, balances | Ledger, Wallet, Transactions |
| Financial | Detailed financial breakdown | Ledger, Transactions |
| Salary Distribution | Salary payment distribution | Timesheets, Bulk Transfers |
| Recent Activities | Latest system actions | Audit logs, Events |
| Notifications | Recent notifications | Notification service |
| New Employees | Recently added employees | Employee service |
| Historical | Historical data with date range | All modules |
| Monthly Financials | Monthly financial reports | Ledger, Transactions |
| Employee Activity | Employee work patterns | Timesheets, Attendance |
| Top Paid Employees | Highest-paid employees | Timesheets, Payroll |
| Bank Usage | Payment methods by bank | Wallet, Disbursement |
| Bank Usage by Project | Bank usage per project | Wallet, Projects |
| Project Profitability | Revenue vs cost per project | Transactions, Ledger |
| Project Weekly Profit | Weekly profit trends | Timesheets, Payroll |
| Partner Dashboard | Partner-specific summary | Partner's projects |
| Partner Employees | Partner's employee list | Partner's assignments |

## Test Scenarios

### F22 — Dashboard

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F22-01 | Get summary dashboard | Happy | GET `/dashboard/summary` | 200, aggregated KPIs (employee count, project count, financials) |
| F22-02 | Get financial overview | Happy | GET `/dashboard/financial-overview` | 200, revenue, expenses, profit |
| F22-03 | Get financial dashboard | Happy | GET `/dashboard/financial` | 200, detailed financial breakdown |
| F22-04 | Get salary distribution | Happy | GET `/dashboard/salary-distribution` | 200, salary distribution data |
| F22-05 | Get recent activities | Happy | GET `/dashboard/recent-activities` | 200, latest system events |
| F22-06 | Get notifications | Happy | GET `/dashboard/notifications` | 200, recent notifications |
| F22-07 | Get partner dashboard | Happy | GET `/dashboard/partner` (partner token) | 200, partner's project data only |
| F22-08 | Get partner employees | Happy | GET `/dashboard/partner/employees` (partner token) | 200, partner's assigned employees |
| F22-09 | Dashboard reflects new data | Integration | Create employee → GET summary | Employee count incremented |
| F22-10 | Financial dashboard reflects transactions | Integration | Create transaction → GET financial | Amounts reflected in financials |
| F22-11 | Partner cannot see other partners' data | Negative | Partner A GET dashboard → verify no Partner B data | Only own projects visible |

### F32 — Metrics

| # | Scenario | Type | Steps | Expected Result |
|---|----------|------|-------|-----------------|
| F32-01 | Get API metrics | Happy | GET `/metrics/api` | 200, request counts, response times |
| F32-02 | Get event bus metrics | Happy | GET `/metrics/event-bus` | 200, event processing stats |
| F32-03 | Get cache metrics | Happy | GET `/metrics/cache` | 200, hit/miss rates |
| F32-04 | Get recent errors | Happy | GET `/metrics/errors` | 200, recent error log |
| F32-05 | Get latency trend | Edge | GET `/metrics/latency-trend` | 200, latency over time |
| F32-06 | Metrics reflect API activity | Integration | Make API calls → GET metrics | Request counts updated |

## Automated Test Reference

- Integration test: `backend/tests/integration/flow_dashboard.go`
- Integration test: `backend/tests/integration/flow_metrics.go`
