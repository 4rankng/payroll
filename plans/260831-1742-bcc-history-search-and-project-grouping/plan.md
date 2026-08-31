---
title: "BCC history search and project grouping"
description: "Server-side file-name search and per-project grouping for the Lịch sử tải lên BCC sheet"
status: in-progress
priority: P2
effort: "1d"
tags: [timesheet, bcc, admin, partner]
created: 2026-08-31
---

# BCC history search and project grouping

## Overview

The "Lịch sử tải lên BCC" sheet (`UploadHistorySheet.tsx`, mounted from 4 pages:
admin/partner × desktop/mobile) is a flat, server-paginated list with no way to
find a file. Repeated uploads of the same file (e.g. 5 failed
`GEORIM-luongtuan4TH8.xlsx` retries on 31/8) are indistinguishable noise.

This plan adds:

1. **File-name search** — server-side substring match on
   `metadata.$.original_name`, case- and accent-insensitive via explicit
   `COLLATE utf8mb4_0900_ai_ci` (JSON function results are `utf8mb4_bin`;
   the table collation does not apply — red-team finding A).
2. **Project grouping** — a project dropdown ("Tất cả dự án" / each project,
   reusing the existing `project_id` backend param) plus grouped sections with
   project headers in the all-projects view, backed by a new
   `sort=project` server ordering so groups stay contiguous across pages.
   The two PARTNER mounts must be wired with the `projects` prop — they
   don't pass it today (red-team finding B).

User decision (2026-08-31): dropdown + grouped view; **no** status filter.

## Constraints

- No schema migration — JSON extraction on the existing `assets.metadata`
  column. (Working tree is clean as of 2026-08-31 evening; the morning's BCC
  username fix and Samsung SDS work are committed and deployed — the earlier
  commit-sequencing constraint no longer applies.)
- Search must be server-side (list is paginated 10/page).
- Partner scope unchanged: partners see own uploads only.
- Red-teamed 2026-08-31 — three adversarial reviewers, 12 deduped findings,
  all accepted and folded into the phases below (see Red Team Review).

## Non-goals

- No dedup/collapse of repeated identical uploads.
- No upload-flow/parser/error-format changes.
- No `for_month` picker, no status filter (user declined).
- No new migration or index (an `upload_type` index is the pre-agreed
  escalation if the scan cost ever matters — see Risks).

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Backend: filename search + project ordering](./phase-01-backend-filename-search-project-ordering.md) | Todo |
| 2 | [Frontend: filter row + grouped history view](./phase-02-frontend-filter-row-grouped-history-view.md) | Todo |
| 3 | [Verify and ship](./phase-03-verify-and-ship.md) | Todo |

## Success Criteria

- [ ] Typing part of a file name — any case, with or without diacritics
      (e.g. "georim", "bang cong") — returns only matching uploads;
      pagination totals reflect the filtered result set.
- [ ] All-projects view renders items under project-name section headers on
      every page, ordered project → newest-first; the project dropdown
      narrows to one project (newest-first, no headers).
- [ ] Works for BOTH roles: partner pages show the dropdown and NAMED
      headers (not `Dự án #<id>`); partner still sees own uploads only.
- [ ] Existing behavior intact: download button, error-details expand, status
      badges, 2s polling while pending/processing.
- [ ] No migration; scoped backend build + targeted tests, `pnpm type-check` +
      `pnpm lint`, and `make api-test` all pass.

## Key evidence (scouted 2026-08-31)

- Handler: `backend/internal/transport/http/handlers/timesheet/bcc_import_handler.go:174`
  `ListPartnerImports` — supports `project_id`, `for_month` (exact via
  `MetadataQuery`), `page`/`page_size`; sorted `created_at DESC`.
- Filter plumbing: `backend/internal/infra/persistence/asset_repository.go:182`
  `applyAssetFilters` — `JSON_UNQUOTE(JSON_EXTRACT(metadata,'$.key')) = value`,
  equality only. `List()` at :89 orders via `SanitizeSortColumn` whitelist
  (silently reverts non-identifier input to the default — ordering change
  must be an if/else replacement).
- `assets.metadata` is a native JSON column (`migrations/066_assets_metadata_drop_partner_import.up.sql:2`,
  `domain/asset.go:15` `gorm:"type:json"`) — JSON function results carry
  `utf8mb4_bin`; case-insensitive LIKE requires explicit COLLATE.
- Display name lives in `metadata.$.original_name`
  (`BCCImportStats.OriginalName`); status in `metadata.$.status`; project id
  in `metadata.$.project_id` (JSON number — numeric ordering works).
- Frontend: `UploadHistorySheet.tsx` (flat list, `usePartnerImportHistory`
  hook, queryKey `['partner-imports', params]`); `buildQueryString` skips
  undefined/empty values (`client.ts:353`); `useDebounce` hook exists;
  `PartnerImportListParams` at `timesheet.types.ts:488`. Admin mounts pass
  `projects`; BOTH partner mounts pass only a conditional `projectId`
  (`pages/partner/TimesheetsPage/index.tsx:538`,
  `pages/mobile/partner/TimesheetsPage/index.tsx:342`).

## Risks

- **Scan cost uses the whole `assets` table, not BCC volume** — `assets` is
  shared by 13 upload types and has NO `upload_type` index
  (`migrations/001_init_db.up.sql:322-326`); every list call full-scans, the
  `Count` doubles it, and the sheet polls every 2 s while an upload is
  processing. Acceptable now; the pre-agreed escalation is a small additive
  `upload_type` index (needs user sign-off per no-migration preference) or a
  generated column — NOT the dev-latency signal originally proposed (dev's
  assets table ≈ BCC rows only and cannot see the production-shaped cost).
- Accent-insensitive matching (`ai` collation) is by design — matches
  "bang cong" → "Bảng công". If exact-diacritic matching is ever needed,
  switch to a `_0900_ci` variant.
- `timesheetManagement.projects` availability on partner pages must be
  confirmed during Phase 2 (admin mounts use it; partner pages use
  `timesheetManagement.selectedProject` from the same hook).

## Red Team Review

### Session — 2026-08-31

**Reviewers:** rt-security (Security Adversary), rt-failure (Failure Mode
Analyst), rt-assumptions (Assumption Destroyer) — Standard tier, Fact
Checker + Contract Verifier roles.
**Findings:** 20 raw → 12 deduped (all evidence-passed; 0 auto-rejected)
**Severity breakdown:** 4 Critical/High clusters, 8 Medium
**Dispositions:** 12 accepted, 0 rejected

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| A | JSON function results are `utf8mb4_bin` — case-insensitive LIKE premise false; collation diagnostic unfireable (all 3 reviewers) | High | Accept — explicit `COLLATE utf8mb4_0900_ai_ci` as primary SQL; accent-insensitivity becomes a feature | Phase 1 |
| B | Both PARTNER mounts pass no `projects` — dropdown/named headers dead for the uploader role; Modify list omitted them (rt-security, rt-failure) | High | Accept — wire `projects` at both partner mounts; files added to Modify + commit lists | Phase 2, 3 |
| C | Count totals count rows the renderer drops; NULL-metadata rows lead page 1 under `sort=project` (rt-failure) | High | Accept — `metadata IS NOT NULL` filter on this endpoint | Phase 1 |
| D | sqlite harness cannot run MySQL JSON SQL; api-test has zero GET coverage of the list endpoint (rt-assumptions, rt-failure) | High | Accept — escape-helper unit test + behavioral dev-curl as the explicit gate; false "if harness exists" conditional removed | Phase 1, 3 |
| E | 100-cap counts bytes while the message promises "ký tự" (rt-security) | Medium | Accept — `utf8.RuneCountInString` | Phase 1 |
| F | LIKE-escape order/backslash contract underspecified — silent wildcard bypass (rt-security) | Medium | Accept — exact contract: `\`→`%`→`_`, single backslashes, bound param; fixture `a\b%c_d` | Phase 1 |
| G | "Bypass sanitize whitelist" ambiguity — naive path silently reverts to `created_at DESC` (rt-security) | Medium | Accept — explicit if/else that REPLACES the Order clause | Phase 1 |
| H | Stale search survives sheet close/reopen → duplicate-upload loop (rt-failure) | Medium | Accept — clear search + project filter when sheet closes | Phase 2 |
| I | Headers vanish on continuation pages; page-local counts read as project totals (rt-failure, rt-assumptions) | Medium | Accept — headers whenever all-projects view (`>=1`, not `>1`); counts dropped | Phase 2 |
| J | Perf denominator wrong: whole shared assets table, no `upload_type` index, Count 2×, 2 s polling amplification (rt-security, rt-assumptions) | Medium | Accept — risk restated; escalation = additive index with user sign-off | plan.md Risks |
| K | `go test ./internal/...` gate unachievable — 7 pre-existing services failures (rt-assumptions) | Medium | Accept — scoped gate + pre-declared exclusion list | Phase 3 |
| L | 400 on >100-char search renders as "no uploads yet" (rt-assumptions) | Medium | Accept — client clamps to 100 runes | Phase 2 |

**Reviewer contradiction resolved:** rt-assumptions listed "all four mount
points DO pass projects" as a verified non-finding; direct grep of both
partner mounts (conditional `projectId` only, no `projects`) contradicts it —
overruled by primary evidence, cluster B stands.

**Non-decisions preserved:** the declined status filter stays out of scope;
no finding reversed a user decision.

### Whole-Plan Consistency Sweep

- Files reread: plan.md, phase-01, phase-02, phase-03 (post-application)
- Decision deltas checked: 12 (clusters A–L) + 1 external (in-flight work
  deployed → tree clean → sequencing constraint removed)
- Reconciled stale references: collation-as-assumption prose (plan.md
  Constraints/Risks, phase-01 Requirements), LOWER-fallback risk text,
  "BCC upload volume is small" denominator, "mobile call sites may lack
  projects / graceful" hedging, "if any gorm test harness exists"
  conditional, "sequence commits after in-flight work" constraint,
  `go test ./internal/...` full-tree gate, phase-3 admin-only QA list,
  phase-3 commit file list (now includes the 2 partner pages), phase-2
  `groups.length > 1` condition and header counts
- Unresolved contradictions: 0

<!-- slug: bcc-history-search-and-project-grouping -->
