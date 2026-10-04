# W0 visual-baseline harness — CERTIFIED — 2026-10-04 ~23:55

Window opened after W5 (50efc462), W6 (db7b834d), W7 (457661f3) landed;
frontend/src clean; nothing else touched the tree during the runs.

## Final sequence results

| Run | Result | Duration |
|-----|--------|----------|
| Regenerate (--update-snapshots) | 24/24 passed | 4.2m |
| Compare 1 | 24/24 passed | 2.9m |
| Compare 2 | 24/24 passed | 2.5m |

Target met: 24/24 x3, back-to-back, on the frozen tree. The runs are also
notably faster than earlier windows (4.2m vs 10m for regeneration) because
the pinned endpoints removed the >9s stall waits.

What the machinery now holds constant, all verified in this window:
rendering nondeterminism (network-quiet window + skeleton/"Dang tai" waits),
roster-banner presence (notifications pin), login stalls (retry),
slow-endpoint stalls (pins), and business-data drift (31 captured-real
fixture pins, role-scoped).

## Certified state for wave reviews (W1+ workflow)

- Reference baselines: frontend/tests/e2e/visual-baseline.spec.ts-snapshots/
  (24 PNGs, regenerated this window on the W7 tree).
- After each migration wave: compare mode only —
  `pnpm exec playwright test --config playwright.visual.config.ts`
  — review -diff.png in the HTML report; regenerate individual baselines
  with --update-snapshots only for intentionally changed surfaces.
- The harness is data-stable by construction now: normal QA activity no
  longer breaks it (all volatile data endpoints are pinned).

## Handoff for commit (team-lead)

- frontend/tests/e2e/visual-baseline.spec.ts (modified)
- frontend/tests/e2e/visual-baseline.spec.ts-snapshots/ (24 PNGs, final)
- frontend/tests/fixtures/visual-baseline/ (31 JSON fixtures, NEW)
- Reports: plans/reports/visual-baseline-261004-{1812,1958,2100,2215,2320,2355}.md

Status: DONE
Summary: final sequence executed clean — 24/24 regeneration, 24/24 compare
x2 on the frozen W7 tree; W0 harness certified for the migration waves.
Concerns/Blockers: none.
