# Plan Report: Track invalid bank info employees on bảng công

- Date: 2026-08-18
- Mode: fast (HOLD scope), auto-detected after inline scouting
- Plan: `plans/260818-1849-track-invalid-bank-info-employees-on-bang-cong/`
- Status: created, validated, active pointer set (`ak plan use`), tasks hydrated (#1–#3)

## Key Finding

Feature is ~90% built already. OnePay verdict persistence (migration 095 +
`BankAccountValidator`), the `GET /employees/missing-bank-details` endpoint
(predicate already includes `bank_account_status = 'invalid'`), and
`MissingBankDetailsSection` mounted on all four bảng công pages
(admin/partner × desktop/mobile) all shipped previously.

## Actual Gap

`buildMissingBankDetailsBaseQuery` gates the whole list on "has pending
timesheet or PENDING advance request" — invalid-bank employees without
pending work vanish from the reminder list.

## Locked Decisions (user, scope challenge)

1. HOLD SCOPE.
2. Invalid → always displayed (active-project rule still applies); missing →
   keep pending-work gate.
3. Wallet "Tra cứu tài khoản" stays read-only (yesterday's plan 260818-1500
   decision preserved — not reversed).

## Phases

1. Backend query relaxation + `bank_warning_kind` response field (9-case test matrix)
2. Frontend row-kind distinction ("Sai thông tin" vs "Thiếu thông tin") with client-side fallback
3. Gates (`make api-test`, `go test -race`, lint/type-check), 6-row manual matrix, docs/api.md, commit

## Notes

- `set-active-plan.cjs` hook script does not exist in this repo; session
  pointer set via `ak plan use` only.
- Knowledge graph stale (436 files / 115 commits behind baseline) — needs
  `/understand --full` re-baseline; not blocking.
- Release note should mention that all historical invalid-status employees
  will appear at once on deploy (intended "always display" behavior).

## Unresolved Questions

None.
