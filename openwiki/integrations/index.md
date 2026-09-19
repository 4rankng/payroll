# Files

- [Auth & RBAC](auth-rbac.md) - JWT issuance and refresh, the Casbin deny-override policy model, role hierarchy (Admin / Partner / Adv-Partner / Employee), project scoping at handler level, and password reset / OTP plumbing.
- [Background Jobs (asynq)](background-jobs.md) - asynq task queue runtime, scheduler, retries, worker taxonomy (disbursement, bulk transfer, attendance, BCC import, IPN, payroll email, audit, recovery sweepers), and the queue topology.
- [Event Bus (Redis Streams)](event-bus.md) - Domain events declared in events.go, per-aggregate factories in event_factory_*.go, the publish-after-commit outbox discipline, Redis Streams as the transport, and the handler set that consumes them.
- [Payment Providers (OnePay / 9Pay)](payment-providers.md) - Provider-agnostic abstraction, OnePay in production, 9Pay in sandbox, IPN contract and signature verification, status inquiry, balance reporting, and the registry that selects the active provider at boot.
- [Zalo OTP & Password Reset](zalo-otp.md) - Zalo OA-based OTP login, employee-mobile password reset via ZNS, the stateless Zalo client, the credentials-via-settings contract, and the admin-managed connect flow.
