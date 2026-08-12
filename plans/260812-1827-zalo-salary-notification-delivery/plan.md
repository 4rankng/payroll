---
title: "Zalo salary notification delivery"
description: "Deliver the approved SalaryNotification-v1 ZNS after requestable flexible payroll imports."
status: completed
priority: P1
effort: ""
tags: []
created: 2026-08-12
---

# Zalo salary notification delivery

## Overview

Repair the existing ZNS flow after an Admin uploads the flexible salary/limit workbook. This is the only upload path that persists the employee's requestable FlexPay amount. BCC imports remain out of scope: they reject flexible employees and cannot make a salary-payment request available.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Send `SalaryNotification-v1` (`619686`) only to eligible flexible employees after a successful salary/limit import | P1 |
| 2 | Make the Admin switch and sample send operate on that exact notification | P1 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Phase 1: Durable delivery and Settings](./phase-01-start.md) | Completed |

## Success Criteria

- [x] `zns.flexpay_enabled` is the authoritative hot toggle for salary ZNS delivery.
- [x] A successful import persistently schedules one delivery per upload, project, employee, and template; queue failures recover without unbounded sends.
- [x] A successful import notifies only a positive-amount employee with a mobile number, flexible payment schedule, and `check_in_enabled=false`.
- [x] The test action delivers `619686` with `customer_name`, `max_amount`, and `expiry_date` sample values.
- [x] The shared Settings component remains usable at desktop and mobile widths, with focused backend/frontend tests passing.

## Verification

- Focused backend race tests and frontend Settings tests pass.
- Frontend lint and production build pass (lint has three existing generated-coverage warnings).
- `make -C backend api-test` requires a running local API; it could not connect to `localhost:8080` in this workspace.

<!-- slug: zalo-salary-notification-delivery -->
