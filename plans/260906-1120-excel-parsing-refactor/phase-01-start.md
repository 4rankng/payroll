---
title: "Phase 0: Characterization safety net"
status: in-progress
priority: P1
effort: "0.5d"
dependencies: []
---

# Phase 0: Characterization safety net

## Overview
Lock current behavior of all 5 BCC parsers + routing + unknown-template failure in
tests and real-file goldens BEFORE any production code moves. Tests + fixtures only.

## Requirements
- [x] Test matrix written to `testplan/260906-excel-refactor-characterization.md` BEFORE any verification run (repo rule)
- [x] Real fixtures copied from `testplan/excelfixture/` → `backend/tests/fixtures/bcc/{legacy,weekly_bcc,weekly_payment,multi_position,date_row}/` (anonymize LGD "Mr Đức" filename)
- [x] `excel/characterization_test.go`: golden parse JSON per format (`GOLDEN_UPDATE=1` regen)
- [x] `excel/routing_characterization_test.go`: workbook → DetectFormat → ParseBCCData winner; covers `"STK "` trailing space, hidden sheets skipped, unknown-template error, empty workbook, BCC-named date-row (T09)
- [x] `employee/import_characterization_test.go` (island has zero coverage today)
- [x] Env-gated real-file harness `BCC_REAL_FILE_DIR` (default `../../../testplan/excelfixture`, skip-if-absent)

## Implementation Steps
1. Write test matrix doc
2. Copy + anonymize fixtures
3. Golden test helper (`assertGolden`) + per-format golden tests on real files
4. Routing table test (synthetic edge cases + real files)
5. Employee import characterization (synthetic workbook → parsed rows incl. silently-skipped bad rows)
6. Run suites; regen goldens; commit

## Success Criteria
- [x] `go test ./internal/app/services/excel/... ./internal/app/services/employee/...` green
- [x] Baseline `make api-test` green (flow_bcc_import, flow_bcc_weekly_import, flow_bcc_weekly_payment_import, flow_flexpay_import)
- [x] GATE: this phase green before Phase 2 (registry) starts

## Risk Assessment
None — no production code. Rollback: delete test files.

## Fixture map (real files, from testplan/excelfixture/)
| File | Format | Notes |
|------|--------|-------|
| BCC BUMHAN Thang8.xlsx | date-row under "BCC" sheet | T09 strategy-fallback path |
| BCC LGD lương tuần (anonymized name) | weekly BCC (BCC-HC/OT30…OT390 + STK) | |
| BCC Thái Bình Dương Kỳ 4 T08 | weekly payment (Lương 500–900 + STK) | |
| BCC PQC Hải Phòng Kỳ 4 T08 | multi-position (Có/Không tay nghề, Stk, Hỗ trợ khác) | |
| BCC EVA T08 (BCC + STK) | legacy | second legacy sample beside committed eva06.xlsx |
