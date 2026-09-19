# Files

- [Backend Application Services](application-services.md) - How use cases are orchestrated in internal/app — service packages by capability, the transaction-manager unit-of-work pattern, the after-commit callback discipline, and the asynq worker taxonomy.
- [Backend Domain Layer](domain-layer.md) - The inner ring of DDD — entities, value objects, domain events, ports, domain services, the wallet aggregate with its state machine and double-entry ledger, and the property-based tests that guard the accounting invariants.
- [Backend Infrastructure Adapters](infrastructure.md) - How domain ports are implemented — GORM repositories with ToDomain mappers, Redis Streams event bus, cache invalidation after commit, asynq client, OnePay/9Pay disbursement adapters, anti-corruption layer, email and Zalo integrations.
- [Architecture Overview](overview.md) - The dependency-direction diagram (transport → app → domain ← infra), the frontend role separation, the three async surfaces (domain events via Redis Streams, asynq background jobs, payment-provider IPN webhooks), and the desktop+mobile parity rule.
- [Backend HTTP Transport](transport-http.md) - Handlers, middleware chain, request/response contracts, RBAC enforcement, and the conventions that keep the HTTP layer thin and free of GORM types.
