# Software Development Context Engineering

Use this standard to give coding agents the smallest reliable set of
instructions, source, state, and evidence needed to complete a Payroll task.
The objective is task correctness and continuity, not minimum token count by
itself.

## Scope

This standard governs the software-development process: how coding agents
understand a task, retrieve repository context, make and preserve decisions,
coordinate work, and verify completion. It does not define application runtime
behavior, chatbot conversation context, RAG, user memory, or Payroll business
logic.

This project-specific contract adapts the holistic context layers, progressive
disclosure, structured state, retrieval, and evaluation ideas from the shared
[Context Engineering Guide](https://claude.ai/public/artifacts/f498a4cc-4c45-481c-a6dd-8e1d196dadb0)
to Payroll's repository and production constraints.

## Context Contract

Every task should be grounded in five things:

1. **Intent** — the requested outcome and the behavior that must change.
2. **Authority** — the instruction files and product contracts that govern it.
3. **Active path** — the persisted data or control flow that produces the
   behavior.
4. **State** — decisions, progress, blockers, and scope changes that must
   survive a long session.
5. **Evidence** — checks that support the final completion claim.

If one of these is unknown, retrieve specifically for that unknown. Do not load
more context merely because it is available.

## Instruction Priority

Apply instructions in this order:

1. Platform requirements and safety or privacy controls.
2. Explicit user constraints and accepted scope.
3. The closest applicable `AGENTS.md` for files being read or changed.
4. Root `AGENTS.md` and `CLAUDE.md` non-negotiable project rules.
5. Task-relevant ADRs, standards, API/schema docs, and plan decisions.
6. Examples, historical journals, lessons, and git history.

Examples and history are evidence of precedent, not authority over a current
contract. When two sources conflict, prefer the closer authoritative source and
record the conflict if it changes implementation or verification.

## Start With a Task Packet

For a small task, keep the packet mentally or in the working plan. For
cross-module, risky, delegated, or resumable work, copy
[`plans/templates/task-context.md`](../../plans/templates/task-context.md) to the
active plan directory as `context.md`.

The packet must identify:

- goal and observable success;
- in-scope and out-of-scope behavior;
- target roles, viewports, providers, or environments;
- authoritative contracts and explicit user decisions;
- current hypothesis and unresolved questions;
- validation required for the completion claim.

Update the packet when scope changes, a hypothesis is disproved, or a decision
is made. Keep conclusions and file references; omit conversational narrative
and raw command output.

## Retrieval Ladder

Move down this ladder only when the previous step leaves a concrete unknown.

### 1. Establish the boundary

- Read the user request, root `AGENTS.md`, `CLAUDE.md`, and `README.md`.
- Read the nearest `AGENTS.md` for every target area.
- Read the relevant backend or frontend `CLAUDE.md` only when that layer is in
  scope.
- Inspect `git status` before editing and preserve unrelated user changes.

### 2. Locate the active path

- Run `graphify query "<focused question>"` when the graph is available.
- For relationships, prefer `graphify path` or `graphify explain`.
- Identify the owner, entry point, persisted contract, consumers, and affected
  role or responsive variants.
- Read the narrow source slices and tests returned by that investigation.

### 3. Retrieve governing contracts

Read only the documents needed by the located path:

| Change | Context to add |
|--------|----------------|
| Domain or transaction behavior | Relevant ADR, domain port/entity, service, repository, unit tests |
| API or authorization | Route, handler, DTO, Casbin policy, `docs/api.md`, integration flow |
| Schema or query | Migration rules, repository model, `docs/database.md`, query tests or `EXPLAIN` |
| Frontend behavior | Route/wrapper, desktop and mobile render paths, role permissions, hooks, UI tests |
| Payment or payroll money | Provider abstraction, accounting/ledger contract, reconciliation tests |
| Deployment | Deployment guide, Make target, health checks, rollback evidence |

### 4. Add precedent only if needed

Use a similar implementation, focused git history, troubleshooting entry,
lesson, or journal to answer a specific design or failure question. Do not load
the whole history of the feature.

### 5. Broaden only after a failed focused search

Use raw repository search or broad architecture reports only when focused graph
queries and known indexes do not locate the owner. Record what was missing so
the index or documentation can be improved later.

## Stop Rule

Stop retrieving and begin implementation or reporting when all five questions
have evidence-backed answers:

1. Which code or document owns the behavior?
2. What public, persisted, financial, permission, or responsive contract must
   remain true?
3. Which existing pattern is the closest valid precedent?
4. Which focused checks can prove the requested outcome?
5. What uncertainty or risk remains?

Resume retrieval if a test contradicts the model, the active path differs from
the assumed path, or the user changes scope.

## Compression and Tool Output

- Keep exact values for money, counts, IDs, dates, role scope, provider
  semantics, error text, and acceptance criteria.
- Compress long source or tool output into: finding, evidence location,
  decision, and consequence.
- Discard repeated boilerplate, superseded hypotheses, and output already
  represented by a durable file or test result.
- Put large reusable evidence in the active plan's `reports/` directory. Do not
  create reports for routine command output.
- Keep stable project rules in instruction files and variable task details in
  the task packet so repeated sessions can reuse the stable prefix.

## Task State and Handoffs

Use these states for phases and verification:

- `PENDING` — not attempted.
- `IN_PROGRESS` — actively being worked.
- `PASS` — completed with named evidence.
- `FAIL` — attempted and failed, with evidence and the next action.
- `N/A` — not applicable, with a short reason.
- `BLOCKED` — required but cannot proceed, with the blocker and needed action.

A task is not complete while required items are `PENDING`, `IN_PROGRESS`, or
`FAIL`, or `BLOCKED`. A handoff should contain only the current goal, changed
files, decisions, remaining work, blockers, and latest evidence.

When delegating, send the relevant task-packet fields and exact file ownership,
not the full conversation. Require a compact status and evidence summary back.

## Context Quality Check

Evaluate the packet and final handoff with four probes:

| Probe | It should answer |
|-------|------------------|
| Recall | What exact bug, contract, or requested outcome is being handled? |
| Artifact | Which files changed, and which user-owned changes were preserved? |
| Continuation | What is the next required action or remaining gate? |
| Decision | Why was this path chosen, and what evidence ruled out alternatives? |

Also check:

- no irrelevant document tree or full report was loaded without a reason;
- source and tests support claims, rather than only a prompt or screenshot;
- desktop/mobile and Admin/Partner paths were both traced when applicable;
- skipped validation is explicit and never presented as passing;
- links and file references remain usable by the next session.

## Maintenance

Keep this standard concise and stable. Add a rule only after a recurring failure
or a verified project requirement. Put domain-specific details in the nearest
`AGENTS.md`, durable architectural choices in ADRs, and task-specific facts in
the task packet.
