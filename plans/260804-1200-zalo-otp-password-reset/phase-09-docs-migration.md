---
phase: 9
title: "Docs, Migration & Go-Live Checklist"
status: pending
priority: P2
dependencies: ["6", "7"]
effort: "S"
---

# Phase 9: Docs, Migration & Go-Live Checklist

## Overview

Document the feature, add the env-var reference (marked as bootstrap-only seed), write
the **admin connect runbook** (the primary path via the `/admin/settings?tab=zalo` UI
built in Phase 5) plus a disaster-recovery runbook, and capture the go-live checklist
(template approval being the main external gate; callback-URL registration the second).

This phase has **no production code** — Phase 4 owns the backend, Phase 5 owns the UI.
There is **no DB migration**: the feature reuses the existing generic `settings` table
(two new key-value rows, written by Phase 4's service at runtime). This phase is purely
documentation + go-live operability.

## Requirements

- **Functional**
  - `docs/` gets an ADR + a short feature doc describing the flow, security contract, and admin OAuth connect.
  - `.env.example` (backend) lists all `ZALO_*` vars, **clearly marked as bootstrap-only seed** (superseded by the admin UI after first save).
  - Admin connect runbook: step-by-step for an admin using `/admin/settings?tab=zalo` (no shell/SQL).
  - DR runbook: how to reseed/recover if the refresh_token is lost or the settings row is corrupted.
  - `backend/AGENTS.md` cross-references the new packages + the admin-managed toggle.
- **Non-functional**
  - No secrets committed. Runbooks describe *how* to connect; they never contain real tokens/keys.
  - Go-live checklist is reviewable by a non-coder (PM/ops).

## Architecture

### Artifact list

| Artifact | Path | Owner | Notes |
|---|---|---|---|
| Feature doc | `docs/features/zalo-otp-password-reset.md` | this phase | User-facing flow diagram + security contract summary (link to `plan.md`). |
| ADR | `docs/adr/NNNN-zalo-otp-password-reset.md` | this phase | Records D1–D8 decisions + the "employees only" scope rationale (D6) + the "DB-driven toggle" rationale (D5). |
| Env reference | `backend/configs/.env.example` (or repo-root `.env.example`) | this phase | `ZALO_*` vars documented as **bootstrap-only seed** (superseded by DB after first admin save). |
| Admin UI guide | `docs/runbooks/zalo-admin-connect.md` | this phase | How an admin connects Zalo via the settings tab (Phase 5 UI) — the primary path. |
| DR runbook | `docs/runbooks/zalo-oauth-disaster-recovery.md` | this phase | Fallback: how to re-OAuth if the refresh_token is lost (env re-seed or direct DB write). DR-only. |
| AGENTS.md cross-ref | `backend/AGENTS.md` | this phase | One-line note under the auth/security section. |

### Env vars to document

```dotenv
# Zalo ZNS password reset (employee mobile channel).
# These are BOOTSTRAP-ONLY SEED VALUES. On first boot, if the `zalo.credentials`
# settings row is empty, the system seeds it from these env vars. Thereafter the
# admin-managed DB settings (via /admin/settings?tab=zalo) are authoritative and
# these env vars are IGNORED. Changing them after first boot has no effect unless
# the settings row is deleted (disaster recovery).
ZALO_RESET_ENABLE=false
# Payroll OA "Ting Ting Software Solution" / app "TingTing Soft" — NOT the tuyennhanvien.vn OA.
ZALO_APP_ID=
ZALO_SECRET_KEY=           # never commit; rotate via Zalo OA console or admin UI
# Transactional template OTP-ZNS-v1 (3 params). Default 617976.
ZALO_RESET_TEMPLATE_ID=617976
# OAuth callback URL — must match exactly what's registered in the Zalo OA console.
ZALO_OAUTH_CALLBACK_URL=https://api.tingting.vip/api/v1/admin/zalo/oauth/callback
# Tunables (sane defaults; not admin-overridable)
ZALO_RESET_CODE_TTL=10m
ZALO_RESET_RATE_LIMIT=3    # per normalized mobile per hour
```

### Admin connect runbook (primary path)

```
1. In the Zalo OA console ("Ting Ting Software Solution" / "TingTing Soft"), ensure
   the OA is verified and has the ZNS permission.
2. Register the OAuth callback URL shown in /admin/settings?tab=zalo (copy button)
   into the OA console's allowed redirect_uris.
3. Confirm template OTP-ZNS-v1 (id 617976) is APPROVED ("Đang duyệt" → "Đã duyệt").
   Prod go-live blocked until approved; dev/test proceed in sandbox (-127 non-fatal).
4. In /admin/settings?tab=zalo:
   a. Enter App ID + Secret Key + Template ID (617976). Click "Lưu thông tin".
   b. Click "Kết nối Zalo" → authorize in the Zalo popup → return to the settings tab.
   c. Verify status badge flips to "Đã kết nối" with a live expiry.
   d. Toggle "Bật tính năng" ON.
5. Smoke test: as an employee, /forgot-password → Zalo tab → enter mobile → receive OTP.
```

### Disaster-recovery runbook (refresh_token lost / DB row corrupted)

```
Only use if the admin UI cannot recover the connection (e.g. refresh_token exhausted,
DB row deleted). Two options:
  A) Re-seed from env: set ZALO_* env vars to known-good values, DELETE the
     zalo.credentials + zalo.enabled settings rows, restart api-server (SeedFromEnvIfEmpty
     repopulates), then re-run the admin connect flow.
  B) Direct DB write (last resort): UPDATE settings SET value='<json>' WHERE
     `key`='zalo.credentials'. Document in the DR runbook; never routine.
```

(The OAuth code-exchange is invoked by Phase 4's admin callback handler
`/admin/zalo/oauth/callback`, which calls `provider.ExchangeCode`. The admin triggers it
by clicking "Kết nối Zalo" in the Phase 5 UI. No CLI subcommand or `make` target needed
for routine operation — only the DR runbook's env-reseed path touches the DB directly.)

### Settings-row verification (no schema migration)

```bash
# After an admin saves credentials + connects in the UI, verify the two rows exist:
mysql -e "SELECT \`key\`, LEFT(value,40) FROM settings WHERE \`key\` LIKE 'zalo.%';"
# Expected:
#   zalo.enabled      | true
#   zalo.credentials  | {"app_id":"...","template_id":"617976" ...   (tokens redacted in this query)
```

No `migrate-up`/`migrate-down` is needed — Phase 4's service writes these rows lazily on
first admin save (or seeds them from env on first boot via `SeedFromEnvIfEmpty`).

## Related Code Files

- **Create:**
  - `docs/features/zalo-otp-password-reset.md`
  - `docs/adr/NNNN-zalo-otp-password-reset.md`
  - `docs/runbooks/zalo-admin-connect.md`
  - `docs/runbooks/zalo-oauth-disaster-recovery.md`
  - `backend/configs/.env.example` entries (or extend the existing one)
- **Modify:**
  - `backend/AGENTS.md` — cross-reference note.
- **Verify (no migration — settings rows written at runtime by Phase 4):**
  - After first admin save, confirm the two `zalo.*` rows exist in the `settings` table (see "Settings-row verification" above).

## Implementation Steps

1. **Feature doc** — diagram (reuse the one in `plan.md`), security contract summary, link to `plan.md` and the ADR.
2. **ADR** — number it (next in `docs/adr/`); record D1–D8 + the residual risks R-Z1/R-Z3/R-Z5/R-Z7/R-Z18. Reference the email-reset ADR for cross-channel parity. Call out the D5 DB-driven-toggle rationale explicitly (it diverges from OTP_ENABLE's env-only model).
3. **Env reference** — append the `ZALO_*` block; **clearly mark them bootstrap-only seed** (most operators will instead use the admin UI).
4. **Admin connect runbook** — the primary path (steps 1–5 above); screenshot-worthy.
5. **DR runbook** — refresh_token lost / DB corrupted; the env-reseed + direct-DB-write options.
6. **`backend/AGENTS.md`** — one paragraph under auth/security noting the new channel, the admin-managed toggle, and the settings keys.
7. **`/understand` update** — per root `AGENTS.md`, run `/understand` after the code lands so the KB reflects the new packages + endpoints. (Session step, not a doc artifact.)

## Success Criteria

- [ ] Feature doc + ADR + both runbooks + env reference all written and cross-linked.
- [ ] A non-coder admin can follow the **admin connect runbook** end-to-end (peer-review with PM/ops if possible) — no shell/SQL required.
- [ ] The DR runbook's env-reseed path is tested once in staging (delete settings rows, restart, verify reseed).
- [ ] Go-live checklist is a single page with: (a) template `617976` approved, (b) callback URL registered in OA console, (c) admin connected via UI, (d) toggle ON, (e) employee smoke test passed.
- [ ] `/understand` KB updated after merge (commit-level incremental rebuild).

## Risk Assessment

- **R-Z16 (template approval latency):** Template `617976` (`OTP-ZNS-v1`) was submitted 2026-08-04 16:45 and is **Đang duyệt** (Zalo quotes 2–3 business days). Dev/test can proceed immediately (sandbox returns `-127`, treated as non-fatal), but **prod go-live is blocked** until the status flips to "Đã duyệt". **Mitigation:** the go-live checklist sign-off explicitly requires re-checking the OA console status; no code change is needed when approval lands (the same `617976` is used in both envs).
- **R-Z17 (OAuth handshake is a one-person ritual):** Only someone with OA-admin access can do it; if they leave, the refresh chain breaks. **Mitigation:** document the re-handshake procedure in the runbook; the DB row makes the current tokens inspectable (access only; refresh is the sensitive one) for debugging.
