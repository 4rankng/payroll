# Dream Consolidation #14 — Cross-Project Changes (2026-07-17)

> Run from the **payroll** session. Cross-project memory writes were blocked by
> privacy isolation (each project's `~/.claude/projects/<proj>/memory/` is only
> writable from within that project). This file captures the exact edits that
> need to be applied from inside the owning project so no signal is lost.
> Re-run `/dream` from the ChatBot and nepocorp sessions (or apply the edits
> below directly) to land them.

## Scope of this dream

- **Phase 1 ORIENT:** 13 native-memory projects mapped. Last-dream was
  2026-07-15 21:52 for most; payroll 2026-07-17 16:57 (Dream #13, 2 min prior).
- **Phase 2 GATHER SIGNAL:** Only 3 projects had transcripts newer than their
  last-dream: **ChatBot** (5, Jul 15 night), **nepocorp** (1, Jul 15–16),
  **payroll** (already consolidated in Dream #13). Other 10 dormant.
- **Phase 3 CONSOLIDATE:** 2 new durable findings (below). Both target
  non-payroll memory → blocked from this session.
- **Phase 4 PRUNE & INDEX:** All 13 MEMORY.md indexes verified — every file
  under 200 lines, all MEMORY.md → topic-file references resolve, no genuine
  stale relative-date references (the flagged "today/yesterday" hits are
  technical algorithm/bug-mechanism descriptions, not unanchored dates).

## Finding 1 — ChatBot: record user's chosen resolution for the 5th outage

The 5 ChatBot sessions (Jul 15, post-Dream-12) all continue the incident already
captured in `2026-07-15-runtime-authority-gate-mutes-bot.md` (image `0f89cedf`
fail-closed runtime-authority gate → bot silently muted). **New durable signal:
the user picked a resolution path not listed in that file.**

**File:** `~/.claude/projects/-Users-dev-Documents-projects-vfic-ats-ChatBot/memory/2026-07-15-runtime-authority-gate-mutes-bot.md`

**Apply this edit** (replace the final "Fix options" paragraph's lead-in):

Find:
```
**Fix options (PROTECTED — need approval):** (1) roll the image back to `dacebe9c`; or (2) provision + activate an installation revision
```

Replace with:
```
**Fix options:** (1) roll the image back to `dacebe9c`; or (2) provision + activate an installation revision — create → validate → activate via `POST /api/v1/admin/installation/revisions`, `…/validate`, `…/activate`, or the InstallationWizard UI (commit `9d0fd34a`); activation flips prod runtime state so it requires explicit approval; or (3) **the user's CHOSEN path (2026-07-15, session 53c8d5c4): add an admin-disable control** so the installation/runtime-authority gate can be turned OFF in the current prod web without rollback or activation — verbatim: "allow admin to turn off this installation feature in the current prod web" → then "if code fix then please do it". Implementation/deploy status not confirmed in these transcripts; verify next session.
```

Source: session `53c8d5c4-3ca7-450b-9f38-04c2c6978e40` (2026-07-15), verbatim user
turns "how to fill in missing data given prod is working app" → "allow admin to
turn off this installation feature in the current prod web" → "if code fix then
please do it".

## Finding 2 — nepocorp: NEVER wipe the demo/vantai DB

The nepocorp session (Jul 15–16, `66b40d79`) had 2 real user turns: (1) the
vantai Jan–Jul realistic-data task (already captured in
`vantai-demo-jan-july-data.md`), and (2) a strong correction that is **not yet**
in memory.

**File:** `~/.claude/projects/-Users-dev-Documents-projects-nepocorp/memory/preferences.md`

**Apply this edit** (insert as a new bullet before the "Never stage
pre-existing uncommitted files" bullet):

```markdown
- **NEVER wipe the demo / vantai DB — only add or update rows** — when seeding realistic data (e.g. vantai Jan–Jul 2026 trips), UPSERT / append only; never destroy existing demo data. Verbatim 2026-07-15: "dont fucking wipe the existing db in demo, you just add on, or update, dont fucking wipe". (source: session 66b40d79, confidence: high)
```

Source: session `66b40d79-9b7b-4ec9-bb35-380ccaa7a168` (2026-07-15), verbatim user
turn "dont fucking wip the existing db in demo, you just add on, or update, dont
fucking wipe".

## Phase 4 — timestamp bookkeeping

`.last-dream` can only be advanced for **payroll** from this session (written
below). The 12 other projects' `.last-dream` files were last set 2026-07-15
21:52 and remain valid; they will advance on their next in-project dream run
(which is also when Findings 1 & 2 should be applied). No data loss — the
findings are preserved here.

## Verification summary (all 13 projects)

| Project | MEMORY.md lines | Refs resolve | Stale rel-dates | New signal |
|---------|-----------------|--------------|-----------------|------------|
| payroll | 141 | ✓ | none real | (Dream #13, 2 min prior) |
| vfic-ats-ChatBot | 43 | ✓ | none real | Finding 1 (blocked write) |
| nepocorp | 103 | ✓ | none real | Finding 2 (blocked write) |
| vfic-ats-ChatBotN8N | 98 | ✓* | none | dormant |
| vantaiphucloc | 94 | ✓* | none real | dormant |
| tuyennhanvien-vn | 23 | ✓* | none | dormant |
| kiosk-app | 26 | ✓* | none real | dormant |
| vfic-ats-tuyennhanvien-vn | 12 | ✓ | none | dormant |
| -Users-dev | 7 | ✓ | none | dormant |
| -Users-dev-Documents-projects | 5 | ✓ | none | dormant |
| vfic-ats-ChatbotX | 6 | ✓ | none | dormant |
| vfic-ats-n8n | 4 | ✓ | none | dormant |
| MetaGPT | 3 | ✓ | none | dormant |

`✓*` = all MEMORY.md → topic-file refs resolve; flagged "missing .md" items are
external plan/repo doc references in prose (e.g. `Plan: feedback-remaining-blueprint.md`),
not memory topic files.
