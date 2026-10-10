---
phase: 2
title: "Align references with new contract"
status: pending
priority: P2
dependencies: ["1"]
---

# Phase 2: Align references with new contract

## Overview
Update the reference files so none contradicts the new mode/consent/PRO-first contract; light consistency pass on the remainder.

## Requirements
- Functional: auth-as-step-0 + failure messaging in CLI guide; mode + token-adaptation rules in conventions; per-mode design thesis; mode-compliance check in quality gates.
- Non-functional: no unrelated rewrites; keep each reference's existing structure and tone.

## Architecture
- **`references/mcp-and-cli.md`** (primary): retitle Authentication → "Authentication (step 0)"; add failure protocol (exact login command, stop, never silent FREE fallback, never bypass); add PRO-vs-FREE retrieval note (request PRO via `get_component`/`get_component_bundle`; treat FREE as fallback only when PRO access is legitimately unavailable *and user-approved*); reinforce `example --yes` hazard (unchanged).
- **`references/project-conventions.md`**: add "Theme modes" section — `preserve-theme`: re-tokenize installed PRO components to local brand scale/semantic tokens (mapping steps mirror existing "Brand customization"); `adopt-pro-theme`: PRO values become source of truth only within consented scope; never introduce second dark-mode strategy.
- **`references/design-direction.md`**: thesis step gains mode awareness — preserve mode infers direction from repo (no new palette/type unless consented); adopt mode documents PRO-derived direction + counterfactual check against project identity.
- **`references/quality-gates.md`**: add checklist items — correct mode stated; consent/standing-consent recorded; no theme tokens changed outside consented scope; PRO auth failure handled per protocol.
- **`references/source-map.md`, `references/vite-react.md`**: read-through only; fix any statement that now contradicts PRO-first or mode rules (expected: minimal/no edits).

## Related Code Files
- Modify: `~/.agents/skills/uui/references/mcp-and-cli.md`
- Modify: `~/.agents/skills/uui/references/project-conventions.md`
- Modify: `~/.agents/skills/uui/references/design-direction.md`
- Modify: `~/.agents/skills/uui/references/quality-gates.md`
- Review: `~/.agents/skills/uui/references/source-map.md`, `references/vite-react.md`

## Implementation Steps
1. Read each target file; edit primary four per architecture above.
2. Read-through source-map + vite-react; edit only on contradiction.
3. Confirm every mode name string (`preserve-theme`, `adopt-pro-theme`) is byte-identical across files.

## Success Criteria
- [ ] No reference advises behavior forbidden by the new SKILL.md contract.
- [ ] Auth failure protocol present in `mcp-and-cli.md` matching SKILL.md step 0.
- [ ] Quality gates can verify mode + consent.
- [ ] Mode names identical across all files.

## Risk Assessment
- Over-editing references (scope creep) → only contract-relevant sections touched; record any intentionally untouched file here.
