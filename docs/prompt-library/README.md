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

For cross-module, risky, delegated, or resumable work, create a small task
packet from
[`plans/templates/task-context.md`](../../plans/templates/task-context.md).
Follow the [Context Engineering Standard](../standards/context-engineering.md):
load the owning path and contracts first, add references only when a concrete
unknown requires them, and preserve decisions/evidence instead of raw output.

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

1. **Scout first.** Read the root and nearest applicable `AGENTS.md`, then use `graphify query` to locate the owning path.
2. **Adapt, don't copy.** Replace placeholders with your specific context. Remove sections that don't apply.
3. **Verify after.** Every prompt includes a verification step. Don't skip it.
4. **Use with `/ck:cook`.** These prompts pair well with the cook skill — pass the prompt as the task description.

## Relationship to Plan Templates

The [`task-context` template](../../plans/templates/task-context.md) preserves
intent, authority, active-path findings, decisions, and evidence across a long
task. Prompt-library files define task instructions; active execution plans and
phase files live in their timestamped directory under `plans/`.
