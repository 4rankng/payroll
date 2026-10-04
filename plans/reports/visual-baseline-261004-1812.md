# Visual baseline harness — 2026-10-04

Task: pre-migration Playwright screenshot baselines for the Untitled UI PRO migration (key surfaces × admin/partner × 1280/390/320), exact-match, deterministic.

## What the spec covers

File: `frontend/tests/e2e/visual-baseline.spec.ts` (new; the only file created besides auto-generated snapshot outputs).

24 tests: 8 surfaces × 3 viewports.

| Role | Surfaces | Viewports |
|------|----------|-----------|
| admin (frankng) | `/admin` (dashboard index), `/admin/users`, `/admin/projects`, `/admin/employees`, `/admin/timesheet` | 1280×900, 390×844, 320×700 |
| partner (cuongnv) | `/partner/dashboard`, `/partner/timesheet`, `/partner/projects` | 1280×900, 390×844, 320×700 |

Mechanics:

- Real logins against the live backend (no API mocks) — the login form selectors (`#emailOrUsername`, `#password`, `button[type="submit"]`) and post-login redirect targets were verified against `src/pages/Login.tsx`; routes verified against `src/App.tsx`.
- Determinism: `contextOptions: { reducedMotion: "reduce" }` (applies from context creation) + `page.emulateMedia({ reducedMotion: "reduce" })` as a second layer + injected CSS kill style (`animation: none !important; transition: none !important; caret-color: transparent !important`) + networkidle (best-effort) + 1s settle + `toHaveScreenshot` with `animations: "disabled"`, `fullPage: true`, `maxDiffPixelRatio: 0` (exact match).
- `test.skip` guard confines the file to the chromium project even if someone runs the whole suite without `--project=chromium` (prevents 5× snapshot multiplication across the config's firefox/webkit/mobile projects without touching playwright.config.ts).
- Per-test timeout raised in-spec (`test.setTimeout(90_000)`) to absorb dev-server on-demand compile + real login; playwright.config.ts untouched.
- Snapshot names: `admin-dashboard@1280.png`, `partner-dashboard@390.png`, … (platform/project suffixes appended by Playwright). Test titles keep the requested `admin/dashboard@1280` shape for HTML-report readability.
- Login diagnostics: a `Promise.race` distinguishes success, CAPTCHA lockout (`#captchaCode` visible), and rejected credentials/backend-down (`[role="alert"]`), each with an actionable error message.
- Employee portal deliberately excluded: no spec in the repo performs a real employee login (employee suites use synthetic tokens + route mocks) and no employee QA account is documented. Noted as a follow-up, per task instructions.
- Volatile-content masking: `VOLATILE_SELECTORS` table in the spec is empty — nothing masked by default (nothing has been observed yet). Register a selector there with a comment if a surface flakes; prefer this over loosening the tolerance.

## Validation performed (no servers running)

- `pnpm exec playwright test tests/e2e/visual-baseline.spec.ts --list --project=chromium` → 24 tests listed, naming pattern as requested.
- Standalone `tsc --noEmit --strict` on the spec → clean. This caught a real bug: `test.use({ reducedMotion: "reduce" })` is not a valid fixture in @playwright/test 1.55 (`TS2353`); the working form is `test.use({ contextOptions: { reducedMotion: "reduce" } })`, which is what shipped.

## Baseline generation: BLOCKED

Both servers are down (curl to `http://localhost:8080/api/v1/health` → 000; `http://127.0.0.1:5173` → 000). Per instructions I did not start long-running dev servers, so no snapshots exist yet (`frontend/tests/e2e/visual-baseline.spec.ts-snapshots/` is not created until a run).

Exact commands for the main session (backend must be up; Playwright's webServer block auto-starts vite on 5173 if nothing is listening):

```bash
make dev   # or ensure the Go backend is listening on :8080
cd frontend && pnpm exec playwright test tests/e2e/visual-baseline.spec.ts --project=chromium --update-snapshots
```

Snapshots land in `frontend/tests/e2e/visual-baseline.spec.ts-snapshots/` and are committable (root .gitignore excludes only `frontend/test-results/` and `frontend/playwright-report/`). Commit them before wave 1 starts.

## Flakiness assessment

No empirical data — the run could not execute. Structural risks, in likelihood order:

1. Dashboard canvas charts (chart.js/recharts) animate in JS; the 1s settle + `animations: "disabled"` covers CSS and most chart durations, but a slow first compile could exceed it.
2. Live clocks / relative timestamps ("x phút trước") on dashboards.
3. Notification polling defeating networkidle (handled: networkidle is best-effort with catch; the settle delay is the real budget).
4. CAPTCHA: if the QA accounts carry a server-side captcha flag, login aborts with an explicit error (test does not fight it).

Mitigations are built in (settle budget, animation kill, empty `VOLATILE_SELECTORS` table with usage comment, chromium-only guard, generous timeouts). If specific surfaces still flake after generation, add their selectors to `VOLATILE_SELECTORS` instead of raising tolerance.

## Recommendation

Advisory, trending blocking: the harness is ready and validated statically, but the baselines do not exist yet. Wave 1 review must not use this harness as a gate until the main session runs the command above and commits the snapshots; once committed, treat exact-match failures as blocking for wave reviews. First run doubles as the flakiness survey — report any unstable surfaces and mask them via `VOLATILE_SELECTORS`.

Status: DONE_WITH_CONCERNS
Summary: visual-baseline.spec.ts written and statically validated (24 tests, tsc clean, naming per spec); baseline generation is BLOCKED because neither the backend (:8080) nor the frontend (:5173) was running and I did not start dev servers.
Concerns/Blockers: baselines must be generated by the main session (`make dev` then the playwright command above) before wave 1; employee-portal coverage absent pending a documented employee QA account; chart/clock flakiness unverified until the first run.
