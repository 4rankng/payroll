# Journal: Excel Parsing Architecture Refactor — 2026-09-06

**Scope:** 11 commits on `main` (`f39cdb72..e240a369`), plan
`plans/260906-1120-excel-parsing-refactor/`, audit evidence
`plans/reports/excel-audit-260906-1025.md`.

## What changed

- **Safety net first.** Real-partner fixtures for all five BCC formats
  (user-staged in `testplan/excelfixture/`, anonymized copies committed under
  `backend/tests/fixtures/bcc/<format>/`) with golden parse output, a routing
  table test (hidden sheets, `"STK "` trim, unknown-template failure,
  priority order), and the employee import parser's first-ever tests.
- **One routing truth.** `excel/bcc_formats.go`'s ordered registry replaced
  three stacked detection layers; `DetectFormat` is a loop over it. Strategy
  Match methods and the detector now share one fingerprint source. Goldens
  stayed byte-identical through the rewrite.
- **God processors decomposed.** The 1,324-line weekly file split into
  per-format files with named methods; the 609-line legacy pipeline shrank to
  348 with the STK block, finalizers, and contract predicates extracted and
  unit-tested by name (`bcc_import_dispatch_test.go`).
- **Shared primitives.** `internal/pkg/excelkit` (verbatim ports only);
  OnePay/wallet-bulk header resolution unified through `AliasTable`; FlexPay
  reads each sheet once instead of three times.
- **Upload hardening (approved behavior change).** New
  `internal/transport/http/uploadguard` (MaxBytesReader before FormFile,
  size caps, ZIP/OLE2 magic sniff) on six Excel endpoints — the
  unbounded-io.ReadAll and ~16GiB-decompression exposure is closed.
- **Docs.** `services/excel/AGENTS.md` documents the add-a-template
  walkthrough (parser + registry line + test + fixture) and the invariants.

## What made it work

Writing the characterization net before touching code. Every phase gate was
goldens-byte-identical + `go test ./...` clean + integration suite identical
to baseline (281/304/0 across all eight gates), so each step was shippable
and a regression would have been caught at the phase that caused it.

## What didn't go as planned (honesty section)

Four plan items died on contact with the code — all in the direction of
*less* unification, all verified before deciding:

1. The two `parseForMonth`s looked duplicated but have divergent contracts
   (different accepted formats, different user-visible error text). Merging
   would have changed frontend-visible strings.
2. The audit's "getOrCreate trio ~180 LOC near-verbatim" claim was wrong at
   the signature level — the employee variant carries bank validation and
   StartDate/PaymentSchedule parsing the FlexPay variant lacks. Forced
   unification would have been behavior drift in an untested provisioning
   path; invoked the plan's documented fallback.
3. `includeFlexibleEmployees` removal was abandoned mid-phase: the flag is
   hashed into the BCC idempotency fingerprint and persisted on
   `timesheet_import_jobs`, so removal breaks stored-fingerprint comparison
   for in-flight retries. Deferred with a written rationale.
4. `ResultStrategyFactory` was reported dead by the audit; grep found live
   callers. Audit findings are hypotheses until grepped.

Also: a scripted import-insertion corrupted two files' import blocks (caught
by build, fixed immediately) — regex-editing import blocks is a trap; use
goimports or explicit anchors.

## Follow-ups

- Deploy the series (not yet deployed at journal time); watch the upload
  guard's first production day (rejected-upload logs).
- `includeFlexibleEmployees` fingerprint-migration decision.
- `bulktransfer/row_parser.go` `time.Local` → clock location.
- Employee import silent row-skips (re-introduce `dto.ImportRowIssue` at
  first adoption — see review below).
- `.ua` knowledge graph needs `/understand --full` (55 structural files;
  incremental update correctly refused).

## Review gate (post-series adversarial pass)

Independent reviewer + independent test run. Tests: GREEN — baseline matched
exactly (304/281/0/23), goldens byte-stable double-run, guard negative paths
passing. Review returned FIX-FIRST with two blockers, both fixed:

1. **The upload-guard sweep underclaimed its own scope**: the first wave
   guarded 6 endpoints but the commit said "all". Five more multipart Excel
   endpoints were still unbounded (flexible-employee list, advance-payment
   result, wallet reconciliation, template import, bulk-transfer result).
   All guarded now — bulk-transfer result takes `allowXLS` (OLE2), the only
   `.xls` acceptor. Lesson: when a commit message says "all", enumerate the
   full set before writing it.
2. **`dto.ImportRowIssue` shipped dead** — zero consumers, from the same
   series that deleted dead code. Removed; re-introduce at first adoption.

Disclosed and accepted: `AliasTable.Resolve` is a deliberate union (each
island gained the other's resolution steps — strictly more permissive on
pathological headers, canonical mappings unchanged); employee/BCC caps moved
10MB→20MB and rejection strings are now Vietnamese (within the approved
behavior-change class).

## Review gate, round 2 (max-effort adversarial pass)

A second review (`/code-review --fix max`, multi-agent) returned 13 findings;
all fixed in three commits (`9ab5f463`, `d9d0af8c`, `159613b9`), verified by
build/vet/unit (304 ok), `make api-test` identical to baseline (281/304/0/23),
and a live fixture spot-check on the reloaded dev server (real EVA file
completes 0/1092/0 as before; `.docx` → immediate 400 extension rejection;
blank-filename part → 400 at the multipart layer). The two that mattered:

1. **PostForm before the guard deadened the body cap** on UploadBCC and
   ImportFlexPayFile — gin's PostForm parses the whole unbounded body before
   MaxBytesReader wraps it. Fix: guard first; PostForm reads after the
   guard's own parse are memory-only. Lesson: a request-body cap only works
   if it wraps the body before ANY multipart read, including form fields.
2. **AliasTable segment-before-collapsed ordering could rebind previously
   correct columns** on stacked bilingual headers (the round-1 "strictly more
   permissive" framing understated it — preemption, not just addition).
   Fix: segments resolve last; provably restores both islands' original
   per-cell outcomes.

Also: extension/filename gates restored in uploadguard; sao-kê/OnePay-fee/
template-converter/bulk-transfer opens routed through excelkit caps;
date-row fingerprint deferred to a second classification phase (restores the
old detector's early exit, outcome-identical); dead helpers deleted. Caps
(50/10 MiB excelkit, 20 MiB guard) deliberately untouched — policy pending
with the user (legit ~12k-row workbooks vs decompression-bomb bound).
