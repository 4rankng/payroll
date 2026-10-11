---
title: "Phase 6: Diagnosis & Attachments"
status: todo
---

# Phase 6: Diagnosis & Attachments (2b)

## Overview

Turns the chatbot into the support-deflection powerhouse: grounded "why did my Excel upload fail?" answers (D1 from real error data, D2 via in-chat upload dry-run), screenshot diagnosis via omni-modal vision, and void/correct proposals for wrong timesheets (never hard delete as the default verb).

## Requirements

- [ ] **Error-catalog quick win comes first (days, not a phase)**: `frontend/src/utils/import-errors.ts` already maps ~15 patterns → Vietnamese explanations and is already rendered in `BCCUploadModal`/`UploadHistorySheet` — extend it with fix-hints + links and (follow-up) error *codes*; chat diagnosis layers on top of this catalog, never replaces it
- [ ] D1 tool `diagnose_last_import`: reads caller's latest failed import (`timesheet_import_jobs`, `asset_import_metadata.Error` → `domain.ImportError{Row,Employee,Reason}` JSON) + template doc → LLM explains in Vietnamese with row references; answer grounded in the real error rows (never free-form guessing). Note: `Reason` strings embed full names and CCCD verbatim (`bcc_import_multi_position.go`) — send only the minimum slice; retention of PII-bearing chat rows needs the business retention decision applied (not "12 months" by default)
- [ ] Error explanation catalog: pattern → explanation + fix hint for the free-text `Reason` values (content task; note in code docs that error *codes* would be the durable fix — track as follow-up)
- [ ] D2 chat attachments: `POST /chat/attachments` reusing `asset` service **plus hardening** — MIME sniffing (current `ValidateFileType` is extension-only), size caps, **decompression limits for `.xlsx` (it is a ZIP — zip-bomb/entity-expansion guard in `services/excel/` parsing)**, RBAC-gated + owner-checked download, retention cap; `accept` must handle iOS HEIC camera-roll screenshots (convert or reject with clear Vietnamese copy — `image/*` admits HEIC but the vision path wants png/jpg); rejected files get designed Vietnamese error copy, not raw errors
- [ ] Excel path: server-side parse (`services/excel/`) → headers + failing rows slice (PII-minimized) + **validation-only dry-run** (new `bcc_import_dryrun` path — provably zero DB writes) → structured text to the model; raw file bytes never reach the model
- [ ] Screenshot path: `png/jpg` → base64 image input to `mimo-v2.6-pro` (omni-modal); image text treated as untrusted data; best answers reconcile with real job state via tools
- [ ] Wrong timesheet → `propose_timesheet_correction` tool: routes into the **existing timesheet edit-request flow** (`CreateEditRequest`) or files `void_timesheet` approval for true removal needs; approval card shows deterministic impact summary (rows, cycle, payroll already computed?)

## Implementation Steps

1. Import-error read model: query helpers to fetch latest failed import + errors for the caller (scoped like other read tools).
2. Dry-run: extract validation logic from `bcc_import_*` into side-effect-free functions; `bcc_import_dryrun.go` runs template validation + STK/name/position cross-checks in a read-only context; assert zero writes in tests. **Hard gate: before/after equivalence suite** — the live import path creates employees and timesheets feeding payroll (`bcc_import_process.go` `autoCreateEmployeesFromSTK` + `BulkCreateTimesheets` in one flow); any behavior drift in the live path blocks the phase.
3. Attachments endpoint + chat composer file affordance (reuse `components/ui/FileUpload.tsx`, `utils/file-upload.ts` limits incl. `validateEmailAttachments` multi-file/images); mobile sheet variant.
4. Vision call in `llm` client (image content parts); diagnosis tool orchestration combines D1/D2/vision + reconciliation.
5. Correction proposals: service + approval types + impact-summary builder (deterministic).
6. Tests: dry-run produces errors with DB assertions unchanged; upload rejects bad MIME; screenshot smoke (fixture image); correction proposal → approval card with impact; injection via image text (fixture) stays contained.

## Related Code Files

- Create: `internal/app/services/bcc_import_dryrun.go`, `internal/app/services/llmchat/{diagnosis,attachments}.go`, `frontend/src/components/chat/AttachmentUpload.tsx`
- Modify: `internal/app/services/asset/service.go` (MIME sniffing), `internal/infra/storage/file_storage.go` (`ValidateFileType`), chat composer/panel, approval impact-summary builder

## Success Criteria

- [ ] "Why did my upload fail?" answered with real row numbers + causes + fix hints (grounding verified in test)
- [ ] Dry-run proven side-effect-free (DB snapshot assertions)
- [ ] Excel + screenshot attachments work desktop + mobile (320px clean)
- [ ] Wrong-timesheet proposal lands as edit-request/void approval with deterministic impact summary — no hard-delete default
- [ ] `make api-test` green including new diagnosis flows

## Risks

- Dry-run correctness is the phase's biggest hazard — a "dry" run that writes is worse than no dry-run. Mitigate: shared validation functions, read-only tx, DB-unchanged assertions in CI.
- Free-text `ImportError.Reason` makes catalog maintenance ongoing — acceptable for MVP; error codes tracked as a follow-up refactor.
