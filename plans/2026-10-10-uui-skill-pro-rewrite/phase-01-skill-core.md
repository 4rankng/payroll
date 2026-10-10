---
phase: 1
title: "Rewrite SKILL.md core"
status: pending
priority: P1
dependencies: []
---

# Phase 1: Rewrite SKILL.md core

## Overview
Restructure `~/.agents/skills/uui/SKILL.md` around the dual-mode system, consent gate, and PRO-first sourcing contract, without disturbing unchanged sections (validate/report stages).

## Requirements
- Functional: step-0 mode selection; decision classification (routine vs theme-level); PRO auth hard-fail; PRO-over-FREE sourcing; PRO templates as page default; standing-plan consent; mode recorded in final report.
- Non-functional: keep file scannable (no bloat); preserve licensing/safety invariants; bump `metadata.version` to 1.2.0.

## Architecture
Insertion points in current SKILL.md:
1. **New section "0. Session setup: PRO auth + theme mode"** before "## 1. Orient" — auth check (MCP OAuth / `npx untitledui@latest login`; on failure stop with exact command, explicitly forbid silent FREE fallback); then mode selection: detect established theme during orientation → present `preserve-theme` (default) vs `adopt-pro-theme` (requires explicit consent + before/after scope summary); accept standing consent from an approved repo plan.
2. **New subsection "Consent gate" under Operating principles** — routine decisions (component choice within tokens, composition, states, copy) run silently; theme-level decisions (brand palette, typography, radius/spacing/shadow system, dark-mode mechanism, wholesale redesign of polished surfaces, base-component edits with many consumers) require user approval with: change summary, blast radius (files/surfaces), what is preserved, revert path.
3. **Rewrite "Component sourcing contract"** to: (1) PRO component/template via MCP with verified auth — preferred over FREE equivalent; (2) installed local Untitled (PRO or FREE); (3) Untitled + React Aria composition on project tokens; (4) product-specific UI with documented gap; (5) non-Untitled only with documented constraint. New pages: shortlist 2–3 PRO page templates first.
4. **Mode-aware behavior line** in "Operating principles": in `preserve-theme`, every installed PRO component is re-tokenized to local theme (expression layer yields to project); in `adopt-pro-theme`, PRO visual identity applies to touched surfaces within the consented scope.
5. **Report section (## 8)**: add required line — mode used + consent decisions (or standing-consent reference).

## Related Code Files
- Modify: `~/.agents/skills/uui/SKILL.md`

## Implementation Steps
1. Read current SKILL.md in full (already read this session; re-read before edit to confirm unchanged).
2. Insert section 0; rewrite sourcing contract; add consent-gate subsection; add mode-aware operating-principle line; extend report section; bump version.
3. Verify no licensing invariant weakened (grep for `overwrite`, `PRO`, `API key`).

## Success Criteria
- [ ] Step-0 exists with auth hard-fail + explicit "no silent FREE fallback".
- [ ] Consent gate enumerates theme-level decision classes and required ask format.
- [ ] Sourcing contract ranks PRO above FREE; PRO templates default for new pages.
- [ ] Standing-consent clause present and references active approved plans.
- [ ] `version: "1.2.0"`; frontmatter otherwise intact.

## Risk Assessment
- Inserting a large step-0 can bloat the skill → keep it ≤ ~30 lines, details deferred to `references/mcp-and-cli.md`.
- Contradicting existing "resolve ambiguity … and proceed" line → amend it to explicitly exclude theme-level decisions.
