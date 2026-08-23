# Payrate timeline guard — task context

## Intent

- Goal: prevent duplicate or invalid effective-dated payrate timelines.
- Success: calendar-date comparisons are timezone-safe; active timeline writes are serialized; MySQL rejects invalid ranges and duplicate live start dates; predecessor dates are synchronized after a valid move.
- In scope: payrate temporal service, repository locking, request date parsing, migration, regression tests, and one authorized production repair of project 74's predecessor end date.
- Out of scope: frontend redesign and changing the paid-work-date rule.

## Evidence and decisions

- Root cause: two 22 Aug records were created as UTC/local midnight values. Instant equality missed the duplicate and closed the predecessor at 21 Aug.
- Production state after user removal: project 74 has #69 (1–21 Aug) and #72 (from 15 Aug); the rate resolver is correct from 15 Aug, but the recorded range overlaps.
- Date-only contract: use `clock.DefaultLocation` (Asia/Ho_Chi_Minh), not UTC or process-local time.
- Timeline writes must lock the project row and derive the predecessor `to_date` from the next configuration's `from_date`.

## Verification

- RED: reproduced UTC/local same-calendar-day duplicate and predecessor range synchronization failures.
- GREEN: focused tests and `go test ./... -race` pass; `go vet ./...` and formatting check pass; MySQL 8 migration guards were exercised in an isolated schema. The broad local API suite passed its payrate flow but has four unrelated baseline failures in FlexPay cutoff and loan-interest flows.
- Production repair: after a verified backup, project 74 now has #69 (1–14 Aug) followed by #72 (from 15 Aug), with zero overlaps.
