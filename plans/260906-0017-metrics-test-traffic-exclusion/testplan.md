# Test Plan — Exclude integration-test traffic from api_metrics

## Context

System Health dashboard ("Tổng lỗi" per endpoint) showed 72/54/36/27/27/18/18/18 errors
across PUT /settings/*, POST /me/advance-payment/request, POST /timesheets/partner-import,
POST /wallet/bulk-transfer/upload, POST /email/history/*/settle,
PATCH /projects/*/employees/*/advance-request-enabled, GET /wallet/bulk-transfer/batches/*/kq,
POST /projects/*/employees/remove.

Diagnosis (2026-09-06): every one of those rows lives in the LOCAL dev `api_metrics`
(user_id=1, paired sub-ms bursts, 0-3ms durations, evening blocks on 09-04/09-05) — the
signature of `make api-test` negative-case flows. Prod `api_metrics` has ~zero errors on
these routes (verified read-only on tingting.vip). No product defect; the middleware
records synthetic test traffic.

## Change

1. `backend/internal/transport/http/middleware/api_metrics.go` — skip recording when
   request carries `X-Client-Source: integration-test`.
2. `backend/tests/integration/client.go` — `doRequest` sets that header on every call.
3. Local dev DB cleanup: delete the synthetic 4xx rows for the 8 routes (since 09-04)
   so the dashboard reflects reality immediately.

## Matrix

| # | Case | Expected |
|---|------|----------|
| 1 | New unit test: request WITH header | repo.FindOrCreateEndpoint/Create never called |
| 2 | New unit test: request WITHOUT header | Create called exactly once |
| 3 | New unit test: `/api/v1/metrics/*` path (existing skip) | not recorded |
| 4 | `go build ./...` (backend) | clean — goimports/alias hazard check |
| 5 | `go test ./internal/transport/http/middleware/...` | all pass |
| 6 | `go vet` scoped to touched packages | clean |
| 7 | Local DB: error rows for the 8 routes deleted; counts before/after recorded | dashboard "Tổng lỗi" for these routes → 0 |
| 8 | Frontend | no change needed — dashboard consumes same API; no frontend defect found |

## Out of scope (reported, not fixed here)

Prod's actual error surface (last 3 days): POST /auth/login 401×149 (real-user failed
logins), GET /projects/*/payrate 403×46, GET /projects 500×8, GET /dashboard/bank-usage/projects 500×5,
POST /auth/change-password 400×19, POST /auth/zalo-reset/request 429×20.

## Results (2026-09-06)

| # | Case | Result |
|---|------|--------|
| 1 | Unit: header set → not recorded | PASS |
| 2 | Unit: no header → recorded once | PASS |
| 3 | Unit: /api/v1/metrics/* skip | PASS |
| 4 | `go build ./...` | PASS |
| 5 | `go test ./internal/transport/http/middleware/` (full package) | PASS |
| 6 | `go vet` on touched packages | PASS |
| 7 | Local DB: 417 synthetic 4xx rows deleted, 0 remaining | PASS |
| 8 | Frontend: no change needed (no defect found) | n/a |

Follow-up evidence: next `make api-test` run should leave api_metrics untouched
(verify with the same grouped query).

