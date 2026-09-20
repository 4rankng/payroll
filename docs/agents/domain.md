# Domain Docs

How the engineering skills should consume this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`CONTEXT.md`** at the repo root, or
- **`CONTEXT-MAP.md`** at the repo root if it exists: it points at one `CONTEXT.md` per context. Read each one relevant to the topic.
- **`docs/decisions/`**: read ADRs that touch the area you're about to work in. (This repo keeps ADRs in `docs/decisions/`, not `docs/adr/`.)

If any of these files don't exist, **proceed silently**. Don't flag their absence; don't suggest creating them upfront. The `/domain-modeling` skill (reached via `/grill-with-docs` and `/improve-codebase-architecture`) creates them lazily when terms or decisions actually get resolved.

## File structure

Single-context repo:

```
/
├── CONTEXT.md
├── docs/decisions/
│   ├── ADR-001-ddd-clean-architecture.md
│   ├── ADR-006-clock-injection-pattern.md
│   └── ...
└── backend/ + frontend/
```

`CONTEXT.md` doesn't exist yet — that's expected. `/domain-modeling` creates it lazily when a term or decision actually gets resolved.

## Use the glossary's vocabulary

When your output names a domain concept (in an issue title, a refactor proposal, a hypothesis, a test name), use the term as defined in `CONTEXT.md`. Don't drift to synonyms the glossary explicitly avoids.

If the CONTEXT.md doesn't exist yet, proceed silently and use the vocabulary already established in `docs/decisions/` and the `CLAUDE.md` files.

## Flag ADR conflicts

If your output contradicts an existing ADR, surface it explicitly rather than silently overriding:

> _Contradicts ADR-007 (cache invalidation after commit), but worth reopening because…_
