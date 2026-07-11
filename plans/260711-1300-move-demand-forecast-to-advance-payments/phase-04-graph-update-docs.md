---
phase: 4
title: "Graph update & docs"
status: pending
priority: P3
dependencies: [3]
---

# Phase 4: Graph update & docs

## Overview

Keep the repo's knowledge graph and component-level docs in sync with the new import edges, so future queries like "who renders WalletDemandChart?" return the advance-payments pages instead of (only) the wallet pages. Per the workspace `AGENTS.md` rule: *"After modifying code, run `graphify update .` to keep the graph current."*

## Requirements

- Functional: `graphify-out/graph.json` reflects that `WalletDemandChart` / `WalletDemandCard` are consumed by the advance-payments pages (desktop + mobile), not the wallet pages.
- Non-functional: the two `AGENTS.md` files in `components/wallet/` and `components/advance-payment/` accurately describe the consumer relationship.

## Implementation Steps

1. **Regenerate the graph** (workspace rule, AST-only, no API cost):
   ```bash
   graphify update .
   ```
   Verify with a scoped query:
   ```bash
   graphify query "who renders WalletDemandChart and WalletDemandCard"
   ```
   Expect edges from `pages/admin/AdvancePaymentsPage/index.tsx` and `pages/mobile/admin/AdvancePaymentsPage/index.tsx`; the wallet-page edges should be gone.

2. **Docs touch-up** — only if a `graphify update` doc-sweep flags them; otherwise leave the AGENTS.md files as-is. Optional, low-priority:
   - `frontend/src/components/wallet/AGENTS.md` — the "Wallet balance card shows three amounts…" description is already generic; the `WalletDemandChart` / `WalletDemandCard` entries are not listed in its Key Files table anyway, so no edit strictly required.
   - `frontend/src/components/advance-payment/AGENTS.md` — optionally note that the advance-payments page now also renders the wallet demand-forecast chart + card. Add a one-line note under "Working In This Directory" only if you want explicitness. **Skip if pressed for time** — the graph carries this signal.

3. **Commit** (per repo convention `<type>(<scope>): <subject>`). Two commits for meaningful bisect granularity (red-team correction — a single commit across 4 surfaces would prevent isolating a mobile regression):
   - Commit 1 (Phases 1–2 code + Phase 3 verify):
     ```
     refactor(ui): move wallet demand forecast to /admin/advance-payments
     ```
   - Commit 2 (this phase):
     ```
     chore(graph): regenerate graphify after demand-forecast relocation
     ```
   Run `graphify update .` and validate `graphify query "who renders WalletDemandChart"` shows the advance-payments edges **before** staging `graphify-out/` for commit 2.

## Success Criteria

- [ ] `graphify update .` ran clean; `graphify query` confirms new import edges.
- [ ] (Optional) AGENTS.md notes added or consciously skipped.
- [ ] Commit message follows the conventional format.

## Risk Assessment

**None.** Documentation and graph hygiene only; no runtime impact. Skipping the graph update would leave stale edges — annoying for future codebase questions but not a defect.
