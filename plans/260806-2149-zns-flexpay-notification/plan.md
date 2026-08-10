# ZNS Notification for FlexPay Import

**Status:** Draft  
**Created:** 2026-08-06  
**Issue:** ZNS notifications for flexible project payments

## Overview

Send Zalo ZNS notifications to employees when FlexPay Excel files (e.g., LGD) are uploaded for flexible projects. Employees receive their accumulated income amount via ZNS after successful import.

## ZNS Template

**Template ID:** `619686` (SalaryNotification-v1)  
**Status:** Pending approval (Đang duyệt - 2-3 days)

| Parameter | Type | Max | Source |
|-----------|------|-----|--------|
| `customer_name` | string | 30 | Excel: Họ và tên (Col 8) |
| `max_amount` | number | 20 | Excel: Mức thu nhập tích lũy (Col 11) |
| `expiry_date` | date | 20 | Payment/processing date |

## Phases

1. [Create ZNS Service](phase-01-zns-service.md) - Extract ZNS sending logic
2. [Integrate with Worker](phase-02-worker-integration.md) - Add ZNS call after import
3. [Testing & Verification](phase-03-testing.md) - Test with real data

## Dependencies

- ZNS Template 619686 approval pending
- Existing Zalo credentials (already configured)

## Acceptance Criteria

- ZNS sent to all employees with mobile numbers in uploaded Excel
- `max_amount` = accumulated income from Excel column 11
- Failures logged but don't block import
- Phone numbers normalized to `84xxxxxxxxx` format
- Template parameters correctly mapped

## Risks

- Template not approved → ZNS will fail with error -131
- Employees without Zalo accounts → error -118 (expected)
- Insufficient ZBS balance → error -115/-137
