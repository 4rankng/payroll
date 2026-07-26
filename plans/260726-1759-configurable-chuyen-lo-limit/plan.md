---
title: "Configurable Chuyển lô workbook limit"
description: "Admin-configurable strict total threshold for manual MBank Chuyển lô workbooks"
status: in-progress
priority: P2
branch: "main"
tags: [settings, payroll, bulk-transfer, frontend]
blockedBy: []
blocks: []
created: "2026-07-26T10:03:10.989Z"
createdBy: "ck:plan"
source: skill
---

# Configurable Chuyển lô workbook limit

## Overview

Replace the hardcoded `500_000_000` VND manual Chuyển lô workbook threshold
with the existing Settings authority. Seed the new setting at `400_000_000`
VND, preserve strict `< threshold` behavior, capture it once per export, and
expose a full-VND Admin control on both desktop and mobile Settings pages.

## Exact requirements

- **Expected output:** an Admin can edit `bulk_transfer_workbook_limit_vnd` on
  both Settings routes, and the next manual MBank Chuyển lô export uses that
  persisted value to validate and partition generated workbooks.
- **Acceptance:** the default is `400.000.000 đ`; `399.999.999` is allowed,
  while an individual `400.000.000` or larger row is rejected and a workbook
  is split before its total reaches the threshold; failed generation creates
  no transaction-code writes.
- **Scope boundary:** OnePay export, automatic provider transfer, wallet
  upload/results, file-size/row limits, payment percentages, and accounting are
  unchanged.
- **Constraints:** reuse the existing Go Settings row/API and React settings
  primitives; whole VND uses strict base-10 `int64`; no new endpoint/table or
  response shape; Vietnamese copy; desktop 1280 and mobile 390/320 parity.
- **Touchpoints:** typed settings config/service, Excel generation and export
  ordering, idempotent seed/migration, shared settings hook/card, both Admin
  settings routes, focused tests, QA documentation, and graphify index.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Backend setting](./phase-01-backend-setting.md) | Completed |
| 2 | [Export enforcement](./phase-02-export-enforcement.md) | Completed |
| 3 | [Frontend UI](./phase-03-frontend-ui.md) | In Progress |
| 4 | [Verification docs](./phase-04-verification-docs.md) | In Progress |

## Dependencies

1 → 2 → 3 → 4. Phase 3 may begin after the Phase 1 persisted key and validation
contract are fixed. No cross-plan dependency.

## Acceptance criteria

- [x] Migration and development seeder create `bulk_transfer_workbook_limit_vnd`
      as number `400000000` without overwriting an existing Admin value.
- [x] Backend rejects null, non-integer, `< 2`, or `> math.MaxInt64` values for
      this known key and falls back to 400M when a stored value is unavailable
      or corrupt.
- [x] The next export sees a successful Admin update without process restart
      and uses one immutable threshold for the full export.
- [x] Strict boundary, deterministic partitioning, overflow safety, XLSX/ZIP
      output, and no-write-on-generation-failure are covered by tests.
- [ ] Desktop and mobile Admin pages show the same accessible, formatted VND
      control and complete loading/error/dirty/saving/success states.
- [x] Focused tests plus required backend/frontend broad gates pass, or any
      unrelated baseline failure is named with evidence.
- [ ] Authenticated UI inspection passes at 1280, 390, and 320 pixels with no
      horizontal overflow and 44px touch targets.

## Rollback

Revert the application changes and delete only the seeded setting row if it has
not been intentionally edited. Existing exports then return to the prior
hardcoded behavior; no financial or workbook data migration is required.
