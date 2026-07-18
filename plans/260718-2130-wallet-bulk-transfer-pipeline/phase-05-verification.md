---
phase: 5
title: "Verification"
status: pending
priority: P2
dependencies: [1, 2, 3, 4]
---

# Phase 5: Verification

## Overview

End-to-end integration tests proving the full pipeline works, plus the regression gate (`make api-test`) and frontend lint gates. This is where we confirm the locked decisions hold: no double-charges on duplicates, fees booked correctly on both success and failure, KQ format matches the reference file.

## Requirements

- **Functional**: All acceptance criteria from `plan.md` verified by automated tests or manual runbook.
- **Non-functional**: Tests run in ≤60s (no real OnePay calls — use mock provider). Frontend gates ≤30s.

## Architecture

Three test layers:

1. **Backend unit tests** (already required per phase) — fast, mocked.
2. **Backend integration tests** — new file under `backend/tests/integration/` exercising the HTTP routes against a test DB with a mock OnePay provider.
3. **Manual smoke test runbook** — for final sign-off against the sandbox OnePay endpoint.

### Mock OnePay provider

Reuse the existing test infra: the bootstrap already supports injecting a mock `DisbursementProvider` (look at how existing wallet tests substitute the provider). The mock should:
- Return `Success` for rows where `AccountNo` doesn't start with `9999`.
- Return `Failed` with error code `1001` for rows where `AccountNo` starts with `9999`.
- Track call count so tests can assert exactly N calls were made.

## Integration Test Scenarios

### Test 1: Happy path (full pipeline)

1. Seed a `DisbursementFeeScheduleEntry` for `1pay` with `fee_vnd = 3850`, `effective_date = today`.
2. POST `/wallet/bulk-transfer/upload` with `Yeu cau chuyen tien.xlsx` (2 rows).
3. Assert 202 response with `batch_id`, `total_count=2`, `transfer_amount = 1192500 + 2772000 = 3964500`, `estimated_fee_total = 7700`.
4. Poll `GET /wallet/bulk-transfer/batches/:id` until `status = 'completed'` (timeout 30s).
5. Assert `success_count=2`, `failed_count=0`, `total_fee=7700`.
6. Assert exactly 2 `wallet_payments` rows exist with `bulk_transfer_batch_id = id`, status `completed`, `fee=3850`.
7. Assert exactly 1 `transactions` row exists with `type='expense'`, `party='OnePay'`, `amount=7700`, `status='settled'`.
8. Assert 2 `ledger_entries` rows: Debit Expense 7700, Credit Cash 7700.
9. GET `/wallet/bulk-transfer/batches/:id/kq` → assert 200, `Content-Type` is xlsx, parse the bytes and verify 2 data rows show `Thành công` with FT numbers.

### Test 2: Duplicate content-hash rejection

1. Upload `Yeu cau chuyen tien.xlsx` (Test 1 already uploaded it).
2. Assert 409 with body `{"error":"duplicate_batch","existing_batch_id":<id>}`.
3. Assert NO new `wallet_payments` rows created (count unchanged).
4. Assert mock OnePay was NOT called (count unchanged).

### Test 3: Duplicate VFIC rejection

1. Upload a different file with the same VFIC codes as Test 1 but different account numbers (so content hash differs).
2. Assert 409 with `{"error":"duplicate_vfic","conflicts":["VFIC38a9d698","VFIC86e8585f"]}`.
3. Assert NO new rows, NO OnePay call.

### Test 4: Partial failure ledger booking

1. Build an Excel with 3 rows: 2 normal + 1 with `AccountNo='9999...'` (triggers mock failure).
2. Upload → wait for completion.
3. Assert `success_count=2`, `failed_count=1`, `total_fee = 3 × 3850 = 11550` (per locked decision: failure counts toward fee).
4. Assert 1 Expense transaction with `amount=11550`.
5. Assert KQ shows 2 `Thành công` rows + 1 `Thất bại` row with error message in column I.

### Test 5: All-preflight-failure (no fee, no ledger)

1. Build an Excel with valid rows (amount ≥100000, valid bank code) but configure the mock OnePay provider to return `ErrPreflightValidation` for every row (simulating OnePay's pre-transfer account check rejecting the recipient).
2. Upload → wait.
3. Assert `failed_count = N`, `total_fee = 0` (per migration 077: pre-flight rejections have fee zeroed because `error_code` is NULL/non-numeric).
4. Assert NO Expense transaction created (sum is 0 → skip booking per Phase 2 step 6).
5. Assert KQ shows all rows as `Thất bại`.

Note: This test triggers preflight rejection at the **provider layer** (OnePay's `CheckAccount` fails after the row passes our parser's local validation), not at the parser's `Amount < 100000` check. The parser-side amount floor and provider-side preflight are distinct rejection paths.

### Test 6: Idempotency of ledger booking

1. After Test 1 completes, manually trigger the batch completion checker again (or call the internal `bookBatchLedger` directly).
2. Assert NO duplicate `transactions` row created. Either the function is idempotent (checks `ledger_txn_id IS NOT NULL`) or the unique constraint prevents the duplicate.

### Test 7: 3 TPS throttle respected

1. Build a 10-row Excel.
2. Upload → measure time from upload response to `status='completed'`.
3. Assert total processing time ≥ 3s (10 rows / 3 TPS ≈ 3.3s) — confirms throttle is active.

### Test 8: KQ format byte-level comparison

1. Generate KQ for the Test 1 batch.
2. Parse with `excelize` and assert cell-by-cell:
   - A1 = `"DANH SÁCH GIAO DỊCH/ LIST OF BULK TRANSACTION"`.
   - A2 starts with `"Số tham chiếu (Ref No): WB"`.
   - A3 matches `"Ngày giao dịch (Transaction date): DD/MM/YYYY"`.
   - Row 4 headers match the reference exactly (all 9 columns).
   - Row 5 column H = `"Thành công"`, column I = FT number.
   - Column G (fee) = `3850` for both rows.

## Related Code Files

- **Create**: `backend/tests/integration/wallet_bulk_transfer_test.go` — Tests 1-8.
- **Create**: `backend/testdata/yeu_cau_chuyen_tien.xlsx` — copy of the attached file.
- **Create**: `backend/testdata/kq_chuyen_tien_reference.xlsx` — copy of the attached KQ file.
- **Create**: `backend/testdata/yeu_cau_chuyen_tien_duplicate_vfic.xlsx` — for Test 3.
- **Create**: `backend/testdata/yeu_cau_chuyen_tien_failure.xlsx` — for Test 4 (row with 9999 account).
- **Create**: `backend/testdata/yeu_cau_chuyen_tien_preflight.xlsx` — for Test 5 (valid rows, amount ≥100000; mock provider returns `ErrPreflightValidation`).
- **Create**: `docs/runbooks/wallet-bulk-transfer-smoke-test.md` — manual smoke test against sandbox.

## Implementation Steps

1. **Copy test fixtures** from `~/Downloads/` into `backend/testdata/`.

2. **Set up mock OnePay provider** in the integration test bootstrap (or reuse existing test helpers — search for `mockProvider` or `MockDisbursementProvider`).

3. **Write Test 1 (happy path)** first. This will exercise the most code and reveal integration issues early.

4. **Write Tests 2-8** in order. Each test should be independent (uses a fresh batch or unique fixture).

5. **Run `make api-test`** — must be green with all new tests passing.

6. **Write the smoke-test runbook** (`docs/runbooks/wallet-bulk-transfer-smoke-test.md`):
   - Precondition: sandbox OnePay credentials configured.
   - Steps: upload real `Yeu cau chuyen tien.xlsx` → observe transfers in OnePay sandbox dashboard → download KQ → verify ledger entry in DB.
   - Expected: same as Test 1 but with real FT numbers.

7. **Frontend verification**: run `cd frontend && pnpm lint && pnpm type-check`. Optionally do a manual click-through of the upload dialog.

8. **Update `HANDOFF.md`** (gitignored) with the new feature flag/endpoint list for the next session.

## Success Criteria

- [ ] All 8 integration tests pass under `make api-test`.
- [ ] Tests confirm: duplicate rejection at both levels (content hash + VFIC).
- [ ] Tests confirm: ledger books aggregate fee including failed rows (Test 4).
- [ ] Tests confirm: no ledger booking when all rows pre-flight fail (Test 5).
- [ ] Tests confirm: ledger booking is idempotent (Test 6).
- [ ] Tests confirm: 3 TPS throttle is respected (Test 7).
- [ ] KQ format matches reference byte-for-byte at the cell level (Test 8).
- [ ] `make api-test` overall green (no regressions in existing wallet/disbursement/9Pay tests).
- [ ] `cd frontend && pnpm lint && pnpm type-check` pass.
- [ ] Smoke-test runbook written and reviewed.

## Risk Assessment

- **Risk**: Integration tests are slow because of asynq worker startup. **Mitigation**: Use asynq's in-process test mode (`asynqtest`) or call workers synchronously in tests via a test-only direct-invocation mode. If neither works, accept 60s test time.
- **Risk**: Mock provider doesn't perfectly emulate OnePay's behavior (e.g. preflight validation). **Mitigation**: Mock has explicit modes for "success", "failure", and "preflight_rejection"; tests select the mode per scenario.
- **Risk**: Test fixtures contain real VFIC codes that collide with existing dev DB rows. **Mitigation**: Test setup truncates `wallet_payments` and `bulk_transfer_batches` before each test, or uses unique prefixes.
- **Risk**: KQ byte-comparison is brittle. **Mitigation**: Test 8 compares parsed cell values, not raw bytes — timestamps in row 3 are validated by regex, not equality.
