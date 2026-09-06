---
phase: 5
title: "Verification & rollout"
status: completed
priority: P1
effort: "0.5d"
dependencies: [2, 3, 4]
---

# Phase 5: Verification & rollout

## Overview
Full test matrix (testplan doc authored in pre-flight, before any verification),
mandatory tester + code-reviewer subagents, then the deploy chain and the first
real campaign.

## Requirements
- Functional: every acceptance criterion in `plan.md` demonstrated.
- Non-functional: tester + code-reviewer subagents spawned (cook-skill mandate);
  auto review cycle fixes criticals (≤3 loops); commits conventional with explicit
  paths; deploy babysat to completion with evidence.

## Implementation Steps
1. Local gates: mig up, `make dev`, backend `make api-test`, frontend
   vitest/tsc/lint, then the manual matrix in
   `testplan/260906-employee-ad-banner.md` (sheet → card → dismiss → edit →
   version-bump → expiry → untargeted employee, on BOTH employee pages).
2. Tester subagent: run + interpret the suites; debugger if failures.
3. Code-reviewer subagent with acceptance criteria + scout context; auto-fix cycle.
4. Commits per phase (`feat(ad-banner): …`), explicit paths only.
5. Demo: mig 106 manual + `make demo` + verify demo.tingting.vip.
6. Prod: mig 106 manual + `make deploy` from repo root (amd64), verify prod
   tag == HEAD, babysit, re-run `make api-test`.
7. Create the LG Display campaign through the admin UI (audit trail).
8. Plan sync-back (all phase files + index), docs-impact check, journal entry.

## Success Criteria
- [x] All suites green; review auto-approved (≥9.5, zero criticals) or user-escalated
- [x] Demo + prod verified; LGD campaign live for the targeted project
- [x] Rollback available: previous image; tables may remain

## Risk Assessment
Deploy-time drift: build from the pushed SHA and confirm prod tag == HEAD (prior
lesson — piped make hides failures). If the LGD project id is uncertain at publish
time, pick it from the projects list in the composer (names, not ids) and confirm
with the user before going live.
