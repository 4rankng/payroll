# Files

- [Double-Entry Ledger and Chart of Accounts](double-entry-ledger.md) - Wallet aggregate, double-entry ledger, chart of accounts, and settlement semantics that enforce financial integrity.
- [Architecture Overview](overview.md) - DDD layered architecture for the Go backend, request lifecycle, and the boundary between domain, application, transport, and infrastructure.
- [Transaction Manager and Outbox](transaction-manager-and-outbox.md) - Atomicity across GORM writes, the after-commit hook for cache invalidation and event publication, and the event bus replacing the historical outbox table.
