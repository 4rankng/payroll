# Payroll System QA Test Plan

> **Version**: 1.0 | **Created**: 2026-06-12 | **Status**: Draft

## Overview

Comprehensive QA plan covering all 28 business flows in the Payroll system. Each flow includes test scenarios for happy path, state transitions, validation failures, cross-module integration, and edge cases.

## Plan Structure

| # | Document | Module | Flows Covered |
|---|----------|--------|---------------|
| 01 | [Master Test Matrix](01-master-test-matrix.md) | All | Cross-reference of all flows, risk levels, priority |
| 02 | [Auth & User Management](02-auth-user.md) | Auth, Users | Login, RBAC, profile, password |
| 03 | [Employee Management](03-employee-management.md) | Employee | CRUD, bank details, assignments, self-service |
| 04 | [Project Management](04-project-management.md) | Project | CRUD, assignments, salary periods, payrates |
| 05 | [Timesheet Operations](05-timesheet-operations.md) | Timesheet | Manual entry, calendar, validation, approval |
| 06 | [BCC Import](06-bcc-import.md) | Import | Standard BCC, Weekly BCC, auto-employee creation |
| 07 | [Bulk Transfer & Payment](07-bulk-transfer-payment.md) | Payment | Auto (9Pay), Manual, disbursement |
| 08 | [Advance Payment (FlexPay)](08-advance-payment.md) | FlexPay | Request window, fees, disbursement, cancellation |
| 09 | [Wallet & Financial](09-wallet-financial.md) | Wallet, Transaction, Ledger | Balance, sync, double-entry, settlements |
| 10 | [Loan Management](10-loan-management.md) | Loan, Lender | CRUD, disbursement, repayment schedules |
| 11 | [Sao Ke & Statements](11-saoke-statements.md) | Sao Ke | Email, reconciliation, settlement |
| 12 | [Dashboard & Reporting](12-dashboard-reporting.md) | Dashboard, Reporting | All dashboard views, metrics, exports |
| 13 | [Notifications & Push](13-notifications-push.md) | Notification | CRUD, push, email |
| 14 | [Infrastructure & Admin](14-infrastructure-admin.md) | Admin | Clock, cron, settings, audit, assets, bank |
| 15 | [End-to-End Scenarios](15-e2e-scenarios.md) | All | Full revenue chain, cross-module journeys |

## Core Business Chain (Primary Revenue)

```
Project Setup → Employee Assignment → Timesheet Entry (manual/BCC) → Approval → Bulk Transfer → Payment → Wallet Settlement
```

## Test Environment

- **Backend**: Port 8080, MySQL at `localhost:3306/payroll_db`
- **Auth**: JWT + Casbin RBAC (admin, partner roles)
- **Clock**: `clock.Now()` Asia/Ho_Chi_Minh timezone, FakeClock for testing
- **Provider**: 9Pay (sandbox/dev), OnePay (production)

## Risk-Based Priority

| Priority | Criteria | Example Flows |
|----------|----------|---------------|
| **P0 - Critical** | Revenue-impacting, money movement | Bulk Transfer, Advance Payment, Wallet, Ledger |
| **P1 - High** | Core operations, data integrity | Timesheet, BCC Import, Project/Employee CRUD |
| **P2 - Medium** | Supporting features | Dashboard, Notifications, Assets, Audit |
| **P3 - Low** | Admin/infrastructure | Clock, Settings, Metrics, Cron |

## Test Execution Strategy

1. **Phase 1 — Smoke** (P0 flows, happy path only): ~30 min
2. **Phase 2 — Core** (P0 + P1, all scenarios): ~3 hours
3. **Phase 3 — Extended** (P2 + P3, all scenarios): ~2 hours
4. **Phase 4 — E2E** (Full cross-module journeys): ~1 hour

## Existing Integration Test Coverage

Integration tests exist at `backend/tests/integration/flow_*.go`. This QA plan covers **both** the existing automated tests (documented for reference) and **additional manual/QA scenarios** that should be verified.
