---
title: "Rewrite uui skill: Full Untitled UI PRO + Theme Consent Gate"
status: pending
created: 2026-10-10
source: skill
mode: fast
---

# Rewrite `uui` Skill — Full PRO Utilization + Theme Consent Gate

## Context

- **Brainstorm report (authoritative for rationale):** [`reports/brainstorm-summary.md`](reports/brainstorm-summary.md)
- **Target artifact:** `~/.agents/skills/uui/SKILL.md` + references (global skill — affects all projects)
- **Agreed approach:** Option B — dual-mode (`preserve-theme` default / `adopt-pro-theme` user-opted) + consent gate + all three PRO points (auth step 0, PRO-over-FREE sourcing, PRO templates as page starting point).
- **Compatibility constraint:** `plans/261004-1812-uu-full-migration/` (approved) grants standing consent for theme evolution on this repo. The gate must treat an approved plan's theme authorization as valid standing consent — do not re-ask per session what an active plan already approved; still surface scope before exceeding it.
- **Invariants:** no PRO source/assets/API keys in the skill directory; no auth bypass; no silent FREE fallback on PRO failure; no blind `example --yes` guidance.

## Phases

| # | Phase | File | Depends on |
|---|-------|------|-----------|
| 1 | Rewrite SKILL.md core (mode system, consent gate, PRO-first sourcing) | `phase-01-skill-core.md` | — |
| 2 | Align references with new contract | `phase-02-align-references.md` | 1 |
| 3 | Validation & whole-plan consistency sweep | `phase-03-validate.md` | 2 |

## Acceptance criteria

- [ ] Default invocation on a themed repo changes zero theme tokens without explicit user approval (standing-plan consent honored).
- [ ] PRO auth checked at step 0; failure stops with exact login command — no silent FREE fallback anywhere in the skill text.
- [ ] Sourcing contract ranks PRO above FREE; new pages shortlist PRO page templates first.
- [ ] Theme-level decision classes enumerated with a required ask + blast-radius summary.
- [ ] All 7 files consistent (frontmatter valid, cross-links resolve, mode names identical).
- [ ] No PRO assets/keys/license material present in `~/.agents/skills/uui/`.

## Risks

- **Global blast radius** — skill edits affect every session; mitigate with focused diffs, no unrelated rewrites.
- **Reference drift** — SKILL.md-only edits leave contradictions; phase 2 + 3 enforce cross-file consistency.
- **Over-gating** — gate only at session start + theme-level decisions, never per component.
