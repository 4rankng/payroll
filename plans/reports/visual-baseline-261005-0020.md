# W-surface re-baseline — CERTIFIED — 2026-10-05 ~00:20

Window opened after the four surface waves (c8c8e4a7 employee, 312c6091
partner, 15f1855b admin+icon-widening, 4434feae adv-partner/accountant) plus
the auth wave (e24c5fef, which also renewed e2e auth specs). frontend/src
clean throughout; nothing else touched the tree.

## Enumeration run (compare mode, pre-rebaseline)

24/24 visual tests failed — ALL 8 surfaces moved (full-surface restyles, no
partial movement), i.e. the re-baseline set equals the whole visual suite.
The 18 auth.spec.ts rider tests passed.

## Per-surface re-baseline (exactly the moved set)

| Surface | Snapshots | Moved by | Commit |
|---------|-----------|----------|--------|
| admin/dashboard | 3 (1280/390/320) | admin shell+dashboards wave, icon widening | 15f1855b |
| admin/users | 3 | admin wave | 15f1855b |
| admin/projects | 3 | admin wave | 15f1855b |
| admin/employees | 3 | admin wave | 15f1855b |
| admin/timesheet | 3 | admin wave | 15f1855b |
| partner/dashboard | 3 | partner surfaces+shell wave | 312c6091 |
| partner/timesheets | 3 | partner wave | 312c6091 |
| partner/projects | 3 | partner wave | 312c6091 |

24 snapshots regenerated, per-surface runs (8 x --update-snapshots -g
"<surface>"), 3 passed each. Not re-baselined (no such surfaces in suite):
employee (c8c8e4a7) and adv-partner/accountant (4434feae) waves — outside
the harness's admin/partner scope; auth (e24c5fef) rebuilt the login page
but the harness's login selectors (#emailOrUsername / #password /
button[type=submit]) survive (9 emailOrUsername hits in the new Login.tsx)
and all logins succeeded.

Pre-rebaseline check: mask anchors survived the waves —
`.admin-dashboard-page-mobile` shell class, MobilePageHeader /
MobileOperationsPanel / MobileTaskList structure, and the
`button > span.rounded-full.bg-warning` badge are all still present in
source, so the volatile-date/badge/row masks kept working during capture.

## Certification (compare-twice)

| Run | Result | Duration |
|-----|--------|----------|
| Compare 1 | 42/42 passed | 3.2m |
| Compare 2 | 42/42 passed | 3.2m |

Note the suite is now 42 tests, not 24: commit 5b9db6d5 widened
playwright.visual.config.ts testMatch to `**/{visual-baseline,auth}.spec.ts`,
riding the renewed auth.spec.ts (18 mocked-API tests, no screenshots) in
every run. They pass and do not affect baselines; the visual certification
remains 24/24 with the auth riders green alongside.

## Handoff for commit (team-lead)

- frontend/tests/e2e/visual-baseline.spec.ts-snapshots/ (24 PNGs, re-baselined)
- This report. (Spec and fixtures unchanged since 584f5363.)

Status: DONE
Summary: all 8 surfaces moved by the restyle waves; re-baselined per-surface
with wave/commit attribution (15f1855b admin, 312c6091 partner) and
certified 42/42 twice (24/24 visual + 18 auth riders).
Concerns/Blockers: none.
