# Brainstorm Summary: Rewrite `uui` Skill for Full Untitled UI PRO + Theme Consent Gate

**Date:** 2026-10-10
**Status:** Agreed — ready for planning
**Target artifact:** `~/.agents/skills/uui/SKILL.md` + references (global skill, all projects)

## Problem statement

The `uui` skill (v1.1.0) treats PRO as optional licensed material and has only scattered, implicit rules about preserving a project's theme. Two gaps:

1. **PRO underused** — MCP/CLI PRO flows exist but are not the default engine: no step-0 auth check, FREE components not explicitly ranked below PRO, PRO page templates not the default starting point for new pages.
2. **No theme consent gate** — nothing forces the skill to stop and ask the user before changing a well-defined theme. "Utilize PRO fully" and "don't destroy the existing theme" conflict, because PRO components ship with Untitled UI's own visual identity (grays, radii, shadows, type).

### Requirements (user-confirmed)

- Fully utilize Untitled UI PRO: (1) PRO auth check at step 0 with hard, actionable failure messaging; (2) sourcing contract prefers PRO components over FREE equivalents; (3) PRO page templates are the default starting point for new pages.
- Never silently destroy or totally change an existing well-defined theme/design.
- Theme-level changes are always decided **with the user** (consent gate).
- Rewrite `SKILL.md` **and** align references: `mcp-and-cli.md`, `project-conventions.md`, `design-direction.md` (+ light consistency passes on `quality-gates.md`, `source-map.md`, `vite-react.md` as needed).
- Constraints that must survive: no PRO source/assets/API keys copied into the skill; never bypass PRO auth; no blind `example --yes` overwrites in established repos.

## Evaluated approaches

| | A. Theme-shield (conservative) | **B. Dual-mode + consent gate (CHOSEN)** | C. PRO-look default |
|---|---|---|---|
| Idea | Keep structure; add theme-preservation contract; PRO first-choice source | Explicit modes: `preserve-theme` (default) vs `adopt-pro-theme` (user-opted); theme decisions always user-gated | PRO visual identity wins on touched regions |
| PRO utilization | Full catalog, but local theme governs appearance | Full — literally, in adopt mode | Full |
| Theme risk | Near zero | Zero by default; change only with user | High — recreates the stated problem |
| Rewrite cost | Small | Medium | Medium |
| Second-order effects | "PRO polish" doesn't *look* like PRO; expectation mismatch | Consent friction per session; two code paths to maintain | Design drift; two visual grammars to reconcile forever |

**Simplest viable option:** A — rejected because it deflates "PRO fully" into "PRO as scaffolding."

**Chosen: B** — only option delivering both halves of the request. Consent friction kept cheap: **one mode decision per session at step 0, not per component.**

## Final recommended solution

### 1. Mode system (new step 0, before orient)

- Detect theme status during repo orientation: established theme (existing token/brand system, dark-mode mechanism, polished surfaces) vs greenfield/no theme.
- Present mode choice in visible text, then ask:
  - **`preserve-theme` (default):** local theme governs all expression. PRO supplies structure/behavior/IA/states; every installed PRO component is re-tokenized to local tokens via an explicit token-mapping step. Never changes palette, typography, radius/spacing system, or dark-mode mechanism.
  - **`adopt-pro-theme`:** PRO visual identity applied to touched surfaces — requires explicit user consent, with a before/after scope summary (which surfaces, which token groups change).
- Mode is fixed for the session; switching mid-session re-runs the gate.

### 2. Consent gate (decision classification)

- **Routine (no gate):** component choice within existing tokens, page composition, state additions, copy/layout fixes.
- **Theme-level (gate — ask user, show proposed change + blast radius):** brand palette, typography, radius/spacing/shadow system, dark-mode mechanism, wholesale redesign of polished surfaces, base-component edits affecting all consumers.
- Gate output: what changes, what's preserved, files touched, revert path. User approves/edits before code.

### 3. PRO-first sourcing contract (replaces current order)

1. PRO component/template via MCP (auth verified) — preferred over FREE equivalent;
2. installed local Untitled UI component (PRO or FREE);
3. composition of Untitled UI + React Aria primitives on project tokens;
4. product-specific UI (documented domain gap);
5. non-Untitled component (documented compatibility/a11y/migration constraint only).
- New pages: shortlist 2–3 **PRO page templates** as the default starting point, chosen against IA/states/customization cost.
- Auth at step 0: MCP OAuth or `npx untitledui@latest login`. On failure → hard stop with exact login command. **No silent FREE fallback, no auth bypass.**

### 4. Reference updates

- `mcp-and-cli.md`: PRO auth as step 0, failure messaging, PRO-vs-FREE retrieval, hardened `example --yes` warnings.
- `project-conventions.md`: mode rules + token-adaptation (re-tokenizing PRO components to local theme).
- `design-direction.md`: thesis written per mode; adopt mode requires counterfactual check against project identity.
- `quality-gates.md`: validation must confirm mode compliance + consent record in final report.

## Implementation considerations & risks

- **Global blast radius:** skill is at `~/.agents/skills/uui` — affects all projects/sessions. User accepted.
- **Over-chattiness:** mitigate with single session-start gate; routine component work stays silent.
- **Mode confusion:** mode must be stated in the final implementation report of every session.
- **Reference drift:** SKILL.md-only edits would leave references contradicting the new contract — full set updated in same change.
- **Licensing:** PRO assets never enter the skill directory; keys via OAuth/env only.
- **Version safety:** keep v7/v8 `components.json` detection rules unchanged.

## Success metrics

- Default invocation against a themed repo changes **zero** theme tokens without explicit user approval.
- With PRO auth: component + template discovery returns PRO results first; new pages start from a shortlisted PRO template.
- Without PRO auth: skill stops with exact login instructions (never silent fallback).
- All references consistent with mode system; frontmatter valid; no PRO assets/keys in skill dir.
- Final reports include mode used + consent decisions.

## Next steps

1. Run `/ck:plan --fast` with this report as context → produces `plan.md` (`status: pending`) + phase file(s).
2. Execute rewrite of `SKILL.md` + references per plan.
3. Validate: frontmatter, cross-reference consistency, grep for banned patterns (silent FREE fallback, PRO assets in skill).
