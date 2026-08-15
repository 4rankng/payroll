---
phase: 3
title: "Runbook + guardrail"
status: pending
priority: P2
effort: "30m"
dependencies: [1, 2]
---

# Phase 3: Runbook + guardrail

## Overview

Persist the recovery procedure so future 429 incidents (server-side, can recur) are a lookup, not an investigation. Add the troubleshooting entry in the payroll repo's docs (it's cross-project applicable) and a memory note.

## Requirements

- Functional: `docs/troubleshooting.md` gains an "Untitled UI MCP 429" entry with diagnosis + recovery steps
- Functional: auto-memory records the single-registration convention
- Non-functional: no secrets in the runbook

## Architecture

N/A — documentation only.

## Related Code Files

- Modify: `docs/troubleshooting.md` (payroll repo)

## Implementation Steps

1. Read `docs/troubleshooting.md`, match its existing entry format
2. Add entry covering:
   - Symptom: `claude mcp list` → `untitledui ✘ HTTP 429 / Error 1015 rate_limited`
   - Diagnosis: `curl -s -o /dev/null -w "%{http_code}" -X POST https://www.untitledui.com/react/api/mcp ...` (single-request probe distinguishes zone-level ban from session overload)
   - Rule: ONE user-scope registration only — never `claude mcp add` per-project
   - Recovery: wait out the ban window (minutes–hours), use CLI fallback meanwhile, verify with `claude mcp list`
   - Auth: OAuth preferred; if an API key is ever needed, env var — never inline
3. Commit as `docs(troubleshooting): add Untitled UI MCP 429 recovery runbook`
4. Write auto-memory file `reference_uui_single_mcp_registration.md` + MEMORY.md index line

## Success Criteria

- [ ] Troubleshooting entry exists and matches doc's format
- [ ] Memory saved; future sessions see the single-registration rule

## Risk Assessment

Minimal. Only follow repo doc conventions; no code touched.
