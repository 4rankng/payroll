---
title: MBank new bulk-transfer result template
date: 2026-09-10
summary: "Bank switched result export to banner+header statement; added shared header locator + strategy in both result-import islands, verified E2E on local dev"
---

# MBank new bulk-transfer result template

## What happened
MBank changed the bulk-transfer **result** export from the flat
"Kết quả chuyển tiền theo bảng kê" layout to "Kết quả giao dịch - <ref>.xlsx"
(banner rows 1-11, header row 12: C=account, F=name, G=bank, H=amount,
J=VFIC code, M=fee, P=method, Q=status, R=accounting status, X=FT ref,
data row 13+). The old positional parsers broke: `FindDataByTransactionCodes`
hard-fails on unknown codes, and the flat strategy fed it banner cells
(account numbers, names) as "codes".

## Changes
- `excelkit.LocateMBankStatementHeader` (shared, content-located header,
  NFC + collapsed Vietnamese keyword matching; requires account+amount+
  payment-detail+transaction-status headers in one row).
- `payroll/bulktransfer.MBankStatementStrategy` + factory registration
  BEFORE the default-true TransactionCodeStrategy (weekly flow —
  `/payrolls/bulk-transfer-result`).
- `advance_payment.extractBankResultRows`: MBank branch + legacy flat
  branch preserved verbatim (`/advance-payments/upload-result`).
- Tests: in-memory layouts for both islands, env-gated real-file harness
  `MBANK_RESULT_REAL_FILE` (skips `~$` lock files).

## Decision
Review round (code-reviewer) caught the advance island failing OPEN on
unrecognized statement statuses — a future "Đang xử lý" would mark requests
Completed and create irreversible transaction+ledger entries. Fixed
fail-closed (unrecognized/blank status → THẤT BẠI), matching the
bulktransfer strategy's existing default. Also hardened the "bút toán"
header claim against a renamed accounting-status column.

## Verification
Real files: 166/166 and 22/22 completed. Live dev E2E: API + UI uploads,
btf #144/#145 (weekly), transactions #250/#251, +4 ledger entries,
duplicate re-upload idempotent, UI "Tải PDF" produced an application/pdf
blob (29,529 bytes). Full `go test ./...`: 81 pkgs ok / 0 FAIL
(baseline-identical). Old-format synthetic file still routes to
TransactionCodeStrategy and fails at the DB lookup naming the code —
legacy extraction intact.

## Next steps
- Residual risk: old-format regression is synthetic-only; validate Detect
  against one real archived old file when one is at hand (none exists
  locally; prod files not pulled).
- Not deployed; commit pending user approval.

> Historical work record — not durable authority. Prefer docs/specs/ADRs for current decisions.
