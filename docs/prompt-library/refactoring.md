# Prompt: Refactoring

Use when improving code structure, performance, or maintainability without changing functionality.

## Prompt

```
Goal: Refactor [COMPONENT/MODULE] — [WHAT_TO_IMPROVE and WHY]

Context to gather:
1. Read the AGENTS.md in the target directory
2. Read docs/decisions/ADR-001-ddd-clean-architecture.md for layer boundary rules
3. Identify all callers of the code being refactored:
   grep -rn "functionName" backend/internal/ --include="*.go"
4. Check existing tests for the component — these are your safety net
5. Read docs/standards/review-checklist.md

Constraints:
- NO functionality changes — behavior must be identical before and after
- Public contracts (function signatures, exported types, API responses) must not change
- Follow existing patterns in the codebase
- Keep the diff minimal — don't reformat unrelated code
- All existing tests must pass after refactoring

Output format:
1. Document the "before" state: what the code looks like now and why it needs refactoring
2. Document the "after" state: what it will look like and what improves
3. List all files that will change
4. List all callers that may be affected
5. Implement the refactoring
6. Run all tests and lint

Verification:
- All existing tests pass: go test ./... -race -cover
- Integration tests pass: make api-test
- No lint errors: gofmt -l . && golangci-lint run ./...
- No new public API surface (unless intentional)
- Frontend unaffected (if backend-only): cd frontend && pnpm type-check
- If performance-related: benchmark before and after
```

## When to Refactor

- Code is hard to understand or modify.
- Duplicated logic that should be shared.
- N+1 queries or performance bottlenecks.
- Layer boundary violations (e.g., handler touching GORM directly).
- Tests are hard to write because of tight coupling.

## When NOT to Refactor

- Adding a new feature (use the feature implementation prompt instead).
- Fixing a bug (use the bug fix prompt instead).
- "While I'm here" refactoring during a bug fix — keep the bug fix minimal.
