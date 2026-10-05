# QA Report — UU PRO Migration Pre-Deploy QA + Production Deploy

Date: 2026-10-05. Release: `e149a39f..a470432b` (27 commits, the full UU PRO migration) → origin/main → production via `make deploy`.

## Pre-deploy QA battery (localhost:3000, tree at a470432b)

| Stage | Result |
|---|---|
| Lint + tsc (app + node) | ✓ |
| Vitest | ✓ 173 files / 813 tests |
| Auth e2e (visual config, chromium) | ✓ 18/18 |
| Employee portal e2e (chromium) | ✓ 14/14 |
| Visual baseline compare (24 surfaces) | ✓ 24/24 |
| Production build | ✓ |

Notes:

- First vitest run failed at machine load ~49 (import 1187s cumulative) — the documented load-flake class; passed with `--testTimeout=30000`. Not a regression.
- Manual visual sweep (Playwright, tolerant): partner dashboard, admin users desktop + mobile (390px), login (post `a470432b` padding fix) all render correctly. Admin dashboard shows infinite skeletons — the known backend stall (`dashboard/summary`, `bank-usage/projects`; 504s, pre-existing prod perf bug), not a frontend issue.

## Deploy

- `git push origin main`: `e149a39f..a470432b`, fast-forward.
- `make deploy`: exit 0 — amd64 frontend + backend images pushed to ghcr, `docker compose up -d --force-recreate --no-deps frontend backend` on tingting.vip, image prune done. Frontend-only change; no DB migrations.
- A second `make deploy` from a parallel session ran concurrently against the same commit; both converge to identical images.

## Post-deploy verification (https://tingting.vip)

- New bundle live: entry `index.s1DT8C_s.js` (200, 424 KB gzipped; hash differs from local dist because the image build injects env at build time — vendor chunks `utils/ui/charts/icons` and `index.BwM7sdkU.css` are identical to the QA'd local build).
- `sw.js` 200 — PWA clients self-update on next load (hard-refresh recommended once for stale-SW clients).
- Backend probe: `POST /api/v1/auth/login` with bad creds → proper JSON 401 (`Thông tin đăng nhập không hợp lệ`) — API, routing, and auth stack healthy.

## Known console errors (pre-existing, not regressions)

- `GET /api/v1/dashboard/bank-usage/projects` 504 and slow `dashboard/summary` — documented backend perf bug (integration-suite known failures; admin dashboards stall).
- `GET /api/v1/push/vapid-key` 500 — VAPID keys unconfigured on the local dev backend; app logs and continues. Prod unaffected.

## Follow-ups (owner-side)

1. Manual QA of the 15 DEV_COMPLETED kanban cards → QA_TESTED (creds in frontend/CLAUDE.md).
2. Backend perf fix for `bank-usage/projects` / `dashboard/summary` (real prod 504 bug, on record).
3. Optional: configure VAPID keys on dev environments to silence the local 500.
