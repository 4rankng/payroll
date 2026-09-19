# Files

- [Attendance & Geofence](attendance-geofence.md) - Self check-in/out with geofence validation, the immutable quota-credit hold, deferred enable switches, and the auto-reject sweepers that protect earning integrity.
- [BCC Weekly Import](bcc-import.md) - Bank Confirmation Certificate upload — the multi-format pipeline that parses weekly Excel files, deduplicates, applies payrate label mapping, and dispatches rows into the timesheet system.
- [FlexPay (Advance Payments)](flexpay.md) - Employee advance-request lifecycle, the tiered fee schedule, per-employee kill switch, the wallet-backed funds flow, and the salary-notification hook that tells employees what their next salary will deduct.
- [Web Push Notifications](push-notifications.md) - Web Push subscription storage (VAPID), the notification fan-out path, the PWA service worker that receives the push, and the relationship with in-app and email channels.
- [Salary Disbursement](salary-disbursement.md) - The weekly bulk-transfer pipeline — payroll export, wallet booking, per-row execute via the active provider, IPN-driven state transitions, status inquiry, settlement, and the verified-transfer recovery sweepers.
- [Timesheets](timesheet.md) - Timesheet entry, validation (daily hours, daytype, status, zero rate, assignment), edit requests, bulk import, and approval workflow — the input to salary disbursement and the FlexPay quota pool.
- [Wallet & Double-Entry Ledger](wallet-ledger.md) - The wallet aggregate (balance, top-ups, payments, IPN), the qmuntal/stateless state machine, the double-entry ledger with property-based accounting invariants, settlement, and the statistical demand forecast that drives top-up timing.
