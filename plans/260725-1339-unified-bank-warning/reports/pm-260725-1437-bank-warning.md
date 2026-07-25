# PM Report: Unified Bank Warning

## Status

| Item | State |
|---|---|
| Phase 1 backend warning + import semantics | Done |
| Phase 2 frontend parity + import UX | Done |
| Phase 3 verification + rollout safety | Done |

## Verified

- Focused backend race tests passed:
  - `go test -race ./internal/app/services/employee ./internal/app/services ./internal/infra/persistence/query_builders -count=1`
- Focused frontend tests passed: 23/23, covering the unified warning,
  sanitized import errors, partial completion, polling refresh, and direct
  terminal upload refresh.
- Frontend typecheck/build passed:
  - `pnpm exec tsc -p tsconfig.json --noEmit`
  - `pnpm build`
- `git diff --check` passed.
- `graphify update .` completed.

## Shared Gates

- `make api-test` from repo root is not a valid target.
- `make api-test` from `backend/` ran and failed in unrelated baseline flows:
  - `PasswordReset`: garbage token / missing token returned `429` instead of expected `401` / `400`
  - `Assets`: upload / metadata / download failures
  - `Transactions`: export flow missing `fromDate`
- These failures are outside the bank-warning change.

## Scope Notes

- Backend import path now allows confirmed-invalid bank data from BCC imports and still rejects manual invalid writes.
- Unified warning list now includes missing bank, missing account number, missing account holder, and confirmed-invalid rows.
- Frontend now shows one Vietnamese warning category and refreshes it after terminal BCC runs.
- The direct upload idempotency path now also invalidates the warning list.

## Risks

- The broad integration suite is not fully green until the unrelated baseline
  failures are fixed or explicitly waived for release.
- Authenticated browser QA was not used as release evidence; the affected UI
  states are covered by component and hook tests.

## Next Actions

- Decide whether the unrelated integration failures should block release or be
  waived.
- If release is approved, commit and deploy only the scoped bank-warning files;
  preserve the unrelated dirty-worktree changes.

## Unresolved Questions

- None on the bank-warning behavior itself.
