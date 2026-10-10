---
phase: 3
title: "Validation & consistency sweep"
status: pending
priority: P2
dependencies: ["1", "2"]
---

# Phase 3: Validation & consistency sweep

## Overview
Mechanical + editorial verification that the rewritten skill is internally consistent, safe, and license-clean before handoff.

## Requirements
- Functional: frontmatter valid; cross-links resolve; banned patterns absent; end-to-end read-through simulates two scenarios (themed repo default session; PRO auth failure).
- Non-functional: read-only unless a defect is found; fixes stay within phase 1–2 scope.

## Architecture
Checks:
1. **Frontmatter**: `name`, `description`, `license`, `compatibility`, `metadata.version` present in SKILL.md; phase/heading structure intact.
2. **Link check**: every `references/*.md` link in SKILL.md resolves on disk.
3. **Banned-pattern grep** in `~/.agents/skills/uui/` (excluding LICENSE/NOTICE): API keys/tokens (`sk-`, `api_key`, bearer literals); PRO source blobs; wording that implies silent fallback on auth failure (`fall back to FREE` without consent); any `example ... --yes` recommendation without the hazard warning.
4. **Consistency**: `preserve-theme` / `adopt-pro-theme` strings consistent; consent-gate decision classes match between SKILL.md, project-conventions, quality-gates; sourcing order identical in SKILL.md and mcp-and-cli.
5. **Scenario walkthrough** (code-read): (a) themed repo, "polish UI" brief → lands in preserve-theme, gate fires only on theme-level change; (b) MCP auth expired → step-0 hard stop with login command; (c) repo with approved migration plan → standing consent honored, scope check still applies.
6. **Diff review**: `git status`/diff not applicable (skill dir outside repo) — instead `ls -la` + full re-read of changed files.

## Related Code Files
- Review: all of `~/.agents/skills/uui/SKILL.md` + `references/*.md`

## Implementation Steps
1. Run checks 1–4 (grep + link resolution).
2. Run scenario walkthrough (5); fix defects via targeted edits to phase 1–2 files only.
3. Re-run failed checks; produce final report: files changed, check results, any deviations.

## Success Criteria
- [ ] All six checks pass or have documented, justified exceptions.
- [ ] Scenario (a) yields zero unconsented theme changes; (b) hard-stops; (c) honors standing consent.
- [ ] No banned patterns present.

## Risk Assessment
- False-positive greps on benign words (`fallback` in legitimate contexts) → inspect matches manually; judge against intent (silent vs consented fallback).
