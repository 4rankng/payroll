# Files

- [Cash Readiness Forecast](cash-readiness-forecast.md) - Statistical forecast pipeline that tells the admin how much cash to prepare for the next weekly bulk transfer, behind a swappable ForecastProvider.
- [FlexPay Advance Payments](flexpay-advance-payments.md) - Advance-payment request lifecycle, tiered fee schedule with effective-date history, and reconciliation against the ledger.
- [Salary Disbursement and Payment Providers](salary-disbursement-and-payment-providers.md) - Bulk transfer pipeline, provider-agnostic disbursement abstraction (OnePay prod, 9Pay sandbox), IPN handling, and asynq-driven background workers.
- [Timesheet Engine](timesheet-engine.md) - Timesheet creation, validation, bulk import (BCC pipeline), and approval lifecycle that feeds salary payout.
