---
title: "Phase 6: upload guard"
status: todo
priority: P1
effort: "0.5d"
dependencies: [phase-02]
---

# Phase 6: upload guard

## Overview
USER-APPROVED BEHAVIOR CHANGE. Every multipart Excel endpoint gets wallet_bulk-grade
guards. Oversized/misnamed/crafted files that were accepted now get 4xx.

## Requirements
- [ ] NEW `internal/transport/http/uploadguard.go` + tests, modeled on `wallet_bulk_transfer_handler.go:51-289` (gold standard): `MaxBytesReader` BEFORE `FormFile`, extension check, ZIP/OLE magic sniff, env-configurable cap (default ~20 MiB), typed Vietnamese errors matching handler style
- [ ] Adopt at: flexpay `advance_payment/file_handler.go` (replaces unbounded `io.ReadAll :205`), settlement ×2 (`timesheet/settlement_upload.go`, `settlement/upload_settlement.go`), OnePay fee (`handlers/ledger.go`), employee import, BCC import
- [ ] All service-side `excelize.OpenReader` → `excelkit.OpenReader` (unzip caps)

## Implementation Steps
1. Guard helper + unit tests (oversized, wrong magic, valid, .xls OLE variant)
2. Adopt per endpoint, one commit each
3. Handler tests + `make api-test`; deploy watched

## Success Criteria
- [ ] No multipart Excel endpoint accepts unbounded bodies; all open through capped reader
- [ ] Valid files (incl. `.xls` OLE for bulk-transfer result) still pass

## Risk Assessment
Medium, INTENTIONAL behavior change (user-approved 2026-09-06). Rollback: revert
per-endpoint commits (guard is additive).
