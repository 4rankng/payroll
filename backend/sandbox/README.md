# payroll-sandbox

A single Go binary that mocks the two disbursement providers the payroll app
talks to, so local development and tests work **without real provider
credentials or network access**.

- **9Pay disbursement mock** — routes `/disbursement/*`, `/api/transaction/*`
- **OnePay PayOut mock** — routes `/onepayout/*` (account info, funds transfer,
  inquiry, balance)

It is a **separate Go module** (`module payroll-sandbox`) — it is *not*
compiled into the main backend. It runs as its own HTTP server on `:9001`.

## Run it locally

Easiest — start only the sandbox container (builds `./sandbox`, port `9001`):

```bash
make sandbox            # from repo root, or: cd backend && make sandbox
make sandbox-logs       # follow logs
make sandbox-rebuild    # rebuild without cache
```

`make db` also starts it alongside MySQL/Redis/Adminer. The dev `docker-compose.dev.yml`
wires it so the mock's IPN callbacks reach the host backend on `:8080`
(`host.docker.internal:8080`), and the backend's dev defaults already point at
it (`ONEPAY_BASE_URL=http://localhost:9001`, `ENABLE_ONEPAY=true` in non-prod).

Run it standalone (no Docker):

```bash
cd backend/sandbox
ONEPAY_PARTNER_ID=TESTVFICPO \
ONEPAY_PARTNER_KEY=C2B5DA903DAE19E215454211C66A59DD \
ONEPAY_ACCOUNT_ID=666894931888 \
MOCK_ONEPAY_IPN_URL=http://localhost:8080/api/v1/webhooks/disbursement/1pay \
go run .
```

Health check: `curl -s localhost:9001/healthz` → `ok`.

## How advance payouts flow through the mock

No provider is involved when an advance *request* is created — that is internal.
The mock handles the **disbursement** leg, end to end:

1. Admin approves a request → the request becomes `APPROVED`.
2. `DisbursementExecuteWorker` claims it, calls `registry.Active()` → OnePay
   provider → `CreateFundsTransfer` (`PUT /onepayout/api/v1/accounts/{accountId}/funds_transfers/{fundsTransferId}`).
3. The mock stores the transfer, returns `state: approved`, and (after
   `MOCK_ONEPAY_IPN_DELAY`, default 3s) fires a **signed IPN** back to
   `PUT /api/v1/webhooks/disbursement/1pay`.
4. The backend verifies the IPN signature and transitions the payment
   `authorised → completed`; the advance request moves to `COMPLETED`.

To watch it: create + approve a request locally, then `make sandbox-logs` and
look for the `requestFundsTransfer` / IPN lines. The OnePay recoverer
(`MOCK_ONEPAY_DB_DSN`) also self-heals any transfer stuck in `authorised` for
>30s.

## Injecting scenarios

The mock reads the transfer's `remark` / `funds_transfer_info` for keywords and
behaves accordingly (default = `completed`):

| Keyword in remark      | Behavior                                                        |
| ---------------------- | --------------------------------------------------------------- |
| `mock_completed`       | Sync approve + IPN `approved` (default)                         |
| `mock_failed_sync`     | Rejects the PUT immediately (`INVALID_ACCOUNT_INFO`)            |
| `mock_failed_async`    | Approves sync, then IPN `failed` (code 91)                      |
| `mock_reversed`        | IPN `approved` then a later IPN `reverted`                      |
| `mock_slow`            | Approves sync, IPN delayed by `MOCK_ONEPAY_SLOW_DELAY` (5m)     |
| `mock_no_ipn`          | Approves sync, **no IPN** sent (use to exercise the recoverer)  |
| `mock_500`             | Returns HTTP 500                                                |
| `mock_invalid_rsa`     | Returns `INVALID_RSA_USER_ID` (AccountID/FundsTransferID leak)  |

## Balance

The mock keeps a SQLite balance (`onepay_mock_balance.db`, seeded at
**100,000,000 ₫**). Each `completed`/`approved` transfer debits it. Top it up:

```bash
make onepay-topup                       # +100,000,000 ₫ on localhost:9001
AMOUNT=50000000 make onepay-topup       # custom amount
# or directly:
curl -X POST 'localhost:9001/admin/topup/1pay?amount=500000000'
```

Query it: `GET /onepayout/api/v1/accounts/{accountId}` (OWS1-signed, or set
`MOCK_ONEPAY_SKIP_VERIFY=true` to bypass signature checks during development).

## Configuration

Reuses the real `ONEPAY_PARTNER_ID` / `ONEPAY_PARTNER_KEY` /
`ONEPAY_ACCOUNT_ID` so signatures match the backend. OnePay-specific knobs:

| Env var                     | Default                                                         | Purpose                              |
| --------------------------- | --------------------------------------------------------------- | ------------------------------------ |
| `MOCK_LISTEN_ADDR`          | `:9001`                                                         | Listen address (shared with 9Pay)    |
| `MOCK_ONEPAY_IPN_URL`       | `http://host.docker.internal:8080/api/v1/webhooks/disbursement/1pay` | Where IPN callbacks are sent   |
| `MOCK_ONEPAY_IPN_DELAY`     | `3s`                                                            | Delay before sending IPN             |
| `MOCK_ONEPAY_IPN_RETRY_DELAY` | `5s`                                                          | Retry backoff between IPN attempts   |
| `MOCK_ONEPAY_SLOW_DELAY`    | `5m`                                                            | Delay for `mock_slow`                |
| `MOCK_ONEPAY_SKIP_VERIFY`   | `false`                                                         | Skip inbound OWS1 signature verify   |
| `MOCK_ONEPAY_DB_DSN`        | _(dev compose: points at MySQL)_                               | Enables the stale-payment recoverer  |
| `MOCK_ONEPAY_HOLDER_NAME`   | _(empty)_                                                       | Overrides the database-resolved employee name for mismatch testing |
| `MOCK_BALANCE_DB`           | `onepay_mock_balance.db`                                        | SQLite balance file                  |

## Signing gotcha

OnePay's IPN canonical URI is a **hardcoded internal gateway URL**
(`http://localhost/payout-merchants/https/tingting.vip/443/api/v1/webhooks/disbursement/1pay`),
mirrored in both the production signer (`internal/infra/disbursement/onepay/signing.go`)
and this mock (`onepay/ipn.go`). Both sides use the same constant, so
signature verification passes — but if you ever retarget the webhook host, you
must update that canonical URI in **both** places or IPNs will be rejected.
