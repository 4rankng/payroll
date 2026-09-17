Backend scope and verification notes

Local-only environment: synthetic payroll_db, separate payroll_unit_qa for destructive repository tests, local Redis, local OnePay/9Pay mock. No deployment, production reads, restores, real payment provider, external email, or commits.

Confirmed fixes:
- Fresh dev MySQL bootstrapping applies forward schemas only, guards empty local DB, skips historical production financial repair, and handles specific historical schema/index omissions. Forward migration 107 adds accountant role without removing existing role values, settlement linkage, missing asset timestamp, and bank branch-code default. Populated legacy sentinel replay preserves financial counts/values and custom role data; second application is idempotent.
- Native payrate seeder now populates the domain JSON field correctly.
- asynq duplicate/task-ID sentinel checks use errors.Is to tolerate wrapped library errors and preserve idempotent enqueue contracts.
- Async payroll import reads newly inserted employees/assignments from the existing transaction, and EmployeeCreated publishes only after its outer transaction commits.
- OnePay export uses the plan's captured payout percentage for XLSX, header, and stored transfer code. New OnePay codes include an optional transfer_amount_snapshot; completion apportions that exact total using integer arithmetic. Already-paid shares must match or the update fails before writes for reconciliation. Existing unversioned bank/OnePay codes retain legacy behavior; no historical repair is performed.
- Advance-payment statement exports now record the authenticated exporter as notification sender, fixing the foreign-key failure and missing history entry.
- Local OnePay sandbox accepts string-form provider amounts and returns the correct balance JSON contract, so tests execute real nonzero mock debits.
- API harness uses complete fixtures and pagination, unique nonexistent IDs, valid approval payloads, strict async payment/import outcomes, actual generated statement reupload, and required full wallet pipeline with paid-amount parity when fixture is configured. It also uses current disbursement routes, requires successful bank persistence with a valid SWIFT reference, and explicitly tests employee access denial instead of counting unavailable admin payroll history as a successful fetch.
- AdvPartner can read the bank reference catalog needed by its scoped editor; bank detail/mutations and global advance-payment files remain denied. Authorization unit and live role matrices verify the limited scope.

Known verification limits:
- 11 backend unit skips: 7 unavailable developer-local payroll spreadsheets, 2 optional bank-report fixture files, 2 pending external OnePay signing fixtures. Synthetic parser/import and local provider flows are exercised independently.
- 2 API conditional skips: current-month removal invisibility assertions require an imported advance period different from current month. Current fixture period matches current month, so these mutually inapplicable assertions are explicitly skipped.
- Real bank certificates/signatures, live provider behavior, push/email delivery, production schema state and historical financial records are not verified by local mocks.
- Existing unversioned OnePay in-flight records are not automatically repaired; their Amount field cannot safely be reinterpreted globally because regular bank exports stored gross values in the same field.

Primary evidence:
/tmp/payroll-ui-qa/db-bootstrap-populated-test.log
/tmp/payroll-ui-qa/backend-unit-race-final.log
/tmp/payroll-ui-qa/backend-vet-final.log
/tmp/payroll-ui-qa/api-test-strict-final.log
/tmp/payroll-ui-qa/payment-provenance-tests.log
/tmp/payroll-ui-qa/onepay-percentage-red.log
/tmp/payroll-ui-qa/onepay-percentage-green.log
/tmp/payroll-ui-qa/import-tx-red.log
/tmp/payroll-ui-qa/import-tx-green.log
/tmp/payroll-ui-qa/import-after-commit.log
/tmp/payroll-ui-qa/sandbox-tests.log

Final policy-only delta evidence (stable API PID 26579, /tmp/payroll-ui-qa/bin/api-server-policy-final):
/tmp/payroll-ui-qa/backend-auth-policy-final.log
/tmp/payroll-ui-qa/adv-partner-policy-live-final.log
/tmp/payroll-ui-qa/integration-harness-final.log
/tmp/payroll-ui-qa/api-test-delivery-final.log
/tmp/payroll-ui-qa/payment-parity-policy-final.log

Final delivery API gate: make api-test exit 0; 340 total, 338 passed, 0 failed, 2 calendar-prerequisite skips; duration 1m19s. All latest harness strictness changes included. Provider/bank/employee permission checks are genuine exercised assertions. OnePay batch 7 transfer and timesheet paid total 168000, receivable 171360, ledger transaction 88; advance request 18 completed. Final financial readback: /tmp/payroll-ui-qa/payment-parity-delivery-final.log.
