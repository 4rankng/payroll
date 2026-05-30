# Orchestration Layer

## Purpose

The orchestration layer contains workflows that coordinate operations across multiple domains. This prevents domains from directly importing each other, which would create coupling and import cycles.

## Pattern

```
Domain Services (pure, isolated)
         ↓
   Orchestrators (cross-domain workflows)
         ↓
   Ports (interfaces at boundaries)
```

## Key Principles

1. **Domains never import other domains directly** - They only import ports (interfaces)
2. **Orchestrators coordinate workflows** - Complex business processes live here
3. **Ports define boundaries** - Interfaces prevent coupling
4. **Keep orchestrators thin** - They coordinate, not implement business logic

## Current Orchestrators

- **PayrollGeneratorOrchestrator** - Coordinates payroll generation workflow
- **SettlementProcessorOrchestrator** - Coordinates settlement processing
- **LoanSynchronizerOrchestrator** - Coordinates loan payment synchronization

## Usage Example

```go
// BAD: Direct cross-domain dependency
func (s *PayrollService) Generate(ctx context.Context) error {
    // Direct imports of settlement, ledger, notification services
    transaction := s.settlementService.CreateTransaction(...)
    s.ledgerService.CreateEntry(...)
    s.notificationService.Send(...)
}

// GOOD: Use orchestrator
func (o *PayrollGeneratorOrchestrator) GeneratePayroll(ctx context.Context, req *Request) error {
    // Coordinates via ports (interfaces)
    transaction := o.transactionPort.CreateTransaction(...)
    o.ledgerPort.CreateEntry(...)
    o.notificationPort.Send(...)
}
```

## Benefits

- ✅ No import cycles
- ✅ Domains stay pure and focused
- ✅ Easy to test (mock ports)
- ✅ Clear separation of concerns
- ✅ Workflows can be reused by APIs, workers, crons
