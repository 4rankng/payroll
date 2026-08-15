---
title: "Untitled UI PRO reliable tooling"
description: "Make the Untitled UI PRO MCP server and CLI usable reliably across all projects by eliminating duplicate registrations that trigger Cloudflare rate limiting, removing the hardcoded bearer token, and verifying a CLI fallback path."
status: pending
priority: P1
effort: "1-2h"
tags: [tooling, mcp, untitled-ui]
created: 2026-08-15
---

# Untitled UI PRO reliable tooling

## Overview

The `untitledui` MCP server (`https://www.untitledui.com/react/api/mcp`) currently fails to connect with **HTTP 429 / Cloudflare error 1015 (rate limited)** — confirmed by both `claude mcp list` and a fresh single-request probe on 2026-08-15. Root cause: the server is registered **5 times across per-project scopes** in `~/.claude.json` (payroll, nepocorp, ChatBot×2, silversea), so concurrent sessions multiply connection/request volume against one Cloudflare-protected zone. Additionally, the ChatBot registration embeds a **plaintext Bearer token**, and the CLI fallback (`npx untitledui@latest login`) has never been verified on this machine.

This plan fixes all three so the PRO catalog (components, templates, icons) is reachable reliably from any project.

## Evidence (2026-08-15)

- `claude mcp list` → `untitledui: ✘ Failed to connect — HTTP 429 ... error 1015: rate_limited`
- `curl` initialize probe → `429` on a single fresh request (block is zone/IP-level, not session-level)
- `~/.claude.json` contains 5 `untitledui` entries under `projects/*/mcpServers`; one carries `Authorization: Bearer 82c9aa…` in plaintext
- `untitledui` / `uui` CLI not on PATH; `npx untitledui@latest` login state unverified

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Single user-scope `untitledui` MCP registration; all per-project duplicates removed | P1 |
| 2 | No plaintext API token in `~/.claude.json`; auth via OAuth or env-var header | P1 |
| 3 | Verified CLI fallback (`npx untitledui@latest login` + one `search` + one `add` dry run) | P1 |
| 4 | Runbook documenting recovery steps when 429 recurs (it is server-side; it can come back) | P2 |

## Non-Goals

- No changes to any project's frontend code (payroll stays shadcn/ui).
- No re-init or modification of Untitled UI components in silversea or other repos.
- No work around the vendor's rate limiting itself (no proxies/rotation — that violates ToS and is out of scope).

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Consolidate MCP registration](./phase-01-consolidate-mcp.md) | Pending |
| 2 | [Verify CLI fallback + auth](./phase-02-cli-fallback.md) | Pending |
| 3 | [Runbook + guardrail](./phase-03-runbook.md) | Pending |

## Success Criteria

- [ ] `claude mcp list` shows exactly ONE `untitledui` entry, at user scope, connected (✓, not ✘)
- [ ] `grep -c untitledui ~/.claude.json` reflects a single config block; zero Bearer tokens in the file
- [ ] `npx untitledui@latest search "button"` returns PRO results from an authenticated session
- [ ] A real MCP tool call (e.g. `search_components`) succeeds end-to-end in this session
- [ ] Runbook committed to `docs/troubleshooting.md` (payroll repo, cross-project applicable)

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| 429 block persists after cleanup (Cloudflare ban window can last minutes–hours) | Confirmed 2026-08-15: block outlasted cleanup and persists through re-registration. Do NOT re-probe aggressively; re-check `claude mcp list` after a few hours. Phase 2 CLI path may use a different route |
| Removing ChatBot's Bearer token breaks its auth | ChatBot session re-authenticates on next use; the old token should be rotated on the vendor dashboard since it sat in plaintext |
| `~/.claude.json` hand-edit corrupts config | Backup taken (`~/.claude.json.bak-20260815114718`); `claude mcp remove`/`add` CLI used |
| Auth decision (2026-08-15): `UNTITLEDUI_API_KEY` from `~/.zshrc`, referenced via `${UNTITLEDUI_API_KEY}` placeholder in the user-scope header — raw key never in config | Placeholder verified stored, not the literal key. Connection test still blocked by 429 — validate header expansion + auth once the ban lifts |

<!-- slug: untitled-ui-pro-reliable-tooling -->
