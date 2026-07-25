---
title: "Unified bank warning invariant"
date: "2026-07-25 14:40"
severity: "High"
component: "partner bank-warning flow"
status: "Resolved"
---

## Context

Partner timesheet users need one warning surface for bank data that is missing, partial, or confirmed invalid. The UI must stay Vietnamese-only, concise, and free of provider/internal error text.

## What Changed

- Manual employee create/update still rejects confirmed-invalid bank edits before persistence.
- BCC import now uses a trusted import-only employee creation path so confirmed-invalid bank info can be stored with its status and reason.
- The missing-bank query now includes missing bank ID, blank account number, blank account name, and `invalid`, while excluding `unverified`.
- BCC terminal results now refresh the warning list whether they arrive through polling or are returned directly from an idempotent upload replay.
- Import error copy was sanitized into short Vietnamese row messages and partial-completion presentation.

## Key Invariant

Manual writes and import writes now diverge intentionally:

- Manual invalid bank edits are rejected and never persisted.
- Import-created invalid bank info is persisted so it can be corrected from the unified warning list.
- Existing-employee bank corrections validate outside the row lock, then use a short independent transaction to lock, recheck, and write the bank tuple. A concurrent manual edit wins instead of being overwritten.
- Because that correction transaction is intentionally independent, its corrected/invalid flag can survive a later workbook rollback and still appear in the warning list.

## Verification

- `go test ./internal/app/services/employee ./internal/infra/persistence/query_builders`
- Focused frontend Vitest suite: 23 tests passed.
- Frontend TypeScript and scoped ESLint passed.
- `make -C backend api-test`

## Notes

- `make -C backend api-test` ran 278 integration cases: 258 passed, 5 unrelated baseline cases failed, and 15 were skipped.
- The unrelated failures were in password-reset rate limiting, asset setup/download, and transaction export parameter expectations.
