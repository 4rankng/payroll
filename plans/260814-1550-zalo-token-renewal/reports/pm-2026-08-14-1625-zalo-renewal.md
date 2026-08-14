---
title: Zalo renewal completion report
status: completed
created: 2026-08-14
---

# Zalo Renewal Completion Report

## Summary

Implementation complete locally. Payroll now validates token-chain health at
save time, coordinates exchanges across API instances, and prevents stale JSON
writes from restoring consumed tokens.

## Acceptance Evidence

| Requirement | Result | Evidence |
|---|---|---|
| Invalid refresh fails immediately | PASS | Save-time `-14014` regression test |
| Valid pair rotates and persists successor | PASS | Provider/service persistence tests |
| Cross-instance single writer | PASS | Two-Provider and Redis lease race tests |
| Stale token cannot be restored | PASS | Service CAS and repository binary-CAS tests |
| Temporary OAuth failure stays distinct | PASS | 502/malformed-response regression test |
| Secrets absent from SQL logs | PASS | Parameterized logger regression test |
| Existing backend behavior | PASS | `go test ./...` and `go vet ./...` |

## Verification Classification

- Repeated touched-package `-race`: PASS.
- Full repository `-race`: FAIL on unrelated existing cache invalidation test
  race at `internal/infra/events/cache_invalidation_handler_test.go:99`.
- Live API integration: BLOCKED because local API port 8080 was unavailable.
- Graphify AST knowledge graph: updated.

## Operational Next Step

Deploy only with explicit authorization. Then obtain one fresh pair for
Payroll's own Zalo App ID, save it through Payroll, and send one controlled OTP.

