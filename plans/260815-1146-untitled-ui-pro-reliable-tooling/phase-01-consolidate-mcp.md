---
phase: 1
title: "Consolidate MCP registration"
status: pending
priority: P1
effort: "30m"
dependencies: []
---

# Phase 1: Consolidate MCP registration

## Overview

Collapse the 5 per-project `untitledui` MCP registrations in `~/.claude.json` into a single user-scope registration, removing the plaintext Bearer token in the process, then re-probe the endpoint.

## Requirements

- Functional: exactly one `untitledui` MCP entry, at user scope, reachable by every project
- Functional: zero plaintext credentials in `~/.claude.json`
- Non-functional: `~/.claude.json` edit is done safely (backup first, CLI commands over hand-edits)

## Architecture

Claude Code stores MCP servers per scope: user (`~/.claude.json` top level `mcpServers`), project (`.mcp.json`), and local (per-project entry in `~/.claude.json` `projects/*/mcpServers`). Local-scope servers are spawned per session for that project — 5 registrations × concurrent sessions = N simultaneous clients hammering one Cloudflare-protected zone, which is what trips error 1015. One user-scope registration shares the connection profile and stops the multiplication.

## Related Code Files

- Modify: `~/.claude.json` (via `claude mcp` commands, not manual JSON edit)

## Implementation Steps

1. Back up: `cp ~/.claude.json ~/.claude.json.bak-$(date +%Y%m%d%H%M%S)`
2. Remove all local-scope duplicates:
   ```bash
   claude mcp remove untitledui -s local   # repeat per project cwd, or edit JSON after backup:
   ```
   Practical route (one-shot, avoids cd-ing into 5 projects):
   ```bash
   python3 - <<'EOF'
   import json
   p = '/Users/dev/.claude.json'
   d = json.load(open(p))
   removed = []
   for proj, cfg in d.get('projects', {}).items():
       if 'untitledui' in cfg.get('mcpServers', {}):
           del cfg['mcpServers']['untitledui']
           removed.append(proj)
   json.dump(d, open(p, 'w'), indent=2)
   print('removed from:', *removed, sep='\n  ')
   EOF
   ```
3. Add user-scope registration (OAuth-capable, no token):
   ```bash
   claude mcp add --scope user untitledui --transport http https://www.untitledui.com/react/api/mcp
   ```
4. Verify: `claude mcp list` → one `untitledui` entry; `grep -c '"untitledui"' ~/.claude.json` → 1
5. Confirm no Bearer remains: `grep -i bearer ~/.claude.json` → empty
6. Re-probe endpoint (`claude mcp list` shows connect status). If still 429, note the time and proceed to Phase 2 — the block is server-side and expires on its own.

## Success Criteria

- [x] `claude mcp list` shows exactly one `untitledui` (user scope)
- [x] No Bearer token in `~/.claude.json`
- [x] Backup of pre-edit config exists

## Risk Assessment

Cloudflare ban window may outlast the cleanup — acceptable; Phase 2 doesn't depend on MCP. The vendor token that sat in plaintext should be rotated on the Untitled UI dashboard (tell the user; cannot be automated).
