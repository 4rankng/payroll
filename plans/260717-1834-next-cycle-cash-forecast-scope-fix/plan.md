---
title: "Target-Kỳ cash forecast scope fix"
status: completed
created: "2026-07-17T18:34:00+08:00"
---

# Target-Kỳ cash forecast scope fix

## Goal

Make “Dự báo tiền trả” represent only the next Kỳ shown on the card. The
“Chờ thanh toán” backlog must not be read or added.

## Phases

1. [Correct forecast composition](./phase-01-correct-forecast-composition.md)
2. [Align API and UI](./phase-02-align-api-and-ui.md)
3. [Regression verification and documentation](./phase-03-regression-verification-and-documentation.md)

## Acceptance criteria

- [x] Forecast point and interval contain only the target Kỳ.
- [x] `PendingPaymentAmount` cannot affect the forecast.
- [x] Current target-Kỳ approved value is retained as the observed floor.
- [x] Existing card format and responsive range layout remain intact.
- [x] Focused backend and frontend quality gates pass.

## Verification notes

- Focused Go tests pass with race detection and coverage.
- Changed backend packages pass `go build`, `go vet`, and `golangci-lint`.
- Frontend lint, TypeScript validation, and production build pass.
- The repository-wide suite still has unrelated existing environment/fixture
  failures; the live API suite could not authenticate against the local server.
