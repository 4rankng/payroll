# Prompt Library

Reusable prompts for common engineering tasks. Each prompt is structured for use with Claude Code, Codex, or Gemini CLI. Adapt the placeholders to your specific task.

## Structure

Every prompt follows this pattern:

```
Goal: What to achieve
Context to gather: Where to look first
Constraints: Rules that must be followed
Output format: What the result should look like
Verification: How to confirm it's done
```

## Prompts

| Prompt | Use When |
|--------|----------|
| [Feature Implementation](feature-implementation.md) | Adding a new backend feature (entity → port → repo → service → handler → route → casbin → DI) |
| [Bug Fix](bug-fix.md) | Fixing a specific issue — uses the Locate → Repair → Validate loop |
| [Refactoring](refactoring.md) | Improving code structure without changing functionality |
| [Security Review](security-review.md) | STRIDE/OWASP review of a change or component |
| [Performance Optimization](performance-optimization.md) | Profiling and eliminating N+1 queries, adding indexes |
| [API Design](api-design.md) | Adding a new API endpoint end-to-end |
| [Database Migration](database-migration.md) | Creating a safe, idempotent migration |
| [Architecture Review](architecture-review.md) | Auditing layer boundaries and dependency direction |

## Usage Tips

1. **Scout first.** Before using a prompt, read the relevant `AGENTS.md` file in the target directory. Every major module has one.
2. **Adapt, don't copy.** Replace placeholders with your specific context. Remove sections that don't apply.
3. **Verify after.** Every prompt includes a verification step. Don't skip it.
4. **Use with `/ck:cook`.** These prompts pair well with the cook skill — pass the prompt as the task description.

## Relationship to Plan Templates

The [`plans/templates/`](../../plans/templates/) directory has structured plan templates (feature-implementation, bug-fix, refactor). These prompts are the *instructions* you give an AI agent; the plan templates are the *output format* the agent fills in.
