# Prompt: Bug Fix

Use when fixing a specific issue. Follows the **Locate → Repair → Validate** loop (Agentless philosophy) instead of one massive prompt.

## Prompt

```
Goal: Fix [BUG_DESCRIPTION] — [EXPECTED_BEHAVIOR vs ACTUAL_BEHAVIOR]

Phase 1: LOCATE
- Search for the relevant code using graphify query "<question>" if available
- Read the AGENTS.md in the affected module
- Identify the root cause — don't guess, verify by reading the code
- Check git log for recent changes to the affected files:
  git log --oneline -10 -- <file_path>
- Check docs/troubleshooting.md for known issues
- State the root cause in one sentence before proceeding

Phase 2: REPAIR
- Make the minimal change needed to fix the root cause
- Follow existing patterns in the file — match naming, style, error handling
- Do NOT introduce new patterns or refactor unrelated code
- If the fix touches a public contract (function signature, API response, DB schema),
  STOP and ask before proceeding

Phase 3: VALIDATE
- Run the affected test:
  cd backend && go test ./internal/[path]... -v -run TestXxx
- Run the full test suite:
  cd backend && go test ./... -race -cover
- Run integration tests if the bug is API-level:
  make api-test
- Run lint:
  cd backend && gofmt -l . && golangci-lint run ./...
- Verify no side effects: check all callers of changed functions

If validation reveals a side effect or regression:
- STOP. Do not silently patch around it.
- Present what broke and 2-4 options for resolution.
```

## Key Principles

1. **Locate before repairing.** Don't start coding until you can state the root cause in one sentence.
2. **Minimal change.** Fix the root cause, not the symptoms. Don't refactor unrelated code.
3. **Validate immediately.** Run tests after the fix, not at the end.
4. **Never trust the first answer.** If the fix doesn't work, re-locate and try again.
