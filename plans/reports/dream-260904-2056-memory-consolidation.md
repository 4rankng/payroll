# Dream run report — 2026-09-04 20:56–21:08 SGT

Consolidation across all 25 `~/.claude/projects/*/memory/` dirs. Signal window: sessions modified ≤7 days (2026-08-28 → 09-04, 127 transcripts). Raw scan: `plans/reports/dream-signal-scan-260904-2056.md`.

## Consolidations applied

### payroll (session-writable)
| # | Change | Source |
|---|--------|--------|
| 1 | `reference_zalo_token_rotation.md`: new section "Delivery verified (2026-09-04)" — password-reset ZNS sends CONFIRMED going out (log writes only after `s.zalo.Send()` success, `error_code:0` + `msg_id`, `backend/internal/app/services/zaloreset/service.go:185-193`); alert spike = UX-driven same-day feature rollout, not token failure. Side-finding recorded: **no Sentry/CloudWatch/APM in payroll** — observability = `/opt/payroll/logs/app.log` + ephemeral Docker stdout. Frontmatter description + modified date updated. | session `e3d7a8d6` 09-04 |
| 2 | `MEMORY.md`: zalo pointer line extended with the verified-delivery + no-APM summary. | same |

Not consolidated (already recorded, no duplication): BCC BUMHAN T09/Position-column/strategy-routing work (`project_bcc_bumhan_template_guide.md`, current through `a0c16e95`), payrate active-before-entry-date rule (= `project_payrate_effective_from.md` rule 1, effective_from-only model), "make deploy works every time" (`feedback_make_deploy_always_works`), payrate update-as-create split + effective_to ruling (file current through 09-04).

## Permission-blocked (intended edits — apply or approve)

This session is anchored to the payroll project; edits to other projects' memory files were denied as sensitive. Intended content, ready to apply:

### silversea — `git-workflow.md`
Add after the 08-16 history-rewrite paragraph:
> **Branch hygiene (2026-09-04, user directive):** repo keeps ONLY `main` and `prod` branches; delete all others (dependabot/*, feat/*, main-backup). `prod` must always carry the latest `main` (fast-forward prod to main after pushes). (source: session 2026-09-04, confidence: high)

Also extend the frontmatter `description` with `; branches main+prod only, prod tracks latest main (2026-09-04)`, and update the `modified:` field.

### silversea — `MEMORY.md`
Index line update:
> - [Git workflow](git-workflow.md) — trunk-based; main+prod only, prod tracks latest main (09-04); plans/+prompts/ never committed

### nepocorp — `MEMORY.md`
1. Fix internally-contradicted index row (topic-file row said CI running; Servers table correctly says removed) — replace the row with:
> \| [ci-billing-blocked-manual-deploy.md](ci-billing-blocked-manual-deploy.md) \| CI history: billing block resolved 08-22 → **GitHub Actions removed entirely `2371af3f` 09-03** — deploys manual-only (`make deploy` / `make demo`) \| 2026-09-04 \|

2. Refresh the header "Last consolidated" line to 2026-09-04 (content unchanged otherwise).

### Cross-project orphans (informational, no write needed)
- `vantaiphucloc/MEMORY.md`: add index line for `project_ocr_prompt_invariance.md` (CORRECTED 07-07: OCR prompt count-enforcement wording DOES move 32B recall; check-digit self-correction fails on Orion-32B).
- `vfic-ats-tuyennhanvien-vn/MEMORY.md`: add index line for `zns-per-project-template-routing.md` (PR #4 open, per-project ZNS template routing for phongvan_pass w/ LGD variant, generic 599840 fallback).
- `vfic-ats-ChatBot/dream-history.md` — internal log, intentionally unindexed, no action.

## Phase 4 verification results

| Check | Result |
|---|---|
| MEMORY.md ≤ 200 lines | ✅ max = payroll 157 |
| Referenced topic files exist | ✅ 0 missing across 25 projects |
| Relative dates in indexes | ✅ none |
| Unreferenced-anywhere topic files | 3 (2 merit index lines — blocked, listed above; 1 internal log) |
| Entries >90d stale w/ no refs | none — oldest active project = tuyennhanvien-vn (Jun 20, 76d) |
| Contradictions resolved | 1 applied (zalo remediation timeline); 1 identified (nepocorp CI row) but blocked |
| .last-dream stamped | ✅ 25/25 dirs, `.dream-pending` absent/cleared |

## Projects with no new signal (phase 3 no-op, phase 4 verified only)

silversea (all scanned signals already captured — dispatch-reassign-free, driver-app spec, QuyTrinhO2C, realistic-data directive, prod-live-no-resets), ttsoft (landing-page file current), vantaiphucloc, ChatBot, silversea-frontend/backend/testplan, nepocorp (09-04 directives already recorded: one-time anonymized snapshot, demo Abc123, CI removal `2371af3f`), kiosk-app, vfic family, MetaGPT, -Users-dev-Documents-projects, -Users-dev, -agents-skills, ati-solutions(+hdv-payout), l3-sdlc-poc, Documents-silversea + Library/Application-Support-Claude (empty dirs, no MEMORY.md — left as-is; stray slugs, real silversea memory lives at projects-silversea).

## Unresolved

1. **Cross-project memory edits denied** — the 4 silversea/nepocorp/vantaiphucloc/vfic-tuyennhanvien-vn index/topic edits above need either (a) running /dream from a session anchored to each project, or (b) explicit approval of writes to `~/.claude/projects/*/memory/` outside this session's project.
2. Zalo prevention (make-restore nulling `zalo.credentials` / env-gated cron) still unimplemented in code — standing OPEN item, unchanged by this run.
