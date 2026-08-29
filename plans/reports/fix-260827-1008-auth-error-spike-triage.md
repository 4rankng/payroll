# Auth API Error Spike — Triage & Noise Fix (2026-08-27)

**Trigger:** monitoring dashboard flagged `POST /api/v1/auth/login` (97 errors: 58×401, 39×400), `POST /api/v1/auth/change-password` (70×400), `POST /api/v1/auth/logout` (25×401).

**Method:** nginx access logs (status×endpoint, 24h window) + `/opt/payroll/logs/app.log` structured audit events + read-only DB lookups on prod. No UI login ([[feedback_no_prod_ui_login]]).

## Verdict: no bug in login or change-password backend. All buckets accounted for; 0×5xx in 24h.

| Bucket (24h nginx) | Evidence | Classification |
|---|---|---|
| change-password 400 ×56 | 56/56 audit "Incorrect current password"; 11 distinct users; 0 policy failures; 10/11 eventually succeeded (22×200 total) | User error in captive dialog (default pwd `Vfic@1234` is case+symbol sensitive) — compounded by frontend input defect (fixed) |
| login 401 ×49 | 37 "sai mật khẩu" + 11 "không tìm thấy tài khoản"; spread across 22 IPs / 8 identifiers (max 12 tries) | Working as designed — no brute-force signature; per-account rate limit + captcha active |
| login 400 ×38 | 33 "captcha không hợp lệ" (gate engages after 3 consecutive failures) | Working as designed |
| logout 401 ×23 | **23/23 correlate (≤8s, same IP) with a change-password 200** | Deterministic artifact: `ChangePasswordAndBlacklistToken` blacklists token on success → frontend `setTimeout(logout, 3000)` POSTs `/auth/logout` with the dead token |
| /auth/me 401 ×11 | stray queries racing the forced logout | Known/expected (documented 08-26) |

## Fixes (frontend only; deliberate API shapes untouched — 400-not-401 stays, 3s-toast UX stays)

1. **Kill the guaranteed logout-401** — `ForceChangePasswordDialog` now uses `useLogout({ skipApiCall: true })`: after a successful change the token is blacklisted server-side, so the logout POST could only ever 401. New `authService.clearLocalSession()` (extracted from `logout()`'s finally) performs the local cleanup. Eliminates ~23 logout-401s/day at current change volume and shrinks the /auth/me stray-401 race.
2. **Input hardening on all password fields** (forced dialog, `ChangePasswordModal`, `Login`): `autoComplete="current-password"/"new-password"`, `autoCapitalize="none"`, `autoCorrect="off"`, `spellCheck={false}`. Root cause of the 400 bucket: password managers had no autofill hook in the dialog, and the eye-toggle (`type="text"`) re-enabled autocorrect/autocapitalize on the exact string that must be typed byte-exact — 11 users failing to retype a password they had just logged in with.

## Verification
- `pnpm lint` (eslint + tsc --noEmit): exit 0
- Vitest: 470/470 existing + 2 new (`useLogout.test.tsx`: skip-path must not call `authService.logout`, must call `clearLocalSession`; default path unchanged) — all pass
- Blast radius: only 2 `useLogout` call sites exist (dialog + `useProfile`, latter unchanged); no test depended on logout internals; no API/route/schema/type contract changes

## Support action (not code)
- **User 1135 (linhntt, Nguyễn Thị Thuỳ Linh): 22/22 dialog failures over 2 days, never succeeded** — logs in fine (last_login 08-26 11:03) but cannot retype the current password. Needs an admin password reset / outreach.
- User 640 (hoanvv) — self-recovered 08-26, no action.

## Status
- Changes uncommitted on `main`; deploy-pending. Logout-401 metric will drop to ~0 after deploy; change-password 400 / login 401 should trend down as autofill lands on devices.
