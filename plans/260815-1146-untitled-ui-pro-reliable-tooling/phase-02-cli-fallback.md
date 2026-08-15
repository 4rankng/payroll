---
phase: 2
title: "Verify CLI fallback + auth"
status: pending
priority: P1
effort: "30m"
dependencies: []
---

# Phase 2: Verify CLI fallback + auth

## Overview

Establish and verify `npx untitledui@latest` as a working fallback path to PRO material when MCP is rate-limited: login, search, and a non-destructive add dry run.

## Requirements

- Functional: `npx untitledui@latest login` completes browser OAuth and persists credentials locally
- Functional: `npx untitledui@latest search "input" --type components` returns PRO results
- Non-functional: no credentials in shell history, repos, or skill files

## Architecture

The CLI (per `~/.claude/skills/uui/references/mcp-and-cli.md`) uses the same semantic search backend as MCP but authenticates independently via persisted browser login. It is the documented fallback when MCP discovery is unavailable. `add <slug>` without `--overwrite` is safe on existing projects.

## Related Code Files

- None modified (CLI state lives in the user's home dir cache)

## Implementation Steps

1. `npx untitledui@latest login` — complete browser sign-in for the PRO account
2. `npx untitledui@latest search "button" --type components` — confirm PRO results render
3. `npx untitledui@latest search "fintech dashboard" --type templates` — confirm template tier access
4. Dry-run add in a throwaway dir (NOT in payroll/silversea):
   ```bash
   mkdir -p /tmp/uui-check && cd /tmp/uui-check && npx untitledui@latest add button --yes --path ./src/components/uui
   ```
   Inspect output files exist, then `rm -rf /tmp/uui-check`
5. Record CLI version (`npx untitledui@latest --version`) for the runbook

## Success Criteria

- [ ] Login persisted (second command doesn't re-prompt)
- [ ] Component and template searches return results
- [ ] Component files written to throwaway path and cleaned up

## Risk Assessment

If the CLI shares the same Cloudflare-fronted API and also 429s, this phase blocks — retry after the ban window (documented in Phase 3 runbook) rather than hammering. Do NOT loop retries; that worsens the rate limit.
