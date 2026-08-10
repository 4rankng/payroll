# BCC WeeklyPayment import contract

Date: 2026-08-10

- Excel sheet name is the payrate position, for example `Lương 520`.
- Row-10 headers are shift types, for example `HC` and `TCN`.
- Every imported row uses `ngày thường`.
- The importer no longer derives a `Lương 520HC` key.

Focused backend service/parser tests, the frontend error test, lint/type-check, and the production frontend build passed. Live import verification remains pending because the local API on `:8080` was unavailable.
